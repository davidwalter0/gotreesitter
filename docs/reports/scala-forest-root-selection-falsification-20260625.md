# Scala Forest Root Selection Falsification, 2026-06-25

## Scope

Bounded follow-up for Scala frame 1:
`/workspace/corpus_sources/scala/project/AutomaticModuleName.scala`
(`sha256=782827151878e9af2e21e1c9691b6eed34c3fc9dfd51a6e7d40d31045d07609d`,
658 bytes).

The hypothesis under test was that forest finalization chooses a shorter
same-score `bestLink` root while a longer/full-span same-score root alternative
exists. No generalized parser code change is supported by this diagnostic.

## Diagnostic Runs

Temporary local instrumentation was added and removed before this report-only
change. It traced the forest recovery finalizer because the Scala forest variant
does not reach an accept root; with `REPRO_FOREST=1`, recovery is enabled and
`collectForestErrorRoot` builds the returned root from the surviving frontier.

Direct single-file Docker command shape:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-forest-root-candidates-trace-20260625 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- \
  "cd /workspace && mkdir -p cgo_harness/harness_out/scala-forest-root-direct-trace-20260625 && cd cgo_harness && CGO_ENABLED=1 go test -c -tags treesitter_c_parity -o harness_out/scala-forest-root-direct-trace-20260625/measure.test . && cd /workspace && timeout --kill-after=10s 60 env CGO_ENABLED=1 REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/AutomaticModuleName.scala REPRO_PROGRESS=1 REPRO_SIGNATURES=1 REPRO_N=1 REPRO_ROUNDS=1 GOT_PARSE_PROGRESS=1 REPRO_FOREST=1 GOT_FOREST_ROOT_TRACE=1 cgo_harness/harness_out/scala-forest-root-direct-trace-20260625/measure.test -test.run '^TestMeasureDtierVsC$' -test.count=1 -test.v"
```

Artifact:
`harness_out/docker/20260625T110818Z-scala-forest-root-candidates-trace-20260625`

Result: `exit_code=0`, `oom_killed=false`.

## Evidence

The recovery root candidate set at EOF contained only two frontier candidates:

```text
FOREST-ROOT-CANDIDATES keys=2
FOREST-ROOT-CANDIDATE state=21248 byte=395 err=0 links=1 bestScore=0 bestEnd=395
FOREST-ROOT-CANDIDATE state=16120 byte=395 err=0 links=1 bestScore=0 bestEnd=395
FOREST-ROOT accepted state=21248 byte=395 links=1
FOREST-ROOT link=0 score=0 sym=103 extra=false subtreeEnd=395 prevByte=391 prevLinks=1
```

The same trace repeated during the comparison reparse. The measured forest
variant remained unchanged:

```text
goRootSpan=0:395 cRootSpan=0:658 goRootCC=12 cRootCC=5 goErrors=0 cErrors=0 goMissing=0 cMissing=0
MEASURE-DTIER scala mode=forest files=1 ... parityMatch=0/1(0%) diverge=1 trunc=1 errTree=0 panics=0
```

## Conclusion

The suspected `bestLink` finalization bug is falsified for this Scala frame.
There is no same-score longer root link to select at the recovery root:

- the frame uses the recovery synthetic-root path, not `collectForestRootAndExtras`
  from an accepted root;
- the candidate frontier has no node beyond byte `395`;
- both root candidates have one link, score `0`, and best subtree end `395`.

Do not add a generalized link ranking change on this evidence. A score/span
tie-break may still be useful for other forest divergences, but Scala frame 1
does not prove it.

## Next Generic Target

The next diagnostic should move earlier than finalization and inspect why forest
recovery stops with only byte-`395` frontier candidates after the block comment.
Useful generic probes:

- token/action trace at the first no-shift recovery after byte `395`;
- reduce/goto misses around the state pair `21248` and `16120`;
- whether block-comment repeat materialization consumes a valid successor state
  or loses the object-definition path before EOF.

Keep the lane classified as `forest/materialization`, refined to
`forest/recovery-frontier-before-finalization`, not final root-link ranking.
