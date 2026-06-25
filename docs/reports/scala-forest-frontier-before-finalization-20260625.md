# Scala Forest Frontier Before Finalization, 2026-06-25

## Scope

Bounded follow-up for Scala frame 1:
`/workspace/corpus_sources/scala/project/AutomaticModuleName.scala`
(`sha256=782827151878e9af2e21e1c9691b6eed34c3fc9dfd51a6e7d40d31045d07609d`,
658 bytes).

This diagnostic moved earlier than forest recovery finalization. The prior
root-candidate trace showed only byte-`395` candidates; this run inspected why
the forest frontier stopped there.

No grammar-specific normalizer or language-name policy was added.

## Diagnostic Commands

Pre-fix temporary trace command shape:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-forest-frontier-before-finalization-20260625 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- \
  "cd /workspace && mkdir -p cgo_harness/harness_out/scala-forest-frontier-before-finalization-20260625 && cd cgo_harness && CGO_ENABLED=1 go test -c -tags treesitter_c_parity -o harness_out/scala-forest-frontier-before-finalization-20260625/measure.test . && cd /workspace && timeout --kill-after=10s 60 env CGO_ENABLED=1 REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/AutomaticModuleName.scala REPRO_PROGRESS=1 REPRO_SIGNATURES=1 REPRO_N=1 REPRO_ROUNDS=1 GOT_PARSE_PROGRESS=1 REPRO_FOREST=1 GOT_FOREST_FRONTIER_TRACE=1 GOT_FOREST_FRONTIER_TRACE_MIN_BYTE=380 cgo_harness/harness_out/scala-forest-frontier-before-finalization-20260625/measure.test -test.run '^TestMeasureDtierVsC$' -test.count=1 -test.v"
```

Artifact:
`harness_out/docker/20260625T111504Z-scala-forest-frontier-before-finalization-20260625`

Post-fix trace command shape:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-forest-no-lookahead-postfix-trace-20260625 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- \
  "cd /workspace && mkdir -p cgo_harness/harness_out/scala-forest-no-lookahead-postfix-trace-20260625 && cd cgo_harness && CGO_ENABLED=1 go test -c -tags treesitter_c_parity -o harness_out/scala-forest-no-lookahead-postfix-trace-20260625/measure.test . && cd /workspace && timeout --kill-after=10s 60 env CGO_ENABLED=1 REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/AutomaticModuleName.scala REPRO_PROGRESS=1 REPRO_SIGNATURES=1 REPRO_N=1 REPRO_ROUNDS=1 GOT_PARSE_PROGRESS=1 REPRO_FOREST=1 GOT_FOREST_FRONTIER_TRACE=1 GOT_FOREST_FRONTIER_TRACE_MIN_BYTE=390 cgo_harness/harness_out/scala-forest-no-lookahead-postfix-trace-20260625/measure.test -test.run '^TestMeasureDtierVsC$' -test.count=1 -test.v"
```

Artifact:
`harness_out/docker/20260625T112100Z-scala-forest-no-lookahead-postfix-trace-20260625`

Temporary trace instrumentation was removed after the diagnostic.

## Root Cause Fixed

The forest parser treated every symbol-`0` token as real EOF. At byte `395`,
the token source emitted a synthetic no-lookahead EOF (`NoLookahead=true`) for
state `21248` so EOF-table reductions could run without consuming input. The
production parser re-lexes after such reductions; forest instead entered EOF
root collection and returned the short byte-`395` recovery root.

Generic fix:

- after a forest no-lookahead EOF pass, carry the reduced forest index forward
  as the next frontier;
- request another token instead of finalizing;
- keep final EOF collection restricted to real EOF (`NoLookahead=false`).

Focused unit coverage:
`TestParseForestNoLookaheadReductionContinuesToRealToken`.

Final focused unit artifact:
`harness_out/docker/20260625T112314Z-forest-no-lookahead-unit-final-20260625`

## Scala Evidence After Fix

The old byte-`395` root is no longer produced by synthetic-EOF finalization.
The bounded Scala forest replay now exposes the next earlier frontier loss:

```text
FOREST-FRONTIER token sym=0 name="end" start=395 end=395 eof=true noLookahead=true frontier=1
FOREST-FRONTIER state=21248 byte=395 links=1 err=0 actions=1 shift=0 reduce=1 accept=0 recover=0 recoverState=0
FOREST-GOTO-MISS token="end" noLookahead=true nodeState=16120 nodeByte=395 popState=16120 popByte=58 reduceSym=133 reduceName="namespace_wildcard" childCount=1 score=0
FOREST-FRONTIER token sym=104 name="_automatic_semicolon" start=396 end=396 eof=false noLookahead=false frontier=2
FOREST-FRONTIER state=21248 byte=395 links=1 err=0 actions=0 shift=0 reduce=0 accept=0 recover=0 recoverState=0
FOREST-FRONTIER state=16120 byte=395 links=1 err=0 actions=1 shift=0 reduce=1 accept=0 recover=0 recoverState=0
FOREST-GOTO-MISS token="_automatic_semicolon" noLookahead=false nodeState=16120 nodeByte=395 popState=16120 popByte=58 reduceSym=133 reduceName="namespace_wildcard" childCount=1 score=0
FOREST-NO-SHIFT token sym=104 name="_automatic_semicolon" start=396 end=396 noLookahead=false frontier=2 recoverActive=true recoverCount=0
```

The post-fix bounded replay completed cleanly (`exit_code=0`,
`oom_killed=false`) but still did not reach parity:

```text
MEASURE-PROGRESS ... phase=comparison_result result=go_no_tree
MEASURE-DTIER scala mode=forest files=1 ... parityMatch=0/1(0%) diverge=0 trunc=0 errTree=0 panics=0
```

Artifact:
`harness_out/docker/20260625T111935Z-scala-forest-no-lookahead-fix-20260625`

Final no-trace Scala replay artifact:
`harness_out/docker/20260625T112328Z-scala-forest-no-lookahead-final-20260625`

## Refined Next Target

Scala frame 1 remains in `forest/materialization`, now refined from
`forest/recovery-frontier-before-finalization` to:

`forest/zero-width-successor-frontier-after-no-lookahead`

The next generic probe should focus on the real successor token after the block
comment:

- `_automatic_semicolon` at byte `396` has no shift from surviving states
  `21248` and `16120`;
- state `16120` repeats a `namespace_wildcard` reduction whose goto from
  `16120` misses;
- forest recovery declines the zero-width token before it can build an error
  root or advance;
- the object-definition path is not present in the surviving frontier after the
  no-lookahead reduction.

No performance conclusion is drawn from this diagnostic.
