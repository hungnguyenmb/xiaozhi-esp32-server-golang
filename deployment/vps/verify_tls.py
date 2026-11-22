"""Kiểm tra certificate, OTA HTTPS và bắt tay WSS qua tên miền public."""
import asyncio
import json
import socket
import ssl

import aiohttp

DOMAIN = "otazeroone.hungnguyenaistem.online"


async def main():
    context = ssl.create_default_context()
    with socket.create_connection((DOMAIN, 443), timeout=10) as raw:
        with context.wrap_socket(raw, server_hostname=DOMAIN) as conn:
            cert = conn.getpeercert()
            print(json.dumps({"tls": conn.version(), "subject_alt_names": cert["subjectAltName"], "expires": cert["notAfter"]}))
    async with aiohttp.ClientSession(timeout=aiohttp.ClientTimeout(total=30)) as session:
        async with session.post("https://" + DOMAIN + "/xiaozhi/ota/", headers={"Device-Id": "02:00:00:00:00:01", "Client-Id": "codex-smoke-test"}, json={}) as response:
            assert response.status == 200, response.status
            ota = await response.json()
        expected = "wss://" + DOMAIN + "/xiaozhi/v1/"
        assert ota["websocket"]["url"] == expected
        headers = {"Authorization": "Bearer " + ota["websocket"]["token"], "Device-Id": "02:00:00:00:00:01", "Client-Id": "codex-smoke-test", "Protocol-Version": "1"}
        async with session.ws_connect(expected, headers=headers) as ws:
            await ws.send_json({"type": "hello", "version": 1, "transport": "websocket", "features": {"mcp": True}, "audio_params": {"format": "opus", "sample_rate": 16000, "channels": 1, "frame_duration": 60}})
            hello = await ws.receive_json(timeout=20)
            assert hello["type"] == "hello" and hello["transport"] == "websocket"
        async with session.get("https://" + DOMAIN + "/xiaozhi/v1/") as response:
            assert response.status == 401, response.status
        async with session.get("https://" + DOMAIN + "/admin/inject_msg") as response:
            assert response.status == 404, response.status
        async with session.get("http://" + DOMAIN + "/xiaozhi/ota/", allow_redirects=False) as response:
            assert response.status == 308 and response.headers["Location"] == "https://" + DOMAIN + "/xiaozhi/ota/"
        print(json.dumps({"ota_https": "PASS", "wss_hello": "PASS", "unauthenticated_ws": 401, "internal_route": 404, "http_redirect": 308, "websocket_url": expected}))


asyncio.run(main())
