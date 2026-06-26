# CUDA frame 1 token-frontier experiment, 2026-06-26

## Scope

Objective: build the next bounded generalized source-facing experiment for the
CUDA frame 1 token-frontier gap at:

```cpp
template <typename T> void initialise_tasks(std::vector<Task<T>> &TaskList) {}
```

The target was a table-driven GLR token-frontier mechanism that can preserve
per-version token alternatives at the same byte without CUDA policy,
language-name parser policy, or per-grammar normalization. The proof target was
row-225-equivalent `state=2126`, then `state=4600`, at `&TaskList`.

No source fix was kept. Temporary diagnostic code was removed after the probe.

## Commands

Required starting context:

```bash
git status --short --branch
git rev-parse --short HEAD
sed -n '1,260p' AGENTS.md
sed -n '1,560p' docs/reports/cuda-frame1-state130-c-vs-go-token-frontier-20260626.md
sed -n '1,320p' docs/reports/cuda-frame1-parameter-branch-admission-diagnostics-20260626.md
```

Focused host check while the temporary probe was present:

```bash
go test . -run 'TestNextDFATokenSplitsCompactCloseAngles|TestNextDFATokenKeepsRightShiftWhenRightShiftHasAction|TestNextGLRUnionDFAToken' -count=1
```

Temporary global close-angle split plus token-frontier diagnostic:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame1-token-frontier-probe-20260626 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8553:8587 GOT_GLR_TOKEN_FRONTIER_TRACE_WINDOW=8565:8576 REPRO_DEBUG_DFA=1 GOT_GLR_MAX_STACKS=64 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Clean-tokenization token-frontier diagnostic, with the existing close-angle
language gate restored:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame1-token-frontier-clean-trace-20260626 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8553:8587 GOT_GLR_TOKEN_FRONTIER_TRACE_WINDOW=8565:8576 REPRO_DEBUG_DFA=1 GOT_GLR_MAX_STACKS=64 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Artifacts:

```text
harness_out/docker/20260626T010027Z-cuda-frame1-token-frontier-probe-20260626/container.log
harness_out/docker/20260626T010144Z-cuda-frame1-token-frontier-clean-trace-20260626/container.log
harness_out/docker/20260626T010410Z-cuda-frame1-token-frontier-report-clean-20260626/container.log
harness_out/docker/20260626T010427Z-cuda-frame39-token-frontier-report-control-20260626/container.log
```

## Attempted mechanism

The temporary probe added a generic byte-window diagnostic to
`nextGLRUnionDFAToken`. For every unique active GLR lex-mode pair, it printed:

- origin parser state,
- candidate token symbol/name/span/text,
- lexer end position,
- number of live GLR states with an action on that token,
- supporting live states,
- and the single token selected by the existing scoring rule.

A second temporary variant removed the existing close-angle language-name gate
from `splitCompactCloseAngleToken`, modeling the broad table-driven global
close-angle split that prior reports had already falsified as a complete fix.

Both changes were removed after the experiment.

## Clean token-frontier result

With the existing source behavior restored, the Go token source exposes two
same-byte table-derived candidates at byte `8572`:

```text
GLR-TOKEN-FRONTIER pos=8572 primary=2576
  cand[0] origin=2576 sym=195 name=">"  span=8572:8573 text=">"  score=12
  cand[1] origin=2387 sym=41  name=">>" span=8572:8574 text=">>" score=18
  best sym=41 name=">>" span=8572:8574 text=">>"
```

This is the source-facing shape expected from the upstream C trace: different
versions at the same byte want different token boundaries. The current Go
runtime then emits one global token and advances one global cursor:

```text
DFA tok 41 >> 8572 8574 >> state=2576
```

The expression route survives. The declaration/type route that needed a
one-byte `>` at `8572` does not get its own lookahead/cursor.

## Global split result

When the close-angle split gate was temporarily removed, the runtime emitted
two one-byte close angles:

```text
DFA tok 195 > 8572 8573 > state=2576
DFA tok 36  > 8573 8574 > state=2404
```

That admitted inner type/template reductions:

```text
type_specifier [8571:8572] -> state=4810
qualified_identifier [8554:8573] -> state=3694
```

But it still did not admit the declaration route at `&`:

```text
no row-225 state=2126
no row-225 state=4600
no reference_declarator
no parameter_declaration
no parameter_list
no function_declarator
no _function_declarator_seq
```

The close-paren failure stayed on the argument-list route:

```text
state=3101 on "{":
  reduce argument_list [8553:8585] -> state=8235

state=8235 on "{":
  no action
```

Frame 1 did not move:

```text
go.child[23]: template_declaration [8510:8585]
go.child[24]: compound_statement [8586:8797]
c.child[23]:  template_declaration [8510:8797]
```

## Final clean validation

After removing all temporary parser/token-source code, the focused host package
check passed:

```text
ok github.com/odvcencio/gotreesitter 0.003s [no tests to run]
```

Clean CUDA frame 1 remained unmoved:

```text
go rootHasError=true cRootHasError=false
go.child[23]: ERROR [8510:8585]
go.child[24]: compound_statement [8586:8797]
c.child[23]:  template_declaration [8510:8797]
```

Clean CUDA frame 39 stayed controlled:

```text
go rootHasError=false cRootHasError=false
go: sizeof_expression [4132:4155] child-count=4
c:  sizeof_expression [4132:4155] child-count=2
```

## Why no source fix was kept

A queue inside `dfaTokenSource.Next()` would still feed queued alternatives to
all stacks globally. That is not equivalent to C parser versions carrying their
own lexer result.

The parser loop currently has a single `tok` and a single lexer cursor. A true
frontier mechanism needs to preserve at least:

- token alternatives keyed to the parser versions that can use them,
- per-version input byte after shifting alternatives with different widths,
- and reduce/relex behavior when a stack reduces under one lookahead while a
different stack shifts a different same-byte token.

Without that, choosing `>>` loses the declaration/type branch; choosing `>`/`>`
globally loses the compact expression branch and still does not reach
`state=2126` or `state=4600` on this witness.

Keeping a global close-angle split would therefore be both too broad and
insufficient.

## Conclusion

The bounded experiment confirms that the Go runtime can see the relevant
same-byte token alternatives from the grammar tables, but the existing
single-token parser/token-source contract cannot preserve them per GLR version.

The immediate CUDA frame 1 failure remains before `state=4600`. Global splitting
gets as far as inner type-ish reductions, including `qualified_identifier
[8554:8573]`, but the route still falls back to `argument_list -> state=8235`
at `{`.

## Next generalized target

Prototype per-version token dispatch outside `dfaTokenSource.Next()` rather
than as a token-source queue. The smallest useful falsification target is a
non-shipping harness that:

1. collects all table-admissible token candidates at the same byte,
2. dispatches only stacks whose current state has an action on each candidate,
3. keeps separate post-shift byte offsets for candidates with different widths,
4. then proves or falsifies whether the `>`/`>` version reaches
   `state=2126` on `&`.

Falsification criteria:

- If per-version dispatch reaches `state=2126` but not `state=4600`, inspect
  reduce-chain/action ordering for `qualified_identifier -> type_specifier ->
  _declaration_specifiers` under the `&` lookahead.
- If it reaches `state=4600` but does not reduce `parameter_declaration` and
  `parameter_list`, inspect reference-declarator reductions after shifting
  `TaskList`.
- If it still does not reach `state=2126`, the next target is not token
  boundaries but preservation/coalescing of the `qualified_identifier` branch
  after the second `>`.
