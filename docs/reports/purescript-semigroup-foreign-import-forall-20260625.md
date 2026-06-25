# PureScript Semigroup Foreign Import Forall Audit

Date: 2026-06-25

Isolate: `/home/draco/work/gotreesitter-build-baseline`

Base commit: `857b8561db6af94cc412eb52b6b1b4ab53b31e95`

Main repo changes: this report only.

## Question

Investigate the next PureScript Semigroup accepted-shape residual around:

```purescript
foreign import concatArray :: forall a. Array a -> Array a -> Array a
```

The target was to decide whether the mismatch is caused by action/stack election,
reduce grouping/window selection, hidden flattening, alias materialization, raw
span handling, or post-result materialization, then apply a narrow generalized
fix only if a C-compatible invariant was clear.

## Reproduction

Semigroup first-diff diagnostic:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label purescript-semigroup-repro \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && REPRO_LANG=purescript REPRO_FILE=/workspace/corpus_sources/purescript/src/Data/Semigroup.purs REPRO_SYMBOL_AUDIT=1 go test . -tags treesitter_c_parity -run '^TestFirstDiffDiag$' -count=1 -v"
```

Artifact:
`harness_out/docker/20260625T181710Z-purescript-semigroup-repro`

Result:

- parser runtime accepted
- non-truncated
- no Go or C root error
- first structural diff remains `FIRST-DIFF @root[38]`
- Go `foreign_import` `[1883:1952]` has `cc=6`
- C `foreign_import` `[1883:1952]` has `cc=5`

Shape at the mismatch:

```text
Go foreign_import
  foreign
  import
  variable
  ::
  forall [1913:1922]
  type_infix [1923:1952]

C foreign_import
  foreign
  import
  variable
  ::
  type_infix [1913:1952]
    forall [1913:1922]
    type_apply [1923:1930]
    type_operator [1931:1933]
    type_infix [1934:1952]
```

## Layer Isolation

Materialization toggles did not move the Semigroup first diff:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label purescript-semigroup-no-pending-parents \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && REPRO_LANG=purescript REPRO_FILE=/workspace/corpus_sources/purescript/src/Data/Semigroup.purs REPRO_SYMBOL_AUDIT=1 GOT_GLR_V2_PENDING_PARENTS=0 go test . -tags treesitter_c_parity -run '^TestFirstDiffDiag$' -count=1 -v"
```

Artifact:
`harness_out/docker/20260625T181856Z-purescript-semigroup-no-pending-parents`

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label purescript-semigroup-mat-off \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && REPRO_LANG=purescript REPRO_FILE=/workspace/corpus_sources/purescript/src/Data/Semigroup.purs REPRO_SYMBOL_AUDIT=1 GOT_GLR_V2_PENDING_PARENTS=0 GOT_GLR_V2_FINAL_CHILD_REFS=0 GOT_GLR_V2_COMPACT_FULL_LEAVES=0 go test . -tags treesitter_c_parity -run '^TestFirstDiffDiag$' -count=1 -v"
```

Artifact:
`harness_out/docker/20260625T181912Z-purescript-semigroup-mat-off`

Both runs kept the same `foreign_import` split. That rules out the current
pending-parent, final-child-ref, compact-leaf, and post-result hidden-flattening
paths as the direct cause.

Stack and merge-cap sweeps also did not recover the C shape:

- `GOT_GLR_MAX_STACKS=64`: same first diff
  (`harness_out/docker/20260625T182110Z-purescript-semigroup-stacks64`)
- `GOT_GLR_MAX_MERGE_PER_KEY=1`: same first diff
  (`harness_out/docker/20260625T182110Z-purescript-semigroup-merge1`)
- `GOT_GLR_MAX_MERGE_PER_KEY=24`: moved to an earlier mismatch at
  `@root[30][5][2]`, so it is not a fix path
  (`harness_out/docker/20260625T182110Z-purescript-semigroup-merge24`)

Forest mode moved the first diff earlier into the same PureScript type-expression
ambiguity class:

```text
FIRST-DIFF @root[36][3][0][1]
Go constraint [1693:1714] cc=3: class_name, type_name, type_name
C  constraint [1693:1714] cc=2: class_name, type_apply row list
```

Artifact:
`harness_out/docker/20260625T182147Z-purescript-semigroup-forest`

The C tree dump with parser logging confirmed that C materializes one visible
`type_infix` under `foreign_import`, with `forall` as the left operand:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label purescript-semigroup-c-log \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && set -o pipefail && REPRO_LANG=purescript REPRO_FILE=/workspace/corpus_sources/purescript/src/Data/Semigroup.purs REPRO_CLOG=1 go test . -tags treesitter_c_parity -run '^TestCTreeDumpDiag$' -count=1 -v 2>&1 | grep -E '191[0-9]|192[0-9]|193[0-9]|194[0-9]|195[0-2]|foreign_import|type_infix|type_apply|forall|_type|reduce|accept'"
```

Artifact:
`harness_out/docker/20260625T182210Z-purescript-semigroup-c-log`

The PureScript symbol/action audit found:

- visible named `forall` symbol `128`
- visible named `type_infix` symbol `139`
- hidden named `_type` symbol `140`
- `foreign_import` reduce uses `child_count=4`
- `type_infix` reductions are available with `child_count=3`

That means the `foreign_import` RHS still reduces through one hidden `_type`.
The divergent visible output is not because `foreign_import` itself expects a
different child count. The Go accepted stack has selected an `_type` shape whose
flattened visible children are `forall` plus the trailing `type_infix`; C selects
an `_type`/`type_infix` shape spanning the whole `forall ... -> ...` expression.

## Control Witness

HeytingAlgebra remains structurally clean:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label purescript-heyting-nonregression \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && REPRO_LANG=purescript REPRO_FILE=/workspace/corpus_sources/purescript/src/Data/HeytingAlgebra.purs REPRO_SYMBOL_AUDIT=1 go test . -tags treesitter_c_parity -run '^TestFirstDiffDiag$' -count=1 -v"
```

Artifact:
`harness_out/docker/20260625T182356Z-purescript-heyting-nonregression`

Result:

```text
(no structural divergence)
```

## Focused Verification

No reducer or materialization helper was changed in this pass. A focused root
package compile-only test still passes:

```sh
go test . -run '^$' -count=1
```

Result:

```text
ok github.com/odvcencio/gotreesitter 0.003s [no tests to run]
```

## Conclusion

No code fix was applied.

The exact residual is pre-result parse election for ambiguous PureScript type
expressions, most likely reduce grouping/window or accepted stack election before
final tree materialization. The current evidence rules out hidden flattening,
alias materialization, raw-span handling, pending-parent materialization, final
child refs, and compact full leaves as the direct cause for this witness.

A safe generalized invariant is not clear yet. In particular, forcing a
post-result wrapper around `forall` would be a PureScript-specific shape
normalizer and would not address the earlier forest-mode `constraint` /
`type_apply` residual. The next useful diagnostic should compare candidate
accepted stack materializing shapes for the hidden `_type` spanning
`[1913:1952]`, including their reduce-window provenance and parse-state path,
then decide whether C's preference can be expressed as a grammar-agnostic
subtree election rule.

Because Semigroup is not clean, the PureScript N=40 classification was skipped.
