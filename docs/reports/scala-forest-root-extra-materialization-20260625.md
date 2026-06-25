# Scala Forest Root Extra Materialization, 2026-06-25

## Scope

Bounded follow-up for Scala frame 1:
`/workspace/corpus_sources/scala/project/AutomaticModuleName.scala`
(`sha256=782827151878e9af2e21e1c9691b6eed34c3fc9dfd51a6e7d40d31045d07609d`,
658 bytes).

This is a report-only classification artifact. It does not add a Scala
normalizer, does not add per-grammar parser policy, and does not update the
canonical tier smoke artifact.

## Diagnostic Support

`TestFirstDiffDiag` now honors `REPRO_FOREST=1`, so the same first-diff sibling
dump can be used for the experimental forest path. This is diagnostic-only and
does not affect parser behavior.

## Reproduction

Forest first-diff trace:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-root-child-trace-20260625 \
  --memory 8g \
  --cpus 4 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- \
  "cd /workspace/cgo_harness && REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/AutomaticModuleName.scala REPRO_FOREST=1 REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Artifact:
`harness_out/docker/20260625T121757Z-scala-root-child-trace-20260625`

Result: `exit_code=0`, `oom_killed=false`.

Production control, same file and diagnostic:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-production-child-trace-20260625 \
  --memory 8g \
  --cpus 4 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- \
  "cd /workspace/cgo_harness && REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/AutomaticModuleName.scala REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Artifact:
`harness_out/docker/20260625T122133Z-scala-production-child-trace-20260625`

Result: `exit_code=0`, `oom_killed=false`.

Focused forest frame validation:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-forest-frame-validation-20260625 \
  --memory 8g \
  --cpus 4 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- \
  "cd /workspace/cgo_harness && REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/AutomaticModuleName.scala REPRO_PROGRESS=1 REPRO_SIGNATURES=1 REPRO_N=1 REPRO_ROUNDS=1 REPRO_FOREST=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestMeasureDtierVsC$' -count=1 -v"
```

Artifact:
`harness_out/docker/20260625T122330Z-scala-forest-frame-validation-20260625`

Result: `exit_code=0`, `oom_killed=false`,
`parityMatch=0/1`, `diverge=1`, root `go=compilation_unit[0:658]`,
root `c=compilation_unit[0:658]`, child count `go=4 c=5`,
errors `go=0 c=0`, missing `go=0 c=0`. The harness reports `trunc=1`
because `ParseForestExperimental` leaves runtime `stopReason=none`, not
`accepted`, even though the returned root is full span.

## Forest Evidence

The current forest frame reaches the full source span and has no errors/missing
nodes, but the root omits one named extra child:

```text
go stopReason=none
go root: compilation_unit [0:658] childCount=4 hasError=false
c  root: compilation_unit [0:658] childCount=5 hasError=false

go.child[0] package_clause      [0:19]   "package scala.build"
go.child[1] import_declaration  [21:40]  "import sbt.{Def, _}"
go.child[2] import_declaration  [41:58]  "import sbt.Keys._"
go.child[3] object_definition   [396:657] "object AutomaticModuleName  {..."

c.child[0] package_clause       [0:19]   "package scala.build"
c.child[1] import_declaration   [21:40]  "import sbt.{Def, _}"
c.child[2] import_declaration   [41:58]  "import sbt.Keys._"
c.child[3] block_comment        [60:395] named=true extra=true
c.child[4] object_definition    [396:657] "object AutomaticModuleName  {..."
```

The frame therefore moved beyond the earlier recovery-root-at-byte-395 state:
it now returns a full-span/no-error tree, but remains non-parity on root child
count because the root-level `block_comment` extra is not materialized.

## Production Control

The production parser is not a useful accepted-shape control for this file. It
still stops at `iteration_limit`:

```text
go stopReason=iteration_limit
truncated=true
tokens=220
lastTokenEnd=312
expectedEOF=658
root=compilation_unit [0:311]
go root childCount=199
c root childCount=5
```

The production tree explodes inside the block-comment repeat and does not reach
the object definition. This confirms the remaining forest frame is distinct
from the production runtime-frontier failure.

## Classification

Current lane:
`forest/root-extra-materialization-after-repetition-shift`

Refined hypothesis:
the forest real-shift gap handling can treat a grammar-defined named extra as
covered parser padding, allowing the following real token to shift and the parse
to accept, but without materializing the skipped named extra into the root-level
tree.

The next generalized fix should target forest parser machinery, not Scala:

- detect real-shift gaps that are covered by grammar extras but correspond to
  named extra nodes in the C tree;
- materialize those extras into the GSS chain before shifting the following
  real token, or otherwise preserve them for the next reduce window;
- add a grammar-agnostic invariant test that a forest parse cannot silently
  drop a named `extra=true` node from a real-shift gap.

No safe generalized code fix is committed in this slice. The canonical smoke
artifact and generated residual ledger remain unchanged.

## Follow-up: Shift-Gap Materialization Probe

Generalized forest shift-gap materialization was implemented for terminal
grammar extras accepted by the broad DFA skip path:

- skip accepts can now expose their accepted symbol to internal gap scanning;
- `gapIsCoveredByGrammarExtras` is backed by a token-returning scanner;
- forest `ParseActionShift` inserts transparent extra leaves before shifting a
  real token across a fully grammar-extra gap;
- whitespace-only gaps remain parser padding and do not create nodes;
- non-extra stale gaps still dead-end.

Focused coverage:

- `TestParseForestMaterializesNamedGrammarExtraInRealShiftGap`
- `TestParseForestRealShiftGapKeepsPaddingButRejectsNonExtra`

Scala frame validation after this scoped fix remained unchanged:

```text
MEASURE-DTIER scala mode=forest files=1 ... parityMatch=0/1(0%) diverge=1
diff="child-count" firstDiffPath="root"
goRoot="compilation_unit" goRootSpan=0:658 goRootCC=4 goRootErr=false
cRoot="compilation_unit" cRootSpan=0:658 cRootCC=5 cRootErr=false
goErrors=0 cErrors=0 goMissing=0 cMissing=0 goStop="none"
```

The follow-up first-diff diagnostic still shows the same missing C
`block_comment [60:395]` root child. This falsifies the terminal skip-token gap
hypothesis for the Scala frame. The remaining residual should stay classified
as `forest/root-extra-materialization-after-repetition-shift`, likely involving
nonterminal extra-chain selection/materialization rather than the terminal
grammar-extra gap handled here.

Canonical smoke remains unchanged.
