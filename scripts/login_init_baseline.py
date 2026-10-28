"""生成登录初始化查询的原型差分，只使用合成配置与临时存档"""

import ast
import copy
import hashlib
import json
from pathlib import Path

import python_baseline as baseline


EXPECTED_SOURCE = "13ef3e5dfcda8ed996693711c5c573df533b41b89d812d49ce4a132fc49bda25"


def main():
    source_hash = hashlib.sha256(baseline.SOURCE.read_bytes()).hexdigest()
    if source_hash != EXPECTED_SOURCE:
        raise ValueError("原型版本变化，先核对初始化查询分支")
    tree = ast.parse(baseline.SOURCE.read_text(encoding="utf-8-sig"))
    constants = {}
    for node in tree.body:
        if isinstance(node, ast.Assign):
            for target in node.targets:
                if isinstance(target, ast.Name) and target.id in {"GACHA_BASE_SIX", "GACHA_PITY_LIMIT"}:
                    constants[target.id] = ast.literal_eval(node.value)
    original_namespace = baseline.namespace

    def namespace(tables, directory, options):
        ns, choices = original_namespace(tables, directory, options)
        ns.update(constants)
        ns["DEBUG_POOL_LIMIT"] = 0
        return ns, choices

    baseline.namespace = namespace
    tables = baseline.synthetic_tables()
    tables["DrawCard"] = {
        "AAA_Limited": {"Type": "limited", "CountType": "Limit", "Coupon": ["TicketL"], "CouponCost": [1, 10], "Number": [1, 10]},
        "ZZZ_Normal": {"Type": "normal", "CountType": "New", "Coupon": ["TicketN"], "CouponCost": [1], "Number": [1]},
    }
    tables["ResourcesMap"] = {"TrialA": {"ResourcesPoint": ["TrialPoint"]}}
    tables["ResourcesPoint"]["TrialPoint"]["Id"] = "TrialPoint"
    tables["TaskBase"] = {}
    tables["ShopMonthCard"] = {"ShopMonthCard_01": {}, "ShopMonthCard_Permanent": {}}
    tables["BaseConfig"]["Resources_DailyNum"] = {"Config": 3}
    for table, key in [("ArenaBase", "Arena_Season1"), ("ArenaRealTimeBase", "ArenaRealTime_1"), ("ArenaFairPlayBase", "ArenaFairPlay_Week1")]:
        tables[table] = {key: {"Id": key}}
    requests = [
        (101, {}), (1011, None), (10115, {}), (1254, {"start": 0, "stop": 50, "version": 0}),
        (1552, {"version": 0}), (1670, {}),
        (10101, {"start": 1, "stop": 20, "type": 0}),
        (10124, {"start": 1, "stop": 20}), (10129, {"chat_id": 10001}),
        (2016, {"show_error": False, "version": "synthetic"}),
        (10151, {"version": 0}), (2101, {}), (1151, {"version": 0}),
        (1601, {}), (1701, {}), (1810, {}), (1802, {}),
        (2351, {"req_pool": True}), (1951, {}),
        (905, {"track_type": "synthetic", "track_info": {}}),
        (901, {"clientTimestamp": 123}),
    ]
    initial = {"gacha": {"total": 23, "pity": 7, "pool_times": {"AAA_Limited": 3, "ZZZ_Normal": 20}},
               "resource_pass": {"TrialPoint": {"achievement": [1, 3], "pass_num": 4}}}
    cases = [baseline.make_case("new", None, requests, tables, {"now": 1800000000}),
             baseline.make_case("saved-query-state", initial, requests, tables, {"now": 1800000000})]
    # 维护者已授权以 B224 为准，仅这两项预期与 Python 原输出不同
    # 原输出同时保留，避免把目标客户端适配误报成逐字节原型等价
    for case in cases:
        for step in case["steps"]:
            if step["command"] in (10129, 2016):
                step["python_messages"] = copy.deepcopy(step["messages"])
                response = step["messages"][0]["payload"]
                if step["command"] == 10129:
                    del response["chat_list"]
                    response["contactList"] = {}
                else:
                    del response["rooms"]
                    response["room_info"] = None
    keys = {}

    def collect(name, value):
        if isinstance(value, dict):
            keys[name] = list(value)
            for key, child in value.items():
                collect(name + "/" + key, child)

    for name, table in tables.items():
        collect(name, table)
    artifact = {"source": str(baseline.SOURCE.relative_to(baseline.ROOT)), "source_sha256": source_hash,
                "tables_kind": "synthetic", "client_target": "bili-224",
                "client_adaptations": {"10129": "contactList:{}", "2016": "room_info:null"},
                "tables": tables, "table_keys": keys, "cases": cases}
    output = Path(__file__).resolve().parents[1] / "internal/game/testdata/login_init_python.json"
    output.write_text(json.dumps(artifact, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"generated {len(cases)} cases, {sum(len(c['steps']) for c in cases)} initialization steps")


if __name__ == "__main__":
    main()
