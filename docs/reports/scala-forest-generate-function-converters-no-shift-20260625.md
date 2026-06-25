# Scala Forest GenerateFunctionConverters No-Shift Classification

Date: 2026-06-25

Scope: bounded investigation of the Scala forest N=40 `go_no_tree` residual in
`/workspace/corpus_sources/scala/project/GenerateFunctionConverters.scala` after
the nil-tree accounting fix from `a241e618`.

This is a report-only classification. No parser behavior change is included.

## Reproduction

Command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-generate-function-converters-forest-n1-debug \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/GenerateFunctionConverters.scala REPRO_FOREST=1 REPRO_N=1 REPRO_ROUNDS=1 REPRO_PROGRESS=1 REPRO_SIGNATURES=1 go test . -tags treesitter_c_parity -run '^TestMeasureDtierVsC$' -count=1 -v"
```

Artifact:
`harness_out/docker/20260625T144725Z-scala-generate-function-converters-forest-n1-debug`

Result:

```text
MEASURE-DTIER scala mode=forest files=1 medianRatio=0.49x aggRatio=0.49x parityMatch=0/1(0%) diverge=0 trunc=1 errTree=0 panics=0 goNS=5940230 cNS=12153530
```

Both the timed parse and the comparison reparse returned nil:

```text
result=not_accepted stopReason= runtime="nil_tree forestDeclineReason=no-shift-death"
result=go_no_tree runtime="nil_tree forestDeclineReason=no-shift-death"
```

This confirms the N=40 residual is still reproducible after nil-tree accounting,
and it is a generalized forest parser survival failure rather than a
post-materialization comparison normalizer.

## Trace

Command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-generate-function-converters-forest-dfa-trace \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && REPRO_LANG=scala REPRO_FILE=/workspace/corpus_sources/scala/project/GenerateFunctionConverters.scala REPRO_FOREST=1 REPRO_DEBUG_DFA=1 REPRO_GLR_TRACE=1 go test . -tags treesitter_c_parity -run '^TestFirstDiffDiag$' -count=1 -v"
```

Artifact:
`harness_out/docker/20260625T144832Z-scala-generate-function-converters-forest-dfa-trace`

The replay fails before a Go tree is available. The final trace window is:

```text
DFA tok 79 operator_identifier 5592 5594 == state=8336
DFA tok 62 ) 5596 5597 ) state=8336
KILL stale shift: stack_byte=5594 tok=5596..5597 gap=" 0"
```

At the Scala source window this is inside:

```text
if (k == 0) Vector.fill(3)("  ") ++ ...
```

The forest token source skipped over the real `0` and later tried to attach the
`)` token to a frontier node still at byte `5594`. The generic real-shift gap
guard correctly rejected that stack because the gap `" 0"` is not parser
padding or a grammar extra.

## Negative Probes

These probes were intentionally generalized and did not introduce or keep
Scala-specific behavior.

1. Raised `forestMaxLinksPerNode` from `8` to `16`.

   Artifact:
   `harness_out/docker/20260625T144947Z-scala-generate-function-converters-forest-cap16-probe`

   Result: no movement. The witness still returned
   `forestDeclineReason=no-shift-death`.

2. Disabled pending-parent/final-child-ref/compact-leaf materialization:

   ```sh
   GOT_GLR_V2_PENDING_PARENTS=0
   GOT_GLR_V2_FINAL_CHILD_REFS=0
   GOT_GLR_V2_COMPACT_FULL_LEAVES=0
   ```

   Artifact:
   `harness_out/docker/20260625T145027Z-scala-generate-function-converters-forest-matoff-probe`

   Result: no movement. The witness still returned
   `forestDeclineReason=no-shift-death`.

3. Tried a generic forest primary parser-state selector that chose the maximum
   frontier byte offset, then lower error cost, then higher best-link score.

   Artifact:
   `harness_out/docker/20260625T145150Z-scala-generate-function-converters-forest-primary-probe`

   Result: unsafe/inconclusive. The single-file run exceeded a practical bound
   compared with the baseline millisecond-scale decline and was interrupted.
   The probe was reverted and is not a valid fix.

## Canary

The previously clean Scala forest canary remains clean on the unchanged code:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-automatic-module-name-forest-canary-report \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/AutomaticModuleName.scala REPRO_FOREST=1 REPRO_N=1 REPRO_ROUNDS=1 REPRO_PROGRESS=1 REPRO_SIGNATURES=1 go test . -tags treesitter_c_parity -run '^TestMeasureDtierVsC$' -count=1 -v"
```

Artifact:
`harness_out/docker/20260625T145405Z-scala-automatic-module-name-forest-canary-report`

Result:

```text
MEASURE-DTIER scala mode=forest files=1 medianRatio=12.43x aggRatio=12.43x parityMatch=1/1(100%) diverge=0 trunc=0 errTree=0 panics=0 goNS=5185465 cNS=417331
```

## Classification

The `GenerateFunctionConverters.scala` residual is best classified as:

```text
forest/frontier-lex-state-survival/no-shift-death-after-real-token-skip
```

The parser is not reaching materialization or tree comparison. It loses the
frontier state that should lex or shift the integer literal `0` inside an
ordinary expression, then the generic stale-gap guard declines when the next
token attempts to skip over that real text.

The completed probes rule out a simple per-node link-cap increase and the
existing materialization representation toggles. The primary-state probe shows
that lex-state/frontier selection can materially change behavior, but it is not
a safe small fix as attempted.

## Next Generalized Experiment

Instrument the forest loop around byte `5594` to record, per token step:

- frontier states and byte offsets before lexing;
- selected primary parser state and GLR state union;
- lex-mode pair per frontier state;
- candidate DFA token per unique lex-mode pair and its action score;
- coalesced nodes dropped by fan-out cap or pre-cap pruning for the same byte
  window.

The next fix should be a generic forest frontier/token-source invariant, not a
Scala normalizer, not a language-name parser policy, and not an edit to
`parser_result_scala_compilation.go`.
