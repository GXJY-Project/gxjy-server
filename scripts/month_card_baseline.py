"""复用原 Session.dispatch 生成月卡查询差分，合成存档只写入临时目录"""

import argparse
import copy
import hashlib
import json
from pathlib import Path
import subprocess

import python_baseline as baseline


EXPECTED_SOURCE = "13ef3e5dfcda8ed996693711c5c573df533b41b89d812d49ce4a132fc49bda25"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=baseline.ROOT / "gxjy-server/internal/game/testdata/month_card_python.json")
    parser.add_argument("--real-assets", action="store_true")
    parser.add_argument("--bili-assets", action="store_true")
    args = parser.parse_args()
    source_hash = hashlib.sha256(baseline.SOURCE.read_bytes()).hexdigest()
    if source_hash != EXPECTED_SOURCE:
        raise ValueError("Python 原型版本已变化，请先核对月卡分支")
    if args.real_assets and args.bili_assets:
        raise ValueError("一次只验证一个资源来源")
    real = args.real_assets or args.bili_assets
    if real:
        if subprocess.run(["git", "check-ignore", "-q", str(args.output.resolve())], cwd=baseline.ROOT).returncode != 0:
            raise ValueError("真实资源差分只能写入 Git 忽略目录")
        baseline.TABLES = [*baseline.TABLES, "ShopMonthCard"]
        if args.bili_assets:
            from check_bili_runtime import load_bili_tables
            project = baseline.ROOT / "gxjy-server"
            tables, evidence = load_bili_tables(project / "data/assets/bili", project / "data/keys/asset.key")
        else:
            tables = baseline.real_tables()
            evidence = {}
    else:
        tables = baseline.synthetic_tables()
        tables["ShopMonthCard"] = {
        "ShopMonthCard_01": {}, "ShopMonthCard_Permanent": {"IsShow": "0"},
        "SyntheticNumericZero": {"IsShow": 0}, "SyntheticFalse": {"IsShow": False},
        "SyntheticNegative": {"IsShow": -1}, "SyntheticInvalid": {"IsShow": "invalid"},
        "SyntheticFloat": {"IsShow": 0.75}, "SyntheticEmpty": {"IsShow": []},
        "SyntheticDict": {"IsShow": {"bad": 1}},
        "_subJsonCount": 0, "_synthetic_meta": {}, "SyntheticNonRow": 7,
        }
    options = {"now": 1800000000}
    requests = [(101, {}), (1670, {}), (1670, {"msg": None, "seq": 2}), (901, {"clientTimestamp": 123})]
    saved = {"month_card": {
        "ShopMonthCard_01": {"until": " 1800000001 "},
        "ShopMonthCard_Permanent": {"until": 1700000000},
        "SyntheticNumericZero": {"until": 1.75}, "SyntheticFalse": {"until": True},
        "SyntheticNegative": {"until": -1}, "SyntheticInvalid": {"until": "1.5"},
        "SyntheticFloat": {"until": {"bad": 1}}, "SyntheticEmpty": [],
        "SyntheticDict": {"until": []},
    }}
    cases = [baseline.make_case("new", None, requests, tables, options),
             baseline.make_case("saved-and-boundaries", saved, requests, tables, options),
             baseline.make_case("empty-state", {"month_card": {}}, [(1670, [])], tables, options)]
    # 独立配置输入确保空表回退预期取自原型，不能从 Go 输出生成
    if not real:
        fallback = copy.deepcopy(tables)
        fallback["ShopMonthCard"] = {"_meta": {}, "NotARow": False}
        fallback_case = baseline.make_case("empty-table", saved, requests, fallback, options)
        fallback_case["tables_override"] = {"ShopMonthCard": fallback["ShopMonthCard"]}
        fallback_case["table_keys_override"] = {"ShopMonthCard": list(fallback["ShopMonthCard"])}
        cases.append(fallback_case)
    keys = {}
    def collect(name, value):
        if isinstance(value, dict):
            keys[name] = list(value)
            for key, child in value.items():
                collect(name + "/" + key, child)
    for name, table in tables.items():
        collect(name, table)
    fixture = {"source": str(baseline.SOURCE.relative_to(baseline.ROOT)), "source_sha256": source_hash,
               "tables_kind": "bili-281-authorized" if args.bili_assets else "real-assets" if args.real_assets else "synthetic",
               "tables": tables, "table_keys": keys, "cases": cases}
    if args.bili_assets:
        fixture["asset_evidence"] = evidence
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(fixture, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"generated {len(cases)} month-card cases, {sum(len(c['steps']) for c in cases)} steps")


if __name__ == "__main__":
    main()
