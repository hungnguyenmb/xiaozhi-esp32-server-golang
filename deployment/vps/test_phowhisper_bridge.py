"""Contract và backpressure không cần model weights; không thay cho benchmark thật."""

import asyncio
import threading
import unittest
from unittest.mock import patch

from aiohttp import WSMsgType, WSServerHandshakeError
from aiohttp.test_utils import TestClient, TestServer

from phowhisper_bridge import BusyError, InferenceQueue, make_app


class BridgeTests(unittest.IsolatedAsyncioTestCase):
    async def asyncSetUp(self):
        self.calls = []

        def recognize(pcm):
            self.calls.append(pcm)
            return "Xin chào Việt Nam"

        self.client = TestClient(TestServer(make_app(recognize, max_audio_bytes=3200),
                                            handler_cancellation=True))
        await self.client.start_server()

    async def asyncTearDown(self):
        await self.client.close()

    async def test_multiple_utterances_preserve_names_and_pcm(self):
        async with self.client.ws_connect("/") as ws:
            for name in ("robot-a", "robot-b"):
                await ws.send_json({"is_speaking": True, "audio_fs": 16000,
                                    "wav_name": name, "mode": "offline", "wav_format": "pcm"})
                await ws.send_bytes(b"\x01\x00" * 16)
                await ws.send_json({"is_speaking": False})
                result = await ws.receive_json()
                self.assertEqual(result["text"], "Xin chào Việt Nam")
                self.assertEqual(result["wav_name"], name)
                self.assertTrue(result["is_final"])
                self.assertEqual(result["mode"], "offline")
                self.assertIn("inference_ms", result["metrics"])
        self.assertEqual(self.calls, [b"\x01\x00" * 16] * 2)

    async def test_empty_does_not_call_model(self):
        async with self.client.ws_connect("/") as ws:
            await ws.send_json({"is_speaking": True})
            await ws.send_json({"is_speaking": False})
            self.assertEqual((await ws.receive_json())["text"], "")
        self.assertEqual(self.calls, [])

    async def assert_control_rejected(self, data, code):
        async with self.client.ws_connect("/") as ws:
            await ws.send_str(data)
            msg = await ws.receive()
            self.assertEqual(msg.type, WSMsgType.CLOSE)
            self.assertEqual(msg.data, code)

    async def test_invalid_control(self):
        for raw in ("[]", "broken", '{"is_speaking":1}', '{"is_speaking":true,"wav_name":[]}'):
            await self.assert_control_rejected(raw, 1007)

    async def test_wrong_format_and_rate(self):
        for extra in ('"audio_fs":8000', '"wav_format":"wav"', '"mode":"online"'):
            await self.assert_control_rejected('{"is_speaking":true,' + extra + '}', 1003)

    async def test_invalid_pcm_and_sequence(self):
        async with self.client.ws_connect("/") as ws:
            await ws.send_json({"is_speaking": True})
            await ws.send_bytes(b"x")
            await ws.send_json({"is_speaking": False})
            self.assertEqual((await ws.receive()).data, 1007)
        async with self.client.ws_connect("/") as ws:
            await ws.send_bytes(b"xx")
            self.assertEqual((await ws.receive()).data, 1008)

    async def test_audio_limit_across_chunks(self):
        async with self.client.ws_connect("/") as ws:
            await ws.send_json({"is_speaking": True})
            await ws.send_bytes(b"x" * 2000)
            await ws.send_bytes(b"x" * 2000)
            self.assertEqual((await ws.receive()).data, 1009)

    async def test_health(self):
        response = await self.client.get("/health")
        self.assertEqual(response.status, 200)
        self.assertTrue((await response.json())["ready"])

    async def test_disconnect_discards_unfinished_audio(self):
        async with self.client.ws_connect("/") as ws:
            await ws.send_json({"is_speaking": True})
            await ws.send_bytes(b"xx")
        await asyncio.sleep(0)
        self.assertEqual(self.calls, [])

    async def test_connection_limit(self):
        client = TestClient(TestServer(make_app(lambda pcm: "ok", max_connections=1)))
        await client.start_server()
        try:
            async with client.ws_connect("/"):
                with self.assertRaises(WSServerHandshakeError) as caught:
                    await client.ws_connect("/")
                self.assertEqual(caught.exception.status, 503)
        finally:
            await client.close()

    async def test_startup_failure_never_serves_ready(self):
        with patch("phowhisper_bridge.load_recognizer", side_effect=ValueError("Bad checksum")):
            client = TestClient(TestServer(make_app()))
            with self.assertRaisesRegex(ValueError, "Bad checksum"):
                await client.start_server()
            await client.close()


class QueueTests(unittest.IsolatedAsyncioTestCase):
    async def test_queued_deadline_expires_before_native_inference(self):
        release = threading.Event()
        entered = threading.Event()
        calls = []

        def recognize(pcm):
            calls.append(pcm)
            entered.set()
            release.wait(3)
            return "ok"

        engine = InferenceQueue(recognize, timeout=0.01)
        engine.start()
        try:
            first = engine.submit(b"first")
            while not entered.is_set():
                await asyncio.sleep(0.001)
            second = engine.submit(b"second")
            await asyncio.sleep(0.02)
            release.set()
            await first
            with self.assertRaises(TimeoutError):
                await second
            self.assertEqual(calls, [b"first"])
        finally:
            release.set()
            await engine.close()

    async def test_timeout_keeps_worker_occupied_and_skips_cancelled(self):
        entered = threading.Event()
        release = threading.Event()
        calls = []

        def recognize(pcm):
            calls.append(pcm)
            entered.set()
            release.wait(3)
            return "done"

        engine = InferenceQueue(recognize, capacity=1, timeout=0.04)
        engine.start()
        try:
            first = engine.submit(b"first")
            while not entered.is_set():
                await asyncio.sleep(0.001)
            second = engine.submit(b"second")
            with self.assertRaises(BusyError):
                engine.submit(b"third")
            with self.assertRaises(TimeoutError):
                await asyncio.wait_for(first, 0.04)
            self.assertTrue(engine.running)
            second.cancel()
            release.set()
            await engine.queue.join()
            self.assertEqual(calls, [b"first"])
        finally:
            release.set()
            await engine.close()

    async def test_worker_survives_model_error(self):
        def recognize(pcm):
            if pcm == b"fail":
                raise ValueError("test failure")
            return "ok"

        engine = InferenceQueue(recognize)
        engine.start()
        try:
            with self.assertLogs("phowhisper_bridge", level="ERROR"):
                with self.assertRaisesRegex(RuntimeError, "ASR inference failed"):
                    await engine.submit(b"fail")
            self.assertEqual((await engine.submit(b"pass"))[0], "ok")
        finally:
            await engine.close()


if __name__ == "__main__":
    unittest.main()
