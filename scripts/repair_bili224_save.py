"""离线修复本地 B224 配置和旧存档；默认只预览，--apply 才备份并写入"""

import argparse
import base64
import copy
from contextlib import ExitStack
from datetime import datetime
import gzip
import hashlib
import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import uuid

from decode_bili_scripts import prototype_decrypt


def encoded(value):
    return (json.dumps(value, ensure_ascii=False, indent=2) + "\n").encode("utf-8")


def plan(player, tables):
    """只隔离在目标表中不存在的记录，不重映射编号、不修改有效记录"""
    result = copy.deepcopy(player)
    quarantine = result.setdefault("_bili224_quarantine", {})
    if not isinstance(quarantine, dict):
        raise ValueError("已有隔离区不是对象")
    counts = {}
    for field, table, reference in (
        ("equip_state", "EquipBase", lambda key, value: value.get("template_id")),
        ("tasks", "TaskBase", lambda key, value: key),
        ("bag_items", "ItemConfig", lambda key, value: key),
    ):
        records = result.get(field, {})
        if not isinstance(records, dict):
            raise ValueError("存档字段不是对象：" + field)
        invalid = {}
        for key, value in records.items():
            if field != "bag_items" and not isinstance(value, dict):
                raise ValueError("存档记录不是对象：" + field)
            ref = reference(key, value)
            if not isinstance(ref, str) or not ref:
                raise ValueError("存档配置引用不是非空字符串：" + field)
            if ref not in tables[table]:
                invalid[key] = value
        if invalid:
            target = quarantine.setdefault(field, {})
            if not isinstance(target, dict) or any(k in target and target[k] != v for k, v in invalid.items()):
                raise ValueError("已有隔离记录冲突：" + field)
            target.update(invalid)
            for key in invalid:
                del records[key]
        counts[field] = len(invalid)
    for hero in player.get("hero_state", {}):
        if hero not in tables["HeroBase"]:
            raise ValueError("英雄编号不属于 B224，需人工确认，拒绝写入")
    if not quarantine:
        result.pop("_bili224_quarantine")
    return result, counts


def load_tables(project, names=("EquipBase", "TaskBase", "ItemConfig", "HeroBase")):
    # 复用已核验原型的 XXTEA 函数，仅在内存解码既有合法本地资源
    decrypt = prototype_decrypt(project.parent / "gxjy-release-20261019/gxjy-release-20261019/3-CLIENT-SRC/patch_cvguard.py")
    key = (project / "data/keys/asset.key").read_bytes().strip()
    root = project / "data/assets/bili/source/apk/assets/assets"

    def decode(raw):
        if raw.startswith(b"REF:"):
            body = raw[8:]
            decoded = decrypt(body[:len(body) // 4 * 4], key)
            if not decoded.startswith((b"\x1f\x8b", b"{", b"[")):
                length = int.from_bytes(body[2:6], "big")
                if length > len(body) - 6 or length % 4:
                    raise ValueError("REF 内层长度无效")
                decoded = decrypt(body[6:6 + length], key) + body[6 + length:]
            raw = decoded
        return gzip.decompress(raw) if raw.startswith(b"\x1f\x8b") else raw

    source = project / "data/assets/bili/analysis/local-client/base224-local/original-index.js"
    if hashlib.sha256(source.read_bytes()).hexdigest() != "89513c054d788a55617a86b4820578019b0135e5f3aeb86692bb76b1af39d5c6":
        raise ValueError("B224 原脚本与已核验版本不符")
    bundle = json.loads(decode((root / "resources/config.json").read_bytes()))
    cache = {}

    def table(name):
        if name in cache:
            return cache[name]
        indices = [k for k, value in bundle["paths"].items() if value[0] == "config/data/" + name]
        if len(indices) != 1:
            raise ValueError("配置表索引不唯一：" + name)
        identifier = bundle["uuids"][int(indices[0])]
        if len(identifier) == 22:
            identifier = str(uuid.UUID(bytes=bytes.fromhex(identifier[:2]) + base64.b64decode(identifier[2:])))
        files = list((root / "resources/import" / identifier[:2]).glob(identifier + "*.json"))
        if len(files) != 1:
            raise ValueError("配置表文件不唯一：" + name)
        document = json.loads(decode(files[0].read_bytes()))
        result = dict(document["json"] if isinstance(document, dict) else document[5][0][2])
        cache[name] = result
        for index in range(1, int(result.pop("_subJsonCount", 0)) + 1):
            result.update(table(name + "_sub" + str(index)))
        for sub in result.pop("_subFiles", []) or []:
            result.update(table(sub))
        return result

    return {name: table(name) for name in names}


def atomic_write(path, raw):
    descriptor, temporary = tempfile.mkstemp(prefix=".b224-", dir=path.parent)
    try:
        with os.fdopen(descriptor, "wb") as stream:
            stream.write(raw)
            stream.flush()
            os.fsync(stream.fileno())
        os.replace(temporary, path)
    finally:
        if os.path.exists(temporary):
            os.unlink(temporary)


def apply_files(project, originals, replacements, ports):
    # 持有端口直至迁移结束，拒绝服务仍在运行的情况
    with ExitStack() as stack:
        for port in sorted(set(ports)):
            connection = stack.enter_context(socket.socket())
            if hasattr(socket, "SO_EXCLUSIVEADDRUSE"):
                connection.setsockopt(socket.SOL_SOCKET, socket.SO_EXCLUSIVEADDRUSE, 1)
            connection.bind(("0.0.0.0", port))
        if any(path.read_bytes() != raw for path, raw in originals.items()):
            raise ValueError("预览后文件已变化，拒绝覆盖")
        backup = project / "data/save" / ("b224-repair-" + datetime.now().strftime("%Y%m%d-%H%M%S-%f"))
        ignored = subprocess.run(["git", "check-ignore", "-q", str(backup / "config.json")], cwd=project, check=False)
        if ignored.returncode != 0:
            raise ValueError("备份目录未被 Git 忽略")
        backup.mkdir(mode=0o700)
        for path, raw in originals.items():
            target = backup / path.name
            with target.open("xb") as stream:
                stream.write(raw)
                stream.flush()
                os.fsync(stream.fileno())
            target.chmod(0o600)
            if target.read_bytes() != raw:
                raise ValueError("备份校验失败，拒绝修改")
        written = []
        try:
            for path, raw in replacements.items():
                if path.read_bytes() != originals[path]:
                    raise ValueError("备份后文件已变化，拒绝覆盖")
                atomic_write(path, raw)
                written.append(path)
                if path.read_bytes() != raw:
                    raise ValueError("写入校验失败")
        except Exception:
            for path in reversed(written):
                atomic_write(path, originals[path])
            raise
        return backup


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    project = Path(__file__).resolve().parents[1]
    config_path = project / "config.json"
    config_raw = config_path.read_bytes()
    config = json.loads(config_raw)
    player_path = (project / config["save_dir"] / "save_localplayer.json").resolve()
    if not player_path.is_relative_to(project / "data/save"):
        raise ValueError("只允许修复项目 data/save 中的本地存档")
    player_raw = player_path.read_bytes()
    repaired, counts = plan(json.loads(player_raw), load_tables(project))
    config.update(client_profile="bili-224", asset_root="data/assets/bili/source/apk/assets/assets",
                  bundle_config="data/assets/bili/source/apk/assets/assets/resources/config.json")
    report = {"mode": "apply" if args.apply else "preview", "quarantined": counts,
              "original_save_sha256": hashlib.sha256(player_raw).hexdigest()}
    originals = {player_path: player_raw, config_path: config_raw}
    replacements = {player_path: encoded(repaired), config_path: encoded(config)}
    if args.apply:
        if all(json.loads(originals[p]) == json.loads(raw) for p, raw in replacements.items()):
            report["status"] = "unchanged"
        else:
            backup = apply_files(project, originals, replacements, [config["http_port"], config["ws_port"]])
            report["backup"] = backup.relative_to(project).as_posix()
            report["status"] = "applied"
    print(json.dumps(report, ensure_ascii=True))


if __name__ == "__main__":
    main()
