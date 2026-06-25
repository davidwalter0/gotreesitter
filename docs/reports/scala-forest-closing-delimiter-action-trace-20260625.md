# Scala Forest Closing Delimiter Action Trace

Date: 2026-06-25

## Scope

This is a report-only diagnostic for Scala frame 1 forest materialization. It
answers whether the forest path shifts/admits the closing block-comment
delimiter token `*/` around bytes `392..395` and then exposes a
`block_comment` wrapper reduce, or whether that wrapper action is absent.

No parser behavior change is included in this report. Temporary
`GOT_FOREST_ACTION_TRACE=1` instrumentation was added only for the replay and
removed afterward.

## Command

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --label scala-forest-closing-delimiter-action-trace-20260625 \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  -- "cd /workspace/cgo_harness && timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=scala REPRO_DIR=/workspace/corpus_sources REPRO_FILE=/workspace/corpus_sources/scala/project/AutomaticModuleName.scala REPRO_PROGRESS=1 REPRO_SIGNATURES=1 REPRO_N=1 REPRO_ROUNDS=1 REPRO_FOREST=1 GOT_FOREST_ACTION_TRACE=1 GOT_FOREST_ACTION_TRACE_MIN=380 GOT_FOREST_ACTION_TRACE_MAX=405 GOT_FOREST_ACTION_TRACE_NAMES=block_comment,block_comment_repeat1,block_comment_token1 go test . -tags 'cgo treesitter_c_parity' -run '^TestMeasureDtierVsC$' -count=1 -v"
```

## Artifact And Status

Artifact:

`/home/draco/work/gotreesitter-build-baseline/harness_out/docker/20260625T135025Z-scala-forest-closing-delimiter-action-trace-20260625`

Metadata:

- `exit_code=0`
- `oom_killed=false`
- `label=scala-forest-closing-delimiter-action-trace-20260625`
- `extra_mounts=/home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro`

The replay still diverges at the root child count:

```text
MEASURE-DTIER scala mode=forest files=1 ... parityMatch=0/1(0%) diverge=1 trunc=1 errTree=0
DIVERGE-SIG ... path="root" diff="child-count" ... goCC=4 cCC=5
```

## Evidence

The trace includes two Go forest parses from the measurement/comparison path,
so delimiter-local events appear twice. The same pattern appears in both.

Counts from `container.log`:

- `FOREST-SHIFT ... start=392 end=395`: 0
- `FOREST-ACTION label=raw ... start=392 end=395`: 18
- `FOREST-ACTION label=resolved ... start=392 end=395`: 18
- `FOREST-REDUCE ... reduceName=block_comment`: 0
- `FOREST-ACTION-ITEM ... reduceName=block_comment`: 0
- `FOREST-REDUCE ... reduceName=block_comment_repeat1`: 584
- `FOREST-REDUCE-DROP`: 16, all for capped duplicate `block_comment_repeat1`
  reductions at `parentEnd=391`
- `rawActions=2 resolvedActions=1`: 524, from earlier repeat-token conflict
  pruning, not from the delimiter token

Representative delimiter window from `container.log:3432-3507`:

```text
FOREST-ACTION label=raw state=18979 nodeByte=391 token=*//103 start=392 end=395 ... rawActions=1 resolvedActions=1
FOREST-ACTION-ITEM label=raw index=0 type=reduce ... reduceName=block_comment_repeat1 childCount=1
FOREST-ACTION label=resolved state=18979 nodeByte=391 token=*//103 start=392 end=395 ... rawActions=1 resolvedActions=1
FOREST-ACTION-ITEM label=resolved index=0 type=reduce ... reduceName=block_comment_repeat1 childCount=1
FOREST-REDUCE state=18979 nodeByte=391 token=*//103 start=392 end=395 reduceName=block_comment_repeat1 ... parentSpan=390..391 parentEnd=391 goto=17537
FOREST-ACTION label=raw state=17537 nodeByte=391 token=*//103 start=392 end=395 ... rawActions=1 resolvedActions=1
FOREST-ACTION-ITEM label=raw index=0 type=reduce ... reduceName=block_comment_repeat1 childCount=2
FOREST-ACTION label=resolved state=17537 nodeByte=391 token=*//103 start=392 end=395 ... rawActions=1 resolvedActions=1
FOREST-ACTION-ITEM label=resolved index=0 type=reduce ... reduceName=block_comment_repeat1 childCount=2
```

Immediately before the delimiter, the body token at `390..391` does shift:

```text
FOREST-SHIFT state=17537 nodeByte=390 token=block_comment_token1/102 start=390 end=391 target=18979 ...
FOREST-ACTION label=post-shift-same-token state=18979 nodeByte=391 ... reduceName=block_comment_repeat1
```

After the delimiter token is processed, the next traced frontier is already at
`nodeByte=395` on the following body token:

```text
FOREST-ACTION label=raw state=18979 nodeByte=395 token=block_comment_token1/102 start=396 end=397 ... rawActions=1 resolvedActions=1
FOREST-ACTION-ITEM label=raw index=0 type=reduce ... reduceName=block_comment_repeat1 childCount=1
```

## Answer

At the closing delimiter token `*/` around bytes `392..395`, the forest path
does not shift/admit the delimiter. The raw action set is already reduce-only:
`block_comment_repeat1` from states `18979` and `17537`, with no shift action
and no `block_comment` wrapper reduce. Conflict resolution does not prune a
delimiter shift there; raw and resolved action counts are both `1`.

The `block_comment` wrapper action is never present in the traced forest run.
The only reductions observed for the block-comment family are
`block_comment_repeat1`, and they continue to materialize spans ending at
`391`, before the delimiter. The later cap drops are duplicate repeat reductions
after the 8-link fanout cap is reached, not dropped wrapper reductions.

This supports a generalized wrapper-reduction/admission investigation: the
problem is not Scala-specific normalization, not a Scala parser policy, and not
a `block_comment` reduce being pruned after a successful delimiter shift. The
shared parser machinery never exposes the closing-delimiter shift / wrapper
reduce sequence to the forest at this boundary.
