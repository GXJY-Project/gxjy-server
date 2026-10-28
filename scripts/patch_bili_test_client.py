"""为已授权的 B 服底包生成本地连接补丁，原文件和真实会话数据不进入输出"""

import argparse
import ast
import gzip
import hashlib
import json
from pathlib import Path
import re
import struct
import subprocess

from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import padding

from decode_bili_scripts import PROTOTYPE_SHA256, script_key, write_identical_or_new


BASE_JSC_SHA256 = "65fec07161fa8c7a73b0dc973d4bb728bdc57ebb620b4ec696c9aa030dc50520"
ORIGINAL_PUBLIC_SHA256 = "2e742265a6e64446c029abdc6bc055aa1756b32d71ecc208001af6fc520233b9"
ORIGINAL_URL = 'G_Config.url = [ "https://l1-ob-bili-login-klight.bilibiligame.net/", "https://l2-ob-bili-login-klight.bilibiligame.net/" ];'
LOCAL_URL = 'G_Config.url = [ "http://127.0.0.1:8080/" ]; console.log("GXJY_LOCAL_LOGIN_ROOT", G_Config.url[0]);'


def digest(data):
    return hashlib.sha256(data).hexdigest()


def prototype_codec(path):
    raw = path.read_bytes()
    if digest(raw) != PROTOTYPE_SHA256:
        raise ValueError("原 XXTEA 工具版本不匹配")
    names = {"_s2l", "_l2s", "xxtea_dec", "xxtea_enc"}
    functions = [node for node in ast.parse(raw.decode("utf-8")).body
                 if isinstance(node, ast.FunctionDef) and node.name in names]
    if {node.name for node in functions} != names:
        raise ValueError("原工具缺少已确认的编码函数")
    # 只运行既有算法函数，保留原型文件及其版权说明，不执行旧补丁或安装入口
    scope = {"struct": struct}
    exec(compile(ast.Module(body=functions, type_ignores=[]), str(path), "exec"), scope)
    return scope["xxtea_dec"], scope["xxtea_enc"]


def main():
    project = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--node", required=True)
    args = parser.parse_args()
    output = args.output.resolve()
    ignored = subprocess.run(["git", "check-ignore", "-q", str(output / "patched-index.jsc")],
                             cwd=project, check=False)
    if ignored.returncode != 0:
        raise ValueError("客户端输出必须位于 Git 忽略目录")
    root = project / "data/assets/bili"
    original = root / "source/apk/assets/assets/main/index.jsc"
    raw = original.read_bytes()
    if digest(raw) != BASE_JSC_SHA256:
        raise ValueError("输入不是本轮核对过的 B 服底包脚本")
    prototype = project.parent / "gxjy-release-20261019/gxjy-release-20261019/3-CLIENT-SRC/patch_cvguard.py"
    decrypt, encrypt = prototype_codec(prototype)
    key = script_key((root / "source/apk/lib/x86_64/libcocos2djs.so").read_bytes())
    source_bytes = gzip.decompress(decrypt(raw, key))
    source = source_bytes.decode("utf-8")
    if source.count(ORIGINAL_URL) != 1:
        raise ValueError("登录地址替换点不唯一或已改变")
    public_blocks = list(re.finditer(r"-----BEGIN PUBLIC KEY-----[A-Za-z0-9+/=\s]+-----END PUBLIC KEY-----", source))
    if len(public_blocks) != 1:
        raise ValueError("客户端公钥替换点不唯一")
    original_public = public_blocks[0].group()
    normalized_public = original_public.replace("-----BEGIN PUBLIC KEY-----", "-----BEGIN PUBLIC KEY-----\n").replace("-----END PUBLIC KEY-----", "\n-----END PUBLIC KEY-----")
    old_key = serialization.load_pem_public_key(normalized_public.encode("ascii"))
    old_der = old_key.public_bytes(serialization.Encoding.DER, serialization.PublicFormat.SubjectPublicKeyInfo)
    if digest(old_der) != ORIGINAL_PUBLIC_SHA256:
        raise ValueError("客户端公钥不是已记录的 B 服公钥")
    config_path = project / "config.json"
    config = json.loads(config_path.read_text("utf-8"))
    private_path = Path(config["rsa_pem"])
    if not private_path.is_absolute():
        private_path = project / private_path
    private = serialization.load_pem_private_key(private_path.read_bytes(), password=None)
    if private.key_size != old_key.key_size:
        raise ValueError("本地 RSA 位宽与客户端已确认的握手长度不同")
    public = private.public_key()
    public_pem = public.public_bytes(serialization.Encoding.PEM, serialization.PublicFormat.SubjectPublicKeyInfo)
    public_js = public_pem.decode("ascii").replace("\n", "")
    patched = source.replace(ORIGINAL_URL, LOCAL_URL, 1).replace(original_public, public_js, 1)
    if patched.replace(LOCAL_URL, ORIGINAL_URL, 1).replace(public_js, original_public, 1) != source:
        raise ValueError("补丁包含计划以外的源码变化")
    patched_bytes = patched.encode("utf-8")
    compressed = gzip.compress(patched_bytes, compresslevel=9, mtime=0)
    encoded = encrypt(compressed, key)
    if gzip.decompress(decrypt(encoded, key)) != patched_bytes:
        raise ValueError("补丁重新编码后的解码结果不一致")
    for name, data in [("original-index.js", source_bytes), ("patched-index.js", patched_bytes),
                       ("patched-index.jsc", encoded), ("test-public.pem", public_pem)]:
        write_identical_or_new(output / name, data)
    subprocess.run([args.node, "--check", str(output / "patched-index.js")], check=True)
    subprocess.run([args.node, str(project / "scripts/verify_bili_test_client.cjs"), str(output)], check=True)
    fixture = json.loads((output / "synthetic-auth.json").read_text("utf-8"))
    opened = private.decrypt(bytes.fromhex(fixture["rsa_block_hex"]), padding.PKCS1v15())
    if opened.hex() != fixture["synthetic_session_hex"]:
        raise ValueError("原客户端认证帧的 RSA 区块不能由配置私钥解密")
    report = {
        "source": str(original.relative_to(project)), "source_jsc_sha256": digest(raw),
        "source_js_sha256": digest(source_bytes), "patched_js_sha256": digest(patched_bytes),
        "patched_jsc_sha256": digest(encoded), "patched_jsc_bytes": len(encoded),
        "login_root": "http://127.0.0.1:8080/", "original_public_spki_sha256": digest(old_der),
        "test_public_spki_sha256": digest(public.public_bytes(serialization.Encoding.DER, serialization.PublicFormat.SubjectPublicKeyInfo)),
        "roundtrip_verified": True, "javascript_syntax_verified": True,
        "original_client_rsa_pkcs1v15_verified": True, "changes": ["login root and startup marker", "test public key"],
    }
    write_identical_or_new(output / "patch-report.json", (json.dumps(report, ensure_ascii=False, indent=2) + "\n").encode("utf-8"))
    print(json.dumps(report, ensure_ascii=False))


if __name__ == "__main__":
    main()
