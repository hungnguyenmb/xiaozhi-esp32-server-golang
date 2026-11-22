"""Cổng trial và quay lui release, không gọi Docker/VPS thật."""

import hashlib
import io
import json
import os
import tarfile
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import patch

import deploy_phowhisper as rollout
from configure_phowhisper import CONFIG_ID
from test_configure_phowhisper import FakeManager


class RolloutTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name) / "server"
        self.root.mkdir()
        self.payload = Path(self.temp.name) / "payload"
        self.payload.mkdir()
        self.manager = FakeManager()
        self.release_id = "a" * 40 + "-123-1"
        self.image_id = "sha256:" + "b" * 64
        for name in ("image.tar.gz", "speech.wav", "compose.phowhisper.yaml", "verify_phowhisper.py", "verify_tls.py"):
            (self.payload / name).write_bytes(b"fixture")
        with tarfile.open(self.payload / "model.tar.gz", "w:gz") as archive:
            info = tarfile.TarInfo("manifest.json")
            info.size = 2
            archive.addfile(info, io.BytesIO(b"{}"))
        (self.payload / "release.json").write_text(json.dumps({
            "release_id": self.release_id, "image": "xiaozhi-phowhisper:" + self.release_id,
            "image_id": self.image_id,
            "sha256": {name: hashlib.sha256((self.payload / name).read_bytes()).hexdigest()
                       for name in ("image.tar.gz", "model.tar.gz", "speech.wav")},
        }))
        root_patch = patch.object(rollout, "ROOT", self.root)
        root_patch.start()
        self.addCleanup(root_patch.stop)

    def test_trial_does_not_change_default_and_pins_image_id(self):
        previous_mask = os.umask(0o077)
        try:
            with patch.object(rollout, "run", return_value=SimpleNamespace(stdout=self.image_id)), \
                    patch.object(rollout, "compose") as compose, \
                    patch.object(rollout, "smoke"), patch.object(rollout, "verify_public"):
                rollout.install(self.payload, self.manager)
        finally:
            os.umask(previous_mask)
        current = json.loads((self.root / "runtime/phowhisper-current.json").read_text())
        release = Path(current["release"])
        self.assertEqual(release.stat().st_mode & 0o777, 0o755)
        self.assertIn(self.image_id, (release / "release.env").read_text())
        self.assertEqual((release / "models").stat().st_mode & 0o777, 0o755)
        self.assertEqual(self.manager.rows()[0]["config_id"], "gipformer")
        self.assertEqual(len(self.manager.rows()), 1)
        self.assertEqual(compose.call_args.args[1:], ("up", "-d", "--no-deps", "asr-phowhisper"))

    def test_trial_refuses_to_replace_active_service(self):
        self.manager.configs[0]["config_id"] = CONFIG_ID
        with patch.object(rollout, "run") as run:
            with self.assertRaisesRegex(ValueError, "active"):
                rollout.install(self.payload, self.manager)
        run.assert_not_called()

    def test_tampered_artifact_rejected_before_docker(self):
        (self.payload / "image.tar.gz").write_bytes(b"tampered")
        with patch.object(rollout, "run") as run:
            with self.assertRaisesRegex(ValueError, "checksum mismatch"):
                rollout.install(self.payload, self.manager)
        run.assert_not_called()

    def test_failed_health_restores_previous_sidecar_and_pointer(self):
        current_file = self.root / "runtime/phowhisper-current.json"
        current_file.parent.mkdir()
        previous = {"release": str(self.root / "old-release")}
        current_file.write_text(json.dumps(previous))
        with patch.object(rollout, "run", return_value=SimpleNamespace(stdout=self.image_id)), \
                patch.object(rollout, "compose") as compose, \
                patch.object(rollout, "smoke", side_effect=[RuntimeError("bad health"), None]):
            with self.assertRaisesRegex(RuntimeError, "bad health"):
                rollout.install(self.payload, self.manager)
        self.assertEqual(json.loads(current_file.read_text()), previous)
        self.assertEqual(compose.call_args.args[0], Path(previous["release"]))


if __name__ == "__main__":
    unittest.main()
