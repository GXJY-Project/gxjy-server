"""在临时存档上执行原型探索分支，只导出合成数据差分"""

import ast
import hashlib
import json
from pathlib import Path

import task_baseline
import python_baseline as baseline


original_namespace = baseline.namespace


def explore_namespace(tables, directory, options):
    ns, choices = original_namespace(tables, directory, options)
    names = {"_MEN_DOOR_MECH_CASES", "_MEN_DOOR_OPEN_CASE", "_MEN_BOSS_CASE"}
    tree = ast.parse(baseline.SOURCE.read_text(encoding="utf-8-sig"))
    for node in tree.body:
        if isinstance(node, ast.Assign) and any(isinstance(t, ast.Name) and t.id in names for t in node.targets):
            exec(compile(ast.Module(body=[node], type_ignores=[]), str(baseline.SOURCE), "exec"), ns)
    ns["_initial_scene"] = None
    return ns, choices


baseline.namespace = explore_namespace


def main():
    digest = hashlib.sha256(baseline.SOURCE.read_bytes()).hexdigest()
    if digest != task_baseline.EXPECTED_SOURCE:
        raise ValueError("原型变化，先核对探索调用链")
    tables = baseline.synthetic_tables()
    tables.update({name: {} for name in ("ExploreScene", "ExploreObject", "ExploreCaseDo", "ExploreCaseEnd", "ExploreDPTask", "TaskBase", "ItemConfig", "HeroLevel")})
    tables["BaseConfig"]["Initial_ExploreScene"] = {"Config": "Scene_Test"}
    tables["ExploreScene"] = {
        "Scene_Test": {"InitialPos": "Spawn"},
        "Scene_Community": {"PlayerPos": [{"z": {"Pos": [9, 9]}, "a": {"Pos": [3, 4], "Face": "Left"}}], "NormalObject": ["Legacy" + str(i) for i in range(8)]},
        "Scene_SP_0": {"InitialPos": "Spawn"},
    }
    tables["ExploreObject"] = {
        "Spawn": {"Transfer": {"Pos": [10, 20], "Face": "Left"}},
        "Chest": {"Scene": "Scene_Test", "Position": [[30, 40]], "Cases": ["OpenChest"], "CaseDoPassword": ["Key"]},
        "Switch": {"Scene": "Scene_Test", "Position": [{"z": [9, 9], "a": [50, 60]}], "Cases": ["SwitchA", "SwitchB", "SwitchC"]},
        "TaskObject": {"Scene": "Scene_Test", "Position": [[70, 80]]},
        "Deleted": {"Scene": "Scene_Test", "Position": [[1, 1]], "IsDel": True},
        "BadXY": {"Scene": "Scene_Test", "Position": [[None, "bad"]]},
        "Other": {"Scene": "Other", "Position": [[1, 1]]},
        **{"Legacy" + str(i): {"Position": [[i, i + 1]]} for i in range(8)},
    }
    tables["ExploreCaseDo"] = {
        "OpenChest": {"CaseEnd": {"End": ["GiveGold"]}},
        "SwitchA": {"CaseEnd": {"End": ["StateA"]}},
        "SwitchB": {"CaseEnd": {"Win": ["StateB"]}},
        "SwitchC": {"CaseEnd": {"End": []}},
        "TaskCase": {"CaseEnd": {"End": []}},
    }
    tables["ExploreCaseEnd"] = {
        "GiveGold": {"Order": "GetReward", "Value1": "ChestReward"},
        "StateA": {"Order": "ChangeSpecialState1", "Value2": {"ObjectID": "Switch", "SpecialState": 1}},
        "StateB": {"Order": "ChangeSpecialState1", "Value2": {"SpecialState": "2"}},
    }
    tables["RewardConfig"]["ChestReward"] = {"Content": [{"type": 2, "code": "IC_Gold", "amount": 7}]}
    tables["TaskBase"] = {
        "Daily_Explore": {"Type": "DAILY", "Condition": {"TaskCondition": "TC_Explore_CaseDo", "factor": 1, "params": ["TaskCase"]}, "Reward": [], "ExploreCaseDo": [{"ObjectID": "TaskObject", "CaseDoID": ["TaskCase", "TaskCase"]}]},
    }
    tables["ExploreDPTask"] = {
        "DP1": {"WorldRegion": "Region", "CaseDoID": ["OpenChest", "SwitchA", "SwitchB"], "CaseDoNum": 2},
        "DP2": {"WorldRegion": "Region", "CaseDoID": ["TaskCase"], "CaseDoNum": 0},
    }
    options = {"now": 1800000000}
    commands = [(101, {}), (1852, {"scene_id": "Scene_Test", "pos": {"x": 123, "y": 456}}),
                (1851, {}), (1853, {"pos": {"x": 124, "y": 457}}), (1854, {"scene_id": "Scene_Test"}),
                (1863, {}), (1864, {}), (1857, {"case_do_id": "OpenChest"}),
                (1857, {"case_do_id": "OpenChest"}), (1857, {"case_do_id": "TaskCase"}),
                (1852, {"msg": {"scene_id": "Scene_Community"}, "seq": 9}), (1854, None), (1852, None)]
    cases = [baseline.make_case("new-scene", None, commands, tables, options)]
    ordered = {"scene_id": "Scene_Test", "cases": {"SwitchB": False, "SwitchA": True}, "currency": {"IC_Gold": 20}}
    cases.append(baseline.make_case("ordered-false-case", ordered, [(1851, {}), (1852, {}), (1857, {"case_do_id": "SwitchC"}), (1851, {})], tables, options))
    cases[-1]["initial_json"] = json.dumps(ordered)
    mechanisms = {"SP_0_MenA_KaiGun1_Do1": True, "SP_0_MenA_KaiGun2_Do1": True, "SP_0_MenA_KaiGun_Do2": True}
    for boss in (False, True):
        initial = {"scene_id": "Scene_SP_0", "cases": {**mechanisms, "SP_0_Master_ShiLaiMu_Do1": boss}}
        cases.append(baseline.make_case("door-boss-" + str(boss), initial, [(1852, {}), (1851, {})], tables, options))
        cases[-1]["initial_json"] = json.dumps(initial)
    keys = {}

    def collect(path, value):
        if isinstance(value, dict):
            keys[path] = list(value)
            for key, child in value.items():
                collect(path + "/" + key, child)

    for name, table in tables.items():
        collect(name, table)
    result = {"source": str(baseline.SOURCE.relative_to(baseline.ROOT)), "source_sha256": digest,
              "tables_kind": "synthetic", "tables": tables, "table_keys": keys, "cases": cases}
    path = Path(__file__).resolve().parents[1] / "internal/game/testdata/explore_python.json"
    path.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"generated {len(cases)} cases, {sum(len(c['steps']) for c in cases)} explore steps")


if __name__ == "__main__":
    main()
