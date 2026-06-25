# 206 Frame/Domain Classification, 2026-06-25

## Scope

This is a report-only classification artifact for the current 206 grammar goal.
It does not regenerate the canonical tier TSV, does not update
`tier_classification.tsv`, and does not claim a new gating baseline.

Evidence sources:

- Current 206 smoke artifact:
  `cgo_harness/harness_out/tier_scan_parallel/current-206-smoke-20260625`
- N=40 coding accepted-shape artifact:
  `cgo_harness/harness_out/tier_scan_parallel/coding-accepted-shape-n40-20260625`
- Post-F# first N=40 artifact:
  `cgo_harness/harness_out/tier_scan_parallel/fsharp-post-normalizer-n40-20260625`
- Post-hardening witness artifact:
  `harness_out/docker/20260625T081254Z-fsharp-post-hardening-witness-20260625`

## Methodology

Classify first by observed frame, then by domain so the next fix can target
shared machinery or a language family.

| Observed frame | Classification lane | Meaning |
| --- | --- | --- |
| Accepted/no-error divergence | Materialization lane | The parser accepts, but the produced tree shape differs from the cgo baseline. |
| Error tree | Recovery lane | The parse completes with error-node structure that needs recovery-cost or recovery-shape work. |
| Trunc/no_stacks_alive/node/memory/iteration | Runtime frontier lane | GLR frontier progress, stack survival, node budget, memory, or iteration limits stop the run before a useful accepted comparison. |
| Timeout/fail | Terminal/control lane | The run exits through timeout, process failure, or another terminal harness/control condition. |

Domain axis: coding, query, config, markup, prose, data, other.

## Current 206 Snapshot

Artifact:
`cgo_harness/harness_out/tier_scan_parallel/current-206-smoke-20260625`

| Metric | Count |
| --- | ---: |
| Visited grammars | 206 |
| Measured grammars | 205 |
| Clean | 146 |
| Tier IV | 59 |
| Unmeasured | 1 |

Failure-family counts:

| Failure family | Count |
| --- | ---: |
| runtime_frontier_stop | 41 |
| recovery_error_cost | 30 |
| accepted_shape_materialization | 11 |
| accepted_divergence_cost | 13 |
| terminal_timeout_or_fail | 0 |
| scanner_token_accounting | 0 |
| unclear_needs_diagnostic | 0 |

Zero-file/unmeasured: elsa external zero-file.

## Coding Accepted-Shape N=40

Artifact:
`cgo_harness/harness_out/tier_scan_parallel/coding-accepted-shape-n40-20260625`

| Grammar | Clean | Diverge | Trunc | Error tree | Failed files |
| --- | ---: | ---: | ---: | ---: | ---: |
| fennel | 15/40 | 25 | 9 | 18 | 0 |
| fsharp stale pre-patch | 0/40 | 40 | 29 | 12 | 2 |
| perl | 0/40 | 40 | 7 | 7 | 0 |
| purescript | 1/40 | 39 | 1 | 0 | 0 |
| swift | 0/40 | 40 | 27 | 9 | 0 |

The N=40 accepted-shape sample proves all five seed grammars are mixed, not
pure materialization cases.

## Coding-Language Priority Queue

Accepted/materialization seed: fsharp, fennel, perl, purescript, swift.

- All accepted/materialization seeds are mixed.
- PureScript is the cleanest next materialization target because its N=40 row
  has 1 truncation, 0 error trees, and no failed files.
- F# had a real accepted-shape frame available, but later evidence shows its
  residual state is no longer primarily a pure materialization problem.

Runtime/frontier coding-first:

- cpp, cuda, asm, uxntal, scala, wat, glsl, groovy
- haxe, matlab, commonlisp, cobol, nushell, powershell, cairo, teal

Recovery coding-first: objc, odin, pascal, wolfram.

Nearby query/DSL recovery target: sql.

Lower-priority non-true-coding buckets: config/build, markup/template/style,
prose/docs, query/policy, data/grammar.

These lower-priority groups are still important for the 206 goal, but they
should not displace coding-language fixes while the accepted-shape,
runtime-frontier, and recovery queues still have coding-first candidates.

## F# State

Post-F# first N=40 artifact, built after the initial F# normalizer but before
later hardening:
`cgo_harness/harness_out/tier_scan_parallel/fsharp-post-normalizer-n40-20260625`

| Metric | Result |
| --- | ---: |
| Parity | 1/40 |
| Accepted/no-error | 8 |
| Error tree | 12 |
| Trunc/frontier | 29 |
| Terminal timeout/fail | 2 |

Classification:

- One accepted-shape frame was cleared.
- F# remains Tier IV.
- The residual F# lane is runtime/recovery/terminal, not a completed
  materialization-only win.
- A full post-hardening N=40 row is still pending if we want a complete
  after-hardening comparison.

Post-hardening witness artifact:
`harness_out/docker/20260625T081254Z-fsharp-post-hardening-witness-20260625`

- `go test . -run "TestNormalizeFSharp" -count=1` passed.
- F# `DesignTimeBuild.fs` first-diff witness passed with
  `(no structural divergence)`.

The witness confirms the targeted F# normalization behavior, but it does not
replace a full post-hardening N=40 classification row.

## Next Steps

1. Bank the F# normalizer patch.
2. Use PureScript for the next materialization proof.
3. Create a generated/domain-tagged classification artifact, or update
   `tier_classification.tsv`, as a separate step.
4. Defer performance work until classification and parity are stable.
