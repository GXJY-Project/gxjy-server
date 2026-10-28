"""离线复核已授权 B 服快照，不连接网络，输出仅写入 Git 忽略目录"""

import argparse
import ast
import base64
import gzip
import hashlib
import json
from pathlib import Path
import struct
import subprocess
import uuid


NATIVE_SHA256 = "30c643281dc4241e6d1c55428529372c5ac9751f2d2b34cc59ae2f9d49b35020"
PROTOTYPE_SHA256 = "53b3874bad499d7491b1cdf90d1ca38f9432ee247ec4524c38b0c9c66793fae8"


def sha256(data):
    return hashlib.sha256(data).hexdigest()


def script_key(native):
    # 地址来自该 SHA-256 对应 x86_64 AppDelegate 的反汇编，不用于其他 APK
    if sha256(native) != NATIVE_SHA256:
        raise ValueError("原生库不是已核实的 B 服版本，拒绝套用地址")
    section_offset = struct.unpack_from("<Q", native, 40)[0]
    entry_size, count = struct.unpack_from("<HH", native, 58)
    sections = [struct.unpack_from("<IIQQQQIIQQ", native, section_offset + i * entry_size)
                for i in range(count)]

    def offset(address):
        return next(s[4] + address - s[3] for s in sections
                    if s[1] != 8 and s[3] <= address < s[3] + s[5])

    first, second = offset(0x7AB5A4), offset(0x7AB5B6)
    if native[first:first + 2] != b"\x48\xb8" or native[second:second + 3] != b"\xc7\x84\x24":
        raise ValueError("设置脚本密钥的指令不匹配")
    return native[first + 2:first + 10] + native[second + 7:second + 11]


def prototype_decrypt(path):
    raw = path.read_bytes()
    if sha256(raw) != PROTOTYPE_SHA256:
        raise ValueError("Python 原型版本已改变，请重新核对来源")
    tree = ast.parse(raw.decode("utf-8"))
    names = {"_s2l", "_l2s", "xxtea_dec"}
    functions = [node for node in tree.body
                 if isinstance(node, ast.FunctionDef) and node.name in names]
    if {node.name for node in functions} != names:
        raise ValueError("原型缺少 XXTEA 函数")
    # 只复用原有算法函数，不导入模块、不运行旧工具的补丁或写入入口
    scope = {"struct": struct}
    exec(compile(ast.Module(body=functions, type_ignores=[]), str(path), "exec"), scope)
    return scope["xxtea_dec"]


def write_identical_or_new(path, data):
    if path.exists():
        if path.read_bytes() != data:
            raise ValueError(f"输出已存在且内容不同：{path.name}")
        return
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("xb") as stream:
        stream.write(data)


def main():
    project = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=project / "data/assets/bili")
    parser.add_argument("--prototype", type=Path, default=project.parent /
                        "gxjy-release-20261019/gxjy-release-20261019/3-CLIENT-SRC/patch_cvguard.py")
    parser.add_argument("--asset-key-file", type=Path, default=project / "data/keys/asset.key")
    args = parser.parse_args()
    root = args.root.resolve()
    output = root / "analysis/verified"
    ignored = subprocess.run(["git", "check-ignore", "-q", str(output / "index.js")],
                             cwd=project, check=False)
    if ignored.returncode != 0:
        raise ValueError("输出目录未被 Git 忽略，拒绝写入客户端材料")
    decrypt = prototype_decrypt(args.prototype)
    native = (root / "source/apk/lib/x86_64/libcocos2djs.so").read_bytes()
    key = script_key(native)
    asset_key = args.asset_key_file.read_bytes().strip()
    manifest = json.loads((root / "resolved/project.manifest").read_text("utf-8"))
    evidence = {"native_sha256": sha256(native), "scripts": [], "tables": []}

    def resource(relative):
        raw = (root / "resolved" / relative).read_bytes()
        expected = manifest["assets"][relative]
        if len(raw) != expected["size"] or hashlib.md5(raw).hexdigest() != expected["md5"]:
            raise ValueError(f"资源与 project.manifest 不符：{relative}")
        return raw

    def decode_asset(raw):
        if raw.startswith(b"REF:"):
            body = raw[8:]
            decoded = decrypt(body[:len(body) // 4 * 4], asset_key)
            if not decoded.startswith((b"\x1f\x8b", b"{", b"[")):
                length = int.from_bytes(body[2:6], "big")
                if length > len(body) - 6 or length % 4:
                    raise ValueError("REF 内层长度无效")
                decoded = decrypt(body[6:6 + length], asset_key) + body[6 + length:]
            raw = decoded
        return gzip.decompress(raw) if raw.startswith(b"\x1f\x8b") else raw

    for relative, name in [("assets/main/index.jsc", "index.js"),
                           ("src/settings.jsc", "settings.js"),
                           ("src/assets/Script/Global.jsc", "Global.js")]:
        raw = resource(relative)
        if len(raw) < 8 or len(raw) % 4:
            raise ValueError("JSC 长度无效")
        compressed = decrypt(raw, key)
        if not compressed.startswith(b"\x1f\x8b"):
            raise ValueError("JSC 解密结果不是该快照预期的 GZip")
        decoded = gzip.decompress(compressed)
        decoded.decode("utf-8")
        write_identical_or_new(output / name, decoded)
        evidence["scripts"].append({"source": relative, "input_bytes": len(raw),
                                    "input_sha256": sha256(raw), "gzip_bytes": len(compressed),
                                    "output_bytes": len(decoded), "output_sha256": sha256(decoded)})

    main_raw = (root / "source/apk/assets/main.js").read_bytes()
    write_identical_or_new(output / "apk-main.js", decode_asset(main_raw))
    bundle = json.loads(decode_asset(resource("assets/resources/config.json")))

    def table(name):
        index = next(k for k, v in bundle["paths"].items() if v[0] == "config/data/" + name)
        identifier = bundle["uuids"][int(index)]
        if len(identifier) == 22:
            identifier = str(uuid.UUID(bytes=bytes.fromhex(identifier[:2]) + base64.b64decode(identifier[2:])))
        relative = f"assets/resources/import/{identifier[:2]}/{identifier}.json"
        raw = resource(relative)
        document = json.loads(decode_asset(raw))
        payload = document["json"] if isinstance(document, dict) else document[5][0][2]
        evidence["tables"].append({"name": name, "source": relative, "sha256": sha256(raw)})
        return payload

    protocol = table("SERVER_Protocol")
    for index in range(1, protocol.pop("_subJsonCount", 0) + 1):
        for name, row in table(f"SERVER_Protocol_sub{index}").items():
            if not name.startswith("_"):
                # 更新脚本 JsonReader._createJsonObj 使用 Object.assign 后表覆盖前表
                protocol[name] = row
    if "_subFiles" in protocol:
        raise ValueError("出现尚未核实的 SERVER_Protocol 引用表布局")
    mapping = {name: row["ProtocolCode"] for name, row in protocol.items()}
    by_code = {}
    for name, code in mapping.items():
        by_code.setdefault(code, []).append(name)
    # 原表可能存在别名，不根据 u16 传输字段擅自删除或改写配置项
    evidence["protocol_aliases"] = {str(code): names for code, names in by_code.items() if len(names) > 1}
    evidence["protocol_outside_u16"] = {name: code for name, code in mapping.items() if not 0 <= code <= 65535}
    for name, value in [("protocol_map.json", mapping),
                        ("table-fixture.json", {"tables": {"SERVER_Protocol": protocol},
                                                "table_keys": {"SERVER_Protocol": list(protocol)}}),
                        ("evidence.json", evidence)]:
        write_identical_or_new(output / name, (json.dumps(value, ensure_ascii=False, indent=2) + "\n").encode("utf-8"))
    print(json.dumps({"status": "verified", "script_count": len(evidence["scripts"]),
                      "protocol_count": len(mapping), "output": str(output)}, ensure_ascii=False))


if __name__ == "__main__":
    main()
