"""Cầu nối PCM16/FunASR WebSocket sang Gipformer offline trên CPU.

Chỉ dùng trong mạng Docker; trả kết quả sau is_speaking=false.
"""

import asyncio
import json
import logging
import os
import time
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

import numpy as np
import sherpa_onnx
from aiohttp import WSMsgType, web

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
MAX_AUDIO_BYTES = 16000 * 2 * 60
MAX_CONNECTIONS = 16


def make_app():
    model = Path(os.environ.get("MODEL_DIR", "/models"))
    recognizer = sherpa_onnx.OfflineRecognizer.from_transducer(
        encoder=str(model / "encoder.int8.onnx"),
        decoder=str(model / "decoder.int8.onnx"),
        joiner=str(model / "joiner.int8.onnx"),
        tokens=str(model / "tokens.txt"),
        num_threads=2,
        sample_rate=16000,
        feature_dim=80,
        decoding_method="greedy_search",
    )
    # Một worker giữ tổng số luồng suy luận ổn định; kết nối khác đợi có giới hạn.
    executor = ThreadPoolExecutor(max_workers=1)
    slots = asyncio.Semaphore(4)
    connections = set()

    def recognize(pcm):
        samples = np.frombuffer(pcm, dtype="<i2").astype(np.float32) / 32768.0
        stream = recognizer.create_stream()
        stream.accept_waveform(16000, samples)
        recognizer.decode_streams([stream])
        return stream.result.text.strip()

    async def health(request):
        return web.json_response({"ready": True, "model": "gipformer1.5-68M-rnnt-int8"})

    async def handle(request):
        if len(connections) >= MAX_CONNECTIONS:
            raise web.HTTPServiceUnavailable(text="ASR connection limit")
        ws = web.WebSocketResponse(max_msg_size=1024 * 1024, receive_timeout=90)
        await ws.prepare(request)
        connections.add(ws)
        pcm = bytearray()
        wav_name = "stream"
        try:
            async for msg in ws:
                if msg.type == WSMsgType.BINARY:
                    pcm.extend(msg.data)
                    if len(pcm) > MAX_AUDIO_BYTES:
                        await ws.close(code=1009, message=b"Maximum audio: 60 seconds")
                        break
                elif msg.type == WSMsgType.TEXT:
                    try:
                        control = json.loads(msg.data)
                        if not isinstance(control, dict):
                            raise ValueError("Expected object")
                    except (ValueError, TypeError):
                        await ws.close(code=1007, message=b"Invalid JSON")
                        break
                    wav_name = control.get("wav_name", wav_name)
                    if control.get("audio_fs", 16000) != 16000:
                        await ws.close(code=1003, message=b"Expected 16000Hz PCM16")
                        break
                    if control.get("is_speaking") is True:
                        pcm.clear()
                    elif control.get("is_speaking") is False:
                        if len(pcm) % 2:
                            await ws.close(code=1007, message=b"Invalid PCM16")
                            break
                        started = time.monotonic()
                        try:
                            await asyncio.wait_for(slots.acquire(), timeout=10)
                        except TimeoutError:
                            await ws.close(code=1013, message=b"ASR busy")
                            break
                        try:
                            text = await asyncio.get_running_loop().run_in_executor(
                                executor, recognize, bytes(pcm)
                            ) if pcm else ""
                        finally:
                            slots.release()
                        logging.info("ASR audio_s=%.2f elapsed_s=%.3f", len(pcm) / 32000, time.monotonic() - started)
                        await ws.send_json({"text": text, "is_final": True, "mode": "offline", "wav_name": wav_name})
                        pcm.clear()
                elif msg.type == WSMsgType.ERROR:
                    break
        finally:
            connections.discard(ws)
        return ws

    async def cleanup(app):
        for ws in list(connections):
            await ws.close(code=1001)
        executor.shutdown(wait=True)

    app = web.Application()
    app.router.add_get("/health", health)
    app.router.add_get("/", handle)
    app.on_cleanup.append(cleanup)
    return app


if __name__ == "__main__":
    web.run_app(make_app(), host="0.0.0.0", port=10096)
