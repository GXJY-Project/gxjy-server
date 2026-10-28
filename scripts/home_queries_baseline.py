"""从原型分支生成主界面查询差分，只使用合成配置和临时存档"""

import hashlib
import json
from pathlib import Path

import python_baseline as baseline


def main():
    digest = hashlib.sha256(baseline.SOURCE.read_bytes()).hexdigest()
    if digest != "13ef3e5dfcda8ed996693711c5c573df533b41b89d812d49ce4a132fc49bda25":
        raise ValueError("原型版本变化，先核对主界面查询分支")
    tables = baseline.synthetic_tables()
    commands = [
        (1002, {}), (1307, {}), (1502, {"start": 1, "stop": 20, "open": True}),
        (1955, {}), (2401, {}), (10102, {"start": 1, "stop": 20}),
        (10152, {}), (10157, {}), (10167, {"name": "synthetic"}), (10175, {}),
        (1752, {}), (1752, {"id": "Active"}), (1752, {"month_card_id": "Expired"}),
        (1752, {"card_id": "Boundary"}), (1752, {"id": "Invalid"}),
        (1752, {"id": "Boolean"}), (1752, {"id": "Active", "card_id": "Expired"}),
        (1752, {"msg": {"id": "Active"}, "seq": 9}),
        (1752, None), (1307, None),
    ]
    options = {"now": 1800000000}
    initial = {"month_card": {"Active": {"until": options["now"] + 60},
                               "Expired": {"until": options["now"] - 1},
                               "Boundary": {"until": options["now"]},
                               "Invalid": {"until": "invalid"}, "Boolean": {"until": True}},
               "patrol_auto": {"today_received": True}, "summit": {"rank": 3, "score": 456},
               "unknown": {"precise": 9007199254740993}}
    cases = [baseline.make_case("new-home", None, commands, tables, options),
             baseline.make_case("saved-home", initial, commands, tables, options)]
    keys = {}

    def collect(name, value):
        if isinstance(value, dict):
            keys[name] = list(value)
            for key, child in value.items():
                collect(name + "/" + key, child)

    for name, table in tables.items():
        collect(name, table)
    result = {"source": str(baseline.SOURCE.relative_to(baseline.ROOT)), "source_sha256": digest,
              "tables_kind": "synthetic", "tables": tables, "table_keys": keys, "cases": cases}
    path = Path(__file__).resolve().parents[1] / "internal/game/testdata/home_queries_python.json"
    path.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"generated {len(cases)} cases, {sum(len(c['steps']) for c in cases)} home query steps")


if __name__ == "__main__":
    main()
