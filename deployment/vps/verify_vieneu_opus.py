"""Kiểm tra WAV → Go → Opus WSS bằng thiết bị mô phỏng, giải mã frame bằng libopus."""
import base64
import ctypes
import json
import math
import subprocess
from pathlib import Path

code = '''import asyncio,base64,json,time,aiohttp
async def main():
    async with aiohttp.ClientSession() as s:
        async with s.post('https://otazeroone.hungnguyenaistem.online/xiaozhi/ota/',headers={'Device-Id':'02:00:00:00:00:01','Client-Id':'codex-vieneu-test'},json={}) as r:
            assert r.status==200
            ota=await r.json()
        headers={'Device-Id':'02:00:00:00:00:01','Client-Id':'codex-vieneu-test','Authorization':'Bearer '+ota['websocket']['token'],'Protocol-Version':'1'}
        async with s.ws_connect(ota['websocket']['url'],headers=headers) as ws:
            await ws.send_json({'type':'hello','version':1,'transport':'websocket','features':{'mcp':False},'audio_params':{'format':'opus','sample_rate':16000,'channels':1,'frame_duration':60}})
            hello=await ws.receive_json(timeout=15)
            assert hello['type']=='hello'
            start=time.monotonic()
            async with s.post('http://backend:8989/admin/inject_msg',json={'device_id':'02:00:00:00:00:01','message':'Xin chào, đây là giọng nói tiếng Việt chạy trên máy chủ của bạn, âm thanh được chuyển thành các gói nhỏ và gửi liên tục để thiết bị có thể phát trong khi phần còn lại đang được tổng hợp.','skip_llm':True,'auto_listen':False}) as r:
                assert r.status==200,await r.text()
            packets=[];first=None;stopped=False
            async with asyncio.timeout(90):
                while True:
                    msg=await ws.receive()
                    if msg.type==aiohttp.WSMsgType.BINARY:
                        if first is None:first=round((time.monotonic()-start)*1000)
                        packets.append(base64.b64encode(msg.data).decode())
                    elif msg.type==aiohttp.WSMsgType.TEXT:
                        event=json.loads(msg.data)
                        if event.get('type')=='tts' and event.get('state')=='stop' and packets:
                            stopped=True;break
                    elif msg.type in (aiohttp.WSMsgType.CLOSE,aiohttp.WSMsgType.CLOSED,aiohttp.WSMsgType.ERROR):
                        break
            elapsed=round((time.monotonic()-start)*1000)
            assert packets and stopped and first<elapsed-100
            print(json.dumps({'first_opus_ms':first,'completed_ms':elapsed,'hello_audio':hello.get('audio_params'),'packets':packets}))
asyncio.run(main())
'''
result = subprocess.run(["docker", "compose", "exec", "-T", "asr", "python", "-u", "-"], input=code.encode(), stdout=subprocess.PIPE, check=True)
data = json.loads(result.stdout)
library = Path("runtime/vieneu-test-libopus.so")
if not library.exists():
    container = subprocess.check_output(["docker", "compose", "ps", "-q", "backend"], text=True).strip()
    subprocess.run(["docker", "cp", "-L", container + ":/lib/x86_64-linux-gnu/libopus.so.0", str(library)], check=True, stdout=subprocess.DEVNULL)
opus = ctypes.CDLL(str(library.resolve()))
opus.opus_decoder_create.argtypes = [ctypes.c_int, ctypes.c_int, ctypes.POINTER(ctypes.c_int)]
opus.opus_decoder_create.restype = ctypes.c_void_p
opus.opus_decode.argtypes = [ctypes.c_void_p, ctypes.c_void_p, ctypes.c_int, ctypes.POINTER(ctypes.c_int16), ctypes.c_int, ctypes.c_int]
opus.opus_decode.restype = ctypes.c_int
opus.opus_decoder_destroy.argtypes = [ctypes.c_void_p]
error = ctypes.c_int()
decoder = opus.opus_decoder_create(24000, 1, ctypes.byref(error))
assert decoder and error.value == 0
samples = 0
energy = 0
sizes = set()
audio_bytes = 0
try:
    for encoded in data.pop("packets"):
        packet = base64.b64decode(encoded)
        audio_bytes += len(packet)
        pcm = (ctypes.c_int16 * 2880)()
        count = opus.opus_decode(decoder, packet, len(packet), pcm, len(pcm), 0)
        assert count == 1440, count  # 60ms @24kHz
        sizes.add(count)
        samples += count
        energy += sum(x*x for x in pcm[:count])
finally:
    opus.opus_decoder_destroy(decoder)
assert samples and energy > 0
data.update(opus_bytes=audio_bytes, decoded_seconds=round(samples/24000, 3),
    frame_ms=60, decoded_rms=round(math.sqrt(energy/samples), 1), streaming_before_completion=True)
print(json.dumps(data, ensure_ascii=False))
