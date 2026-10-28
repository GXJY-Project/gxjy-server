"""执行原任务查询调用链，生成合成或本地授权资源差分，不读取真实存档"""

import argparse
import ast
import copy
import hashlib
import json
from pathlib import Path
import random
import re
import subprocess

import python_baseline as baseline


EXPECTED_SOURCE = "13ef3e5dfcda8ed996693711c5c573df533b41b89d812d49ce4a132fc49bda25"
TASK_TABLES = "TaskBase ItemConfig HeroLevel ShopMonthCard".split()
original_namespace = baseline.namespace


def task_namespace(tables, directory, options):
    ns, choices = original_namespace(tables, directory, options)
    # 原差分入口只提取登录常量，任务调用链必须补入真实章节门和奖励路由
    tree = ast.parse(baseline.SOURCE.read_text(encoding="utf-8-sig"))
    names = set("CHAPTER_ORDER CHAPTER_EXEMPT KEXP_TYPE KSPECIAL_TYPE EXP_CODE_ROUTE SPECIAL_CODE_ROUTE PLAYER_EXP_RATE _BANDED_TYPES".split())
    for node in tree.body:
        if isinstance(node, ast.Assign) and any(isinstance(t, ast.Name) and t.id in names for t in node.targets):
            exec(compile(ast.Module(body=[node], type_ignores=[]), str(baseline.SOURCE), "exec"), ns)
    ns["re"] = re
    ns["random"] = random.Random(1)
    ns["random"].choice = lambda seq: seq[0]
    return ns, choices


baseline.namespace = task_namespace


def synthetic_tables():
    tables = baseline.synthetic_tables()
    tables.update({name: {} for name in TASK_TABLES})
    def row(kind="Unknown", factor=1, params=None, **extra):
        return {"Type": "STORYMAIN", "Condition": {"TaskCondition": kind, "factor": factor, "params": params or []}, "Reward": [], **extra}
    # 顺序故意逆转，覆盖多轮推进、章节门和同线 BFS 当前节点
    tables["TaskBase"] = {
        "SM01T01_200": row("TC_TaskFinish", params=["SM01T01_100"], PreTask=["SM01T01_100"]),
        "SM01T01_100": row("TC_Level", 2),
        "SMSPT01_400": row(PreTask=["SMSPT01_300"]),
        "SMSPT01_300": row("TC_Explore_CaseDo", params=["CaseA"], PreTask=["SMSPT01_200"], ExploreTaskTarget=[1, 2, 3]),
        "SMSPT01_200": row("TC_PassStage", params=["P0"], PreTask=["SMSPT01_100"]),
        "SMSPT01_100": row("TC_Level", 2, Reward=["TaskReward"]),
        "SM99T01_100": row("TC_Level", 1),
        "SM05TE3_0100": row("TC_Level", 1),
        "DailyTask_001": row("TC_Level", 1, Type="DAILY"),
        "DailyTask_002": row("TC_TeamMember_Count", 2, [1], Type="DAILY"),
        "DailyTask_NoParams": row("TC_TeamMember_Count", 100, Type="DAILY"),
        "DailyTask_Unsupported": row("TC_ConsumeItem", 1, Type="DAILY"),
        "DailyTask_NoReward": {"Type": "DAILY", "Condition": {"factor": 1}},
        "DailyTask_NoFactor": {"Type": "DAILY", "Reward": [], "Condition": {}},
        "DailyTask_NoType": {"Reward": [], "Condition": {"factor": 1}},
        "_meta": {}, "NotRow": 123,
    }
    tables["RewardConfig"]["TaskReward"] = {"Content": [
        {"type": 999, "code": "NestedTaskReward", "amount": 3},
        {"type": 2, "code": "IC_Gold", "amount": 7},
        {"type": 1, "code": "IC_PlayerExp", "amount": 1},
        {"type": 1, "code": "IC_HeroExp", "amount": 2},
        {"type": 1, "code": "IC_EquipExp", "amount": 3},
        {"type": 10, "code": "0", "amount": 4},
    ]}
    tables["RewardConfig"]["NestedTaskReward"] = {"Content": [
        {"z-first": [{"type": 2, "code": "Potion", "amount": 2}], "a-second": [{"type": 10, "code": "Potion", "amount": 1}]},
        {"type": 4, "code": "Equip_S3_1", "amount": 1},
        {"type": 2, "code": "Potion", "amount": 3},
        {"type": 2, "code": "NoGrant", "amount": -1},
    ]}
    tables["PlayerLevel"]["2"].update({"PlayerExp": 50, "GetActionPoint": 10, "LevelReward": "LevelReward"})
    tables["RewardConfig"]["LevelReward"] = {"Content": [{"type": 2, "code": "LevelPotion", "amount": 1}]}
    tables["HeroLevel"] = {"1": {"Exp": {"1": 2}}}
    return tables


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path, default=baseline.ROOT / "gxjy-server/internal/game/testdata/tasks_python.json")
    parser.add_argument("--real-assets", action="store_true")
    parser.add_argument("--bili-assets", action="store_true")
    args = parser.parse_args()
    source_hash = hashlib.sha256(baseline.SOURCE.read_bytes()).hexdigest()
    if source_hash != EXPECTED_SOURCE:
        raise ValueError("原型版本变化，须先核对任务调用链")
    if args.real_assets and args.bili_assets:
        raise ValueError("一次只验证一个资源来源")
    real = args.real_assets or args.bili_assets
    if real:
        if subprocess.run(["git", "check-ignore", "-q", str(args.output.resolve())], cwd=baseline.ROOT).returncode:
            raise ValueError("真实资源样本只能写入 Git 忽略目录")
        baseline.TABLES = [*baseline.TABLES, *TASK_TABLES]
        if args.bili_assets:
            from check_bili_runtime import load_bili_tables
            project = baseline.ROOT / "gxjy-server"
            tables, evidence = load_bili_tables(project / "data/assets/bili", project / "data/keys/asset.key")
        else:
            tables, evidence = baseline.real_tables(), {}
    else:
        tables, evidence = synthetic_tables(), {}
    options = {"now": 1800000000}
    requests = [(101, {}), (10115, {}), (1670, {}), (1552, {"version": -1}), (1552, {"msg": None, "seq": 4}), (1254, {}), (901, {})]
    saved = {"level": 2, "exp": 5, "heroes": ["H004", "H002"], "currency": {"IC_Gold": 3, "IC_Stamina": 60},
             "stamina_at": options["now"], "points": {"P0": {"pass": True}}, "cases": {"CaseA": False},
             "tasks": {"SMSPT01_300": {"status": 0, "process": 1, "reset_time": 42}}}
    cases = [baseline.make_case("new-login-task-heartbeat", None, requests, tables, options),
             baseline.make_case("advance-reward-and-repeat", saved, requests, tables, options),
             baseline.make_case("saved-status-and-guide", {"tasks": {"SMSPT01_100": {"status": 1}, "SMSPT01_200": {"status": 2},
                                                                   "SMSPT01_300": {"status": 0, "process": 99, "reset_time": 0}}}, [(1552, [])], tables, options)]
    if not real:
        empty = copy.deepcopy(tables)
        empty["TaskBase"] = {}
        case = baseline.make_case("empty-table", {}, [(1552, None), (901, {})], empty, options)
        case["tables_override"] = {"TaskBase": {}}
        cases.append(case)
        ordered = copy.deepcopy(saved)
        ordered["equip_state"] = {
            "Z_FirstSaved": {"template_id": "Equip_S3_1", "level": 1, "exp": 0},
            "A_SecondSaved": {"template_id": "Equip_S3_2", "level": 1, "exp": 0},
        }
        cases.append(baseline.make_case("equipment-exp-keeps-saved-order", ordered, requests, tables, options))
        limited = copy.deepcopy(tables)
        limited["TaskBase"] = {f"SyntheticChainT01_{n:02}": {
            "Type": "STORYBRANCH", "Condition": {"TaskCondition": "TC_Level", "factor": 1},
            "Reward": [], "PreTask": [f"SyntheticChainT01_{n-1:02}"] if n > 0 else [],
        } for n in reversed(range(9))}
        case = baseline.make_case("six-round-limit-and-repeat", {}, [(1552, {}), (1552, {})], limited, options)
        case["tables_override"] = {"TaskBase": limited["TaskBase"]}
        cases.append(case)
    for case in cases:
        case["table_keys_override"] = {name: list(value) for name, value in case.get("tables_override", {}).items()}
        if case["initial"] is not None:
            case["initial_json"] = json.dumps(case["initial"], ensure_ascii=False)
    keys = {}
    def collect(name, value):
        if isinstance(value, dict):
            keys[name] = list(value)
            for key, child in value.items():
                collect(name + "/" + key, child)
        elif isinstance(value, list):
            for index, child in enumerate(value):
                collect(name + "/" + str(index), child)
    for name, value in tables.items():
        collect(name, value)
    fixture = {"source": str(baseline.SOURCE.relative_to(baseline.ROOT)), "source_sha256": source_hash,
               "tables_kind": "bili-281-authorized" if args.bili_assets else "real-assets" if real else "synthetic",
               "tables": tables, "table_keys": keys, "cases": cases, "asset_evidence": evidence}
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(fixture, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"generated {len(cases)} task cases, {sum(len(c['steps']) for c in cases)} steps")


if __name__ == "__main__":
    main()
