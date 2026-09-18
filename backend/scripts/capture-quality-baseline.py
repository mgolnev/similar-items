#!/usr/bin/env python3
"""Capture an exact and aggregate quality baseline from the public Similar API.

The script intentionally treats the public API as a black box. It reads the
currently eligible catalog from PostgreSQL, requests recommendations for every
card, and stores:

* exact ordered API responses for every requested limit;
* aggregate structural quality proxies;
* a deterministic manual-review sample enriched with catalog metadata.

Only Python's standard library and the `psql` executable are required.
"""

from __future__ import annotations

import argparse
import csv
import gzip
import hashlib
import io
import json
import math
import os
import pathlib
import statistics
import subprocess
import sys
import time
import urllib.error
import urllib.request
from collections import Counter, defaultdict
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Any, Iterable


CATALOG_SQL = r"""
WITH pool AS (
    SELECT
        TRIM(p.sku) AS sku,
        TRIM(COALESCE(p.sales_sku, '')) AS sales_sku,
        TRIM(COALESCE(p.group_id, '')) AS group_id,
        COALESCE(
            NULLIF(TRIM(p.group_id), ''),
            NULLIF(TRIM(p.sales_sku), ''),
            TRIM(p.sku)
        ) AS card_key,
        COALESCE(p.name, '') AS name,
        COALESCE(p.root_section, '') AS root_section,
        COALESCE(p.leaf_category, '') AS leaf_category,
        COALESCE(p.category_path, '[]') AS category_path,
        COALESCE(p.gender, '') AS gender,
        COALESCE(p.price, 0) AS price,
        COALESCE(p.image_url, '') AS image_url,
        ROW_NUMBER() OVER (
            PARTITION BY COALESCE(
                NULLIF(TRIM(p.group_id), ''),
                NULLIF(TRIM(p.sales_sku), ''),
                TRIM(p.sku)
            )
            ORDER BY p.sku ASC
        ) AS rn
    FROM products p
    WHERE p.publication_status = 'published'
      AND p.available = TRUE
      AND p.image_url IS NOT NULL
      AND TRIM(p.image_url) != ''
)
SELECT
    sku,
    sales_sku,
    group_id,
    card_key,
    name,
    root_section,
    leaf_category,
    category_path,
    gender,
    price,
    image_url
FROM pool
WHERE rn = 1
ORDER BY card_key ASC;
"""


@dataclass(frozen=True)
class Product:
    sku: str
    sales_sku: str
    group_id: str
    card_key: str
    name: str
    root_section: str
    leaf_category: str
    category_path: tuple[str, ...]
    gender: str
    price: float
    image_url: str

    @property
    def parent_category(self) -> str:
        if len(self.category_path) < 2:
            return ""
        return self.category_path[-2]

    @property
    def model_key(self) -> str:
        return model_key(self.card_key)

    def compact(self) -> dict[str, Any]:
        return {
            "article": self.card_key,
            "name": self.name,
            "root_section": self.root_section,
            "leaf_category": self.leaf_category,
            "parent_category": self.parent_category,
            "gender": self.gender,
            "price": self.price,
            "image_url": self.image_url,
        }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--api-base",
        default="http://127.0.0.1:8081",
        help="RecommendationEngine base URL (default: %(default)s)",
    )
    parser.add_argument(
        "--database-url",
        default=os.environ.get(
            "DATABASE_URL",
            "postgresql://similar:similar@127.0.0.1:5434/similar?sslmode=disable",
        ),
        help="PostgreSQL DSN; defaults to DATABASE_URL or the local dev DSN",
    )
    parser.add_argument(
        "--output-dir",
        required=True,
        help="New directory for baseline artifacts",
    )
    parser.add_argument(
        "--limits",
        default="12,40",
        help="Comma-separated API limits to capture (default: %(default)s)",
    )
    parser.add_argument(
        "--batch-size",
        type=int,
        default=50,
        help="Articles per batch API call, max 50 (default: %(default)s)",
    )
    parser.add_argument(
        "--workers",
        type=int,
        default=4,
        help="Concurrent batch requests (default: %(default)s)",
    )
    parser.add_argument(
        "--timeout",
        type=float,
        default=60,
        help="HTTP timeout per batch in seconds (default: %(default)s)",
    )
    parser.add_argument(
        "--manual-sample-per-section",
        type=int,
        default=25,
        help="Deterministic base products per section for manual review",
    )
    return parser.parse_args()


def parse_limits(raw: str) -> list[int]:
    values = sorted({int(part.strip()) for part in raw.split(",") if part.strip()})
    if not values or any(value <= 0 or value > 40 for value in values):
        raise ValueError("limits must contain values from 1 to 40")
    return values


def batched(values: list[str], size: int) -> Iterable[list[str]]:
    for start in range(0, len(values), size):
        yield values[start : start + size]


def model_key(article: str) -> str:
    article = article.strip()
    index = article.rfind("-")
    if index > 0:
        return article[:index]
    return article


def normalized(value: str) -> str:
    return value.strip().lower().replace("ё", "е")


def parse_category_path(raw: str) -> tuple[str, ...]:
    try:
        value = json.loads(raw or "[]")
    except json.JSONDecodeError:
        return ()
    if not isinstance(value, list):
        return ()
    return tuple(str(part).strip() for part in value if str(part).strip())


def run_psql(database_url: str) -> str:
    command = [
        "psql",
        "--no-psqlrc",
        "--set",
        "ON_ERROR_STOP=1",
        "--csv",
        database_url,
        "-c",
        CATALOG_SQL,
    ]
    completed = subprocess.run(
        command,
        check=True,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    return completed.stdout


def load_catalog(database_url: str) -> list[Product]:
    reader = csv.DictReader(io.StringIO(run_psql(database_url)))
    products: list[Product] = []
    for row in reader:
        try:
            price = float(row["price"] or 0)
        except ValueError:
            price = 0
        products.append(
            Product(
                sku=row["sku"].strip(),
                sales_sku=row["sales_sku"].strip(),
                group_id=row["group_id"].strip(),
                card_key=row["card_key"].strip(),
                name=row["name"].strip(),
                root_section=normalized(row["root_section"]),
                leaf_category=normalized(row["leaf_category"]),
                category_path=tuple(
                    normalized(part)
                    for part in parse_category_path(row["category_path"])
                ),
                gender=normalized(row["gender"]),
                price=price,
                image_url=row["image_url"].strip(),
            )
        )
    if not products:
        raise RuntimeError("eligible catalog is empty")
    return products


def http_json(
    method: str,
    url: str,
    payload: dict[str, Any] | None,
    timeout: float,
) -> tuple[Any, float]:
    body = None
    headers = {"Accept": "application/json"}
    if payload is not None:
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        headers["Content-Type"] = "application/json"
    request = urllib.request.Request(url, data=body, headers=headers, method=method)
    started = time.perf_counter()
    with urllib.request.urlopen(request, timeout=timeout) as response:
        raw = response.read()
    elapsed_ms = (time.perf_counter() - started) * 1000
    return json.loads(raw.decode("utf-8")), elapsed_ms


def get_optional_json(api_base: str, path: str, timeout: float) -> Any:
    try:
        value, _ = http_json("GET", api_base.rstrip("/") + path, None, timeout)
        return value
    except Exception as error:  # metadata failure must not lose the baseline
        return {"capture_error": str(error)}


def strategy_hash(strategy: Any) -> str:
    raw = json.dumps(
        strategy, ensure_ascii=False, sort_keys=True, separators=(",", ":")
    ).encode("utf-8")
    return hashlib.sha256(raw).hexdigest()


def request_batch(
    api_base: str,
    articles: list[str],
    limit: int,
    timeout: float,
    retries: int = 2,
) -> tuple[list[dict[str, Any]], float]:
    url = api_base.rstrip("/") + "/api/v1/client/products/similar"
    payload = {
        "articles": articles,
        "section": "all",
        "limit": limit,
        "exclude_product_ids": [],
    }
    last_error: Exception | None = None
    for attempt in range(retries + 1):
        try:
            response, elapsed_ms = http_json("POST", url, payload, timeout)
            results = response.get("results")
            if not isinstance(results, list):
                raise RuntimeError("batch response has no results array")
            return results, elapsed_ms
        except (urllib.error.URLError, TimeoutError, RuntimeError) as error:
            last_error = error
            if attempt < retries:
                time.sleep(0.25 * (attempt + 1))
    raise RuntimeError(f"batch failed after retries: {last_error}")


class QualityAccumulator:
    def __init__(self, limit: int) -> None:
        self.limit = limit
        self.base_count = 0
        self.error_count = 0
        self.error_codes: Counter[str] = Counter()
        self.lengths: list[int] = []
        self.full_count = 0
        self.empty_count = 0
        self.total_items = 0
        self.valid_items = 0
        self.unknown_items = 0
        self.self_items = 0
        self.duplicate_items = 0
        self.same_model_items = 0
        self.section_comparable = 0
        self.same_section_items = 0
        self.leaf_comparable = 0
        self.same_leaf_items = 0
        self.parent_comparable = 0
        self.same_parent_items = 0
        self.top1_valid = 0
        self.top1_same_section = 0
        self.top1_same_leaf = 0
        self.by_section: dict[str, QualityAccumulator] = {}

    def add(
        self,
        base: Product,
        items: list[str],
        error: str,
        aliases: dict[str, Product],
        include_section: bool = True,
    ) -> None:
        self.base_count += 1
        if error:
            self.error_count += 1
            self.error_codes[error] += 1
        length = len(items)
        self.lengths.append(length)
        if length >= self.limit:
            self.full_count += 1
        if length == 0:
            self.empty_count += 1

        seen: set[str] = set()
        for rank, article in enumerate(items, start=1):
            self.total_items += 1
            if article in seen:
                self.duplicate_items += 1
            seen.add(article)
            candidate = aliases.get(article)
            if candidate is None:
                self.unknown_items += 1
                continue
            self.valid_items += 1
            if article in {
                base.card_key,
                base.sku,
                base.sales_sku,
                base.group_id,
            }:
                self.self_items += 1
            if candidate.model_key == base.model_key:
                self.same_model_items += 1

            if base.root_section and candidate.root_section:
                self.section_comparable += 1
                if base.root_section == candidate.root_section:
                    self.same_section_items += 1
            if base.leaf_category and candidate.leaf_category:
                self.leaf_comparable += 1
                if base.leaf_category == candidate.leaf_category:
                    self.same_leaf_items += 1
            if base.parent_category and candidate.parent_category:
                self.parent_comparable += 1
                if base.parent_category == candidate.parent_category:
                    self.same_parent_items += 1

            if rank == 1:
                self.top1_valid += 1
                if (
                    base.root_section
                    and candidate.root_section
                    and base.root_section == candidate.root_section
                ):
                    self.top1_same_section += 1
                if (
                    base.leaf_category
                    and candidate.leaf_category
                    and base.leaf_category == candidate.leaf_category
                ):
                    self.top1_same_leaf += 1

        if include_section:
            section = base.root_section or "unknown"
            child = self.by_section.get(section)
            if child is None:
                child = QualityAccumulator(self.limit)
                self.by_section[section] = child
            child.add(base, items, error, aliases, include_section=False)

    @staticmethod
    def ratio(numerator: int, denominator: int) -> float | None:
        if denominator == 0:
            return None
        return numerator / denominator

    @staticmethod
    def percentile(values: list[int], q: float) -> float | None:
        if not values:
            return None
        ordered = sorted(values)
        index = max(0, min(len(ordered) - 1, math.ceil(q * len(ordered)) - 1))
        return float(ordered[index])

    def summary(self, with_sections: bool = True) -> dict[str, Any]:
        result: dict[str, Any] = {
            "limit": self.limit,
            "base_count": self.base_count,
            "error_count": self.error_count,
            "error_rate": self.ratio(self.error_count, self.base_count),
            "error_codes": dict(sorted(self.error_codes.items())),
            "empty_count": self.empty_count,
            "empty_rate": self.ratio(self.empty_count, self.base_count),
            "full_count": self.full_count,
            "fill_rate": self.ratio(self.full_count, self.base_count),
            "result_length": {
                "min": min(self.lengths) if self.lengths else None,
                "mean": statistics.fmean(self.lengths) if self.lengths else None,
                "p50": self.percentile(self.lengths, 0.50),
                "p95": self.percentile(self.lengths, 0.95),
                "max": max(self.lengths) if self.lengths else None,
            },
            "total_items": self.total_items,
            "valid_item_rate": self.ratio(self.valid_items, self.total_items),
            "unknown_item_count": self.unknown_items,
            "self_item_count": self.self_items,
            "duplicate_item_count": self.duplicate_items,
            "same_model_item_count": self.same_model_items,
            "same_section_rate": self.ratio(
                self.same_section_items, self.section_comparable
            ),
            "same_leaf_rate": self.ratio(
                self.same_leaf_items, self.leaf_comparable
            ),
            "same_parent_rate": self.ratio(
                self.same_parent_items, self.parent_comparable
            ),
            "top1_same_section_rate": self.ratio(
                self.top1_same_section, self.top1_valid
            ),
            "top1_same_leaf_rate": self.ratio(
                self.top1_same_leaf, self.top1_valid
            ),
        }
        if with_sections:
            result["sections"] = {
                section: child.summary(with_sections=False)
                for section, child in sorted(self.by_section.items())
            }
        return result


def aliases_for(products: list[Product]) -> dict[str, Product]:
    aliases: dict[str, Product] = {}
    for product in products:
        for key in (
            product.card_key,
            product.sku,
            product.sales_sku,
            product.group_id,
        ):
            if key:
                aliases.setdefault(key, product)
    return aliases


def deterministic_manual_sample(
    products: list[Product], per_section: int
) -> list[Product]:
    groups: dict[str, list[Product]] = defaultdict(list)
    for product in products:
        groups[product.root_section or "unknown"].append(product)
    sample: list[Product] = []
    for section in sorted(groups):
        ranked = sorted(
            groups[section],
            key=lambda product: hashlib.sha256(
                product.card_key.encode("utf-8")
            ).hexdigest(),
        )
        sample.extend(ranked[:per_section])
    return sample


def write_catalog_snapshot(path: pathlib.Path, products: list[Product]) -> None:
    fields = [
        "sku",
        "sales_sku",
        "group_id",
        "card_key",
        "name",
        "root_section",
        "leaf_category",
        "category_path",
        "gender",
        "price",
        "image_url",
    ]
    with gzip.open(path, "wt", encoding="utf-8", newline="") as stream:
        writer = csv.DictWriter(stream, fieldnames=fields)
        writer.writeheader()
        for product in products:
            writer.writerow(
                {
                    "sku": product.sku,
                    "sales_sku": product.sales_sku,
                    "group_id": product.group_id,
                    "card_key": product.card_key,
                    "name": product.name,
                    "root_section": product.root_section,
                    "leaf_category": product.leaf_category,
                    "category_path": json.dumps(
                        product.category_path, ensure_ascii=False
                    ),
                    "gender": product.gender,
                    "price": product.price,
                    "image_url": product.image_url,
                }
            )


def file_sha256(path: pathlib.Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        while chunk := stream.read(1024 * 1024):
            digest.update(chunk)
    return digest.hexdigest()


def git_metadata(repo: pathlib.Path) -> dict[str, Any]:
    def run(*args: str) -> str:
        completed = subprocess.run(
            ["git", *args],
            cwd=repo,
            check=True,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        return completed.stdout.strip()

    try:
        return {
            "commit": run("rev-parse", "HEAD"),
            "branch": run("branch", "--show-current"),
            "algorithm_diff": run(
                "diff",
                "--",
                "cmd",
                "internal",
                "go.mod",
                "go.sum",
            ),
        }
    except Exception as error:
        return {"capture_error": str(error)}


def capture_limit(
    *,
    api_base: str,
    products: list[Product],
    aliases: dict[str, Product],
    limit: int,
    batch_size: int,
    workers: int,
    timeout: float,
    output_path: pathlib.Path,
) -> tuple[dict[str, Any], dict[str, list[str]]]:
    articles = [product.card_key for product in products]
    batches = list(batched(articles, batch_size))
    responses: dict[str, dict[str, Any]] = {}
    batch_latencies: list[float] = []

    with ThreadPoolExecutor(max_workers=workers) as executor:
        futures = {
            executor.submit(
                request_batch, api_base, batch, limit, timeout
            ): batch
            for batch in batches
        }
        completed_batches = 0
        for future in as_completed(futures):
            batch = futures[future]
            try:
                batch_results, elapsed_ms = future.result()
                batch_latencies.append(elapsed_ms)
                for result in batch_results:
                    article = str(result.get("article", "")).strip()
                    if article:
                        responses[article] = {
                            "article": article,
                            "items": [
                                str(item).strip()
                                for item in result.get("items", [])
                                if str(item).strip()
                            ],
                            "error": str(result.get("error", "")).strip(),
                        }
            except Exception as error:
                for article in batch:
                    responses[article] = {
                        "article": article,
                        "items": [],
                        "error": f"capture_failed:{error}",
                    }
            completed_batches += 1
            if completed_batches % 25 == 0 or completed_batches == len(batches):
                print(
                    f"limit={limit}: {completed_batches}/{len(batches)} batches",
                    file=sys.stderr,
                    flush=True,
                )

    accumulator = QualityAccumulator(limit)
    exact: dict[str, list[str]] = {}
    with gzip.open(output_path, "wt", encoding="utf-8") as stream:
        for product in products:
            result = responses.get(
                product.card_key,
                {
                    "article": product.card_key,
                    "items": [],
                    "error": "missing_batch_result",
                },
            )
            items = result["items"]
            error = result["error"]
            exact[product.card_key] = items
            accumulator.add(product, items, error, aliases)
            stream.write(
                json.dumps(result, ensure_ascii=False, separators=(",", ":"))
                + "\n"
            )

    summary = accumulator.summary()
    summary["batch_latency_ms"] = {
        "count": len(batch_latencies),
        "mean": statistics.fmean(batch_latencies) if batch_latencies else None,
        "p50": QualityAccumulator.percentile(
            [round(value) for value in batch_latencies], 0.50
        ),
        "p95": QualityAccumulator.percentile(
            [round(value) for value in batch_latencies], 0.95
        ),
        "max": max(batch_latencies) if batch_latencies else None,
    }
    return summary, exact


def write_manual_review(
    path: pathlib.Path,
    sample: list[Product],
    exact: dict[str, list[str]],
    aliases: dict[str, Product],
    top_n: int = 12,
) -> None:
    fields = [
        "base_article",
        "base_name",
        "base_section",
        "base_leaf",
        "rank",
        "candidate_article",
        "candidate_name",
        "candidate_section",
        "candidate_leaf",
        "same_section",
        "same_leaf",
        "same_parent",
        "same_model",
        "candidate_image_url",
        "review_relevant",
        "review_comment",
    ]
    with path.open("w", encoding="utf-8", newline="") as stream:
        writer = csv.DictWriter(stream, fieldnames=fields)
        writer.writeheader()
        for base in sample:
            for rank, article in enumerate(exact.get(base.card_key, [])[:top_n], 1):
                candidate = aliases.get(article)
                writer.writerow(
                    {
                        "base_article": base.card_key,
                        "base_name": base.name,
                        "base_section": base.root_section,
                        "base_leaf": base.leaf_category,
                        "rank": rank,
                        "candidate_article": article,
                        "candidate_name": candidate.name if candidate else "",
                        "candidate_section": (
                            candidate.root_section if candidate else ""
                        ),
                        "candidate_leaf": (
                            candidate.leaf_category if candidate else ""
                        ),
                        "same_section": (
                            base.root_section == candidate.root_section
                            if candidate
                            else False
                        ),
                        "same_leaf": (
                            base.leaf_category == candidate.leaf_category
                            if candidate
                            else False
                        ),
                        "same_parent": (
                            base.parent_category == candidate.parent_category
                            if candidate
                            else False
                        ),
                        "same_model": (
                            base.model_key == candidate.model_key
                            if candidate
                            else False
                        ),
                        "candidate_image_url": (
                            candidate.image_url if candidate else ""
                        ),
                        "review_relevant": "",
                        "review_comment": "",
                    }
                )


def percent(value: Any) -> str:
    if value is None:
        return "n/a"
    return f"{float(value) * 100:.2f}%"


def write_markdown_summary(
    path: pathlib.Path,
    metadata: dict[str, Any],
    summaries: dict[str, dict[str, Any]],
) -> None:
    lines = [
        "# Baseline качества RecommendationEngine",
        "",
        f"- Время: `{metadata['captured_at']}`",
        f"- Git commit: `{metadata['git']['commit']}`",
        f"- Стратегия SHA-256: `{metadata['strategy_sha256']}`",
        f"- Товаров: **{metadata['catalog_count']}**",
        f"- API: `{metadata['api_base']}`",
        "",
        "> Метрики категорий — автоматические proxy-метрики, а не человеческая",
        "> оценка релевантности. Exact snapshots и ручная выборка являются",
        "> обязательной основой сравнения после рефакторинга.",
        "",
        "## Сводка",
        "",
        "| Limit | Ошибки | Пустые | Заполнено | Валидные ID | Same section | Same leaf | Same parent | Self | Дубли | Same model |",
        "|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|",
    ]
    for limit in sorted(summaries, key=int):
        value = summaries[limit]
        lines.append(
            "| {limit} | {errors} | {empty} | {fill} | {valid} | {section} | "
            "{leaf} | {parent} | {self_count} | {duplicates} | {same_model} |".format(
                limit=limit,
                errors=percent(value["error_rate"]),
                empty=percent(value["empty_rate"]),
                fill=percent(value["fill_rate"]),
                valid=percent(value["valid_item_rate"]),
                section=percent(value["same_section_rate"]),
                leaf=percent(value["same_leaf_rate"]),
                parent=percent(value["same_parent_rate"]),
                self_count=value["self_item_count"],
                duplicates=value["duplicate_item_count"],
                same_model=value["same_model_item_count"],
            )
        )
    lines.extend(
        [
            "",
            "## Критерий для performance-only рефакторинга",
            "",
            "- 100% одинаковых ordered ID для каждого article и каждого limit.",
            "- Нулевой рост errors, empty, unknown, self, duplicates и same-model.",
            "- Нулевое ухудшение fill rate и категориальных proxy-метрик.",
            "",
            "Осознанные изменения бизнес-логики проверяются отдельным diff-отчётом,",
            "ручной выборкой и явно утверждёнными порогами.",
            "",
            "## Артефакты",
            "",
        ]
    )
    for artifact in metadata["artifacts"]:
        lines.append(
            f"- `{artifact['file']}` — SHA-256 `{artifact['sha256']}`"
        )
    path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def main() -> int:
    args = parse_args()
    limits = parse_limits(args.limits)
    if args.batch_size <= 0 or args.batch_size > 50:
        raise ValueError("batch-size must be from 1 to 50")
    if args.workers <= 0:
        raise ValueError("workers must be positive")

    output_dir = pathlib.Path(args.output_dir).resolve()
    output_dir.mkdir(parents=True, exist_ok=False)
    repo = pathlib.Path(__file__).resolve().parents[1]
    api_base = args.api_base.rstrip("/")

    print("loading eligible catalog...", file=sys.stderr)
    products = load_catalog(args.database_url)
    aliases = aliases_for(products)
    print(f"eligible catalog: {len(products)} cards", file=sys.stderr)

    catalog_path = output_dir / "catalog.csv.gz"
    write_catalog_snapshot(catalog_path, products)

    strategy = get_optional_json(
        api_base, "/api/admin/similar-strategy", args.timeout
    )
    status = get_optional_json(api_base, "/api/data-status", args.timeout)
    warm_status = get_optional_json(
        api_base, "/api/admin/similar-strategy/warm-status", args.timeout
    )

    summaries: dict[str, dict[str, Any]] = {}
    exact_by_limit: dict[int, dict[str, list[str]]] = {}
    response_paths: list[pathlib.Path] = []
    for limit in limits:
        output_path = output_dir / f"responses-limit-{limit}.jsonl.gz"
        summary, exact = capture_limit(
            api_base=api_base,
            products=products,
            aliases=aliases,
            limit=limit,
            batch_size=args.batch_size,
            workers=args.workers,
            timeout=args.timeout,
            output_path=output_path,
        )
        summaries[str(limit)] = summary
        exact_by_limit[limit] = exact
        response_paths.append(output_path)

    manual_limit = max(limits)
    manual_path = output_dir / "manual-review-sample.csv"
    write_manual_review(
        manual_path,
        deterministic_manual_sample(
            products, args.manual_sample_per_section
        ),
        exact_by_limit[manual_limit],
        aliases,
    )

    artifacts = [catalog_path, *response_paths, manual_path]
    metadata: dict[str, Any] = {
        "schema_version": 1,
        "captured_at": datetime.now(timezone.utc).isoformat(),
        "api_base": api_base,
        "catalog_count": len(products),
        "limits": limits,
        "git": git_metadata(repo),
        "strategy_sha256": strategy_hash(strategy),
        "strategy": strategy,
        "data_status": status,
        "warm_status": warm_status,
        "artifacts": [
            {
                "file": artifact.name,
                "sha256": file_sha256(artifact),
                "bytes": artifact.stat().st_size,
            }
            for artifact in artifacts
        ],
    }

    summary_json_path = output_dir / "summary.json"
    summary_json_path.write_text(
        json.dumps(
            {"metadata": metadata, "quality": summaries},
            ensure_ascii=False,
            indent=2,
            sort_keys=True,
        )
        + "\n",
        encoding="utf-8",
    )
    metadata["artifacts"].append(
        {
            "file": summary_json_path.name,
            "sha256": file_sha256(summary_json_path),
            "bytes": summary_json_path.stat().st_size,
        }
    )

    summary_markdown_path = output_dir / "README.md"
    write_markdown_summary(summary_markdown_path, metadata, summaries)
    print(f"baseline written to {output_dir}", file=sys.stderr)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
