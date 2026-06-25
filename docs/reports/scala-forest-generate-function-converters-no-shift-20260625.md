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

## Transition Diagnostic

Command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-generate-function-converters-forest-transition-trace-v2 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && GOT_GLR_FOREST_TRACE_WINDOW=5588:5600 GOT_GLR_FOREST_TRACE_TRANSITIONS=1 REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/GenerateFunctionConverters.scala REPRO_FOREST=1 REPRO_N=1 REPRO_ROUNDS=1 REPRO_PROGRESS=1 REPRO_SIGNATURES=1 go test . -tags treesitter_c_parity -run '^TestMeasureDtierVsC$' -count=1 -v -timeout=60s"
```

Artifact:
`harness_out/docker/20260625T151558Z-scala-generate-function-converters-forest-transition-trace-v2`

Result: no movement. The witness still returned
`forestDeclineReason=no-shift-death`:

```text
MEASURE-DTIER scala mode=forest files=1 medianRatio=0.54x aggRatio=0.54x parityMatch=0/1(0%) diverge=0 trunc=1 errTree=0 panics=0 goNS=6338480 cNS=11757641
```

The transition immediately before byte `5594` is not a successful shift and is
not a conflict-resolution loss. At step `1361`, the only surviving state is
`8336`; raw and resolved actions for `operator_identifier` are identical and
contain only `reduce(identifier)`:

```text
FOREST-X actions step=1361 node_state=8336 node_byte=5591 tok=sym=79(operator_identifier) 5592..5594 text="==" raw=[reduce(sym=281(identifier) cc=1 dyn=0 prod=0)] resolved=[reduce(sym=281(identifier) cc=1 dyn=0 prod=0)] filtered_shift=false
FOREST-X reduce-goto-miss step=1361 node_state=8336 node_byte=5591 act=reduce(sym=281(identifier) cc=1 dyn=0 prod=0) pop_state=8336 pop_byte=5590 child_score=0 no_extras=true children=[65535:5590..5591]
FOREST-X recover step=1361 from_state=8336 from_byte=5591 tok=sym=79(operator_identifier) 5592..5594 text="==" recover_state=8336 target_byte=5594 target_lex=(70,0 active=70) error_cost=370 new_frontier=true links=1
```

The byte-`5594` frontier is therefore built by forest recovery absorbing `==` as
an `ERROR` leaf in the same state, after `reduce(identifier)` has no goto from
the popped predecessor. The same pattern is already active for `if`, `(`, and
`k` in the preceding steps. At byte `5594`, the next token source candidate is
`)`; recovery cannot absorb it because that would skip real text `" 0"`:

```text
FOREST-X recover-gap-reject step=1362 state=8336 byte=5594 tok=sym=62()) 5596..5597 text=")"
```

This refines the immediate classification: the local `==` transition is
`forest/recovery-error-leaf-survival-after-reduce-goto-miss`, not
`forest/conflict-resolution-filtered-shift` and not an in-window coalescing or
fan-out cap drop. The viable expression state was already gone before the
`if (k == 0)` window; by byte `5569`, the only surviving frontier is already
state `8336` with accumulated error cost.

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

After adding the gated transition diagnostics, both canaries remained clean:

```text
artifact: harness_out/docker/20260625T151804Z-scala-automatic-module-name-forest-transition-diagnostics-canary
MEASURE-DTIER scala mode=forest files=1 medianRatio=14.32x aggRatio=14.32x parityMatch=1/1(100%) diverge=0 trunc=0 errTree=0 panics=0 goNS=5889361 cNS=411255

artifact: harness_out/docker/20260625T151847Z-css-style-forest-transition-diagnostics-canary
MEASURE-DTIER css mode=forest files=1 medianRatio=3.05x aggRatio=3.05x parityMatch=1/1(100%) diverge=0 trunc=0 errTree=0 panics=0 goNS=1184986 cNS=388967
```

## Classification

The `GenerateFunctionConverters.scala` residual is best classified as:

```text
forest/frontier-lex-state-survival/recovery-chain-no-shift-death-after-real-token-skip
```

The parser is not reaching materialization or tree comparison. It loses the
frontier state that should parse the ordinary expression before the inspected
window, then forest recovery keeps state `8336` alive by absorbing `if (k ==` as
error leaves. The generic stale-gap guard correctly declines when the next token
attempts to skip over the real integer literal text `" 0"`.

The completed probes rule out a simple per-node link-cap increase, the existing
materialization representation toggles, and in-window fan-out/pre-cap pruning.
The primary-state probe shows that lex-state/frontier selection can materially
change behavior, but the window trace shows this frame is already single-state
when it fails, and the transition trace shows conflict resolution does not
filter any shift-capable action in this window. A token-source primary selector
change or local coalescing change is therefore not a justified small fix.

## Next Generalized Experiment

Trace backward from the first transition that introduces the high-error-cost
single-state chain before byte `5569`. The next generalized question is:
which ordinary reduce/shift frontier first gives way to recovery-only survival
in state `8336`, and was a lower-error, shift-capable branch dropped by
coalescing, cap/pre-cap pruning, reduce-goto miss, or token-source candidate
selection before recovery began?

The trace should stay grammar-neutral and record the same raw/resolved actions,
reduce paths, goto targets, shift targets, target lex modes, coalescing
dedup/replacement, cap/pre-cap drops, and recovery absorptions around the first
error-cost jump into this chain. The next fix should be a generic forest
frontier/recovery/error-cost invariant, not a Scala normalizer, not a
language-name parser policy, and not an edit to
`parser_result_scala_compilation.go`.

## Backward Transition Trace

Command sequence:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-gfc-forest-transition-5520-5570 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && GOT_GLR_FOREST_TRACE_WINDOW=5520:5570 GOT_GLR_FOREST_TRACE_TRANSITIONS=1 REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/GenerateFunctionConverters.scala REPRO_FOREST=1 REPRO_N=1 REPRO_ROUNDS=1 REPRO_PROGRESS=1 REPRO_SIGNATURES=1 go test . -tags treesitter_c_parity -run '^TestMeasureDtierVsC$' -count=1 -v -timeout=60s"

bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-gfc-forest-transition-5400-5520 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && GOT_GLR_FOREST_TRACE_WINDOW=5400:5520 GOT_GLR_FOREST_TRACE_TRANSITIONS=1 REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/GenerateFunctionConverters.scala REPRO_FOREST=1 REPRO_N=1 REPRO_ROUNDS=1 REPRO_PROGRESS=1 REPRO_SIGNATURES=1 go test . -tags treesitter_c_parity -run '^TestMeasureDtierVsC$' -count=1 -v -timeout=60s"

bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-gfc-forest-transition-5200-5405 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && GOT_GLR_FOREST_TRACE_WINDOW=5200:5405 GOT_GLR_FOREST_TRACE_TRANSITIONS=1 REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/GenerateFunctionConverters.scala REPRO_FOREST=1 REPRO_N=1 REPRO_ROUNDS=1 REPRO_PROGRESS=1 REPRO_SIGNATURES=1 go test . -tags treesitter_c_parity -run '^TestMeasureDtierVsC$' -count=1 -v -timeout=60s"

bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-gfc-forest-transition-5000-5205 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && GOT_GLR_FOREST_TRACE_WINDOW=5000:5205 GOT_GLR_FOREST_TRACE_TRANSITIONS=1 REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/GenerateFunctionConverters.scala REPRO_FOREST=1 REPRO_N=1 REPRO_ROUNDS=1 REPRO_PROGRESS=1 REPRO_SIGNATURES=1 go test . -tags treesitter_c_parity -run '^TestMeasureDtierVsC$' -count=1 -v -timeout=60s"

bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-gfc-forest-transition-4970-5040 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && GOT_GLR_FOREST_TRACE_WINDOW=4970:5040 GOT_GLR_FOREST_TRACE_TRANSITIONS=1 REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/GenerateFunctionConverters.scala REPRO_FOREST=1 REPRO_N=1 REPRO_ROUNDS=1 REPRO_PROGRESS=1 REPRO_SIGNATURES=1 go test . -tags treesitter_c_parity -run '^TestMeasureDtierVsC$' -count=1 -v -timeout=60s"
```

Artifacts:

```text
harness_out/docker/20260625T152318Z-scala-gfc-forest-transition-5520-5570
harness_out/docker/20260625T152347Z-scala-gfc-forest-transition-5400-5520
harness_out/docker/20260625T152410Z-scala-gfc-forest-transition-5200-5405
harness_out/docker/20260625T152430Z-scala-gfc-forest-transition-5000-5205
harness_out/docker/20260625T152457Z-scala-gfc-forest-transition-4970-5040
```

All runs left the witness unmoved:

```text
5520:5570 MEASURE-DTIER scala mode=forest ... parityMatch=0/1(0%) ... trunc=1 ... runtime="nil_tree forestDeclineReason=no-shift-death"
5400:5520 MEASURE-DTIER scala mode=forest ... parityMatch=0/1(0%) ... trunc=1 ... runtime="nil_tree forestDeclineReason=no-shift-death"
5200:5405 MEASURE-DTIER scala mode=forest ... parityMatch=0/1(0%) ... trunc=1 ... runtime="nil_tree forestDeclineReason=no-shift-death"
5000:5205 MEASURE-DTIER scala mode=forest ... parityMatch=0/1(0%) ... trunc=1 ... runtime="nil_tree forestDeclineReason=no-shift-death"
4970:5040 MEASURE-DTIER scala mode=forest ... parityMatch=0/1(0%) ... trunc=1 ... runtime="nil_tree forestDeclineReason=no-shift-death"
```

The requested `5520:5570` start window was still too late. It showed the same
single-state chain at byte `5514`:

```text
FOREST-TRACE step=1346 lexer_start=5514 lexer_end=5528 selected_state=8336 glr=[8336] tok=sym=1(_alpha_identifier) 5514..5528 text="implicitToJava"
FOREST-X recover step=1357 from_state=8336 from_byte=5553 tok=sym=60(() 5568..5569 text="(" recover_state=8336 target_byte=5569 target_lex=(70,0 active=70) error_cost=364 new_frontier=true links=1
```

The `5400:5520` and `5200:5405` windows showed the chain was already active at
byte `5395` with `error_cost=261`, then at byte `5198` with
`error_cost=108`:

```text
FOREST-TRACE step=1319 lexer_start=5395 lexer_end=5407 selected_state=8336 glr=[8336] tok=sym=1(_alpha_identifier) 5395..5407 text="priorityName"
FOREST-TRACE step=1265 lexer_start=5198 lexer_end=5202 selected_state=8336 glr=[8336] tok=sym=1(_alpha_identifier) 5198..5202 text="defs"
```

The tight `4970:5040` replay identified the first error-cost transition:

```text
FOREST-X shift step=1225 from_state=716 from_byte=5025 shift_base_state=716 shift_base_byte=5025 tok=sym=1(_alpha_identifier) 5026..5029 text="pre" act=shift(state=8336) target=8336 target_byte=5029 target_lex=(70,0 active=70) new_frontier=true links=1
FOREST-TRACE step=1226 lexer_start=5029 lexer_end=5037 selected_state=8336 glr=[8336] tok=sym=5(}) 5036..5037 text="}"
  state=8336 lex=(70,0 active=70) candidate=sym=5(}) 5036..5037 text="}" ... actions=none
    frontier state=8336 byte=5029 links=1 error_cost=0 ... tok_actions=none gap_ok=true
FOREST-X recover step=1226 from_state=8336 from_byte=5029 tok=sym=5(}) 5036..5037 text="}" recover_state=8336 target_byte=5037 target_lex=(70,0 active=70) error_cost=1 new_frontier=true links=1
```

Source context:

```scala
      def priorityName(n: Int, pure: Boolean = false): String = {
        val pre =
          if (pure) s"Priority${n}FunctionExtensions"
          else s"trait ${priorityName(n, pure = true)}"
        if (!pure && n < (sccDepthSet.size-1)) s"$pre extends ${priorityName(n+1, pure = true)}" else pre
      }
      val impls =
```

The lower-error path does not disappear through coalescing, dedup, fan-out cap,
pre-cap, conflict resolution, reduce-goto miss, or shift-gap rejection at the
first transition. Immediately before the chain starts, the ordinary zero-cost
frontier has exactly one state and shifts `pre` to state `8336`. The next
selected token is `}` after the line break/indentation gap; state `8336` has no
action for `}`, so forest recovery absorbs it as the first `ERROR` leaf. The
first `reduce-goto-miss` in the chain happens one step later at `val`, after
the branch is already error-bearing:

```text
FOREST-X reduce-goto-miss step=1227 node_state=8336 node_byte=5037 act=reduce(sym=281(identifier) cc=1 dyn=0 prod=0) pop_state=8336 pop_byte=5029 child_score=0 no_extras=true children=[65535:5036..5037]
```

This classifies the first handoff as:

```text
forest/token-source-external-boundary/recovery-after-single-state-no-action
```

The live evidence points at a generic token-source/frontier problem around
context-sensitive external boundary tokens such as automatic semicolons or
newline-sensitive statement terminators. It is not yet a proven behavior fix:
single-state external token selection uses that state's external lex row
directly, while GLR-scored external selection only runs when more than one
state is live. At this failure point the parser has already narrowed to one
state, so a broader external candidate experiment must prove that an alternate
external boundary token is valid and usable before changing behavior.

No generalized behavior fix was applied in this pass.

## Updated Next Generalized Experiment

Add a tightly gated token-source diagnostic around the single-state external
path to print, for one byte window, the active state's external lex-state ID,
valid external symbols, scanner result, and whether each produced external
candidate has a parse action in the current frontier. Re-run the `4970:5040`
window to prove or disprove that an automatic semicolon or newline-sensitive
external boundary token is available before `}` at byte `5036`.

Only if that diagnostic proves a usable lower-cost external boundary exists,
test a generalized fix that lets forest recovery perform a bounded alternate
external-boundary probe before absorbing a later real token in a single-state
frontier. Gate it by parse-action usability and byte progress, then verify the
`GenerateFunctionConverters.scala` witness, `AutomaticModuleName.scala`, and one
non-Scala external-scanner forest canary. Do not add Scala-specific behavior or
edit `parser_result_scala_compilation.go`.
