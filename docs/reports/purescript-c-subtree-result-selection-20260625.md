# PureScript C-Subtree Result Selection, 2026-06-25

Scope: post-relex PureScript frame methodology target. No grammar-specific
normalizer, language-name policy, or PureScript-specific parser branch was
added.

## Target

Primary frame:
`/home/draco/work/gotreesitter-corpora/corpus_sources/purescript/src/Data/Boolean.purs`

Baseline artifact:
`cgo_harness/harness_out/tier_scan_parallel/purescript-post-relex-n40-20260625/measure-purescript-external-frame-0007.log`

Baseline first diff:

```text
firstDiffPath="root[10][2]" goType="exp_apply" cType="exp_name" goSpan=215:219 cSpan=215:219 goRootErr=false cRootErr=false goErrors=0 cErrors=0 goStop="accepted"
```

## Diagnosis

Plain first-diff replay reproduced the target exactly:

```sh
cd cgo_harness
REPRO_LANG=purescript REPRO_FILE=/home/draco/work/gotreesitter-corpora/corpus_sources/purescript/src/Data/Boolean.purs \
  go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v
```

`REPRO_GLR_TRACE=1` showed the decisive fork at state `283` before EOF:
`_exp_apply(187)` and `_fexp(188)` both survived to accepted stacks. The
Boolean failure was therefore not token accounting, scanner behavior, recovery
cost, or forest materialization. It was final accepted-stack election.

The C runtime source in the `go-tree-sitter` dependency uses
`ts_parser__select_tree`: after error cost and dynamic precedence ties, it calls
`ts_subtree_compare`, which compares subtree symbol ids and child counts
lexicographically. Go result selection instead fell through to alias/depth/order
tie-breakers before any C-style subtree order.

## Fix

`parser_result.go` now applies a C-style subtree comparison for accepted stack
selection when dynamic scores tie. The comparator walks stack-entry payloads
without materializing normal nodes or pending parents, compares symbol ids and
child counts in the same order as `ts_subtree_compare`, and then falls back to
the existing alias/score/depth/order tie-breakers.

The alias-target invariant remains intact for non-equal scores; this is covered
by the existing alias-selection test plus the new score-tie subtree-order test.

## Validation

Focused tests:

```sh
go test . -run 'TestBuildResultFromGLR|Test.*Result|Test.*GLR|Test.*Reduce|Test.*Stack' -count=1
```

Result: pass.

Boolean first-diff replay after the fix:

```text
go stopReason=accepted ... rootHasError=false cRootHasError=false
(no structural divergence)
```

Repeated witness replay:

```sh
cd cgo_harness
REPRO_LANG=purescript REPRO_FILE=/home/draco/work/gotreesitter-corpora/corpus_sources/purescript/src/Data/HeytingAlgebra.purs \
  go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v
```

Result: still divergent:

```text
FIRST-DIFF @root[38][5][2]
go: type="exp_apply" [1501:1506] cc=1
c : kind="exp_name" [1501:1506] cc=1
```

The Heyting trace showed the `_exp_apply`/`_fexp` alternatives are pruned before
final result selection when twelve same-key stacks collapse to six. That points
to the next generalized layer: arena-aware merge/cull election for pending-parent
stack payloads, not final result selection.

Docker N=40 PureScript classification:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label purescript-c-subtree-selection-n40-20260625 \
  --memory 8g --cpus 4 -- \
  'cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_TIER_SCAN_N=40 GTS_TIER_SCAN_ROUNDS=1 GTS_TIER_SCAN_TIMEOUT=300 GTS_TIER_SCAN_ISOLATE_FILES=1 GTS_TIER_SCAN_SKIP_TIER_PUBLISH=1 GTS_TIER_SCAN_LANGS=purescript bash cgo_harness/docker/run_tier_scan.sh cgo_harness/harness_out/tier_scan_parallel/purescript-c-subtree-selection-n40-20260625'
```

Result:

- exit code 0, `oom_killed=false`
- `parityMatch=2/40(5%)`, `diverge=38`, `trunc=0`, `errTree=1`, `panics=0`
- `Boolean.purs` frame 0007: `comparison_result result=match`
- `HeytingAlgebra.purs` frame 0020: still `exp_apply` vs `exp_name`
- `Semigroup.purs` frame 0037: still `exp_apply` vs `exp_name`

## Status

The frame methodology found a generalized final-result election invariant and
the narrow fix clears the Boolean target. The broader repeated family remains a
generalized merge/cull-election target because the losing `_fexp` branch is
discarded before final result selection in larger files.
