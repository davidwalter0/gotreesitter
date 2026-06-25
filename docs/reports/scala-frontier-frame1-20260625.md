# Scala Frontier Frame 1, 2026-06-25

## Scope

Report-only bounded correctness diagnostic for Scala frame 1 from the current
206 smoke:
`/workspace/corpus_sources/scala/project/AutomaticModuleName.scala`
(`sha256=782827151878e9af2e21e1c9691b6eed34c3fc9dfd51a6e7d40d31045d07609d`,
658 bytes).

No parser code was changed and no performance conclusion is drawn.

Docker artifact:
`harness_out/docker/20260625T105437Z-scala-frontier-frame1-20260625`

Wringer artifact:
`cgo_harness/harness_out/scala-frontier-frame1-20260625`

Command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-frontier-frame1-20260625 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- \
  "cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=40 GTS_WRINGER_BASELINE_FRAMES=1 GTS_WRINGER_FIRSTDIFF_FRAMES=1 GTS_WRINGER_VARIANT_FRAMES=1 GTS_WRINGER_VARIANTS='stack2 stack8 node3 forest' GTS_WRINGER_STAGES=baseline,firstdiff,variants,summary GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_TIMEOUT=60 GTS_WRINGER_DIAG_TIMEOUT=60 bash cgo_harness/docker/run_grammar_integrity_wringer.sh scala cgo_harness/harness_out/scala-frontier-frame1-20260625"
```

The run completed cleanly: `exit_code=0`, `oom_killed=false`,
`baseline rc=0`, `infra status=0`. The wringer control contract was closed:
6 planned actions, 6 completed actions, 0 incomplete actions.

## Baseline Result

| Field | Result |
| --- | --- |
| Parity | `0/1` |
| Family | `truncation_frontier_loss` |
| Stop reason | `iteration_limit` |
| Truncated | `true` |
| EOF progress | EOF not reached; `tokens=220`, `lastTokenEnd=312`, `expectedEOF=658`, `lastTokenEOF=false` |
| Root pair | `compilation_unit->compilation_unit` |
| Root spans | Go `0:311`, C `0:658` |
| Root child counts | Go `199`, C `5` |
| Errors/missing | Go `0/0`, C `0/0` |
| Iterations | `19740/19740` |
| Nodes | `39065/300000` |
| Max stacks | `3` |
| Peak depth | `208/1316` |

The default production parser still reproduces the documented current-smoke
state: non-parity, truncated, `iteration_limit`, no error or missing nodes,
and a shorter Go root span.

First-diff diagnostic also completed with `rc=0`. It reports the first diff at
the root. The C tree has five root children:
`package_clause`, two `import_declaration` nodes, one `block_comment`, and one
`object_definition`. The Go tree stops inside the block comment and materializes
many `block_comment_repeat1` children before truncating at byte 311.

## Variant Results

| Variant | Parity | Stop reason | Truncated/runtime | EOF/span progress | Error shape |
| --- | ---: | --- | --- | --- | --- |
| `stack2` | `0/1` | `iteration_limit` | `truncated=true`; `iterations=19740/19740`, `nodes=39065/300000`, `maxStacks=3` | unchanged: tokens `220`, root span `0:311` | unchanged: Go/C errors `0/0`, missing `0/0` |
| `stack8` | `0/1` | `iteration_limit` | `truncated=true`; `iterations=19740/19740`, `nodes=39065/300000`, `maxStacks=3` | unchanged: tokens `220`, root span `0:311` | unchanged: Go/C errors `0/0`, missing `0/0` |
| `node3` | `0/1` | `iteration_limit` | `truncated=true`; `iterations=19740/19740`, `nodes=39065/900000`, `maxStacks=3` | unchanged: tokens `220`, root span `0:311` | unchanged: Go/C errors `0/0`, missing `0/0` |
| `forest` | `0/1` | `none` in runtime telemetry | `truncated=false` in runtime telemetry; DTIER still counts `trunc=1` because Go span is shorter than C | improved span to `0:395`, still not EOF/full C span `0:658` | unchanged no-error shape: Go/C errors `0/0`, missing `0/0` |

Stack cap changes did not alter the observed production frontier. Raising the
node budget by 3x also did not alter the stop reason, token progress, span, or
parity. Forest mode is the only variant that changes the failure shape: it
removes the runtime `iteration_limit` stop and reaches the end of the block
comment at byte 395, but it still does not produce the C root span or parity.

## Classification

Current HEAD evidence keeps Scala frame 1 as a true-coding runtime-frontier
witness under production settings, but the next machinery lane is
`forest/materialization`, not node budget or stack/frontier cap.

Evidence:

- `node3` leaves the production state unchanged despite `nodes=39065/900000`,
  so the next fix is not a simple node-budget increase.
- `stack2` and `stack8` leave the production state unchanged with
  `maxStacks=3`, so the next fix is not explained by the configured stack cap.
- The default first-diff shape is rooted in block-comment materialization:
  C emits one `block_comment`, while Go emits many `block_comment_repeat1`
  fragments and truncates inside the comment.
- `forest` changes the stop reason and advances the Go root span from `0:311`
  to `0:395`, which points at forest/materialization behavior as the useful
  next lane. It does not prove parity is close; object-definition coverage
  after the comment is still missing.

This diagnostic does not support classifying the frame as recovery-shape:
both Go and C report `0` errors and `0` missing nodes. It also does not support
version/corpus classification, because the C baseline accepts the same corpus
file and the current failure is a shorter Go tree under controlled variants.
