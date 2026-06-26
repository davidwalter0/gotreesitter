# PureScript Semigroup Accepted Type Shape Probe

Date: 2026-06-26

Workspace: `/home/draco/work/gotreesitter-build-baseline`

Base HEAD:

```text
08baeea0647c8baf434ccb0a122c6fa326070310
```

## Question

Narrow the next PureScript residual around:

```purescript
foreign import concatArray :: forall a. Array a -> Array a -> Array a
```

The target was to compare accepted/materializing shapes for the hidden `_type`
covering the type annotation payload, especially the C shape:

```text
type_infix [1913:1952]
  forall [1913:1922]
  type_apply [1923:1930]
  type_operator [1931:1933]
  type_infix [1934:1952]
```

## Result

No code fix was applied.

The C-compatible `type_infix [1913:1952]` is proven in the C output, but it was
not found as a Go accepted-stack candidate. The closest Go candidate is a hidden
`_type` reduction over two visible payloads, `forall [1913:1922]` and
`type_infix [1923:1952]`. After hidden flattening through `foreign_import`, this
becomes the observed Go child-count `6` shape.

Because the C-compatible subtree is absent before final result selection, the
next safe invariant is not a final accepted-stack/result-election rule. A
grammar-agnostic fix would need to explain why C can reduce an infix node with a
full-span left operand at this point while Go keeps the quantifier as a sibling.
That invariant is not proven here.

## Go Candidate Evidence

Clean first-diff replay:

```text
FIRST-DIFF @root[38]
go: foreign_import [1883:1952] cc=6
c : foreign_import [1883:1952] cc=5

Go children:
  foreign [1883:1890]
  import [1891:1897]
  variable [1898:1909]
  :: [1910:1912]
  forall [1913:1922]
  type_infix [1923:1952]

C children:
  foreign [1883:1890]
  import [1891:1897]
  variable [1898:1909]
  :: [1910:1912]
  type_infix [1913:1952]
```

The Go reduce-window trace for `1913:1952` shows the relevant alternatives
near the final foreign import reduction. Dynamic precedence is `0` for all rows
below.

| Shape | Reduce window evidence |
| --- | --- |
| standalone quantifier | `forall`, `child_count=1`, `top_state=1123`, `target_state=1125`, payload `[1913:1922]` |
| tail infix | `type_infix`, `child_count=3`, `top_state=1125`, `target_state=6162`, payload starts at `1923` |
| hidden whole type, Go shape | `_type`, `child_count=2`, `top_state=1123`, `target_state=6028`, left payload starts at `1913` |
| tail hidden type | `_type`, `child_count=1`, `top_state=1125`, `target_state=6165`, payload starts at `1923` |
| C-compatible full infix | not observed in Go trace |

The trace lines use `raw_span=...:2084` for some rows because the reduce window
includes trailing comment extras before `_layout_semicolon`; the non-extra
payload is the `foreign_import [1883:1952]` subtree proven by the clean first
diff and accepted-stack dump.

Temporary uncommitted accepted-stack instrumentation was used and then reverted.
It showed two accepted stacks:

```text
ACCEPTED-STACK-DUMP window=1913:1952 stacks=2
ACCEPTED-STACK index=0 score=50 depth=2 byte=2925 branch=1063
ACCEPTED-STACK index=1 score=50 depth=2 byte=2925 branch=1067
```

Both accepted stacks had the same materializing subtree at the witness:

```text
foreign_import [1883:1952] cc=6
  forall [1913:1922] cc=3
  type_infix [1923:1952] cc=3
```

No accepted stack carried `type_infix [1913:1952]`.

## C Evidence

The C tree dump confirms the expected shape:

```text
C foreign_import [1883:1952] cc=5
  type_infix [1913:1952] cc=4
    forall [1913:1922]
    type_apply [1923:1930]
    type_operator [1931:1933]
    type_infix [1934:1952]
```

The C parser log also shows `type_infix` reductions in the relevant type region,
but the public logger does not include byte spans or dynamic precedence on the
reduce lines. Therefore the C-compatible full-span tree is proven from the C
tree, while exact C reduce-window provenance remains a follow-up instrumentation
task if this residual is pursued further.

## Interpretation

This is not the hidden-flattening/post-result class. The Go accepted tree is
already wrong before final visible-child comparison:

- Go creates/retains a hidden `_type` shape whose visible descendants are
  `forall` plus a tail `type_infix`.
- C creates a visible `type_infix` whose first child is the `forall` node.
- Existing C-style subtree ordering cannot elect a candidate that is not
  present among Go accepted stacks.

An unsafe fix would special-case PureScript quantifier/infix structure or
post-wrap `forall`. That would violate the generalized-parser-only constraint
and would not explain the related `constraint` / `type_apply` residual observed
in the prior forest-mode audit.

## Next Action

Instrument a reusable, grammar-agnostic reduce provenance probe that records, for
all reduce candidates overlapping a byte window:

- stack/version id before and after reduce,
- source state, goto state, symbol, child count, production id, dynamic
  precedence,
- non-extra payload span separate from trailing extras,
- whether the candidate survives merge/cull/condense,
- C logger equivalent, if possible, with byte spans.

The next invariant to test is pre-result reduce-candidate survival, not final
result selection. Specifically, compare C condense/selection after the competing
`_type` and `type_infix` reductions with Go's merge/cull state for the same
state/byte keys.

## Validation

Clean Semigroup control:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label purescript-semigroup-clean-control-20260626 -- \
  "cd /workspace/cgo_harness && REPRO_LANG=purescript REPRO_FILE=/workspace/corpus_sources/purescript/src/Data/Semigroup.purs REPRO_SYMBOL_AUDIT=1 go test . -tags treesitter_c_parity -run '^TestFirstDiffDiag$' -count=1 -v"
```

Result: accepted, non-truncated, no root errors, first diff remains
`foreign_import [1883:1952]` child-count `6/5`.

Clean HeytingAlgebra control:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label purescript-heyting-clean-control-20260626 -- \
  "cd /workspace/cgo_harness && REPRO_LANG=purescript REPRO_FILE=/workspace/corpus_sources/purescript/src/Data/HeytingAlgebra.purs REPRO_SYMBOL_AUDIT=1 go test . -tags treesitter_c_parity -run '^TestFirstDiffDiag$' -count=1 -v"
```

Result: accepted, non-truncated, no structural divergence.

Additional diagnostics:

- Go reduce-window trace:
  `harness_out/docker/20260626T021420Z-purescript-semigroup-type-window-trace`
- Temporary accepted-stack dump:
  `harness_out/docker/20260626T021606Z-purescript-semigroup-accepted-stack-dump`
- C parser/tree log:
  `harness_out/docker/20260626T021646Z-purescript-semigroup-c-forall-type-log`

PureScript N=40 was not rerun because no code fix was applied and Semigroup did
not clear. The high-value Tier IV set is unchanged.
