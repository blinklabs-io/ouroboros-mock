# Copyright 2026 Blink Labs Software
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
"""Exercise fixture maintenance without network or an installed Haskell oracle."""

import os
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def tree_bytes(root):
    return {str(p.relative_to(root)): p.read_bytes() for p in root.rglob("*") if p.is_file()}


class E34ToolsTests(unittest.TestCase):
    def test_upstream_refresh_preserves_corpus(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "scripts").mkdir()
            script = root / "scripts/update-upstream-fixtures.sh"
            shutil.copyfile(ROOT / "scripts/update-upstream-fixtures.sh", script)
            shutil.copytree(ROOT / "fixtures/upstream", root / "fixtures/upstream")
            before = tree_bytes(root / "fixtures/upstream/e34")
            env = dict(os.environ)
            for name, variable in (
                ("ouroboros-consensus", "OUROBOROS_CONSENSUS_SRC"),
                ("cardano-ledger", "CARDANO_LEDGER_SRC"),
                ("cardano-api", "CARDANO_API_SRC"),
                ("cardano-node", "CARDANO_NODE_SRC"),
            ):
                source = root / "sources" / name
                shutil.copytree(ROOT / "fixtures/upstream" / name, source)
                env[variable] = str(source)
            subprocess.run(["bash", str(script)], env=env, check=True, capture_output=True)
            self.assertEqual(before, tree_bytes(root / "fixtures/upstream/e34"))
            manifest = (root / "fixtures/upstream/manifest.txt").read_text().splitlines()
            for name in before:
                self.assertIn("./e34/" + name, manifest)

    def test_verifier_rejects_changed_freeze_before_refresh(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            reference = root / "src/test/reference/leaderlog"
            shutil.copytree(ROOT / "fixtures/upstream/e34/reference", reference)
            command = [sys.executable, str(ROOT / "scripts/verify-e34-reference.py"),
                       "--jormanager-root", str(root)]
            # A valid reference reaches the original verifier, whose help needs no oracle.
            control = subprocess.run(command + ["--help"], capture_output=True, text=True)
            self.assertEqual(control.returncode, 0, control.stderr)
            freeze = reference / "cabal.project.freeze"
            freeze.write_bytes(freeze.read_bytes() + b"\n-- changed dependency freeze\n")
            result = subprocess.run(command + ["--refresh", "--ledger-source", str(root),
                                               "--oracle", "not-an-oracle"],
                                    capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("cabal.project.freeze: reference source differs", result.stderr)
            self.assertNotIn("Traceback", result.stderr)


if __name__ == "__main__":
    unittest.main()
