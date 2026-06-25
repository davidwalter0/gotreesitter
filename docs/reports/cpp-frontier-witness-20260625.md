# cpp frontier witness diagnostic: args.h frame 1

Date: 2026-06-25
Worktree: `/home/draco/work/gotreesitter-build-baseline`
Base commit: `dc2d86945e52712ef1dbe523f3561c532ab8f263`

## Scope

Bounded machinery-only diagnostic for cpp frame 1 from the current 206 smoke:

- Grammar: `cpp`
- Corpus file: `/workspace/corpus_sources/cpp/include/fmt/args.h`
- Size: 7192 bytes
- SHA256: `ce9705a22f68469891cba708b3c4ae459edb3f41c8a83f891e93013d485190f8`
- Baseline frames: `1`
- Variant frames: `1`
- Firstdiff frames: `1`
- Variants: `stack2 stack8 node3 merge1 merge24`

Successful Docker run:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label cpp-frontier-witness-20260625-rerun \
  --memory 8g --cpus 4 -- \
  'cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=1 GTS_WRINGER_BASELINE_FRAMES=1 GTS_WRINGER_VARIANT_FRAMES=1 GTS_WRINGER_FIRSTDIFF_FRAMES=1 GTS_WRINGER_VARIANTS="stack2 stack8 node3 merge1 merge24" GTS_WRINGER_PARSE_PROGRESS=1 GTS_WRINGER_TIMEOUT=60 GTS_WRINGER_DIAG_TIMEOUT=60 bash cgo_harness/docker/run_grammar_integrity_wringer.sh cpp cgo_harness/harness_out/cpp-frontier-witness-20260625'
```

The initially suggested nested `bash -lc ...` wrapper shape failed before diagnostics: the baseline build was terminated with `rc=143`, leaving an empty grammar manifest. A direct command string through the wrapper avoided that infra issue. No parser code was edited.

## Artifacts

- Wringer output: `cgo_harness/harness_out/cpp-frontier-witness-20260625`
- Summary JSON: `cgo_harness/harness_out/cpp-frontier-witness-20260625/wringer_summary.json`
- Frame matrix: `cgo_harness/harness_out/cpp-frontier-witness-20260625/frame_matrix.jsonl`
- Baseline log: `cgo_harness/harness_out/cpp-frontier-witness-20260625/baseline/measure-cpp-external-frame-0001.log`
- Firstdiff log: `cgo_harness/harness_out/cpp-frontier-witness-20260625/firstdiff/frame-0001.log`
- Docker metadata: `harness_out/docker/20260625T084211Z-cpp-frontier-witness-20260625-rerun/metadata.txt`

## Result

The run completed with `exit_code=0`, `oom_killed=false`, control status `closed`, and all 7 planned actions completed. The summary family is `truncation_frontier_loss`.

| mode | parity | trunc | root pair | Go span | C span | stop reason | tokens | last token end | expected EOF | nodes | max stacks |
| --- | --- | --- | --- | --- | --- | --- | ---: | ---: | ---: | --- | ---: |
| baseline | 0/1 (0%) | 1 | ERROR -> translation_unit | 0:641 | 0:7192 | no_stacks_alive | 71 | 652 | 7192 | 132/373984 | 3 |
| stack2 | 0/1 (0%) | 1 | ERROR -> translation_unit | 0:641 | 0:7192 | no_stacks_alive | 71 | 652 | 7192 | 188/373984 | 8 |
| stack8 | 0/1 (0%) | 1 | ERROR -> translation_unit | 0:641 | 0:7192 | no_stacks_alive | 71 | 652 | 7192 | 132/373984 | 3 |
| node3 | 0/1 (0%) | 1 | ERROR -> translation_unit | 0:641 | 0:7192 | no_stacks_alive | 71 | 652 | 7192 | 132/1121952 | 3 |
| merge1 | 0/1 (0%) | 1 | ERROR -> translation_unit | 0:641 | 0:7192 | no_stacks_alive | 71 | 652 | 7192 | 132/373984 | 3 |
| merge24 | 0/1 (0%) | 1 | ERROR -> translation_unit | 0:641 | 0:7192 | no_stacks_alive | 71 | 652 | 7192 | 188/373984 | 8 |

Stack, merge, and node variants do not move token progress, stop reason, root pair, or parity. `stack2` and `merge24` change internal node count / observed max-stack telemetry, but they still die at the same token frontier and produce the same truncated root mismatch. `node3` only expands the node ceiling; consumed nodes stay at 132 and progress does not move.

## Firstdiff evidence

Firstdiff reports the divergence at root:

- Go: `ERROR` root `[0:641]`, child count 26, `rootHasError=true`
- C: `translation_unit` root `[0:7192]`, child count 8, `cRootHasError=true`
- Go runtime: `truncated=true stopReason=no_stacks_alive tokenEOFEarly=false tokens=71 lastTokenEnd=652 expectedEOF=7192`

The visible firstdiff shape is preprocessor recovery related. C keeps a large `preproc_ifdef` child spanning `[209:7175]` under the translation unit. Go instead emits loose top-level preprocessor tokens and declarations, then stops around:

```text
template <typename T> struct is_reference_wrapper<std::reference_wrapper<T>> : std::true_type {}
```

The immediate Go root children include an `ERROR` at byte `[622:623]` for the inheritance colon, followed by `std::true_type`, `{`, and `}`, after which the accepted frontier is gone.

## Interpretation

This is not behaving like a node-budget limit: `node3` raises the budget from `373984` to `1121952`, but nodes consumed remain `132` and token progress remains `71`.

This is also not explained by the simple stack/merge limit knobs tested here: `stack2`, `stack8`, `merge1`, and `merge24` leave token progress, stop reason, root pair, and parity unchanged.

The strongest current read is recovery-cost/admission selection around the generalized C++ preprocessor frontier, especially the survival of the enclosing `preproc_ifdef` recovery path across the template specialization inheritance colon. A cull/merge frontier bug is not ruled out in the implementation sense, but the cap/merge variants did not move the witness, so the next useful diagnostic should focus on why the C-like recovery/preprocessor path is not admitted or selected before `no_stacks_alive`.
