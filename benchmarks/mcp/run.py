#!/usr/bin/env -S uv run --script
"""Compare application JSON backends without editing production source files."""

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
    parser.add_argument("--benchtime", default="500ms")
    parser.add_argument("--cpu", type=int, default=4)
    parser.add_argument("--affinity", help="Linux logical CPU IDs, e.g. 8,9,10,11")
    parser.add_argument("--bench", default="^BenchmarkMCPHTTP$")
    parser.add_argument("--output", type=Path, default=Path("benchmarks/mcp/results"))
    parser.add_argument("--validate-only", action="store_true")
    args = parser.parse_args()
    if args.count < 1 or args.cpu < 1:
        parser.error("count and cpu must be positive")
    affinity = {int(cpu) for cpu in args.affinity.split(",")} if args.affinity else None

    root = Path(__file__).resolve().parents[2]
    architecture = run(["go", "env", "GOARCH"], root, capture_output=True).stdout.strip()
    if architecture not in {"amd64", "arm64"}:
        parser.error("backend overlays require amd64 or arm64; otherwise Sonic is already a fallback")
    output = (root / args.output).resolve()
    output.mkdir(parents=True, exist_ok=True)
    metadata = {
        "go": run(["go", "version"], root, capture_output=True).stdout.strip(),
        "sonic": run(["go", "list", "-m", "github.com/bytedance/sonic"], root, capture_output=True).stdout.strip(),
        "platform": platform.platform(),
        "affinity": sorted(affinity) if affinity else (sorted(os.sched_getaffinity(0)) if hasattr(os, "sched_getaffinity") else None),
        "args": {**vars(args), "output": str(output)},
        "started_utc": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    }
    (output / "environment.json").write_text(json.dumps(metadata, indent=2) + "\n")

    with tempfile.TemporaryDirectory(prefix="i18n-mcp-bench-") as temporary:
        scratch = Path(temporary)
        # The fallback implementation is already the exact v1 wrapper used by
        # production on other architectures. Change its constraint, not its API.
        v1 = scratch / "v1.go"
        v1.write_text((root / "internal/jsonutil/json_std.go").read_text().replace(
            "//go:build !amd64 && !arm64", "//go:build amd64 || arm64", 1))
        replacements = {
            "sonic": None,
            "stdlib_v1": v1,
            "stdlib_v2_compat": root / "benchmarks/mcp/backends/v2.go.txt",
        }
        binaries = {}
        for name, replacement in replacements.items():
            flags = []
            if replacement:
                overlay = scratch / f"{name}.json"
                overlay.write_text(json.dumps({"Replace": {
                    str(root / "internal/jsonutil/json_sonic.go"): str(replacement),
                }}))
                flags = [f"-overlay={overlay}"]
            # Every backend must pass the application suite and end-to-end
            # workload assertions before any timed samples are collected.
            result = run(["go", "test", *flags, "-tags=mcpbench", "./...", "-count=1"], root, capture_output=True)
            (output / f"{name}-tests.txt").write_text(result.stdout + result.stderr)
            binary = scratch / name
            run(["go", "test", *flags, "-tags=mcpbench", "-c", "./benchmarks/mcp", "-o", str(binary)], root)
            binaries[name] = binary

        if args.validate_only:
            return
        if affinity:
            os.sched_setaffinity(0, affinity)
        for name in binaries:
            (output / f"{name}.txt").write_text("")
        names = list(binaries)
        for repetition in range(args.count):
            # Rotate order to reduce bias from CPU frequency/thermal drift.
            order = names[repetition % len(names):] + names[:repetition % len(names)]
            for name in order:
                result = run([
                    str(binaries[name]), "-test.run=^$", f"-test.bench={args.bench}",
                    "-test.benchmem", f"-test.benchtime={args.benchtime}", f"-test.cpu={args.cpu}",
                    "-test.count=1", "-test.timeout=10m",
                ], root, capture_output=True)
                with (output / f"{name}.txt").open("a") as destination:
                    destination.write(result.stdout)
                    destination.write(result.stderr)
                print(f"completed {name}, sample {repetition + 1}/{args.count}", flush=True)
    print(f"Results: {output}")


if __name__ == "__main__":
    main()
