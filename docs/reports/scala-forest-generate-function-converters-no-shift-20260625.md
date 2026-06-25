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

## Window Diagnostic

Command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-generate-function-converters-forest-window-trace-v2 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && GOT_GLR_FOREST_TRACE_WINDOW=5588:5600 REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/GenerateFunctionConverters.scala REPRO_FOREST=1 REPRO_N=1 REPRO_ROUNDS=1 REPRO_PROGRESS=1 REPRO_SIGNATURES=1 go test . -tags treesitter_c_parity -run '^TestMeasureDtierVsC$' -count=1 -v -timeout=60s"
```

Artifact:
`harness_out/docker/20260625T150602Z-scala-generate-function-converters-forest-window-trace-v2`

Result: no movement. The witness still returned
`forestDeclineReason=no-shift-death`.

The byte-window trace shows the forest has already collapsed to a single
frontier state by the failing expression:

```text
FOREST-TRACE step=1361 lexer_start=5591 lexer_end=5594 selected_state=8336 glr=[8336] tok=sym=79(operator_identifier) 5592..5594 text="=="
  state=8336 lex=(70,0 active=70) candidate=sym=79(operator_identifier) 5592..5594 text="==" ...
    frontier state=8336 byte=5591 ... tok_actions=count=1 shift=0 reduce=1 accept=0 gap_ok=true
FOREST-TRACE step=1362 lexer_start=5594 lexer_end=5597 selected_state=8336 glr=[8336] tok=sym=62()) 5596..5597 text=")"
  state=8336 lex=(70,0 active=70) candidate=sym=62()) 5596..5597 text=")" ...
    frontier state=8336 byte=5594 ... tok_actions=count=1 shift=0 reduce=1 accept=0 gap_ok=false
```

No fan-out cap or pre-cap drops were reported in the requested byte window.
The immediate invariant is therefore narrower than "primary state selected the
wrong branch from a live union": there is no live union at byte `5594`. The only
surviving frontier state uses lex mode `70`, whose DFA cannot emit the integer
literal at the real input byte and instead skips to `)`, where the stale-gap
guard correctly declines.

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

After adding the gated byte-window diagnostics, the same canary remained clean:

```text
artifact: harness_out/docker/20260625T150644Z-scala-automatic-module-name-forest-window-trace-canary-v2
MEASURE-DTIER scala mode=forest files=1 medianRatio=12.36x aggRatio=12.36x parityMatch=1/1(100%) diverge=0 trunc=0 errTree=0 panics=0 goNS=5118081 cNS=414232
```

A small non-Scala forest canary also remained clean:

```text
artifact: harness_out/docker/20260625T150702Z-css-style-forest-window-trace-canary-v2
MEASURE-DTIER css mode=forest files=1 medianRatio=3.00x aggRatio=3.00x parityMatch=1/1(100%) diverge=0 trunc=0 errTree=0 panics=0 goNS=1263941 cNS=421964
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

The completed probes rule out a simple per-node link-cap increase, the existing
materialization representation toggles, and in-window fan-out/pre-cap pruning.
The primary-state probe shows that lex-state/frontier selection can materially
change behavior, but the window trace shows this frame is already single-state
when it fails, so a token-source primary selector change is not a justified
small fix.

## Next Generalized Experiment

Trace the reduction/shift transition that consumes `==` at `5592..5594` and
builds the next single frontier state. The useful generalized question is now:
which predecessor frontier, reduce path, or conflict-resolution decision removes
the state whose lex mode can emit the integer literal at byte `5595`?

The trace should stay grammar-neutral and record, for the token immediately
before the skip:

- every reduce path emitted from the pre-`==` frontier;
- each shift target that reaches byte `5594`;
- the lex mode of each target before coalescing;
- whether coalescing dedup/cap replacement collapses distinct viable histories;
- whether conflict resolution filters a shift-capable action before the target
  frontier is formed.

The next fix should be a generic forest frontier/token-source invariant, not a
Scala normalizer, not a language-name parser policy, and not an edit to
`parser_result_scala_compilation.go`.
