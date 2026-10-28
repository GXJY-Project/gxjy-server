"""使用合成状态验证隔离、备份、回滚和幂等，不读取真实玩家数据"""

import json
from pathlib import Path
import socket
import tempfile
import unittest
from unittest.mock import patch

import repair_bili224_save as repair


class RepairTest(unittest.TestCase):
    def test_quarantine_preserves_records_and_is_idempotent(self):
        player = {"role_id": 1, "currency": {"Gold": 123}, "hero_state": {"H002": {"level": 3}},
                  "equip_state": {"valid": {"template_id": "Current"}, "old": {"template_id": "Legacy", "level": 9}},
                  "tasks": {"Current": {"status": 2}, "Legacy": {"status": 1}}, "bag_items": {"Current": 3},
                  "custom": {"common": {"guide": True}}}
        original = repair.encoded(player)
        tables = {name: {"Current": {}} for name in ("EquipBase", "TaskBase", "ItemConfig")}
        tables["HeroBase"] = {"H002": {}}
        result, counts = repair.plan(player, tables)
        self.assertEqual(counts, {"equip_state": 1, "tasks": 1, "bag_items": 0})
        self.assertEqual(repair.encoded(player), original)
        for field in ("role_id", "currency", "hero_state", "custom", "bag_items"):
            self.assertEqual(result[field], player[field])
        for field, key in (("equip_state", "old"), ("tasks", "Legacy")):
            self.assertEqual(result["_bili224_quarantine"][field][key], player[field][key])
            self.assertNotIn(key, result[field])
        again, counts = repair.plan(result, tables)
        self.assertEqual(again, result)
        self.assertEqual(sum(counts.values()), 0)
        tables["HeroBase"] = {}
        with self.assertRaises(ValueError):
            repair.plan(player, tables)

    def test_backup_rollback_drift_and_busy_port(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "data/save").mkdir(parents=True)
            first, second = root / "config.json", root / "data/save/save_localplayer.json"
            first.write_bytes(b'{"old":1}')
            second.write_bytes(b'{"old":2}')
            originals = {first: first.read_bytes(), second: second.read_bytes()}
            replacements = {first: b'{"new":1}', second: b'{"new":2}'}
            # 仅测试隔离目录，模拟 Git 忽略检查成功
            with patch.object(repair.subprocess, "run") as git:
                git.return_value.returncode = 0
                backup = repair.apply_files(root, originals, replacements, [])
                for path in originals:
                    self.assertEqual((backup / path.name).read_bytes(), originals[path])
                    self.assertEqual(path.read_bytes(), replacements[path])
                with self.assertRaises(ValueError):
                    repair.apply_files(root, originals, replacements, [])
                for path, raw in originals.items():
                    path.write_bytes(raw)
                actual_write = repair.atomic_write
                calls = 0

                def fail_second(path, raw):
                    nonlocal calls
                    calls += 1
                    if calls == 2:
                        raise OSError("synthetic second write failure")
                    actual_write(path, raw)

                with patch.object(repair, "atomic_write", side_effect=fail_second):
                    with self.assertRaises(OSError):
                        repair.apply_files(root, originals, replacements, [])
                for path, raw in originals.items():
                    self.assertEqual(path.read_bytes(), raw)
                with socket.socket() as listener:
                    listener.bind(("127.0.0.1", 0))
                    listener.listen()
                    with self.assertRaises(OSError):
                        repair.apply_files(root, originals, replacements, [listener.getsockname()[1]])


if __name__ == "__main__":
    unittest.main()
