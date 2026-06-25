# Coding Accepted-Shape N=40 Reclassification, 2026-06-25

This is classification-only report plumbing for the 206 residual ledger. It does not change parser behavior, grammar normalizers, Docker harness behavior, or performance conclusions.

Evidence source: `cgo_harness/harness_out/tier_scan_parallel/coding-accepted-shape-n40-20260625/merged/diagnostic_summary.md`.

## Result

The N=40 evidence reclassifies the five true-coding accepted/materialization seeds as mixed residuals, not pure accepted materialization:

| Grammar | N=40 evidence | Updated generalized target |
| --- | --- | --- |
| fennel | 15/40 parity, 25 diverge, 9 trunc, 18 errTree; `runtime_frontier_stop` plus `recovery_error_cost` | `generalized_recovery_shape_and_cost` |
| fsharp | 0/40 parity, 29 trunc, 12 errTree; terminal `timeout:c_parse_start` plus `recovery_error_cost` | `generalized_recovery_shape_and_cost` |
| perl | 0/40 parity, 7 trunc, 7 errTree; `runtime_frontier_stop` plus `recovery_error_cost` | `generalized_recovery_shape_and_cost` |
| purescript | 1/40 parity, 39 diverge, 1 trunc, 0 errTree; `runtime_frontier_stop` only | `glr_frontier_survival_and_reuse_selection` |
| swift | 0/40 parity, 27 trunc, 9 errTree; `runtime_frontier_stop` plus `recovery_error_cost` | `generalized_recovery_shape_and_cost` |

F# includes terminal C-oracle timeout/fail evidence. That lifecycle signal is not parser materialization evidence and should remain separate from parser recovery/cost classification.

## Policy

This reclassification supports generalized machinery work only: GLR frontier survival and reuse selection, plus generalized recovery shape and cost handling where error-tree evidence is present. It does not support per-grammar normalizers, language-name parser policy, or performance claims.
