"""Triển khai dist đã build qua symlink nguyên tử; không restart backend/âm thanh."""
import datetime
import os
from pathlib import Path
import shutil
import sqlite3
import subprocess
import tarfile

root = Path(__file__).resolve().parent
stamp = datetime.datetime.now(datetime.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
backup = root.parent / "backups" / ("xiaozhi-console-vi-" + stamp)
backup.mkdir(mode=0o700)
site = Path("/etc/nginx/sites-available/adminzeroone.hungnguyenaistem.online.conf")
shutil.copy2(site, backup / "admin-nginx.conf")
src = sqlite3.connect(root / "runtime/data/xiaozhi.db")
dst = sqlite3.connect(backup / "xiaozhi.db")
src.backup(dst)
dst.close()
src.close()
frontend = root / "frontend"
frontend.mkdir(exist_ok=True)
release = frontend / "releases" / ("vi-" + stamp)
release.mkdir(parents=True)
with tarfile.open("/tmp/xiaozhi-ui-vi.tgz", "r:gz") as archive:
    archive.extractall(release, filter="data")
assert (release / "index.html").exists()
assert '<html lang="vi">' in (release / "index.html").read_text()
# All public files must be readable by nginx; the archive contains dist only.
for item in release.rglob("*"):
    item.chmod(0o755 if item.is_dir() else 0o644)
shared = frontend / "public/assets"
shared.mkdir(parents=True, exist_ok=True)
shutil.copytree(release / "assets", shared, dirs_exist_ok=True)
current = frontend / "current"
previous = os.readlink(current) if current.is_symlink() else None
if previous:
    (backup / "previous-frontend.txt").write_text(previous)
pending = frontend / ("current-" + stamp)
pending.symlink_to(release)
pending.replace(current)
shutil.copy2(root / "adminzeroone.nginx.conf", site)
try:
    subprocess.run(["nginx", "-t"], check=True)
    subprocess.run(["systemctl", "reload", "nginx"], check=True)
except Exception:
    shutil.copy2(backup / "admin-nginx.conf", site)
    if previous:
        fallback = frontend / ("rollback-" + stamp)
        fallback.symlink_to(previous)
        fallback.replace(current)
    subprocess.run(["nginx", "-t"], check=True)
    subprocess.run(["systemctl", "reload", "nginx"], check=True)
    raise
print("FRONTEND_RELEASE", release)
print("BACKUP", backup)
