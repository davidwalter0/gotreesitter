# Common Lisp str_lit error-child widening experiment, 2026-06-26

Scope: bounded proof experiment for the generalized Common Lisp `str_lit`
residual invariant. No per-grammar normalizer, language-name parser policy, or
default-on parser behavior was kept.

Worktree: `/home/draco/work/gotreesitter-build-baseline`

Source head before experiment: `539e8123`

## Conclusion

The narrow final result-ordering experiment did not move any of the five
repeated Common Lisp `str_lit` residual frames.

Temporary code added a default-off `GOT_RESULT_ERROR_CHILD_WIDENING=1` branch in
`compareStackEntryCSubtreeOrder`. The branch only applied to same-symbol,
same-span candidate parents and preferred the candidate whose same-start child
was wider, error-bearing, and covered the start of a later same-symbol sibling
in the other candidate. This was intentionally narrower than a child-count
preference inversion.

With that flag enabled:

| Control | Result |
| --- | --- |
| Common Lisp frame 1 | unchanged `str_lit` first diff, Go child count 10 vs C 9 |
| Common Lisp frame 4 | unchanged `str_lit` first diff, Go child count 10 vs C 9 |
| Common Lisp frame 11 | unchanged `str_lit` first diff, Go child count 6 vs C 5 |
| Common Lisp frame 20 | unchanged `str_lit` first diff, Go child count 7 vs C 6 |
| Common Lisp frame 21 | unchanged `str_lit` first diff, Go child count 21 vs C 19 |
| Common Lisp clean frame 2 | still no structural divergence |
| CUDA frame 1 recovery/materialization control | unchanged known root diff with Go trailing `ERROR [8796:8797]` |

The temporary parser code was reverted. This report is the only committed
artifact from the experiment.

Common Lisp remains Tier IV.

## Interpretation

The experiment does not support landing a real generalized parser-machinery fix
in final result ordering alone. The result is consistent with the prior audit:
the C-like wider errored `format_specifier` branch is either not present in the
final result candidate set, or is not competing at the `compareStackCSubtreeOrder`
surface where same-symbol/same-span parent ordering can choose it.

The next invariant should be checked earlier:

> At the contested `~` token, a candidate reduction that builds or retains the
> wider error-bearing child must survive through the repeated-child reduction
> (`str_lit_repeat1` in Common Lisp, but the invariant must be symbol-agnostic)
> until final result selection. If the wider child is lost before
> `compareStackCSubtreeOrder`, the repair belongs in reduction survival or
> same-pop/result-child selection, not final tree materialization or a
> per-language normalizer.

## Temporary code tested and reverted

The reverted proof hook was default-off:

```text
GOT_RESULT_ERROR_CHILD_WIDENING=1
```

It inserted a pre-child-count comparator condition for same-symbol/same-span
parents:

- find one candidate child with the same symbol and same start byte as the other;
- require the preferred child to end later and have errors;
- require the other candidate to have a later same-symbol sibling starting
  inside the wider errored child span;
- prefer the wider errored child candidate only under that condition.

No default-on behavior was kept because the proof matrix showed no material
residual improvement.

## Verification

Host compile/behavior smoke while the temporary flag existed:

```bash
go test . -run '^TestBuildResultFromGLRUsesCSubtreeOrderOnScoreTie$' -count=1
```

Result:

```text
ok  	github.com/odvcencio/gotreesitter	0.005s
```

All Docker runs were one grammar/frame at a time.

### Common Lisp frame 1

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label commonlisp-str-lit-widen-frame1 --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-widen-experiment-20260626 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 GOT_RESULT_ERROR_CHILD_WIDENING=1 REPRO_LANG=commonlisp REPRO_FILE=/workspace/corpus_sources/commonlisp/benchmarks/bbtrees.lisp REPRO_DIR=/workspace/corpus_sources REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-widen-experiment-20260626/frame-0001-bbtrees.log"
```

Result: exit 0, no OOM, accepted/full EOF, unchanged first diff:
`@root[23][3][3]`, `str_lit [2958:2993]`, Go child count 10, C child count 9.
Max RSS: 1319604 KB.

Artifact:
`harness_out/docker/20260626T025946Z-commonlisp-str-lit-widen-frame1`

### Common Lisp frame 4

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label commonlisp-str-lit-widen-frame4 --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-widen-experiment-20260626 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 GOT_RESULT_ERROR_CHILD_WIDENING=1 REPRO_LANG=commonlisp REPRO_FILE=/workspace/corpus_sources/commonlisp/benchmarks/grab-mutex.lisp REPRO_DIR=/workspace/corpus_sources REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-widen-experiment-20260626/frame-0004-grab-mutex.log"
```

Result: exit 0, no OOM, accepted/full EOF, unchanged first diff:
`@root[20][0][2][10][4][3]`, `str_lit [4155:4212]`, Go child count 10, C
child count 9. Max RSS: 1322648 KB.

Artifact:
`harness_out/docker/20260626T030031Z-commonlisp-str-lit-widen-frame4`

### Common Lisp frame 11

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label commonlisp-str-lit-widen-frame11 --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-widen-experiment-20260626 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 GOT_RESULT_ERROR_CHILD_WIDENING=1 REPRO_LANG=commonlisp REPRO_FILE=/workspace/corpus_sources/commonlisp/benchmarks/rwlbench2.lisp REPRO_DIR=/workspace/corpus_sources REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-widen-experiment-20260626/frame-0011-rwlbench2.log"
```

Result: exit 0, no OOM, accepted/full EOF, unchanged first diff:
`@root[18][0][6][3]`, `str_lit [3314:3341]`, Go child count 6, C child count 5.
Max RSS: 1315176 KB.

Artifact:
`harness_out/docker/20260626T030136Z-commonlisp-str-lit-widen-frame11`

### Common Lisp frame 20

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label commonlisp-str-lit-widen-frame20 --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-widen-experiment-20260626 && /usr/bin/time -v timeout --kill-after=10s 120s env CGO_ENABLED=1 GOT_RESULT_ERROR_CHILD_WIDENING=1 REPRO_LANG=commonlisp REPRO_FILE=/workspace/corpus_sources/commonlisp/contrib/sb-aclrepl/inspect.lisp REPRO_DIR=/workspace/corpus_sources REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-widen-experiment-20260626/frame-0020-inspect.log"
```

Result: exit 0, no OOM, accepted/full EOF, unchanged first diff:
`@root[145][0][2][4][3]`, `str_lit [24388:24422]`, Go child count 7, C child
count 6. Max RSS: 1995416 KB.

Artifact:
`harness_out/docker/20260626T030218Z-commonlisp-str-lit-widen-frame20`

### Common Lisp frame 21

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label commonlisp-str-lit-widen-frame21 --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-widen-experiment-20260626 && /usr/bin/time -v timeout --kill-after=10s 120s env CGO_ENABLED=1 GOT_RESULT_ERROR_CHILD_WIDENING=1 REPRO_LANG=commonlisp REPRO_FILE=/workspace/corpus_sources/commonlisp/contrib/sb-aclrepl/repl.lisp REPRO_DIR=/workspace/corpus_sources REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-widen-experiment-20260626/frame-0021-repl.log"
```

Result: exit 0, no OOM, accepted/full EOF, unchanged first diff:
`@root[14][3][3]`, `str_lit [1081:1138]`, Go child count 21, C child count 19.
Max RSS: 1685164 KB.

Artifact:
`harness_out/docker/20260626T030412Z-commonlisp-str-lit-widen-frame21`

### Clean Common Lisp frame 2

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label commonlisp-str-lit-widen-frame2-control --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-widen-experiment-20260626 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 GOT_RESULT_ERROR_CHILD_WIDENING=1 REPRO_LANG=commonlisp REPRO_FILE=/workspace/corpus_sources/commonlisp/benchmarks/finalize.lisp REPRO_DIR=/workspace/corpus_sources REPRO_SYMBOL_AUDIT=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-widen-experiment-20260626/frame-0002-finalize-control.log"
```

Result: exit 0, no OOM, accepted/full EOF, no structural divergence. Max RSS:
1170400 KB.

Artifact:
`harness_out/docker/20260626T030548Z-commonlisp-str-lit-widen-frame2-control`

### CUDA frame 1 control

This control was chosen from
`docs/reports/cuda-frame1-faithful-condense-merge-cap-diagnostics-20260626.md`,
where CUDA frame 1 is documented as a known recovery/materialization residual
with a trailing top-level `ERROR`.

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label cuda-frame1-result-error-widen-control --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/commonlisp-str-lit-widen-experiment-20260626 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 GOT_RESULT_ERROR_CHILD_WIDENING=1 GOT_C_RECOVERY=all GOT_GLR_TOKEN_FRONTIER_DISPATCH=1 GOT_GLR_MAX_STACKS=64 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/commonlisp-str-lit-widen-experiment-20260626/cuda-frame-0001-control.log"
```

Result: exit 0, no OOM, accepted/full EOF, unchanged known first diff:
`@root`, Go `translation_unit [0:12254]` child count 26, C child count 25. Go
has trailing `ERROR [8796:8797]`; C keeps `template_declaration [8510:8797]`.
Max RSS: 1122608 KB.

Artifact:
`harness_out/docker/20260626T030610Z-cuda-frame1-result-error-widen-control`
