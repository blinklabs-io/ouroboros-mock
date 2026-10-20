#!/usr/bin/env python3
"""Generate deterministic CIP-0177 leader selection vectors."""

import hashlib
import json
from pathlib import Path


SPEC_REVISION = "86b89208d3b2aabb5dcc5b778dfbe09096b4e114"
OUTPUT = Path(__file__).parents[1] / "fixtures" / "tachys" / "leader_selection.json"
TWO_TO_256 = 1 << 256


def pool_id(value: int) -> str:
    return f"{value:02x}" * 28


POOLS = [
    {"pool_id": pool_id(1), "stake": 1},
    {"pool_id": pool_id(2), "stake": 2},
    {"pool_id": pool_id(3), "stake": 3},
]


def leader(sample: int) -> str:
    cumulative = 0
    for pool in sorted(POOLS, key=lambda item: bytes.fromhex(item["pool_id"])):
        cumulative += pool["stake"]
        if sample * 6 < cumulative * TWO_TO_256:
            return pool["pool_id"]
    raise AssertionError("sample was outside [0, 2^256)")


def hash_case(nonce_byte: int, slot: int) -> dict:
    nonce = bytes([nonce_byte]) * 32
    slot_bytes = slot.to_bytes(8, "big")
    digest = hashlib.blake2b(nonce + slot_bytes, digest_size=32).digest()
    sample = int.from_bytes(digest, "big")
    return {
        "epoch_nonce": nonce.hex(),
        "slot": slot,
        "hash_input": (nonce + slot_bytes).hex(),
        "sample": digest.hex(),
        "leader": leader(sample),
    }


def boundary_case(name: str, sample: int) -> dict:
    return {
        "name": name,
        "sample": sample.to_bytes(32, "big").hex(),
        "leader": leader(sample),
    }


def generate() -> dict:
    first_boundary = (TWO_TO_256 + 5) // 6
    second_boundary = TWO_TO_256 // 2
    return {
        "format_version": 1,
        "specification": {
            "name": "CIP-0177",
            "revision": SPEC_REVISION,
            "source": "https://github.com/cardano-foundation/CIPs/blob/"
            + SPEC_REVISION
            + "/CIP-0177/README.md",
        },
        "provenance": {
            "generator": "scripts/generate_tachys_vectors.py",
            "method": "independent Python standard-library calculation",
        },
        "encoding": {
            "nonce": "32 raw bytes",
            "slot": "absolute uint64 big-endian",
            "hash_input": "nonce || slot",
            "hash": "BLAKE2b-256",
            "sample": "unsigned big-endian integer over [0, 2^256)",
            "pool_order": "lexicographic raw 28-byte pool ID",
            "intervals": "inclusive lower bound, exclusive upper bound",
        },
        "distribution": {"pools": POOLS, "total_stake": 6},
        "ordering_cases": [
            {
                "name": "unsorted input uses canonical pool ID order",
                "pools": [POOLS[2], POOLS[0], POOLS[1]],
                "expected_pool_ids": [pool["pool_id"] for pool in POOLS],
            }
        ],
        "selection_cases": [
            hash_case(0, 0),
            hash_case(0, 1),
            hash_case(0xA5, 4_492_800),
            hash_case(0xFF, (1 << 64) - 1),
        ],
        "boundary_cases": [
            boundary_case("zero", 0),
            boundary_case("first interval last sample", first_boundary - 1),
            boundary_case("second interval first sample", first_boundary),
            boundary_case("second interval last sample", second_boundary - 1),
            boundary_case("third interval first sample", second_boundary),
            boundary_case("maximum sample", TWO_TO_256 - 1),
        ],
        "invalid_cases": [
            {"name": "empty distribution", "pools": [], "total_stake": 0},
            {
                "name": "duplicate pool ID",
                "pools": [
                    {"pool_id": pool_id(1), "stake": 1},
                    {"pool_id": pool_id(1), "stake": 1},
                ],
                "total_stake": 2,
            },
            {
                "name": "zero pool stake",
                "pools": [
                    {"pool_id": pool_id(1), "stake": 1},
                    {"pool_id": pool_id(2), "stake": 0},
                ],
                "total_stake": 1,
            },
            {
                "name": "inconsistent total",
                "pools": [{"pool_id": pool_id(1), "stake": 1}],
                "total_stake": 2,
            },
            {
                "name": "stake sum overflow",
                "pools": [
                    {"pool_id": pool_id(1), "stake": (1 << 64) - 1},
                    {"pool_id": pool_id(2), "stake": 1},
                ],
                "total_stake": (1 << 64) - 1,
            },
            {"name": "short nonce", "epoch_nonce": "00" * 31, "slot": 0},
            {"name": "long nonce", "epoch_nonce": "00" * 33, "slot": 0},
            {"name": "short sample", "sample": "00" * 31},
        ],
    }


def main() -> None:
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT.write_text(json.dumps(generate(), indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
