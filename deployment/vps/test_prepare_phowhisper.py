"""Artifact phải đúng revision, đủ file, checksum và phiên bản runtime."""

import json
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

from prepare_phowhisper import checksums, verify_model
from verify_phowhisper import distance, normalize


class ArtifactTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        spec = json.loads(Path(__file__).with_name("phowhisper-manifest.json").read_text())
        for name in ("model.bin", "config.json", "tokenizer.json", "preprocessor_config.json"):
            (self.root / name).write_text("fixture")
        spec["sha256"] = checksums(self.root)
        (self.root / "manifest.json").write_text(json.dumps(spec))
        self.versions = {**spec["runtime"], **spec["conversion"]}

    def test_valid_artifact(self):
        with patch("prepare_phowhisper.version", side_effect=self.versions.__getitem__):
            self.assertEqual(verify_model(self.root)["quantization"], "int8")

    def test_tamper_is_rejected(self):
        (self.root / "model.bin").write_text("modified")
        with self.assertRaisesRegex(ValueError, "checksum mismatch"):
            verify_model(self.root)

    def test_wrong_revision_is_rejected(self):
        path = self.root / "manifest.json"
        spec = json.loads(path.read_text())
        spec["revision"] = "main"
        path.write_text(json.dumps(spec))
        with self.assertRaisesRegex(ValueError, "manifest mismatch"):
            verify_model(self.root)

    def test_wrong_runtime_is_rejected(self):
        with patch("prepare_phowhisper.version", return_value="0.0.0"):
            with self.assertRaisesRegex(ValueError, "Runtime version"):
                verify_model(self.root)

    def test_metric_normalization_preserves_vietnamese_diacritics(self):
        self.assertEqual(normalize(" XIN CHÀO, Việt Nam! "), "xin chào việt nam")
        self.assertEqual(distance("xin chào".split(), "xin chao".split()), 1)
        self.assertEqual(distance([], ["hello"]), 1)


if __name__ == "__main__":
    unittest.main()
