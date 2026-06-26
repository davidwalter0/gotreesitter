# CUDA frame 1 state 130 C-vs-Go token frontier diagnostic, 2026-06-26

## Scope

Objective: compare upstream tree-sitter C behavior against the Go runtime for
the CUDA frame 1 ambiguous declarator path:

```cpp
template <typename T> void initialise_tasks(std::vector<Task<T>> &TaskList) {}
```

The prior reports already proved that the Go runtime reaches `6074 -> 130` but
does not admit `parameter_list`, `parameter_declaration`,
`function_declarator`, or `_function_declarator_seq` from byte `8553` through
`8585`. This report advances the proof by tracing upstream C on the same full
file and by falsifying close-angle splitting as a complete explanation.

This is report-only. No parser source fix, grammar blob change, per-grammar
normalizer, CUDA-specific runtime policy, or language-name parser policy was
kept.

## Commands

Requested context:

```bash
git status --short --branch
git rev-parse --short=12 HEAD
sed -n '1,220p' AGENTS.md
sed -n '1,220p' docs/reports/cuda-frame1-parameter-branch-admission-diagnostics-20260626.md
sed -n '1,220p' docs/reports/cuda-frame1-action-table-fidelity-20260626.md
sed -n '1,220p' docs/reports/cuda-frame1-function-branch-preservation-diagnostics-20260626.md
```

Upstream C oracle trace for the full CUDA frame 1 file:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame1-c-log-state130-20260626 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_CLOG=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestCTreeDumpDiag$' -count=1 -v 2>&1 | grep -E 'CLOG|initialise_tasks|parameter|function_declarator|argument_list|type_descriptor|template|TaskList|8553|8585|8586|state:130|state:6074|state:3101|state:8235|state:6973|state:6908|state:6969'"
```

Clean Go runtime trace for the same full CUDA frame 1 file:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame1-go-state130-descendants-20260626 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8553:8587 REPRO_DEBUG_DFA=1 GOT_GLR_MAX_STACKS=64 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Temporary table-driven close-angle split probe. The source change removed the
existing language-name gate for compact close-angle token preference, then was
reverted because it did not admit the parameter route:

```bash
go test . -run 'TestNextDFATokenSplitsCompactCloseAngles|TestNextDFATokenKeepsRightShiftWhenRightShiftHasAction|TestNormalizeBash|TestBashGenerated' -count=1

bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame1-table-driven-angle-split-proof-20260626 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8553:8587 REPRO_DEBUG_DFA=1 GOT_GLR_MAX_STACKS=64 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Final clean-source validation:

```bash
go test . -run '^$' -count=1

bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame1-state130-report-clean-20260626 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"

bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame39-state130-report-control-20260626 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/simpleTexture3D/simpleTexture3D_kernel.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

No minimized snippet was kept for this pass. The full-file C trace is already
specific enough to identify the missing route; the next useful snippet should
be run after per-version token/reduce instrumentation exists, so it can prove
or falsify row-225-equivalent `state=2126 -> state=4600` admission rather than
only reproducing the top-level shape.

## Byte map

The relevant full-file offsets are:

```text
8510 row 225 col 0  "template <typename T"
8553 row 225 col 43 "(std::vector<Task<T>"
8565 row 225 col 55 "<Task<T>> &TaskList)"
8570 row 225 col 60 "<T>> &TaskList)\n{\n  "
8572 row 225 col 62 ">> &TaskList)\n{\n    "
8574 row 225 col 64 " &TaskList)\n{\n    fo"
8575 row 225 col 65 "&TaskList)\n{\n    for"
8585 row 225 col 75 "\n{\n    for (unsigned"
8586 row 226 col 0  "{\n    for (unsigned "
8797 row 233 col 1  "\n\nint main(int argc,"
```

## Upstream C behavior

The upstream C parser produces a single body-bearing template declaration:

```text
c.child[23]: template_declaration [8510:8797]
```

The full C trace reaches the same state-130 path:

```text
state:6390 row:225 col:43
  lex "("
  reduce _declarator
  shift state:130

state:130 row:225 col:44
  lex identifier "std"
  shift state:917

state:917 row:225 col:47
  lex "::"
  shift state:7257

state:7257 row:225 col:49
  lex identifier "vector"
  reduce _scope_resolution
  shift state:1061
```

At the first `<`, C preserves both qualified-identifier/type-like and
expression-like versions:

```text
state:1061 row:225 col:55
  lex "<"
  reduce qualified_identifier
  shift state:560

state:3694 row:225 col:55
  lex "<"
  reduce expression
  shift state:1523
```

At `Task<T>>`, upstream C uses per-version lexer state. One expression version
lexes compact `>>`, while type/template versions lex separate `>` tokens at
the same byte range:

```text
state:2387 row:225 col:62
  lex ">>" size:2
  reduce expression
  shift state:1524

state:4810 row:225 col:62
  lex ">" size:1
  reduce type_descriptor
  shift state:2404

state:2404 row:225 col:63
  lex ">" size:1
  reduce template_argument_list
  reduce template_type
  reduce template_function
  reduce expression
  shift state:945
```

The body-capable branch appears at `&`, before `TaskList`:

```text
state:2126 row:225 col:64
  lex "&"
  reduce qualified_identifier
  reduce type_specifier
  reduce _declaration_specifiers
  shift state:4600
```

That route then reduces the reference parameter and function declarator:

```text
state:4600 row:225 col:66
  lex identifier "TaskList"
  shift state:6390

state:6390 row:225 col:74
  reduce _declarator
  reduce reference_declarator
  reduce _declarator
  reduce parameter_declaration
  shift state:4509

state:4509 row:225 col:75
  lex "{"
  reduce parameter_list
  reduce _function_declarator_seq
  reduce function_declarator
  reduce _declarator
  shift state:3
```

The competing expression route also exists, but C detects it as the failing
version:

```text
state:3101 row:225 col:75
  reduce argument_list
  detect_error
```

## Go behavior

The clean Go trace selects one token for the whole active GLR frontier at the
inner template close:

```text
DFA tok 41 >> 8572 8574 >> state=2576
```

All live stacks therefore see `>>`; type-oriented versions that require `>` at
8572 never get their own token. The surviving stack at `&` is expression-like:

```text
DFA tok 33 & 8575 8576 & state=1387
DFA tok 1 identifier 8576 8584 TaskList state=1349
```

No row-225 `state=4600` is reached in the Go trace. The only `state=4600`
tokens found in the run are unrelated earlier spans:

```text
6660:6663 one
6809:6810 t
8392:8395 one
```

At `{`, the Go runtime is still on the argument-list route:

```text
state=3101 on "{":
  reduce argument_list [8553:8585] -> state=8235

state=8235 on "{":
  no action
```

In the traced `GOT_GLR_MAX_STACKS=64` diagnostic, the final split remains
unchanged even though the short node label can differ from the default
validation:

```text
go.child[23]: template_declaration [8510:8585]
go.child[24]: compound_statement [8586:8797]
c.child[23]:  template_declaration [8510:8797]
```

## Close-angle split falsification

A temporary probe removed the compact close-angle language-name gate so CUDA
could split the `>>` based on active table actions. The probe did split the
token:

```text
DFA tok 195 > 8572 8573 > state=2576
DFA tok 36  > 8573 8574 > state=2404
```

It also admitted the expected inner type/template reductions:

```text
type_descriptor [8571:8572]
template_argument_list [8565:8573]
template_function [8559:8573]
qualified_identifier [8554:8573]
```

But it still did not admit the parameter route:

```text
no row-225 state=4600
no reference_declarator
no parameter_declaration
no parameter_list
no function_declarator
no _function_declarator_seq
```

After the split, the live route at `&` was still expression-like:

```text
DFA tok 33 & 8575 8576 & state=1387
DFA tok 1 identifier 8576 8584 TaskList state=1349
```

And the close-paren failure was unchanged:

```text
state=3101 on "{":
  reduce argument_list [8553:8585] -> state=8235
```

Therefore close-angle splitting is a necessary-looking part of upstream C's
behavior for this span, but it is not sufficient in the Go runtime.

## Final validation results

The focused host package check passed:

```text
ok github.com/odvcencio/gotreesitter 0.003s [no tests to run]
```

The clean-source CUDA frame 1 Docker validation did not move the frame. Go
still separates the declaration header from the body:

```text
go rootHasError=true cRootHasError=false
go.child[23]: ERROR [8510:8585]
go.child[24]: compound_statement [8586:8797]
c.child[23]:  template_declaration [8510:8797]
```

The CUDA frame 39 control stayed bounded and root-error-free on both parsers.
Its known local shape difference remained:

```text
go rootHasError=false cRootHasError=false
go: sizeof_expression [4132:4155] with type_descriptor child
c:  sizeof_expression [4132:4155] with parenthesized_expression child
```

## Conclusion

Upstream C does not choose between `>>` and `>`/`>` globally. It carries
simultaneous parser versions with different lexer states at the same byte: the
expression branch can lex compact `>>`, while the declaration/type branch can
lex two separate `>` tokens.

The Go runtime currently uses a single token stream for the active GLR
frontier. That loses C's per-version tokenization at byte `8572`. However, the
temporary table-driven split proves that tokenization alone is not the whole
bug: after the split, Go reaches `qualified_identifier [8554:8573]` but still
does not preserve or derive the C route:

```text
state:2126 on "&"
  qualified_identifier
  type_specifier
  _declaration_specifiers
  shift state:4600
```

The next missing transition is before `state=4600`, not at `6074` and not at
body-capable states `6973/6908/6969`.

## Recommended next target

Target a generalized GLR token-frontier and reduce-preservation investigation
around `qualified_identifier -> type_specifier -> _declaration_specifiers` in
ambiguous declarator contexts.

Falsification criteria:

1. Instrument per-stack/per-version token choices at bytes `8572:8575`, without
   a CUDA or language-name policy. If a per-version token queue admits both
   `>>` and `>`/`>` but still does not produce row-225 `state=2126`, then the
   remaining bug is stack/reduce preservation before `&`.
2. If row-225 `state=2126` appears but does not reduce to `_declaration_specifiers`
   and shift `&` to `state=4600`, inspect action ordering/reduce chaining for
   `qualified_identifier` under the `&` lookahead.
3. A source fix is justified only when it is table-driven/generalized, admits
   the `state=4600 -> 6390 -> 4509` route, moves CUDA frame 1 to a body-bearing
   `template_declaration [8510:8797]`, and keeps the CUDA frame 39 control
   bounded.
