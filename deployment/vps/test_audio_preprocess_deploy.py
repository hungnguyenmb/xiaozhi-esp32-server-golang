"""Exercise the actual Actions YAML migration against a private temporary config."""
import contextlib
import io
import os
from pathlib import Path
import tempfile
import textwrap
import unittest
import subprocess
import sys
from unittest.mock import patch

import yaml


class DeployConfigTest(unittest.TestCase):
    def test_full_script_native_health_failure_rolls_back_binary_and_config(self):
        workflow = yaml.safe_load((Path(__file__).parents[2] /
            ".github/workflows/deploy-backend.yml").read_text())
        steps = workflow["jobs"]["build-and-deploy-backend"]["steps"]
        script = next(s["with"]["script"] for s in steps
                      if s["name"] == "Deploy and Reload Backend on VPS")
        for fail in (False, True):
            with self.subTest(native_failure=fail), tempfile.TemporaryDirectory() as temp:
                root = Path(temp)
                (root / "incoming").mkdir()
                (root / "bin").mkdir()
                (root / "bundle").mkdir()
                (root / "runtime/config").mkdir(parents=True)
                config = root / "runtime/config/main_config.yaml"
                before = {"chat": {"audio_capture": {"enabled": True}}, "other": "keep"}
                config.write_text(yaml.safe_dump(before))
                config.chmod(0o600)
                (root / "bundle/xiaozhi_server").write_text("old-binary")
                for artifact in ("xiaozhi_server", "audio_capture_export", "audio_frontend_replay",
                        "libxiaozhi_rnnoise_v01.so", "libspeexdsp.so.1", "RNNOISE-COPYING",
                        "RNNOISE-REVISION", "SPEEXDSP-COPYRIGHT"):
                    (root / "incoming" / artifact).write_text("new-artifact")
                commands = {
                    "curl": "#!/bin/sh\nexit 0\n",
                    "sleep": "#!/bin/sh\nexit 0\n",
                    "python3": f'#!/bin/sh\nexec "{sys.executable}" "$@"\n',
                    "docker": '#!/bin/sh\ncase "$*" in *check-native*) '
                        'if [ "$FAIL_NATIVE" = 1 ]; then exit 1; fi;; esac\nexit 0\n',
                }
                for name, content in commands.items():
                    path = root / "bin" / name
                    path.write_text(content)
                    path.chmod(0o700)
                local = script.replace("cd /opt/docker/antigravity/xiaozhi-go", 'cd "$MOCK_ROOT"')
                local = local.replace("/tmp/", str(root / "incoming") + "/")
                env = {**os.environ, "MOCK_ROOT": temp, "PATH": str(root / "bin") + ":" + os.environ["PATH"],
                    "FAIL_NATIVE": "1" if fail else "0", "AUDIO_PREPROCESS": "shadow",
                    "AUDIO_DIAGNOSTICS": "unchanged", "AUDIO_CAPTURE": "unchanged",
                    "ASR_INPUT_GATE": "unchanged", "POST_TTS_GUARD_MS": "300"}
                result = subprocess.run(["bash", "-c", local], env=env,
                                        capture_output=True, text=True, timeout=20)
                self.assertEqual(result.returncode, 1 if fail else 0, result.stderr)
                actual = yaml.safe_load(config.read_text())
                if not fail:
                    before["chat"]["audio_preprocess"] = {"mode": "shadow"}
                self.assertEqual(actual, before)
                self.assertEqual(config.stat().st_mode & 0o777, 0o600)
                self.assertEqual((root / "bundle/xiaozhi_server").read_text(),
                                 "old-binary" if fail else "new-artifact")

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
