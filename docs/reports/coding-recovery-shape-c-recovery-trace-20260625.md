# Coding Recovery-Shape C-Recovery Trace - 2026-06-25

Scope: report-only C-recovery trace-window diagnostics for the accepted
recovery-shape witnesses selected by
`docs/reports/coding-recovery-shape-frame-triangulation-20260625.md`. No parser
behavior, grammar policy, or normalizer changes are included.

## Runs

All runs used Docker isolation via `cgo_harness/docker/run_parity_in_docker.sh`
with the external corpus mounted read-only at `/workspace/corpus_sources`.
The focused test was `TestFirstDiffDiag`, so each command parsed one witness and
preserved the existing structural divergence in the same log as the
`C-REC-TRACE` window.

### C++ `args.h`

Command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label c-rec-trace-cpp-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/coding-recovery-shape-c-recovery-trace-20260625 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cpp REPRO_FILE=/workspace/corpus_sources/cpp/include/fmt/args.h REPRO_DIR=/workspace/corpus_sources GOT_PARSE_PROGRESS=1 GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=400:7200 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/coding-recovery-shape-c-recovery-trace-20260625/cpp-args-h-crecovery-all-firstdiff.log"
```

Status: pass, Docker exit `0`, `oom_killed=false`, max RSS `1438296 kB`.

Artifacts:

- `cgo_harness/harness_out/coding-recovery-shape-c-recovery-trace-20260625/cpp-args-h-crecovery-all-firstdiff.log`
- `harness_out/docker/20260625T160825Z-c-rec-trace-cpp-20260625/container.log`
- `harness_out/docker/20260625T160825Z-c-rec-trace-cpp-20260625/metadata.txt`

Trace volume in the bounded window: `764` `C-REC-TRACE` lines, including `96`
`cRecoverToState`, `45` `cCondenseAndResume`, and `176`
`cAbsorbTokenIntoError` lines.

High-signal excerpt:

```text
C-REC-TRACE cCondenseAndResume resume_stack=0 state=1136 byte=7167 tok_sym=11 tok=7169:7175
C-REC-TRACE cHandleError stack=0 state=1136 stack_byte=7167 tok_sym=11 tok=7169:7175
C-REC-TRACE cRecoverStrategy1Election pos=7167 tok_sym=11 tok=7169:7175 member=0 entry_state=72 entry_depth=2 entry_pos=7148 recover_depth=2 current_has_action=false better_version_reject=false
C-REC-TRACE cRecoverStrategy1Election pos=7167 tok_sym=11 tok=7169:7175 member=0 entry_state=72 entry_depth=3 entry_pos=7147 recover_depth=3 current_has_action=false better_version_reject=false
C-REC-TRACE cRecoverStrategy1Election pos=7167 tok_sym=11 tok=7169:7175 member=0 entry_state=50 entry_depth=12 entry_pos=432 recover_depth=12 current_has_action=true better_version_reject=false
C-REC-TRACE cRecoverToState goal_state=50 depth=12 raw_span=433:7167 child_count=18 trailing_extras=0
C-REC-TRACE cAbsorbTokenIntoError tok_sym=11 tok=7169:7175 old_open=none new_open=7169:7175 visible_leaf=true
```

First diff remains the accepted recovery-shape witness:

- Go `preproc_ifdef [209:7175]` has child count `7`; C has child count `9`.
- Go child `5` is `ERROR [433:7167]`.
- C child `5` is `function_definition [433:7147]`.
- C then materializes `expression_statement [7147:7148]` and
  `expression_statement [7150:7167]` before `#endif`.

Classification: the trace confirms a broad recover-to-state materialization at
the same boundary as the extra Go `ERROR`. The elected recovery path at byte
`7167` walks through candidate stack entries, selects `goal_state=50`, rebuilds
`raw_span=433:7167`, and then resumes/absorbs the following preprocessor token.

### CUDA `UnifiedMemoryStreams.cu`

Explicit C-recovery command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label c-rec-trace-cuda-all-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/coding-recovery-shape-c-recovery-trace-20260625 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_PARSE_PROGRESS=1 GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8480:8810 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/coding-recovery-shape-c-recovery-trace-20260625/cuda-unified-memory-streams-crecovery-all-firstdiff.log"
```

Status: pass, Docker exit `0`, `oom_killed=false`, max RSS `1444364 kB`.

Artifacts:

- `cgo_harness/harness_out/coding-recovery-shape-c-recovery-trace-20260625/cuda-unified-memory-streams-crecovery-all-firstdiff.log`
- `harness_out/docker/20260625T160857Z-c-rec-trace-cuda-all-20260625/container.log`
- `harness_out/docker/20260625T160857Z-c-rec-trace-cuda-all-20260625/metadata.txt`

Trace volume in the bounded window: `92` `C-REC-TRACE` lines, including `7`
`cRecoverToState`, `7` `cCondenseAndResume`, and `3`
`cAbsorbTokenIntoError` lines.

High-signal excerpt:

```text
C-REC-TRACE cCondenseAndResume resume_stack=0 state=8235 byte=8585 tok_sym=70 tok=8586:8587
C-REC-TRACE cHandleError stack=0 state=8235 stack_byte=8585 tok_sym=70 tok=8586:8587
C-REC-TRACE cRecoverStrategy1Election pos=8585 tok_sym=70 tok=8586:8587 member=0 entry_state=5483 entry_depth=2 entry_pos=8536 recover_depth=2 current_has_action=false better_version_reject=false
C-REC-TRACE cRecoverStrategy1Election pos=8585 tok_sym=70 tok=8586:8587 member=0 entry_state=811 entry_depth=3 entry_pos=8531 recover_depth=3 current_has_action=false better_version_reject=false
C-REC-TRACE cRecoverStrategy1Election pos=8585 tok_sym=70 tok=8586:8587 member=0 entry_state=984 entry_depth=4 entry_pos=8518 recover_depth=4 current_has_action=false better_version_reject=false
C-REC-TRACE cRecoverStrategy1Election pos=8585 tok_sym=70 tok=8586:8587 member=0 entry_state=85 entry_depth=5 entry_pos=8509 recover_depth=5 current_has_action=true better_version_reject=false
C-REC-TRACE cRecoverToState goal_state=85 depth=5 raw_span=8510:8585 child_count=4 trailing_extras=0
C-REC-TRACE cAbsorbTokenIntoError tok_sym=70 tok=8586:8587 old_open=none new_open=8586:8587 visible_leaf=true
```

First diff remains the accepted recovery-shape witness:

- Go `translation_unit [0:12254]` has child count `26`; C has child count `25`.
- Go child `23` is `ERROR [8510:8585]`.
- Go child `24` is `compound_statement [8586:8797]`.
- C child `23` is `template_declaration [8510:8797]`.

Classification: the trace confirms a narrow recover-to-state materialization at
the extra Go `ERROR` header. The elected recovery path at byte `8585` selects
`goal_state=85`, rebuilds `raw_span=8510:8585`, then resumes at the opening
compound token. C keeps the same byte range plus body as one
`template_declaration`, while Go materializes the header as `ERROR` and leaves
the body as a separate `compound_statement`.

### CUDA Default Gate Check

Default-gate command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label c-rec-trace-cuda-default-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/coding-recovery-shape-c-recovery-trace-20260625 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_PARSE_PROGRESS=1 GOT_C_RECOVERY_TRACE_WINDOW=8480:8810 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/coding-recovery-shape-c-recovery-trace-20260625/cuda-unified-memory-streams-default-firstdiff.log"
```

Status: pass, Docker exit `0`, `oom_killed=false`, max RSS `1243152 kB`.

Artifacts:

- `cgo_harness/harness_out/coding-recovery-shape-c-recovery-trace-20260625/cuda-unified-memory-streams-default-firstdiff.log`
- `harness_out/docker/20260625T160916Z-c-rec-trace-cuda-default-20260625/container.log`
- `harness_out/docker/20260625T160916Z-c-rec-trace-cuda-default-20260625/metadata.txt`

The default-gate run did not change the witness classification in this worktree.
It emitted the same target-window recovery spine (`goal_state=85`,
`raw_span=8510:8585`) and the same first-diff shape. The explicit
`GOT_C_RECOVERY=all` run emitted more repeated trace instances of the same
spine, but no target-shape difference was observed.

## Answer

C++ and CUDA share the relevant mechanics for this proof slice. In both
witnesses, the accepted residual is produced through the same generalized
C-recovery phases:

- an error point reaches `cCondenseAndResume` / `cHandleError` at the end of
  the extra Go error region;
- recovery election walks stack entries until a candidate with
  `current_has_action=true` is found;
- `cRecoverToState` materializes a byte span matching the extra Go `ERROR`
  boundary;
- parsing resumes by consuming or redispatching the next token outside that
  materialized region.

The trace therefore does not falsify the shared proof. It narrows it: the
shared mechanism is recover-to-state plus resume after a materialized error
span, but the witness-specific spans differ. C++ proves the broad-body case
(`433:7167` inside a preprocessor parent). CUDA proves the template-header case
(`8510:8585` before a separately materialized compound body). Any next proof
step should stay at the generalized recovery materialization/resume boundary,
not at grammar-specific normalization.

## Caveats

- `TestFirstDiffDiag` is diagnostic evidence, not a full corpus gate.
- C++ used `GOT_C_RECOVERY=all` so the trace definitely exercised the
  C-recovery path; no C++ default-gate comparison was run.
- GLSL was intentionally skipped as secondary/noisier per task scope.
- The C++ window `400:7200` is broad and produces many trace lines; the cited
  excerpt is the high-signal terminal recovery at the first extra Go `ERROR`
  boundary.

## Validation

Commands run for this report:

```sh
git status --short --ignored=matching
rg -n "C-REC-TRACE|go stopReason|FIRST-DIFF|go.child\\[5\\]|c.child\\[5\\]|c.child\\[6\\]|c.child\\[7\\]|go.child\\[23\\]|go.child\\[24\\]|c.child\\[23\\]|Maximum resident|Exit status" cgo_harness/harness_out/coding-recovery-shape-c-recovery-trace-20260625/*.log
for f in cgo_harness/harness_out/coding-recovery-shape-c-recovery-trace-20260625/*.log; do printf '%s\n' "$f"; rg -c '^C-REC-TRACE' "$f"; rg -c 'cRecoverToState' "$f"; rg -c 'cCondenseAndResume' "$f"; rg -c 'cAbsorbTokenIntoError' "$f"; done
```

Final validation after writing the report:

```sh
git diff --check
git diff --cached --check
```
