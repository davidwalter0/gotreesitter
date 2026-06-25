# Post-Retry Coding Controls N=40 Classification Sweep - 2026-06-25

Scope: correctness-only post-patch classification sweep for the three true-coding controls that migrated in N=1 witnesses: `cpp`, `cuda`, and `glsl`.

Context: run after commit `7ec499a7`, with worktree HEAD `cf5a5cf`. This updates classification evidence only. It does not move performance gates or support performance conclusions.

Run shape:

```sh
GTS_CORPUS_DIR=/workspace/corpus_sources \
GTS_WRINGER_N=40 \
GTS_WRINGER_STAGES="baseline,summary" \
GTS_WRINGER_PARSE_PROGRESS=1 \
GTS_WRINGER_TIMEOUT=60 \
bash cgo_harness/docker/run_grammar_integrity_wringer.sh <grammar> \
  cgo_harness/harness_out/post-retry-coding-controls-n40-20260625/<grammar>
```

Each grammar was run separately in Docker with `--memory 8g --cpus 4` and the corpus mount `/home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro`.

## Results

| Grammar | Docker exit | OOM killed | Parity | Diverge | Trunc | ErrTree | Panics | Median ratio | Agg ratio | Family counts |
| --- | ---: | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `cpp` | 0 | false | 10/40 | 30 | 0 | 24 | 0 | 13.01x | 6.22x | clean=10, recovery_error_shape=20, version_or_corpus=10 |
| `cuda` | 0 | false | 21/40 | 19 | 0 | 7 | 0 | 25.74x | 30.60x | clean=21, recovery_error_shape=7, version_or_corpus=12 |
| `glsl` | 0 | false | 14/40 | 26 | 0 | 27 | 0 | 59.77x | 37.26x | clean=14, recovery_error_shape=9, version_or_corpus=17 |

Ratios are retained only as context from the baseline MEASURE-DTIER aggregate. They are not interpreted as performance evidence here.

## Diagnostic Buckets

| Grammar | runtime_frontier/truncation | recovery_error_shape/cost | accepted_shape/materialization | terminal timeout/fail |
| --- | ---: | ---: | ---: | ---: |
| `cpp` | 0 | 1 grammar bucket; 20 wringer frames | 0 materialization buckets; accepted divergence cost bucket=1 | 0 |
| `cuda` | 0 | 1 grammar bucket; 7 wringer frames | 0 materialization buckets; accepted divergence cost bucket=1 | 0 |
| `glsl` | 0 | 1 grammar bucket; 9 wringer frames | 0 materialization buckets; accepted divergence cost bucket=1 | 0 |

All three diagnostic summaries reported `frontierStopCount=0`, `terminalCount=0`, `truncated=false`, and `stopReasons=["accepted"]`.

## Terminal Status

- `cpp`: Docker completed with `exit_code: 0`, `oom_killed: false`. Note: the local zsh wrapper epilogue used readonly variable name `status`, so the outer shell command returned 1 after Docker had already completed successfully. The Docker wrapper output and generated summaries both show baseline/infrastructure status 0.
- `cuda`: Docker completed with `exit_code: 0`, `oom_killed: false`, wrapper status 0.
- `glsl`: Docker completed with `exit_code: 0`, `oom_killed: false`, wrapper status 0.

No frame timeouts, terminal failures, or panics were reported in the generated summaries.

## Artifacts

- `cpp`: `cgo_harness/harness_out/post-retry-coding-controls-n40-20260625/cpp`
- `cuda`: `cgo_harness/harness_out/post-retry-coding-controls-n40-20260625/cuda`
- `glsl`: `cgo_harness/harness_out/post-retry-coding-controls-n40-20260625/glsl`
- Docker wrapper artifacts:
  - `harness_out/docker/20260625T093603Z-cpp-post-retry-n40-20260625`
  - `harness_out/docker/20260625T093953Z-cuda-post-retry-n40-20260625`
  - `harness_out/docker/20260625T094313Z-glsl-post-retry-n40-20260625`
