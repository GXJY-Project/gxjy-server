"""将已验证补丁部署到维护者授权的本地 B 服客户端，只操作明确的脚本路径"""

import argparse
import hashlib
import json
from pathlib import Path
import re
import shlex
import subprocess


PACKAGE = "com.bilibili.gxjy"
BASE_JSC_SHA256 = "65fec07161fa8c7a73b0dc973d4bb728bdc57ebb620b4ec696c9aa030dc50520"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--device", default="emulator-5554")
    parser.add_argument("--stage", type=Path, required=True)
    parser.add_argument("--restore-after-update", action="store_true")
    args = parser.parse_args()
    stage = args.stage.resolve()
    report = json.loads((stage / "patch-report.json").read_text("utf-8"))
    patch = stage / "patched-index.jsc"
    patch_hash = hashlib.sha256(patch.read_bytes()).hexdigest()
    if patch_hash != report["patched_jsc_sha256"] or not report["original_client_rsa_pkcs1v15_verified"]:
        raise ValueError("补丁哈希或认证验证状态不匹配")
    deployment_report = stage / ("redeployment-report.json" if args.restore_after_update else "deployment-report.json")
    if deployment_report.exists():
        raise ValueError("本次部署记录已存在，请先审阅，不重复部署")
    if args.restore_after_update:
        previous = json.loads((stage / "deployment-report.json").read_text("utf-8"))
        if previous["deployed_jsc_sha256"] != patch_hash or previous["device"] != args.device:
            raise ValueError("恢复范围与本次已授权部署记录不一致")

    def adb(*arguments):
        return subprocess.run(["adb", "-s", args.device, *arguments], capture_output=True, check=True)

    def root(command):
        # 单一、正确引用的 Android shell 命令，不将本机路径交给 Android 执行
        return adb("shell", "su -c " + shlex.quote(command)).stdout.decode("utf-8").strip()

    if root("id -u") != "0":
        raise ValueError("当前测试环境未授予管理权限")
    apk = adb("shell", "pm", "path", PACKAGE).stdout.decode("utf-8").strip()
    if not apk.startswith("package:") or "\n" in apk:
        raise ValueError("包路径不唯一，拒绝选择未知 split APK")
    apk = apk.removeprefix("package:")
    if not re.fullmatch(r"/data/app/[A-Za-z0-9_/~+=.-]+/base\.apk", apk):
        raise ValueError("安装包路径不是已确认的本地应用位置")
    installed_hash = root("unzip -p " + shlex.quote(apk) + " assets/assets/main/index.jsc | sha256sum").split()[0]
    if installed_hash != BASE_JSC_SHA256:
        raise ValueError("实际安装的底包主脚本已改变，拒绝套用补丁")
    app_data = "/data/user/0/" + PACKAGE
    directory = app_data + "/files/remote-asset/assets/main"
    target = directory + "/index.jsc"
    existing = root("if [ -e " + shlex.quote(target) + " ]; then sha256sum " + shlex.quote(target) + "; else echo ABSENT; fi")
    if existing != "ABSENT":
        raise ValueError("更新目录已有脚本，拒绝覆盖来源不明的客户端改动")
    owner = root("stat -c %u:%g " + shlex.quote(app_data))
    if not re.fullmatch(r"\d+:\d+", owner):
        raise ValueError("无法确认应用目录所有者")
    created = []
    for relative in ["files", "files/remote-asset", "files/remote-asset/assets", "files/remote-asset/assets/main"]:
        path = app_data + "/" + relative
        exists = root("if [ -d " + shlex.quote(path) + " ]; then echo EXISTS; else echo ABSENT; fi")
        if exists == "ABSENT":
            root("mkdir " + shlex.quote(path))
            root("chown " + owner + " " + shlex.quote(path))
            root("chmod 755 " + shlex.quote(path))
            root("restorecon " + shlex.quote(path))
            created.append(path)
    temporary_directory = "/data/local/tmp/gxjy-bili-local-" + patch_hash[:16]
    adb("shell", "mkdir", "-p", temporary_directory)
    staged = temporary_directory + "/index.jsc"
    adb("push", str(patch), staged)
    temporary_target = directory + "/index.jsc.gxjy-local.tmp"
    root("cp " + shlex.quote(staged) + " " + shlex.quote(temporary_target))
    root("chown " + owner + " " + shlex.quote(temporary_target))
    root("chmod 644 " + shlex.quote(temporary_target))
    root("restorecon " + shlex.quote(temporary_target))
    root("mv " + shlex.quote(temporary_target) + " " + shlex.quote(target))
    deployed_hash = root("sha256sum " + shlex.quote(target)).split()[0]
    if deployed_hash != patch_hash:
        raise ValueError("设备中的补丁哈希不一致")
    metadata = {"package": PACKAGE, "device": args.device, "target": target,
                "source_jsc_sha256": installed_hash, "deployed_jsc_sha256": deployed_hash,
                "previous_target": "absent", "created_directories": created,
                "original_apk_preserved": True, "application_data_preserved": True}
    with deployment_report.open("x", encoding="utf-8") as stream:
        json.dump(metadata, stream, ensure_ascii=False, indent=2)
        stream.write("\n")
    print(json.dumps(metadata, ensure_ascii=False))


if __name__ == "__main__":
    main()
