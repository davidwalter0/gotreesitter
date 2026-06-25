# Coding Recovery-Shape Reduce Action Dedupe Correction - 2026-06-25

Scope: generalized C-recovery parser-machinery correction. No grammar-specific
normalizer, language-name branch, corpus-specific rule, or result-shape special
case was added.

## Correction

This report previously claimed that upstream tree-sitter C dedupes potential
reductions by full `ReduceAction` identity. That was wrong.

The faithful upstream behavior is in:

```text
/home/draco/go/pkg/mod/github.com/tree-sitter/go-tree-sitter@v0.25.0/src/reduce_action.h:20-28
```

`ts_reduce_action_set_add` compares only `symbol` and `count` before returning.
Although `ReduceAction` also carries dynamic precedence and production id, those
fields are not part of set membership.

## Correct Fix

`cReduceActionKey` must contain only:

- reduced symbol;
- child count.

`cCollectPotentialReductions` should preserve the first reduce action for a
given `(symbol, count)` pair and dedupe later actions with the same pair, even
when production id or dynamic precedence differ.

## Unit Proof

`TestCCollectPotentialReductionsDedupeMatchesReduceActionSet` builds a
synthetic grammar-neutral table containing four reduce actions with identical
symbol/count:

- one baseline action;
- one exact duplicate;
- one differing only by `ProductionID`;
- one differing only by `DynamicPrecedence`.

The test proves the exact C behavior: all later same-symbol/count actions are
collapsed, and the first action survives.

## CUDA Witness

The earlier witness run for the now-reverted full-identity interpretation was:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label c-reduce-action-key-cuda-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "cd /workspace/cgo_harness && env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8480:8810 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

That run should not be used as evidence for full reduce-action identity. The
correct reduce-action key is `(symbol, count)`.

## Validation

Inspection:

```sh
sed -n '1,70p' /home/draco/go/pkg/mod/github.com/tree-sitter/go-tree-sitter@v0.25.0/src/reduce_action.h
rg -n "cReduceActionKey|cCollectPotentialReductions|TestCCollectPotentialReductions" parser_recover_c.go parser_recover_c_test.go
```

Focused host unit test:

```sh
go test . -run '^(TestCCollectPotentialReductionsDedupeMatchesReduceActionSet|TestCRecoverGroupMemberIndexSurvivesStackReorder|TestCBuildMergedGroupSummaryMatchesStackIterDelayedBranchOrder)$' -count=1
```
