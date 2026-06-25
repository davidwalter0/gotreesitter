# Post-Repair C-Recovery Controls N=40 Recertification - 2026-06-25

Scope: correctness-only recertification of the accepted/non-truncated C-recovery coding controls after current repair work on HEAD `df0eb99d`.

User constraint: no per-grammar normalizers. This report records evidence only and does not edit parser source.

Reference: `docs/reports/post-retry-coding-controls-n40-20260625.md`, which reported `cpp` 10/40 parity, `cuda` 21/40 parity, and `glsl` 14/40 parity, all accepted/non-truncated with no terminal failures.

Run shape:

```sh
GTS_CORPUS_DIR=/workspace/corpus_sources \
GTS_WRINGER_N=40 \
GTS_WRINGER_STAGES="baseline,summary" \
GTS_WRINGER_PARSE_PROGRESS=1 \
GTS_WRINGER_TIMEOUT=60 \
bash cgo_harness/docker/run_grammar_integrity_wringer.sh <grammar> \
  cgo_harness/harness_out/post-repair-c-recovery-controls-n40-20260625/<grammar>
```

Each grammar was run separately in Docker through `cgo_harness/docker/run_parity_in_docker.sh` with repo root `/home/draco/work/gotreesitter-build-baseline`, memory `8g`, cpus `4`, and corpus mount `/home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro`.

## Results

| Grammar | Docker exit | OOM killed | Parity | Prior parity | Movement | Diverge | Trunc | ErrTree | Panics | Terminal failures | Family counts |
| --- | ---: | --- | ---: | ---: | --- | ---: | ---: | ---: | ---: | ---: | --- |
| `cpp` | 0 | false | 13/40 | 10/40 | improved | 27 | 0 | 24 | 0 | 0 | clean=13, recovery_error_shape=17, version_or_corpus=10 |
| `cuda` | 0 | false | 23/40 | 21/40 | improved | 17 | 0 | 7 | 0 | 0 | clean=23, recovery_error_shape=5, version_or_corpus=12 |
| `glsl` | 0 | false | 22/40 | 14/40 | improved | 18 | 0 | 27 | 0 | 0 | clean=22, recovery_error_shape=8, version_or_corpus=10 |

All three controls remain accepted and non-truncated: `stop reasons=accepted`, `frontierStops=0`, `trunc=0`, `panics=0`, and no terminal timeout/fail bucket.

No performance conclusion is drawn from this correctness recertification. Diagnostic ratio fields remain harness context only.

## Diagnostic Buckets

| Grammar | runtime_frontier_stop | recovery_error_cost | accepted_shape_materialization | accepted_divergence_cost | scanner_token_accounting | unclear_needs_diagnostic |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `cpp` | 0 | 1 | 0 | 1 | 0 | 0 |
| `cuda` | 0 | 1 | 0 | 1 | 0 | 0 |
| `glsl` | 0 | 1 | 0 | 1 | 0 | 0 |

## Terminal Status

- `cpp`: Docker completed with `exit_code: 0`, `oom_killed: false`; baseline rc 0, infra status 0.
- `cuda`: Docker completed with `exit_code: 0`, `oom_killed: false`; baseline rc 0, infra status 0. The wringer printed `wait_for: No record of process 2825` after diagnostic summary publication, but still emitted the final summary and successful infra status.
- `glsl`: Docker completed with `exit_code: 0`, `oom_killed: false`; baseline rc 0, infra status 0.

No frame timeouts, panics, truncation, or terminal failures were reported in the generated summaries.

## Comparison

| Grammar | Prior parity | Current parity | Parity delta | Current classification | Moved out of residual/Tier-IV? |
| --- | ---: | ---: | ---: | --- | --- |
| `cpp` | 10/40 | 13/40 | +3 | Tier IV recovery residual | no |
| `cuda` | 21/40 | 23/40 | +2 | Tier IV recovery residual | no |
| `glsl` | 14/40 | 22/40 | +8 | Tier IV recovery residual | no |

The generalized C-recovery repair reduced residual counts for all three true-coding controls in this N=40 sample, but did not fully clear any of them from residual/Tier-IV status.

## Artifacts

- `cpp`: `cgo_harness/harness_out/post-repair-c-recovery-controls-n40-20260625/cpp`
- `cuda`: `cgo_harness/harness_out/post-repair-c-recovery-controls-n40-20260625/cuda`
- `glsl`: `cgo_harness/harness_out/post-repair-c-recovery-controls-n40-20260625/glsl`
- Docker wrapper artifacts:
  - `harness_out/docker/20260625T210118Z-cpp-post-repair-c-recovery-controls-n40-20260625`
  - `harness_out/docker/20260625T210605Z-cuda-post-repair-c-recovery-controls-n40-20260625`
  - `harness_out/docker/20260625T211016Z-glsl-post-repair-c-recovery-controls-n40-20260625`

## Commands

```sh
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --memory 8g --cpus 4 --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label cpp-post-repair-c-recovery-controls-n40-20260625 -- "cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=40 GTS_WRINGER_STAGES='baseline,summary' GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_TIMEOUT=60 bash cgo_harness/docker/run_grammar_integrity_wringer.sh cpp cgo_harness/harness_out/post-repair-c-recovery-controls-n40-20260625/cpp"

bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --memory 8g --cpus 4 --no-build --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label cuda-post-repair-c-recovery-controls-n40-20260625 -- "cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=40 GTS_WRINGER_STAGES='baseline,summary' GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_TIMEOUT=60 bash cgo_harness/docker/run_grammar_integrity_wringer.sh cuda cgo_harness/harness_out/post-repair-c-recovery-controls-n40-20260625/cuda"

bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --memory 8g --cpus 4 --no-build --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label glsl-post-repair-c-recovery-controls-n40-20260625 -- "cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=40 GTS_WRINGER_STAGES='baseline,summary' GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_TIMEOUT=60 bash cgo_harness/docker/run_grammar_integrity_wringer.sh glsl cgo_harness/harness_out/post-repair-c-recovery-controls-n40-20260625/glsl"
```
