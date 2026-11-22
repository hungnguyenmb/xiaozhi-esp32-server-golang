"""Smoke test/benchmark ASR FunASR, không in bản chép hoặc audio cá nhân.

Suite JSON: [{"id":"sample-1", "wav":"relative.wav", "reference":"xin chào"}].
Reference rỗng biểu thị mẫu im lặng/tiếng động cần kết quả rỗng.
"""

import argparse
import asyncio
import json
import math
import statistics
import time
import unicodedata
import wave
from pathlib import Path

import aiohttp


def distance(a, b):
    row = list(range(len(b) + 1))
    for i, x in enumerate(a, 1):
        new = [i]
        for j, y in enumerate(b, 1):
            new.append(min(new[-1] + 1, row[j] + 1, row[j - 1] + (x != y)))
        row = new
    return row[-1]


def normalize(text):
    text = unicodedata.normalize("NFC", text).lower()
    return " ".join("".join(" " if unicodedata.category(c).startswith("P") else c
                            for c in text).split())


def read_wav(path):
    with wave.open(str(path)) as wav:
        if (wav.getnchannels(), wav.getsampwidth(), wav.getframerate(), wav.getcomptype()) != (1, 2, 16000, "NONE"):
            raise ValueError("Expected mono PCM16 16kHz WAV")
        pcm = wav.readframes(wav.getnframes())
    if not pcm or len(pcm) > 32000 * 60:
        raise ValueError("Expected 0 < audio length <= 60 seconds")
    return pcm


async def transcribe(session, endpoint, pcm, name, timeout):
    async with session.ws_connect(endpoint, max_msg_size=1024 * 1024) as ws:
        await ws.send_json({"is_speaking": True, "audio_fs": 16000,
                            "mode": "offline", "wav_format": "pcm", "wav_name": name})
        for offset in range(0, len(pcm), 3200):
            await ws.send_bytes(pcm[offset:offset + 3200])
        started = time.monotonic()
        await ws.send_json({"is_speaking": False})
        result = await asyncio.wait_for(ws.receive_json(), timeout)
        elapsed = (time.monotonic() - started) * 1000
        if not (result.get("is_final") is True and result.get("mode") == "offline"
                and result.get("wav_name") == name and isinstance(result.get("text"), str)):
            raise ValueError("Invalid ASR response contract")
        return result, elapsed


async def run(args):
    if args.suite:
        samples = json.loads(args.suite.read_text())
        for sample in samples:
            sample["pcm"] = read_wav(args.suite.parent / sample["wav"])
    elif args.wav:
        samples = [{"id": "speech", "pcm": read_wav(args.wav)}]
    else:
        samples = [{"id": "silence", "pcm": b"\0\0" * 16000, "reference": ""}]
    if not samples:
        raise ValueError("Empty suite")
    rows = []
    errors = {"word_errors": 0, "words": 0, "char_errors": 0, "chars": 0}
    async with aiohttp.ClientSession() as session:
        # One warm-up per process, reported separately; no transcripts printed.
        _, warmup = await transcribe(session, args.endpoint, samples[0]["pcm"], "warmup", args.timeout)
        gate = asyncio.Semaphore(args.concurrency)

        async def measure(sample, index):
            async with gate:
                result, elapsed = await transcribe(session, args.endpoint, sample["pcm"],
                                                   f"sample-{index}", args.timeout)
                row = {"id": sample["id"], "asr_total_ms": round(elapsed, 2),
                       "audio_s": len(sample["pcm"]) / 32000, **result.get("metrics", {})}
                row["asr_total_rtf"] = round(elapsed / 1000 / row["audio_s"], 4)
                if "inference_ms" in row:
                    row["inference_rtf"] = round(row["inference_ms"] / 1000 / row["audio_s"], 4)
                if args.expect_speech and not result["text"].strip():
                    raise ValueError("Speech smoke test returned empty text")
                if "reference" in sample:
                    reference, actual = normalize(sample["reference"]), normalize(result["text"])
                    if not reference:
                        row["unexpected_text"] = bool(actual)
                    else:
                        row["word_errors"] = distance(reference.split(), actual.split())
                        row["words"] = len(reference.split())
                        row["char_errors"] = distance(reference, actual)
                        row["chars"] = len(reference)
                        for key in errors:
                            errors[key] += row[key]
                rows.append(row)

        await asyncio.gather(*(measure(sample, i) for i, sample in enumerate(samples * args.repeat)))
    times = sorted(row["asr_total_ms"] for row in rows)
    report = {"samples": len(rows), "concurrency": args.concurrency, "warmup_ms": round(warmup, 2),
              "p50_ms": round(statistics.median(times), 2),
              "p95_ms": times[math.ceil(len(times) * .95) - 1], "rows": rows,
              "wer": errors["word_errors"] / errors["words"] if errors["words"] else None,
              "cer": errors["char_errors"] / errors["chars"] if errors["chars"] else None}
    print(json.dumps(report, ensure_ascii=False, indent=2))
    if any(row.get("unexpected_text") for row in rows):
        raise ValueError("Unexpected text on silence/noise samples")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--endpoint", default="ws://127.0.0.1:10096/")
    source = parser.add_mutually_exclusive_group()
    source.add_argument("--wav", type=Path)
    source.add_argument("--suite", type=Path)
    parser.add_argument("--expect-speech", action="store_true")
    parser.add_argument("--concurrency", type=int, choices=range(1, 5), default=1)
    parser.add_argument("--repeat", type=int, choices=range(1, 101), default=1)
    parser.add_argument("--timeout", type=float, default=30)
    asyncio.run(run(parser.parse_args()))
