# Coding Recovery-Shape C-Recovery Merged Election Experiments - 2026-06-25

Scope: bounded generic behavior experiments for the CUDA recovery-shape
residual. No grammar-specific normalizer, grammar-name policy, byte-span
policy, symbol-name policy, or result-shape special case was kept.

## Baseline Residual

Primary witness:

```text
/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu
```

Current first diff before the experiments:

- Go child `23`: `ERROR [8510:8585]`.
- Go child `24`: `compound_statement [8586:8797]`.
- C child `23`: `template_declaration [8510:8797]`.

Prior trace evidence remained true in both experiments: before
`recover_to_state`, `cDoAllPotentialReductions` can build a
`template_declaration` ending at the same failure byte:

```text
C-REC-TRACE reduceWindow phase=gss tok_sym=70 tok_name="{" tok=8586:8587 reduce_symbol=399 reduce_symbol_name="template_declaration" child_count=3 ... raw_span=8510:8585 top_state=85 ... target_state=85 reduced_error_extras=0 trailing_error_extras=0
```

## Experiment 1: Cross-Member Election Fan-Out

Change tested locally: after `cRecoverStrategy1Election` elected the first
valid `(depth,state)` summary key, attempt `cRecoverToState` across every live
member in the same `cRecGroup` that carried the same key, bounded by
`cRecoverMaxVersionCount`.

CUDA command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label c-rec-merged-election-cuda-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/c-rec-merged-election-20260625 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_PARSE_PROGRESS=1 GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8480:8810 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/c-rec-merged-election-20260625/cuda-unified-memory-streams.log"
```

Result: Docker exit `0`, `oom_killed=false`, max RSS `1122400 kB`.

CUDA first diff stayed unchanged. The elected recovery key at the failure byte
still reported only `member=0`, followed by the same `recover_to_state` span:

```text
C-REC-TRACE cRecoverStrategy1Election pos=8585 tok_sym=70 tok=8586:8587 member=0 entry_state=85 entry_depth=5 entry_pos=8509 recover_depth=5 current_has_action=true better_version_reject=false
C-REC-TRACE cRecoverToState goal_state=85 depth=5 raw_span=8510:8585 child_count=4 trailing_extras=0 pushed_error=true top_state=85 top_symbol=65535 top_symbol_name="?" top_pre_goto=85 top_parse_state=85
```

## Experiment 2: Packed GSS Pop-Slice Recovery

Additional root evidence showed that upstream C `ts_stack_pop_count` uses
`stack__iter` over merged links, and `ts_parser__recover_to_state` keeps every
popped slice whose resulting version state equals the goal. A second local
experiment therefore made `recover_to_state` enumerate bounded pop slices
through the selected member's packed GSS `extraLinks`, linearize each selected
prefix, and push the same generic ERROR/trailing extras for each viable slice.

CUDA command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label c-rec-gss-pop-slices-cuda-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/c-rec-gss-pop-slices-20260625 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_PARSE_PROGRESS=1 GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8480:8810 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/c-rec-gss-pop-slices-20260625/cuda-unified-memory-streams.log"
```

Result: Docker exit `0`, `oom_killed=false`, max RSS `1129092 kB`.

CUDA first diff again stayed unchanged:

- Go child `23`: `ERROR [8510:8585]`.
- Go child `24`: `compound_statement [8586:8797]`.
- C child `23`: `template_declaration [8510:8797]`.

The high-signal trace still showed the useful reduction immediately before
recovery:

```text
C-REC-TRACE reduceWindow ... reduce_symbol_name="template_declaration" ... raw_span=8510:8585 ... target_state=85 ...
C-REC-TRACE reduceWindow ... reduce_symbol_name="translation_unit_repeat1" ... raw_span=4686:8585 ... target_state=85 ...
```

But the accepted recovery path still materialized only the header span as
extra `ERROR`:

```text
C-REC-TRACE cRecoverStrategy1Election pos=8585 tok_sym=70 tok=8586:8587 member=0 entry_state=85 entry_depth=5 entry_pos=8509 recover_depth=5 current_has_action=true better_version_reject=false
C-REC-TRACE cRecoverToState goal_state=85 depth=5 raw_span=8510:8585 child_count=4 trailing_extras=0 pushed_error=true top_state=85 top_symbol=65535 top_symbol_name="?" top_pre_goto=85 top_parse_state=85
```

## Conclusion

Both generic approximations were rejected. They are plausible ports of C merged
stack behavior, but they do not move the CUDA witness out of the current
recovery-shape residual, and keeping them would be speculative.

The remaining root-cause boundary is earlier than post-election
`recover_to_state` fan-out. The useful reduced syntax version is visible during
`cDoAllPotentialReductions`, but it is not available to the elected recovery as
a distinct viable pop slice that produces the C-shaped
`template_declaration [8510:8797]`. The next generic target should inspect how
the Go port renumbers, merges, prunes, or records summaries for the
`do_all_potential_reductions` versions before entering the C error state.

Because CUDA did not improve, the C++ secondary witness was not broadened.

## Validation

Focused host tests run before each Docker witness:

```sh
go test . -run '^(TestParseCRecoveryTraceWindow|TestCDoAllPotentialReductionsRejectsUndrainedFaithfulForks|TestRequirementsTrailingCommentRecoveryMatchesC|TestRequirementsCleanParseUnchanged)$' -count=1
```

Result after the packed-slice experiment:

```text
ok  	github.com/odvcencio/gotreesitter	0.009s
```
