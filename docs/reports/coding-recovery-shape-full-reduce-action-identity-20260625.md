# Coding Recovery-Shape Full Reduce Action Identity - 2026-06-25

Scope: generalized C-recovery parser-machinery fix. No grammar-specific
normalizer, language-name branch, corpus-specific rule, or result-shape special
case was added.

## Finding

Upstream tree-sitter C dedupes potential reductions in
`ts_parser__do_all_potential_reductions` by the full reduce action identity:

```text
/home/draco/go/pkg/mod/github.com/tree-sitter/go-tree-sitter@v0.25.0/src/parser.c:1144-1151
ts_reduce_action_set_add(&self->reduce_actions, (ReduceAction) {
  .symbol = action.reduce.symbol,
  .count = action.reduce.child_count,
  .dynamic_precedence = action.reduce.dynamic_precedence,
  .production_id = action.reduce.production_id,
});
```

The Go C-recovery port only keyed `cCollectPotentialReductions` by reduced
symbol and child count. That could collapse two C-distinct reduce actions when
they share the same symbol/count but differ in production id or dynamic
precedence.

## Fix

`cReduceActionKey` now includes:

- reduced symbol;
- child count;
- dynamic precedence;
- production id.

`cCollectPotentialReductions` uses that full key before appending a reduce
action. The change is grammar-neutral and only aligns the C-recovery potential
reduction set with upstream C's `ReduceAction` identity.

## Unit Proof

Added `TestCCollectPotentialReductionsKeepsFullReduceIdentity`, which builds a
synthetic grammar-neutral table containing four reduce actions with identical
symbol/count:

- one baseline action;
- one exact duplicate;
- one differing only by `ProductionID`;
- one differing only by `DynamicPrecedence`.

The test proves the exact duplicate is collapsed while the production-id and
dynamic-precedence variants both survive collection.

## CUDA Witness

Command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label c-reduce-action-key-cuda-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "cd /workspace/cgo_harness && env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8480:8810 GOT_PARSE_PROGRESS=1 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v"
```

Result: Docker exit `0`, `oom_killed=false`.

Artifact:

- `harness_out/docker/20260625T203155Z-c-reduce-action-key-cuda-20260625/container.log`

The CUDA first diff is unchanged from the prior recovery-shape frame:

```text
FIRST-DIFF @root
go: translation_unit [0:12254] cc=26
c : translation_unit [0:12254] cc=25
go.child[23]: ERROR [8510:8585]
go.child[24]: compound_statement [8586:8797]
c.child[23]: template_declaration [8510:8797]
```

Runtime status remains accepted and non-truncated:

```text
go stopReason=accepted runtime=truncated=false tokens=1993 lastTokenEnd=12254 expectedEOF=12254
nodes=30367/637208 maxStacks=36 rootHasError=true cRootHasError=false
```

Interpretation: this patch fixes a real C-fidelity bug in potential-reduction
collection, but the CUDA witness is still controlled by the later recovery
shape/materialization path around `initialise_tasks`.

## Validation

Inspection:

```sh
canopy --version
rg -n "cReduceActionKey|cCollectPotentialReductions|DynamicPrecedence|ProductionID" parser_recover_c.go parser_recover_c_test.go language.go
sed -n '1120,1175p' /home/draco/go/pkg/mod/github.com/tree-sitter/go-tree-sitter@v0.25.0/src/parser.c
```

Focused host unit test:

```sh
go test . -run '^(TestCCollectPotentialReductionsKeepsFullReduceIdentity|TestCDoAllPotentialReductionsRejectsUndrainedFaithfulForks|TestParseCRecoveryTraceWindow)$' -count=1
```

Result:

```text
ok  	github.com/odvcencio/gotreesitter	0.005s
```

Pre-commit check:

```sh
git diff --check
```

## Next Frame

The next generalized target remains earlier than recovery materialization:
inspect how `cDoAllPotentialReductions` renumbers, merges, prunes, and records
summary fidelity before the C error discontinuity is pushed. The surviving CUDA
diff still indicates that the useful `template_declaration [8510:8797]` shape is
not preserved through the selected recovery path.
