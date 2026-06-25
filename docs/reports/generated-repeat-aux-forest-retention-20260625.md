# Generated Repeat Aux Forest Retention

Date: 2026-06-25

## Invariant

When a forest node is capped at one `(state, byte)` and a candidate reduction is an invisible, unnamed generated repeat auxiliary, equal-score candidates are not interchangeable. A wider/earlier-span repeat auxiliary preserves more stack history and can be the only retained representation that lets an enclosing wrapper reach its closing delimiter.

The forest retention rule now keeps score as the primary ordering, but for equal-score same-symbol generated repeat auxiliaries it admits the candidate past the pre-cap drop and replaces a narrower retained repeat link with the wider candidate.

## Scala Table Facts

Temporary local dump against `grammars.ScalaLanguage()` showed:

- `block_comment_token1` is token `102`.
- `*/` is token `103`.
- `block_comment` is nonterminal `306`.
- `block_comment_repeat1` is nonterminal `352`.
- `state=18038` shifts `*/` to `21251` and gotos `block_comment_repeat1 -> 17521`.
- `state=17521` shifts `*/` to `21248` and gotos `block_comment_repeat1 -> 17537`.
- `state=17537` is reduce-only on `*/`: `reduce(block_comment_repeat1, child_count=2)`.
- `state=18979` is reduce-only on `*/`: `reduce(block_comment_repeat1, child_count=1)`.
- `state=21248` reduces `block_comment` with `child_count=3` on EOF.
- `state=21251` reduces `block_comment` with `child_count=2` on EOF.

This supports the retained-history hypothesis: a short repeat representation coalesced at `17537` can lose the predecessor path that would return to `17521` or `18038`, where the close delimiter is shiftable.

The decoded Scala blob did not have `GeneratedRepeatAux` populated for `block_comment_repeat1`; embedded grammar loading now runs `InferGeneratedRepeatAuxMetadata` so older blobs expose this generalized metadata bit.

## Verification

Focused host unit tests:

```sh
go test . -run '^(TestCoalesceForest|TestForestCoalescePreCapKeepsGeneratedRepeatAuxCandidate|TestReduceOverForest|TestForestResolveConflictPrefersBlockCommentRepetitionShift|TestGSSForest)' -count=1
```

Result: pass.

Single Scala Docker forest replay:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-forest-repeat-cap \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/AutomaticModuleName.scala REPRO_FOREST=1 REPRO_N=1 REPRO_ROUNDS=1 REPRO_PROGRESS=1 REPRO_SIGNATURES=1 go test . -tags treesitter_c_parity -run '^TestMeasureDtierVsC$' -count=1 -v"
```

Artifact: `harness_out/docker/20260625T140625Z-scala-forest-repeat-cap/container.log`

Result: `parityMatch=1/1(100%) diverge=0 errTree=0 panics=0`; `comparison_result result=match` for `AutomaticModuleName.scala`.

Remaining nuance: `TestMeasureDtierVsC` still counts `trunc=1` because `ParseForestExperimental` returns a tree without an accepted parse runtime, so `stopReason=none` is classified as not accepted. The structural comparison against C now matches.
