# Accepted-Error C-Recovery Pressure Floor Negative Investigation

Date: 2026-06-25

## Summary

The accepted-error C-recovery pressure-floor hypothesis was tested and
falsified for the current worktree. No parser code or parser tests from this
experiment were kept.

The trial change would have admitted accepted/full-EOF/non-truncated error
trees using a generic observed stack-pressure floor instead of requiring
saturation of the caller's initial stack cap:

```text
minPositive(initialMaxStacks, maxGLRStacks)
```

That code path was not committed because CUDA frame 1 was already admitted by
the existing retry gate in this branch and still did not move under default
settings. The useful evidence is that forced `GOT_C_RECOVERY=all` improves the
same frame, so the next machinery target is scoped-vs-forced C-recovery
behavior, selection, and materialization rather than a generic pressure-floor
admission rule.

## Focused Unit Validation

Focused unit tests passed for the temporary trial code.

Command:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --label crecovery-pressure-unit-final -- "cd /workspace && go test . -run 'TestCRecoveryRetry|TestShouldRunInitialFullParseMergeRetry|TestRetryFullParseCRecoveryCandidateRestoresParserFlag' -count=1 -v"
```

Result: pass.

Artifact:

```text
harness_out/docker/20260625T102729Z-crecovery-pressure-unit-final
```

Covered cases:

- accepted full-EOF error tree at `maxGLRStacks` is admitted even below a higher initial cap
- accepted full-EOF error tree below `maxGLRStacks` is rejected
- clean accepted tree is rejected
- token-source EOF-early, truncated, missing EOF token, and root-short cases are rejected
- existing no-stacks C-recovery scoped retry behavior still restores the parser flag

These tests validate only the mechanics of the temporary hypothesis. They do
not justify landing it after the CUDA frame proof below failed to show default
parity movement.

## CUDA Frame Evidence

Command:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label accepted-error-crecovery-pressure-floor-cuda-final2 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- \
  "cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=40 GTS_WRINGER_BASELINE_FRAMES=1,3 GTS_WRINGER_FIRSTDIFF_FRAMES=1,3 GTS_WRINGER_STAGES=baseline,firstdiff,summary GTS_WRINGER_PARSE_PROGRESS=1 bash cgo_harness/docker/run_grammar_integrity_wringer.sh cuda cgo_harness/harness_out/accepted-error-crecovery-pressure-floor-cuda-frames-20260625"
```

Result: completed, no OOM.

Artifacts:

```text
cgo_harness/harness_out/accepted-error-crecovery-pressure-floor-cuda-frames-20260625/frame_matrix.jsonl
cgo_harness/harness_out/accepted-error-crecovery-pressure-floor-cuda-frames-20260625/firstdiff/frame-0001.log
cgo_harness/harness_out/accepted-error-crecovery-pressure-floor-cuda-frames-20260625/firstdiff/frame-0003.log
harness_out/docker/20260625T102435Z-accepted-error-crecovery-pressure-floor-cuda-final2
```

Frame outcomes from `frame_matrix.jsonl`:

| frame | file | status | goErrors/goMissing | cErrors/cMissing | maxStacks | result |
|---:|---|---|---:|---:|---:|---|
| 1 | `UnifiedMemoryStreams.cu` | nonmatch | `17/11` | `0/0` | 24 | already admitted, no default frame-matrix movement |
| 3 | `clock.cu` | nonmatch | `0/0` | `0/0` | 18 | clean no-error control stayed clean |

Forced control rerun:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label accepted-error-crecovery-pressure-floor-cuda-forced \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- \
  "cd /workspace && GOT_C_RECOVERY=all GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=40 GTS_WRINGER_BASELINE_FRAMES=1,3 GTS_WRINGER_FIRSTDIFF_FRAMES=1,3 GTS_WRINGER_STAGES=baseline,firstdiff,summary GTS_WRINGER_PARSE_PROGRESS=1 bash cgo_harness/docker/run_grammar_integrity_wringer.sh cuda cgo_harness/harness_out/accepted-error-crecovery-pressure-floor-cuda-forced-frames-20260625"
```

Forced control artifact:

```text
cgo_harness/harness_out/accepted-error-crecovery-pressure-floor-cuda-forced-frames-20260625/frame_matrix.jsonl
```

Forced control outcome:

| frame | file | goErrors/goMissing | maxStacks |
|---:|---|---:|---:|
| 1 | `UnifiedMemoryStreams.cu` | `1/2` | 24 |
| 3 | `clock.cu` | `0/0` | 18 |

## Negative Finding

This worktree did not reproduce the expected "24 vs initial about 36" admission blocker for CUDA frame 1. A temporary guarded trace run showed the parser saw:

```text
stop=accepted maxStacks=24 initial=8 threshold=8 admit=true
```

That means the existing `rt.MaxStacksSeen >= initialMaxStacks` gate was
already open for this frame in this branch: `initial=8`, `maxStacks=24`,
`admit=true`. The pressure-floor helper was mechanically testable, but CUDA
frame 1 is not evidence that admission was the remaining blocker.

The forced control is still useful evidence. With `GOT_C_RECOVERY=all`, CUDA
frame 1 dropped to `1/2` Go errors/missing while frame 3 remained a `0/0`
no-error control. The default path did not achieve that shape even though frame
1 was already admitted, so the remaining delta appears to be in scoped
C-recovery behavior, recovery selection, or materialization.

Because the CUDA proof did not show real frame-matrix movement, C++ and GLSL follow-up frame proofs were not run.

## Commit Decision

No parser code was committed from this validation pass. The focused tests
passed for the trial, but the requested behavior did not produce real CUDA
frame-matrix movement toward the forced C-recovery shape in the current
worktree.
