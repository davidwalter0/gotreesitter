# Coding Recovery-Shape C-Recovery Reduce Window Trace - 2026-06-25

Scope: generalized, env-gated diagnostics for accepted post-retry recovery
shape divergence in true-coding grammars. No grammar-specific normalizer,
grammar-name branch, or result-shape policy was added.

The diagnostic reuses `GOT_C_RECOVERY_TRACE_WINDOW` and extends `C-REC-TRACE`
with:

- parse-table action lookup details for stacks/tokens in the byte window;
- reduce symbol names, child counts, production IDs, dynamic precedence;
- reduce window `start`, `reducedEnd`, `actualEnd`, raw byte span, top/pre-goto/target states;
- counts of extra `ERROR` nodes inside the reduced window versus left trailing.

## CUDA Primary Witness

Command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label c-rec-reduce-window-cuda-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/coding-recovery-shape-crecovery-reduce-window-20260625 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_PARSE_PROGRESS=1 GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8480:8810 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/coding-recovery-shape-crecovery-reduce-window-20260625/cuda-unified-memory-streams-crecovery-all-reduce-window.log"
```

Status: Docker exit `0`, `oom_killed=false`, max RSS `1488252 kB`.

Artifacts:

- `cgo_harness/harness_out/coding-recovery-shape-crecovery-reduce-window-20260625/cuda-unified-memory-streams-crecovery-all-reduce-window.log`
- `harness_out/docker/20260625T162630Z-c-rec-reduce-window-cuda-20260625/container.log`
- `harness_out/docker/20260625T162630Z-c-rec-reduce-window-cuda-20260625/metadata.txt`

High-signal trace:

```text
C-REC-TRACE actionLookup stack=0 state=8235 tok_sym=70 tok_name="{" tok=8586:8587 action_count=0
C-REC-TRACE cHandleError stack=0 state=8235 stack_byte=8585 tok_sym=70 tok=8586:8587
C-REC-TRACE reduceWindow phase=gss tok_sym=70 tok_name="{" tok=8586:8587 reduce_symbol=399 reduce_symbol_name="template_declaration" child_count=3 production_id=57 ... raw_span=8510:8585 top_state=85 ... target_state=85 reduced_error_extras=0 trailing_error_extras=0
C-REC-TRACE cRecoverStrategy1Election pos=8585 tok_sym=70 tok=8586:8587 ... entry_state=85 entry_depth=5 entry_pos=8509 recover_depth=5 current_has_action=true better_version_reject=false
C-REC-TRACE cRecoverToState goal_state=85 depth=5 raw_span=8510:8585 child_count=4 trailing_extras=0 pushed_error=true top_state=85 ... top_pre_goto=85 top_parse_state=85
C-REC-TRACE actionLookup stack=0 state=85 tok_sym=70 tok_name="{" tok=8586:8587 action_count=2
C-REC-TRACE reduceWindow phase=gss tok_sym=70 tok_name="{" tok=8586:8587 reduce_symbol=507 reduce_symbol_name="translation_unit_repeat1" child_count=2 ... raw_span=4353:8585 ... reduced_error_extras=0 trailing_error_extras=1
C-REC-TRACE reduceWindow phase=gss tok_sym=106 tok_name="primitive_type" tok=8799:8802 reduce_symbol=507 reduce_symbol_name="translation_unit_repeat1" child_count=2 ... raw_span=4686:8797 ... reduced_error_extras=1 trailing_error_extras=0
```

First diff remains:

- Go `translation_unit` child `23`: `ERROR [8510:8585]`.
- Go child `24`: `compound_statement [8586:8797]`.
- C child `23`: `template_declaration [8510:8797]`.

Interpretation: at the recovery point, generalized potential reductions can
build `template_declaration [8510:8585]`, but the resumed recovery path chooses
`cRecoverToState` at `goal_state=85` and materializes the same header span as an
extra `ERROR`. The immediately following translation-unit reduction leaves that
extra ERROR trailing; after the compound body is parsed, a later
translation-unit reduction counts the recovered extra ERROR into the reduced
window. The missing match is not a terminal/frontier failure; it is the recovery
version/materialization boundary around a same-position viable reduction.

## C++ Secondary Witness

Command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label c-rec-reduce-window-cpp-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/coding-recovery-shape-crecovery-reduce-window-20260625 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cpp REPRO_FILE=/workspace/corpus_sources/cpp/include/fmt/args.h REPRO_DIR=/workspace/corpus_sources GOT_PARSE_PROGRESS=1 GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=400:7200 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v > harness_out/coding-recovery-shape-crecovery-reduce-window-20260625/cpp-args-h-crecovery-all-reduce-window.log 2>&1"
```

Status: Docker exit `0`, `oom_killed=false`, max RSS `1439072 kB`.

Artifacts:

- `cgo_harness/harness_out/coding-recovery-shape-crecovery-reduce-window-20260625/cpp-args-h-crecovery-all-reduce-window.log`
- `harness_out/docker/20260625T162816Z-c-rec-reduce-window-cpp-20260625/container.log`
- `harness_out/docker/20260625T162816Z-c-rec-reduce-window-cpp-20260625/metadata.txt`

High-signal trace:

```text
C-REC-TRACE actionLookup stack=0 state=1136 tok_sym=11 tok_name="#endif" tok=7169:7175 action_count=0
C-REC-TRACE cHandleError stack=0 state=1136 stack_byte=7167 tok_sym=11 tok=7169:7175
C-REC-TRACE cRecoverStrategy1Election pos=7167 tok_sym=11 tok=7169:7175 ... entry_state=50 entry_depth=12 entry_pos=432 recover_depth=12 current_has_action=true better_version_reject=false
C-REC-TRACE cRecoverToState goal_state=50 depth=12 raw_span=433:7167 child_count=18 trailing_extras=0 pushed_error=true top_state=50 ... top_pre_goto=50 top_parse_state=50
C-REC-TRACE actionLookup stack=0 state=50 tok_sym=11 tok_name="#endif" tok=7169:7175 action_count=1
C-REC-TRACE reduceWindow phase=gss tok_sym=11 tok_name="#endif" tok=7169:7175 reduce_symbol=520 reduce_symbol_name="translation_unit_repeat1" child_count=2 ... raw_span=250:7167 ... reduced_error_extras=0 trailing_error_extras=1
C-REC-TRACE reduceWindow phase=gss tok_sym=0 tok_name="end" tok=7192:7192 reduce_symbol=234 reduce_symbol_name="preproc_ifdef" child_count=4 ... raw_span=209:7191 ... reduced_error_extras=1 trailing_error_extras=0
```

First diff remains:

- Go `preproc_ifdef` child `5`: `ERROR [433:7167]`.
- C child `5`: `function_definition [433:7147]`, followed by two expression statements before `#endif`.

Interpretation: C++ confirms the same generalized mechanism. The recovered
extra ERROR is first left trailing by a translation-unit-repeat reduction and
then counted inside the enclosing preprocessor reduction. The C oracle instead
keeps recoverable syntax nodes over the same broad span.

## Root-Cause Conclusion

This slice does not justify a behavior fix yet. The divergence is now narrowed
to generalized C-recovery version/materialization semantics:

- `cDoAllPotentialReductions` can produce syntactic reductions at the failure
  byte (`template_declaration` in CUDA; many expression/function reductions in
  C++);
- `cHandleError` then pushes the C error discontinuity onto every potential
  reduction result and records summaries;
- `cRecoverStrategy1Election` skips same-position summary entries and recovers
  to an earlier state with an action on the lookahead;
- `cRecoverToState` materializes the popped span as an extra `ERROR`;
- subsequent reductions decide whether that extra ERROR is trailing or counted
  into a parent, preserving acceptance but diverging from the C oracle shape.

A safe generalized fix likely needs to examine C's multi-version stack
semantics at this handoff: whether same-position potential-reduction versions
should be allowed to survive and dispatch the lookahead before
`recover_to_state`, or whether the Go linear-stack approximation loses a C
merged-link path that keeps syntax rather than materializing an ERROR. Changing
`cRecoverToState` itself to avoid ERROR materialization would be speculative and
would risk breaking grammars already relying on faithful C recovery.

## Validation

Focused host unit test:

```sh
go test . -run '^TestParseCRecoveryTraceWindow$' -count=1
```

Result: `ok github.com/odvcencio/gotreesitter 0.003s`.

Diff checks should be run before committing:

```sh
git diff --check
git diff --cached --check
```
