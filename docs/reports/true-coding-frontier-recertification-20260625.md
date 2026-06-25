# True-Coding Frontier Recertification - 2026-06-25

Scope: correctness evidence only for the 206 classification goal. No parser, grammar, or normalizer source was edited.

Worktree: `/home/draco/work/gotreesitter-build-baseline`

Branch/head at start of this continuation: `repair/java-token-admission-reduced` at `7195a0fb add(docs): docs: add true-coding frontier recertification report`.

## Inputs

- N=1 206 smoke artifact: `cgo_harness/harness_out/tier_scan_parallel/current-206-smoke-20260625`
- Existing incomplete N=40 artifact: `cgo_harness/harness_out/tier_scan_parallel/current-tieriv-n40-20260625`
- New completed N=40 artifacts:
  - `cgo_harness/harness_out/tier_scan/nushell-true-coding-frontier-n40-20260625`
  - `cgo_harness/harness_out/tier_scan/teal-true-coding-frontier-n40-20260625`
  - `cgo_harness/harness_out/tier_scan/uxntal-true-coding-frontier-n40-20260625`
  - `cgo_harness/harness_out/tier_scan/powershell-true-coding-frontier-n40-20260625`
  - `cgo_harness/harness_out/tier_scan/disassembly-true-coding-frontier-n40-20260625`
  - `cgo_harness/harness_out/tier_scan/matlab-true-coding-frontier-n40-20260625`
- New completed available-corpus artifact:
  - `cgo_harness/harness_out/tier_scan/wat-true-coding-frontier-n40-20260625` (`GTS_TIER_SCAN_N=40`, corpus only yielded 34 selected files)
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

The same command shape was run for `teal`, `groovy`, `uxntal`, `wat`, `powershell`, `disassembly`, and `matlab`, changing `GTS_TIER_SCAN_LANGS` and the output directory. The `groovy` invocation was interrupted after repeated per-frame timeouts and does not have a completed aggregate N=40 row. The `wat` invocation used `GTS_TIER_SCAN_N=40`, but only 34 matching corpus files were selected, so it is recorded as completed available-corpus evidence rather than a full 40-file row.

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
| uxntal | completed new run | `files=40 diverge=40 trunc=6 errTree=40 panics=0`; parity `0/40`; median `580.48x`; aggregate `882.42x`; `failedFiles=0` | `runtime_frontier_stop` plus `recovery_error_cost`; stop reasons `accepted,no_stacks_alive,node_limit`; `frontierStops=36`; `acceptedDiv=136` | true-coding frontier remains runtime frontier/recovery |
| wat | completed available-corpus run | `files=34 diverge=30 trunc=0 errTree=29 panics=0`; parity `4/34`; median `17.01x`; aggregate `48.49x`; `failedFiles=1` | `terminal_timeout_or_fail` plus `recovery_error_cost`; terminal status `timeout:go_compare_reparse_start:rc=124` | true-coding frontier now has terminal timeout evidence on available corpus |
| powershell | completed new run | `files=40 diverge=15 trunc=5 errTree=11 panics=0`; parity `25/40`; median `6.61x`; aggregate `11.82x`; `failedFiles=0` | `runtime_frontier_stop` plus `recovery_error_cost`; stop reasons `accepted,no_stacks_alive`; `frontierStops=30`; `acceptedDiv=40` | true-coding frontier remains runtime frontier/recovery |
| disassembly | completed new run | `files=40 diverge=40 trunc=40 errTree=40 panics=0`; parity `0/40`; median `0.54x`; aggregate `101.14x`; `failedFiles=0` | `runtime_frontier_stop` plus `recovery_error_cost`; stop reasons `memory_budget,no_stacks_alive`; `frontierStops=240`; `acceptedDiv=0` | true-coding frontier remains runtime frontier/recovery |
| matlab | completed new run | `files=40 diverge=36 trunc=26 errTree=33 panics=0`; parity `4/40`; median `0.76x`; aggregate `0.35x`; `failedFiles=0` | `runtime_frontier_stop` plus `recovery_error_cost`; stop reasons `accepted,no_stacks_alive`; `frontierStops=156`; `acceptedDiv=40` | true-coding frontier remains runtime frontier/recovery |

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

This report now includes seven completed priority recertification rows, including five added in the continuation from `7195a0fb`:

- `nushell`: remains true-coding frontier with runtime frontier stop and recovery error cost evidence.
- `teal`: remains true-coding frontier, now with N=40 terminal timeout/fail evidence plus recovery error cost.
- `uxntal`: remains true-coding frontier with runtime frontier stop and recovery error cost evidence.
- `wat`: completed available-corpus coverage (`34/34` selected files), now with terminal timeout/fail evidence plus recovery error cost.
- `powershell`: remains true-coding frontier with runtime frontier stop and recovery error cost evidence.
- `disassembly`: remains true-coding frontier with runtime frontier stop and recovery error cost evidence.
- `matlab`: remains true-coding frontier with runtime frontier stop and recovery error cost evidence.

`groovy` was attempted but is incomplete. It has concrete timeout artifacts, but no completed aggregate row, so its N=40 classification remains unrecertified from this turn.
