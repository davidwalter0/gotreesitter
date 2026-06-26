# Common Lisp str_lit Child Materialization Audit, 2026-06-26

Scope: report-only correctness narrowing for the 206 true-coding classification goal. No parser, grammar, or normalizer source was edited.

Worktree: `/home/draco/work/gotreesitter-build-baseline`

Source head during diagnostics: `8c0b7419`

Primary focused output:

- `cgo_harness/harness_out/commonlisp-str-lit-audit-20260626/`

## Conclusion

The five repeated accepted/full-EOF Common Lisp residuals share one shape:

- Go closes an errored `format_specifier` early and resumes at a later `~`, materializing one or more extra sibling `format_specifier` children under `str_lit`.
- C keeps the next `~` inside the previous errored `format_specifier`, usually as part of a wider `format_directive_type` child with nested error materialization.
- Root spans and root child counts remain matched; the first diff is localized under the same `str_lit`.

This is not the gated C-recovery port. Frame 1 with `GOT_C_RECOVERY=0` produced the same `str_lit` first diff as the default run. Frame 1 with `GOT_C_RECOVERY=all` moved in the wrong direction, producing a root `ERROR` vs C `source` diff.

No small generalized machinery fix is proven yet. The evidence points to baseline branch/reduction selection before final result build, not to a final-only child filtering issue. A final-build splice could hide this specific child-count delta, but it would not explain or safely reproduce C's wider errored child boundaries.

Common Lisp remains Tier IV.

## Child-Level Evidence

All runs used `TestFirstDiffDiag` with `REPRO_SYMBOL_AUDIT=1`.

### Frame 1: bbtrees.lisp

File: `/workspace/corpus_sources/commonlisp/benchmarks/bbtrees.lisp`

Result:

- `accepted`, full EOF, not truncated
- root errors: Go `true`, C `true`
- first diff: `@root[23][3][3]`
- first-diff node: `str_lit` span `2958:2993`
- Go child count: 10
- C child count: 9

Go visible children:

| index | kind | span | named | extra | hasError | text |
| ---: | --- | --- | --- | --- | --- | --- |
| 0 | `"` | `2958:2959` | false | false | false | `"` |
| 1 | `format_specifier` | `2959:2961` | true | false | false | `~&` |
| 2 | `format_specifier` | `2968:2971` | true | false | false | `~:{` |
| 3 | `format_specifier` | `2971:2973` | true | false | false | `~%` |
| 4 | `format_specifier` | `2975:2978` | true | false | true | `~8a` |
| 5 | `format_specifier` | `2979:2982` | true | false | false | `~8D` |
| 6 | `format_specifier` | `2983:2988` | true | false | false | `~3,1f` |
| 7 | `format_specifier` | `2988:2990` | true | false | false | `~}` |
| 8 | `format_specifier` | `2990:2992` | true | false | false | `~%` |
| 9 | `"` | `2992:2993` | false | false | false | `"` |

C visible children:

| index | kind | span | named | extra | hasError | text |
| ---: | --- | --- | --- | --- | --- | --- |
| 0 | `"` | `2958:2959` | false | false | false | `"` |
| 1 | `format_specifier` | `2959:2961` | true | false | false | `~&` |
| 2 | `format_specifier` | `2968:2971` | true | false | false | `~:{` |
| 3 | `format_specifier` | `2971:2973` | true | false | false | `~%` |
| 4 | `format_specifier` | `2975:2980` | true | false | true | `~8a=~` |
| 5 | `format_specifier` | `2983:2988` | true | false | false | `~3,1f` |
| 6 | `format_specifier` | `2988:2990` | true | false | false | `~}` |
| 7 | `format_specifier` | `2990:2992` | true | false | false | `~%` |
| 8 | `"` | `2992:2993` | false | false | false | `"` |

Nested mismatch:

- Go child 4: `format_directive_type` span `2977:2978`, text `a`.
- C child 4: `format_directive_type` span `2976:2980`, text `8a=~`, `hasError=true`, `childCount=3`.

### Frame 4: grab-mutex.lisp

File: `/workspace/corpus_sources/commonlisp/benchmarks/grab-mutex.lisp`

Result:

- `accepted`, full EOF, not truncated
- root errors: Go `true`, C `true`
- first diff: `@root[20][0][2][10][4][3]`
- first-diff node: `str_lit` span `4155:4212`
- Go child count: 10
- C child count: 9

Go visible children: `"`, `~20a` error, `~8d`, `~8d`, `~8d`, `~@[`, `~d`, `~]`, `~%`, `"`.

C visible children: `"`, `~20a: cpu=(sum=~` error, `~8d`, `~8d`, `~@[`, `~d`, `~]`, `~%`, `"`.

Nested mismatch:

- Go child 1 ends at `4159` with text `~20a`.
- C child 1 spans `4155:4172`; its `format_directive_type` spans `4157:4172` with text `20a: cpu=(sum=~`.
- Go resumes a new `format_specifier` at byte `4171`.

### Frame 11: rwlbench2.lisp

File: `/workspace/corpus_sources/commonlisp/benchmarks/rwlbench2.lisp`

Result:

- `accepted`, full EOF, not truncated
- root errors: Go `true`, C `true`
- first diff: `@root[18][0][6][3]`
- first-diff node: `str_lit` span `3314:3341`
- Go child count: 6
- C child count: 5

Go visible children:

| index | kind | span | named | extra | hasError | text |
| ---: | --- | --- | --- | --- | --- | --- |
| 0 | `"` | `3314:3315` | false | false | false | `"` |
| 1 | `format_specifier` | `3315:3319` | true | false | true | `~10A` |
| 2 | `format_specifier` | `3322:3330` | true | false | true | `~15A | ~` |
| 3 | `format_specifier` | `3336:3338` | true | false | false | `~A` |
| 4 | `format_specifier` | `3338:3340` | true | false | false | `~%` |
| 5 | `"` | `3340:3341` | false | false | false | `"` |

C visible children:

| index | kind | span | named | extra | hasError | text |
| ---: | --- | --- | --- | --- | --- | --- |
| 0 | `"` | `3314:3315` | false | false | false | `"` |
| 1 | `format_specifier` | `3315:3323` | true | false | true | `~10A | ~` |
| 2 | `format_specifier` | `3329:3337` | true | false | true | `~15A | ~` |
| 3 | `format_specifier` | `3338:3340` | true | false | false | `~%` |
| 4 | `"` | `3340:3341` | false | false | false | `"` |

Nested mismatch:

- Go child 1 has an extra `ERROR` child over `3316:3318` with text `10`.
- C child 1 has `format_directive_type` spanning `3316:3323` with text `10A | ~`.

### Frame 20: inspect.lisp

File: `/workspace/corpus_sources/commonlisp/contrib/sb-aclrepl/inspect.lisp`

Result:

- `accepted`, full EOF, not truncated
- root errors: Go `true`, C `true`
- first diff: `@root[145][0][2][4][3]`
- first-diff node: `str_lit` span `24388:24422`
- Go child count: 7
- C child count: 6

Go visible children: `"`, `~A`, `~D`, `~:*` error, `~P`, `~A`, `"`.

C visible children: `"`, `~A`, `~D`, `~:*~` error, `~A`, `"`.

Nested mismatch:

- Go child 3 ends before the `~` at byte `24417`.
- C child 3 includes that `~` in `format_directive_type` span `24417:24418`; Go resumes a fresh `~P` sibling there.

### Frame 21: repl.lisp

File: `/workspace/corpus_sources/commonlisp/contrib/sb-aclrepl/repl.lisp`

Result:

- `accepted`, full EOF, not truncated
- root errors: Go `true`, C `true`
- first diff: `@root[14][3][3]`
- first-diff node: `str_lit` span `1081:1138`
- Go child count: 21
- C child count: 19

Relevant mismatch:

- Go has `~:*` error followed by a fresh `~D` sibling at byte `1094`.
- C has one wider errored child `~:*~`, then resumes at the later `~:[`.
- Go repeats the same pattern at byte `1105`, creating another extra `~D` sibling.
- C's corresponding errored child is `~:*:~`.

## Trace Evidence

Frame 1 was rerun with a reduction trace window around `2970:2985` and `GOT_C_RECOVERY=0`.

Key trace facts:

- At token `~` span `2975:2976`, stacks with `str_lit_repeat1` can either reduce the current repeat or shift into state `1766` as a repeated string-format item.
- The Go branch set includes shorter repeat reductions ending before the contested directive, for example raw spans `2959:2975`, `2961:2975`, `2968:2975`, and `2971:2975`.
- At the closing quote span `2992:2993`, the surviving branch set includes repeat reductions beginning at `2975`, `2978`, `2979`, `2982`, and `2983`.
- The final `str_lit` reduction has raw span `2958:2993`, but the child list already contains the shorter Go sibling sequence. That means the extra Go child is present before final result materialization of the `str_lit`.

Representative trace lines from `frame-0001-bbtrees-reduce-window.log`:

```text
C-REC-TRACE actionLookup stack=0 state=3963 tok_sym=14 tok_name="~" tok=2975:2976 action_count=2
C-REC-TRACE actionLookup action[0] type=1 state=0 symbol=173 symbol_name="str_lit_repeat1" child_count=2
C-REC-TRACE actionLookup action[1] type=0 state=1766 symbol=0 symbol_name="end" repetition=true
C-REC-TRACE reduceWindow ... tok=2975:2976 reduce_symbol=173 reduce_symbol_name="str_lit_repeat1" raw_span=2959:2975
C-REC-TRACE reduceWindow ... tok=2992:2993 reduce_symbol=173 reduce_symbol_name="str_lit_repeat1" raw_span=2975:2992
C-REC-TRACE reduceWindow ... tok=2993:3004 reduce_symbol=122 reduce_symbol_name="str_lit" raw_span=2958:2993
```

The trace therefore narrows the insertion point to baseline reduction/branch survival for errored string-format directives. It is not explained by transient parent flattening after parse completion, final result build alone, or the gated C-recovery error wrapping path.

## Verification Commands

All commands were run from `/home/draco/work/gotreesitter-build-baseline`. Docker runs were one frame at a time.

Frame 1 symbol audit:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label commonlisp-str-lit-audit-frame1 --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-audit-20260626 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=commonlisp REPRO_FILE=/workspace/corpus_sources/commonlisp/benchmarks/bbtrees.lisp REPRO_DIR=/workspace/corpus_sources REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-audit-20260626/frame-0001-bbtrees-symbol-audit.log"
```

Result: exit 0; `accepted`; full EOF; `str_lit` `2958:2993`, Go child count 10, C child count 9.

Frame 4 symbol audit:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label commonlisp-str-lit-audit-frame4 --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-audit-20260626 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=commonlisp REPRO_FILE=/workspace/corpus_sources/commonlisp/benchmarks/grab-mutex.lisp REPRO_DIR=/workspace/corpus_sources REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-audit-20260626/frame-0004-grab-mutex-symbol-audit.log"
```

Result: exit 0; `accepted`; full EOF; `str_lit` `4155:4212`, Go child count 10, C child count 9.

Frame 11 symbol audit:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label commonlisp-str-lit-audit-frame11 --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-audit-20260626 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=commonlisp REPRO_FILE=/workspace/corpus_sources/commonlisp/benchmarks/rwlbench2.lisp REPRO_DIR=/workspace/corpus_sources REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-audit-20260626/frame-0011-rwlbench2-symbol-audit.log"
```

Result: exit 0; `accepted`; full EOF; `str_lit` `3314:3341`, Go child count 6, C child count 5.

Frame 20 symbol audit:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label commonlisp-str-lit-audit-frame20 --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-audit-20260626 && /usr/bin/time -v timeout --kill-after=10s 120s env CGO_ENABLED=1 REPRO_LANG=commonlisp REPRO_FILE=/workspace/corpus_sources/commonlisp/contrib/sb-aclrepl/inspect.lisp REPRO_DIR=/workspace/corpus_sources REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-audit-20260626/frame-0020-inspect-symbol-audit.log"
```

Result: exit 0; `accepted`; full EOF; `str_lit` `24388:24422`, Go child count 7, C child count 6.

Frame 21 symbol audit:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label commonlisp-str-lit-audit-frame21 --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-audit-20260626 && /usr/bin/time -v timeout --kill-after=10s 120s env CGO_ENABLED=1 REPRO_LANG=commonlisp REPRO_FILE=/workspace/corpus_sources/commonlisp/contrib/sb-aclrepl/repl.lisp REPRO_DIR=/workspace/corpus_sources REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-audit-20260626/frame-0021-repl-symbol-audit.log"
```

Result: exit 0; `accepted`; full EOF; `str_lit` `1081:1138`, Go child count 21, C child count 19.

Frame 1 with C recovery disabled:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label commonlisp-str-lit-audit-frame1-crecovery-off --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-audit-20260626 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 GOT_C_RECOVERY=0 REPRO_LANG=commonlisp REPRO_FILE=/workspace/corpus_sources/commonlisp/benchmarks/bbtrees.lisp REPRO_DIR=/workspace/corpus_sources REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-audit-20260626/frame-0001-bbtrees-symbol-audit-crecovery-off.log"
```

Result: exit 0; same first diff as default, `str_lit` `2958:2993`, Go child count 10, C child count 9.

Frame 1 with C recovery forced for all languages:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label commonlisp-str-lit-audit-frame1-crecovery-all --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-audit-20260626 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=2970:2985 REPRO_LANG=commonlisp REPRO_FILE=/workspace/corpus_sources/commonlisp/benchmarks/bbtrees.lisp REPRO_DIR=/workspace/corpus_sources REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-audit-20260626/frame-0001-bbtrees-symbol-audit-crecovery-all.log"
```

Result: exit 0; worse shape, first diff at root: Go `ERROR [0:7395]` child count 183, C `source [0:7395]` child count 38.

Frame 1 reduction trace:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label commonlisp-str-lit-reduce-window-frame1 --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-audit-20260626 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 GOT_C_RECOVERY=0 GOT_C_RECOVERY_TRACE_WINDOW=2970:2985 REPRO_LANG=commonlisp REPRO_FILE=/workspace/corpus_sources/commonlisp/benchmarks/bbtrees.lisp REPRO_DIR=/workspace/corpus_sources REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-audit-20260626/frame-0001-bbtrees-reduce-window.log"
```

Result: exit 0; same first diff as default; trace shows shorter `str_lit_repeat1` reductions are present before the final `str_lit` reduction.

Clean Common Lisp control frame 2:

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label commonlisp-str-lit-audit-frame2-control --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-audit-20260626 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=commonlisp REPRO_FILE=/workspace/corpus_sources/commonlisp/benchmarks/finalize.lisp REPRO_DIR=/workspace/corpus_sources REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-audit-20260626/frame-0002-finalize-control.log"
```

Result: exit 0; `accepted`; full EOF; root errors false; no structural divergence.

## Next Generalized Invariant To Test

Test a result-selection or reduction-survival invariant for same-parent errored children:

When two surviving candidates have the same visible parent span and the same left context, and the first visible mismatch is an error-bearing named child with the same start byte, compare the branch that keeps the next would-be sibling starter token inside the errored child against the branch that resumes a new sibling at that token. C appears to prefer the wider errored child in these `str_lit` cases.

This invariant must remain language-agnostic. It should be validated with:

- a synthetic or existing reduced fixture that has same-parent sibling starter ambiguity under an errored child,
- the five Common Lisp `str_lit` frames above,
- clean Common Lisp frame 2,
- at least one non-Common-Lisp recovery/materialization control before any parser change is committed.

