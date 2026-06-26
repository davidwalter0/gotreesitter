# CUDA frame 1 GSS tie-preserve falsification, 2026-06-26

## Scope

Bounded follow-up to
`docs/reports/cuda-frame1-merge-result-selection-invariant-20260626.md`.

Question tested: can an env-gated generalized parser-machinery experiment
preserve same-key GSS main-link comparator ties at the default merge cap, before
hash overflow replacement/drop, and clear CUDA frame 1's trailing
`ERROR [8796:8797]` without reproducing the plain cap-one regression?

No CUDA, C++, grammar-name, language-name, grammar blob, or normalizer policy
was added. The experimental parser code and focused unit tests were reverted
after the run because the invariant failed the acceptance condition.

## Temporary experiment

Temporary env:

```text
GOT_GLR_GSS_TIE_PRESERVE=1
```

Temporary implementation shape:

- Add a cached env flag in `parser_config.go`.
- At same-key merge overflow in `glr.go`, before hash replacement/drop, try to
  GSS-main-merge the candidate into an existing same-key survivor when
  accepted/error-rank/score/shifted tie and `gssMainMerge` can represent the
  alternative.
- Add focused synthetic tests proving non-cap-one same-key overflow alternatives
  could be packed into retained GSS main-links.

Focused host unit slice while the experiment was present:

```bash
go test . -run '^(TestMergeStacksGSSTiePreservePacksSameKeyOverflowAtDefaultCap|TestParseGLRGSSTiePreserveEnabled|TestMergeStacksGeneralFaithfulGSSUnionPreservesMoreThanFourStacks|TestMergeStacksDeferExactFaithfulGSSUnionPreservesMoreThanFourStacks)$' -count=1
```

Result:

```text
ok  	github.com/odvcencio/gotreesitter	0.004s
```

## Docker validation

All Docker runs used:

```text
--repo-root /home/draco/work/gotreesitter-build-baseline
--mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro
--memory 8g --cpus 4 --no-build
```

Common parser env:

```text
GOT_C_RECOVERY=all
GOT_GLR_TOKEN_FRONTIER_DISPATCH=1
GOT_GLR_MAX_STACKS=64
```

### Frame 1 control

Command label:

```text
cuda-frame1-gss-tie-control-20260626
```

Artifact:

```text
harness_out/docker/20260626T020811Z-cuda-frame1-gss-tie-control-20260626/container.log
```

Result reproduced the known target failure:

```text
go stopReason=accepted truncated=false rootHasError=true cRootHasError=false
go.child[23]: template_declaration [8510:8795]
go.child[24]: ERROR [8796:8797]
c.child[23]:  template_declaration [8510:8797]
```

### Frame 1 experiment

Added:

```text
GOT_GLR_GSS_TIE_PRESERVE=1
```

Command label:

```text
cuda-frame1-gss-tie-experiment-20260626
```

Artifact:

```text
harness_out/docker/20260626T020841Z-cuda-frame1-gss-tie-experiment-20260626/container.log
```

Result did not clear the target:

```text
go stopReason=accepted truncated=false rootHasError=true cRootHasError=false
go.child[23]: template_declaration [8510:8795]
go.child[24]: ERROR [8796:8797]
c.child[23]:  template_declaration [8510:8797]
```

The run changed pressure but not the failing shape:

```text
control:    tokens=1996 iterations=3981 nodes=30645 maxStacks=42
experiment: tokens=1995 iterations=3872 nodes=28496 maxStacks=36
```

### Frame 39 experiment

Added:

```text
GOT_GLR_GSS_TIE_PRESERVE=1
```

Command label:

```text
cuda-frame39-gss-tie-experiment-20260626
```

Artifact:

```text
harness_out/docker/20260626T020900Z-cuda-frame39-gss-tie-experiment-20260626/container.log
```

Result:

```text
go stopReason=accepted truncated=false rootHasError=true cRootHasError=false
FIRST-DIFF @root[1][14][3][7][2][3]
go declaration [2399:2425]
c  expression_statement [2399:2425]
```

This is not an acceptable safety signal for the proposed invariant.

## Conclusion

The tested invariant is falsified for this bounded slice. GSS main-link
tie-preservation at same-key overflow, as implemented in the temporary
experiment, is not sufficient to preserve the full-span frame 1
`template_declaration [8510:8797]` route at the default cap. It also leaves
frame 39 in a root-error state under the experiment.

CUDA frame 1 stays in Tier IV. The probe narrowed the failure away from a
simple default-cap overflow tie that `gssMainMerge` can represent directly.

Next action: trace whether the full-span route is lost before the tested
overflow point, or whether the relevant candidate is not a rank-tie under the
accepted/error-rank/score/shifted same-key predicate. A targeted merge trace
should count attempted/merged GSS tie-preserve opportunities near bytes
`8794:8802` without keeping parser behavior changes.
