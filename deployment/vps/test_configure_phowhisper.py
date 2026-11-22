"""Kiểm tra đổi cấu hình qua API giả lập, không cần VPS/credentials."""

import copy
import json
import tempfile
import unittest
from pathlib import Path

from configure_phowhisper import CONFIG_ID, SETTINGS, configure, default


class FakeManager:
    def __init__(self):
        self.configs = [{"id": 1, "config_id": "gipformer", "name": "Giữ tên riêng",
                         "provider": "funasr", "json_data": '{"host":"asr"}',
                         "enabled": True, "is_default": True}]
        self.fail_once = False

    def rows(self):
        return copy.deepcopy(self.configs)

    def api(self, method, path, body):
        assert method == "POST" and path == "/api/admin/asr-configs"
        row = {**body, "id": 2}
        self.configs.append(copy.deepcopy(row))
        return {"data": row}

    def set_default(self, row):
        if self.fail_once:
            self.fail_once = False
            raise RuntimeError("API failure")
        for existing in self.configs:
            existing["is_default"] = existing["id"] == row["id"]
            if existing["is_default"]:
                existing["enabled"] = True


class ConfigureTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.state = Path(self.tmp.name) / "state.json"
        self.manager = FakeManager()

    def test_activate_and_rollback_preserve_other_edits(self):
        self.assertEqual(configure(self.manager, "activate", self.state), CONFIG_ID)
        self.assertEqual(self.state.stat().st_mode & 0o777, 0o600)
        self.manager.configs[0]["name"] = "Tên sửa sau rollout"
        self.manager.configs[0]["json_data"] = '{"host":"new-gipformer"}'
        self.assertEqual(configure(self.manager, "rollback", self.state), "gipformer")
        self.assertEqual(self.manager.configs[0]["name"], "Tên sửa sau rollout")
        self.assertEqual(self.manager.configs[0]["json_data"], '{"host":"new-gipformer"}')
        self.assertEqual(configure(self.manager, "rollback", self.state), "gipformer")

    def test_failed_activation_restores_previous_default(self):
        self.manager.fail_once = True
        with self.assertRaisesRegex(RuntimeError, "API failure"):
            configure(self.manager, "activate", self.state)
        self.assertEqual(default(self.manager.rows())["config_id"], "gipformer")

    def test_repeated_activation_cannot_destroy_snapshot(self):
        configure(self.manager, "activate", self.state)
        saved = self.state.read_text()
        with self.assertRaisesRegex(ValueError, "already active"):
            configure(self.manager, "activate", self.state)
        self.assertEqual(saved, self.state.read_text())

    def test_candidate_with_changed_endpoint_is_rejected(self):
        self.manager.configs.append({"id": 2, "config_id": CONFIG_ID, "is_default": False,
                                     "enabled": True, "provider": "funasr", "json_data": "{}"})
        with self.assertRaisesRegex(ValueError, "differs"):
            configure(self.manager, "activate", self.state)
        self.assertEqual(default(self.manager.rows())["id"], 1)

    def test_rollback_does_not_override_new_admin_default(self):
        configure(self.manager, "activate", self.state)
        self.manager.configs.append({**self.manager.configs[0], "id": 3, "config_id": "manual-new"})
        self.manager.set_default(self.manager.configs[-1])
        with self.assertRaisesRegex(ValueError, "outside this rollout"):
            configure(self.manager, "rollback", self.state)
        self.assertEqual(default(self.manager.rows())["id"], 3)

    def test_settings_match_existing_funASR_contract(self):
        self.assertFalse(SETTINGS["auto_end"])
        self.assertEqual(SETTINGS["sample_rate"], 16000)
        self.assertEqual(SETTINGS["host"], "asr-phowhisper")


if __name__ == "__main__":
    unittest.main()
