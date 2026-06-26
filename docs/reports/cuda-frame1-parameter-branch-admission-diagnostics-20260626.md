# CUDA frame 1 parameter branch admission diagnostic, 2026-06-26

## Scope

Objective: trace live stacks and reductions in CUDA frame 1 from byte `8553`
(`(` after `initialise_tasks`) through byte `8585` (`)`) to determine whether
the `parameter_list` / function-declarator branch is ever admitted and then
lost, or never admitted.

Primary frame:

```text
cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu
```

This is report-only. Temporary generic probe code was removed. No parser source
fix, grammar blob change, per-grammar normalizer, CUDA-specific runtime policy,
or language-name parser policy was kept.

## Commands

Initial status and requested context:

```bash
git status --short --branch
git rev-parse HEAD
sed -n '1,220p' AGENTS.md
sed -n '1,220p' docs/reports/cuda-frame1-action-table-fidelity-20260626.md
sed -n '1,220p' docs/reports/cuda-frame1-function-branch-preservation-diagnostics-20260626.md
sed -n '1,220p' docs/reports/cuda-frame1-template-candidate-falsification-20260625.md
```

Clean-source close-paren reduction trace:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame1-closeparen-reduce-existing-trace-20260626 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8553:8586 GOT_GLR_MAX_STACKS=64 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

DFA/token debug around the template close:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame1-dfa-angle-debug-20260626 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8565:8576 REPRO_DEBUG_DFA=1 GOT_GLR_MAX_STACKS=64 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Temporary generic close-angle split probe. The source change was removed after
this run:

```bash
go test . -run 'TestNextDFATokenSplitsCompactCloseAngles|TestNextDFATokenKeepsRightShiftWhenRightShiftHasAction' -count=1

bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame1-generic-angle-split-probe-20260626 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8553:8586 GOT_GLR_MAX_STACKS=64 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Final clean-source validation:

```bash
go test . -run '^$' -count=1

bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame1-parameter-admission-report-20260626 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"

bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame39-parameter-admission-control-20260626 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/simpleTexture3D/simpleTexture3D_kernel.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

## Evidence

The clean trace confirms the observed entry route:

```text
state 6390 on tok_sym=20 "(":
  REDUCE _declarator

state 6074 on tok_sym=20 "(":
  SHIFT -> state 130
```

The shipped table has the expected body-capable route, but the live branch does
not reach it. Read-only `parser.c` inspection showed:

```text
state 6074:
  "(" sym 20 -> SHIFT 130
  parameter_list -> 2927
  _function_declarator_seq -> 6619
  argument_list -> 8235

state 6973 "{" -> SHIFT 55
state 6908 "{" -> SHIFT 53
state 6969 "{" -> SHIFT 43
```

In the requested window, no live stack reduced the close-paren span, or any
interior span, as a function-declarator node. The clean trace reduced only
these symbols:

```text
66 expression
33 translation_unit_repeat1
18 binary_expression
 6 type_specifier
 6 qualified_identifier
 6 pointer_expression
 6 argument_list
 6 _scope_resolution
 6 _declarator
 6 _declaration_specifiers
 2 translation_unit
 2 compound_statement
 1 template_declaration
 1 init_declarator
 1 declaration
```

There were no reductions named:

```text
parameter_list
parameter_declaration
function_declarator
_function_declarator_seq
```

At the close paren, the surviving branch was already the argument-list route:

```text
state 5006 on ")" -> SHIFT 3101

state 3101 on "{":
  REDUCE argument_list child_count=3 raw_span=8553:8585

state 8235 on "{":
  no action
```

The final shape remained the known split:

```text
go stopReason=accepted truncated=false rootHasError=true cRootHasError=false
go.child[23]: type="template_declaration" [8510:8585]
go.child[24]: type="compound_statement" [8586:8797]
c.child[23]:  kind="template_declaration" [8510:8797]
```

The anonymous open-paren symbols are not the immediate mismatch. Upstream has
both `anon_sym_LPAREN` (`5`) and `anon_sym_LPAREN2` (`20`) mapped to `"("`, and
the function-parameter opener in the relevant state is symbol `20`, matching
the observed Go trace:

```text
state 5483:
  anon_sym_LPAREN2(20) -> SHIFT 5551
  sym__declarator -> 6074
  sym_function_declarator -> 6660

state 6390:
  anon_sym_LPAREN2(20) -> REDUCE _declarator

state 6074:
  anon_sym_LPAREN2(20) -> SHIFT 130
```

The DFA debug run showed that the clean CUDA path tokenizes the nested template
close as a single `>>` token in this context:

```text
DFA tok 20 (  8553 8554
DFA tok 39 <  8565 8566
DFA tok 39 <  8570 8571
DFA tok 41 >> 8572 8574
DFA tok 27 &  8575 8576
```

A temporary generic probe that split `>>` before `&` did admit more type-ish
inner reductions:

```text
12 type_specifier
 6 type_descriptor
 6 template_function
 5 template_argument_list
```

But it still admitted no function-declarator branch:

```text
0 parameter_list
0 parameter_declaration
0 function_declarator
0 _function_declarator_seq
```

The probe still reached the same close-paren failure route:

```text
state 3101 on "{":
  REDUCE argument_list [8553:8585] -> state 8235

state 8235 on "{":
  no action
```

The frame did not move under the probe:

```text
go.child[23]: type="template_declaration" [8510:8585]
go.child[24]: type="compound_statement" [8586:8797]
c.child[23]:  kind="template_declaration" [8510:8797]
```

Final clean-source validation remained green:

```text
ok   github.com/odvcencio/gotreesitter  0.002s [no tests to run]

cuda frame 1:
  go stopReason=accepted truncated=false rootHasError=true cRootHasError=false
  go.child[23]: type="ERROR" [8510:8585]
  go.child[24]: type="compound_statement" [8586:8797]
  c.child[23]:  kind="template_declaration" [8510:8797]

cuda frame 39:
  go stopReason=accepted truncated=false rootHasError=false cRootHasError=false
  FIRST-DIFF @root[1][16][2][2][16][0][1][5]
  go sizeof_expression [4132:4155] child-count=4
  c  sizeof_expression [4132:4155] child-count=2
```

Artifacts:

```text
harness_out/docker/20260626T003222Z-cuda-frame1-closeparen-reduce-existing-trace-20260626/container.log
harness_out/docker/20260626T003612Z-cuda-frame1-dfa-angle-debug-20260626/container.log
harness_out/docker/20260626T003706Z-cuda-frame1-generic-angle-split-probe-20260626/container.log
harness_out/docker/20260626T004036Z-cuda-frame1-parameter-admission-report-20260626/container.log
harness_out/docker/20260626T004051Z-cuda-frame39-parameter-admission-control-20260626/container.log
```

## Answers

1. No live stack reduced the close-paren span as `parameter_list`,
   `parameter_declaration`, `function_declarator`, `_function_declarator_seq`,
   or a related function-declarator node before `{`.

2. Because that branch was not admitted, there is no evidence that it
   disappeared in post-reduce fork merge, `mergeStacksWithScratch`,
   same-key cap, cull, shifted skip, reduce-chain `forceAdvanceAfterReduce`, or
   final selection in this window. With `GOT_GLR_MAX_STACKS=64`, the immediate
   failure is still not stack-budget culling.

3. The cause is earlier than final branch selection. It is not explained by a
   missing body action in the table or by a simple anonymous `(` symbol mismatch.
   The clean path remains on expression/argument-list reductions. Splitting
   compact `>>` is relevant to the inner template region but is not sufficient:
   after that split, the parser admits `template_argument_list` /
   `template_function` / `type_descriptor` reductions, yet still does not
   reduce `std::vector<Task<T>> &TaskList` as a `parameter_declaration` or the
   enclosing parens as a `parameter_list`.

## Conclusion

The CUDA frame 1 function-declarator/body branch is never admitted in the
requested live-stack window. The current failure is not a late loss of a
`parameter_list` / `function_declarator` branch; it is an admission failure
before the parser can reach states `6973`, `6908`, or `6969`.

A source fix is not justified from the current evidence. The temporary
close-angle split probe is too broad to keep on this witness alone and does not
move frame 1. It should only return as part of a generalized, table-driven fix
that also proves the parameter-declaration route is admitted.

## Next target

Target the generalized C++ ambiguous declarator admission path inside state
`130` and its descendants: why `std::vector<Task<T>> &TaskList` does not reduce
to `parameter_declaration` even when the inner template close is split and
type-like reductions are present.

Falsification criteria:

- If a trace shows `parameter_declaration`, `parameter_list`, or
  `function_declarator` reductions for this span and the branch later loses,
  move back to branch preservation, merge/cull, or final selection.
- If a table-driven compact close-angle split plus context-correct type
  admission reaches `6973`, `6908`, or `6969` before `{`, validate it against
  CUDA frame 1 and frame 39 and then consider a generalized source fix.
- If upstream C emits the same tokens and reductions but Go does not, target
  GLR reduce-chain/action ordering. If upstream C emits different token
  boundaries or lexer states in this parameter context, target DFA token-source
  state selection and compact close-angle splitting.
