# Post-Pop-Slice Repair C-Recovery Controls N=40 Recertification - 2026-06-25

Scope: correctness-only recertification of the true-coding C-recovery controls after pop-slice repair on HEAD `d31f39e3935717cd204bd894b60f84b232ebbdeb`.

Reference: `docs/reports/post-repair-c-recovery-controls-n40-20260625.md`, which reported after `df0eb99d`: `cpp` 13/40, `cuda` 23/40, and `glsl` 22/40, all with `trunc=0` and terminal failures `0`.

No parser source was changed for this recertification. No performance conclusion is drawn.

## Run Shape

Each grammar was run separately in Docker through `cgo_harness/docker/run_parity_in_docker.sh` with repo root `/home/draco/work/gotreesitter-build-baseline`, memory `8g`, cpus `4`, and corpus mount `/home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro`.

```sh
GTS_CORPUS_DIR=/workspace/corpus_sources \
GTS_WRINGER_N=40 \
GTS_WRINGER_STAGES="baseline,summary" \
GTS_WRINGER_PARSE_PROGRESS=1 \
GTS_WRINGER_TIMEOUT=60 \
bash cgo_harness/docker/run_grammar_integrity_wringer.sh <grammar> \
  cgo_harness/harness_out/post-pop-slice-repair-c-recovery-controls-n40-20260625/<grammar>
```

## Results

| Grammar | Docker exit | OOM killed | Prior parity | Current parity | Diverge | Trunc | ErrTree | Panics | Terminal failures | Stop reasons | Families |
| --- | ---: | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- | --- |
| `cpp` | 0 | false | 13/40 | 13/40 | 27 | 0 | 24 | 0 | 0 | accepted | clean=13, recovery_error_shape=17, version_or_corpus=10 |
| `cuda` | 0 | false | 23/40 | 23/40 | 17 | 0 | 7 | 0 | 0 | accepted | clean=23, recovery_error_shape=5, version_or_corpus=12 |
| `glsl` | 0 | false | 22/40 | 22/40 | 18 | 0 | 27 | 0 | 0 | accepted | clean=22, recovery_error_shape=8, version_or_corpus=10 |

All three controls completed with baseline rc `0`, infra status `0`, `frontierStops=0`, `trunc=0`, `panics=0`, and no `terminal_timeout_or_fail` bucket.

## Residual Status

| Grammar | Prior classification | Current classification | Moved out of residual/Tier IV? |
| --- | --- | --- | --- |
| `cpp` | Tier IV recovery residual | Tier IV recovery residual | no |
| `cuda` | Tier IV recovery residual | Tier IV recovery residual | no |
| `glsl` | Tier IV recovery residual | Tier IV recovery residual | no |

The pop-slice repair recertification preserved the prior post-repair N=40 outcomes for these controls. None moved out of residual/Tier IV.

## Artifacts

- `cpp`: `cgo_harness/harness_out/post-pop-slice-repair-c-recovery-controls-n40-20260625/cpp`
- `cuda`: `cgo_harness/harness_out/post-pop-slice-repair-c-recovery-controls-n40-20260625/cuda`
- `glsl`: `cgo_harness/harness_out/post-pop-slice-repair-c-recovery-controls-n40-20260625/glsl`
- Docker wrapper artifacts:
  - `harness_out/docker/20260625T214744Z-cpp-post-pop-slice-repair-c-recovery-controls-n40-20260625`
  - `harness_out/docker/20260625T215238Z-cuda-post-pop-slice-repair-c-recovery-controls-n40-20260625`
  - `harness_out/docker/20260625T215641Z-glsl-post-pop-slice-repair-c-recovery-controls-n40-20260625`

## Commands

```sh
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --memory 8g --cpus 4 --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label cpp-post-pop-slice-repair-c-recovery-controls-n40-20260625 -- "cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=40 GTS_WRINGER_STAGES='baseline,summary' GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_TIMEOUT=60 bash cgo_harness/docker/run_grammar_integrity_wringer.sh cpp cgo_harness/harness_out/post-pop-slice-repair-c-recovery-controls-n40-20260625/cpp"

bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --memory 8g --cpus 4 --no-build --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label cuda-post-pop-slice-repair-c-recovery-controls-n40-20260625 -- "cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=40 GTS_WRINGER_STAGES='baseline,summary' GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_TIMEOUT=60 bash cgo_harness/docker/run_grammar_integrity_wringer.sh cuda cgo_harness/harness_out/post-pop-slice-repair-c-recovery-controls-n40-20260625/cuda"

bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --memory 8g --cpus 4 --no-build --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label glsl-post-pop-slice-repair-c-recovery-controls-n40-20260625 -- "cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=40 GTS_WRINGER_STAGES='baseline,summary' GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_TIMEOUT=60 bash cgo_harness/docker/run_grammar_integrity_wringer.sh glsl cgo_harness/harness_out/post-pop-slice-repair-c-recovery-controls-n40-20260625/glsl"
```
