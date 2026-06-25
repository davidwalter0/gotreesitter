# C-recovery pop-slice reduction version retention

Date: 2026-06-25

## Question

CUDA still diverges at `UnifiedMemoryStreams.cu`:

- Go: `ERROR [8510:8585]` plus `compound_statement [8586:8797]`
- C: `template_declaration [8510:8797]`

The suspected mismatch was that `cDoAllPotentialReductions` reduced one viable
merged-GSS pop slice but discarded sibling reduced slices left in
`pendingForkStacks` before `cHandleError` could push the error discontinuity and
record summaries.

## Proof

Added the grammar-neutral regression
`TestCDoAllPotentialReductionsRetainsMergedPopSliceForks`. It builds a synthetic
merged GSS where one reduce action has two viable pop slices.

Current behavior before the implementation change failed as expected:

```text
go test . -run '^TestCDoAllPotentialReductionsRetainsMergedPopSliceForks$' -count=1
--- FAIL: TestCDoAllPotentialReductionsRetainsMergedPopSliceForks (0.00s)
    parser_recover_c_test.go:96: version count = 1, want reduced current plus sibling
FAIL
```

The expected C-faithful behavior is:

- no shift on the current lookahead,
- one reduced version renumbered into the current slot,
- the sibling reduced pop slice retained as another surviving version,
- `pendingForkStacks` drained, not rejected.

This matches upstream `parser.c`: `ts_parser__reduce` creates/retains versions
for merged-stack pop slices, and
`ts_parser__do_all_potential_reductions` can renumber the returned reduction
version into the active version.

## Fix

`cDoAllPotentialReductions` now drains `pendingForkStacks` into its local
`versions` slice after applying a reduce action. This is scoped to C-recovery
reduction mode. The normal `rejectUndrainedPendingForkStacks` safety path remains
unchanged for non-C-recovery parser paths.

No per-grammar normalizer, language-name policy, byte-span policy,
corpus-specific rule, or performance change was added.

## Verification

Host:

```text
go test . -run '^TestCDoAllPotentialReductionsRetainsMergedPopSliceForks$' -count=1
ok  	github.com/odvcencio/gotreesitter	0.004s

go test . -run 'TestC' -count=1
ok  	github.com/odvcencio/gotreesitter	0.261s
```

Docker CUDA witness:

```text
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label c-recovery-pop-slice-retention-cuda-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8480:8810 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Result:

```text
--- PASS: TestFirstDiffDiag (3.11s)
PASS
ok  	github.com/odvcencio/gotreesitter/cgo_harness	5.524s
artifacts: /home/draco/work/gotreesitter-build-baseline/harness_out/docker/20260625T220547Z-c-recovery-pop-slice-retention-cuda-20260625
oom_killed: false
```

The CUDA witness did not move. It still reports the same root first diff:

```text
go.child[23]: type="ERROR" [8510:8585]
go.child[24]: type="compound_statement" [8586:8797]
c.child[23]: kind="template_declaration" [8510:8797]
```

## Result

The suspected generalized C-fidelity mismatch was real and is fixed in the
synthetic C-recovery frame. The CUDA witness remains unresolved, so the next
frame should continue after retained pop-slice versions enter recovery summary
and strategy-1 election.

## Review repair

The first repair over-retained same-pop-target sibling slices as independent
local versions. Upstream C does not do that: `stack__add_slice` assigns slices
that pop to the same stack node to the same stack version, and
`ts_parser__reduce` merges reduced versions back into earlier equivalent
headers before returning the first new version.

The follow-up repair changes only C-recovery's local
`cDoAllPotentialReductions` drain. Pending sibling reductions are now appended
only when they survive same-pop-target/header-equivalence collapse as distinct
C versions; same-pop-target duplicates collapse to one local version. The
normal parser pending-fork drain/reject paths remain unchanged.

Focused grammar-neutral coverage was split into:

- same-pop-target duplicate pop slices collapse to one C version,
- distinct pop-target sibling slices remain separate versions,
- multiple reduce actions overwrite `reduction_version` with the last action's
  first surviving new version.

CUDA witness status: a bounded review-repair rerun still did not move the
witness. `TestFirstDiffDiag` passed as a diagnostic and still reported the same
root first diff at `UnifiedMemoryStreams.cu` `[8510:8797]`: Go has
`ERROR [8510:8585]` plus `compound_statement [8586:8797]`, while C has
`template_declaration [8510:8797]`.

```text
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label c-recovery-pop-slice-review-repair-cuda-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8480:8810 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

```text
--- PASS: TestFirstDiffDiag (2.83s)
PASS
ok  	github.com/odvcencio/gotreesitter/cgo_harness	5.301s
artifacts: /home/draco/work/gotreesitter-build-baseline/harness_out/docker/20260625T222049Z-c-recovery-pop-slice-review-repair-cuda-20260625
oom_killed: false
```

## Forward repair

Reviewer blockers found two remaining C-equivalence gaps in the review repair:

- `cDoAllPotentialReductions` retained the last successful Go reduction, but
  upstream C overwrites `reduction_version` for every reduce action, including
  `STACK_VERSION_NONE`.
- same-pop-target slices were collapsed too late by merging already-pushed
  parent links, which preserved multiple reduced children alternatives that C's
  `stack__add_slice` / `ts_parser__select_children` flow reduces to one.

The forward repair keeps the behavior grammar-neutral:

- each reduce action now records its raw action-local reduction version, and a
  later no-new-version action clears the earlier renumber candidate;
- same-pop-target sibling slices collapse only within the current reduce action,
  before the normal header-equivalence merge path;
- distinct pop targets that reduce to equal top state/byte/error-cost still
  merge through the C `ts_stack_merge`-style path, while non-equivalent distinct
  pop targets remain separate.

Focused coverage was strengthened with synthetic GSS tests for:

- same-pop-target collapse leaving a single reduced top link;
- a later no-new-version reduce preventing renumbering of an earlier reduction;
- header-equivalent distinct pop targets merging into one version with two top
  links;
- non-equivalent distinct pop targets remaining separate versions.

No bounded CUDA witness was rerun for this forward repair. The changed behavior
is fully covered by grammar-neutral synthetic parser/GSS tests, and no CUDA
runtime or language-specific path participates in the repaired helpers.

## Same-pop C-fidelity repair

The review follow-up found three remaining generalized C-fidelity gaps:

- same-pop reduction collapse chose the survivor with whole-stack error cost and
  stack score instead of the temporary parent subtree ordering used by C's
  `ts_parser__select_children`;
- candidates from one reduce action could merge into older header-equivalent
  versions before all same-pop siblings from that action had been collapsed;
- trailing extras could hide the original pop target because Go pushes parent
  then extras, while C removes trailing extras before selecting children and
  replays only the selected extras afterward.

The repair stays local to the generalized C-recovery machinery:

- `cDoAllPotentialReductions` now batches all candidates from one reduce action,
  collapses same-pop candidates action-locally, then performs generic
  header-equivalence merge;
- same-pop collapse identifies the reduced parent by walking past replayed
  trailing extras and compares the original pop target, not the final pushed
  stack header or byte offset;
- the parent survivor is selected by parent subtree error cost, positive-error
  replacement behavior, then recursive symbol/child-count node order. Flags are
  intentionally excluded to match C's `ts_subtree_compare`. This baseline's
  `Node` model does not retain subtree dynamic precedence, so that C tie-break
  remains a documented C-fidelity residual rather than a broad layout change in
  this repair.

Focused coverage was added for:

- same-pop child-array survivor selection;
- same-pop alternatives with different selected child spans/final byte offsets
  collapsing by original pop target;
- trailing-extra same-pop collapse and selected-extra replay;
- collapse-before-older-merge ordering;
- flag-only subtree differences not affecting survivor selection.

Host verification:

```text
go test . -run 'TestCDoAllPotentialReductions|TestCCollectPotentialReductions|TestCAppendActionReductions' -count=1
ok  	github.com/odvcencio/gotreesitter	0.017s
```
