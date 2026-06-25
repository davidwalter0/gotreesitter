# Real Token Gap Guard Extraction Blockers - 2026-06-25

## Summary

Attempted a clean extraction of the generalized source-aware real-token
shift/attachment gap guard slice from a new worktree rooted at `03f1dfd1`.

The narrow extraction itself was implementable without importing multi-link GSS,
broader C-recovery changes, Java/JSON per-grammar lexer files, or per-grammar
normalizers. The trial patch touched only:

- `lexer.go`
- `parser.go`
- `parser_dfa_token_source.go`
- `parser_shift_gap_test.go`

The trial patch added:

- `Token.ExternalScannerToken` / `Token.ExternalScannerStartByte`
- external-token provenance propagation through DFA external scanner paths and
  relex
- `gapIsCoveredByGrammarExtras`
- escaped-newline parser padding
- grammar-extra/comment gap acceptance in `guardRealTokenAttachmentGap`
- focused tests for non-trivia rejection, grammar extra/comment acceptance,
  NoLookahead/missing acceptance, external scanner-owned gaps, and escaped
  newline padding

## Focused Gate Result

Focused Docker package test passed:

```text
bash cgo_harness/docker/run_parity_in_docker.sh -- "cd /workspace && go test . -run 'TestReal.*Gap|TestForest.*Gap' -count=1"
ok  	github.com/odvcencio/gotreesitter	0.015s
artifacts: /home/draco/work/gotreesitter-real-token-gap-guard/harness_out/docker/20260625T042816Z
```

## Blocking Gate

The required Java and JSON parity gates cannot run from clean `03f1dfd1`
because `./grammargen` fails to compile before any grammar-specific parity
cases execute.

Java on the extraction worktree failed with:

```text
bash cgo_harness/docker/run_single_grammar_parity.sh java
grammargen/diagnostics.go:558:15: assignment mismatch: 1 variable but resolveConflicts returns 2 values
grammargen/diagnostics.go:558:40: not enough arguments in call to resolveConflicts
	have (*LRTables, *NormalizedGrammar)
	want (context.Context, *LRTables, *NormalizedGrammar)
grammargen/lr_lalr.go:270:42: ctx.definitionBoundaryTagBySym undefined
grammargen/lr_repetition_conflict_test.go:921:13: meta.GeneratedRepeatAux undefined
grammargen/python_keyword_test.go:160:11: lang.CRecoveryCostCompetitionCapable undefined
```

Artifacts:

```text
/home/draco/work/gotreesitter-real-token-gap-guard/harness_out/docker/20260625T042839Z-diag-java
```

The same Java command was then run from an unmodified baseline worktree at
`/tmp/gotreesitter-rtgg-baseline` checked out directly at `03f1dfd1`, and it
failed with the same `./grammargen` compile errors:

```text
/tmp/gotreesitter-rtgg-baseline/harness_out/docker/20260625T042915Z-diag-java
```

JSON on the extraction worktree also failed at the same `./grammargen` compile
step:

```text
bash cgo_harness/docker/run_single_grammar_parity.sh json
FAIL	github.com/odvcencio/gotreesitter/grammargen [build failed]
```

Artifacts:

```text
/home/draco/work/gotreesitter-real-token-gap-guard/harness_out/docker/20260625T042931Z-diag-json
```

## Decision

Do not land the parser gap-guard code slice from this task because the stated
commit policy requires focused + Java + JSON to pass before committing code, and
the clean `03f1dfd1` baseline cannot build the parity package needed by Java and
JSON.

The blocker is pre-existing relative to this extraction branch, not caused by
the trial gap-guard changes.

## Next Step

First restore `./grammargen` buildability on clean `03f1dfd1` or choose a clean
base commit where:

```text
bash cgo_harness/docker/run_single_grammar_parity.sh java
bash cgo_harness/docker/run_single_grammar_parity.sh json
```

can reach parity execution. Then re-apply the four-file gap-guard extraction and
rerun the focused, Java, and JSON gates before landing the code slice.
