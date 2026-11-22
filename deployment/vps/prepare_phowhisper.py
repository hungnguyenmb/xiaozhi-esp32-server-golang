"""Tạo/kiểm tra artifact PhoWhisper INT8, không tải model trong request ASR."""

import argparse
import hashlib
import json
from importlib.metadata import version
from pathlib import Path


def checksums(root):
    result = {}
    for path in sorted(root.rglob("*")):
        if path.is_file() and path.name != "manifest.json":
            with path.open("rb") as stream:
                result[path.relative_to(root).as_posix()] = hashlib.file_digest(stream, "sha256").hexdigest()
    return result


def verify_model(root):
    root = Path(root)
    manifest = json.loads((root / "manifest.json").read_text())
    expected = json.loads(Path(__file__).with_name("phowhisper-manifest.json").read_text())
    for key in ("source", "revision", "quantization", "language", "runtime", "conversion"):
        if manifest[key] != expected[key]:
            raise ValueError(f"Model manifest mismatch: {key}")
    if not {"model.bin", "config.json", "tokenizer.json", "preprocessor_config.json"} <= manifest["sha256"].keys():
        raise ValueError("Incomplete model artifact")
    if checksums(root) != manifest["sha256"]:
        raise ValueError("Model checksum mismatch")
    for package, required in expected["runtime"].items():
        if version(package).split("+")[0] != required:
            raise ValueError(f"Runtime version mismatch: {package}")
    return manifest


def convert(output):
    from ctranslate2.converters import TransformersConverter
    from huggingface_hub import snapshot_download

    spec = json.loads(Path(__file__).with_name("phowhisper-manifest.json").read_text())
    for package, required in {**spec["runtime"], **spec["conversion"]}.items():
        if version(package).split("+")[0] != required:
            raise ValueError(f"Conversion version mismatch: {package}")
    source = snapshot_download(repo_id=spec["source"], revision=spec["revision"],
                               allow_patterns=["*.json", "*.txt", "pytorch_model.bin"])
    TransformersConverter(source, copy_files=["tokenizer.json", "preprocessor_config.json"]).convert(
        str(output), quantization="int8"
    )
    spec["sha256"] = checksums(output)
    (output / "manifest.json").write_text(json.dumps(spec, indent=2) + "\n")
    verify_model(output)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=["convert", "verify"])
    parser.add_argument("directory", type=Path)
    args = parser.parse_args()
    if args.action == "convert":
        convert(args.directory)
    else:
        verify_model(args.directory)
    print("MODEL_OK")
