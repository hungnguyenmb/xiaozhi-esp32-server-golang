"""Kiểm tra cầu nối ASR bằng WAV mono PCM16 16kHz, gồm tải đồng thời."""
import asyncio
import io
import json
import sys
import time
import urllib.request
import wave

import aiohttp


async def main():
    raw = urllib.request.urlopen("https://raw.githubusercontent.com/ggroup-ai-lab/gipformer/main/data/audio1.wav", timeout=30).read()
    with wave.open(io.BytesIO(raw)) as wav:
        assert (wav.getnchannels(), wav.getsampwidth(), wav.getframerate()) == (1, 2, 16000)
        pcm = wav.readframes(wav.getnframes())
    endpoint = sys.argv[1] if len(sys.argv) > 1 else "ws://127.0.0.1:10096/"
    async with aiohttp.ClientSession() as session:
        async def recognize(index):
            async with session.ws_connect(endpoint) as ws:
                await ws.send_json({"is_speaking": True, "audio_fs": 16000, "wav_name": str(index), "mode": "offline"})
                await ws.send_bytes(pcm)
                start = time.monotonic()
                await ws.send_json({"is_speaking": False})
                result = await ws.receive_json(timeout=30)
                assert result["is_final"] and result["wav_name"] == str(index) and result["text"].strip()
                return {"seconds": round(time.monotonic() - start, 3), "text": result["text"]}
        single = await recognize(0)
        concurrent = await asyncio.gather(*(recognize(i) for i in range(1, 5)))
        async with session.ws_connect(endpoint) as ws:
            await ws.send_str("[]")
            invalid = await ws.receive()
            assert invalid.type == aiohttp.WSMsgType.CLOSE and invalid.data == 1007
    print(json.dumps({"audio_seconds": round(len(pcm) / 32000, 3), "single": single, "four_concurrent": concurrent, "invalid_json_rejected": True}, ensure_ascii=False, indent=2))


asyncio.run(main())
