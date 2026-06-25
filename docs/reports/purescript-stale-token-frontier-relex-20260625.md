# PureScript Stale-Token Frontier Relex Diagnostic - 2026-06-25

## Scope

Bounded generalized parser-machinery diagnostic/fix slice for the 206 true-coding frontier witness:

- Grammar: `purescript`
- File: `/corpus_sources/purescript/src/Data/Ord.purs`
- Frame: `31`
- Baseline lane: `glr_frontier_survival_and_reuse_selection`

No per-grammar normalizer, language-name policy, or performance work was used.

## Baseline Reproduction

The single-file Docker tier scan reproduced the witness:

```text
truncated=true stopReason=no_stacks_alive tokens=392 lastTokenEnd=1971 expectedEOF=6654
iterations=981/199620 nodes=8600/346008 peakDepth=68/13308 maxStacks=18
Go root span=0:1964 C root span=0:6654 goErrors=0 cErrors=0 diff=span
```

The root had no error node, but the Go parse stopped at byte 1964 while C consumed the full file.

## Probe Matrix

All probes stayed on the same PureScript file/frame.

| Probe | Result |
| --- | --- |
| Default | `no_stacks_alive`, `tokens=392`, `lastTokenEnd=1971`, Go root span `0:1964` |
| `GOT_GLR_MAX_STACKS=64` | `no_stacks_alive`, unchanged token/span progress, `nodes=10398`, `maxStacks=24` |
| `GOT_PARSE_NODE_LIMIT_SCALE=3` | `no_stacks_alive`, unchanged token/span progress, node cap raised to `1038024` |
| Both stack cap and node scale | `no_stacks_alive`, unchanged token/span progress |

This distinguishes the witness from a static node budget failure. Raising stack or node budgets retained more work in some cases, but did not advance the frontier.

## Frontier Invariant

GLR trace showed a stale lookahead token after layout handling:

```text
stack_byte=1964
tok=_varid(1)[1965-1971]
gap="f"
```

The source byte at `1964` is the `f` in `foreign`. The token source had produced a current-state lookahead that began one byte too late (`oreign`) while every live stack still needed to attach real source text at byte `1964`. The real-shift gap guard then killed every remaining stack, producing terminal `no_stacks_alive`.

The generalized invariant is:

> Before killing all live stacks for a real non-extra token gap, a shared parser-state frontier may relex from the stack byte with the current parser state's DFA lex mode, but only if the replacement token starts exactly at that stack byte and has a parse-table action in that state.

## Fix

`parser.go` now adds a guarded current-state DFA relex from the stack byte before the stale real-gap kill path:

- It only runs when `tok.StartByte > stack.byteOffset`.
- It rejects parser padding and grammar-extra-covered gaps.
- It requires all live stacks to share the same parser state.
- It requires the relexed token to start exactly at `stack.byteOffset`.
- It requires the replacement token to have a parse action in the current state.

This is a parser machinery fix; it does not inspect the grammar name.

## Post-Fix Evidence

PureScript `Data/Ord.purs` advanced past the frontier stop:

```text
truncated=false stopReason=accepted tokens=1272 lastTokenEnd=6654 expectedEOF=6654 lastTokenEOF=true
iterations=3492/199620 nodes=26913/346008 maxStacks=18
Go root span=0:6654 C root span=0:6654 diff=child-count
goRootErr=true goErrors=76 cErrors=0
```

Cross-language frontier witness `uxntal/projects/examples/blank.tal` also advanced past `no_stacks_alive`:

```text
truncated=false stopReason=accepted tokens=860 lastTokenEnd=2722 expectedEOF=2722 lastTokenEOF=true
iterations=5411/81660 nodes=41821/300000 maxStacks=18
Go root span=0:2722 C root span=0:2722 diff=named
goRootErr=true goErrors=105 cErrors=23
```

## Next Generalized Target

The stale-token frontier death is cleared for the PureScript witness and one cross-language frontier witness, but both now expose full-span accepted error-tree divergence. The next generalized machinery target is recovery/error-cost election after successful current-state relex: inspect why the Go recovery path materializes extra error nodes once the frontier survives, especially around recover dispatch, missing-token insertion, and accepted-stack competition.
