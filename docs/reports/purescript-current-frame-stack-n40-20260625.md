# PureScript Current Frame Stack N=40, 2026-06-25

## Summary

Current HEAD `38e1154e9f599d9b9396fe56280dff2f51fc02d7` was scanned with the generalized frame-methodology fixes through the PureScript N=40 tier scan.

- Result: Tier IV, `parityMatch=7/40(18%)`, `diverge=33`, `trunc=0`, `errTree=1`, `panics=0`.
- Performance shape: `medianRatio=13.94x`, `aggRatio=20.60x`, `goNS=1918565034`, `cNS=93146900`.
- Isolation status: 40 isolated file runs, 0 failed files, every frame process exited `rc=0`, no timeouts.
- Docker status: wrapper `exit_code=0`, `oom_killed=false`, memory `8g`, CPUs `4`.

Compared with prior PureScript scans:

| Artifact | Parity | Diverge | Trunc | ErrTree | Notes |
| --- | ---: | ---: | ---: | ---: | --- |
| `purescript-post-relex-n40-20260625` | 1/40 | 39 | 0 | 1 | Frontier stop cleared; recovery residual remained. |
| `purescript-c-subtree-selection-n40-20260625` | 2/40 | 38 | 0 | 1 | Added `Boolean.purs` match over post-relex. |
| `purescript-current-frame-stack-n40-20260625` | 7/40 | 33 | 0 | 1 | Adds five more matches after generalized frame-methodology fixes. |

## Cleared Frames

The current scan matches these 7/40 frames:

- `Control/Category.purs`
- `Data/Boolean.purs`
- `Data/Bounded.purs`
- `Data/Field.purs`
- `Data/HeytingAlgebra.purs`
- `Data/NaturalTransformation.purs`
- `Data/Ordering.purs`

Relative to `purescript-c-subtree-selection-n40-20260625`, the newly cleared frames are `Category.purs`, `Bounded.purs`, `HeytingAlgebra.purs`, `NaturalTransformation.purs`, and `Ordering.purs`.

## Remaining Families

The diagnostic summary classifies PureScript as `recovery_error_cost` plus `accepted_divergence_cost`. There are no terminal failures, no runtime frontier stops, no truncation, and no token-accounting evidence.

Representative remaining first-diff families from frame logs:

| Family | Count | Representative |
| --- | ---: | --- |
| `signature` child-count `4/3` | 17 | `Control/Applicative.purs`, `root[26][2][1]`, `signature -> signature` |
| `patterns` child-count `3/1` | 10 | `Data/Monoid/Additive.purs`, `root[18][5][1]`, `patterns -> patterns` |
| `constraint` child-count `3/2` | 2 | `Data/BooleanAlgebra.purs`, `root[26][2][0][0]`, `constraint -> constraint` |
| `foreign_import` child-count `6/5` | 2 | `Data/Semigroup.purs`, `root[38]`, `foreign_import -> foreign_import` |
| expression type mismatch | 1 | `Data/Monoid.purs`, `root[39][5][2]`, `exp_apply -> exp_name` |
| root error-tree outlier | 1 | `Data/Ord.purs`, `root`, Go root child count `35` vs C root child count `96`, Go errors `76` vs C errors `0` |

This keeps the generalized target on recovery shape/cost and accepted structural divergence. The Semigroup `forall` mismatch remains report-only evidence and is not treated as a parser-code change request.

## Command And Status

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label purescript-current-frame-stack-n40-20260625 \
  --memory 8g --cpus 4 -- \
  'cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_TIER_SCAN_N=40 GTS_TIER_SCAN_ROUNDS=1 GTS_TIER_SCAN_TIMEOUT=300 GTS_TIER_SCAN_ISOLATE_FILES=1 GTS_TIER_SCAN_SKIP_TIER_PUBLISH=1 GTS_TIER_SCAN_LANGS=purescript bash cgo_harness/docker/run_tier_scan.sh cgo_harness/harness_out/tier_scan_parallel/purescript-current-frame-stack-n40-20260625'
```

- Scan output: `cgo_harness/harness_out/tier_scan_parallel/purescript-current-frame-stack-n40-20260625`
- Docker metadata: `harness_out/docker/20260625T182747Z-purescript-current-frame-stack-n40-20260625/metadata.txt`
- Docker inspect: `harness_out/docker/20260625T182747Z-purescript-current-frame-stack-n40-20260625/inspect.json`
- Status: `exit_code=0`, `oom_killed=false`, container state `exited`, Docker state exit code `0`.
