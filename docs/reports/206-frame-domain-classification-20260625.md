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

Accepted/materialization-invariant seed: fsharp, fennel, perl, purescript,
swift.

- All accepted/materialization seeds are mixed.
- Accepted/no-error divergence should be used as a generalized
  materialization-invariant probe, not as a trigger for per-language
  normalizer patches.
- PureScript and F# are retained as evidence-bearing probes only; they are not
  per-language materialization targets.
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

## Generalized Machinery Witness Set

The first machinery experiment should prefer `runtime_frontier_stop` and
frontier survival over accepted-shape normalization.

Representative witnesses:

| Grammar/file | Evidence | Raw log |
| --- | --- | --- |
| cpp `args.h` | `stopReason=no_stacks_alive`, `trunc=true`, errors `go=2 c=10`, nodes `132/373984`, `maxStacks=3`, root `ERROR->translation_unit`, diff type | `cgo_harness/harness_out/tier_scan_parallel/current-206-smoke-20260625/workers/shard-000/measure-cpp-external-frame-0001.log` |
| cuda `UnifiedMemoryStreams.cu` | `stopReason=no_stacks_alive`, `trunc=true`, errors `0/0`, nodes `17793/637208`, `maxStacks=24`, root `translation_unit->translation_unit`, diff span | `cgo_harness/harness_out/tier_scan_parallel/current-206-smoke-20260625/workers/shard-004/measure-cuda-external-frame-0001.log` |
| scala `AutomaticModuleName.scala` | `stopReason=iteration_limit`, `trunc=true`, errors `0/0`, nodes `39065/300000`, `maxStacks=3`, root `compilation_unit->compilation_unit`, diff span | `cgo_harness/harness_out/tier_scan_parallel/current-206-smoke-20260625/workers/shard-006/measure-scala-external-frame-0001.log` |
| glsl `100.frag` | `stopReason=no_stacks_alive`, `trunc=true`, errors `go=1 c=5`, nodes `26642/300000`, `maxStacks=72`, root `ERROR->translation_unit`, diff type | `cgo_harness/harness_out/tier_scan_parallel/current-206-smoke-20260625/workers/shard-016/measure-glsl-external-frame-0001.log` |
| fsharp `DesignTimeBuild.fs` | Accepted/no-error control only: `stopReason=accepted`, `trunc=false`, errors `0/0`, nodes `507228/600000`, `maxStacks=24`, root `file->file`, first diff `value_declaration_left->function_declaration_left`, diff type | `cgo_harness/harness_out/tier_scan_parallel/current-206-smoke-20260625/workers/shard-008/measure-fsharp-external-frame-0001.log` |

Generic hypothesis: viable GLR frontiers are being culled, merged incorrectly,
or cost-ranked behind recovery paths before EOF. The accepted/no-error control
probes the generalized materialization invariant only.

Falsification criteria:

- No improvement in token progress, stop reason, root pair, or parity.
- No viable frontier is observed near the failure.
- Only one grammar improves while the others remain unchanged or regress.

Do not add per-grammar normalizers or language-name policies for these
witnesses.

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

The witness captured the targeted F# normalization behavior from the discarded
patch, but it does not justify banking grammar-specific normalizer code or
replace a full post-hardening N=40 classification row.

## Next Steps

1. Treat accepted/no-error divergence as a generalized
   materialization-invariant probe.
2. Fix shared parser materialization machinery only; do not introduce
   language-specific compatibility behavior for this 206 effort.
3. Create a generated/domain-tagged classification artifact, or update
   `tier_classification.tsv`, as a separate step.
4. Defer performance work until classification and parity are stable.
