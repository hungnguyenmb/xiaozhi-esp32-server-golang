"""Kiểm tra triển khai qua API localhost và WebSocket gateway."""
import json
import re
import subprocess
import urllib.request
from pathlib import Path


def api(method, path, body=None, token=""):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    req = urllib.request.Request("http://127.0.0.1:8080" + path, data=json.dumps(body).encode() if body is not None else None, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=90) as response:
        return json.load(response)


credentials = json.loads(Path("/root/.config/xiaozhi-go/admin.json").read_text())
challenge = api("GET", "/api/captcha/challenge")
a, op, b = re.search(r"(\d+)\s*([+\-×*])\s*(\d+)", challenge["prompt"]).groups()
answer = int(a) + int(b) if op == "+" else int(a) - int(b) if op == "-" else int(a) * int(b)
credentials.update(captchaId=challenge["captchaId"], captchaAnswer=str(answer))
token = api("POST", "/api/login", credentials)["token"]
assert api("GET", "/api/profile", token=token)["user"]["role"] == "admin"
test = api("POST", "/api/admin/configs/test", {"types": ["vad"], "config_ids": {"vad": ["vps_vad"]}}, token=token)
assert test["data"]["vad"]["vps_vad"]["ok"]
print("VAD_TEST", json.dumps(test, ensure_ascii=False))
code = '''import asyncio, json, aiohttp, time
async def main():
    async with aiohttp.ClientSession() as s:
        async with s.post('http://gateway:8989/xiaozhi/ota/', headers={'Device-Id':'02:00:00:00:00:01','Client-Id':'codex-smoke-test'}, json={}) as r:
            assert r.status == 200, await r.text()
            ota = await r.json()
        assert ota['websocket']['url'] == 'wss://otazeroone.hungnguyenaistem.online/xiaozhi/v1/'
        headers={'Authorization':'Bearer '+ota['websocket']['token'],'Device-Id':'02:00:00:00:00:01','Client-Id':'codex-smoke-test','Protocol-Version':'1'}
        async with s.ws_connect('http://gateway:8989/xiaozhi/v1/',headers=headers) as ws:
            await ws.send_json({'type':'hello','version':1,'transport':'websocket','features':{'mcp':True},'audio_params':{'format':'opus','sample_rate':16000,'channels':1,'frame_duration':60}})
            response=await ws.receive_json(timeout=20)
            assert response['type']=='hello', response
            print(json.dumps({'websocket_hello':response['type'],'transport':response.get('transport'),'ota_url':ota['websocket']['url'],'activation_required': bool(ota.get('activation'))}))
            async with s.post('http://backend:8989/admin/inject_msg', json={'device_id':'02:00:00:00:00:01','message':'Xin chào, đây là kiểm tra giọng nói tiếng Việt.','skip_llm':True,'auto_listen':False}) as r:
                assert r.status == 200, await r.text()
            audio_bytes = 0
            started = time.monotonic()
            first_audio_ms = None
            try:
                async with asyncio.timeout(25):
                    while True:
                        msg = await ws.receive()
                        if msg.type == aiohttp.WSMsgType.BINARY:
                            audio_bytes += len(msg.data)
                            if first_audio_ms is None:
                                first_audio_ms = round((time.monotonic()-started)*1000)
                        elif msg.type == aiohttp.WSMsgType.TEXT:
                            event = json.loads(msg.data)
                            if event.get('type') == 'tts' and event.get('state') == 'stop':
                                break
                        elif msg.type in (aiohttp.WSMsgType.CLOSE, aiohttp.WSMsgType.CLOSED, aiohttp.WSMsgType.ERROR):
                            break
            except TimeoutError:
                pass
            print(json.dumps({'vietnamese_tts_audio_bytes':audio_bytes,'first_audio_ms':first_audio_ms}))
            assert audio_bytes > 0, 'Vietnamese TTS returned no audio'
        async with s.get('http://gateway:8989/xiaozhi/v1/') as r:
            assert r.status==401, r.status
        for path in ['/admin/inject_msg','/api/system/configs','/xiaozhi/api/mcp/tools/test']:
            async with s.get('http://gateway:8989'+path) as r:
                assert r.status==404,(path,r.status)
        print('Gateway auth and internal route isolation: PASS')
asyncio.run(main())
'''
subprocess.run(["docker", "compose", "exec", "-T", "asr", "python", "-"], input=code.encode(), check=True)
