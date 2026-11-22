"""Khởi tạo admin và các cấu hình qua API localhost; không in secrets."""
import json
import re
import urllib.request
from pathlib import Path

import yaml

root = Path(__file__).resolve().parent
base = "http://127.0.0.1:8080"
token = ""


def api(method, path, payload=None):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    req = urllib.request.Request(base + path, data=json.dumps(payload).encode() if payload is not None else None, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=30) as response:
        return json.load(response)


credentials = json.loads(Path("/root/.config/xiaozhi-go/admin.json").read_text())
if api("GET", "/api/setup/status")["needs_setup"]:
    api("POST", "/api/setup/initialize", {"admin_" + key: value for key, value in credentials.items()})
login = {"username": credentials["username"], "password": credentials["password"]}
if api("GET", "/api/captcha/status")["enabled"]:
    challenge = api("GET", "/api/captcha/challenge")
    match = re.search(r"(\d+)\s*([+\-×*])\s*(\d+)", challenge["prompt"])
    if not match:
        raise SystemExit("Không đọc được phép tính đăng nhập của instance riêng")
    a, op, b = match.groups()
    answer = int(a) + int(b) if op == "+" else int(a) - int(b) if op == "-" else int(a) * int(b)
    login.update(captchaId=challenge["captchaId"], captchaAnswer=str(answer))
token = api("POST", "/api/login", login)["token"]
main = yaml.safe_load((root / "runtime/config/main_config.yaml").read_text())


def seed(kind, name, provider, data):
    api("POST", "/api/admin/configs", {"type": kind, "name": name, "config_id": "vps_" + kind, "provider": provider, "json_data": json.dumps(data, ensure_ascii=False), "enabled": True, "is_default": True})
    print("Đã cấu hình:", kind, provider)


seed("vad", "TEN VAD", "ten_vad", main["vad"]["ten_vad"])
seed("asr", "Gipformer tiếng Việt CPU INT8", "funasr", main["asr"]["funasr"])
seed("tts", "Giọng Việt Hoài My", "edge", main["tts"]["edge"])
seed("llm", "Chờ cấu hình API LLM", "openai", main["llm"]["pending"])
seed("voice_identify", "Tắt nhận diện người nói", "service", {"enable": False, "service": {"base_url": "", "threshold": 0.6}})
seed("ota", "OTA VPS", "ota", main["ota"])
seed("mqtt", "Tắt MQTT", "mqtt", {"enable": False})
seed("mqtt_server", "Tắt MQTT server", "mqtt_server", {"enable": False})
api("PUT", "/api/admin/chat-settings", {"auth": {"enable": True, "login_captcha_enabled": True}, "chat": {"max_idle_duration": 60000, "chat_max_silence_duration": 500, "realtime_mode": 4, "global_system_prompt": main["system_prompt"]}})
assert api("GET", "/api/profile")["user"]["role"] == "admin"
print("Đăng nhập admin và cấu hình cơ bản thành công; LLM chưa có API key")
