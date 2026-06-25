# PureScript Alias-Target Unary Wrapper Residual

Date: 2026-06-25

## Witnesses

- `/home/draco/work/gotreesitter-corpora/corpus_sources/purescript/src/Data/HeytingAlgebra.purs`
- `/home/draco/work/gotreesitter-corpora/corpus_sources/purescript/src/Data/Semigroup.purs`

Starting HEAD:

```text
28fd414b3c89409e875537c8f78678c500efeb72 fix(glr): use C-style subtree order for merge election
```

## Root Cause

The PureScript wildcard residual was a generalized alias/materialization issue in
the invisible unary wrapper fast path.

The C parser has `ts_non_terminal_alias_map` coverage for `wildcard ->
pat_wildcard`, and the production alias table also aliases the anonymous `_`
token to `pat_wildcard`. In single-pattern function heads, the Go reducer
collapsed the invisible `_fun_patterns -> pat_wildcard` unary wrapper before the
enclosing `alias($._fun_patterns, $.patterns)` production could see it. That made
the later alias apply to the visible `pat_wildcard` descendant directly. C keeps
the wrapper boundary, so it produces:

```text
patterns -> pat_wildcard -> pat_wildcard -> "_"
```

while Go previously produced:

```text
patterns -> pat_wildcard -> "_"
```

The same issue did not appear for multi-pattern heads because `_fun_patterns`
then has multiple children and the unary collapse path cannot fire.

## Fix

Invisible unary wrapper collapse now refuses to collapse through a child symbol
that is already known as an alias target in the loaded language metadata. This
uses the existing generalized `AliasSequences`-derived alias-target table; it
does not add a PureScript switch or a post-result normalizer.

Focused tests added:

- `TestCollapsibleRawUnarySelfReductionKeepsAliasTargetChild`
- `TestCollapsibleRawUnarySelfReductionEntryKeepsAliasTargetCompactLeaf`

## Validation

Local reducer/materialization slice:

```sh
go test . -run '^Test(CollapsibleRawUnarySelfReduction|CollapsibleUnarySelfReduction|AliasedHidden)' -count=1
```

Result: pass.

Docker reducer/materialization slice:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --label alias-target-unary-unit -- \
  "cd /workspace && go test . -run '^Test(CollapsibleRawUnarySelfReduction|CollapsibleUnarySelfReduction|AliasedHidden)' -count=1"
```

Result: pass, `oom_killed=false`.

HeytingAlgebra first-diff replay:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label purescript-heyting-wildcard-fix-audit -- \
  "cd /workspace/cgo_harness && REPRO_LANG=purescript REPRO_FILE=/workspace/corpus_sources/purescript/src/Data/HeytingAlgebra.purs REPRO_SYMBOL_AUDIT=1 go test . -tags treesitter_c_parity -run '^TestFirstDiffDiag$' -count=1 -v"
```

Result: no structural divergence, accepted, non-truncated, no root errors.

Semigroup first-diff replay:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label purescript-semigroup-wildcard-fix-audit -- \
  "cd /workspace/cgo_harness && REPRO_LANG=purescript REPRO_FILE=/workspace/corpus_sources/purescript/src/Data/Semigroup.purs REPRO_SYMBOL_AUDIT=1 go test . -tags treesitter_c_parity -run '^TestFirstDiffDiag$' -count=1 -v"
```

Result: wildcard residual cleared. New first diff:

```text
FIRST-DIFF @root[38]
  go: type="foreign_import" [1883:1952] cc=6
  c : kind="foreign_import" [1883:1952] cc=5
  go.child[4]: type="forall" [1913:1922] cc=3
  go.child[5]: type="type_infix" [1923:1952] cc=3
  c.child[4]: kind="type_infix" [1913:1952] cc=4
```

## N=40 Status

PureScript N=40 was skipped because both primary witnesses are not full parity
matches yet: HeytingAlgebra now matches, but Semigroup exposes the next
accepted-shape residual at `foreign_import` / `forall` materialization.
