#!/usr/bin/env -S uv run --script
"""Benchmark the current application over its production MCP HTTP handler."""

import argparse
import json
import os
from pathlib import Path
import platform
import subprocess
import tempfile
import time


def run(command, root, **kwargs):
    print("+", " ".join(map(str, command)), flush=True)
    try:
        return subprocess.run(command, cwd=root, check=True, text=True, **kwargs)
    except subprocess.CalledProcessError as error:
        if error.stdout:
            print(error.stdout)
        if error.stderr:
            print(error.stderr)
        raise


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--count", type=int, default=6)
    parser.add_argument("--benchtime", default="1s")
    parser.add_argument("--cpu", type=int, default=4)
    parser.add_argument("--affinity", help="Linux logical CPU IDs, e.g. 8,9,10,11")
    parser.add_argument("--bench", default="^BenchmarkMCPHTTP$")
    parser.add_argument("--output", type=Path, default=Path(tempfile.gettempdir()) / "i18n-mcp-results")
    parser.add_argument("--validate-only", action="store_true")
    args = parser.parse_args()
    if args.count < 1 or args.cpu < 1:
        parser.error("count and cpu must be positive")
    affinity = {int(cpu) for cpu in args.affinity.split(",")} if args.affinity else None
    root = Path(__file__).resolve().parents[2]
    output = (root / args.output).resolve()
    output.mkdir(parents=True, exist_ok=True)
    metadata = {
        "go": run(["go", "version"], root, capture_output=True).stdout.strip(),
        "revision": run(["git", "rev-parse", "HEAD"], root, capture_output=True).stdout.strip(),
        "dirty": bool(run(["git", "status", "--porcelain"], root, capture_output=True).stdout.strip()),
        "platform": platform.platform(),
        "affinity": sorted(affinity) if affinity else (sorted(os.sched_getaffinity(0)) if hasattr(os, "sched_getaffinity") else None),
        "args": {**vars(args), "output": str(output)},
        "started_utc": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    }
    (output / "environment.json").write_text(json.dumps(metadata, indent=2) + "\n")
    result = run(["go", "test", "-tags=mcpbench", "./...", "-count=1"], root, capture_output=True)
    (output / "application-tests.txt").write_text(result.stdout + result.stderr)
    if args.validate_only:
        return
    with tempfile.TemporaryDirectory(prefix="i18n-mcp-bench-") as temporary:
        binary = Path(temporary) / "mcp-benchmark"
        run(["go", "test", "-tags=mcpbench", "-c", "./benchmarks/mcp", "-o", str(binary)], root)
        if affinity:
            os.sched_setaffinity(0, affinity)
        with (output / "application.txt").open("w") as destination:
            for repetition in range(args.count):
                result = run([
                    str(binary), "-test.run=^$", f"-test.bench={args.bench}",
                    "-test.benchmem", f"-test.benchtime={args.benchtime}", f"-test.cpu={args.cpu}",
                    "-test.count=1", "-test.timeout=10m",
                ], root, capture_output=True)
                destination.write(result.stdout)
                destination.write(result.stderr)
                destination.flush()
                print(f"completed sample {repetition + 1}/{args.count}", flush=True)
    print(f"Results: {output}")


if __name__ == "__main__":
    main()
