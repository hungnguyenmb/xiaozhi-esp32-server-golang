"""Exercise the actual Actions YAML migration against a private temporary config."""
import contextlib
import io
import os
from pathlib import Path
import tempfile
import textwrap
import unittest
from unittest.mock import patch

import yaml


class DeployConfigTest(unittest.TestCase):
    def run_migration(self, mode):
        workflow = (Path(__file__).parents[2] / ".github/workflows/deploy-backend.yml").read_text()
        code = workflow.split("python3 - <<'PYCONFIG'\n", 1)[1].split("            PYCONFIG", 1)[0]
        code = textwrap.dedent(code)
        before = {"chat": {"audio_capture": {"enabled": True},
            "asr_input_gate": {"enabled": True, "post_tts_guard_ms": 300},
            "audio_preprocess": {"start": .61}}, "other": {"private": "preserve"}}
        with tempfile.TemporaryDirectory() as root:
            previous = Path.cwd()
            try:
                os.chdir(root)
                path = Path("runtime/config/main_config.yaml")
                path.parent.mkdir(parents=True)
                path.write_text(yaml.safe_dump(before))
                path.chmod(0o600)
                original = path.stat()
                env = dict(AUDIO_PREPROCESS=mode, AUDIO_DIAGNOSTICS="unchanged",
                    AUDIO_CAPTURE="unchanged", ASR_INPUT_GATE="unchanged", POST_TTS_GUARD_MS="300")
                with patch.dict(os.environ, env), contextlib.redirect_stdout(io.StringIO()):
                    if mode == "invalid":
                        with self.assertRaises(ValueError):
                            exec(compile(code, "Actions-config-migration", "exec"), {})
                    else:
                        exec(compile(code, "Actions-config-migration", "exec"), {})
                after = yaml.safe_load(path.read_text())
                current = path.stat()
                self.assertEqual((current.st_uid, current.st_gid, current.st_mode & 0o777),
                                 (original.st_uid, original.st_gid, 0o600))
                if mode not in ("unchanged", "invalid"):
                    before["chat"]["audio_preprocess"]["mode"] = mode
                self.assertEqual(after, before)
            finally:
                os.chdir(previous)

    def test_modes_preserve_other_configuration_and_permissions(self):
        for mode in ("unchanged", "off", "shadow", "enforce", "invalid"):
            with self.subTest(mode=mode):
                self.run_migration(mode)


if __name__ == "__main__":
    unittest.main()
