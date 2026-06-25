# C++ Recovery Admission Witness - 2026-06-25

## Scope

- Worktree: `/home/draco/work/gotreesitter-build-baseline`
- Branch: `repair/java-token-admission-reduced`
- Commit: `8e733826bbebfb6142a20f8cb33c87b6cb038ffb`
- Grammar: `cpp`
- Corpus frame: `/workspace/corpus_sources/cpp/include/fmt/args.h`
- Frame sha256: `ce9705a22f68469891cba708b3c4ae459edb3f41c8a83f891e93013d485190f8`

## Command

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cpp-recovery-admission-witness-20260625 \
  --memory 8g --cpus 4 -- \
  'cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=1 GTS_WRINGER_BASELINE_FRAMES=1 GTS_WRINGER_VARIANT_FRAMES=1 GTS_WRINGER_FIRSTDIFF_FRAMES=1 GTS_WRINGER_VARIANTS="crecovery_all crecovery_off" GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_TIMEOUT=60 GTS_WRINGER_DIAG_TIMEOUT=60 bash cgo_harness/docker/run_grammar_integrity_wringer.sh cpp cgo_harness/harness_out/cpp-recovery-admission-witness-20260625'
```

Exit status: `0`

Docker metadata:

- `/home/draco/work/gotreesitter-build-baseline/harness_out/docker/20260625T085138Z-cpp-recovery-admission-witness-20260625/metadata.txt`
- `/home/draco/work/gotreesitter-build-baseline/harness_out/docker/20260625T085138Z-cpp-recovery-admission-witness-20260625/inspect.json`
- `/home/draco/work/gotreesitter-build-baseline/harness_out/docker/20260625T085138Z-cpp-recovery-admission-witness-20260625/container.log`

Docker result: `exit_code=0`, `oom_killed=false`, `memory=8g`, `cpus=4`.

## Variant Mapping

`cgo_harness/docker/run_grammar_integrity_wringer.sh` maps:

- `crecovery_all` -> `GOT_C_RECOVERY=all`
- `crecovery_off` -> `GOT_C_RECOVERY=0`

## Result

`crecovery_all` moved the frontier witness materially versus baseline and `crecovery_off`: it eliminated truncation, changed stop reason from `no_stacks_alive` to `accepted`, reached EOF, expanded token count from `71` to `1081`, moved last token end from `652` to `7192`, changed the root pair from `ERROR->translation_unit` to `translation_unit->translation_unit`, and increased node count from `132/373984` to `3941/373984`. Parity still did not match; the remaining divergence moved from root type mismatch to a child-count difference at `root[6]`.

| Run | Parity | Trunc | Stop | Tokens | Last token end | Root pair | Go span | Nodes | Max stacks | First diff |
| --- | --- | --- | --- | ---: | ---: | --- | --- | --- | ---: | --- |
| baseline | `0/1(0%)` | `1` | `no_stacks_alive` | 71 | 652 | `ERROR->translation_unit` | `0:641` | `132/373984` | 3 | `root`, `type` |
| `crecovery_all` | `0/1(0%)` | `0` | `accepted` | 1081 | 7192 | `translation_unit->translation_unit` | `209:7175` | `3941/373984` | 8 | `root[6]`, `child-count` |
| `crecovery_off` | `0/1(0%)` | `1` | `no_stacks_alive` | 71 | 652 | `ERROR->translation_unit` | `0:641` | `132/373984` | 3 | `root`, `type` |

## Post-Patch Default Validation

After the generalized retry patch, the default C++ run reproduced the prior forced `crecovery_all` recovery shape without setting `GOT_C_RECOVERY=all`.

- Output dir: `/home/draco/work/gotreesitter-build-baseline/cgo_harness/harness_out/cpp-recovery-retry-default-witness-20260625-variant1`
- Docker metadata: `/home/draco/work/gotreesitter-build-baseline/harness_out/docker/20260625T091102Z-cpp-recovery-retry-default-witness-20260625-variant1/metadata.txt`
- Docker result: `exit_code=0`, `oom_killed=false`
- Zero-variant attempt: `/home/draco/work/gotreesitter-build-baseline/harness_out/docker/20260625T091013Z-cpp-recovery-retry-default-witness-20260625/metadata.txt` used `GTS_WRINGER_VARIANT_FRAMES=0` and failed because zero is not a supported selector ordinal: `invalid frame selector ordinal: '0'`. The successful fallback used `GTS_WRINGER_VARIANT_FRAMES=1`.

Default baseline evidence:

| Run | Parity | Trunc | Stop | Tokens | Last token end | Expected EOF | Root pair | Go span | Nodes | Max stacks | First diff |
| --- | --- | --- | --- | ---: | ---: | ---: | --- | --- | --- | ---: | --- |
| post-patch default | `0/1(0%)` | `0` | `accepted` | 1081 | 7192 | 7192 | `translation_unit->translation_unit` | `209:7175` | `3941/373984` | 8 | `root[6]`, `preproc_ifdef` child-count Go 7 vs C 9 |

This moves the C++ witness from `runtime_frontier_stop` to accepted recovery-shape evidence. It does not establish full parity; parity remains `0/1`.

## Artifacts

- Wringer summary: `/home/draco/work/gotreesitter-build-baseline/cgo_harness/harness_out/cpp-recovery-admission-witness-20260625/wringer_summary.md`
- Wringer summary JSON: `/home/draco/work/gotreesitter-build-baseline/cgo_harness/harness_out/cpp-recovery-admission-witness-20260625/wringer_summary.json`
- Frame matrix: `/home/draco/work/gotreesitter-build-baseline/cgo_harness/harness_out/cpp-recovery-admission-witness-20260625/frame_matrix.jsonl`
- Commands log: `/home/draco/work/gotreesitter-build-baseline/cgo_harness/harness_out/cpp-recovery-admission-witness-20260625/commands.log`
- Baseline log: `/home/draco/work/gotreesitter-build-baseline/cgo_harness/harness_out/cpp-recovery-admission-witness-20260625/baseline/measure-cpp-external-frame-0001.log`
- `crecovery_all` log: `/home/draco/work/gotreesitter-build-baseline/cgo_harness/harness_out/cpp-recovery-admission-witness-20260625/variants/crecovery_all/frame-0001.log`
- `crecovery_off` log: `/home/draco/work/gotreesitter-build-baseline/cgo_harness/harness_out/cpp-recovery-admission-witness-20260625/variants/crecovery_off/frame-0001.log`
- First-diff log: `/home/draco/work/gotreesitter-build-baseline/cgo_harness/harness_out/cpp-recovery-admission-witness-20260625/firstdiff/frame-0001.log`
