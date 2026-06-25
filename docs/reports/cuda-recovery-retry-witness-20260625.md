# CUDA recovery retry witness - 2026-06-25

## Scope

Validation-only run for `cuda` on commit `7ec499a7af0f6ba746fe25fa7b6071de8cc26e7f`
(`improve(parser): improve retry path with conditional C-recovery`).

Command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-recovery-retry-witness-20260625 \
  --memory 8g --cpus 4 -- \
  'cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=1 GTS_WRINGER_BASELINE_FRAMES=1 GTS_WRINGER_VARIANT_FRAMES=1 GTS_WRINGER_FIRSTDIFF_FRAMES=1 GTS_WRINGER_VARIANTS="crecovery_off crecovery_all" GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_TIMEOUT=60 GTS_WRINGER_DIAG_TIMEOUT=60 bash cgo_harness/docker/run_grammar_integrity_wringer.sh cuda cgo_harness/harness_out/cuda-recovery-retry-witness-20260625'
```

Exit status: `0`; Docker wrapper reported `oom_killed=false`.

Primary artifacts:

- `cgo_harness/harness_out/cuda-recovery-retry-witness-20260625/wringer_summary.md`
- `cgo_harness/harness_out/cuda-recovery-retry-witness-20260625/frame_matrix.jsonl`
- `cgo_harness/harness_out/cuda-recovery-retry-witness-20260625/baseline/measure-cuda-external-frame-0001.log`
- `cgo_harness/harness_out/cuda-recovery-retry-witness-20260625/variants/crecovery_off/frame-0001.log`
- `cgo_harness/harness_out/cuda-recovery-retry-witness-20260625/variants/crecovery_all/frame-0001.log`
- `cgo_harness/harness_out/cuda-recovery-retry-witness-20260625/firstdiff/frame-0001.log`

Witness file:

`/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu`

## Result

CUDA moved from the prior runtime-frontier stop shape to accepted recovery-shape
on this witness. It is still non-parity because the accepted Go tree has a root
child-count diff and recovery errors.

| frame | parity | trunc | stop | tokens | lastTokenEnd/expectedEOF | root pair | Go span | C span | nodes | maxStacks | first diff |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| baseline | `0/1` | `false` | `accepted` | `1993` | `12254/12254` | `translation_unit->translation_unit` | `0:12254` | `0:12254` | `31390/637208` | `24` | `root child-count`, Go cc `26`, C cc `25` |
| `crecovery_off` | `0/1` | `false` | `accepted` | `1993` | `12254/12254` | `translation_unit->translation_unit` | `0:12254` | `0:12254` | `31390/637208` | `24` | `root child-count`, Go cc `26`, C cc `25` |
| `crecovery_all` | `0/1` | `false` | `accepted` | `1993` | `12254/12254` | `translation_unit->translation_unit` | `0:12254` | `0:12254` | `27438/637208` | `24` | `root child-count`, Go cc `26`, C cc `25` |

Error-shape detail:

- baseline and `crecovery_off`: Go root has errors, `goErrors=17`, `goMissing=11`; C has `0/0`.
- `crecovery_all`: Go root still has errors, but drops to `goErrors=1`, `goMissing=2`; C remains `0/0`.

First-diff replay:

- `FIRST-DIFF @root`
- Go root: `translation_unit [0:12254] cc=26`
- C root: `translation_unit [0:12254] cc=25`
- Extra Go child at index `23`: `ERROR [8510:8585]` for `template <typename T> void initialise_tasks(std::vector<Task<T>> &TaskList)`
- C child at index `23`: `template_declaration [8510:8797]`

## Prior Current-206 Comparison

Local prior smoke artifact:

`cgo_harness/harness_out/tier_scan_parallel/current-206-smoke-20260625/workers/shard-004/measure-cuda-external-frame-0001.log`

Old line for the same file:

- `parityMatch=0/1`, `trunc=1`, `errTree=0`
- `stopReason=no_stacks_alive`
- `tokens=1280`, `lastTokenEnd=8587`, `expectedEOF=12254`
- `nodes=17793/637208`, `maxStacks=24`
- root pair `translation_unit->translation_unit`
- Go span `0:8585`, C span `0:12254`
- first diff `root span`

Compared with the new bounded wringer, the generalized retry admission now
parses through EOF and returns `accepted` without truncation for CUDA, matching
the expected C-recovery movement pattern. The remaining issue is recovery
shape, not a runtime frontier stop.
