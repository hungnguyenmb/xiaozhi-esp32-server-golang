"""PhoWhisper CPU qua giao thức FunASR offline, chỉ trong mạng Docker."""

import asyncio
import json
import logging
import os
import time
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass
from pathlib import Path

import numpy as np
from aiohttp import WSMsgType, web

LOG = logging.getLogger(__name__)
SAMPLE_RATE = 16000
MAX_AUDIO_BYTES = SAMPLE_RATE * 2 * 60


class BusyError(Exception):
    pass


@dataclass
class Job:
    pcm: bytes
    future: asyncio.Future
    submitted: float
    deadline: float


class InferenceQueue:
    """Một native inference tại một thời điểm; timeout không giải phóng worker sớm."""

    def __init__(self, recognize, capacity=4, timeout=30):
        self.recognize = recognize
        self.queue = asyncio.Queue(maxsize=capacity)
        self.timeout = timeout
        self.executor = ThreadPoolExecutor(max_workers=1)
        self.task = None
        self.running = False

    def start(self):
        self.task = asyncio.create_task(self.work())

    def submit(self, pcm):
        loop = asyncio.get_running_loop()
        job = Job(pcm, loop.create_future(), time.monotonic(), 0)
        job.deadline = job.submitted + self.timeout
        try:
            self.queue.put_nowait(job)
        except asyncio.QueueFull as exc:
            raise BusyError from exc
        return job.future

    async def work(self):
        while True:
            job = await self.queue.get()
            try:
                if job.future.cancelled():
                    continue
                started = time.monotonic()
                if started >= job.deadline:
                    job.future.set_exception(TimeoutError())
                    continue
                self.running = True
                try:
                    text = await asyncio.get_running_loop().run_in_executor(
                        self.executor, self.recognize, job.pcm
                    )
                    elapsed = time.monotonic() - started
                    metrics = {
                        "queue_ms": round((started - job.submitted) * 1000, 2),
                        "inference_ms": round(elapsed * 1000, 2),
                        "audio_s": len(job.pcm) / 32000,
                    }
                    LOG.info("ASR %s", json.dumps(metrics))
                    if not job.future.done():
                        job.future.set_result((text, metrics))
                except Exception:
                    LOG.exception("ASR inference failed")
                    if not job.future.done():
                        job.future.set_exception(RuntimeError("ASR inference failed"))
                finally:
                    self.running = False
            finally:
                self.queue.task_done()

    async def close(self):
        # Drain pending requests before stopping the worker; never run a second
        # model while a timed-out native call is still consuming CPU.
        while not self.queue.empty():
            job = self.queue.get_nowait()
            job.future.cancel()
            self.queue.task_done()
        await self.queue.join()
        self.task.cancel()
        await asyncio.gather(self.task, return_exceptions=True)
        self.executor.shutdown(wait=True)


def load_recognizer(model_dir, threads):
    import ctranslate2
    from faster_whisper import WhisperModel
    from prepare_phowhisper import verify_model

    manifest = verify_model(model_dir)
    if "int8" not in ctranslate2.get_supported_compute_types("cpu"):
        raise RuntimeError("CPU runtime does not support INT8")
    model = WhisperModel(
        str(model_dir), device="cpu", compute_type="int8", cpu_threads=threads,
        num_workers=1, local_files_only=True,
    )
    if model.model.compute_type != "int8_float32" and model.model.compute_type != "int8":
        raise RuntimeError("Runtime did not select INT8 computation")

    def recognize(pcm):
        if not pcm:
            return ""
        samples = np.frombuffer(pcm, dtype="<i2").astype(np.float32) / 32768.0
        if not np.any(samples):
            return ""
        segments, _ = model.transcribe(
            samples, language="vi", task="transcribe", beam_size=1,
            best_of=1, temperature=0.0, condition_on_previous_text=False,
            vad_filter=False, without_timestamps=True,
        )
        # Inference is lazy: consuming the generator belongs inside the worker.
        return " ".join(segment.text.strip() for segment in segments).strip()

    # Nonzero input exercises encoder/decoder rather than the zero-audio bypass.
    recognize((np.ones(SAMPLE_RATE, dtype="<i2") * 2).tobytes())
    return recognize, manifest


def make_app(recognize=None, *, capacity=4, timeout=30, max_connections=16,
             max_audio_bytes=MAX_AUDIO_BYTES):
    connections = set()
    connection_lock = asyncio.Lock()
    state = {"ready": False, "model": None}
    engine = None

    async def startup(app):
        nonlocal engine
        recognizer = recognize
        if recognizer is None:
            recognizer, manifest = await asyncio.to_thread(
                load_recognizer, Path(os.environ.get("MODEL_DIR", "/models")),
                int(os.environ.get("PHOWHISPER_THREADS", "2")),
            )
            state["model"] = manifest["source"]
        else:
            state["model"] = "test-recognizer"
        engine = InferenceQueue(recognizer, capacity, timeout)
        engine.start()
        state["ready"] = True

    async def health(request):
        return web.json_response({**state, "queue": engine.queue.qsize(),
                                  "running": engine.running}, status=200 if state["ready"] else 503)

    async def handle(request):
        ws = web.WebSocketResponse(max_msg_size=max_audio_bytes, receive_timeout=90)
        async with connection_lock:
            if not state["ready"] or len(connections) >= max_connections:
                raise web.HTTPServiceUnavailable(text="ASR unavailable")
            await ws.prepare(request)
            connections.add(ws)
        pcm = bytearray()
        wav_name = "stream"
        speaking = False
        future = None
        try:
            async for msg in ws:
                if msg.type == WSMsgType.BINARY:
                    if not speaking:
                        await ws.close(code=1008, message=b"Expected start control")
                        break
                    pcm.extend(msg.data)
                    if len(pcm) > max_audio_bytes:
                        await ws.close(code=1009, message=b"Maximum audio: 60 seconds")
                        break
                elif msg.type == WSMsgType.TEXT:
                    try:
                        control = json.loads(msg.data)
                        if not isinstance(control, dict):
                            raise ValueError()
                        if not isinstance(control.get("wav_name", wav_name), str):
                            raise ValueError()
                        if "is_speaking" not in control or type(control["is_speaking"]) is not bool:
                            raise ValueError()
                    except (ValueError, TypeError):
                        await ws.close(code=1007, message=b"Invalid control")
                        break
                    if (control.get("audio_fs", SAMPLE_RATE) != SAMPLE_RATE
                            or control.get("wav_format", "pcm") != "pcm"
                            or control.get("mode", "offline") != "offline"):
                        await ws.close(code=1003, message=b"Expected offline 16kHz PCM16")
                        break
                    wav_name = control.get("wav_name", wav_name)
                    if control["is_speaking"]:
                        pcm.clear()
                        speaking = True
                        continue
                    if not speaking or len(pcm) % 2:
                        await ws.close(code=1007, message=b"Invalid PCM16 or sequence")
                        break
                    speaking = False
                    metrics = {"queue_ms": 0, "inference_ms": 0, "audio_s": 0}
                    try:
                        if pcm:
                            future = engine.submit(bytes(pcm))
                            text, metrics = await asyncio.wait_for(future, timeout)
                        else:
                            text = ""
                    except (BusyError, TimeoutError):
                        await ws.close(code=1013, message=b"ASR busy or deadline exceeded")
                        break
                    except RuntimeError:
                        await ws.close(code=1011, message=b"ASR inference failed")
                        break
                    await ws.send_json({"text": text, "is_final": True, "mode": "offline",
                                        "wav_name": wav_name, "metrics": metrics})
                    pcm.clear()
                elif msg.type == WSMsgType.ERROR:
                    break
        finally:
            if future is not None and not future.done():
                future.cancel()
            connections.discard(ws)
        return ws

    async def shutdown(app):
        state["ready"] = False
        await asyncio.gather(*(ws.close(code=1001) for ws in list(connections)))

    async def cleanup(app):
        if engine is not None:
            await engine.close()

    app = web.Application()
    app.router.add_get("/health", health)
    app.router.add_get("/", handle)
    app.on_startup.append(startup)
    app.on_shutdown.append(shutdown)
    app.on_cleanup.append(cleanup)
    return app


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(message)s")
    web.run_app(make_app(), host="0.0.0.0", port=10096, handler_cancellation=True,
                access_log=None)
