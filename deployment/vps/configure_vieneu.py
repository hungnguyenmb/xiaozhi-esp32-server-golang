"""Tạo cấu hình TTS riêng, kích hoạt hoặc quay về cấu hình mặc định đã lưu."""
import argparse
import json
import re
import urllib.request
from pathlib import Path

parser = argparse.ArgumentParser()
parser.add_argument("action", choices=["prepare", "activate", "rollback"])
args = parser.parse_args()
token = ""


def api(method, path, body=None):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    request = urllib.request.Request("http://127.0.0.1:8080" + path,
        data=json.dumps(body).encode() if body is not None else None, headers=headers, method=method)
    with urllib.request.urlopen(request, timeout=30) as response:
        return json.load(response)


credentials = json.loads(Path("/root/.config/xiaozhi-go/admin.json").read_text())
challenge = api("GET", "/api/captcha/challenge")
a, op, b = re.search(r"(\d+)\s*([+\-×*])\s*(\d+)", challenge["prompt"]).groups()
answer = int(a)+int(b) if op == "+" else int(a)-int(b) if op == "-" else int(a)*int(b)
credentials.update(captchaId=challenge["captchaId"], captchaAnswer=str(answer))
token = api("POST", "/api/login", credentials)["token"]
root = Path(__file__).resolve().parent
configs = api("GET", "/api/admin/configs")["data"]
saved = root / "runtime/vieneu_previous_tts.json"


def update(row, enabled=None, is_default=None):
    body = {k: row[k] for k in ["name", "provider", "json_data", "enabled", "is_default"]}
    if enabled is not None:
        body["enabled"] = enabled
    if is_default is not None:
        body["is_default"] = is_default
    api("PUT", "/api/admin/configs/" + str(row["id"]), body)


if args.action == "prepare":
    if not saved.exists():
        previous = [row for row in configs if row["type"] == "tts" and row["is_default"]]
        assert previous, "Không có cấu hình TTS mặc định để sao lưu"
        saved.touch(mode=0o600)
        saved.write_text(json.dumps(previous))
    if not any(row["config_id"] == "vieneu_tts" and row["type"] == "tts" for row in configs):
        secret = next(line.split("=", 1)[1] for line in (root / "runtime/vieneu.env").read_text().splitlines() if line.startswith("VIENEU_API_KEY="))
        data = {"api_url": "http://tts:8000/v1/audio/speech", "api_key": secret,
            "model": "vieneu-v3-nano", "voice": "Mai Anh", "response_format": "wav", "stream": True}
        api("POST", "/api/admin/configs", {"type": "tts", "name": "VieNeu Nano CPU — Mai Anh",
            "config_id": "vieneu_tts", "provider": "openai", "json_data": json.dumps(data, ensure_ascii=False),
            "enabled": True, "is_default": False})
        # v0.6.5 tự đặt cấu hình mới làm mặc định nếu không có dòng memory.
        for row in json.loads(saved.read_text()):
            update(row)
elif args.action == "activate":
    row = next(row for row in configs if row["type"] == "tts" and row["config_id"] == "vieneu_tts")
    data = json.loads(row["json_data"])
    data["model"] = "vieneu-v3-nano"
    row["name"] = "VieNeu Nano CPU — Mai Anh"
    row["json_data"] = json.dumps(data, ensure_ascii=False)
    update(row, enabled=True, is_default=True)
else:
    assert saved.exists(), "Chưa có snapshot TTS trước tích hợp"
    previous = json.loads(saved.read_text())
    for row in previous:
        update(row)

print("TTS_DEFAULT", [(row["config_id"], row["provider"]) for row in api("GET", "/api/admin/configs")["data"] if row["type"] == "tts" and row["is_default"]])
