"""Entrypoint do GitHub Actions gọi trên VPS; trial không đổi ASR mặc định."""

import argparse
import hashlib
import json
import os
import re
import shutil
import subprocess
import tarfile
import time
from pathlib import Path

from configure_phowhisper import CONFIG_ID, Manager, configure, default, save_state

ROOT = Path("/opt/docker/antigravity/xiaozhi-go")


def run(*args, **kwargs):
    return subprocess.run(args, check=kwargs.pop("check", True), text=True, **kwargs)


def compose(release, *args, **kwargs):
    return run("docker", "compose", "--project-directory", str(ROOT),
               "--env-file", str(release / "release.env"), "-f", str(ROOT / "compose.yaml"),
               "-f", str(release / "compose.phowhisper.yaml"), *args, **kwargs)


def smoke(release):
    deadline = time.monotonic() + 180
    while time.monotonic() < deadline:
        container = compose(release, "ps", "-q", "asr-phowhisper", capture_output=True).stdout.strip()
        if container:
            info = json.loads(run("docker", "inspect", container, capture_output=True).stdout)[0]
            if info["State"].get("Health", {}).get("Status") == "healthy":
                break
        time.sleep(3)
    else:
        raise RuntimeError("PhoWhisper health deadline exceeded")
    network = next(iter(info["NetworkSettings"]["Networks"]))
    metadata = json.loads((release / "release.json").read_text())
    for extra in ([], ["--wav", "/checks/speech.wav", "--expect-speech"]):
        run("docker", "run", "--rm", "--network", network, "--read-only",
            "--cap-drop", "ALL", "--security-opt", "no-new-privileges:true",
            "-v", f"{release}:/checks:ro", "--entrypoint", "python", metadata["image_id"],
            "/checks/verify_phowhisper.py", "--endpoint", "ws://asr-phowhisper:10096/", *extra)


def verify_public(release):
    metadata = json.loads((release / "release.json").read_text())
    run("docker", "run", "--rm", "--read-only", "--cap-drop", "ALL",
        "-v", f"{release}:/checks:ro", "--entrypoint", "python", metadata["image_id"],
        "/checks/verify_tls.py")


def install(payload, manager):
    if default(manager.rows())["config_id"] == CONFIG_ID:
        raise ValueError("PhoWhisper active; rollback before replacing the trial service")
    metadata = json.loads((payload / "release.json").read_text())
    release_id = metadata["release_id"]
    if not re.fullmatch(r"[a-f0-9]{40}-[0-9]+-[0-9]+", release_id):
        raise ValueError("Invalid release ID")
    if metadata["image"] != "xiaozhi-phowhisper:" + release_id:
        raise ValueError("Invalid image tag")
    for name in ("image.tar.gz", "model.tar.gz", "speech.wav"):
        with (payload / name).open("rb") as stream:
            actual = hashlib.file_digest(stream, "sha256").hexdigest()
        if actual != metadata["sha256"][name]:
            raise ValueError(f"Artifact checksum mismatch: {name}")
    release = ROOT / "runtime/phowhisper-releases" / release_id
    if release.exists():
        raise ValueError("Release already exists; dispatch a new workflow run")
    release.mkdir(parents=True, mode=0o755)
    release.chmod(0o755)
    for name in ("release.json", "compose.phowhisper.yaml", "verify_phowhisper.py", "verify_tls.py", "speech.wav"):
        shutil.copyfile(payload / name, release / name)
        (release / name).chmod(0o644)
    model = release / "models"
    model.mkdir()
    model.chmod(0o755)
    with tarfile.open(payload / "model.tar.gz") as archive:
        archive.extractall(model, filter="data")
    # Image's non-root user needs read access; no credentials in model artifact.
    for item in model.rglob("*"):
        item.chmod(0o755 if item.is_dir() else 0o644)
    image_id = metadata["image_id"]
    if not re.fullmatch(r"sha256:[a-f0-9]{64}", image_id):
        raise ValueError("Invalid image digest")
    run("docker", "load", "-i", str(payload / "image.tar.gz"))
    actual = run("docker", "image", "inspect", metadata["image"], "--format", "{{.Id}}",
                 capture_output=True).stdout.strip()
    if actual != image_id:
        raise ValueError("Loaded image digest mismatch")
    (release / "release.env").write_text(
        f"PHOWHISPER_IMAGE={image_id}\nPHOWHISPER_MODEL_DIR={model}\n"
    )
    current_file = ROOT / "runtime/phowhisper-current.json"
    previous = json.loads(current_file.read_text()) if current_file.exists() else None
    try:
        compose(release, "up", "-d", "--no-deps", "asr-phowhisper")
        smoke(release)
        verify_public(release)
    except Exception:
        if previous:
            compose(Path(previous["release"]), "up", "-d", "--no-deps", "asr-phowhisper")
            smoke(Path(previous["release"]))
        else:
            compose(release, "stop", "asr-phowhisper")
        raise
    save_state(current_file, {"release": str(release), "image_id": image_id,
                              "previous": previous["release"] if previous else None})
    print("PHOWHISPER_TRIAL_READY", release_id)


def benchmark(release, manager):
    suite = ROOT / "runtime/asr-benchmark/suite.json"
    samples = json.loads(suite.read_text())
    if (sum(bool(s["reference"].strip()) for s in samples) < 30
            or sum(not s["reference"].strip() for s in samples) < 20):
        raise ValueError("Benchmark needs >=30 speech and >=20 silence/noise references")
    previous = default(manager.rows())
    if previous["config_id"] == CONFIG_ID or previous["provider"] != "funasr":
        raise ValueError("Benchmark needs the original FunASR default for A/B")
    settings = json.loads(previous["json_data"])
    smoke(release)
    container = compose(release, "ps", "-q", "asr-phowhisper", capture_output=True).stdout.strip()
    info = json.loads(run("docker", "inspect", container, capture_output=True).stdout)[0]
    network = next(iter(info["NetworkSettings"]["Networks"]))
    metadata = json.loads((release / "release.json").read_text())
    reports = ROOT / "runtime/asr-benchmark/reports" / str(time.time_ns())
    reports.mkdir(parents=True)
    failed = False
    for name, endpoint in (("gipformer", "ws://{}:{}/".format(settings["host"], settings["port"])),
                           ("phowhisper", "ws://asr-phowhisper:10096/")):
        for concurrency in (1, 2, 4):
            result = run(
                "docker", "run", "--rm", "--network", network, "--read-only", "--user", "0:0",
                "--cap-drop", "ALL", "--security-opt", "no-new-privileges:true",
                "-v", f"{release}:/checks:ro", "-v", f"{suite.parent}:/samples:ro",
                "--entrypoint", "python", metadata["image_id"], "/checks/verify_phowhisper.py",
                "--endpoint", endpoint, "--suite", "/samples/suite.json", "--concurrency", str(concurrency),
                capture_output=True, check=False,
            )
            if not result.stdout.strip():
                raise RuntimeError(f"Benchmark process failed before report: {name}/{concurrency}")
            report = json.loads(result.stdout)
            report["exit_code"] = result.returncode
            failed |= result.returncode != 0
            save_state(reports / f"{name}-{concurrency}.json", report)
            print(json.dumps({"model": name, "concurrency": concurrency,
                              **{k: v for k, v in report.items() if k != "rows"}}))
    print("BENCHMARK_REPORTS", reports)
    print("Validate TTS under concurrent load and review metrics before activation.")
    if failed:
        raise RuntimeError("Benchmark contains failed silence/noise assertions; inspect saved reports")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=["trial", "benchmark", "activate", "rollback"])
    parser.add_argument("--payload", type=Path)
    parser.add_argument("--benchmark-run", default="")
    args = parser.parse_args()
    os.umask(0o077)
    manager = Manager(json.loads(Path("/root/.config/xiaozhi-go/admin.json").read_text()))
    snapshot = ROOT / "runtime/phowhisper-asr-state.json"
    if args.action == "trial":
        install(args.payload, manager)
        return
    if args.action == "benchmark":
        release = Path(json.loads((ROOT / "runtime/phowhisper-current.json").read_text())["release"])
        benchmark(release, manager)
        return
    if args.action == "rollback":
        print("ASR_DEFAULT", configure(manager, "rollback", snapshot))
        # Verify old ASR contract; leaves PhoWhisper available for diagnosis.
        release = Path(json.loads((ROOT / "runtime/phowhisper-current.json").read_text())["release"])
        metadata = json.loads((release / "release.json").read_text())
        container = compose(release, "ps", "-q", "asr", capture_output=True).stdout.strip()
        info = json.loads(run("docker", "inspect", container, capture_output=True).stdout)[0]
        network = next(iter(info["NetworkSettings"]["Networks"]))
        prior = default(manager.rows())
        settings = json.loads(prior["json_data"])
        if prior["provider"] != "funasr":
            raise ValueError("Rollback restored non-FunASR config; requires provider-specific smoke test")
        endpoint = "ws://{}:{}/".format(settings["host"], settings["port"])
        run("docker", "run", "--rm", "--network", network, "-v", f"{release}:/checks:ro",
            "--entrypoint", "python", metadata["image_id"], "/checks/verify_phowhisper.py",
            "--endpoint", endpoint, "--wav", "/checks/speech.wav", "--expect-speech")
        verify_public(release)
        return
    if not args.benchmark_run.strip():
        raise ValueError("Record benchmark evidence before activation")
    release = Path(json.loads((ROOT / "runtime/phowhisper-current.json").read_text())["release"])
    smoke(release)
    activated = False
    try:
        print("ASR_DEFAULT", configure(manager, "activate", snapshot))
        activated = True
        smoke(release)
        verify_public(release)
    except Exception:
        if activated:
            configure(manager, "rollback", snapshot)
        raise
    save_state(ROOT / "runtime/phowhisper-activation.json", {
        "release": str(release), "benchmark_run": args.benchmark_run,
    })
    print("Existing robot sessions may need reconnect; verify WSS and real robot audio.")


if __name__ == "__main__":
    main()
