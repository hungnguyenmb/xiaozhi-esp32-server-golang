"""Chuyển/quay lui ASR mặc định bằng API Manager; không ghi trực tiếp SQLite."""

import argparse
import json
import os
import re
import urllib.request
from pathlib import Path

CONFIG_ID = "phowhisper_small_int8"
SETTINGS = {"host": "asr-phowhisper", "port": "10096", "mode": "offline",
            "sample_rate": 16000, "chunk_size": [5, 10, 5], "chunk_interval": 10,
            "max_connections": 4, "timeout": 30, "auto_end": False}


class Manager:
    def __init__(self, credentials):
        self.token = ""
        challenge = self.api("GET", "/api/captcha/challenge")
        a, op, b = re.search(r"(\d+)\s*([+\-×*])\s*(\d+)", challenge["prompt"]).groups()
        answer = int(a) + int(b) if op == "+" else int(a) - int(b) if op == "-" else int(a) * int(b)
        self.token = self.api("POST", "/api/login", {
            **credentials, "captchaId": challenge["captchaId"], "captchaAnswer": str(answer)
        })["token"]

    def api(self, method, path, body=None):
        headers = {"Content-Type": "application/json"}
        if self.token:
            headers["Authorization"] = "Bearer " + self.token
        req = urllib.request.Request("http://127.0.0.1:8080" + path,
                                     data=json.dumps(body).encode() if body is not None else None,
                                     headers=headers, method=method)
        with urllib.request.urlopen(req, timeout=30) as response:
            return json.load(response)

    def rows(self):
        return self.api("GET", "/api/admin/asr-configs")["data"]

    def set_default(self, row):
        body = {key: row[key] for key in ("name", "config_id", "provider", "json_data", "enabled")}
        body.update(is_default=True, enabled=True)
        self.api("PUT", "/api/admin/asr-configs/" + str(row["id"]), body)


def default(rows):
    selected = [row for row in rows if row["is_default"] and row["enabled"]]
    if len(selected) != 1:
        raise ValueError("Expected exactly one enabled default ASR")
    return selected[0]


def save_state(path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    temp = path.with_suffix(".tmp")
    fd = os.open(temp, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    os.chmod(temp, 0o600)
    with os.fdopen(fd, "w") as stream:
        json.dump(data, stream)
    os.replace(temp, path)


def configure(manager, action, saved):
    rows = manager.rows()
    current = default(rows)
    if action == "status":
        return current["config_id"]
    if action == "rollback":
        state = json.loads(saved.read_text())
        previous = next(row for row in rows if row["id"] == state["previous_id"])
        if current["id"] not in (previous["id"], state["candidate_id"]):
            raise ValueError("Default changed outside this rollout; refuse to overwrite")
        manager.set_default(previous)
        if default(manager.rows())["id"] != previous["id"]:
            raise RuntimeError("ASR rollback verification failed")
        return previous["config_id"]
    if current["config_id"] == CONFIG_ID:
        raise ValueError("PhoWhisper already active; rollback before another rollout")
    candidate = next((row for row in rows if row["config_id"] == CONFIG_ID), None)
    if candidate is None:
        # Typed endpoint avoids the legacy generic CreateConfig memory/default bug.
        candidate = manager.api("POST", "/api/admin/asr-configs", {
            "name": "PhoWhisper small INT8 CPU", "config_id": CONFIG_ID,
            "provider": "funasr", "json_data": json.dumps(SETTINGS),
            "enabled": True, "is_default": False,
        })["data"]
    if candidate["provider"] != "funasr" or json.loads(candidate["json_data"]) != SETTINGS:
        raise ValueError("Existing PhoWhisper config differs; inspect before activation")
    if default(manager.rows())["id"] != current["id"]:
        raise ValueError("ASR default changed while preparing candidate")
    save_state(saved, {"previous_id": current["id"], "candidate_id": candidate["id"]})
    try:
        manager.set_default(candidate)
        if default(manager.rows())["id"] != candidate["id"]:
            raise RuntimeError("ASR activation verification failed")
    except Exception:
        manager.set_default(current)
        raise
    return CONFIG_ID


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=["status", "activate", "rollback"])
    parser.add_argument("--state", type=Path, required=True)
    parser.add_argument("--credentials", type=Path, default=Path("/root/.config/xiaozhi-go/admin.json"))
    args = parser.parse_args()
    manager = Manager(json.loads(args.credentials.read_text()))
    print("ASR_DEFAULT", configure(manager, args.action, args.state))
