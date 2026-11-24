"""Chuẩn bị speech+noise và chấm replay; không đưa dữ liệu riêng lên Git.

mix: tạo suite riêng tư, giữ source_id để chia tập theo nguồn trước hiệu chỉnh.
score: phân biệt VAD acceptance và đúng transcript, không in transcript.
Chỉ dùng thư viện chuẩn Python; không gọi LLM/TTS hoặc tự deploy.
"""
import argparse
import array
import json
import math
import os
from pathlib import Path
import random
import sys
import unicodedata
import wave


def normalize(text):
    text = unicodedata.normalize("NFC", text).lower()
    return " ".join("".join(" " if unicodedata.category(c).startswith("P")
                            else c for c in text).split())


def distance(a, b):
    row = list(range(len(b) + 1))
    for i, x in enumerate(a, 1):
        new = [i]
        for j, y in enumerate(b, 1):
            new.append(min(new[-1] + 1, row[j] + 1, row[j - 1] + (x != y)))
        row = new
    return row[-1]


def score(suite, rows):
    by_id = {r["ID"]: r for r in rows}
    if len(by_id) != len(rows) or set(by_id) != {s["id"] for s in suite}:
        raise ValueError("Replay IDs must match the full suite exactly")
    metrics = {}
    for sample in suite:
        group = sample.get("group", "speech" if sample["reference"] else "noise")
        m = metrics.setdefault(group, dict(samples=0, speech=0, missed_speech=0,
            noise=0, accepted_noise=0, word_errors=0, words=0, processing_ms=0,
            audio_ms=0, open_segments=0))
        row = by_id[sample["id"]]
        m["samples"] += 1
        m["processing_ms"] += row["ProcessingMs"]
        m["audio_ms"] += row["AudioMs"]
        segments = row.get("Segments") or []
        m["open_segments"] += sum(bool(s["Open"]) for s in segments)
        if sample["reference"]:
            m["speech"] += 1
            m["missed_speech"] += not bool(row["Accepted"])
            # Text is deliberately opt-in to the Go replay command.
            if segments and not all("Text" in s for s in segments):
                raise ValueError("Need -include-text for WER, keep output private")
            ref = normalize(sample["reference"]).split()
            hyp = normalize(" ".join(s["Text"] for s in segments)).split()
            m["words"] += len(ref)
            m["word_errors"] += distance(ref, hyp)
        else:
            m["noise"] += 1
            m["accepted_noise"] += bool(row["Accepted"])
    for m in metrics.values():
        m["wer"] = m["word_errors"] / m["words"] if m["words"] else None
        m["dsp_rtf"] = m["processing_ms"] / m["audio_ms"] if m["audio_ms"] else None
    return metrics


def mix(pcm, rate, snr, kind, rng):
    if not pcm:
        raise ValueError("empty speech")
    if kind == "fan":
        noise = [0.5 * math.sin(2 * math.pi * 100 * i / rate)
                 + 0.2 * math.sin(2 * math.pi * 200 * i / rate)
                 + rng.gauss(0, 0.15) for i in range(len(pcm))]
    elif kind == "click":
        noise = [rng.gauss(0, 0.02) for _ in pcm]
        for start in range(rate // 5, len(pcm), rate // 2):
            for i in range(start, min(start + rate // 100, len(pcm))):
                noise[i] += rng.uniform(-1, 1)
    else:
        raise ValueError("unknown noise")
    speech_power = sum(x * x for x in pcm) / len(pcm)
    noise_power = sum(x * x for x in noise) / len(noise)
    if not speech_power:
        raise ValueError("reference speech has zero energy")
    gain = math.sqrt(speech_power / (noise_power * 10 ** (snr / 10)))
    combined = [x + gain * n for x, n in zip(pcm, noise)]
    # Joint scaling preserves SNR while avoiding clipping; do not peak-normalize
    # the speech and noise independently after mixing.
    peak = max(abs(x) for x in combined)
    scale = min(1, .95 / peak) if peak else 1
    return [max(-32768, min(32767, round(x * scale))) for x in combined]


def write_private(path, content):
    with path.open("x", encoding="utf-8") as f:
        os.chmod(path, 0o600)
        f.write(json.dumps(content, ensure_ascii=False, indent=2) + "\n")


def prepare(suite_path, output):
    suite = json.loads(suite_path.read_text())
    output.mkdir(mode=0o700)  # Never overwrite/delete an existing recording set.
    rows = []
    for index, sample in enumerate(suite):
        path = (suite_path.parent / sample["wav"]).resolve()
        if not sample["reference"]:
            rows.append({**sample, "wav": str(path), "group": "noise"})
            continue
        with wave.open(str(path)) as w:
            if (w.getframerate(), w.getnchannels(), w.getsampwidth()) != (16000, 1, 2):
                raise ValueError("need PCM16/16k/mono")
            pcm = array.array("h", w.readframes(w.getnframes()))
            if sys.byteorder != "little":
                pcm.byteswap()
        rows.append({**sample, "wav": str(path), "group": "clean"})
        for kind in ("fan", "click"):
            for snr in (0, 5, 10, 20):
                ident = f"{index:04d}-{kind}-{snr}"
                rng = random.Random(f"xiaozhi-20261124-{ident}")
                audio = array.array("h", mix(pcm, 16000, snr, kind, rng))
                if sys.byteorder != "little":
                    audio.byteswap()
                name = output / f"{ident}.wav"
                with name.open("xb") as f:
                    os.chmod(name, 0o600)
                    with wave.open(f, "wb") as w:
                        w.setparams((1, 2, 16000, 0, "NONE", "not compressed"))
                        w.writeframes(audio.tobytes())
                rows.append({"id": ident, "wav": name.name,
                    "reference": sample["reference"], "source_id": sample["id"],
                    "group": f"{kind}-snr{snr}"})
    write_private(output / "suite.json", rows)
    print(json.dumps({"samples": len(rows), "suite": str(output / "suite.json")}))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    prep = sub.add_parser("mix")
    prep.add_argument("--suite", type=Path, required=True)
    prep.add_argument("--output", type=Path, required=True)
    scoring = sub.add_parser("score")
    scoring.add_argument("--suite", type=Path, required=True)
    scoring.add_argument("--report", type=Path, required=True)
    args = parser.parse_args()
    if args.command == "mix":
        prepare(args.suite, args.output)
    else:
        print(json.dumps(score(json.loads(args.suite.read_text()),
                               json.loads(args.report.read_text())), indent=2))


if __name__ == "__main__":
    main()
