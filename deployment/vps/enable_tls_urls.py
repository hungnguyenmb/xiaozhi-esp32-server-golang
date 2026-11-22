"""Đồng bộ URL HTTPS/WSS vào YAML và Manager; giữ nguyên token."""
import json
import os
import re
import urllib.request
from pathlib import Path

import yaml

DOMAIN = "otazeroone.hungnguyenaistem.online"
ROOT = Path(__file__).resolve().parent
BASE = "http://127.0.0.1:8080"
token = ""


def api(method, path, body=None):
    headers = {"Content-Type": "application/json"}
    if token:
        headers["Authorization"] = "Bearer " + token
    req = urllib.request.Request(BASE + path, data=json.dumps(body).encode() if body is not None else None, headers=headers, method=method)
    with urllib.request.urlopen(req, timeout=30) as response:
        return json.load(response)


credentials = json.loads(Path("/root/.config/xiaozhi-go/admin.json").read_text())
challenge = api("GET", "/api/captcha/challenge")
a, op, b = re.search(r"(\d+)\s*([+\-×*])\s*(\d+)", challenge["prompt"]).groups()
answer = int(a) + int(b) if op == "+" else int(a) - int(b) if op == "-" else int(a) * int(b)
credentials.update(captchaId=challenge["captchaId"], captchaAnswer=str(answer))
token = api("POST", "/api/login", credentials)["token"]
configs = api("GET", "/api/admin/configs")["data"]

for config in configs:
    if config["type"] not in ("ota", "vision"):
        continue
    data = json.loads(config["json_data"])
    changed = False
    if config["type"] == "ota":
        for scope in ("test", "external"):
            if scope in data:
                data[scope].setdefault("websocket", {})["url"] = "wss://" + DOMAIN + "/xiaozhi/v1/"
                changed = True
    elif "vision_url" in data:
        data["vision_url"] = "https://" + DOMAIN + "/xiaozhi/api/vision"
        changed = True
    if changed:
        config["json_data"] = json.dumps(data, ensure_ascii=False)
        api("PUT", "/api/admin/configs/" + str(config["id"]), config)
        print("Đã đồng bộ Manager:", config["type"], config["config_id"])

path = ROOT / "runtime/config/main_config.yaml"
main = yaml.safe_load(path.read_text())
for scope in ("test", "external"):
    main["ota"][scope]["websocket"]["url"] = "wss://" + DOMAIN + "/xiaozhi/v1/"
main["vision"]["vision_url"] = "https://" + DOMAIN + "/xiaozhi/api/vision"
# Ghi vào inode hiện tại để container đang bind mount tiếp tục đọc đúng file.
path.write_text(yaml.safe_dump(main, allow_unicode=True, sort_keys=False))
path.chmod(0o600)
os.chown(path, 10001, 10001)
print("Đã cập nhật YAML; token không thay đổi")
