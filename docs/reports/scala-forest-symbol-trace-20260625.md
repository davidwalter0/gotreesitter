# Scala Forest Symbol Trace: Missing Block Comment Wrapper

Date: 2026-06-25

## Scope

This is a report-only diagnostic artifact for the completed Scala forest symbol trace. It summarizes the evidence from a focused Docker parity run for `AutomaticModuleName.scala`; no parser code changed.

The run shape was:

```sh
cd /workspace/cgo_harness &&
REPRO_LANG=scala \
REPRO_DIR=/workspace/corpus_sources \
REPRO_FILE=/workspace/corpus_sources/scala/project/AutomaticModuleName.scala \
REPRO_PROGRESS=1 \
REPRO_SIGNATURES=1 \
REPRO_N=1 \
REPRO_ROUNDS=1 \
REPRO_FOREST=1 \
GOT_FOREST_SYMBOL_TRACE=1 \
GOT_FOREST_SYMBOL_TRACE_NAMES=block_comment,block_comment_repeat1,block_comment_token1 \
GOT_FOREST_SYMBOL_TRACE_MIN=50 \
GOT_FOREST_SYMBOL_TRACE_MAX=405 \
go test . -tags 'cgo treesitter_c_parity' -run '^TestMeasureDtierVsC$' -count=1 -v
```

## Artifact And Status

Artifact:

`/home/draco/work/gotreesitter-build-baseline/harness_out/docker/20260625T134001Z-scala-forest-symbol-trace-20260625`

Metadata:

- `label=scala-forest-symbol-trace-20260625`
- `exit_code=0`
- `oom_killed=false`
- `container.log` captured the forest symbol trace and final parity measurement.

## Trace Evidence

Counts from `container.log`:

- `FOREST-SYM-REDUCE`: 584
- `reduceName=block_comment_repeat1`: 584
- `reduceName=block_comment`: 0
- `action=dedup-keep`: 42
- `cap-drop`, `cap-replace`, `dedup-replace`: 0
- Matching `FOREST-SYM-ACCEPT-CHAIN` entries: 0

Representative evidence:

- `container.log:8` starts reducing `block_comment_token1` into `block_comment_repeat1` near the start of the comment body: `token=block_comment_token1/64..66`, `parentSpan=62..63`, `reduceName=block_comment_repeat1`.
- `container.log:802` shows coalescing preserving an existing repeat-chain entry: `FOREST-SYM-COALESCE action=dedup-keep`, `name=block_comment_repeat1`, `span=389..391`, `links=2`.
- `container.log:1650-1706` shows the tail under lookahead `*//392..395`. Reductions continue only as `block_comment_repeat1`, with parent spans reaching at most `382..391` and `390..391`; no reduced parent reaches `..395`.
- No line reduces `reduceName=block_comment`, so the enclosing named extra wrapper for bytes `60..395` is never built.
- No matching `FOREST-SYM-ACCEPT-CHAIN` line appears, so the missing root-level extra is not explained by final accepted-chain loss.

The measurement remains non-parity:

- `container.log:1711`: `DIVERGE-SIG` at `path="root"` with `diff="child-count"`; Go root `compilation_unit` span `0:658` has `goRootCC=4`, while C root span `0:658` has `cRootCC=5`.
- `container.log:1713`: `MEASURE-DTIER scala mode=forest ... parityMatch=0/1(0%) diverge=1 trunc=1 errTree=0`.

## Conclusion

This trace falsifies coalescing/drop/final accepted-chain loss as the explanation for the missing root `block_comment`. The forest builds and coalesces `block_comment_repeat1`, and the coalescing events are `dedup-keep` only; there is no cap drop, cap replace, or dedup replace for the traced symbols.

The failure is earlier: the forest never builds the enclosing `block_comment` parent for bytes `60..395`. The repeat chain reaches the closing delimiter as lookahead (`*//392..395`) but does not shift/admit that delimiter into a reduced `block_comment` wrapper.

The next generalized target is therefore closing-delimiter/wrapper reduction/admission after the repeat chain. This points away from grammar normalizers and root folding as primary causes.

## Proposed Invariant/Test

Add a synthetic forest grammar case where a repeated internal token sequence must shift a closing delimiter and reduce a named extra wrapper, then verify the wrapper is preserved as a root-level extra before the following real sibling.

The test should make the wrapper reduction observable independently of Scala grammar details: if the repeat chain builds but the closing delimiter and named extra wrapper do not reduce into the root forest, the test should fail at the wrapper/admission boundary rather than at root folding.
