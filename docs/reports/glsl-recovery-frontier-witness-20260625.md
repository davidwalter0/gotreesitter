# GLSL recovery frontier witness - 2026-06-25

## Run

Worktree: `/home/draco/work/gotreesitter-build-baseline`

Branch/HEAD: `repair/java-token-admission-reduced` at `bf16c9999c8902dddec74259a59cef9cc7a1dfcc`

Command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label glsl-recovery-frontier-witness-20260625 \
  --memory 8g --cpus 4 -- \
  'cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=1 GTS_WRINGER_BASELINE_FRAMES=1 GTS_WRINGER_VARIANT_FRAMES=1 GTS_WRINGER_FIRSTDIFF_FRAMES=1 GTS_WRINGER_VARIANTS="crecovery_off crecovery_all stack48 merge24 node3" GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_TIMEOUT=60 GTS_WRINGER_DIAG_TIMEOUT=60 bash cgo_harness/docker/run_grammar_integrity_wringer.sh glsl cgo_harness/harness_out/glsl-recovery-frontier-witness-20260625'
```

Docker exit status: `0`

OOM killed: `false`

Harness artifacts:

- `cgo_harness/harness_out/glsl-recovery-frontier-witness-20260625/wringer_summary.json`
- `cgo_harness/harness_out/glsl-recovery-frontier-witness-20260625/frame_matrix.jsonl`
- `harness_out/docker/20260625T092856Z-glsl-recovery-frontier-witness-20260625/container.log`

## Baseline versus old smoke

File: `/workspace/corpus_sources/glsl/Test/100.frag`

| Run | Parity | Trunc | Stop | Tokens | Last/EOF | Root pair | Go/C spans | Errors Go/C | Missing Go/C | Nodes | Max stacks | First diff |
| --- | --- | --- | --- | ---: | --- | --- | --- | ---: | ---: | --- | ---: | --- |
| old `current-206-smoke-20260625` | `0/1(0%)` | `true` | `no_stacks_alive` | 320 | `1823/5477` | `ERROR->translation_unit` | `0:1821/0:5477` | `1/5` | `0/2` | `26642/300000` | 72 | `root`, `type` |
| post-patch baseline | `0/1(0%)` | `false` | `accepted` | 1014 | `5477/5477` | `translation_unit->translation_unit` | `0:5477/0:5477` | `6/5` | `3/2` | `37610/300000` | 18 | `root`, `child-count` |

The post-patch baseline moved this frame out of the runtime frontier: it reaches EOF, accepts, and has no runtime-frontier-stop bucket evidence. It remains non-parity due to accepted shape/error divergence at the root.

## Variant outcomes

| Variant | Parity | Trunc | Stop | Tokens | Last/EOF | Root pair | Go/C spans | Errors Go/C | Missing Go/C | Nodes | Max stacks | First diff |
| --- | --- | --- | --- | ---: | --- | --- | --- | ---: | ---: | --- | ---: | --- |
| `crecovery_off` | `0/1(0%)` | `false` | `accepted` | 1014 | `5477/5477` | `translation_unit->translation_unit` | `0:5477/0:5477` | `6/5` | `3/2` | `37610/300000` | 18 | `root`, `child-count` |
| `crecovery_all` | `0/1(0%)` | `false` | `accepted` | 1014 | `5477/5477` | `translation_unit->translation_unit` | `0:5477/0:5477` | `6/5` | `3/2` | `35316/300000` | 36 | `root`, `child-count` |
| `stack48` | `0/1(0%)` | `false` | `accepted` | 1014 | `5477/5477` | `translation_unit->translation_unit` | `0:5477/0:5477` | `6/5` | `3/2` | `37610/300000` | 18 | `root`, `child-count` |
| `merge24` | `0/1(0%)` | `false` | `accepted` | 1014 | `5477/5477` | `translation_unit->translation_unit` | `0:5477/0:5477` | `6/5` | `3/2` | `94988/300000` | 72 | `root`, `child-count` |
| `node3` | `0/1(0%)` | `false` | `accepted` | 1014 | `5477/5477` | `translation_unit->translation_unit` | `0:5477/0:5477` | `6/5` | `3/2` | `37610/900000` | 18 | `root`, `child-count` |

All variants migrated out of runtime-frontier behavior for this frame. None reached parity.

`crecovery_all` did not reduce error counts relative to baseline or `crecovery_off`: Go/C errors stayed `6/5`, missing counts stayed `3/2`, and the first diff stayed `root` child-count. It did reduce node count from `37610/300000` to `35316/300000`, but did not improve the parity/error shape.

## Classification

The generalized conditional C-recovery retry moves this GLSL witness out of the true-coding runtime frontier. The remaining GLSL issue is a separate accepted tree-shape/recovery-materialization frontier: full EOF coverage and accepted parse, but root child-count divergence and one extra Go error/missing node versus C.
