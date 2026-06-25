# Java gap-guard residual classification, 2026-06-25

## Scope

Worktree: `/home/draco/work/gotreesitter-build-baseline`

Branch: `repair/grammargen-build-baseline`

HEAD inspected: `c9312ccb`

This is a parity diagnostic report only. No parser, lexer, grammargen, or Java-specific source was changed.

## Evidence

Primary artifact used:

- `/home/draco/work/gotreesitter-build-baseline/harness_out/docker/20260625T045052Z-diag-java`

That artifact ran Java real-corpus parity in Docker with:

```sh
GTS_GRAMMARGEN_REAL_CORPUS_ENABLE=1
GTS_GRAMMARGEN_REAL_CORPUS_ROOT=/tmp/grammar_parity
GTS_GRAMMARGEN_REAL_CORPUS_PROFILE=aggressive
GTS_GRAMMARGEN_REAL_CORPUS_MAX_CASES=25
GTS_GRAMMARGEN_REAL_CORPUS_REQUIRE_PARITY=0
GTS_GRAMMARGEN_REAL_CORPUS_ALLOW_PARTIAL=1
GTS_GRAMMARGEN_REAL_CORPUS_ONLY=java
go test ./grammargen -run '^TestMultiGrammarImportRealCorpusParity$' -count=1 -v -timeout 15m
```

Result:

```text
real-corpus[aggressive]: no-error 23/25, sexpr parity 23/25, deep parity 23/25 (requireParity=false, seen=25/109)
```

I also ran a Java-only Docker diagnostic to get exact first-error paths for the two logged generated-ERROR samples:

- `/home/draco/work/gotreesitter-build-baseline/harness_out/docker/20260625T045745Z-diag-java-error-paths`

The first attempt at that diagnostic lacked the cloned Java grammar and failed before parsing:

- `/home/draco/work/gotreesitter-build-baseline/harness_out/docker/20260625T045726Z-diag-java-error-paths`

## Failing samples

### Sample 6

Sample ID and location:

```text
sample 6
corpus_block:/tmp/grammar_parity/java/test/corpus/literals.txt
```

Harness class:

```text
gen ERROR on clean ref
```

First generated error path from the focused diagnostic:

```text
root/expression_statement[7]/string_literal/ERROR
```

First generated error range and text:

```text
range=235..237 text="\\\\"
```

Small generated S-expression snippet:

```scheme
(expression_statement
  (string_literal
    (multiline_string_fragment)
    (ERROR)))
```

Small reference S-expression snippet:

```scheme
(expression_statement
  (string_literal
    (multiline_string_fragment)
    (escape_sequence)
    (multiline_string_fragment)))
```

The relevant source block is a Java text block containing a line with `\\`.

### Sample 15

Sample ID and location:

```text
sample 15
corpus_block:/tmp/grammar_parity/java/test/corpus/expressions.txt
```

Harness class:

```text
gen ERROR on clean ref
```

First generated error path from the focused diagnostic:

```text
root/expression_statement[3]/method_invocation/argument_list/string_literal/ERROR
```

First generated error range and text:

```text
range=160..180 text="\\\\{}\n              \\"
```

Small generated S-expression snippet:

```scheme
(method_invocation
  (identifier)
  (argument_list
    (string_literal
      (multiline_string_fragment)
      (escape_sequence)
      (multiline_string_fragment)
      (multiline_string_fragment)
      (ERROR))))
```

Small reference S-expression snippet:

```scheme
(method_invocation
  (identifier)
  (argument_list
    (string_literal
      (multiline_string_fragment)
      (escape_sequence)
      (multiline_string_fragment)
      (multiline_string_fragment)
      (escape_sequence)
      (multiline_string_fragment)
      (escape_sequence)
      (multiline_string_fragment)
      (multiline_string_fragment))))
```

The relevant source block is a Java string-template corpus case with a nested text block:

```java
compPass("""
        StringTemplate result = RAW.\"\"\"
              \\{}
              \""";
         """);
```

## Classification

These two residuals are best classified as a generated lexer/tokenization issue at a repeated text-block string boundary, not as the old stale real-token gap class.

Reasons:

- Both failures are full clean parses from the real-corpus parity harness, not incremental parses. There is no reused old tree or stale real-token source in play.
- The reference tree is clean in both cases; only the generated parser produces `ERROR`.
- Java at the locked grammar revision has no external scanner: `externals: []`.
- The generated parse fails inside `string_literal`, specifically while consuming escape-like text inside Java text blocks/string templates.
- The reference recognizes the divergent regions as `escape_sequence` children, while the generated parse leaves them as `ERROR`.
- The two cases share the same generalized shape: `_multiline_string_literal` contains a repeat over `multiline_string_fragment`, `_escape_sequence`, and `string_interpolation`; the generated lexer/parser fails when the repeated body must switch from multiline fragment text to escape-sequence tokens around backslash-heavy content.

This does not look like C-recovery/result selection. There is no C grammar involved, and the failure is not a choice among multiple clean generated parse results; it is a generated parse error where the reference has deterministic clean structure.

It also does not look like external scanner/provenance. The Java grammar has no external scanner in this lockfile revision, so scanner state provenance cannot explain the mismatch.

The closest generalized family is: generated DFA/LR handling of lexical precedence and repeat-boundary token admission for hidden escape rules inside repeated string bodies. More concretely, this points at `_escape_sequence` versus `_multiline_string_fragment` selection inside `_multiline_string_literal` repeats, especially when a fragment is followed by a backslash sequence that must become `escape_sequence`.

## c9312ccb comparison

The provided current artifact at `20260625T045052Z-diag-java` shows the two failures above.

There is an earlier Java diagnostic artifact:

- `/home/draco/work/gotreesitter-build-baseline/harness_out/docker/20260625T044215Z-diag-java`

It reports the same count and the same two sample IDs:

```text
sample 6  corpus_block:/tmp/grammar_parity/java/test/corpus/literals.txt
sample 15 corpus_block:/tmp/grammar_parity/java/test/corpus/expressions.txt
real-corpus[aggressive]: no-error 23/25, sexpr parity 23/25, deep parity 23/25
```

However, neither artifact records the gotreesitter source commit used for the run. Therefore, from artifacts alone, I can say the two available Java diagnostic artifacts are identical for the failing set, but I cannot directly prove that the earlier artifact was produced at baseline `a18e301f`.

If `20260625T044215Z-diag-java` is the `a18e301f` baseline run, then `c9312ccb` did not alter the Java failing set. If that artifact is not definitively tied to `a18e301f`, the commit-to-commit comparison is not directly comparable from the retained logs.

## Next generalized experiment

Avoid Java-specific parser or source patches. The next useful generalized experiment is to instrument/generated-test the lexer admission path for repeated string-like bodies:

1. Build a tiny grammar-free or imported-grammar reduction with a repeat body containing:
   - a broad fragment token,
   - an immediate/higher-precedence escape token,
   - an interpolation-like token or delimiter alternative.
2. Capture generated DFA accept candidates and LR valid-symbol filtering at the byte where a backslash sequence should switch from fragment to escape token.
3. Compare generated token choices against the reference token stream for the two Java snippets.
4. Try a generalized fix in token precedence/admission or repeat-boundary handling only if the reduced case reproduces outside Java.

The experiment should answer whether the bug is:

- DFA accept precedence choosing or retaining the fragment path too long,
- LR valid-symbol filtering failing to admit `_escape_sequence` at the repeat boundary,
- or repeat-body materialization losing the hidden/alias relationship between `_escape_sequence` and visible `escape_sequence`.

