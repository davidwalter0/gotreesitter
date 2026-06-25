# PureScript Post-Relex N=40 Classification, 2026-06-25

Scope: correctness/classification only after `31de3c48` and `785e279f`. No parser edits, per-grammar normalizers, language-name policy, or performance work.

## Run

Artifact:
`cgo_harness/harness_out/tier_scan_parallel/purescript-post-relex-n40-20260625`

Docker wrapper artifact:
`harness_out/docker/20260625T171433Z-purescript-post-relex-n40-20260625`

Command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label purescript-post-relex-n40-20260625 --memory 8g --cpus 4 -- 'cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_TIER_SCAN_N=40 GTS_TIER_SCAN_ROUNDS=1 GTS_TIER_SCAN_TIMEOUT=300 GTS_TIER_SCAN_ISOLATE_FILES=1 GTS_TIER_SCAN_SKIP_TIER_PUBLISH=1 GTS_TIER_SCAN_LANGS=purescript bash cgo_harness/docker/run_tier_scan.sh cgo_harness/harness_out/tier_scan_parallel/purescript-post-relex-n40-20260625'
```

Run status: exit code 0, `oom_killed=false`, tier publication skipped.

## Result

PureScript remains Tier IV, but its residual lane changed.

| Evidence | Value |
| --- | --- |
| Files | 40 |
| Parity | 1/40 |
| Divergence | 39/40 |
| Truncation | 0 |
| Error-tree aggregate | 1 |
| Panics | 0 |
| Frontier stops | 0 |
| Runtime stop reasons | `accepted` |
| Aggregate ratios | median `4.40x`, aggregate `5.52x` |

Primary files inspected:

- `tier_scan.txt`: `parityMatch=1/40(2%) diverge=39 trunc=0 errTree=1 panics=0`.
- `tier_iv.txt`: `purescript parityMatch=1/40(2%)`.
- `clean.txt`: empty.
- `diagnostic_summary.md/json`: diagnostic family `recovery_error_cost`; `frontierStopCount=0`; `stopReasons=["accepted"]`; `acceptedDivergenceCount=156`; comparison diffs `child-count` and `type`; root pair `purescript->purescript`.

Representative frames:

- `measure-purescript-external-frame-0001.log` (`Control/Applicative.purs`): accepted full-span parse, `truncated=false stopReason=accepted`, then root child-count divergence.
- `measure-purescript-external-frame-0016.log` (`Data/Field.purs`): the one clean frame, `comparison_result result=match`.
- `measure-purescript-external-frame-0020.log` (`Data/HeytingAlgebra.purs`): accepted full-span parse, type divergence `exp_apply` vs `exp_name`, high node/stack activity but no frontier stop.
- `measure-purescript-external-frame-0031.log` (`Data/Ord.purs`): post-fix former frontier witness is now accepted/non-truncated and contributes the lone `errTree` aggregate.

## Before/After

Previous N=40 accepted-shape evidence from `coding-accepted-shape-n40-20260625` classified PureScript as a frontier survival target:

| Artifact | Parity | Diverge | Trunc | ErrTree | Family | Next target |
| --- | ---: | ---: | ---: | ---: | --- | --- |
| `coding-accepted-shape-n40-20260625` | 1/40 | 39 | 1 | 0 | `runtime_frontier_stop` | `glr_frontier_survival_and_reuse_selection` |
| `purescript-post-relex-n40-20260625` | 1/40 | 39 | 0 | 1 | `recovery_error_cost` | `generalized_recovery_shape_and_cost` |

The stale-token frontier failure is cleared at N=40. The current residual is accepted, non-truncated structural divergence with recovery/error-cost evidence, not an active GLR frontier survival failure.

## Ledger Impact

The PureScript overlay in `cgo_harness/tier_scan/generate_206_residual_ledger.py` is updated from `glr_frontier_survival_and_reuse_selection` to `generalized_recovery_shape_and_cost`, and the generated 206 residual ledger files are regenerated.

This is still a PureScript-only N=40 classification result. It does not claim full 206 completion or Tier IV removal.

## Next Generalized Target

Investigate generalized recovery shape and cost after successful current-state relex: accepted-stack election, missing-token/error-node insertion, and structural materialization differences that produce child-count/type divergences under full-span accepted parses.
