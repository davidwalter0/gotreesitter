# CUDA frame 1 faithful-condense / merge-cap diagnostic, 2026-06-26

## Scope

Goal slice: test whether the existing generalized faithful-condense same-key
preservation path explains or fixes CUDA frame 1's remaining trailing
`ERROR [8796:8797]` after `GOT_GLR_TOKEN_FRONTIER_DISPATCH=1` admits the
C-like declarator route.

Witness:

```cpp
template <typename T> void initialise_tasks(std::vector<Task<T>> &TaskList) {}
```

inside:

```text
/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu
```

No parser source change, per-grammar normalizer, CUDA language-name policy, or
grammar blob edit was kept.

## Code path inspected

Relevant generalized machinery:

- `parser.go` token-frontier dispatch clones each live stack for DFA token
  candidates and rejects stale overlapping candidates with
  `cand.tok.StartByte < originals[si].byteOffset`.
- `parser_recover_c.go` `cCondenseAndResume` prunes/resumes C recovery
  versions after a no-reduce dispatch pass.
- `glr.go` `mergeStacksWithScratch` enforces `(state, byteOffset)` merge-key
  caps; `GOT_FAITHFUL_CONDENSE=1` only affects the `perKeyCap == 1` overflow
  path by trying GSS main-branch merge/preservation instead of immediately
  replacing or dropping same-key alternatives.
- `parser_config.go` keeps `GOT_FAITHFUL_CONDENSE` and token-frontier dispatch
  env-gated.

This makes the diagnostic separable: if cap-one faithful preservation is the
cause, `GOT_GLR_MAX_MERGE_PER_KEY=1` should need `GOT_FAITHFUL_CONDENSE=1`;
if narrower merge caps are enough, cap 1 or 2 should move the frame even
without the faithful flag.

## Docker matrix

All runs used Docker isolation, one CUDA frame at a time:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_DIR=/workspace/corpus_sources ... go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

### Frame 1 baseline

Env:

```text
GOT_C_RECOVERY=all
GOT_GLR_TOKEN_FRONTIER_DISPATCH=1
GOT_GLR_MAX_STACKS=64
```

Artifact:

```text
harness_out/docker/20260626T014051Z-cuda-frame1-frontier-base-20260626-debugger/container.log
```

Result:

```text
go stopReason=accepted truncated=false rootHasError=true cRootHasError=false
FIRST-DIFF @root
go.child[23]: template_declaration [8510:8795]
go.child[24]: ERROR [8796:8797]
c.child[23]:  template_declaration [8510:8797]
```

This reconfirmed the known trailing top-level brace error.

### Frame 1 faithful cap-one

Env added:

```text
GOT_FAITHFUL_CONDENSE=1
GOT_GLR_MAX_MERGE_PER_KEY=1
```

Artifact:

```text
harness_out/docker/20260626T014105Z-cuda-frame1-frontier-faithful-cap1-20260626-debugger/container.log
```

Result:

```text
go stopReason=accepted truncated=false rootHasError=true cRootHasError=false
FIRST-DIFF @root[16][2][2][9][2][4][0][1][1][1][3][0][0]
go sizeof_expression [3545:3554] child-count=4
c  sizeof_expression [3545:3554] child-count=2
```

The targeted trailing `ERROR [8796:8797]` cleared: the first diff moved away
from root-level frame 1 and into the earlier `Task` template body. This is a
material advance for the frame, but not a full CUDA frame 1 parity clear because
`rootHasError` remains true.

### Frame 1 cap-width probes

`GOT_FAITHFUL_CONDENSE=1 GOT_GLR_MAX_MERGE_PER_KEY=2`

Artifact:

```text
harness_out/docker/20260626T014130Z-cuda-frame1-frontier-faithful-cap2-20260626-debugger/container.log
```

Result:

```text
go rootHasError=true cRootHasError=false
FIRST-DIFF @root[16][2][2][9][2][9][7][1]
go declaration [3880:3900]
c  expression_statement [3880:3900]
```

`GOT_GLR_MAX_MERGE_PER_KEY=2` without `GOT_FAITHFUL_CONDENSE`

Artifact:

```text
harness_out/docker/20260626T014156Z-cuda-frame1-frontier-cap2-no-faithful-20260626-debugger/container.log
```

Result: identical to faithful cap 2.

`GOT_GLR_MAX_MERGE_PER_KEY=1` without `GOT_FAITHFUL_CONDENSE`

Artifact:

```text
harness_out/docker/20260626T014210Z-cuda-frame1-frontier-cap1-no-faithful-20260626-debugger/container.log
```

Result:

```text
go rootHasError=true cRootHasError=false
FIRST-DIFF @root[19][2][2][2][7]
go compound_statement [4485:4614] contains ERROR [4495:4507] "result[i] *="
c  compound_statement [4485:4614]
```

These probes falsify the narrow claim that cap-one faithful same-key
preservation is the only cause of the frame 1 improvement. Narrower merge caps
also change survivor selection enough to keep the full-span frame 1 function
route. Faithful cap-one is still relevant: at cap 1 it selects a different
survivor set than plain cap 1 and avoids the `result[i] *=` error observed in
the plain cap-one run.

### Frame 1 closing-brace trace

Env:

```text
GOT_C_RECOVERY=all
GOT_C_RECOVERY_TRACE_WINDOW=8788:8798
GOT_GLR_TOKEN_FRONTIER_DISPATCH=1
GOT_GLR_MAX_STACKS=64
GOT_FAITHFUL_CONDENSE=1
GOT_GLR_MAX_MERGE_PER_KEY=1
```

Artifact:

```text
harness_out/docker/20260626T014312Z-cuda-frame1-frontier-faithful-cap1-close-trace-20260626-debugger/container.log
```

Key trace evidence:

```text
state=339 on "}" 8796:8797 reduces compound_statement
for_statement [8592:8795] reduces on "}" 8796:8797
state=64 shifts "}" 8796:8797 -> state=138
state=138 on primitive_type 8799:8802 reduces compound_statement
function_definition [8532:8797] reduces
```

In the baseline report, `state=138` had no action on `}` at `8796:8797`, so
recovery closed `function_definition [8532:8795]` and stranded the final brace.
In this run, the surviving route reaches `state=138` only after consuming the
second brace; the next lookahead is `int` at `8799:8802`, and the function
definition reduces across `[8532:8797]`.

## Frame 39 controls

Frame 39:

```text
/workspace/corpus_sources/cuda/cpp/0_Introduction/simpleTexture3D/simpleTexture3D_kernel.cu
```

Faithful cap-one artifact:

```text
harness_out/docker/20260626T014238Z-cuda-frame39-frontier-faithful-cap1-20260626-debugger/container.log
```

Result:

```text
go rootHasError=false cRootHasError=false
(no structural divergence)
```

Cap 2 without faithful artifact:

```text
harness_out/docker/20260626T014251Z-cuda-frame39-frontier-cap2-no-faithful-20260626-debugger/container.log
```

Result:

```text
go rootHasError=false cRootHasError=false
FIRST-DIFF @root[1][14][3][7][2][3]
go declaration [2399:2425]
c  expression_statement [2399:2425]
```

Neither promising frame 1 variant worsened frame 39 into an error state.
Faithful cap-one improved this control relative to the previously reported
known non-error divergence.

## Conclusion

The existing generalized faithful cap-one path can fix CUDA frame 1's targeted
trailing `ERROR [8796:8797]` when combined with token-frontier dispatch and
`GOT_GLR_MAX_MERGE_PER_KEY=1`. It preserves a route where the inner
`for_statement` closes at `8794:8795`, the outer compound consumes
`8796:8797`, and the function definition reduces through byte `8797`.

However, the evidence does not justify defaulting that machinery yet:

- The target also clears with `GOT_GLR_MAX_MERGE_PER_KEY=2` without
  `GOT_FAITHFUL_CONDENSE`, so the fix is not uniquely explained by cap-one
  faithful same-key preservation.
- Plain cap one clears the target but introduces a different earlier ERROR,
  proving that survivor selection is still delicate.
- Frame 1 still has `rootHasError=true` under every clearing variant.
- Token-frontier dispatch itself remains env-gated.

No small generalized defaultable code change is clearly safe from this evidence
alone.

## Next invariant

The next generalized target should be merge survivor selection around the
function-body route, not CUDA policy or grammar normalization:

1. Trace the same-key groups near bytes `8794:8797` under default cap 6,
   cap 2, plain cap 1, and faithful cap 1.
2. Identify which merge key admits the full-span
   `function_definition [8532:8797]` route under narrow caps and loses it under
   the default cap.
3. Derive a content-based survivor invariant that keeps the recoverable
   compound/function continuation without relying on lower merge caps or
   language-specific policy.

The falsified lane is "faithful cap-one alone explains the CUDA frame 1
residual." The supported lane is "token-frontier dispatch plus generalized
merge survivor selection controls whether the recovered compound path survives."
