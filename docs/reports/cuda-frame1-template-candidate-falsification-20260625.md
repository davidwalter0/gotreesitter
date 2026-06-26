# CUDA frame 1 template candidate falsification, 2026-06-25

## Scope

Objective: determine whether CUDA frame 1 produces a C-shaped
`template_declaration [8510:8797]` candidate that later loses recovery
summary/election/materialization, or whether that candidate is never produced.

Primary frame:

```text
cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu
```

Control frame:

```text
cuda/cpp/0_Introduction/simpleTexture3D/simpleTexture3D_kernel.cu
```

All checks were single-file Docker runs through
`cgo_harness/docker/run_parity_in_docker.sh` with the CUDA grammar only. No host
repo-wide test was run. No source fix was made.

## Commands

Initial bounded C-recovery trace around the disputed span:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame1-crecovery-trace-initial-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8480:8810 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Higher stack-budget check with a narrower trace:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame1-stack64-trace-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8530:8590 GOT_GLR_MAX_STACKS=64 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Frame 39 control:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame39-control-firstdiff-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/simpleTexture3D/simpleTexture3D_kernel.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

## Evidence

The frame 1 trace shows the parser reaches `{` at byte `8586` with every live
version already in `state=8235`, and that state has no action for `{`:

```text
C-REC-TRACE actionLookup stack=0 state=8235 tok_sym=70 tok_name="{" tok=8586:8587 action_count=0
C-REC-TRACE actionLookup stack=1 state=8235 tok_sym=70 tok_name="{" tok=8586:8587 action_count=0
C-REC-TRACE actionLookup stack=2 state=8235 tok_sym=70 tok_name="{" tok=8586:8587 action_count=0
```

When C-recovery closes productions at that point, it reduces the signature as a
declaration/header only:

```text
reduce_symbol_name="init_declarator" raw_span=8537:8585
reduce_symbol_name="declaration" raw_span=8532:8585
reduce_symbol_name="template_declaration" raw_span=8510:8585
```

Strategy 1 then recovers to a previous top-level translation-unit state and
wraps the header as an error:

```text
cRecoverStrategy1Election pos=8585 tok_sym=70 tok=8586:8587 entry_state=85 entry_depth=5 entry_pos=8509 current_has_action=true
cRecoverToState goal_state=85 depth=5 raw_span=8510:8585 child_count=4 pushed_error=true
```

After that, the body is reduced independently:

```text
reduce_symbol_name="compound_statement" raw_span=8586:8797
```

No trace line in either frame 1 run produced:

```text
reduce_symbol_name="template_declaration" raw_span=8510:8797
```

Raising `GOT_GLR_MAX_STACKS=64` did not preserve a full-span candidate. The
final frame 1 shape still split the C child:

```text
go.child[23]: type="template_declaration" [8510:8585]
go.child[24]: type="compound_statement" [8586:8797]
c.child[23]:  kind="template_declaration" [8510:8797]
```

The frame 39 control remained accepted, non-truncated, and no-error on both Go
and C:

```text
go stopReason=accepted truncated=false rootHasError=false cRootHasError=false
FIRST-DIFF @root[1][16][2][2][16][0][1][5]
go sizeof_expression [4132:4155] child-count=4
c  sizeof_expression [4132:4155] child-count=2
```

## Conclusion

The C-shaped CUDA frame 1 candidate is not produced and then lost by recovery
summary/election/materialization. It is absent before recovery. By the time
recovery starts at `{`, the surviving branch has already interpreted the
signature tail as an `argument_list`/`init_declarator` path, not the
function-declarator path that can shift a compound body and later reduce to
`template_declaration [8510:8797]`.

This falsifies the selected next-frame target as the primary cause for this
witness. A recovery source fix is not justified from this evidence.

Tier-IV classification does not improve. The witness remains accepted and
non-truncated but has root-level recovery/error shape against a clean C tree.

## Next target

Move the next diagnostic earlier than C-recovery:

- trace GLR branch preservation and merge/equivalence around
  `std::vector<Task<T>> &TaskList)` before the `{` token;
- identify why the function-declarator branch is not live at byte `8585`;
- focus on generic ambiguous declarator/call-argument branch selection,
  reduction ordering, or stack merge equivalence, not recovery summary election.

The control frame 39 remains a separate no-error ambiguity/materialization
target and should continue to be used as a regression guard.
