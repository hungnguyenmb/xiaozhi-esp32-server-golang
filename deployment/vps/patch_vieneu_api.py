"""Expose lựa chọn CPU của SDK trong API upstream đã ghim commit."""
from pathlib import Path

source = Path(__file__).resolve().parent / "vieneu-source/apps/openai_speech.py"
text = source.read_text()
changes = [
    ('SAMPLE_RATE = 48_000\nMODEL_ID = "vieneu-v3-turbo"',
     'CPU_MODE = os.environ.get("VIENEU_MODE", "v3turbo")\nSAMPLE_RATE = 24_000 if CPU_MODE == "v3nano" else 48_000\nMODEL_ID = "vieneu-v3-nano" if CPU_MODE == "v3nano" else "vieneu-v3-turbo"'),
    ('max_streams=_env_int("VIENEU_MAX_STREAMS", 16))',
     'max_streams=_env_int("VIENEU_MAX_STREAMS", 16), threads=_env_int("VIENEU_THREADS", 2))'),
    ('self.tts = Vieneu(mode="v3turbo", **kw)', 'self.tts = Vieneu(mode=CPU_MODE, **kw)'),
    ('log.info("⏳ loading VieNeu-TTS v3 Turbo (backend=%s device=%s)", backend, device)',
     'log.info("⏳ loading VieNeu-TTS %s (backend=%s device=%s)", MODEL_ID, backend, device)'),
    ('max_chars: int = Field(256, ge=64, le=512)', 'max_chars: int = Field(64, ge=64, le=512)'),
    ('frozenset(self.tts._voice_aliases)', 'frozenset(getattr(self.tts, "_voice_aliases", {}))'),
    ('if req.voice and eng.tts.resolve_voice_name(req.voice) is None:',
     'if req.voice and req.voice not in eng.tts._preset_voices and (not hasattr(eng.tts, "resolve_voice_name") or eng.tts.resolve_voice_name(req.voice) is None):'),
]
for old, new in changes:
    if new in text:
        continue
    assert text.count(old) == 1, "Nguồn upstream thay đổi, cần review lại patch"
    text = text.replace(old, new)
source.write_text(text)
print("API CPU patch applied: model mode, native sample rate, threads, short text chunks")
