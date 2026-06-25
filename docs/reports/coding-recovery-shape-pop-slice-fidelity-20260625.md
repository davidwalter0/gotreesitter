# Coding Recovery-Shape Pop-Slice Fidelity - 2026-06-25

Scope: generalized C-recovery stack behavior only. No grammar-specific
normalizer, language-name branch, byte-span policy, or corpus-specific rule was
added.

## Finding

Upstream `ts_parser__recover_to_state` calls `ts_stack_pop_count`, which walks
the merged stack graph with `stack__iter`. For one elected recovery
`(depth,state)`, C keeps every pop slice whose resulting stack state equals the
goal state, then materializes the recovered ERROR/trailing extras on each
slice.

The Go port previously used the elected `memberID` to recover from one linear
owner stack. That was correct for selecting a stable live member after
`group.mergedSummary`, but it was not fully faithful when that member's GSS
head contained merged `extraLinks`: sibling pop slices at the same
`(depth,state)` were not enumerated.

## Change

`cRecoverToState` now has a pop-slice enumerator over the selected member's GSS
links:

- depth counting uses the same generalized rule as C recovery summaries:
  non-extra subtrees and NULL error discontinuities count; extras do not;
- only slices for the already elected depth and goal state are materialized;
- duplicate resulting GSS nodes are deduped, matching C's effective
  `previous_version` cleanup after `ts_stack__add_slice`;
- recovered forks reuse the existing generic ERROR/trailing-extra
  materialization path.

A grammar-neutral unit test constructs a synthetic merged GSS and verifies that
two viable pop slices at the same elected depth/state produce two recovered
ERROR forks with distinct spans.

## CUDA Witness

Command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --memory 8g --cpus 4 \
  --label c-rec-pop-slice-fidelity-cuda-20260625 -- \
  "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/c-rec-pop-slice-fidelity-20260625 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8480:8810 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/c-rec-pop-slice-fidelity-20260625/cuda-unified-memory-streams.log"
```

Result: pass, Docker exit `0`, `oom_killed=false`, max RSS `1216948 kB`.

The CUDA residual did not move:

- Go child `23`: `ERROR [8510:8585]`
- Go child `24`: `compound_statement [8586:8797]`
- C child `23`: `template_declaration [8510:8797]`

The run remained accepted and non-truncated:

```text
go stopReason=accepted runtime=truncated=false ... rootHasError=true cRootHasError=false
```

## Host Validation

```sh
go test . -run '^(TestCRecoverToStateEnumeratesMergedPopSlices|TestCBuildMergedGroupSummaryMatchesStackIterDelayedBranchOrder|TestCRecoverGroupMemberIndexSurvivesStackReorder|TestCDoAllPotentialReductionsRejectsUndrainedFaithfulForks|TestParseCRecoveryTraceWindow)$' -count=1
go test . -run 'TestC' -count=1
```

Results:

```text
ok  	github.com/odvcencio/gotreesitter	0.004s
ok  	github.com/odvcencio/gotreesitter	0.275s
```

## Conclusion

The old single-linear-path `cRecoverToState` was not fully C-faithful for merged
GSS pop slices. The patch closes that generic fidelity gap, but the CUDA
`template_declaration` residual is unchanged. The next frame should move
earlier than `recover_to_state`: inspect how `cDoAllPotentialReductions`
versions that can form `template_declaration [8510:8797]` are retained,
renumbered, merged, or made available to the recovery election before the
error-state summary is recorded.
