# CUDA frame 1 merge/result-selection invariant diagnostic, 2026-06-26

## Scope

Goal slice: narrow CUDA frame 1 from "merge survivor selection matters" to the
specific generalized merge/result-selection invariant controlling the
full-span route for:

```cpp
template <typename T> void initialise_tasks(std::vector<Task<T>> &TaskList) {}
```

in:

```text
/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu
```

No per-grammar normalizer, CUDA language-name parser policy, grammar blob edit,
or parser source change is kept in this report.

## Temporary instrumentation

Existing logs did not include same-key merge contents, so a temporary generic
`GOT_GLR_MERGE_TRACE_WINDOW=start:end` probe was added and then removed before
commit. The probe printed merge keys, top state/span/symbol, byte offset,
score, error rank, shifted bit, branch order, hash, and append/drop/replace or
GSS-merge behavior for stacks whose top or byte offset touched the byte window.

Focused package compile while the probe was present:

```bash
go test . -run '^TestGLRTokenFrontierDispatchKeepsNarrowCandidateVersion$' -count=1
```

Result:

```text
ok github.com/odvcencio/gotreesitter 0.004s
```

## Docker artifacts

All runs used Docker isolation, one CUDA frame at a time, with:

```text
GOT_C_RECOVERY=all
GOT_GLR_TOKEN_FRONTIER_DISPATCH=1
GOT_GLR_MAX_STACKS=64
```

Frame 1 trace artifacts:

```text
harness_out/docker/20260626T015314Z-cuda-frame1-merge-trace-default-top-20260626-debugger/container.log
harness_out/docker/20260626T015341Z-cuda-frame1-merge-trace-cap2-no-faithful-top-20260626-debugger/container.log
harness_out/docker/20260626T015400Z-cuda-frame1-merge-trace-faithful-cap1-top-20260626-debugger/container.log
harness_out/docker/20260626T015510Z-cuda-frame1-merge-trace-cap1-no-faithful-small-20260626-debugger/container.log
harness_out/docker/20260626T015628Z-cuda-frame1-default-close-action-report-20260626-debugger/container.log
harness_out/docker/20260626T015648Z-cuda-frame1-cap2-close-action-report-20260626-debugger/container.log
```

Frame 39 controls:

```text
harness_out/docker/20260626T015549Z-cuda-frame39-faithful-cap1-report-control-20260626-debugger/container.log
harness_out/docker/20260626T015549Z-cuda-frame39-cap2-no-faithful-report-control-20260626-debugger/container.log
```

## Default-cap failure

Default cap (`GOT_GLR_MAX_MERGE_PER_KEY` unset, effective cap 6) still fails at
the target:

```text
go.child[23]: template_declaration [8510:8795]
go.child[24]: ERROR [8796:8797]
c.child[23]:  template_declaration [8510:8797]
```

The close-brace action trace shows this is not final result selection losing a
surviving full-span route. The surviving route has already closed the outer
compound on the inner `for` close:

```text
state=64 tok="}" 8794:8795 shifts -> state=138
state=138 tok="}" 8796:8797 action_count=0
function_definition raw_span=8532:8795
template_declaration raw_span=8510:8795
```

Recovery then absorbs the second brace and later carries error-bearing
top-level repeat routes. The merge trace around the boundary contains normal
`state=2282 byte=8802` candidates for the following `int`, but also recovery
`state=0 byte=8802` and `state=85 byte=8797` candidates whose top node is
`ERROR [8796:8797]`. It does not show a non-error
`template_declaration [8510:8797]` or `function_definition [8532:8797]`
survivor in the default-cap path.

Conclusion: the full-span frame 1 route is dropped before final result
selection. It is not merely reordered and later rejected by
`stackCompareForResultSelection`.

## Cap 2 without faithful

`GOT_GLR_MAX_MERGE_PER_KEY=2` without `GOT_FAITHFUL_CONDENSE` clears the target
trailing error but leaves an earlier non-error divergence:

```text
FIRST-DIFF @root[16][2][2][9][2][9][7][1]
go declaration [3880:3900]
c  expression_statement [3880:3900]
```

The close-brace trace shows the good structural route:

```text
state=339 tok="}" 8796:8797 reduces inner compound/for
target_state=64 for statement [8592:8795]
state=64 tok="}" 8796:8797 shifts -> state=138
state=138 tok=primitive_type 8799:8802 reduces:
  compound_statement [8586:8797]
  function_definition [8532:8797]
```

The merge trace shows bounded same-key overflow at
`key=(state=2282 byte=8802) cap=2`. Survivors are normal, shifted,
non-error candidates with `score=95`; overflow replacement/drop is governed by
the merge comparator and hash tie-break after equal rank. Unlike default cap,
the boundary trace does not enter the `state=0 byte=8802` recovery group.

Interpretation: cap 2 changes upstream survivor pressure enough to keep the
normal close-brace continuation alive. It does not prove a safe default
invariant by itself, because it also changes unrelated earlier C/CUDA
expression-vs-declaration choices.

## Faithful cap-one

`GOT_GLR_MAX_MERGE_PER_KEY=1 GOT_FAITHFUL_CONDENSE=1` also clears the target and
moves the first diff earlier:

```text
FIRST-DIFF @root[16][2][2][9][2][4][0][1][1][1][3][0][0]
go sizeof_expression [3545:3554]
c  sizeof_expression [3545:3554]
```

The merge trace shows the cleanest same-key invariant:

```text
key=(state=85 byte=8797) cap=1
top=template_declaration[8510:8797] errRank=0 shifted=false score=103
op=cap1-gss-merge / cap1-gss-preserve

key=(state=724 byte=8797) cap=1
top=compound_statement[8586:8797] errRank=0 shifted=false score=103
```

Faithful cap-one does not choose one equal-ranked same-key route by depth,
branch order, insertion order, or hash alone. It GSS-merges or preserves
same-key equal-rank alternatives when the main GSS link can represent both
readings. That preserves the full-span `template_declaration [8510:8797]`
route through the boundary.

Frame 39 faithful cap-one control:

```text
rootHasError=false cRootHasError=false
(no structural divergence)
```

This is promising, but it is not sufficient to default the invariant: frame 1
still has `rootHasError=true`, token-frontier dispatch is still env-gated, and
the faithful run's max stack count is high (`maxStacks=147` on frame 1,
`maxStacks=438` on frame 39).

## Plain cap-one

`GOT_GLR_MAX_MERGE_PER_KEY=1` without faithful also clears the target, but it
introduces the earlier `ERROR "result[i] *="`:

```text
FIRST-DIFF @root[19][2][2][2][7]
go compound_statement [4485:4614] contains ERROR [4495:4507]
c  compound_statement [4485:4614]
```

The target-window trace runs mostly through the small merge path. It keeps a
single high-depth normal candidate at:

```text
key=(state=2282 byte=8802) cap=1
top=primitive_type[8799:8802] errRank=0 shifted=true score=103 depth=26
```

and repeatedly drops shallower same-key `state=2282` candidates:

```text
op=small-cap1-drop cmp=-1
```

It also retains `state=85 byte=8797` top-level repeat routes, including later
error-bearing repeats. Plain cap-one therefore clears the frame 1 trailing brace
by over-pruning the survivor set; the earlier `result[i] *=` regression is
evidence that this is accidental survivor ordering, not a safe generalized
invariant.

## Narrowed invariant

The target is controlled by same-key survivor preservation before final result
selection:

```text
Preserve/merge equal-ranked same-key GSS alternatives that differ only in the
represented branch links, instead of forcing them through depth/hash/branch
tie-breakers, when the alternatives can still represent distinct normal
continuations at the same state and byte offset.
```

Evidence:

- Default cap loses the normal route before result selection and recovers
  `function_definition [8532:8795]`.
- Cap 2 keeps enough alternatives for the normal `state=339 -> state=64 ->
  state=138` close-brace path to reduce `function_definition [8532:8797]`.
- Faithful cap-one explicitly GSS-merges/preserves equal-ranked
  `state=85 byte=8797` full-span alternatives.
- Plain cap-one clears only because it changes survivor pressure more
  aggressively, and it regresses an earlier expression statement into ERROR.

## Not implemented yet

I did not implement a parser behavior change in this slice. The exact safe
generalization is still falsifiable but not proven:

```text
Extend faithful GSS same-key preservation beyond the cap-one diagnostic path,
but only for comparator ties on accepted/error-rank/score/shifted and only when
the GSS main link can represent both alternatives without exceeding link
capacity.
```

Next probe:

1. Add a temporary experiment that enables GSS main-link merge/preserve for
   same-key comparator ties at default cap, before hash overflow replacement.
2. Re-run CUDA frame 1 and frame 39 controls.
3. Reject the invariant if it reproduces plain cap-one's earlier
   `result[i] *=` error, causes frame 39 structural divergence, or increases
   max-stack behavior beyond the faithful cap-one control without a target
   parity gain.

