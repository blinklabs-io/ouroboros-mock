#!/usr/bin/env python3
"""Validate untouched C goldens and regenerate fixtures using the real node oracle.

Requires cc/g++, GMP, and an already-built jormanager-leaderlog-oracle.
No Kotlin numerical implementation is used to create expected values.
"""
import argparse
import gzip
import hashlib
import itertools
import json
import re
import shutil
import subprocess
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[4]
RESOURCES = ROOT / "src/test/resources/leaderlog"
SCALE = 10**34
ROWS = 100000
HASHES = {
    "golden_tests.txt": "54da107c827bf9d21484f88bbbe8dc140071d44aa2a6d665113a1dfd26dcad9a",
    "golden_tests_result.txt": "1918c67d0043dc4b03a430eac7c989c5e753c405b7d7e2d4d7cac177da7b1a69",
}
DECIMAL = re.compile(r"-?\d+\.\d{34}\Z")
INTEGER = re.compile(r"-?\d+\Z")
CLASSES = {"ABOVE", "BELOW", "MAX_REACHED"}


def require(condition, message):
    if not condition:
        raise ValueError(message)


def write_json(path, value):
    path.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def compressed(path, payload):
    path.write_bytes(gzip.compress(payload, mtime=0))


def lines(data, count, fields, label):
    require(data.endswith(b"\n"), f"{label}: truncated final row")
    result = [row.split() for row in data.decode("ascii").splitlines()]
    require(len(result) == count, f"{label}: expected {count} rows, got {len(result)}")
    require(all(len(row) == fields for row in result), f"{label}: malformed field count")
    return result


def validate_corpus(source):
    reference = source / "libs/non-integral/reference"
    payloads = {}
    for name, digest in HASHES.items():
        data = (reference / name).read_bytes()
        require(hashlib.sha256(data).hexdigest() == digest, f"{name}: wrong upstream SHA-256")
        require(gzip.decompress((RESOURCES / (name + ".gz")).read_bytes()) == data,
                f"{name}: committed compressed corpus differs")
        payloads[name] = data
    inputs = lines(payloads["golden_tests.txt"], ROWS, 3, "C inputs")
    outputs = lines(payloads["golden_tests_result.txt"], ROWS, 6, "C outputs")
    require(all(all(INTEGER.fullmatch(v) for v in row) for row in inputs), "malformed C input")
    require(all(all(DECIMAL.fullmatch(v) for v in row[:4]) and row[4] in {"LT", "GT"}
                and INTEGER.fullmatch(row[5]) and 0 < int(row[5]) <= 1000 for row in outputs),
            "malformed C output")
    with tempfile.TemporaryDirectory(prefix="leaderlog-c-") as directory:
        work = Path(directory)
        # Keep upstream source untouched, including headers. Build products stay temporary.
        for name in ("non_integral.c", "non_integral.h", "non_integral.hpp", "non_integral_test.cpp"):
            shutil.copyfile(reference / name, work / name)
        subprocess.run(["cc", "-O2", "-fPIC", "-c", "non_integral.c", "-o", "non_integral.o"], cwd=work, check=True)
        subprocess.run(["g++", "-O2", "non_integral_test.cpp", "non_integral.o", "-lgmp", "-lgmpxx", "-o", "runner"], cwd=work, check=True)
        with (reference / "golden_tests.txt").open("rb") as stdin, (work / "result").open("wb") as stdout:
            subprocess.run([str(work / "runner")], stdin=stdin, stdout=stdout, check=True)
        actual = (work / "result").read_bytes()
        lines(actual, ROWS, 6, "C regenerated output")
        require(actual == payloads["golden_tests_result.txt"], "C regenerated stdout differs byte-for-byte")
    return payloads, inputs, outputs


def oracle_rows(oracle, mode, rows, expected_fields):
    payload = "".join(" ".join(map(str, row)) + "\n" for row in rows).encode("ascii")
    result = subprocess.run([oracle, mode], input=payload, stdout=subprocess.PIPE, check=True).stdout
    parsed = lines(result, len(rows), expected_fields, f"node {mode}")
    return result, parsed


def value_rows(oracle, rows):
    _, result = oracle_rows(oracle, "values", rows, 5)
    for row, output in zip(rows, result):
        require(output[0] in {"true", "false"}, "invalid node boolean")
        require(all(INTEGER.fullmatch(output[i]) for i in (1, 3)), "invalid node sigma/q")
        require(all(INTEGER.fullmatch(output[i]) for i in (2, 4)) if row[4] != row[5]
                else output[2] == output[4] == "none", "invalid node c/x")
    return result


def generate(oracle, directory, inputs, c_outputs, specs):
    arithmetic, node = oracle_rows(oracle, "arithmetic", inputs, 5)
    require(all(all(DECIMAL.fullmatch(v) for v in row[:3]) and row[3] in CLASSES
                and row[4].isdigit() and 0 < int(row[4]) <= 1000 for row in node), "malformed node arithmetic")
    compressed(directory / "node-arithmetic.tsv.gz", arithmetic)
    divergences = []
    for index, (inp, c_row, node_row) in enumerate(zip(inputs, c_outputs, node)):
        for field, c_result, node_result in zip(("exp", "negativeLn", "threshold"), c_row[:3], node_row[:3]):
            if c_result != node_result:
                divergences.append(dict(row=index, field=field, input=" ".join(inp), cResult=c_result, nodeResult=node_result))
    write_json(directory / "oracle-divergences.json", divergences)

    rows = []
    for width, (pool, active), (fn, fd) in itertools.product(specs["widths"], specs["stakes"], specs["coefficients"]):
        maximum = 1 << (8 * width)
        rows.extend([width, cert, pool, active, fn, fd] for cert in (0, 1, maximum - 1, maximum))
    # Batch each bisection round across all boundaries to amortize oracle startup.
    boundaries = []
    for width, (pool, active), (fn, fd) in itertools.product(specs["widths"], specs["boundaryStakes"], specs["boundaryCoefficients"]):
        boundaries.append([width, pool, active, fn, fd, 0, 1 << (8 * width)])
    endpoint_rows = [[w, cert, p, a, fn, fd] for w, p, a, fn, fd, lo, hi in boundaries for cert in (lo, hi)]
    endpoints = value_rows(oracle, endpoint_rows)
    require(all(output[0] == ("true" if index % 2 == 0 else "false") for index, output in enumerate(endpoints)),
            "invalid boundary endpoints")
    while True:
        pending = [boundary for boundary in boundaries if boundary[6] - boundary[5] > 1]
        if not pending:
            break
        queries = [[w, (lo + hi) // 2, p, a, fn, fd] for w, p, a, fn, fd, lo, hi in pending]
        for boundary, query, output in zip(pending, queries, value_rows(oracle, queries)):
            boundary[5 if output[0] == "true" else 6] = query[1]
    for width, pool, active, fn, fd, low, high in boundaries:
        rows.extend([width, cert, pool, active, fn, fd] for cert in (low - 1, low, low + 1) if cert >= 0)
    # Preserve first occurrence/order, including diagnostic large integer inputs.
    rows = [list(row) for row in dict.fromkeys(map(tuple, rows))]
    outputs = value_rows(oracle, rows)
    (directory / "node-vectors.tsv").write_text("".join("\t".join(map(str, row + output)) + "\n" for row, output in zip(rows, outputs)), encoding="ascii")

    comparisons = specs["comparisons"]
    _, outputs = oracle_rows(oracle, "comparison", comparisons, 2)
    require(all(row[0] in CLASSES and row[1].isdigit() and 0 < int(row[1]) <= 1000 for row in outputs), "invalid node comparison")
    (directory / "node-comparisons.tsv").write_text("".join("\t".join(map(str, row + output)) + "\n" for row, output in zip(comparisons, outputs)), encoding="ascii")

    cases = []
    for case in specs["schedules"]:
        keys = ("mode", "firstSlot", "slotCount", "poolStake", "activeStake", "fNumerator", "fDenominator", "dNumerator", "dDenominator", "nonceHex")
        result = subprocess.run([oracle, "schedule"] + [str(case[key]) for key in keys], stdout=subprocess.PIPE, check=True).stdout
        require(result.endswith(b"\n"), "truncated schedule output")
        output = result.decode("ascii").splitlines()
        require(len(output) > 0 and re.fullmatch(r"[0-9a-f]{128}", output[0]), "invalid synthetic VRF key")
        require(all(INTEGER.fullmatch(slot) for slot in output[1:]), "invalid schedule slot")
        slots = list(map(int, output[1:]))
        require(slots == sorted(set(slots)), "schedule not sorted/unique")
        require(all(case["firstSlot"] <= slot < case["firstSlot"] + case["slotCount"] for slot in slots), "schedule slot outside range")
        cases.append(dict(case, vrfSkeyHex=output[0], assignedSlots=slots))
    freeze = Path(__file__).with_name("cabal.project.freeze")
    require(freeze.is_file(), "record resolved transitive pins first: cabal freeze --project-file=src/test/reference/leaderlog/cabal.project")
    metadata = dict(specs["metadata"], freezeSha256=hashlib.sha256(freeze.read_bytes()).hexdigest())
    write_json(directory / "node-schedules.json", dict(metadata=metadata, cases=cases))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--refresh", action="store_true")
    mode.add_argument("--check", action="store_true")
    parser.add_argument("--ledger-source", type=Path, required=True)
    parser.add_argument("--oracle", required=True)
    args = parser.parse_args()
    specs = json.loads((RESOURCES / "fixture-specifications.json").read_text())
    _, inputs, outputs = validate_corpus(args.ledger_source)
    with tempfile.TemporaryDirectory(prefix="leaderlog-node-") as temporary:
        target = Path(temporary)
        generate(str(Path(args.oracle).resolve()), target, inputs, outputs, specs)
        for generated in sorted(target.iterdir()):
            committed = RESOURCES / generated.name
            if args.refresh:
                shutil.copyfile(generated, committed)
            else:
                require(committed.is_file(), f"missing committed fixture: {committed}")
                require(generated.read_bytes() == committed.read_bytes(), f"node fixture differs: {committed}")
    print(f"Validated all {ROWS} C rows and {'refreshed' if args.refresh else 'checked'} every node fixture")


if __name__ == "__main__":
    main()
