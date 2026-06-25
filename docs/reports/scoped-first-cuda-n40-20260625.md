# Scoped-First CUDA N=40 Classification Sweep - 2026-06-25

Scope: correctness-only CUDA wringer classification after commit `b22a5a7a`
(`update(parser): Schedule scoped C-recovery before widening retries`). This
documents bucket movement only. It does not make performance conclusions.

Run command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scoped-first-cuda-n40-20260625 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- \
  "cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=40 GTS_WRINGER_STAGES=baseline,summary GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_TIMEOUT=60 bash cgo_harness/docker/run_grammar_integrity_wringer.sh cuda cgo_harness/harness_out/scoped-first-cuda-n40-20260625"
```

The run used one grammar (`cuda`) in Docker with the existing read-only corpus
mount convention.

## Result

| Commit | Docker exit | OOM killed | Parity | Clean frames | Diverge | Trunc | ErrTree | Terminal failures |
| --- | ---: | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `b22a5a7a` | 0 | false | 21/40 | 21 | 19 | 0 | 7 | 0 |

Wringer family counts:

| Family | Count |
| --- | ---: |
| clean | 21 |
| recovery_error_shape | 6 |
| version_or_corpus | 13 |

Diagnostic-family summary:

| Diagnostic family | Count |
| --- | ---: |
| terminal_timeout_or_fail | 0 |
| runtime_frontier_stop | 0 |
| recovery_error_cost | 1 grammar bucket |
| accepted_shape_materialization | 0 |
| accepted_divergence_cost | 1 grammar bucket |

The baseline scan reported `exit_status=0`, `run_state=complete`,
`events.fail=0`, and `events.timeout=0`. The wrapper metadata reported
`exit_code=0` and `oom_killed=false`.

## Recovery Error Shape Frames

`recovery_error_shape` is present beyond frame 1 in this post-scoped-first
N=40 sweep.

| Frame | File | Family reasons |
| ---: | --- | --- |
| 1 | `UnifiedMemoryStreams.cu` | `go_root_error_exceeds_c`, `measure_err_tree`, `go_error_count_exceeds_c` |
| 7 | `matrixMul.cu` | `go_root_error_exceeds_c`, `measure_err_tree`, `go_error_count_exceeds_c` |
| 10 | `bitonic.cu` | `go_root_error_exceeds_c`, `measure_err_tree`, `go_error_count_exceeds_c` |
| 11 | `mergeSort.cu` | `go_root_error_exceeds_c`, `measure_err_tree`, `go_error_count_exceeds_c` |
| 12 | `simpleAWBarrier.cu` | `go_root_error_exceeds_c`, `measure_err_tree` |
| 34 | `simpleStreams.cu` | `measure_err_tree`, `go_error_count_exceeds_c` |

## Before/After

Previous CUDA N=40 reference:
`docs/reports/post-retry-coding-controls-n40-20260625.md`.

| Sweep | Parity | Clean | Recovery error shape | Version or corpus | Trunc |
| --- | ---: | ---: | ---: | ---: | ---: |
| Post-retry coding controls | 21/40 | 21 | 7 | 12 | 0 |
| Scoped-first CUDA | 21/40 | 21 | 6 | 13 | 0 |

The aggregate parity and truncation counts did not move. One frame moved from
`recovery_error_shape` into `version_or_corpus` classification in this
classification sweep, while `recovery_error_shape` remains represented beyond
frame 1.

## Artifacts

- Wringer output:
  `cgo_harness/harness_out/scoped-first-cuda-n40-20260625`
- Summary JSON:
  `cgo_harness/harness_out/scoped-first-cuda-n40-20260625/wringer_summary.json`
- Diagnostic summary:
  `cgo_harness/harness_out/scoped-first-cuda-n40-20260625/baseline/diagnostic_summary.md`
- Docker wrapper artifact:
  `harness_out/docker/20260625T104610Z-scoped-first-cuda-n40-20260625`
