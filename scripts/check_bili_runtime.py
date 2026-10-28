"""用授权 B 服配置与合成存档复核真实 Go 加载路径，不连接外部服务"""

import argparse
import copy
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

import python_baseline as baseline


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write_new_or_identical(path, value):
    encoded = (json.dumps(value, ensure_ascii=False, indent=2) + "\n").encode("utf-8")
    if path.exists():
        if path.read_bytes() != encoded:
            raise ValueError("已有验证文件内容不同，拒绝覆盖：" + path.name)
    else:
        path.parent.mkdir(parents=True, exist_ok=True)
        with path.open("xb") as stream:
            stream.write(encoded)


def load_bili_tables(root, key_file):
    reader_path = baseline.SOURCE.parent / "read_config_table.py"
    expected = {
        baseline.SOURCE: "13ef3e5dfcda8ed996693711c5c573df533b41b89d812d49ce4a132fc49bda25",
        reader_path: "9e0f0777b1b5b37fdf52088c1d06ea1018cbd93836488bf577452230d8ed2cf9",
        baseline.SOURCE.parent.parent / "gxjy_decrypt.py":
            "5afd80bae5169ef97cfe05cadb8375911c500b53a4cc4d66e5b5507cbd0da0be",
    }
    for path, sha in expected.items():
        if digest(path) != sha:
            raise ValueError("原型版本改变，请重新核对：" + path.name)
    spec = importlib.util.spec_from_file_location("bili_validation_reader", reader_path)
    reader = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(reader)
    if sys.modules["gxjy_decrypt"].KEY != key_file.read_bytes().strip():
        raise ValueError("原型资源密钥与授权本地密钥不一致")
    # 不启用指向另一个历史包名的 GXJY_BUILD=bili，显式指定此次快照
    reader.APK = str(root / "resolved/assets")
    reader.IMPORT = str(root / "resolved/assets/resources/import")
    reader.BUNDLE_CFG = str(root / "resolved/assets/resources/config.json")
    reader._BUNDLE_CACHE = None
    manifest = json.loads((root / "resolved/project.manifest").read_text("utf-8"))
    if manifest["version"] != "1.1.281.1.1":
        raise ValueError("不是本次已核实的 B 服主资源版本")
    files, raw_tables, failures = {}, {}, []

    def checked(path):
        path = Path(path).resolve()
        relative = path.relative_to(root / "resolved").as_posix()
        raw = path.read_bytes()
        row = manifest["assets"][relative]
        if len(raw) != row["size"] or hashlib.md5(raw).hexdigest() != row["md5"]:
            raise ValueError("资源与 manifest 不符：" + relative)
        files[relative] = {"bytes": len(raw), "sha256": hashlib.sha256(raw).hexdigest()}

    checked(reader.BUNDLE_CFG)
    original = reader._load_one
    with tempfile.TemporaryDirectory(prefix="gxjy-bili-tables-") as cache:
        reader.CACHE_DIR = cache

        def load_one(name):
            try:
                path = reader.table_path(name)
                if not path:
                    raise ValueError("缺少配置表：" + name)
                checked(path)
                table = original(name)
                raw_tables[name] = copy.deepcopy(table)
                return table
            except Exception as error:
                failures.append({"table": name, "error": type(error).__name__})
                raise

        reader._load_one = load_one
        tables = {}
        for name in baseline.TABLES:
            tables[name] = reader.load_table(name)
            print(f"loaded B snapshot {name}: {len(tables[name])} rows", flush=True)
    if failures:
        # 原型会吞掉 _subFiles 错误，验收不能将缺表静默当作成功
        raise ValueError("配置读取发生错误：" + json.dumps(failures))

    collisions = []

    def client_merge(name, visiting=()):
        if name in visiting:
            raise ValueError("配置引用循环：" + name)
        table = copy.deepcopy(raw_tables[name])
        parts = [raw_tables[f"{name}_sub{i}"] for i in range(1, int(table.get("_subJsonCount") or 0) + 1)]
        table.pop("_subJsonCount", None)
        for sub in table.pop("_subFiles", {}) or {}:
            parts.append(client_merge(sub, (*visiting, name)))
        for part in parts:
            for key, value in part.items():
                if key.startswith("_"):
                    continue
                if key in table:
                    collisions.append({"table": name, "row": key, "different": table[key] != value})
                table[key] = copy.deepcopy(value)
        return table

    client_differences = {}
    for name, table in tables.items():
        client = client_merge(name)
        differences = [key for key in table.keys() | client.keys() if table.get(key) != client.get(key)]
        if differences:
            client_differences[name] = sorted(differences)
    inventory = []
    for name, table in tables.items():
        fields = {}
        for row in table.values():
            if not isinstance(row, dict):
                continue
            for field, value in row.items():
                kind = type(value).__name__
                fields.setdefault(field, {})[kind] = fields.setdefault(field, {}).get(kind, 0) + 1
        inventory.append({"table": name, "rows": len(table), "field_types": fields})
    return tables, {"version": manifest["version"], "files": files, "inventory": inventory,
                    "merge_collisions": collisions, "client_merge_differences": client_differences,
                    "prototype_sha256": digest(baseline.SOURCE), "reader_sha256": digest(reader_path)}


def make_fixture(project, tables):
    seed = json.loads((project / "internal/game/testdata/existing_paths_python.json").read_text("utf-8"))
    cases = [baseline.make_case(c["name"], c["initial"],
             [(s["command"], s["request"]) for s in c["steps"]], tables, c["options"])
             for c in seed["cases"]]
    visible = [name for name, row in tables["HeroBase"].items()
               if isinstance(row, dict) and row.get("IsShow") == 1]
    all_heroes = {"role_name": "BiliSynthetic", "heroes": visible}
    cases.append(baseline.make_case("bili_all_visible", all_heroes,
                 [(101, {}), (1254, {}), (1301, {})], tables, {"now": 1800000000}))
    advanced = copy.deepcopy(all_heroes)
    advanced["hero_state"] = {}
    for hid in visible:
        row = tables["HeroBase"][hid]
        if len(row.get("Data") or []) > 1 and f"HeroAwaken_{hid}_1" in tables["HeroAwaken"]:
            advanced["hero_state"][hid] = {"level": 7, "quality": 1, "star": 1,
                                           "awaken": 1, "favor": 1, "surface": "invalid-synthetic"}
    cases.append(baseline.make_case("bili_all_advanced", advanced,
                 [(101, {}), (1254, {}), (1254, {})], tables, {"now": 1800000000}))
    # 使用 B 服真实模板，空 user 避免把合成装备分配给不适用的职业
    equipment = {"role_name": "BiliEquipmentSynthetic", "equip_state": {}, "bag_items": {"IEQ_ItemCommon": 3}}
    for index, (tid, row) in enumerate(tables["EquipBase"].items()):
        equipment["equip_state"][f"BILI_EQ_{index}"] = {
            "template_id": tid, "user": "", "level": 7, "quality_num": 1, "star_num": row["InitStar"],
            "exp": 0, "locked": False, "enchantment": [],
        }
    cases.append(baseline.make_case("bili_real_equipment", equipment,
                 [(101, {}), (1254, {}), (1254, {})], tables, {"now": 1800000000}))
    for case in cases:
        reloaded = baseline.make_case("reload", case["steps"][-1]["disk"], [], tables, case["options"])
        case["restart"], case["restart_disk"] = reloaded["startup"], reloaded["startup_disk"]
    keys = {}

    def collect(name, value):
        if isinstance(value, dict):
            keys[name] = list(value)
            for key, child in value.items():
                collect(name + "/" + key, child)

    for name, table in tables.items():
        collect(name, table)
    return {"tables_kind": "bili-281-authorized", "source_sha256": digest(baseline.SOURCE),
            "tables": tables, "table_keys": keys, "cases": cases}


def main():
    project = Path(__file__).resolve().parents[1]
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=project / "data/assets/bili")
    parser.add_argument("--output", type=Path, default=project / "data/assets/bili/analysis/runtime-validation")
    parser.add_argument("--go", default="go")
    parser.add_argument("--node", default="node")
    args = parser.parse_args()
    root, parent = args.root.resolve(), args.output.resolve()
    subprocess.run(["git", "check-ignore", "-q", str(parent / "python-fixture.json")],
                   cwd=project, check=True)
    parent.mkdir(parents=True, exist_ok=True)
    output = Path(tempfile.mkdtemp(prefix="run-", dir=parent))
    def protected():
        paths = [project / "config.json", *sorted((project / "data/save").rglob("*"))]
        return {str(path.relative_to(project)): digest(path) for path in paths if path.is_file()}
    unchanged = protected()
    tables, evidence = load_bili_tables(root, project / "data/keys/asset.key")
    write_new_or_identical(output / "resource-evidence.json", evidence)
    fixture = make_fixture(project, tables)
    write_new_or_identical(output / "python-fixture.json", fixture)
    config = {"http_host": "127.0.0.1", "http_port": 18080,
              "ws_host": "127.0.0.1", "ws_port": 18765, "gate_host": "127.0.0.1",
              "save_dir": "go-saves", "asset_root": str(root / "resolved/assets"),
              "bundle_config": str(root / "resolved/assets/resources/config.json"),
              "asset_key_env": "GXJY_BILI_TEST_ASSET_KEY", "rsa_pem": "", "rsa_json": "",
              "protocol_map": str(root / "analysis/verified/protocol_map.json")}
    write_new_or_identical(output / "config.json", config)
    client_command = [args.node, str(project / "scripts/check_bili_runtime_client.cjs"),
                      str(root / "analysis/verified/index.js"), str(output / "python-fixture.json"),
                      str(output / "client-expectations.json")]
    subprocess.run(client_command, cwd=project, check=True)
    env = os.environ.copy()
    env["GXJY_BILI_TEST_ASSET_KEY"] = (project / "data/keys/asset.key").read_text("utf-8").strip()
    env["GXJY_BILI_RUNTIME_CONFIG"] = str(output / "config.json")
    env["GXJY_BILI_RUNTIME_FIXTURE"] = str(output / "python-fixture.json")
    env["GXJY_BILI_RUNTIME_REPORT"] = str(output / "go-runtime-report.json")
    env["GXJY_BILI_CLIENT_EXPECTATIONS"] = str(output / "client-expectations.json")
    env["GXJY_DIFF_FIXTURE"] = str(output / "python-fixture.json")
    env["GXJY_DIFF_ASSET_ROOT"] = config["asset_root"]
    env["GXJY_DIFF_BUNDLE"] = config["bundle_config"]
    env["GXJY_ASSET_KEY"] = env["GXJY_BILI_TEST_ASSET_KEY"]
    env["GOCACHE"] = str(root / "analysis/go-cache")
    result = subprocess.run([args.go, "test", "./internal/game", "./internal/assets", "-run",
                             "^TestBiliRuntimeMatchesPython$|^TestActualTablesMatchPython$", "-count=1", "-v"],
                            cwd=project, env=env, text=True, encoding="utf-8", stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    write_new_or_identical(output / "protected-files.json", {"files_checked": len(unchanged),
                          "unchanged": protected() == unchanged})
    log = output / "go-test.log"
    log.write_text(result.stdout, "utf-8")
    print(result.stdout, end="")
    report = output / "go-runtime-report.json"
    consumed = None
    if report.exists():
        consumed = subprocess.run([*client_command, str(report)], cwd=project).returncode
    print(json.dumps({"tables": len(tables), "source_files": len(evidence["files"]),
                      "cases": len(fixture["cases"]), "steps": sum(len(c["steps"]) for c in fixture["cases"]),
                      "client_merge_differences": evidence["client_merge_differences"], "go_exit_code": result.returncode,
                      "client_consume_exit_code": consumed,
                      "output": str(output)}, ensure_ascii=False))
    if protected() != unchanged:
        raise ValueError("正常配置或存档在验证期间发生变化")
    sys.exit(result.returncode or consumed or bool(evidence["client_merge_differences"]))


if __name__ == "__main__":
    # Windows 的默认控制台编码可能是 GBK，Go 日志和报告均使用 UTF-8
    sys.stdout.reconfigure(encoding="utf-8")
    sys.stderr.reconfigure(encoding="utf-8")
    main()
