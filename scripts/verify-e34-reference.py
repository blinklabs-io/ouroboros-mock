#!/usr/bin/env python3
# Copyright 2026 Blink Labs Software
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
"""Run the preserved E34 verifier only with its pinned reference sources."""

import argparse
import hashlib
import subprocess
import sys
from pathlib import Path

FREEZE_SHA256 = "867325116626ee6841449f09866200528d83a9321faf2df721cd45ef04cd4a69"


def main():
    parser = argparse.ArgumentParser(description=__doc__, add_help=False)
    parser.add_argument("--jormanager-root", type=Path, required=True)
    args, verifier_args = parser.parse_known_args()
    reference = args.jormanager_root / "src/test/reference/leaderlog"
    corpus = Path(__file__).resolve().parents[1] / "fixtures/upstream/e34"
    pins = dict(
        (name, digest)
        for digest, name in (
            line.split("  ", 1)
            for line in (corpus / "SHA256SUMS").read_text().splitlines()
        )
    )
    # The freeze has an independent pin; refreshing outputs cannot bless new dependencies.
    pins["reference/cabal.project.freeze"] = FREEZE_SHA256
    for name, digest in pins.items():
        if not name.startswith("reference/"):
            continue
        source = reference / Path(name).name
        if not source.is_file() or hashlib.sha256(source.read_bytes()).hexdigest() != digest:
            parser.error(f"{source.name}: reference source differs from pinned SHA-256")
    subprocess.run(
        [sys.executable, str(reference / "verify.py"), *verifier_args], check=True
    )


if __name__ == "__main__":
    main()
