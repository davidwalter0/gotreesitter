# CUDA frame 1 token-frontier trailing-error follow-up, 2026-06-26

## Scope

Follow-up on the env-gated generalized GLR token-frontier dispatch prototype
from commit `17efb886` on branch `repair/java-token-admission-reduced`.

The target witness remains:

```cpp
template <typename T> void initialise_tasks(std::vector<Task<T>> &TaskList) {}
```

in the full CUDA frame 1 file:

```text
/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu
```

No CUDA-specific policy, language-name parser policy, or per-grammar
normalizer was added.

## Source change kept

The prototype advanced the shared DFA lexer cursor to the shortest same-byte
candidate end. That is still only an approximation of C's per-version lexer
position, but it is needed so the one-byte `>` branch can see the second `>` at
byte `8573`.

The unsafe part was that the next frontier was offered to every live stack by
state alone. Stacks that had already shifted the wider `>>` candidate through
byte `8574` could be considered for a follow-up token starting at byte `8573`.
The normal shift guard treats `tok.StartByte <= stack.byteOffset` as not being
a forward gap, so this could dispatch stale overlapping lookahead to a
different-width version.

The kept generalized guard is:

```go
if cand.tok.StartByte < originals[si].byteOffset {
    continue
}
```

inside the env-gated token-frontier dispatch loop. It does not change default
behavior and does not choose policy by language or grammar.

## Gated CUDA frame 1 result

With `GOT_GLR_TOKEN_FRONTIER_DISPATCH=1` and `GOT_GLR_MAX_STACKS=64`, frame 1
still does not fully clear:

```text
go.child[23]: template_declaration [8510:8795]
go.child[24]: ERROR [8796:8797]
c.child[23]:  template_declaration [8510:8797]
```

The token-frontier work still moved the parse substantially versus the clean
default control. The trace admits the C-like declaration route:

```text
target_state=2126
state=2126 on "&"
action[0] shift state=4600
state=4600 on identifier "TaskList"
reduce _function_declarator_seq
reduce function_declarator
reduce parameter_list
```

So the trailing error is no longer caused by missing `state=2126`,
`state=4600`, `parameter_list`, or `function_declarator` admission.

## Trailing-error cause

The remaining failure is later, inside the parsed function body. At byte
`8796:8797`, the parser is in `state=138` with no action on `}`:

```text
tok=8796:8797 state=138 action_count=0
cCondenseAndResume resume_stack=0 state=138 byte=8795 tok=8796:8797
```

Recovery then reduces a compound/function/template route ending before that
lookahead:

```text
compound_statement [8586:8795] reduced_error_extras=1
function_definition [8532:8795]
template_declaration [8510:8795]
```

In source, byte `8794:8795` is the inner `for` body close and byte `8796:8797`
is the function body close:

```text
8788 ';'
8794 '}'
8795 '\n'
8796 '}'
8797 '\n'
```

The gated token-frontier prototype gets through the declarator and opens the
function body, but the later statement/compound path lets the outer compound
close at the inner `for` body brace. The actual function-closing brace is then
absorbed as a top-level trailing `ERROR`.

## Validation

Focused host test:

```bash
go test . -run '^TestGLRTokenFrontierDispatchKeepsNarrowCandidateVersion$' -count=1
```

Result:

```text
ok github.com/odvcencio/gotreesitter 0.004s
```

Gated CUDA frame 1 with diagnostic cap:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label cuda-frame1-frontier-byte-guard --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_GLR_TOKEN_FRONTIER_DISPATCH=1 GOT_GLR_MAX_STACKS=64 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Result:

```text
PASS
go.child[23]: template_declaration [8510:8795]
go.child[24]: ERROR [8796:8797]
c.child[23]:  template_declaration [8510:8797]
```

Gated CUDA frame 1 route trace:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label cuda-frame1-frontier-byte-guard-trace --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8553:8587 GOT_GLR_TOKEN_FRONTIER_DISPATCH=1 GOT_GLR_MAX_STACKS=64 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Result:

```text
PASS
state=2126 admitted
state=4600 admitted
parameter_list admitted
function_declarator admitted
```

Gated CUDA frame 39 default-cap control:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label cuda-frame39-frontier-byte-guard-default-cap --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/simpleTexture3D/simpleTexture3D_kernel.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_GLR_TOKEN_FRONTIER_DISPATCH=1 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Result:

```text
PASS
go rootHasError=false
cRootHasError=false
first diff remains sizeof_expression [4132:4155]
```

CUDA frame 1 default control without env gate:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label cuda-frame1-default-control-byte-guard --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Result:

```text
PASS
go.child[23]: ERROR [8510:8585]
go.child[24]: compound_statement [8586:8797]
c.child[23]:  template_declaration [8510:8797]
```

## Not ready for default-on

The gate should stay off by default. CUDA frame 1 no longer fails at the token
frontier/declarator admission point, but it still has a root error and does not
match C for the body-bearing template declaration.

## Next generalized target

The next target is the generalized compound/statement continuation after the
function body opens:

1. Trace the stack route from the `function_declarator -> state=3` shift on
   `{` at `8586:8587`.
2. Compare the C and Go handling of the nested `for (...) { ... }` body close
   at `8794:8795`.
3. Determine why Go has only the route where the outer compound closes before
   `8796:8797`, instead of preserving or selecting the route where the inner
   compound/for statement closes first and the outer compound consumes the
   second brace.

This is a parser-version continuation/recovery selection issue after the token
frontier has already admitted the C-like declarator route, not a CUDA-specific
token policy problem.
