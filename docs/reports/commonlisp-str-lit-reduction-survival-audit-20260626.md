# Common Lisp str_lit Reduction Survival Audit, 2026-06-26

Scope: report-only diagnostic for the 206 classification objective. Temporary
parser instrumentation was used to dump child-level reduction shapes for Common
Lisp frame 1 and then reverted. No parser, grammar, normalizer, or language-name
policy change was kept.

Worktree: `/home/draco/work/gotreesitter-build-baseline`

Source head before diagnostics: `9c9c01d2`

Diagnostic artifact:

- `cgo_harness/harness_out/commonlisp-str-lit-child-shape-20260626/frame-0001-bbtrees-child-shape.log`

Docker artifact:

- `harness_out/docker/20260626T032146Z-commonlisp-str-lit-child-shape-frame1`

## Conclusion

The C-like wider errored child candidate was not found in Go before final
selection.

- Go constructed the contested errored `format_specifier` only as
  `format_specifier [2975:2978]`, with nested `format_modifiers [2976:2977]`
  and `format_directive_type [2977:2978]`.
- Go separately constructed the next sibling as `format_specifier [2979:2982]`.
  No Go reduction candidate in the trace had `format_specifier [2975:2980]` or
  `format_directive_type [2976:2980]`; those spans appeared only in the C-side
  symbol audit.

This rules out GSS merge/cull, same-key compare, reduce-window selection,
pending-parent/no-tree compaction, same-pop/result-child selection, and final
result ordering as the drop point for this exact wider child shape. The shape is
not lost after construction; it is never constructed by Go in this diagnostic.

Because the proof did not identify a small default-on generalized parser
invariant, no machinery fix was landed. Common Lisp remains Tier IV.

## Temporary Diagnostic

Temporary code added a default-off environment flag:

```text
GOT_REDUCE_CHILD_SHAPE_TRACE=1
```

The hook reused `GOT_C_RECOVERY_TRACE_WINDOW=2970:2985` and printed child-level
shapes after parent reduction construction, plus final result candidates. Each
entry included phase, token, reduce symbol, child count, production id, dynamic
precedence, parent symbol/span/error flags, stack metadata, and nested children
to depth 3.

The temporary code was reverted after the diagnostic run.

## Frame 1 Evidence

Command:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label commonlisp-str-lit-child-shape-frame1 \
  --memory 8g \
  --cpus 4 \
  --no-build \
  -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-child-shape-20260626 && /usr/bin/time -v timeout --kill-after=10s 120s env CGO_ENABLED=1 GOT_C_RECOVERY=0 GOT_C_RECOVERY_TRACE_WINDOW=2970:2985 GOT_REDUCE_CHILD_SHAPE_TRACE=1 REPRO_LANG=commonlisp REPRO_FILE=/workspace/corpus_sources/commonlisp/benchmarks/bbtrees.lisp REPRO_DIR=/workspace/corpus_sources REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-child-shape-20260626/frame-0001-bbtrees-child-shape.log"
```

Result:

- Exit status: `0`
- Go result: `accepted`, full EOF, not truncated
- First diff unchanged: `@root[23][3][3]`, `str_lit [2958:2993]`, Go child
  count `10`, C child count `9`
- Max RSS: `1567908 KB`

Targeted log checks:

```bash
rg -c 'span=2975:2980|span=2976:2980|parent_span=2975:2980|parent_span=2976:2980' \
  cgo_harness/harness_out/commonlisp-str-lit-child-shape-20260626/frame-0001-bbtrees-child-shape.log
```

Result: `2`.

Both matches were from the C-side `SYMBOL-AUDIT` tail:

```text
SYMBOL-AUDIT c.child[4] kind="format_specifier" ... span=2975:2980 ...
SYMBOL-AUDIT c.child[4].child[1] kind="format_directive_type" ... span=2976:2980 ...
```

No `REDUCE-CHILD-SHAPE` line contained either C-like span.

Representative Go construction lines:

```text
REDUCE-CHILD-SHAPE phase=gss-parent tok_sym=12 tok_name="str_lit_token1" tok=2978:2979 reduce_symbol=147 reduce_symbol_name="format_specifier" ... parent_span=2975:2978 parent_has_error=true parent_children=3 ...
REDUCE-CHILD-SHAPE node phase=gss-parent depth=0 index=root sym=145 name="format_modifiers" span=2976:2977 ... has_error=true child_count=2
REDUCE-CHILD-SHAPE node phase=gss-parent depth=0 index=root sym=146 name="format_directive_type" span=2977:2978 ... has_error=false child_count=0
REDUCE-CHILD-SHAPE node phase=gss-parent ... sym=147 name="format_specifier" span=2979:2982 ... has_error=false child_count=2
REDUCE-CHILD-SHAPE phase=gss-parent tok_sym=1 tok_name="_ws" tok=2993:3004 reduce_symbol=122 reduce_symbol_name="str_lit" ... parent_span=2958:2993 parent_has_error=true parent_children=10 ...
```

## Interpretation

At token `~ [2975:2976]`, Go still has reduce/shift competition for the repeat.
After shifting into the contested directive, Go closes the errored
`format_specifier` at byte `2978`, before the `=` byte and the following `~`.
The later `~ [2979:2980]` is then available only as a fresh sibling starter.

The nearest next target is therefore earlier than same-key stack/result
selection: token admission/recovery while forming the errored directive child.
A viable generalized proof would need to explain when an error-bearing child may
retain a later sibling-starter token inside its subtree, without naming Common
Lisp or adding a per-grammar normalizer.

## Verification

Temporary instrumentation compile smoke:

```bash
go test . -run '^TestBuildResultFromGLRUsesCSubtreeOrderOnScoreTie$' -count=1
```

Result:

```text
ok  	github.com/odvcencio/gotreesitter	0.005s
```

After reverting temporary parser code, `git diff -- parser_reduce.go
parser_result.go` was empty.
