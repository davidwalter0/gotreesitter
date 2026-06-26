# Common Lisp Tier IV Residual Narrowing, 2026-06-25

Scope: report-only correctness narrowing for the 206 true-coding classification goal. No parser, grammar, or normalizer source was edited.

Worktree: `/home/draco/work/gotreesitter-build-baseline`

Source head during focused verification: `27c672a0f7540608d7b313312ccc04071c08c1e1`

Primary input artifact:

- `cgo_harness/harness_out/tier_scan_parallel/current-tieriv-n40-20260625/workers/shard-006`

Focused current-source verification artifact:

- `cgo_harness/harness_out/tier_scan/commonlisp-residual-focused-20260625`

## Exact Residual Frames From Current N=40 Artifact

The prompt-cited N=40 artifact contains eight Common Lisp residual frames: 1, 4, 8, 9, 11, 17, 20, and 21. All eight child processes ended with `rc=0`; there are no terminal timeout/fail rows in this artifact.

| frame | corpus path | parity / diverge / trunc / errTree | stopReason | root / first-diff evidence | terminal | runtime accounting |
| ---: | --- | --- | --- | --- | --- | --- |
| 1 | `/home/draco/work/gotreesitter-corpora/corpus_sources/commonlisp/benchmarks/bbtrees.lisp` | `0/1`, diverge=1, trunc=1, errTree=1 | `no_stacks_alive` | `root` type diff: Go `ERROR` span `0:2978`, C `source` span `0:7395`; `goRootErr=true`, `cRootErr=true`, `goErrors=1`, `cErrors=1` | none; `END`, rc=0 | tokens=802, lastTokenEnd=2980, expectedEOF=7395, lastTokenSymbol=14, nodes=91522/384540, maxStacks=56 |
| 4 | `/home/draco/work/gotreesitter-corpora/corpus_sources/commonlisp/benchmarks/grab-mutex.lisp` | `0/1`, diverge=1, trunc=1, errTree=1 | `no_stacks_alive` | `root` type diff: Go `ERROR` span `0:4164`, C `source` span `0:10432`; `goRootErr=true`, `cRootErr=true`, `goErrors=2`, `cErrors=1` | none; `END`, rc=0 | tokens=931, lastTokenEnd=4167, expectedEOF=10432, lastTokenSymbol=53, nodes=98057/542464, maxStacks=56 |
| 8 | `/home/draco/work/gotreesitter-corpora/corpus_sources/commonlisp/benchmarks/hash-table/benchmark-sbcl-hash-tables.lisp` | `0/1`, diverge=1, trunc=1, errTree=0 | `no_stacks_alive` | `root` type diff: Go `source` span `0:17832`, C `ERROR` span `0:28844`; `goRootErr=false`, `cRootErr=true`, `goErrors=0`, `cErrors=2`, `cMissing=1` | none; `END`, rc=0 | tokens=4234, lastTokenEnd=17835, expectedEOF=28844, lastTokenSymbol=8, nodes=160600/1499888, maxStacks=24 |
| 9 | `/home/draco/work/gotreesitter-corpora/corpus_sources/commonlisp/benchmarks/hash-table/deltist.lisp` | `0/1`, diverge=1, trunc=0, errTree=0 | `accepted` | `root[75]` child-count diff on `list_lit`: Go child count 5, C child count 6; `goRootErr=false`, `cRootErr=true`, `goErrors=0`, `cErrors=1` | none; `END`, rc=0 | tokens=4528, lastTokenEnd=21249, expectedEOF=21249, nodes=229618/1104948, maxStacks=24 |
| 11 | `/home/draco/work/gotreesitter-corpora/corpus_sources/commonlisp/benchmarks/rwlbench2.lisp` | `0/1`, diverge=1, trunc=1, errTree=1 | `no_stacks_alive` | `root` type diff: Go `ERROR` span `0:3355`, C `source` span `0:3894`; `goRootErr=true`, `cRootErr=true`, `goErrors=1`, `cErrors=2` | none; `END`, rc=0 | tokens=881, lastTokenEnd=3358, expectedEOF=3894, lastTokenSymbol=62, nodes=83104/300000, maxStacks=56 |
| 17 | `/home/draco/work/gotreesitter-corpora/corpus_sources/commonlisp/contrib/compiler-extras.lisp` | `0/1`, diverge=1, trunc=0, errTree=0 | `accepted` | `root[36][5][5][0]` child-count diff on `loop_macro`: Go child count 13, C child count 10; `goRootErr=false`, `cRootErr=false`, `goErrors=0`, `cErrors=0` | none; `END`, rc=0 | tokens=0, lastTokenEnd=8994, expectedEOF=8994, iterations=0/0, nodes=0/0, maxStacks=0 |
| 20 | `/home/draco/work/gotreesitter-corpora/corpus_sources/commonlisp/contrib/sb-aclrepl/inspect.lisp` | `0/1`, diverge=1, trunc=1, errTree=1 | `no_stacks_alive` | `root` type diff: Go `ERROR` span `0:14908`, C `source` span `0:32494`; `goRootErr=true`, `cRootErr=true`, `goErrors=3`, `cErrors=1`, `cMissing=1` | none; `END`, rc=0 | tokens=2982, lastTokenEnd=14911, expectedEOF=32494, lastTokenSymbol=51, nodes=425519/1689688, maxStacks=56 |
| 21 | `/home/draco/work/gotreesitter-corpora/corpus_sources/commonlisp/contrib/sb-aclrepl/repl.lisp` | `0/1`, diverge=1, trunc=1, errTree=1 | `no_stacks_alive` | `root` type diff: Go `ERROR` span `0:13603`, C `source` span `0:30332`; `goRootErr=true`, `cRootErr=true`, `goErrors=1`, `cErrors=11`, `goMissing=2` | none; `END`, rc=0 | tokens=3251, lastTokenEnd=13607, expectedEOF=30332, lastTokenSymbol=110, nodes=395603/1577264, maxStacks=56 |

## Classification

The N=40 residuals are not one issue.

- Six frames are `no_stacks_alive` frontier truncations before EOF: 1, 4, 8, 11, 20, and 21.
- One frame is accepted/full-EOF recovery-shape cost: 9.
- One frame is accepted/full-EOF scanner/token-accounting or no-tree accounting: 17 (`tokens=0`, `iterations=0/0`, `nodes=0/0`) with a shape-only `loop_macro` child-count diff.

Small generalized stack/merge probes did not produce a defensible machinery fix:

- `GOT_GLR_MAX_STACKS=112` on frame 1 still stopped at the same byte region (`lastTokenEnd=2980`, expected EOF 7395) and remained `0/1` parity; it was slower than the baseline.
- `GOT_GLR_MAX_STACKS=112` on frame 8 timed out in `go_parse_start` under the same 120s frame bound.
- `GOT_GLR_MAX_MERGE_PER_KEY=1` on frame 1 still stopped at `lastTokenEnd=2980`.
- `GOT_GLR_MAX_MERGE_PER_KEY=1` on frame 9 still produced the same accepted `root[75]` `list_lit` child-count family.

Because these probes did not identify a stable generalized invariant, no parser-machinery edit was made.

## Focused Current-Source Docker Verification

Command:

```bash
GTS_CORPUS_DIR=/home/draco/work/gotreesitter-corpora/corpus_sources \
GTS_TIER_SCAN_LANGS=commonlisp \
GTS_TIER_SCAN_N=40 \
GTS_TIER_SCAN_ROUNDS=1 \
GTS_TIER_SCAN_ISOLATE_FILES=1 \
GTS_TIER_SCAN_SKIP_TIER_PUBLISH=1 \
GTS_TIER_SCAN_KEEP_PARSER_PROGRESS_ROWS=0 \
GTS_TIER_SCAN_MAX_FRAME_ROWS=2000 \
GTS_TIER_SCAN_TIMEOUT=120 \
GTS_TIER_SCAN_KILL_AFTER=20s \
GTS_TIER_SCAN_FRAMES=1,2,4,8,9,11,17,20,21 \
bash cgo_harness/docker/run_tier_scan.sh \
  cgo_harness/harness_out/tier_scan/commonlisp-residual-focused-20260625
```

Result:

- Build succeeded.
- Selected frames: the eight residual frames plus clean control frame 2.
- Aggregate: `files=9`, `parityMatch=1/9`, `diverge=8`, `trunc=0`, `errTree=5`, `panics=0`.
- Terminal evidence: frame 8 timed out in `go_parse_start`, `rc=124`.
- Diagnostic families: `terminal_timeout_or_fail`, `recovery_error_cost`, and `scanner_token_accounting`.
- Common Lisp remains Tier IV.

Current-source focused frame outcomes:

| frame | focused result | current-source evidence |
| ---: | --- | --- |
| 1 | residual | Accepted/full-EOF child-count diff on `str_lit`; `goRootErr=true`, `cRootErr=true`; tokens=1814, nodes=80172/384540, maxStacks=24 |
| 2 | clean control | `parityMatch=1/1`, diverge=0, trunc=0, errTree=0 |
| 4 | residual | Accepted/full-EOF child-count diff on `str_lit`; `goRootErr=true`, `cRootErr=true`; tokens=1105, nodes=42228/542464, maxStacks=18 |
| 8 | terminal residual | `TIMEOUT`, `timeout:go_parse_start:rc=124` |
| 9 | residual | Accepted/full-EOF `root[75]` child-count diff on `list_lit`; `goRootErr=false`, `cRootErr=true`; tokens=4528, nodes=308345/1104948, maxStacks=24 |
| 11 | residual | Accepted/full-EOF child-count diff on `str_lit`; `goRootErr=true`, `cRootErr=true`; tokens=1027, nodes=39107/300000, maxStacks=18 |
| 17 | residual | Accepted/full-EOF `tokens=0` child-count diff on `loop_macro`; no root errors; iterations=0/0, nodes=0/0, maxStacks=0 |
| 20 | residual | Accepted/full-EOF child-count diff on `str_lit`; `goRootErr=true`, `cRootErr=true`; tokens=7191, nodes=744912/1689688, maxStacks=18 |
| 21 | residual | Accepted/full-EOF child-count diff on `str_lit`; `goRootErr=true`, `cRootErr=true`; tokens=7500, nodes=515054/1577264, maxStacks=24 |

## Next Generalized Machinery Target

The nearest current-source target is no longer a pure frontier-width change. The residual evidence points to generalized recovery-shape/materialization and string-literal child materialization under accepted error trees, with a separate scanner/token-accounting path for frame 17 and a terminal fanout case on frame 8. A safe next move should start with a first-diff/materialization trace around the repeated `str_lit` child-count family, not with a Common Lisp-specific normalizer or language-name parser policy.
