# CUDA next-frame parser diagnostic, 2026-06-25

## Scope

Bounded true-coding diagnostic for CUDA only, after `HEAD`:

```text
10649548dbb2a678b44044cb6b86f7854717b683 add(docs): add N=40 C-recovery controls recertification report
branch: repair/java-token-admission-reduced
initial worktree status: clean
```

Primary reused artifact:

```text
cgo_harness/harness_out/post-same-pop-crecovery-controls-n40-20260625/cuda
```

Corpus mount:

```text
/home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro
```

All runs used Docker isolation through `cgo_harness/docker/run_parity_in_docker.sh` with `--repo-root /home/draco/work/gotreesitter-build-baseline --memory 8g --cpus 4`, one CUDA grammar/container at a time. No host repo-wide `go test` was run.

## Commands

The first run allowed the Docker image build. Later runs used `--no-build`.

```bash
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --memory 8g --cpus 4 --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label cuda-frame1-firstdiff -- "cd /workspace && env GTS_WRINGER_REUSE_BASELINE=1 GTS_WRINGER_STAGES=firstdiff,summary GTS_WRINGER_FIRSTDIFF_FRAMES=1 GTS_WRINGER_MAX_DIAG_FILES=1 GTS_WRINGER_PARSE_PROGRESS=1 GTS_CORPUS_DIR=/workspace/corpus_sources bash cgo_harness/docker/run_grammar_integrity_wringer.sh cuda /workspace/cgo_harness/harness_out/post-same-pop-crecovery-controls-n40-20260625/cuda"

bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --memory 8g --cpus 4 --no-build --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label cuda-frame1-variants -- "cd /workspace && env GTS_WRINGER_REUSE_BASELINE=1 GTS_WRINGER_STAGES=variants,summary GTS_WRINGER_VARIANT_FRAMES=1 GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_VARIANTS='stack2 stack8 node3 forest' GTS_CORPUS_DIR=/workspace/corpus_sources bash cgo_harness/docker/run_grammar_integrity_wringer.sh cuda /workspace/cgo_harness/harness_out/post-same-pop-crecovery-controls-n40-20260625/cuda"

bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --memory 8g --cpus 4 --no-build --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label cuda-frame39-firstdiff -- "cd /workspace && env GTS_WRINGER_REUSE_BASELINE=1 GTS_WRINGER_STAGES=firstdiff,summary GTS_WRINGER_FIRSTDIFF_FRAMES=39 GTS_WRINGER_MAX_DIAG_FILES=1 GTS_WRINGER_PARSE_PROGRESS=1 GTS_CORPUS_DIR=/workspace/corpus_sources bash cgo_harness/docker/run_grammar_integrity_wringer.sh cuda /workspace/cgo_harness/harness_out/post-same-pop-crecovery-controls-n40-20260625/cuda"

bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --memory 8g --cpus 4 --no-build --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label cuda-frame39-variants -- "cd /workspace && env GTS_WRINGER_REUSE_BASELINE=1 GTS_WRINGER_STAGES=variants,summary GTS_WRINGER_VARIANT_FRAMES=39 GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_VARIANTS='stack2 stack8 node3 forest' GTS_CORPUS_DIR=/workspace/corpus_sources bash cgo_harness/docker/run_grammar_integrity_wringer.sh cuda /workspace/cgo_harness/harness_out/post-same-pop-crecovery-controls-n40-20260625/cuda"
```

Two frame 1 reruns used the same firstdiff and variants commands with labels `cuda-frame1-firstdiff-rerun` and `cuda-frame1-variants-rerun` to restore overwritten frame 1 logs for inspection after frame 39.

## Docker wrapper results

The first two frame 1 commands completed with wrapper exit `0` and `oom_killed=false`:

```text
harness_out/docker/20260625T234752Z-cuda-frame1-firstdiff
harness_out/docker/20260625T234806Z-cuda-frame1-variants
```

Frame 39 replay commands generated the requested firstdiff/variant artifacts, but the wrapper exited `1` during the final `summary` stage with `oom_killed=false`:

```text
harness_out/docker/20260625T234837Z-cuda-frame39-firstdiff
harness_out/docker/20260625T234924Z-cuda-frame39-variants
```

The failure was a shared-artifact sanity failure, not a replay crash:

```text
unexpected lifecycle action stage=firstdiff mode=firstdiff ordinal=1 path=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu
artifact sanity failure: unexpected lifecycle action stage=firstdiff mode=firstdiff ordinal=1 path=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu
```

After frame 39, frame 1 was rerun to restore frame 1 logs for inspection. Those reruns also generated replay artifacts and exited `1` only at the mixed summary sanity step, now because stale frame 39 lifecycle events were present:

```text
harness_out/docker/20260625T235014Z-cuda-frame1-firstdiff-rerun
harness_out/docker/20260625T235030Z-cuda-frame1-variants-rerun
```

Interpretation: running selected frames sequentially into the same reused wringer output directory leaves prior lifecycle events in `wringer_events.jsonl`; the replay logs are usable, but the final shared `wringer_summary.json` is not authoritative after mixed selected-frame reruns.

## Frame 1: `UnifiedMemoryStreams.cu`

Path:

```text
/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu
```

Firstdiff replay artifact:

```text
cgo_harness/harness_out/post-same-pop-crecovery-controls-n40-20260625/cuda/firstdiff/frame-0001.log
```

Outcome:

```text
go stopReason=accepted
truncated=false
tokens=1993
lastTokenEnd=12254 expectedEOF=12254
rootHasError=true cRootHasError=false
FIRST-DIFF @root
go translation_unit [0:12254] child-count=26
c  translation_unit [0:12254] child-count=25
```

The root child split is the template function body around `initialise_tasks`: Go materializes a `template_declaration` followed by a top-level `compound_statement`, while C materializes one `template_declaration` spanning the function body. The baseline family label remains `recovery_error_shape`: Go accepts and is non-truncated, but carries `44` errors and `3` missing nodes against C `0/0`.

Variant outcomes:

```text
stack2: diverge, root type ERROR vs C translation_unit, goErrors=1 goMissing=2, accepted/non-truncated
stack8: diverge, root child-count 26 vs 25, goErrors=44 goMissing=3, accepted/non-truncated
node3:  diverge, root child-count 26 vs 25, goErrors=44 goMissing=3, accepted/non-truncated
forest: go_no_tree, nil_tree forestDeclineReason=no-shift-death
```

Evidence:

- Raising stack cap from 2 to 8 does not repair the root shape; `stack2` makes the selected root worse, while `stack8` matches the baseline bad shape.
- Raising node budget to `node3` does not repair the root shape.
- Forest mode does not expose a clean alternate materialization for this frame; it declines before producing a tree.
- The defect is therefore not primarily node budget or simple stack cap pressure. It points at generalized recovery summary/election/materialization around a recoverable C++/CUDA template function definition.

## Frame 39: `simpleTexture3D_kernel.cu`

Path:

```text
/workspace/corpus_sources/cuda/cpp/0_Introduction/simpleTexture3D/simpleTexture3D_kernel.cu
```

Firstdiff replay artifact from the frame 39 run:

```text
cgo_harness/harness_out/post-same-pop-crecovery-controls-n40-20260625/cuda/firstdiff/frame-0039.log
```

Outcome:

```text
go stopReason=accepted
truncated=false
rootHasError=false cRootHasError=false
FIRST-DIFF @root[1][16][2][2][16][0][1][5]
go sizeof_expression [4132:4155] child-count=4
c  sizeof_expression [4132:4155] child-count=2
go children: sizeof, (, type_descriptor "cudaTextureDesc", )
c children:  sizeof, parenthesized_expression "(cudaTextureDesc)"
```

Variant outcomes:

```text
stack2: diverge, firstDiffPath root[1][15][2][2][3][0][1][5], sizeof_expression child-count 4 vs 2
stack8: diverge, firstDiffPath root[1][16][2][2][16][0][1][5], sizeof_expression child-count 4 vs 2
node3:  diverge, firstDiffPath root[1][16][2][2][16][0][1][5], sizeof_expression child-count 4 vs 2
forest: diverge, firstDiffPath root[1][15][2][2][3][0][1][5], sizeof_expression child-count 4 vs 2
```

Evidence:

- This frame is accepted, non-truncated, and no-error on both Go and C.
- The divergence survives stack cap, node budget, and forest mode.
- The local shape is an ambiguity/materialization choice: Go chooses a `type_descriptor` form under `sizeof_expression`, while C exposes a `parenthesized_expression`.
- This points at dynamic-precedence or child-selection/materialization for ambiguous no-error parses, not recovery.

## Recommendation

The next generalized parser machinery target should be recovery summary/election/materialization, with frame 1 as the primary falsification frame.

Reasoning:

- Frame 1 is a larger correctness failure: accepted/non-truncated parse with root-level error materialization and a top-level child split against a clean C tree.
- The selected variants rule out simple stack cap and node budget pressure.
- Forest mode does not surface a viable clean tree for frame 1, so the next useful work is in how recoveries are summarized, ranked, elected, and materialized for C++/CUDA template bodies.

Frame 39 should remain a follow-up target for dynamic-precedence child-selection or accepted materialization invariant work. It is a clean no-error ambiguity case, but it is not the same failure class as frame 1.

Falsification criteria for the recommended target:

```text
Primary: frame 1 no longer has rootHasError=true, no longer reports Go 44/3 vs C 0/0 errors/missing, and its firstdiff moves away from root or disappears without grammar-specific normalization.
Secondary: the repair does not regress frame 39 from accepted/no-error to recovery/error, and does not rely on per-grammar normalizers or CUDA language-name policy.
Wrong-target signal: if recovery summary/election/materialization changes leave frame 1 at the same root child-count/error shape, then the evidence shifts toward a lower-level accepted materialization invariant or candidate preservation issue rather than recovery ranking.
```

This diagnostic adds no per-grammar normalizers, makes no language-name parser policy change, and makes no performance claim.
