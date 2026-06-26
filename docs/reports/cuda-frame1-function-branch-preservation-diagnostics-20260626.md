# CUDA frame 1 function branch preservation diagnostic, 2026-06-26

## Scope

Objective: determine whether the CUDA frame 1 function-declarator/body branch
for `UnifiedMemoryStreams.cu` ever exists and is later culled, merged, or killed
before `{` at byte `8586`, or whether the generated actions/conflict handling
never preserves that branch.

Primary frame:

```text
cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu
```

This run used temporary generic parser action tracing keyed by byte window and
probe symbol, then removed the temporary source changes. No parser source fix
was kept. No grammar blob was edited. No host repo-wide test was run.

## Commands

Temporary trace compile check:

```bash
go test . -run '^$'
```

Docker frame 1 firstdiff with action tracing:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame1-action-trace-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_GLR_ACTION_TRACE_WINDOW=8530:8587 GOT_GLR_ACTION_TRACE_PROBE='{' GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Action-table conflict decode for the relevant states:

```bash
GTS_DECODE=1 GTS_DECODE_TARGETS='cuda:3101,8235,85,84,6390,6074,1061' go test . -run '^TestDecodeConflictStates$' -count=1 -v
```

Clean-source focused compile check after removing temporary instrumentation:

```bash
go test . -run '^$'
```

Clean-source CUDA frame 1 firstdiff replay:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame1-clean-firstdiff-report-20260626 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

CUDA frame 39 control after removing temporary instrumentation:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame39-control-report-20260626 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/simpleTexture3D/simpleTexture3D_kernel.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Primary trace artifact:

```text
harness_out/docker/20260626T000614Z-cuda-frame1-action-trace-20260625/container.log
```

Clean frame 1 replay artifact:

```text
harness_out/docker/20260626T001127Z-cuda-frame1-clean-firstdiff-report-20260626/container.log
```

## Evidence

At the function name before the parameter list, the parser did have declarator
state, but it was not a forked function-body-preserving conflict. State `6390`
on `(` reduced the identifier to `_declarator`; then state `6074` on the same
`(` shifted:

```text
lookup stack=0 state=6390 tok_sym=20 tok_name="(" tok=8553:8554 action_count=1
action[0] type=1 symbol=284 symbol_name="_declarator" child_count=1
lookup stack=0 state=6074 tok_sym=20 tok_name="(" tok=8553:8554 action_count=1
action[0] type=0 state=130
```

The inner template/type region did create generic ambiguity. For example, state
`1061` on `<` had two actions:

```text
lookup stack=0 state=1061 tok_sym=39 tok_name="<" tok=8565:8566 action_count=2
```

Those forks did not leave a branch that can take the body. At the close paren,
all live stacks were in state `3101` with top symbol `")"` and byte `8585`.
On lookahead `{`, state `3101` had exactly one action: reduce the parameter tail
as `argument_list`.

```text
stack[0] state=3101 dead=false shifted=false byte=8585 top_symbol=")"
lookup stack=0 state=3101 tok_sym=70 tok_name="{" tok=8586:8587 action_count=1
action[0] type=1 symbol=373 symbol_name="argument_list" child_count=3
```

After that reduce, every live stack was state `8235` with top symbol
`argument_list [8553:8585]`. State `8235` had no action for `{`:

```text
stack[0] state=8235 dead=false shifted=false byte=8585 top_symbol="argument_list" top_span=8553:8585 probe_actions=0
stack[1] state=8235 dead=false shifted=false byte=8585 top_symbol="argument_list" top_span=8553:8585 probe_actions=0
stack[2] state=8235 dead=false shifted=false byte=8585 top_symbol="argument_list" top_span=8553:8585 probe_actions=0
lookup stack=0 state=8235 tok_sym=70 tok_name="{" tok=8586:8587 action_count=0
lookup stack=1 state=8235 tok_sym=70 tok_name="{" tok=8586:8587 action_count=0
lookup stack=2 state=8235 tok_sym=70 tok_name="{" tok=8586:8587 action_count=0
```

The only `{` shift observed in the window occurred after C-recovery had wrapped
the header as an error and returned to a top-level translation-unit state:

```text
stack[0] state=85 dead=false shifted=false byte=8585 top_symbol="?" top_span=8510:8585 probe_actions=2
lookup stack=0 state=85 tok_sym=70 tok_name="{" tok=8586:8587 action_count=2
```

That path produced a separate `compound_statement [8586:8797]`, not a
function-body continuation under the template declaration:

```text
stack[0] state=85 dead=false shifted=false byte=8797 top_symbol="compound_statement" top_span=8586:8797
```

The final firstdiff shape was unchanged:

```text
go stopReason=accepted truncated=false rootHasError=true cRootHasError=false
go.child[23]: type="ERROR" [8510:8585]
go.child[24]: type="compound_statement" [8586:8797]
c.child[23]:  kind="template_declaration" [8510:8797]
```

The focused table decode supports the runtime observation:

```text
--- state 3101 ---
  (no multi-action conflicts at this state)
--- state 8235 ---
  (no multi-action conflicts at this state)
--- state 85 ---
  lookahead "{" (sym 70): 2 actions
      REDUCE translation_unit_repeat1 (childCount=2)
      SHIFT -> state 55 REPETITION
```

Clean-source validation remained green after the temporary trace code was
removed:

```text
ok   github.com/odvcencio/gotreesitter  0.003s [no tests to run]
```

The clean frame 1 replay reproduced the same root split:

```text
go stopReason=accepted truncated=false rootHasError=true cRootHasError=false
FIRST-DIFF @root
go.child[23]: type="ERROR" [8510:8585]
go.child[24]: type="compound_statement" [8586:8797]
c.child[23]:  kind="template_declaration" [8510:8797]
```

The frame 39 control remained accepted, non-truncated, and no-error, retaining
only the known `sizeof_expression` divergence:

```text
go stopReason=accepted truncated=false rootHasError=false cRootHasError=false
FIRST-DIFF @root[1][16][2][2][16][0][1][5]
go sizeof_expression [4132:4155] child-count=4
c  sizeof_expression [4132:4155] child-count=2
```

## Conclusion

The function-declarator/body branch was not present as a live branch at byte
`8586`, and the trace did not show it being dropped by post-reduce fork merge,
stack equivalence, same-key merge caps, or global culling immediately before the
failure. The branch that could shift `{` appeared only after recovery
resynchronized to top-level state `85`, so it was not the missing generalized
function-body branch.

For this witness, the evidence points earlier than runtime GLR stack culling or
merge selection: the generated actions/conflict path reaching state `3101` has
only the `argument_list` reduce on `{`, and the subsequent state `8235` has no
body action. A generalized parser machinery fix is not justified from this
trace alone.

This supersedes the earlier recovery-summary/election target for CUDA frame 1.
Recovery is still where the visible error is materialized, but the full-span
`template_declaration [8510:8797]` candidate is absent before recovery begins.

## Next target

Inspect grammargen/action generation for the C++ ambiguous declarator path
around states `6074`, `3101`, and `8235`, comparing against upstream
tree-sitter-c action tables for the analogous CUDA/C++ state if possible.
Specifically, determine why the route that parses
`initialise_tasks(std::vector<Task<T>> &TaskList)` as a function declarator does
not survive to a state with a `{` shift.

Falsification criteria for this next target:

- A new trace shows an unrecovered function-declarator/body stack with a direct
  `{` shift before byte `8586`, and that stack is later removed by
  `mergeStacksWithScratch`, post-reduce fork merge, same-key cap, or global
  cull. That would move the target back to runtime branch preservation.
- A grammargen/action fix is justified only if it is generalized, preserves the
  missing declarator/body alternative without CUDA language-name policy or
  per-grammar normalization, and moves frame 1 away from root-level
  recovery/error shape while keeping frame 39 accepted/no-error with only the
  known `sizeof_expression` divergence.
