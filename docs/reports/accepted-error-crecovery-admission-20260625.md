# Accepted-Error C-Recovery Admission - 2026-06-25

## Scope

Structural retry-admission change for accepted, full-EOF, non-truncated full
parse trees that still carry error evidence under stack pressure. No grammar
normalizers or language-name policy were added.

## Code Change

- `shouldRetryFullParseWithCRecovery` now receives the initial stack budget.
- Existing `no_stacks_alive` and `node_limit` admissions are preserved.
- Accepted trees are admitted only when source size is within the bounded retry
  limit, stop reason is `accepted`, the token source did not EOF early, the EOF
  token was reached, the tree spans expected EOF, structural error evidence is
  present, and `MaxStacksSeen >= initialMaxStacks`.
- Structural error evidence checks the raw root through result child accessors
  so deferred child-ref trees are handled before language-specific result
  normalization.

## Validation

Focused Docker unit run:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh --label accepted-error-crecovery-unit-3 -- \
  "cd /workspace && go test . -run 'TestCRecoveryRetry|TestRetryFullParseCRecoveryCandidateRestoresParserFlag|TestShouldRunInitialFullParseMergeRetry' -count=1 -v"
```

Result: pass, wrapper exit `0`, `oom_killed=false`.

Two-frame proof run:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label accepted-error-crecovery-cuda-frames-final2 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- \
  "cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_N=40 GTS_WRINGER_BASELINE_FRAMES=1,3 GTS_WRINGER_FIRSTDIFF_FRAMES=1,3 GTS_WRINGER_STAGES=baseline,firstdiff,summary GTS_WRINGER_PARSE_PROGRESS=1 bash cgo_harness/docker/run_grammar_integrity_wringer.sh cuda cgo_harness/harness_out/accepted-error-crecovery-cuda-frames-20260625-final2"
```

Result: pass, wrapper exit `0`, `oom_killed=false`.

Variant replay for candidate/control evidence:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label accepted-error-crecovery-cuda-variants \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro -- \
  "cd /workspace && GTS_CORPUS_DIR=/workspace/corpus_sources GTS_WRINGER_REUSE_BASELINE=1 GTS_WRINGER_STAGES=variants,summary GTS_WRINGER_VARIANT_FRAMES=1,3 GTS_WRINGER_VARIANTS='crecovery_off crecovery_all' GTS_WRINGER_PARSE_PROGRESS=1 bash cgo_harness/docker/run_grammar_integrity_wringer.sh cuda cgo_harness/harness_out/accepted-error-crecovery-cuda-frames-20260625-final"
```

Result: pass, wrapper exit `0`, `oom_killed=false`.

## Frame Results

Default final proof artifacts:

- `cgo_harness/harness_out/accepted-error-crecovery-cuda-frames-20260625-final2/frame_matrix.jsonl`
- `cgo_harness/harness_out/accepted-error-crecovery-cuda-frames-20260625-final2/firstdiff/frame-0001.log`
- `cgo_harness/harness_out/accepted-error-crecovery-cuda-frames-20260625-final2/firstdiff/frame-0003.log`

Frame 1 remains non-parity and classified as `recovery_error_shape`:

- Runtime: `accepted`, `truncated=false`, `tokenEOFEarly=false`,
  `lastTokenEOF=true`, `lastTokenEnd=12254`, `expectedEOF=12254`,
  `maxStacks=24`.
- Default result: `goErrors=17`, `goMissing=11`, `cErrors=0`,
  `cMissing=0`, first diff at `root`.
- First diff: Go still splits the declaration area into an `ERROR` child plus a
  following `compound_statement`; C keeps one `template_declaration`.

Frame 3 remains the no-error control shape and classified as
`version_or_corpus`:

- Runtime: `accepted`, `truncated=false`, `tokenEOFEarly=false`,
  `lastTokenEOF=true`, `maxStacks=18`.
- Default result: `goErrors=0`, `goMissing=0`, root error false.
- First diff remains the existing `sizeof_expression` child-count shape.

Variant evidence from
`cgo_harness/harness_out/accepted-error-crecovery-cuda-frames-20260625-final`:

| Frame | Mode | Errors | Missing | Nodes | Classification shape |
| --- | --- | ---: | ---: | ---: | --- |
| 1 | `crecovery_off` | 17 | 11 | 31390 | root child-count with error tree |
| 1 | `crecovery_all` | 1 | 2 | 27438 | root child-count with lower error cost |
| 3 | `crecovery_off` | 0 | 0 | 8213 | no-error child-count control |
| 3 | `crecovery_all` | 0 | 0 | 8212 | no-error child-count control |

Conclusion: the forced C-recovery candidate still lowers frame 1 error/missing
cost but does not reach parity. The default proof did not move frame 1 to the
candidate shape; residual work remains in retry selection or materialization.
Frame 3 did not convert into a recovery target.
