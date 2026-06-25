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

## Post-Retry Evidence, 2026-06-25

Post-patch retry witnesses show three runtime-frontier controls moved
into accepted/non-truncated recovery-shape evidence under the default
generalized C-recovery retry.

| Grammar/file | Prior frame | Post-retry frame | Remaining divergence | Witness |
| --- | --- | --- | --- | --- |
| cpp `args.h` | `runtime_frontier_stop`: `no_stacks_alive`, `truncated` | `accepted`, non-truncated, EOF reached under default retry | parity remains `0/1` because the recovery tree shape still differs | `docs/reports/cpp-recovery-admission-witness-20260625.md` |
| cuda `UnifiedMemoryStreams.cu` | `runtime_frontier_stop`: `no_stacks_alive`, `truncated` | `accepted`, non-truncated, EOF reached under default retry | parity remains `0/1` because `recovery_error_shape` still differs | `docs/reports/cuda-recovery-retry-witness-20260625.md` |
| glsl `100.frag` | `runtime_frontier_stop`: `no_stacks_alive`, `truncated`, root `ERROR->translation_unit`, tokens `320`, EOF not reached | `accepted`, non-truncated, EOF reached under default retry, root `translation_unit->translation_unit` | parity remains `0/1` due `recovery_error_shape`/root child-count; Go/C errors `6/5`, missing `3/2` | `docs/reports/glsl-recovery-frontier-witness-20260625.md` |

N=40 post-retry confirmation:
`docs/reports/post-retry-coding-controls-n40-20260625.md`

| Grammar | Parity | Trunc | Clean | Recovery error shape | Version or corpus |
| --- | ---: | ---: | ---: | ---: | ---: |
| cpp | 10/40 | 0 | 10 | 20 | 10 |
| cuda | 21/40 | 0 | 21 | 7 | 12 |
| glsl | 14/40 | 0 | 14 | 9 | 17 |

All three grammars had `runtime_frontier/truncation=0`, terminal
`timeout/fail=0`, and `accepted_shape/materialization=0` in this sweep.
C++, CUDA, and GLSL are no longer runtime-frontier targets in current
post-retry evidence. Remaining work is recovery-shape/version-or-corpus
classification and generalized recovery/materialization machinery. This does
not change the policy stance: do not add per-grammar normalizers or
language-name policies for these witnesses.

This supports the generalized retry/recovery machinery direction, not
per-grammar normalizers. GLSL joins C++ and CUDA as migrated true-coding
frontier witnesses. Scala remains a distinct iteration-limit frontier control.
The next coding-language target should be recovery-shape and materialization
invariant work.

Accepted-error pressure-floor follow-up:
`docs/reports/accepted-error-crecovery-pressure-floor-20260625.md`
falsified the pressure-floor admission hypothesis for CUDA frame 1. The frame
was already admitted under default retry (`initial=8`, `maxStacks=24`,
`admit=true`) and did not move by default, while forced `GOT_C_RECOVERY=all`
lowered frame 1 to `1/2` errors/missing and kept frame 3 as a no-error control.
Do not treat this as additional parity movement; the next machinery target is
scoped-vs-forced C-recovery behavior, selection, and materialization.

Scoped-first C-recovery follow-up:
`docs/reports/accepted-error-crecovery-scoped-first-20260625.md` moved CUDA
frame 1's default result to the forced candidate shape (`1/2` errors/missing)
by scheduling one scoped C-recovery retry before merge/widening retries for
accepted full-EOF error trees. CUDA frame 3 remained the `0/0` no-error control.
The frame 1 domain is still recovery tree shape/materialization, not admission.

Scoped-first CUDA N=40 classification:
`docs/reports/scoped-first-cuda-n40-20260625.md`

| Sweep | Parity | Trunc | Clean | Recovery error shape | Version or corpus | Terminal failures | OOM |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| CUDA post-retry controls | 21/40 | 0 | 21 | 7 | 12 | 0 | false |
| CUDA scoped-first | 21/40 | 0 | 21 | 6 | 13 | 0 | false |

The scoped-first CUDA N=40 sweep completed cleanly under Docker. The
`recovery_error_shape` family remains present beyond frame 1, with frames
1, 7, 10, 11, 12, and 34 in that family. This is a classification update only
and does not support performance conclusions.

Scala frame 1 follow-up:
`docs/reports/scala-frontier-frame1-20260625.md`

| Grammar/file | Current frame | Variant evidence | Classified next lane |
| --- | --- | --- | --- |
| scala `AutomaticModuleName.scala` | production remains `iteration_limit`, `truncated`, parity `0/1`, no errors or missing nodes, Go span `0:311` vs C span `0:658` | `stack2`, `stack8`, and `node3` do not change stop reason, EOF progress, span, or parity; `forest` removes runtime `iteration_limit` telemetry and advances Go span to `0:395` but remains non-parity and short of EOF | `forest/materialization` |

Scala remains a true-coding runtime-frontier witness under production settings,
but the next useful machinery lane is forest/materialization rather than node
budget, stack/frontier cap, recovery-shape, or version/corpus. The first-diff
shape is a block-comment materialization mismatch: C emits one `block_comment`
root child, while Go materializes many `block_comment_repeat1` children and
truncates inside the comment under production settings.

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
