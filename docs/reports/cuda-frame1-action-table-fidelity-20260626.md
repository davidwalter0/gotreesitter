# CUDA frame 1 action-table fidelity diagnostic, 2026-06-26

## Scope

Question: why does the CUDA/C++ path for
`initialise_tasks(std::vector<Task<T>> &TaskList)` reach the
argument-list/init-declarator route and not a function-declarator route that
can shift `{`?

This is report-only. No parser, grammargen, or grammar blob source change was
kept. No repo-wide host `go test ./...` was run.

## Commands

Read prior diagnostics:

```bash
sed -n '1,260p' docs/reports/cuda-frame1-function-branch-preservation-diagnostics-20260626.md
sed -n '1,260p' docs/reports/cuda-frame1-template-candidate-falsification-20260625.md
```

Checked that CUDA is a `ts2go` parser.c-extracted blob, not a grammargen-owned
blob:

```bash
sed -n '234,246p' cmd/ts2go/batch_run.go
```

Fetched the pinned CUDA grammar source from `grammars/languages.lock`:

```bash
rm -rf /tmp/gts-cuda-48b066f
git clone --filter=blob:none --no-checkout https://github.com/theHamsta/tree-sitter-cuda /tmp/gts-cuda-48b066f
git -C /tmp/gts-cuda-48b066f checkout 48b066f334f4cf2174e05a50218ce2ed98b6fd01 -- src/parser.c src/grammar.json grammar.js
```

Regenerated a temporary ts2go blob from the pinned `parser.c`:

```bash
go run ./cmd/ts2go -input /tmp/gts-cuda-48b066f/src/parser.c -output /tmp/cuda_ts2go_probe.go -name cuda
```

Decoded existing shipped CUDA conflicts:

```bash
GTS_DECODE=1 GTS_DECODE_TARGETS='cuda:3101,8235,85,84,6390,6074,1061' \
  go test . -run '^TestDecodeConflictStates$' -count=1 -v
```

For exact single/no-action entries, a temporary extension to
`TestDecodeConflictStates` printed selected symbols (`(` symbol 5, `(` symbol
20, `<`, `{`, `)`, `compound_statement`, `argument_list`,
`virtual_specifier`). The temporary test code was removed after collecting the
output.

Conflict declarations from pinned `grammar.json`:

```bash
jq -r '.conflicts[] | map(if type=="object" then .name else . end) | @tsv' \
  /tmp/gts-cuda-48b066f/src/grammar.json | nl -ba
```

Docker frame controls:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame1-action-fidelity-report-20260626 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"

bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cuda-frame39-action-fidelity-control-20260626 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/simpleTexture3D/simpleTexture3D_kernel.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

## Evidence

The shipped CUDA blob is parser.c-extracted via `ts2go`. In
`cmd/ts2go/batch_run.go`, only `go` is marked grammargen-owned:

```text
var grammargenOwnedBlobs = map[string]bool{
    "go": true,
}
```

Temporary `ts2go` extraction from pinned CUDA `parser.c` succeeded with the
same state count as the shipped blob:

```text
Generated /tmp/cuda_ts2go_probe.go and /tmp/grammar_blobs/cuda.bin (cuda language, 9557 states, 555 symbols)
```

The symbol count differs from the checked-in blob (`560`) because local scanner
and registry augmentation add metadata, but the relevant parser state IDs and
actions are from the same upstream C table.

The grammar conflict declarations are not missing. Pinned CUDA `grammar.json`
contains these relevant groups:

```text
1   type_specifier  _declarator
15  expression      _declarator
17  expression      _declarator      type_specifier
23  parameter_list  argument_list
24  type_specifier  call_expression
27  _function_declarator_seq
```

The observed path is the argument-list path. The exact action rows from the
shipped Go blob are:

```text
state 1061, lookahead "<" (sym 39):
  REDUCE qualified_identifier (childCount=2 prod=34)
  SHIFT -> state 560

state 3101, lookahead "{" (sym 70):
  REDUCE argument_list (childCount=3)

state 8235, lookahead "{" (sym 70):
  no action
```

The body-capable function route exists elsewhere in the same table:

```text
state 6973, lookahead "{" (sym 70):
  SHIFT -> state 55

state 6908, lookahead "{" (sym 70):
  SHIFT -> state 53

state 6969, lookahead "{" (sym 70):
  SHIFT -> state 43
```

Those states also show the expected function-body neighborhood:

```text
state 6973, lookahead compound_statement:
  REDUCE preproc_if_repeat1 (childCount=2)
  SHIFT -> state 8644 REPETITION

state 6908, lookahead compound_statement:
  REDUCE preproc_if_repeat1 (childCount=2)
  SHIFT -> state 2282 REPETITION
```

The failing stack does not reach those states. Prior tracing showed the path:

```text
state 6390 on "(":
  REDUCE _declarator (childCount=1)

state 6074 on "(":
  SHIFT -> state 130

state 3101 on "{":
  REDUCE argument_list (childCount=3)

state 8235 on "{":
  no action
```

So the table is not missing a `{` shift globally. The missing branch is the
route into the function-declarator/body states before `{`.

Clean-source frame 1 replay remains the same accepted, non-truncated
root-shape divergence:

```text
go stopReason=accepted truncated=false rootHasError=true cRootHasError=false
FIRST-DIFF @root
go.child[23]: type="ERROR" [8510:8585]
go.child[24]: type="compound_statement" [8586:8797]
c.child[23]:  kind="template_declaration" [8510:8797]
```

Frame 39 control remains accepted and no-error, with only the existing
`sizeof_expression` divergence:

```text
go stopReason=accepted truncated=false rootHasError=false cRootHasError=false
FIRST-DIFF @root[1][16][2][2][16][0][1][5]
go sizeof_expression [4132:4155] child-count=4
c  sizeof_expression [4132:4155] child-count=2
```

Artifacts:

```text
harness_out/docker/20260626T002207Z-cuda-frame1-action-fidelity-report-20260626/container.log
harness_out/docker/20260626T002228Z-cuda-frame39-action-fidelity-control-20260626/container.log
```

## Conclusion

The answer is: the observed Go stack is already on the call/argument-list
interpretation before `{`. State `3101` is an argument-list completion state;
on `{` it can only reduce `argument_list`, and after that state `8235` has no
body action. The function-declarator/body route is present in the same CUDA
action table, but it is represented by different states (`6973`, `6908`,
`6969`) that directly shift `{`.

This does not justify a grammargen conflict-import fix. The relevant conflicts
are present in pinned CUDA `grammar.json`, and the checked-in CUDA blob is
`ts2go`-extracted from upstream `parser.c`, not generated by grammargen. It
also does not justify a CUDA-specific runtime policy or a per-grammar
normalizer.

The remaining root cause is earlier runtime branch divergence: determine why
the Go parser path that starts at `6390 -> 6074 -> 130` does not also preserve
or reach the function-declarator path that later lands in `{`-shift states.
The prior report showed this was not lost immediately before `{`; this report
narrows the next trace window earlier, through the transition from
`initialise_tasks (` into the `std::vector<Task<T>> &TaskList )` close.

## Next target

Trace live stacks and reductions from byte `8553` (`(` after
`initialise_tasks`) through byte `8585` (`)`) with exact stack states and top
symbols, not just the final `{` lookup. Specifically:

- identify the stack, if any, that reduces the close-paren span as
  `parameter_list` and then `function_declarator`;
- compare it with the stack that reduces the same span as `argument_list`;
- check whether the function-declarator stack is merged, capped, or
  deprioritized before reaching states `6973`, `6908`, or `6969`;
- if no such stack ever appears, inspect token/context differences around the
  two anonymous `(` tokens (`sym 5` and `sym 20`) and the `parameter_list /
  argument_list` declared conflict.

Falsification criteria:

- If a trace shows a live stack entering `6973`, `6908`, or `6969` before `{`
  and then losing to merge/cull/cost before shifting `{`, the target moves back
  to runtime branch preservation.
- If no stack ever reduces `parameter_list`/`function_declarator`, the next
  target is admission of the parameter-list branch between states `6074`,
  `130`, and the close-paren predecessor, not recovery or table generation.
