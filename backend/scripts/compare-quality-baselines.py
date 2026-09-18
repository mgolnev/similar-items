#!/usr/bin/env python3
"""Compare two baselines produced by capture-quality-baseline.py."""

from __future__ import annotations

import argparse
import gzip
import json
import pathlib
import statistics
import sys
from typing import Any


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--before", required=True, help="Baseline directory")
    parser.add_argument("--after", required=True, help="New baseline directory")
    parser.add_argument(
        "--output",
        help="Optional JSON comparison report",
    )
    parser.add_argument(
        "--mode",
        choices=("exact", "quality"),
        default="exact",
        help="exact requires byte-for-byte ordered output equality",
    )
    parser.add_argument(
        "--min-overlap-at-k",
        type=float,
        default=0.95,
        help="Quality mode gate for average top-K overlap",
    )
    return parser.parse_args()


def load_summary(directory: pathlib.Path) -> dict[str, Any]:
    return json.loads((directory / "summary.json").read_text(encoding="utf-8"))


def load_responses(directory: pathlib.Path, limit: int) -> dict[str, dict[str, Any]]:
    path = directory / f"responses-limit-{limit}.jsonl.gz"
    out: dict[str, dict[str, Any]] = {}
    with gzip.open(path, "rt", encoding="utf-8") as stream:
        for line in stream:
            value = json.loads(line)
            out[value["article"]] = value
    return out


def ratio(numerator: int, denominator: int) -> float | None:
    if denominator == 0:
        return None
    return numerator / denominator


def overlap_at_k(before: list[str], after: list[str], k: int) -> float:
    denominator = min(k, max(len(before), len(after)))
    if denominator == 0:
        return 1.0
    return len(set(before[:k]) & set(after[:k])) / denominator


def compare_limit(
    before_dir: pathlib.Path,
    after_dir: pathlib.Path,
    limit: int,
) -> dict[str, Any]:
    before = load_responses(before_dir, limit)
    after = load_responses(after_dir, limit)
    articles = sorted(set(before) | set(after))
    exact = 0
    same_items_unordered = 0
    position_matches = 0
    compared_positions = 0
    overlaps: list[float] = []
    changed_examples: list[dict[str, Any]] = []
    missing_before = 0
    missing_after = 0
    error_changes = 0

    for article in articles:
        old = before.get(article)
        new = after.get(article)
        if old is None:
            missing_before += 1
            continue
        if new is None:
            missing_after += 1
            continue
        old_items = old.get("items", [])
        new_items = new.get("items", [])
        if old.get("error", "") != new.get("error", ""):
            error_changes += 1
        if old_items == new_items:
            exact += 1
        elif len(changed_examples) < 25:
            changed_examples.append(
                {
                    "article": article,
                    "before": old_items,
                    "after": new_items,
                    "overlap_at_k": overlap_at_k(old_items, new_items, limit),
                }
            )
        if set(old_items) == set(new_items):
            same_items_unordered += 1
        overlaps.append(overlap_at_k(old_items, new_items, limit))
        for old_item, new_item in zip(old_items[:limit], new_items[:limit]):
            compared_positions += 1
            if old_item == new_item:
                position_matches += 1

    comparable = len(articles) - missing_before - missing_after
    return {
        "limit": limit,
        "article_union": len(articles),
        "comparable_articles": comparable,
        "missing_before": missing_before,
        "missing_after": missing_after,
        "error_changes": error_changes,
        "exact_ordered_count": exact,
        "exact_ordered_rate": ratio(exact, comparable),
        "same_items_unordered_rate": ratio(same_items_unordered, comparable),
        "position_match_rate": ratio(position_matches, compared_positions),
        "mean_overlap_at_k": statistics.fmean(overlaps) if overlaps else None,
        "min_overlap_at_k": min(overlaps) if overlaps else None,
        "changed_examples": changed_examples,
    }


def quality_deltas(
    before: dict[str, Any],
    after: dict[str, Any],
    limit: int,
) -> dict[str, Any]:
    old = before["quality"][str(limit)]
    new = after["quality"][str(limit)]
    keys = (
        "error_rate",
        "empty_rate",
        "fill_rate",
        "valid_item_rate",
        "same_section_rate",
        "same_leaf_rate",
        "same_parent_rate",
        "top1_same_section_rate",
        "top1_same_leaf_rate",
    )
    counts = (
        "unknown_item_count",
        "self_item_count",
        "duplicate_item_count",
        "same_model_item_count",
    )
    result: dict[str, Any] = {}
    for key in keys:
        old_value = old.get(key)
        new_value = new.get(key)
        result[key] = {
            "before": old_value,
            "after": new_value,
            "delta": (
                new_value - old_value
                if old_value is not None and new_value is not None
                else None
            ),
        }
    for key in counts:
        result[key] = {
            "before": old.get(key),
            "after": new.get(key),
            "delta": new.get(key, 0) - old.get(key, 0),
        }
    return result


def exact_gate(comparison: dict[str, Any]) -> list[str]:
    failures: list[str] = []
    if comparison["missing_before"] or comparison["missing_after"]:
        failures.append("article sets differ")
    if comparison["error_changes"]:
        failures.append("API errors differ")
    if comparison["exact_ordered_rate"] != 1:
        failures.append("ordered recommendation lists differ")
    return failures


def quality_gate(
    comparison: dict[str, Any],
    deltas: dict[str, Any],
    min_overlap: float,
) -> list[str]:
    failures: list[str] = []
    if comparison["missing_before"] or comparison["missing_after"]:
        failures.append("article sets differ")
    if comparison["mean_overlap_at_k"] < min_overlap:
        failures.append(
            f"mean overlap@K is below {min_overlap:.3f}"
        )
    for key in (
        "error_rate",
        "empty_rate",
        "unknown_item_count",
        "self_item_count",
        "duplicate_item_count",
        "same_model_item_count",
    ):
        if deltas[key]["delta"] > 0:
            failures.append(f"{key} increased")
    for key in (
        "fill_rate",
        "valid_item_rate",
        "same_section_rate",
    ):
        if deltas[key]["delta"] < 0:
            failures.append(f"{key} decreased")
    return failures


def main() -> int:
    args = parse_args()
    before_dir = pathlib.Path(args.before).resolve()
    after_dir = pathlib.Path(args.after).resolve()
    before_summary = load_summary(before_dir)
    after_summary = load_summary(after_dir)
    before_limits = set(before_summary["metadata"]["limits"])
    after_limits = set(after_summary["metadata"]["limits"])
    limits = sorted(before_limits & after_limits)
    if not limits:
        raise RuntimeError("baselines have no common captured limits")

    reports: dict[str, Any] = {}
    failures: list[str] = []
    for limit in limits:
        comparison = compare_limit(before_dir, after_dir, limit)
        deltas = quality_deltas(before_summary, after_summary, limit)
        if args.mode == "exact":
            gate_failures = exact_gate(comparison)
        else:
            gate_failures = quality_gate(
                comparison, deltas, args.min_overlap_at_k
            )
        failures.extend(
            f"limit={limit}: {failure}" for failure in gate_failures
        )
        reports[str(limit)] = {
            "output_comparison": comparison,
            "quality_deltas": deltas,
            "gate_failures": gate_failures,
        }

    report = {
        "mode": args.mode,
        "before": str(before_dir),
        "after": str(after_dir),
        "before_commit": before_summary["metadata"]["git"]["commit"],
        "after_commit": after_summary["metadata"]["git"]["commit"],
        "limits": reports,
        "passed": not failures,
        "failures": failures,
    }
    rendered = json.dumps(report, ensure_ascii=False, indent=2, sort_keys=True)
    if args.output:
        pathlib.Path(args.output).write_text(rendered + "\n", encoding="utf-8")
    print(rendered)
    if failures:
        print("quality gate failed", file=sys.stderr)
        return 1
    print("quality gate passed", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
