# Scala Case-Clause Span Result-Normalizer Follow-up

Date: 2026-06-25

Scope: follow-up on the accepted/no-error Scala `case_clause` span residual in:

`/workspace/corpus_sources/scala/src/compiler/scala/reflect/macros/contexts/Parsers.scala`

## Finding

The residual is not fixed by narrowing the shared `_outdent` / invisible-tail
span bridge in `extendParentSpanToWindow`.

A temporary gated reduce-span trace showed the production reduce helper builds
the witness `case_clause` with the C-compatible span before result
compatibility:

```text
SPAN-TRACE before parentSym=259 parentName="case_clause" parent=1133:1142 entries=2 start=0 reducedEnd=2
SPAN-TRACE entry[0] sym=6 name="case" visible=true extra=false childCount=0 span=1133:1137
SPAN-TRACE entry[1] sym=260 name="_case_pattern" visible=false extra=false childCount=2 span=1138:1142
SPAN-TRACE after parentSym=259 parent=1133:1142
```

The later first-diff still reports:

```text
FIRST-DIFF @root[7][2][3][6][3][1][4][1][2]
go: type="case_clause" [1133:1149] cc=3
c : kind="case_clause" [1133:1142] cc=3
```

The widening comes from the existing Scala compatibility pass
`normalizeScalaCaseClauseEnds` in `parser_result_scala_compilation.go`. That
pass walks `case_block` children and extends each `case_clause` to the next
non-`_automatic_semicolon` boundary when the intervening gap is newline trivia.
For this witness the next boundary is the closing `}` at byte `1149`, so the
pass extends the empty `case _ =>` clause across the newline and indentation.

## Outcome

No parser change was committed. A generalized shared parser invariant was not
found: the shared reducer already produces the matching `1133:1142` span, and
the remaining movement would require changing an existing Scala-specific result
normalizer. That violates the hard constraint for this task: no Scala-specific
fix, no language-name parser policy, and no new per-grammar normalizer.

## Verification

Focused host tests:

```text
go test . -run 'TestExtendParentSpan|TestShouldUseRawSpan|TestComputeReduceRawSpan' -count=1
ok github.com/odvcencio/gotreesitter 0.014s

go test . -run 'Test.*Reduce.*|Test.*Forest.*' -count=1
ok github.com/odvcencio/gotreesitter 3.613s
```

Docker first-diff replays confirmed no movement:

```text
bash cgo_harness/docker/run_parity_in_docker.sh --no-build \
  --label scala-production-parsers-result-normalizer-final \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/src/compiler/scala/reflect/macros/contexts/Parsers.scala REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"

bash cgo_harness/docker/run_parity_in_docker.sh --no-build \
  --label scala-forest-parsers-result-normalizer-final \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/src/compiler/scala/reflect/macros/contexts/Parsers.scala REPRO_FOREST=1 REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Artifacts:

- `harness_out/docker/20260625T144109Z-scala-production-parsers-result-normalizer-final`
- `harness_out/docker/20260625T144109Z-scala-forest-parsers-result-normalizer-final`

Both runs reported Go `case_clause` `1133:1149` and C `case_clause`
`1133:1142`.

The temporary reduce-span trace artifact used for attribution was:

- `harness_out/docker/20260625T143822Z-scala-production-parsers-span-trace`
