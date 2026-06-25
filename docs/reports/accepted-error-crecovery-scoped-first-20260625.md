# Accepted-Error C-Recovery Scoped-First Retry - 2026-06-25

## Scope

Bounded generalized parser machinery experiment for accepted, full-EOF,
non-clean retry-admitted trees. No per-grammar normalizers or language-name
policy were added.

The experiment schedules one scoped C-recovery retry before the existing
merge/widening retry sequence when the original full parse is an accepted
error tree admitted by `shouldRetryFullParseWithCRecovery`. The early retry
uses first-pass-equivalent settings:

- `maxStacks = initialMaxStacks`
- `maxMergePerKeyOverride = 0`
- `maxNodes = 0`

Existing later C-recovery retry paths remain intact.

## Code Change

- `parser_retry.go`: `retryFullParse` now runs the scoped C-recovery candidate
  before initial merge/widening retries for `ParseStopAccepted` admitted trees.
  Selection still goes through `preferRetryTree` and the existing
  `replaceBest`/`release` ownership path.
- `parser_api_internal_test.go`: added generic ordering coverage proving the
  accepted-error scoped candidate runs first, runs under C-recovery scope, and
  uses the initial stack budget with no merge or node override.

## Validation

Focused Docker unit run:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh -- \
  "cd /workspace && go test . -run 'TestRetryFullParse(CRecoveryCandidateRestoresParserFlag|AcceptedErrorCRecoveryRunsBeforeWidening)|TestScopedErrorCostCompetitionRestoresAfterPanic' -count=1"
```

Result: pass, wrapper exit `0`, `oom_killed=false`.

CUDA two-frame proof:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label accepted-error-crecovery-scoped-first-cuda-frames \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- \
  "cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=40 GTS_WRINGER_BASELINE_FRAMES=1,3 GTS_WRINGER_FIRSTDIFF_FRAMES=1,3 GTS_WRINGER_STAGES=baseline,firstdiff,summary GTS_WRINGER_PARSE_PROGRESS=1 bash cgo_harness/docker/run_grammar_integrity_wringer.sh cuda cgo_harness/harness_out/accepted-error-crecovery-scoped-first-cuda-frames-20260625"
```

Result: pass, wrapper exit `0`, `oom_killed=false`.

Artifacts:

- `cgo_harness/harness_out/accepted-error-crecovery-scoped-first-cuda-frames-20260625/wringer_summary.md`
- `cgo_harness/harness_out/accepted-error-crecovery-scoped-first-cuda-frames-20260625/frame_matrix.jsonl`
- `cgo_harness/harness_out/accepted-error-crecovery-scoped-first-cuda-frames-20260625/firstdiff/frame-0001.log`
- `cgo_harness/harness_out/accepted-error-crecovery-scoped-first-cuda-frames-20260625/firstdiff/frame-0003.log`

## CUDA Results

Frame 1 moved to the forced C-recovery shape:

- File: `UnifiedMemoryStreams.cu`
- Runtime: `accepted`, `truncated=false`, `tokenEOFEarly=false`,
  `lastTokenEOF=true`, `lastTokenEnd=12254`, `expectedEOF=12254`,
  `maxStacks=24`
- Result: `goErrors=1`, `goMissing=2`, `cErrors=0`, `cMissing=0`
- First diff remains at `root`: Go splits `initialise_tasks` into an `ERROR`
  declaration header plus following `compound_statement`; C keeps one
  `template_declaration`.
- Family remains `recovery_error_shape`.

Frame 3 remained the no-error control:

- File: `clock.cu`
- Runtime: `accepted`, `truncated=false`, `tokenEOFEarly=false`,
  `lastTokenEOF=true`, `maxStacks=18`
- Result: `goErrors=0`, `goMissing=0`, `cErrors=0`, `cMissing=0`
- First diff remains the existing `sizeof_expression` child-count shape.
- Family remains `version_or_corpus`.

## Optional Follow-Ups

C++ frame 1:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label accepted-error-crecovery-scoped-first-cpp-frame1 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- \
  "cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=40 GTS_WRINGER_BASELINE_FRAMES=1 GTS_WRINGER_FIRSTDIFF_FRAMES=1 GTS_WRINGER_STAGES=baseline,firstdiff,summary GTS_WRINGER_PARSE_PROGRESS=1 bash cgo_harness/docker/run_grammar_integrity_wringer.sh cpp cgo_harness/harness_out/accepted-error-crecovery-scoped-first-cpp-frame1-20260625"
```

Result: pass, wrapper exit `0`, `oom_killed=false`. `args.h` remains
non-parity with `goErrors=30`, `goMissing=6`, `cErrors=10`, `cMissing=3`.

GLSL frame 1:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label accepted-error-crecovery-scoped-first-glsl-frame1 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- \
  "cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=40 GTS_WRINGER_BASELINE_FRAMES=1 GTS_WRINGER_FIRSTDIFF_FRAMES=1 GTS_WRINGER_STAGES=baseline,firstdiff,summary GTS_WRINGER_PARSE_PROGRESS=1 bash cgo_harness/docker/run_grammar_integrity_wringer.sh glsl cgo_harness/harness_out/accepted-error-crecovery-scoped-first-glsl-frame1-20260625"
```

Result: pass, wrapper exit `0`, `oom_killed=false`. `100.frag` remains
non-parity with `goErrors=6`, `goMissing=3`, `cErrors=5`, `cMissing=2`.

## Conclusion

The scoped-first candidate caused the expected CUDA frame 1 movement toward the
forced C-recovery shape while preserving the frame 3 no-error control. This
validates the generalized scheduling experiment. Residual CUDA parity work is
still in recovery tree shape/materialization, not admission.
