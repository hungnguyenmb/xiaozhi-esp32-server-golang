"""Tạo cấu hình triển khai mới, chỉ chạy khi runtime chưa tồn tại."""
import json
import os
import secrets
from pathlib import Path

import yaml

root = Path(__file__).resolve().parent
runtime = root / "runtime"
config_dir = runtime / "config"
if (config_dir / "manager.json").exists():
    raise SystemExit("Runtime đã có cấu hình; không ghi đè")
for name in ("config", "data", "logs", "storage", "models/gipformer"):
    path = runtime / name
    path.mkdir(parents=True, exist_ok=True)
    os.chown(path, 10001, 10001)

internal = secrets.token_hex(32)
endpoint = secrets.token_hex(32)
device = secrets.token_hex(32)
main = yaml.safe_load((root / "bundle/main_config.yaml").read_text())
main["auth"] = {"enable": True}
main["manager"] = {"backend_url": "http://127.0.0.1:8080", "auth_token": internal, "endpoint_auth_token": endpoint, "history_timeout": "5s"}
main["mqtt"]["enable"] = False
main["mqtt_server"]["enable"] = False
main["voice_identify"]["enable"] = False
main["log"]["level"] = "info"
main["system_prompt"] = "Bạn là trợ lý Xiaozhi. Trả lời bằng tiếng Việt tự nhiên, ngắn gọn, hữu ích. Không đọc ký hiệu định dạng."
main["enable_greeting"] = False
main["greeting_list"] = ["Xin chào, tôi có thể giúp gì cho bạn?"]
main["wakeup_words"] = ["xiaozhi", "tiểu trí"]
main["resource_pools"].update(max_size=8, min_size=0, max_idle=4)
main["vad"]["ten_vad"]["pool_size"] = 4
main["asr"] = {"provider": "funasr", "funasr": {"host": "asr", "port": "10096", "mode": "offline", "sample_rate": 16000, "chunk_size": [5, 10, 5], "chunk_interval": 10, "max_connections": 4, "timeout": 30, "auto_end": False}}
main["tts"] = {"provider": "edge", "edge": {"voice": "vi-VN-HoaiMyNeural", "rate": "+0%", "volume": "+0%", "pitch": "+0Hz", "connect_timeout": 10, "receive_timeout": 30}}
main["llm"] = {"provider": "pending", "pending": {"type": "openai", "model_name": "configure-in-console", "api_key": "", "base_url": "http://127.0.0.1:9/v1", "max_tokens": 300}}
main["vision"]["vision_url"] = "http://194.233.73.211:8989/xiaozhi/api/vision"
main["vision"]["enable_auth"] = True
main["ota"]["signature_key"] = secrets.token_hex(32)
for scope in ("test", "external"):
    main["ota"][scope] = {"websocket": {"url": "ws://194.233.73.211:8989/xiaozhi/v1/", "token": device}, "mqtt": {"enable": False}}
manager = json.loads((root / "bundle/manager.json").read_text())
manager["server"] = {"port": "8080", "mode": "release"}
manager["database"] = {"type": "sqlite", "sqlite": {"file_path": "/app/data/xiaozhi.db"}}
manager["jwt"] = {"secret": secrets.token_hex(32), "expire_hour": 24}
manager["internal_auth_token"] = internal
manager["endpoint_auth_token"] = endpoint
manager["history"]["audio_base_path"] = "/app/data/chat_history/audio"

for name, contents in (("main_config.yaml", yaml.safe_dump(main, allow_unicode=True, sort_keys=False)), ("manager.json", json.dumps(manager, indent=2))):
    path = config_dir / name
    path.write_text(contents)
    path.chmod(0o600)
    os.chown(path, 10001, 10001)

nginx = '''map_hash_bucket_size 128;
map $http_upgrade $connection_upgrade { default upgrade; '' close; }
map $http_authorization $device_authorized { default 0; "Bearer DEVICE_TOKEN" 1; }
limit_req_zone $binary_remote_addr zone=ota:10m rate=5r/s;
server {
  listen 8989;
  server_name _;
  client_max_body_size 8m;
  location = /health { access_log off; return 200 "ok\\n"; }
  location = /xiaozhi/v1/ {
    if ($device_authorized = 0) { return 401; }
    proxy_pass http://backend:8989;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_read_timeout 3600s;
    proxy_send_timeout 3600s;
    proxy_buffering off;
  }
  location = /xiaozhi/ota/ {
    limit_req zone=ota burst=10 nodelay;
    proxy_pass http://backend:8989;
    proxy_set_header X-Real-IP $remote_addr;
  }
  location = /xiaozhi/ota/activate {
    limit_req zone=ota burst=10 nodelay;
    proxy_pass http://backend:8989;
    proxy_set_header X-Real-IP $remote_addr;
  }
  location = /xiaozhi/api/vision {
    if ($device_authorized = 0) { return 401; }
    proxy_pass http://backend:8989;
    proxy_set_header X-Real-IP $remote_addr;
  }
  location / { return 404; }
}
'''.replace("DEVICE_TOKEN", device)
(runtime / "nginx.conf").write_text(nginx)
(runtime / "nginx.conf").chmod(0o600)
credential_dir = Path("/root/.config/xiaozhi-go")
credential_dir.mkdir(parents=True, exist_ok=True, mode=0o700)
credentials = credential_dir / "admin.json"
credentials.write_text(json.dumps({"username": "admin", "password": secrets.token_urlsafe(24), "email": "admin@xiaozhi.invalid"}, indent=2))
credentials.chmod(0o600)
print("Đã tạo cấu hình và secrets riêng; không xuất secrets ra log")
