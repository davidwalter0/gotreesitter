# Scala Forest N=40 Post Repeat-Aux Classification

Date: 2026-06-25

Scope: bounded Docker forest classification for Scala after the generated
repeat auxiliary forest retention and accepted-runtime fixes. This is
classification evidence only. It does not update `tier_classification.tsv`,
does not measure production/default mode, and does not move Scala out of
Tier IV by itself.

## Command

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-forest-n40-post-repeat-aux-20260625 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_EXTS=.scala REPRO_FOREST=1 REPRO_N=40 REPRO_ROUNDS=1 REPRO_PROGRESS=1 REPRO_SIGNATURES=1 go test . -tags treesitter_c_parity -run '^TestMeasureDtierVsC$' -count=1 -v"
```

Artifact:
`harness_out/docker/20260625T141622Z-scala-forest-n40-post-repeat-aux-20260625`

An earlier same-label directory,
`harness_out/docker/20260625T141552Z-scala-forest-n40-post-repeat-aux-20260625`,
selected `files=0` because the directory walk was missing `REPRO_EXTS=.scala`;
it is not used as parity evidence.

## Result

```text
MEASURE-DTIER scala mode=forest files=40 medianRatio=1.19x aggRatio=0.89x parityMatch=29/40(72%) diverge=8 trunc=0 errTree=0 panics=0 goNS=57673376 cNS=64564169
```

Container outcome:

- `exit_code=0`
- `oom_killed=false`
- no timeout or terminal harness failure
- no truncation, error trees, or panics in the MEASURE-DTIER aggregate

The selected N=40 corpus starts at
`/workspace/corpus_sources/scala/project/AutomaticModuleName.scala` and then
continues through sorted `.scala` files under the mounted Scala corpus.

## Residual Families

The single frame witness `AutomaticModuleName.scala` is clean in this forest
sweep, but the broader N=40 sample is not clean.

Observed residuals:

| Family | Count | Representative files |
| --- | ---: | --- |
| `go_no_tree` during comparison reparse | 3 | `GenerateFunctionConverters.scala`, `PartestTestListener.scala`, `genprod.scala` |
| Accepted/no-error span divergence | 6 | `ScalaOptionParser.scala`, `VersionUtil.scala`, `PartestUtilTest.scala`, `Validators.scala`, `Parsers.scala`, `Reifiers.scala` |
| Accepted/no-error root child-count divergence | 2 | `ScalaTool.scala`, `TravisOutput.scala` |

The span divergences cluster around accepted full-root materialization of
Scala constructs such as interpolated string expressions, infix expressions,
and case clauses. The root child-count divergences also have accepted,
non-error roots on both sides.

Because `trunc=0`, `errTree=0`, and `panics=0`, this forest sample no longer
looks like the production iteration-limit frontier witness. It instead exposes
generalized forest comparison/materialization residuals that still require
parser machinery work.

## 206 Queue Impact

For the 206 true-coding residual queue, Scala should be treated as:

- frame witness cleared for `AutomaticModuleName.scala` under forest mode;
- forest N=40 accepted/non-truncated but not clean;
- not default-promoted, because this run is `REPRO_FOREST=1` and production
  mode still likely truncates;
- still a generalized machinery target, now refined from the original
  production runtime-frontier frame toward forest materialization/comparison
  residuals under the experimental forest path.

No per-grammar normalizer is justified by this evidence.
