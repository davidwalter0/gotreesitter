# C-Recovery Merged Summary Order - 2026-06-25

Scope: generalized parser machinery only. No grammar-name, language-name, or
corpus-specific normalizer was added.

## Finding

The prior Go grouped recovery summary election was not fully equivalent to C's
merged multi-link stack summary traversal.

C does not just scan one linear summary per potential-reduction version. In
`ts_parser__handle_error`, it pushes `ERROR_STATE` on every version, merges the
versions into one stack version, then calls `ts_stack_record_summary` once.
During `stack_node_add_link`, incoming links can recursively merge below the
ERROR head when the link payload is equivalent and the child stack node has the
same state, byte position, and error cost. `stack__iter` then walks the merged
graph with its iterator queue ordering: link 0 stays in-place while later links
are cloned to the end of the queue.

That creates a generalized ordering case the previous Go loop could not model:
if paths A and B merge below the ERROR link and path C remains a separate top
link, the next depth is visited as A, C, B. The old Go depth/member scan visited
A, B, C because it had only independent per-stack summaries.

## Patch

Implemented a local merged-summary graph for `cRecGroup`:

- builds summary paths from the absorbing stacks after the ERROR discontinuity;
- applies C-shaped recursive link insertion and node merge checks;
- records summary entries using the same breadth-first iterator queue order and
  same backward `(depth,state)` dedupe rule;
- stores each merged summary entry's owning Go stack member for strategy-1
  `cRecoverToState`.

The existing per-stack `cRecordSummary` remains for compatibility/diagnostics,
but `cRecoverStrategy1Election` now consumes `group.mergedSummary`.

Added a grammar-neutral unit test:

- `TestCBuildMergedGroupSummaryMatchesStackIterDelayedBranchOrder`

It constructs synthetic recovery paths that expose the A, C, B delayed-branch
order without relying on CUDA, C++, grammar names, or corpus paths.

## CUDA Result

Command:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label cuda-summary-order-final \
  --memory 8g \
  --cpus 4 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && env GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8480:8810 GOT_PARSE_PROGRESS=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Result:

- exit `0`
- `oom_killed=false`
- Go accepted and did not truncate:
  `stopReason=accepted runtime=truncated=false ... lastTokenEnd=12254 expectedEOF=12254`
- residual unchanged:
  - Go root `translation_unit` child count `26`
  - C root `translation_unit` child count `25`
  - Go still has `ERROR [8510:8585]` plus
    `compound_statement [8586:8797]`
  - C still has `template_declaration [8510:8797]`

Artifacts:

- `harness_out/docker/20260625T204341Z-cuda-summary-order-final`

## Focused Verification

Host, narrow:

```bash
go test . -run 'TestC' -count=1
```

Result:

```text
ok  	github.com/odvcencio/gotreesitter	0.325s
```

Docker, one grammar/file:

```text
ok  	github.com/odvcencio/gotreesitter/cgo_harness	5.905s
```

## Conclusion

The merged-summary traversal/dedupe mismatch was real and generalized, so the
parser machinery was patched. The patch does not improve or regress the CUDA
witness; it remains accepted/non-truncated with the same local template-body
split.

## Next Frame

The next target should move past summary traversal order and inspect the actual
strategy-1 handoff around the first discontinuity:

- C can still produce `template_declaration [8510:8797]` while Go elects or
  materializes a path that leaves `ERROR [8510:8585]` and a sibling
  `compound_statement [8586:8797]`.
- The trace after the patch still shows early viable entries at the template
  token rejected by cost competition or lacking actions, then later reductions
  build the split shape.
- Next evidence should compare C `ts_stack_pop_count` / `ts_parser__recover_to_state`
  fan-out and Go `cRecoverToState` materialization for the elected entries,
  including whether one C recovery creates multiple stack slices where Go
  currently tries one owner member.
