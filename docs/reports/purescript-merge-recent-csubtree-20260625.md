# PureScript Merge Recent C-Subtree Election

Date: 2026-06-25

## Witnesses

- `/home/draco/work/gotreesitter-corpora/corpus_sources/purescript/src/Data/HeytingAlgebra.purs`
- `/home/draco/work/gotreesitter-corpora/corpus_sources/purescript/src/Data/Semigroup.purs`

Starting HEAD:

```text
21dd514096a1201b6c36f3541c8c3e6bb8c1887e fix(glr): Fix GLR result selection to use C-style subtree order
```

Baseline first-diff replays on that HEAD reproduced the known accepted/full-span/no-root-error divergence:

```text
HeytingAlgebra.purs: FIRST-DIFF @root[38][5][2]
  go: type="exp_apply" [1501:1506] cc=1
  c : kind="exp_name" [1501:1506] cc=1

Semigroup.purs: FIRST-DIFF @root[30][5][2]
  go: type="exp_apply" [1267:1279] cc=1
  c : kind="exp_name" [1267:1279] cc=1
```

Commands:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label purescript-heyting-firstdiff-head -- \
  "cd /workspace/cgo_harness && REPRO_LANG=purescript REPRO_FILE=/workspace/corpus_sources/purescript/src/Data/HeytingAlgebra.purs go test . -tags treesitter_c_parity -run '^TestFirstDiffDiag$' -count=1 -v"

bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label purescript-semigroup-firstdiff-head -- \
  "cd /workspace/cgo_harness && REPRO_LANG=purescript REPRO_FILE=/workspace/corpus_sources/purescript/src/Data/Semigroup.purs go test . -tags treesitter_c_parity -run '^TestFirstDiffDiag$' -count=1 -v"
```

## Root Cause

Temporary bounded merge tracing around the HeytingAlgebra witness span pinned the loss to `mergeStacksWithScratch` per-key overflow, not exact dedupe, global cull, later action application, or final accepted-stack selection.

The relevant same-key payloads were both accepted/no-error candidates for the same `(state, byteOffset)` key:

```text
phase=per-key-overflow key=(state=7916 byte=1506)
candidate window: function[1496:1506] -> exp_name[1501:1506]
incumbent window: function[1496:1506] -> exp_apply[1501:1506]
```

`stackCompareMerge` ranked score, shifted, depth, byte offset, and branch order, but did not consult the C-style subtree ordering that final result selection now uses. On deeper same-key stacks, the existing final-result helper also returned `0` because it is root/result-entry oriented and capped for final accepted stack selection. The merge/cull election needed a stack-frontier comparison over recent materializing payloads.

## Fix

`stackCompareMerge` now applies C-style subtree ordering to the most recent materializing stack entries before shifted/depth/offset/branch tie-breakers. The merge scratch carries the current arena so pending-parent payload children can be compared without materializing when available.

The fix is generalized parser machinery only:

- no language-name switches
- no grammar normalizers
- no PureScript-specific policy

Focused tests added:

- `TestStackCompareMergePrefersRecentCSubtreeOrderBeforeDepth`
- `TestMergeStacksPerKeyOverflowUsesRecentCSubtreeOrder`

## Validation

Focused local unit slice:

```sh
go test . -run '^Test(StackCompareMergePrefersRecentCSubtreeOrderBeforeDepth|MergeStacksPerKeyOverflowUsesRecentCSubtreeOrder|MergeStacks|StackComparePtr)' -count=1
```

Result: pass.

Docker-scoped unit slice:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --label merge-recent-csubtree-unit -- \
  "cd /workspace && go test . -run '^Test(StackCompareMergePrefersRecentCSubtreeOrderBeforeDepth|MergeStacksPerKeyOverflowUsesRecentCSubtreeOrder|MergeStacks|StackComparePtr)' -count=1"
```

Result: pass, `oom_killed=false`.

Witness replays after the fix:

```text
HeytingAlgebra.purs:
  exp_apply vs exp_name cleared.
  New first diff: @root[39][10][1][0][0]
    go: type="_" [1743:1744] cc=0
    c : kind="pat_wildcard" [1743:1744] cc=1

Semigroup.purs:
  exp_apply vs exp_name cleared.
  New first diff: @root[32][5][1][0][0]
    go: type="_" [1405:1406] cc=0
    c : kind="pat_wildcard" [1405:1406] cc=1
```

Commands:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label purescript-heyting-firstdiff-final -- \
  "cd /workspace/cgo_harness && REPRO_LANG=purescript REPRO_FILE=/workspace/corpus_sources/purescript/src/Data/HeytingAlgebra.purs go test . -tags treesitter_c_parity -run '^TestFirstDiffDiag$' -count=1 -v"

bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label purescript-semigroup-firstdiff-final -- \
  "cd /workspace/cgo_harness && REPRO_LANG=purescript REPRO_FILE=/workspace/corpus_sources/purescript/src/Data/Semigroup.purs go test . -tags treesitter_c_parity -run '^TestFirstDiffDiag$' -count=1 -v"
```

Runtime sanity command:

```sh
GOMAXPROCS=1 go test . -run '^$' -bench 'BenchmarkGoParseFullDFA|BenchmarkGoParseIncrementalSingleByteEditDFA|BenchmarkGoParseIncrementalNoEditDFA' -benchmem -count=10 -benchtime=750ms
```

Result: pass. Allocations stayed flat for the benchmark trio:

```text
BenchmarkGoParseFullDFA: 889-890 B/op, 8 allocs/op
BenchmarkGoParseIncrementalSingleByteEditDFA: 240 B/op, 4 allocs/op
BenchmarkGoParseIncrementalNoEditDFA: 48 B/op, 1 alloc/op
```

## N=40 Status

PureScript N=40 was not run after this fix because both primary witnesses still have a new structural residual. The requested `_fexp`/`exp_apply` merge-cull layer is cleared, but the files are not parity matches yet.

Next residual:

```text
Go wildcard leaf "_" vs C parent "pat_wildcard" containing "_"
```

That residual appears to be a separate alias/materialization shape issue, not the same same-key merge/cull election failure.
