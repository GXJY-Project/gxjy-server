"""用散列固定的 Python 看板原函数生成合成差分，不读取真实存档"""

import copy
import hashlib
import json
from pathlib import Path

import python_baseline as baseline


def main():
    digest = hashlib.sha256(baseline.SOURCE.read_bytes()).hexdigest()
    if digest != "13ef3e5dfcda8ed996693711c5c573df533b41b89d812d49ce4a132fc49bda25":
        raise ValueError("原型已变化，需重新核对看板分支")
    tables = baseline.synthetic_tables()
    tables["BackGroundHome"] = {"BG1": {"Id": "BG1"}, "BG2": {"Id": "BG2"}, "_meta": {}, "not-a-row": 1}
    tables["HeroBase"]["Hidden"] = {"Id": "Hidden", "IsShow": 0}
    requests = [
        (1005, {}), (1006, {"hero_id": "H004", "back_id": "BG2", "mode": 1}),
        (1006, {"mode": 0}), (1006, {"mode": False}),
        (1006, {"hero_id": "Missing", "back_id": "", "mode": None}),
        (1006, {"hero_id": "Hidden"}), (1006, {"hero_id": "H011"}),
        (1006, {"msg": {"hero_id": "H002"}, "seq": 9}),
        (1006, None), (1006, {}), (1005, {}),
    ]
    options = {"now": 1800000000}
    saved = {"level": 3, "wife": "H002", "back_now": "BG1", "mode": 1,
             "currency": {"IC_Stamina": 1}, "stamina_at": options["now"] - 721,
             "unknown": {"keep": [None, False, 9007199254740993]}}
    cases = [baseline.make_case("new-board", None, requests, tables, options),
             baseline.make_case("saved-board", saved, requests, tables, options)]
    no_unlock = copy.deepcopy(tables)
    no_unlock["UnlockSystem"] = {}
    empty_case = baseline.make_case("board-no-system-opens", saved, [(1006, {"mode": 0})], no_unlock, options)
    empty_case["tables_override"] = {"UnlockSystem": {}}
    empty_case["table_keys_override"] = {"UnlockSystem": []}
    cases.append(empty_case)
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
    path = Path(__file__).resolve().parents[1] / "internal/game/testdata/board_python.json"
    path.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"generated {len(cases)} cases, {sum(len(c['steps']) for c in cases)} board steps")


if __name__ == "__main__":
    main()
