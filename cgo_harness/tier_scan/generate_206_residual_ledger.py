#!/usr/bin/env python3
"""Generate the 206 residual frame/domain ledger from existing artifacts.

This is report/artifact plumbing only. It reads the current smoke scan outputs
and an explicit domain map, then writes a TSV ledger plus a compact markdown
summary. It does not inspect parser implementation files or run parser tests.
"""

from __future__ import annotations

import argparse
import csv
import json
import re
from collections import Counter, defaultdict
from pathlib import Path
from typing import Any


REPO = Path(__file__).resolve().parents[2]
DEFAULT_SCAN = REPO / "cgo_harness/harness_out/tier_scan_parallel/current-206-smoke-20260625/merged"
DEFAULT_TSV = REPO / "docs/reports/206-residual-frame-domain-ledger-20260625.tsv"
DEFAULT_MD = REPO / "docs/reports/206-residual-frame-domain-ledger-20260625.md"

TSV_COLUMNS = [
    "grammar",
    "domain",
    "true_coding_priority",
    "observed_frame_lane",
    "current_artifact_family",
    "latest_overlay_status",
    "next_generalized_machinery_target",
    "evidence_path",
    "selected_file",
    "stop_reason",
    "truncated",
    "comparison_result",
    "root_pair",
    "diff_types",
    "notes",
]


DOMAIN_MAP = {
    "asm": ("true_programming_language", "high", "assembly language"),
    "blade": ("markup_template_style", "no", "template language"),
    "brightscript": ("true_programming_language", "high", "programming language"),
    "caddy": ("config_build", "no", "server configuration"),
    "cairo": ("true_programming_language", "high", "programming language"),
    "cobol": ("true_programming_language", "high", "programming language"),
    "commonlisp": ("true_programming_language", "high", "programming language"),
    "cooklang": ("data_grammar", "no", "recipe/document data grammar"),
    "cpp": ("true_programming_language", "high", "programming language"),
    "cuda": ("true_programming_language", "high", "programming language"),
    "cylc": ("config_build", "no", "workflow configuration"),
    "disassembly": ("true_programming_language", "high", "assembly/disassembly syntax"),
    "djot": ("prose_docs", "no", "documentation markup"),
    "dockerfile": ("config_build", "no", "container build file"),
    "doxygen": ("prose_docs", "no", "documentation comments"),
    "earthfile": ("config_build", "no", "build file"),
    "ebnf": ("data_grammar", "no", "grammar notation"),
    "eds": ("data_grammar", "no", "data/schema grammar"),
    "elsa": ("other", "no", "unmeasured zero-file grammar; domain left conservative"),
    "facility": ("other", "no", "interface/spec grammar; not prioritized as true coding without more evidence"),
    "fennel": ("true_programming_language", "high", "programming language"),
    "fsharp": ("true_programming_language", "high", "programming language"),
    "glsl": ("true_programming_language", "high", "shader programming language"),
    "groovy": ("true_programming_language", "high", "programming language"),
    "haxe": ("true_programming_language", "high", "programming language"),
    "html": ("markup_template_style", "no", "markup language"),
    "hurl": ("config_build", "no", "HTTP test/config DSL"),
    "jinja2": ("markup_template_style", "no", "template language"),
    "just": ("config_build", "no", "build/task file"),
    "kconfig": ("config_build", "no", "kernel configuration language"),
    "kdl": ("data_grammar", "no", "document/data language"),
    "less": ("markup_template_style", "no", "stylesheet language"),
    "matlab": ("true_programming_language", "high", "programming language"),
    "mermaid": ("markup_template_style", "no", "diagram markup"),
    "meson": ("config_build", "no", "build language"),
    "nginx": ("config_build", "no", "server configuration"),
    "norg": ("prose_docs", "no", "documentation markup"),
    "nushell": ("true_programming_language", "high", "shell/programming language"),
    "objc": ("true_programming_language", "high", "programming language"),
    "odin": ("true_programming_language", "high", "programming language"),
    "pascal": ("true_programming_language", "high", "programming language"),
    "perl": ("true_programming_language", "high", "programming language"),
    "powershell": ("true_programming_language", "high", "shell/programming language"),
    "promql": ("query_policy", "no", "query language"),
    "pug": ("markup_template_style", "no", "template language"),
    "purescript": ("true_programming_language", "high", "programming language"),
    "rego": ("query_policy", "no", "policy language"),
    "regex": ("data_grammar", "no", "pattern grammar"),
    "robot": ("config_build", "no", "test automation DSL"),
    "scala": ("true_programming_language", "high", "programming language"),
    "scss": ("markup_template_style", "no", "stylesheet language"),
    "sql": ("query_policy", "no", "query language"),
    "swift": ("true_programming_language", "high", "programming language"),
    "teal": ("true_programming_language", "high", "programming language"),
    "tlaplus": ("query_policy", "no", "formal specification language"),
    "tmux": ("config_build", "no", "terminal configuration"),
    "uxntal": ("true_programming_language", "high", "assembly language"),
    "vimdoc": ("prose_docs", "no", "documentation format"),
    "wat": ("true_programming_language", "high", "WebAssembly text format"),
    "wolfram": ("true_programming_language", "high", "programming language"),
}


OVERLAY_STATUS = {
    "cpp": (
        "post-retry reports migrate C++ from runtime frontier to accepted/non-truncated recovery-shape evidence; parity still residual",
        "recovery_shape_and_materialization_after_generalized_retry",
    ),
    "cuda": (
        "post-retry and scoped-first reports migrate CUDA from runtime frontier to accepted/non-truncated recovery-shape/materialization evidence",
        "scoped_c_recovery_selection_and_materialization",
    ),
    "glsl": (
        "post-retry report migrates GLSL from runtime frontier to accepted/non-truncated recovery-shape evidence; parity still residual",
        "recovery_shape_and_materialization_after_generalized_retry",
    ),
    "scala": (
        "post-smoke Scala reports keep production as iteration-limit frontier, refined to forest zero-width successor frontier after no-lookahead",
        "forest_zero_width_successor_frontier_after_no_lookahead",
    ),
}


RUNTIME_RE = re.compile(r"([A-Za-z0-9_]+)=([^\s]+)")


def read_tier_iv(scan_dir: Path) -> list[str]:
    rows = []
    for line in (scan_dir / "tier_iv.txt").read_text().splitlines():
        if not line.strip():
            continue
        rows.append(line.split()[0])
    return rows


def read_unmeasured(scan_dir: Path) -> list[str]:
    path = scan_dir / "unmeasured.txt"
    if not path.exists():
        return []
    return [line.split()[0] for line in path.read_text().splitlines() if line.strip()]


def load_family_map(scan_dir: Path) -> dict[str, list[str]]:
    summary = json.loads((scan_dir / "diagnostic_summary.json").read_text())
    families: dict[str, list[str]] = defaultdict(list)
    for family in summary.get("failureFamilies", []):
        name = family["name"]
        if name in {"clean", "clean_but_slow_perf", "zero_files_measured"}:
            continue
        for grammar in family.get("grammars", []):
            families[grammar].append(name)
    return {grammar: sorted(names) for grammar, names in families.items()}


def parse_runtime(runtime: str) -> dict[str, str]:
    return dict(RUNTIME_RE.findall(runtime or ""))


def collect_frame_evidence(scan_dir: Path) -> dict[str, dict[str, Any]]:
    evidence: dict[str, dict[str, Any]] = defaultdict(
        lambda: {
            "raw_log": "",
            "selected_file": "",
            "comparison_result": "",
            "root_pairs": set(),
            "diff_types": set(),
            "runtime": {},
            "truncated": "",
            "stop_reason": "",
        }
    )
    for line in (scan_dir / "frames.jsonl").read_text().splitlines():
        if not line:
            continue
        frame = json.loads(line)
        grammar = frame.get("grammar", "")
        if not grammar:
            continue
        row = evidence[grammar]
        if frame.get("raw_log"):
            row["raw_log"] = frame["raw_log"]
        if frame.get("phase") == "selected_file" and frame.get("path"):
            row["selected_file"] = frame["path"]
        runtime = parse_runtime(frame.get("runtime", ""))
        if runtime:
            row["runtime"].update(runtime)
            row["truncated"] = runtime.get("truncated", row["truncated"])
            row["stop_reason"] = runtime.get("stopReason", row["stop_reason"])
        if frame.get("phase") == "comparison_result":
            row["comparison_result"] = frame.get("result", "")
        if frame.get("phase") == "comparison_diag":
            result = frame.get("result", "")
            if result:
                row["comparison_result"] = result
            runtime_text = frame.get("runtime", "")
            for token in runtime_text.split():
                if token.startswith("root="):
                    row["root_pairs"].add(token.removeprefix("root="))
                elif token.startswith("diff="):
                    row["diff_types"].add(token.removeprefix("diff="))
    normalized: dict[str, dict[str, Any]] = {}
    for grammar, row in evidence.items():
        normalized[grammar] = {
            **row,
            "root_pairs": sorted(row["root_pairs"]),
            "diff_types": sorted(row["diff_types"]),
        }
    return normalized


def observed_lane(families: list[str], evidence: dict[str, Any], unmeasured: bool) -> str:
    if unmeasured:
        return "unmeasured_zero_files"
    if "runtime_frontier_stop" in families:
        reason = evidence.get("stop_reason") or "truncated"
        return f"runtime_frontier_stop/{reason}"
    if "recovery_error_cost" in families:
        return "recovery_error_cost/error_tree_or_root_error"
    if "accepted_shape_materialization" in families:
        return "accepted_shape_materialization/accepted_diverge"
    if "accepted_divergence_cost" in families:
        return "accepted_divergence_cost/accepted_diverge"
    return "unclear_needs_diagnostic"


def default_next_target(families: list[str], unmeasured: bool) -> str:
    if unmeasured:
        return "corpus_coverage_or_unmeasured_accounting"
    if "runtime_frontier_stop" in families:
        return "glr_frontier_survival_and_reuse_selection"
    if "recovery_error_cost" in families:
        return "generalized_recovery_shape_and_cost"
    if "accepted_shape_materialization" in families or "accepted_divergence_cost" in families:
        return "generalized_materialization_invariant"
    return "diagnostic_classification"


def build_rows(scan_dir: Path) -> list[dict[str, str]]:
    tier_iv = read_tier_iv(scan_dir)
    unmeasured = read_unmeasured(scan_dir)
    family_map = load_family_map(scan_dir)
    evidence_map = collect_frame_evidence(scan_dir)
    rows: list[dict[str, str]] = []
    for grammar in tier_iv + unmeasured:
        is_unmeasured = grammar in unmeasured
        domain, priority, note = DOMAIN_MAP.get(
            grammar,
            ("other", "no", "domain not classified in static map; conservative fallback"),
        )
        families = family_map.get(grammar, [])
        evidence = evidence_map.get(grammar, {})
        overlay_status, overlay_target = OVERLAY_STATUS.get(grammar, ("current_smoke_only", ""))
        target = overlay_target or default_next_target(families, is_unmeasured)
        rows.append(
            {
                "grammar": grammar,
                "domain": domain,
                "true_coding_priority": priority,
                "observed_frame_lane": observed_lane(families, evidence, is_unmeasured),
                "current_artifact_family": ";".join(families) if families else ("unmeasured" if is_unmeasured else "unclear"),
                "latest_overlay_status": overlay_status,
                "next_generalized_machinery_target": target,
                "evidence_path": evidence.get("raw_log", "") or str(scan_dir / "unmeasured.txt"),
                "selected_file": evidence.get("selected_file", ""),
                "stop_reason": evidence.get("stop_reason", ""),
                "truncated": evidence.get("truncated", ""),
                "comparison_result": evidence.get("comparison_result", ""),
                "root_pair": ";".join(evidence.get("root_pairs", [])),
                "diff_types": ";".join(evidence.get("diff_types", [])),
                "notes": note,
            }
        )
    return rows


def write_tsv(path: Path, rows: list[dict[str, str]]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=TSV_COLUMNS, dialect="excel-tab", lineterminator="\n")
        writer.writeheader()
        writer.writerows(rows)


def write_markdown(path: Path, rows: list[dict[str, str]], scan_dir: Path, tsv_path: Path) -> None:
    domain_counts = Counter(row["domain"] for row in rows)
    target_counts = Counter(row["next_generalized_machinery_target"] for row in rows)
    true_coding = sum(1 for row in rows if row["true_coding_priority"] == "high")
    unmeasured = sum(1 for row in rows if row["current_artifact_family"] == "unmeasured")
    top_targets = target_counts.most_common()
    rel_tsv = tsv_path.relative_to(REPO)
    rel_scan = scan_dir.relative_to(REPO)

    lines = [
        "# 206 Residual Frame/Domain Ledger, 2026-06-25",
        "",
        "This report is generated from the current 206 smoke artifact plus an explicit static domain map in",
        "`cgo_harness/tier_scan/generate_206_residual_ledger.py`. It is report-only evidence and does not update",
        "`cgo_harness/tier_scan/tier_classification.tsv`.",
        "",
        f"- Source artifact: `{rel_scan}`",
        f"- Machine-readable ledger: `{rel_tsv}`",
        f"- Rows: {len(rows)} ({len(rows) - unmeasured} current Tier IV residuals plus {unmeasured} unmeasured)",
        f"- True-coding high-priority rows: {true_coding}",
        "",
        "## Domain Counts",
        "",
        "| Domain | Count |",
        "| --- | ---: |",
    ]
    for domain, count in sorted(domain_counts.items()):
        lines.append(f"| {domain} | {count} |")

    lines += [
        "",
        "## Top Next Machinery Lanes",
        "",
        "| Next generalized machinery target | Count |",
        "| --- | ---: |",
    ]
    for target, count in top_targets:
        lines.append(f"| `{target}` | {count} |")

    lines += [
        "",
        "## High-Priority True-Coding Residuals",
        "",
        "| Grammar | Current frame/lane | Latest overlay | Next target |",
        "| --- | --- | --- | --- |",
    ]
    for row in rows:
        if row["true_coding_priority"] != "high":
            continue
        overlay = row["latest_overlay_status"]
        if overlay != "current_smoke_only":
            overlay = overlay.split(";")[0]
        lines.append(
            f"| {row['grammar']} | `{row['observed_frame_lane']}` | {overlay} | `{row['next_generalized_machinery_target']}` |"
        )

    lines += [
        "",
        "## Classification Policy",
        "",
        "The classification path is generalized parser machinery only: GLR frontier survival, recovery-shape/cost",
        "selection, forest/materialization, materialization invariants, or corpus accounting. The ledger is not a",
        "justification for per-grammar normalizers or language-name parser policy.",
        "",
    ]
    path.write_text("\n".join(lines))


def validate(rows: list[dict[str, str]]) -> None:
    grammars = [row["grammar"] for row in rows]
    if len(grammars) != len(set(grammars)):
        dupes = sorted(grammar for grammar, count in Counter(grammars).items() if count > 1)
        raise SystemExit(f"duplicate ledger grammars: {', '.join(dupes)}")
    missing_domain = [
        row["grammar"]
        for row in rows
        if row["domain"] == "other" and row["grammar"] != "elsa" and "not prioritized" not in row["notes"]
    ]
    if missing_domain:
        raise SystemExit(f"unclassified domains: {', '.join(missing_domain)}")
    if len(rows) != 60:
        raise SystemExit(f"expected 60 residual rows, got {len(rows)}")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--scan-dir", type=Path, default=DEFAULT_SCAN)
    parser.add_argument("--tsv", type=Path, default=DEFAULT_TSV)
    parser.add_argument("--markdown", type=Path, default=DEFAULT_MD)
    args = parser.parse_args()

    rows = build_rows(args.scan_dir)
    validate(rows)
    write_tsv(args.tsv, rows)
    write_markdown(args.markdown, rows, args.scan_dir, args.tsv)
    true_coding = sum(1 for row in rows if row["true_coding_priority"] == "high")
    unmeasured = sum(1 for row in rows if row["current_artifact_family"] == "unmeasured")
    targets = Counter(row["next_generalized_machinery_target"] for row in rows)
    print(f"wrote {args.tsv}")
    print(f"wrote {args.markdown}")
    print(f"rows={len(rows)} unmeasured={unmeasured} true_coding_high={true_coding}")
    print("top_targets=" + ", ".join(f"{name}:{count}" for name, count in targets.most_common(5)))


if __name__ == "__main__":
    main()
