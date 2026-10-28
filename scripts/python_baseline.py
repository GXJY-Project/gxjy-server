"""只读提取原 Python 函数，在临时存档和显式配置上生成行为样本"""

import argparse
import ast
import copy
import gzip
import importlib.util
import hashlib
import json
import os
from pathlib import Path
import random
import shutil
import sys
import tempfile
import time
import types

ROOT = Path(__file__).resolve().parents[2]
SOURCE = ROOT / "gxjy-release-20261019/gxjy-release-20261019/2-SERVER/protocol/gxjy_server.py"
TABLES = "HeroBase HeroData HeroQuality HeroStar HeroAwaken HeroFavor SkillBase SkillSpecial LeaderBase LeaderTeamSkill HeroCostAttr BaseConfig EquipBase EquipExp EquipQuality EquipStar SkillAttr PlayerLevel UnlockSystem ExploreWorldMap ExploreWorldRegion ExploreWorldMain ResourcesPoint ResourcesMap FightMap FightPoint RewardConfig DrawCard Activity SummitBase NameFront NameMiddle NameLast".split()


def synthetic_tables():
    tables = {name: {} for name in TABLES}
    for i, hid in enumerate(["H002", "H004", "H011"]):
        leader = "LeaderHero_" + hid
        tables["HeroBase"][hid] = {
            "Id": hid, "IsShow": 1, "Leader": leader if i else "", "Data": [hid + "_Base", hid + "_Awake"],
            "Rarity": "R3", "FirstCost": 3, "Atk": 5, "Hp": 20, "AtkFactor": 1.5,
            "Quality": "HeroQuality_" + hid + "_0", "Star": "HeroStar_" + hid + "_0",
            "Awaken": "HeroAwaken_" + hid + "_0", "Favor": hid + "_Favor0", "NormalSkill": "Skill_Normal", "PassiveSkill": "Skill_Passive",
        }
        for surface in tables["HeroBase"][hid]["Data"]:
            tables["HeroData"][surface] = {"Id": surface}
        if i:
            tables["LeaderBase"][leader] = {"Id": leader, "HeroID": hid, "Data": [hid + "_Base"], "NormalSkill": "Skill_Normal", "LeaderTeamSkill": [leader + "_Team"], "LeaderAtkFactor": 2}
            tables["LeaderTeamSkill"][leader + "_Team"] = {"TeamSkill": "Skill_Team"}
        for q in [0, 1]:
            tables["HeroQuality"][f"HeroQuality_{hid}_{q}"] = {"AtkQualityBase": 10 + q, "AtkQualityStep": 2, "HpQualityBase": 25}
            tables["HeroStar"][f"HeroStar_{hid}_{q}"] = {"AtkStarBase": q, "AtkStarStep": q, "LeaderSpecialBonus": "Skill_Star" if q else ""}
            tables["HeroAwaken"][f"HeroAwaken_{hid}_{q}"] = {"SpecialBonus": ["Skill_Unlock"] if q else []}
            tables["HeroFavor"][hid + f"_Favor{q}"] = {"NextId": hid + "_Favor1" if q == 0 else ""}
    tables["SkillBase"] = {name: {"Name": name, "Attr": {}, "Tag": [], "Args": 0, "Icon": "", "Desc": None} for name in ["Skill_Normal", "Skill_Passive", "Skill_Team", "Skill_Star"]}
    tables["SkillBase"]["Skill_Unlock"] = {"Special": ["UnlockPassive"]}
    tables["SkillSpecial"]["UnlockPassive"] = {"EffectType": "UnlockSkill", "Value": {"skill": "Skill_Passive"}}
    tables["HeroCostAttr"]["3"] = {"BaseAtk": 1.2}
    tables["BaseConfig"] = {
        "Player_DeckTotalNum": {"Config": 4}, "TeamSynchro": {"Config": ["NORMAL", "SPECIAL"]},
        "Hero_RarityFactor": {"Config": {"R3": {"Atk": 1.1}}},
        "Equip_Attribute": {"Config": {"Atk": 1, "Hp": 1, "PhyDef": 1, "MagDef": 1}},
        "Equip_RarityFactor": {"Config": {"R3": 2}}, "Initial_Storey": {"Config": "SMSP_T00_0100_Story010"},
        "Point_Initail_Auto_Reward": {"Config": {"Show": [{"item": "Gold", "amount": 2}, {"code": "Exp", "amount": "3"}, {"item": "Zero", "amount": 0}]}},
    }
    for i in [1, 2, 3]:
        tpl = f"Equip_S3_{i}"
        tables["EquipBase"][tpl] = {"Id": tpl, "Atk": i, "Rarity": "R3", "AtkFactor": 2, "HeroType": ["All"], "IsShow": 1, "Order": i}
        tables["EquipExp"][tpl] = {"NextId": tpl + "_2"}
        tables["EquipExp"][tpl + "_2"] = {}
        for q in [0, 1]:
            tables["EquipQuality"][f"EQE_{tpl}_{q}"] = {"QualityAttr": {"Atk": [2 + q, 1], "Hp": 4, "Invalid": 99}, "AttrBonus": "EquipBonus" if q else {"Hp": 2}}
            tables["EquipStar"][f"ESE_{tpl}_{q}"] = {"AttrBonus": [{"attr_type": "Atk", "value": 3 + q}]}
    tables["SkillAttr"]["EquipBonus"] = {"AttrType": ["Atk", "Hp"], "Value": [[5], 6]}
    tables["PlayerLevel"] = {str(i): {"ActionPointLimit": 60 + i * 5} for i in [1, 2, 3, 80]}
    tables["UnlockSystem"] = {"Open": {}, "TaskGate": {"UnlockCondition": [{"Task": "T1"}]}, "LevelGate": {"UnlockCondition": [{"Level": 3}]}, "_meta": {}}
    tables["ExploreWorldMap"] = {"World1": {"UnlockCondition": [{"Level": 99}]}}
    tables["ExploreWorldRegion"] = {"Hidden": {"Type": "Story", "ExploreMap": [], "UnlockShow": 1}, "Story": {"Type": "Story", "ExploreMap": ["M1"], "UnlockShow": 1}}
    tables["ExploreWorldMain"] = {"Main1": {"OpenCondition": [{"Level": 3}]}, "Main2": {"UnlockCondition": [{"Task": "T1"}]}}
    tables["ResourcesMap"] = {"TrialA": {"Type": "MONEY", "Id": "TrialA"}, "_meta": {}}
    tables["ResourcesPoint"] = {"TrialPoint": {"Type": "HEROEXP"}}
    tables["FightMap"] = {"M1": {"FightPoint": ["P1"], "ElitePoint": ["P2"], "UnlockCondition": [{"Level": 3}, {"FightPoint": "P0"}]}, "M2": {"FightPoint": ["P0"], "UnlockCondition": [{"FightPointStar": 2}]}}
    tables["FightPoint"] = {"P0": {"Type": "NORMAL"}, "P1": {"Type": "NORMAL"}, "P2": {"Type": "ELITE"}}
    tables["Activity"] = {"DrawA": {}, "ShopA": {"ServerType": "SHOPBAG"}, "Ignored": {"Type": "OTHER"}}
    tables["DrawCard"] = {"Pool1": {"ActivityID": "DrawA"}, "Pool2": {"ActivityID": "Missing"}}
    tables["SummitBase"] = {"Summit_1": {"Id": "Summit_1", "OpenTime": "123", "CloseTime": "456"}}
    tables["NameFront"] = {"F1": {"Content": "甲"}, "F2": {"Content": "乙"}}
    tables["NameMiddle"] = {"M1": {"Content": "之"}}
    tables["NameLast"] = {"L1": {"Content": "光"}, "L2": {"Content": "影"}}
    return tables


def namespace(tables, directory, options):
    tree = ast.parse(SOURCE.read_text(encoding="utf-8-sig"))
    allowed = set("PLAYER_DEFAULTS LEADER_BASE_DEFAULT STARTER_HERO STARTER_EQUIP_IDS STARTER_EQUIP_FALLBACK TEAM_SLOT_CAP_DEFAULT AWAKEN_LOCKS_PASSIVE EXPLORE_FORCE_UNLOCK UNLOCK_ALL_SYSTEMS UNLOCK_TASK_GATES TASK_GATED_SYSTEM_COUNT GUIDE_DONE_FLAGS PROLOGUE_STORY_PREFIX STAMINA_RECOVER_SECONDS STAMINA_PER_TICK _LEADER_ATTR_FACTORS EQUIP_CMDS HERO_DEVELOP_CMDS LOTTERY_CMDS STAGE_REWARD_CMDS GUILD_LIST_CMDS".split())
    ns = {"json": json, "os": os, "shutil": shutil, "random": random.Random(1), "time": types.SimpleNamespace(time=lambda: options["now"], strftime=time.strftime, localtime=time.localtime)}
    for node in tree.body:
        if isinstance(node, ast.Assign):
            for target in node.targets:
                if isinstance(target, ast.Name) and target.id in allowed:
                    ns[target.id] = ast.literal_eval(node.value)
    funcs = [n for n in tree.body if isinstance(n, (ast.FunctionDef, ast.ClassDef))]
    exec(compile(ast.Module(body=funcs, type_ignores=[]), str(SOURCE), "exec"), ns)
    ns.update({"_tables": copy.deepcopy(tables), "_player": copy.deepcopy(ns["PLAYER_DEFAULTS"]), "_max_level": None, "_starter_equip_cache": None, "_name_pool": None, "_all_systems": None, "TEAM_SLOT_CAP": ns["TEAM_SLOT_CAP_DEFAULT"], "SKIP_GUIDE": options.get("skip_guide", False), "REPLAY_PROLOGUE": options.get("replay_prologue", False), "SAVE_FILE": str(directory / "save_localplayer.json"), "SAVE_PREV_FILE": str(directory / "save_localplayer.prev.json"), "SAVE_DIR": str(directory), "SAVE_BACKUPS": 10, "log": lambda *args: None})
    # 原函数内部的表导入只读取显式输入，不执行原服务的顶层启动代码
    sys.modules["read_config_table"] = types.SimpleNamespace(load_table=lambda name: copy.deepcopy(tables.get(name, {})))
    protocol_map = json.loads((SOURCE.parent / "protocol_map.json").read_text(encoding="utf-8-sig"))
    ns.update({"PROTO": protocol_map, "BY_CODE": {}})
    mapping = [node for node in tree.body if 65 <= node.lineno <= 72]
    exec(compile(ast.Module(body=mapping, type_ignores=[]), str(SOURCE), "exec"), ns)
    choices = []
    def choose(seq):
        choices.append(0)
        return seq[0]
    ns["random"] = types.SimpleNamespace(choice=choose)
    return ns, choices


def make_case(name, initial, commands, tables, options):
    with tempfile.TemporaryDirectory(prefix="gxjy-baseline-") as temp:
        directory = Path(temp)
        ns, choices = namespace(tables, directory, options)
        save = directory / "save_localplayer.json"
        if initial is not None:
            save.write_text(json.dumps(initial, ensure_ascii=False), encoding="utf-8")
        ns["load_player"]()
        ns["ensure_starter_equips"]()
        startup = copy.deepcopy(ns["_player"])
        startup_disk = json.loads(save.read_text(encoding="utf-8")) if save.exists() else None
        session = ns["Session"](None)
        messages = []
        session.send = lambda cmd, msg: messages.append({"command": cmd, "payload": copy.deepcopy(msg)})
        steps = []
        for command, request in commands:
            messages.clear()
            session.dispatch(command, copy.deepcopy(request))
            steps.append({"command": command, "request": request, "messages": copy.deepcopy(messages), "state": copy.deepcopy(ns["_player"]), "disk": json.loads(save.read_text(encoding="utf-8")) if save.exists() else None})
        return {"name": name, "initial": initial, "options": options, "startup": startup, "startup_disk": startup_disk, "choices": choices, "steps": steps}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", type=Path, default=ROOT / "gxjy-server/internal/game/testdata/existing_paths_python.json")
    parser.add_argument("--real-assets", action="store_true")
    args = parser.parse_args()
    tables = real_tables() if args.real_assets else synthetic_tables()
    options = {"now": 1800000000}
    commands = [(101, {}), (901, {"msg": {"clientTimestamp": 123}, "seq": 2}), (1011, {}), (1013, {"settings": {"common": {"nested": [1, None, True], "flag": False}, "need_reset": {"daily": 2}, "ignored": {"x": 1}}}), (1254, {}), (1254, {}), (1301, {}), (1016, {}), (1003, {}), (1004, {"role_name": "新名称"}), (1015, {"leader_id": "LeaderHero_H011", "role_name": "新队长"})]
    old = {"role_id": 10001, "role_name": "Old", "level": 3, "exp": 10, "leader": "LeaderHero_H004", "currency": {"IC_Stamina": 1, "IC_Gold": 123}, "stamina_at": options["now"] - 721, "heroes": ["H011", "H002"], "custom": {"common": {"guide_draw": True, "IsSkinGuide": True, "StorySMSP_T00_0100_Test": True, "keep": {"v": 9}}, "need_reset": {}}, "hero_state": {"H004": {"level": 7, "exp": 12, "quality": 1, "star": 1, "awaken": 1, "favor": 20, "favor_exp": 5, "surface": "invalid"}}, "bag_items": {"EQ1_H004": 2, "Potion": "3", "Zero": 0, "Bad": "bad"}, "points": {"P0": {"pass": True, "num": 2, "star_idxs": [2, 1, 2]}}, "teams": {"1": {"leader": "H004", "members": ["H004", "H002"], "team_name": "旧队伍"}, "7": {"leader": "bad", "members": ["H011"]}, "TrialA": {"members": ["H011"]}}, "summit": {"rank": 3, "score": 456}, "unknown_field": {"precise": 9007199254740993, "nested": [None, False]}}
    old["equip_state"] = {"EQ1_H004": {"template_id": "Equip_S3_1", "user": "H004", "level": 2, "quality_num": 1, "star_num": 1, "exp": 7, "locked": True, "enchantment": {"a": 1}}}
    cases = [make_case("new", None, commands, tables, options), make_case("old", old, commands, tables, options), make_case("partial", {"role_name": "Partial", "level": None, "currency": None}, [(101, {}), (1254, {})], tables, options), make_case("replay", old, [(101, {}), (1013, {"settings": {"common": {"StorySMSP_T00_0100_Test": True}}}), (101, {})], tables, {**options, "replay_prologue": True}), make_case("skip_guide", {}, [(101, {})], tables, {**options, "skip_guide": True})]
    boundary = copy.deepcopy(old)
    boundary.update({"role_id": "string-id", "level": 999, "head_image": "saved-head", "head_rect": "saved-frame", "back_now": "saved-background", "wife": "H011", "mode": "saved-mode", "currency": {"IC_Stamina": 999}, "teams": {"0": {"leader": "H011", "members": ["H002"] * 12}, "1": {"team_name": "preserved"}}, "stamina_at": options["now"] + 60})
    cases.append(make_case("boundaries", boundary, [(101, {}), (901, {}), (901, {"clientTimestamp": None}), (1013, {}), (1013, {"settings": None}), (1013, {"settings": {"common": None, "need_reset": []}}), (1004, {"role_name": ""}), (1015, {"leader_id": "bad-id"}), (1254, {})], tables, options))
    if not args.real_assets:
        fallback = copy.deepcopy(tables)
        fallback["BaseConfig"]["Point_Initail_Auto_Reward"] = {"Config": {"Show": []}}
        fallback["BaseConfig"]["Point_Box_Config"] = {"Config": {"z-first": {"content": [{"Reward": "FallbackReward"}]}, "a-second": {"content": [{"Reward": "WrongReward"}]}}}
        fallback["RewardConfig"] = {"FallbackReward": {"Content": [{"code": "First", "amount": 7}]}, "WrongReward": {"Content": [{"code": "Second", "amount": 9}]}}
        # 同一份配置覆盖后续所有用例；已有用例重新生成以保持输入与预期一致
        tables = fallback
        cases = [make_case(c["name"], c["initial"], [(s["command"], s["request"]) for s in c["steps"]], tables, c["options"]) for c in cases]
    keys = {}
    def collect(path, value):
        if isinstance(value, dict):
            keys[path] = list(value)
            for key, child in value.items():
                collect(path + "/" + key, child)
    for name, value in tables.items():
        collect(name, value)
    with tempfile.TemporaryDirectory(prefix="gxjy-http-") as temp:
        ns, _ = namespace(tables, Path(temp), options)
        ns["GATE_HOST"] = "127.0.0.1"
        http_cases = []
        for path, body in [("/player/login", "{}"), ("/player/enter", "{}"), ("/project.manifest", ""), ("/version.manifest", ""), ("/getResource", '{"base_version":"v1"}'), ("/getResource", '{"base_version":0,"version":123}'), ("/getResource", '{"base_version":true}'), ("/getResource", 'invalid')]:
            http_cases.append({"path": path, "body": body, "response": ns["make_login_response"](path, body.encode(), 8765)})
    artifact = {"source": str(SOURCE.relative_to(ROOT)), "source_sha256": hashlib.sha256(SOURCE.read_bytes()).hexdigest(), "tables_kind": "real-assets" if args.real_assets else "synthetic", "tables": tables, "table_keys": keys, "cases": cases, "http": http_cases}
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(artifact, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"generated {len(cases)} cases, {sum(len(c['steps']) for c in cases)} steps: {args.output}")


def real_tables():
    spec = importlib.util.spec_from_file_location("baseline_table_reader", SOURCE.parent / "read_config_table.py")
    reader = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(reader)
    original_cache = Path(reader.CACHE_DIR)
    original_load_one = reader._load_one
    # 旧缓存只读，解密缓存的新增写入全部进入临时目录
    with tempfile.TemporaryDirectory(prefix="gxjy-tables-") as temp:
        reader.CACHE_DIR = temp
        def load_one(name):
            path = Path(reader.table_path(name))
            stat = path.stat()
            cached = original_cache / f"{name}.{int(stat.st_mtime)}_{stat.st_size}.json.gz"
            if cached.exists():
                with gzip.open(cached, "rt", encoding="utf-8") as stream:
                    return json.load(stream)
            return original_load_one(name)
        reader._load_one = load_one
        tables = {}
        for name in TABLES:
            tables[name] = reader.load_table(name)
            print(f"loaded {name}: {len(tables[name])} rows", flush=True)
        return tables


if __name__ == "__main__":
    main()
