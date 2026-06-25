# Scala Case-Clause Span Shared Reduce Diagnosis

Date: 2026-06-25

Scope: one-frame diagnosis for the Scala forest N=40 accepted/no-error span
residual on:

`/workspace/corpus_sources/scala/src/compiler/scala/reflect/macros/contexts/Parsers.scala`

## Witness

The forest replay reproduces the N=40 residual:

```text
DIVERGE-SIG lang="scala" base="Parsers.scala" path="root[7][2][3][6][3][1][4][1][2]" diff="span" goType="case_clause" cType="case_clause" goSpan=1133:1149 cSpan=1133:1142 goCC=3 cCC=3
```

Artifact:

`harness_out/docker/20260625T142405Z-scala-forest-parsers-span-baseline-20260625`

The first-diff audit shows identical visible children:

```text
go: type="case_clause" [1133:1149] cc=3
c : kind="case_clause" [1133:1142] cc=3
go.child[0]: type="case" [1133:1137]
go.child[1]: type="wildcard" [1138:1139]
go.child[2]: type="=>" [1140:1142]
```

Forest artifact:

`harness_out/docker/20260625T142515Z-scala-forest-parsers-firstdiff-20260625`

## Invariant Checked

`ParseForestExperimental` does not skip root finalization. `finalizeForestRoot`
calls `finalizeResultRoot`, and the accepted root therefore goes through the
same compatibility finalization path as production.

For this frame, the wider span is also not a forest-only best-link or trailing
extra materialization defect. Running the same first-diff diagnostic through
the production parser, without `REPRO_FOREST=1`, yields the same Go span:

```text
go stopReason=accepted ... rootHasError=false cRootHasError=false
FIRST-DIFF @root[7][2][3][6][3][1][4][1][2]
go: type="case_clause" [1133:1149] cc=3
c : kind="case_clause" [1133:1142] cc=3
```

Production artifact:

`harness_out/docker/20260625T142631Z-scala-production-parsers-firstdiff-20260625`

## Reduce-Path Comparison

Production reduce and forest reduce share the same parent-span machinery:

- Production reduce trims trailing extras with `reducedEndBeforeTrailingExtras`,
  builds children with `buildReduceChildrenWithPath`, then calls
  `extendParentSpanToWindow` when the child build path may have dropped hidden
  spans.
- Forest reduce uses the same `reducedEndBeforeTrailingExtras`,
  `buildReduceChildrenWithPath`, `computeReduceRawSpan`, and
  `extendParentSpanToWindow` helpers for materialized parents.
- Both paths intentionally avoid scanning trailing extras in
  `extendParentSpanToWindow`; the widened span comes from a hidden non-extra
  entry inside the reduced window, not from root finalization or trailing-extra
  re-push.

The local unit invariant currently expects `_outdent` to be a span-extending
invisible bridge (`TestExtendParentSpanAllowsOutdentGap`). The witness is
consistent with that shared rule over-applying for an empty case clause: visible
children stop at `=>` (`1142`), while Go extends the parent to `1149`.

## Outcome

No code change was made. A small forest-only fix would be incorrect: the
production parser has the same span, and the forest path is faithfully mirroring
the shared generalized reduce-span rule.

Moving this residual would require revisiting the generalized invisible-layout
span invariant, especially when a span-extending hidden tail follows a visible
parent whose own visible children already end at the C span. That is broader
than the requested forest materialization fix and risks regressions against the
existing `_outdent` coverage.

Scala N=40 movement: none. This residual family is diagnosed as shared
production/forest reduce-span behavior, not fixed.
