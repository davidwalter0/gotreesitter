# True-Coding Frontier Recertification - 2026-06-25

Scope: correctness evidence only for the 206 classification goal. No parser, grammar, or normalizer source was edited.

Worktree: `/home/draco/work/gotreesitter-build-baseline`

Branch/head at start: `repair/java-token-admission-reduced` at `6acd66c0 update(tier_scan, docs): Update PureScript residual ledger and scan report`.

## Inputs

- N=1 206 smoke artifact: `cgo_harness/harness_out/tier_scan_parallel/current-206-smoke-20260625`
- Existing incomplete N=40 artifact: `cgo_harness/harness_out/tier_scan_parallel/current-tieriv-n40-20260625`
- New completed N=40 artifacts:
  - `cgo_harness/harness_out/tier_scan/nushell-true-coding-frontier-n40-20260625`
  - `cgo_harness/harness_out/tier_scan/teal-true-coding-frontier-n40-20260625`
- New incomplete N=40 attempt:
  - `cgo_harness/harness_out/tier_scan/groovy-true-coding-frontier-n40-20260625`

Note: the prompt-cited `docs/reports/current-206-smoke-inventory-20260625.md` was not present in this worktree. The cited N=1 artifact directory was present and was used.

## Commands

Completed scans used the same settings:

```bash
GTS_CORPUS_DIR=/home/draco/work/gotreesitter-corpora/corpus_sources \
GTS_TIER_SCAN_N=40 \
GTS_TIER_SCAN_ROUNDS=1 \
GTS_TIER_SCAN_TIMEOUT=120 \
GTS_TIER_SCAN_KILL_AFTER=20s \
GTS_TIER_SCAN_ISOLATE_FILES=1 \
GTS_TIER_SCAN_SKIP_TIER_PUBLISH=1 \
GTS_TIER_SCAN_KEEP_PARSER_PROGRESS_ROWS=0 \
GTS_TIER_SCAN_MAX_FRAME_ROWS=2000 \
GTS_TIER_SCAN_LANGS=nushell \
bash cgo_harness/docker/run_tier_scan.sh cgo_harness/harness_out/tier_scan/nushell-true-coding-frontier-n40-20260625
```

The same command shape was run for `teal` and `groovy`, changing `GTS_TIER_SCAN_LANGS` and the output directory. The `groovy` invocation was interrupted after repeated per-frame timeouts and does not have a completed aggregate N=40 row.

For existing partial N=40 diagnostics, summaries were derived read-only with:

```bash
python3 cgo_harness/tier_scan/summarize_scan.py <worker-dir> --no-write --print
```

## Priority Frontier Rows

| grammar | N=40 status | MEASURE-DTIER evidence | diagnostic family / current frame | classification |
| --- | --- | --- | --- | --- |
| nushell | completed new run | `files=40 diverge=32 trunc=9 errTree=23 panics=0`; parity `8/40`; median `86.82x`; aggregate `245.79x`; `failedFiles=0` | `runtime_frontier_stop` plus `recovery_error_cost`; stop reasons `accepted,no_stacks_alive`; `frontierStops=54`; `acceptedDiv=92` | true-coding frontier remains runtime frontier/recovery |
| teal | completed new run | `files=40 diverge=15 trunc=0 errTree=23 panics=0`; parity `25/40`; median `326.74x`; aggregate `1226.50x`; `failedFiles=3` | `terminal_timeout_or_fail` plus `recovery_error_cost`; terminal statuses `timeout:go_compare_reparse_start:rc=124`, `timeout:go_parse_start:rc=124`, `timeout:go_parse_start:rc=124` | true-coding frontier now has terminal timeout evidence at N=40 |
| groovy | attempted; incomplete | no aggregate row; command exited `130` after frame 11 startup | frame 2 timeout at `measure-groovy-external-frame-0002.log`, `go_parse_start`, `rc=124`; frame 8 timeout at `measure-groovy-external-frame-0008.log`, `go_parse_start`, `rc=124` | incomplete; do not recertify from this run |
| uxntal | not rerun this turn | N=1 smoke only: `files=1 diverge=1 trunc=1 errTree=0 panics=0` | N=1 `runtime_frontier_stop` | needs N=40 |
| wat | not rerun this turn | N=1 smoke only: `files=1 diverge=1 trunc=1 errTree=1 panics=0` | N=1 `runtime_frontier_stop` plus `recovery_error_cost` | needs N=40 |
| powershell | not rerun this turn | N=1 smoke only: `files=1 diverge=1 trunc=1 errTree=0 panics=0` | N=1 `runtime_frontier_stop` | needs N=40 |
| disassembly | not rerun this turn | N=1 smoke only: `files=1 diverge=1 trunc=1 errTree=1 panics=0` | N=1 `runtime_frontier_stop` plus `recovery_error_cost` | needs N=40 |
| matlab | not rerun this turn | N=1 smoke only: `files=1 diverge=1 trunc=1 errTree=1 panics=0` | N=1 `runtime_frontier_stop` plus `recovery_error_cost` | needs N=40 |

## Existing Partial N=40 Rows

The existing `current-tieriv-n40-20260625` run is incomplete. It should not be treated as full coverage, but these aggregate rows are usable as completed row evidence.

| grammar | files | parity | signature | diagnostic family / current frame |
| --- | ---: | --- | --- | --- |
| asm | 40 | `0/40` | `diverge=40 trunc=34 errTree=40 panics=0` | `runtime_frontier_stop`, `recovery_error_cost` |
| ebnf | 40 | `0/40` | `diverge=40 trunc=40 errTree=16 panics=0` | `runtime_frontier_stop`, `recovery_error_cost` |
| blade | 40 | `24/40` | `diverge=16 trunc=16 errTree=8 panics=0` | `runtime_frontier_stop`, `recovery_error_cost` |
| eds | 1 | `0/1` | `diverge=1 trunc=1 errTree=0 panics=0` | `runtime_frontier_stop` |
| brightscript | 40 | `11/40` | `diverge=29 trunc=29 errTree=0 panics=0` | `runtime_frontier_stop` |
| facility | 4 | `1/4` | `diverge=3 trunc=3 errTree=1 panics=0` | `runtime_frontier_stop`, `recovery_error_cost` |
| nginx | 1 | `0/1` | `diverge=1 trunc=0 errTree=0 panics=0` | `accepted_shape_materialization`, `accepted_divergence_cost` |
| caddy | 40 | `11/40` | `diverge=29 trunc=28 errTree=8 panics=0` | `runtime_frontier_stop`, `recovery_error_cost` |
| fennel | 40 | `15/40` | `diverge=25 trunc=9 errTree=18 panics=0` | `runtime_frontier_stop`, `recovery_error_cost` |
| norg | 2 | `0/2` | `diverge=2 trunc=2 errTree=0 panics=0` | `runtime_frontier_stop` |
| cairo | 40 | `1/40` | `diverge=39 trunc=20 errTree=31 panics=0` | `runtime_frontier_stop`, `recovery_error_cost` |
| cobol | 40 | `0/40` | `diverge=40 trunc=33 errTree=0 panics=0`; `failedFiles=7` | `terminal_timeout_or_fail` |
| commonlisp | 40 | `32/40` | `diverge=8 trunc=6 errTree=5 panics=0` | `runtime_frontier_stop`, `recovery_error_cost`, `scanner_token_accounting` |
| cooklang | 3 | `1/3` | `diverge=2 trunc=0 errTree=3 panics=0` | `recovery_error_cost`, `accepted_divergence_cost` |
| haxe | 40 | `10/40` | `diverge=30 trunc=21 errTree=28 panics=0` | `runtime_frontier_stop`, `recovery_error_cost` |

## Result

This turn adds two completed priority N=40 recertification rows:

- `nushell`: remains true-coding frontier with runtime frontier stop and recovery error cost evidence.
- `teal`: remains true-coding frontier, now with N=40 terminal timeout/fail evidence plus recovery error cost.

`groovy` was attempted but is incomplete. It has concrete timeout artifacts, but no completed aggregate row, so its N=40 classification remains unrecertified from this turn.
