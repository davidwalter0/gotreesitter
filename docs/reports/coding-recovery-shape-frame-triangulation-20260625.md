# Coding Recovery-Shape Frame Triangulation - 2026-06-25

Scope: report-only methodology and next-target artifact for the 206 residual goal. No parser behavior, grammar policy, or normalizer changes are included here.

## Methodology

The current 206 goal should be triaged frame-first, then domain-second. The frame says which generalized parser machinery is failing: runtime frontier survival, accepted materialization, recovery error/cost, post-retry recovery shape, forest C-recovery election, or corpus accounting. The domain then decides priority: true programming languages outrank data/config/prose residuals when the frame evidence is comparable.

This keeps the next slice aligned with the stated policy: generalized parser machinery only. The ledger is not evidence for per-grammar normalizers, language-name branches, or language-specific result policy. A candidate fix must explain a parser-mechanics failure that can carry across grammars or be rejected.

## High-Priority True-Coding Lanes

Source ledger: `docs/reports/206-residual-frame-domain-ledger-20260625.md` and `docs/reports/206-residual-frame-domain-ledger-20260625.tsv`. Latest report commit `c46e2aed` classifies the Scala `GenerateFunctionConverters.scala` witness as `faithful_forest_c_recovery_summary_election_error_cost_preservation`.

| Lane | Ledger count | High-priority true-coding representatives | Current read |
| --- | ---: | --- | --- |
| Runtime frontier | 37 | `uxntal`, `asm`, `nushell`, `wat`, `disassembly`, `brightscript`, `powershell`, `cairo`, `teal`, `groovy`, `cobol`, `haxe`, `matlab`, `commonlisp` | Still a broad GLR frontier-survival lane, but many rows only have current-smoke evidence. |
| Accepted materialization | 11 | `fennel`, `fsharp`, `perl`, `swift`, `purescript` | Accepted parses diverge without the C-recovery retry shape now visible in C++/CUDA/GLSL. |
| Recovery error/cost | 7 | `objc`, `odin`, `wolfram`, `pascal` | Direct recovery tree or error-count divergence under accepted parses. |
| Post-retry recovery-shape | 2 | `cpp`, `glsl` in the committed ledger, with `cuda` as an adjacent post-retry/scoped-first coding control | Former runtime-frontier rows migrated to accepted/non-truncated residual recovery shape after generalized retry. |
| Scala forest C-recovery | 1 | `scala` | Forest C-recovery summary election/error-cost preservation, now explicitly classified by `c46e2aed`. |

## Next Proof Slice

Select C++/CUDA/GLSL post-retry recovery-shape as the next generalized proof slice.

The reason is narrow and testable: all three true-coding controls now reach EOF, accept, avoid truncation, and report no terminal failures in the post-retry control report. Their residual is no longer a runtime frontier stop; it is recovery shape, root child-count, or materialized `ERROR` boundary after C-recovery retry.

N=40 control counts from `docs/reports/post-retry-coding-controls-n40-20260625.md`:

| Grammar | Parity | Diverge | Trunc | ErrTree | Terminal fail/panic | Family counts |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| `cpp` | 10/40 | 30 | 0 | 24 | 0 | clean=10, recovery_error_shape=20, version_or_corpus=10 |
| `cuda` | 21/40 | 19 | 0 | 7 | 0 | clean=21, recovery_error_shape=7, version_or_corpus=12 |
| `glsl` | 14/40 | 26 | 0 | 27 | 0 | clean=14, recovery_error_shape=9, version_or_corpus=17 |

The same report records `frontierStopCount=0`, `terminalCount=0`, `truncated=false`, and `stopReasons=["accepted"]` for all three diagnostic summaries. That makes this a better next machinery proof than jumping to the larger current-smoke runtime frontier: it already isolates accepted recovery materialization after a generalized retry.

## Representative Evidence

### C++: `args.h`

Primary first-diff artifact:

`cgo_harness/harness_out/cpp-recovery-retry-default-witness-20260625-variant1/firstdiff/frame-0001.log`

The witness file is `/workspace/corpus_sources/cpp/include/fmt/args.h`. The Go parse is accepted, reaches EOF, is non-truncated, and has `rootHasError=true`; C also has a root error, so the useful signal is the finer recovery shape.

First diff:

- `FIRST-DIFF @root[6]`
- Go `preproc_ifdef [209:7175]` has child count 7.
- C `preproc_ifdef [209:7175]` has child count 9.
- Go child 5 is `ERROR [433:7167]`.
- C child 5 is `function_definition [433:7147]`.
- C then materializes `expression_statement [7147:7148]` and `expression_statement [7150:7167]` before `#endif`.

This is the clearest C++ shape: Go wraps a broad namespace/template region in one `ERROR`, while C recovers into a function definition plus trailing expression statements under the same preprocessor parent.

### CUDA: `UnifiedMemoryStreams.cu`

Primary report and first-diff artifacts:

- `docs/reports/cuda-recovery-retry-witness-20260625.md`
- `cgo_harness/harness_out/cuda-recovery-retry-witness-20260625/firstdiff/frame-0001.log`

The witness file is `/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu`. The parse is accepted, reaches EOF at `12254/12254`, and is non-truncated. The report records that Go errors exceed C: baseline/`crecovery_off` had `goErrors=17`, `goMissing=11`, while C had `0/0`; `crecovery_all` reduced Go to `goErrors=1`, `goMissing=2`, but still did not reach parity.

First diff:

- `FIRST-DIFF @root`
- Go `translation_unit [0:12254]` has child count 26.
- C `translation_unit [0:12254]` has child count 25.
- Go extra child 23 is `ERROR [8510:8585]` for the template function header.
- Go child 24 is `compound_statement [8586:8797]`.
- C child 23 is `template_declaration [8510:8797]`.

This is the best lead because C has no errors, while Go splits one C template declaration into an `ERROR` header and separate compound statement after retry.

### GLSL: `100.frag`

Primary report:

`docs/reports/glsl-recovery-frontier-witness-20260625.md`

The witness file is `/workspace/corpus_sources/glsl/Test/100.frag`. The post-patch baseline reaches EOF, is accepted, and is non-truncated: tokens `1014`, last/EOF `5477/5477`, root pair `translation_unit->translation_unit`. The first diff is root child-count. Go has `6` errors and `3` missing nodes; C has `5` errors and `2` missing nodes.

GLSL is therefore a useful secondary control, not the root-cause lead. It confirms post-retry migration out of the runtime frontier, but real C-side errors make it noisier than C++/CUDA for proving the generalized recovery-shape mechanism.

## Proposed Generalized Proof

Add or run diagnostics that compare C-recovery resume/recover-to-state/materialized `ERROR` boundaries at the first extra Go `ERROR` child. For the C++ `args.h` and CUDA `UnifiedMemoryStreams.cu` witnesses, the diagnostic should capture:

- the C-recovery candidate stack before recovery election,
- the chosen recover-to-state and resume token,
- whether the same-token redispatch path is used,
- the byte range that becomes a materialized `ERROR`,
- the first recovered named node C materializes at the same byte range,
- the rejected alternative stacks and their error costs.

The proof target is to decide which generalized mechanism is wrong:

- Go wraps too much source into a materialized `ERROR`;
- Go resumes from the wrong recover-to-state;
- Go elects a worse recovered stack despite an available C-like continuation.

The implementation target remains generalized parser machinery. No grammar-specific fix, per-grammar normalizer, or language-name policy follows from this report.

## Falsification Criteria

Reject or reframe this proof slice if any of these are true:

- C++ and CUDA do not share recover-to-state, resume, or materialized-`ERROR` mechanics at the first extra Go error child.
- Recovery-state election already matches C and the divergence is instead grammar version, corpus mismatch, or oracle behavior.
- A generalized change improves only one grammar and has no shared explanation across C++/CUDA/GLSL controls.
- A generalized change creates new clean regressions in accepted, non-error frames.

## Validation

Commands run while preparing this report:

```sh
git status --short
rg --files docs/reports cgo_harness/harness_out | rg 'post-retry-coding-controls|cuda-recovery-retry-witness|glsl-recovery-frontier-witness|cpp-recovery-retry-default-witness|ledger|206|frame'
sed -n '1,240p' docs/reports/post-retry-coding-controls-n40-20260625.md
sed -n '1,220p' docs/reports/cuda-recovery-retry-witness-20260625.md
sed -n '1,220p' docs/reports/glsl-recovery-frontier-witness-20260625.md
sed -n '1,220p' docs/reports/206-residual-frame-domain-ledger-20260625.md
sed -n '1,220p' cgo_harness/harness_out/cpp-recovery-retry-default-witness-20260625-variant1/firstdiff/frame-0001.log
sed -n '1,180p' cgo_harness/harness_out/cuda-recovery-retry-witness-20260625/firstdiff/frame-0001.log
git show --stat --oneline --no-renames c46e2aed -- docs/reports/206-residual-frame-domain-ledger-20260625.md docs/reports/206-residual-frame-domain-ledger-20260625.tsv
git diff --check
```

No Docker was run. This artifact reuses existing reports and harness outputs only.
