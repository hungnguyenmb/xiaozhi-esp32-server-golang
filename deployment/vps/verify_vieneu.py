"""Chạy trong container tts: đo WAV streaming, bỏ header khi tính gói đầu."""
import json
import os
import struct
import time
import urllib.error
import urllib.request

endpoint = "http://127.0.0.1:8000"
headers = {"Authorization": "Bearer " + os.environ["VIENEU_API_KEY"], "Content-Type": "application/json"}
try:
    urllib.request.urlopen(endpoint + "/v1/voices", timeout=5)
except urllib.error.HTTPError as error:
    assert error.code == 401
else:
    raise AssertionError("API phải yêu cầu xác thực")
with urllib.request.urlopen(urllib.request.Request(endpoint + "/v1/voices", headers=headers), timeout=10) as response:
    voices = json.load(response)
print("VOICES", json.dumps(voices, ensure_ascii=False), flush=True)
texts = [
    "Xin chào, mình đang nghe bạn nói đây.",
    "Xin chào, đây là giọng nói tiếng Việt chạy ngay trên máy chủ. Bạn muốn mình giúp điều gì hôm nay?",
]
for text in texts:
    request = urllib.request.Request(endpoint + "/v1/audio/speech", headers=headers, data=json.dumps({
        "model": "vieneu-v3-nano", "voice": "Mai Anh", "input": text, "response_format": "wav",
    }).encode())
    started = time.monotonic()
    first_pcm_ms = None
    with urllib.request.urlopen(request, timeout=120) as response:
        header = response.read(44)
        assert header[:4] == b"RIFF" and header[8:12] == b"WAVE" and header[36:40] == b"data"
        channels, rate = struct.unpack_from("<HI", header, 22)
        bits = struct.unpack_from("<H", header, 34)[0]
        assert (channels, bits) == (1, 16)
        total = 0
        chunks = 0
        peak = 0
        while True:
            pcm = response.read(1920)
            if not pcm:
                break
            if first_pcm_ms is None:
                first_pcm_ms = round((time.monotonic() - started) * 1000)
            chunks += 1
            total += len(pcm)
            samples = struct.unpack("<" + "h" * (len(pcm) // 2), pcm)
            peak = max(peak, max(abs(x) for x in samples))
        transfer = response.headers.get("Transfer-Encoding")
    elapsed = time.monotonic() - started
    duration = total / (rate * channels * bits / 8)
    assert total > 0 and peak > 10 and chunks > 1 and transfer == "chunked"
    print(json.dumps({"text": text, "sample_rate": rate, "format": "wav_s16le_mono", "transfer": transfer,
        "first_pcm_ms": first_pcm_ms, "generation_seconds": round(elapsed, 3), "audio_seconds": round(duration, 3),
        "rtf": round(elapsed / duration, 3), "chunks": chunks, "peak": peak}, ensure_ascii=False), flush=True)
