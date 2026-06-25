# Coding Recovery-Shape Same-Position C-Recovery Rejection - 2026-06-25

Scope: report-only evidence for the C-recovery version/materialization frame
shared by post-retry coding residuals. No parser behavior, grammar policy,
normalizer, language-name branch, or corpus-specific shape rule was added.

## Decision

Reject the bounded fix "allow same-position summary entries in
`cRecoverStrategy1Election` to survive and redispatch the same lookahead."

The primary tree-sitter C runtime explicitly skips same-position strategy-1
summary entries:

```text
/home/draco/go/pkg/mod/github.com/tree-sitter/go-tree-sitter@v0.25.0/src/parser.c:1276-1278
if (entry.state == ERROR_STATE) continue;
if (entry.position.bytes == position.bytes) continue;
unsigned depth = entry.depth;
```

The same runtime pushes the C error discontinuity onto every
`do_all_potential_reductions` version, merges those versions, records the stack
summary, and then calls `ts_parser__recover` on the current lookahead:

```text
/home/draco/go/pkg/mod/github.com/tree-sitter/go-tree-sitter@v0.25.0/src/parser.c:1508-1527
ts_stack_push(... ERROR_STATE);
ts_stack_merge(...);
ts_stack_record_summary(...);
ts_parser__recover(...);
```

Therefore, removing the Go port's `entry.posBytes == pos` skip would be an
intentional divergence from the C runtime, not a parity-preserving generalized
parser-machinery fix.

## CUDA Witness

Command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label c-rec-same-position-reject-cuda-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/c-rec-same-position-reject-20260625 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_PARSE_PROGRESS=1 GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8480:8810 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/c-rec-same-position-reject-20260625/cuda-unified-memory-streams.log"
```

Result: Docker exit `0`, `oom_killed=false`, max RSS `1480312 kB`.

Artifact:

- `cgo_harness/harness_out/c-rec-same-position-reject-20260625/cuda-unified-memory-streams.log`
- `harness_out/docker/20260625T202230Z-c-rec-same-position-reject-cuda-20260625`

High-signal trace:

```text
C-REC-TRACE cHandleError stack=0 state=8235 stack_byte=8585 tok_sym=70 tok=8586:8587
C-REC-TRACE reduceWindow ... reduce_symbol_name="template_declaration" ... raw_span=8510:8585 top_state=85 ... target_state=85 ...
C-REC-TRACE cRecoverStrategy1Election pos=8585 tok_sym=70 tok=8586:8587 ... entry_state=85 entry_depth=5 entry_pos=8509 recover_depth=5 current_has_action=true better_version_reject=false
C-REC-TRACE cRecoverToState goal_state=85 depth=5 raw_span=8510:8585 child_count=4 trailing_extras=0 pushed_error=true top_state=85 ...
```

Current first diff remains the same CUDA recovery-shape split:

```text
go.child[23]: ERROR [8510:8585]
go.child[24]: compound_statement [8586:8797]
c.child[23]: template_declaration [8510:8797]
```

Interpretation: the useful reduction exists before recovery election, but the
faithful C strategy-1 loop cannot choose same-position summary entries. The
accepted Go residual is still the materialized `ERROR` boundary after recovery
to the earlier valid state.

## C++ Control

Command:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh \
  --repo-root /home/draco/work/gotreesitter-build-baseline \
  --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro \
  --label c-rec-same-position-reject-cpp-20260625 \
  --memory 8g --cpus 4 --no-build -- \
  "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/c-rec-same-position-reject-20260625 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cpp REPRO_FILE=/workspace/corpus_sources/cpp/include/fmt/args.h REPRO_DIR=/workspace/corpus_sources GOT_PARSE_PROGRESS=1 GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=400:7200 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v > harness_out/c-rec-same-position-reject-20260625/cpp-args-h.log 2>&1"
```

Result: Docker exit `0`, `oom_killed=false`, max RSS `1194684 kB`.

Artifact:

- `cgo_harness/harness_out/c-rec-same-position-reject-20260625/cpp-args-h.log`
- `harness_out/docker/20260625T202253Z-c-rec-same-position-reject-cpp-20260625`

Current C++ still accepts and reaches EOF without truncation, but the current
branch's first diff is narrower than the earlier broad `ERROR [433:7167]`
example:

```text
FIRST-DIFF @root[6][5][1]
go: ERROR [453:462]
c : identifier [453:462]
```

The older byte window still shows C-recovery materialization at the end of the
file-region residual, but no longer reproduces the exact older broad first-diff
frame:

```text
C-REC-TRACE cRecoverStrategy1Election pos=7167 tok_sym=11 tok=7169:7175 ... entry_state=50 entry_depth=2 entry_pos=7148 recover_depth=2 current_has_action=true better_version_reject=false
C-REC-TRACE cRecoverToState goal_state=50 depth=2 raw_span=7150:7167 child_count=1 trailing_extras=0 pushed_error=true ...
```

Interpretation: C++ remains an accepted recovery-shape residual, but it is no
longer the decisive broad-frame control for the same-position hypothesis on
this branch. CUDA is the cleaner rejection witness.

## Rejected Patch Shape

A generalized patch that merely admits `entry.posBytes == pos` in
`cRecoverStrategy1Election` would:

- contradict the primary C runtime's strategy-1 election;
- potentially create zero-progress redispatch loops unless extra conditions are
  added;
- special-case a symptom after `cDoAllPotentialReductions` has already exposed
  the useful syntax version;
- leave the already-recorded merged-election experiments unchanged in their
  conclusion that post-election fan-out does not move CUDA.

No source change was kept.

## Next Frame

The frame methodology still points next to C-recovery, not runtime frontier
survival and not broad recovery cost.

The next generalized target should be earlier than `recover_to_state`
materialization:

- how `cDoAllPotentialReductions` renumbers, merges, and prunes reduced
  versions before the error discontinuity is pushed;
- whether the Go linearized group summary preserves C's merged-stack traversal
  and dedupe order for those versions;
- whether additional C-runtime instrumentation is needed to compare the exact
  merged summary entries and selected recovery state on the CUDA witness.

The current evidence does not justify a grammar-specific normalizer, a
language-name branch, a corpus-specific result rule, or blanket suppression of
`cRecoverToState` `ERROR` materialization.

## Validation

Inspection:

```sh
canopy --version
rg -n "cDoAllPotentialReductions|cHandleError|cRecoverStrategy1Election|cRecoverToState|same-position|posBytes == pos" \
  parser_recover_c.go parser_recover_c_test.go docs/reports/206-residual-frame-domain-ledger-20260625.md \
  docs/reports/coding-recovery-shape-frame-triangulation-20260625.md \
  docs/reports/coding-recovery-shape-crecovery-reduce-window-20260625.md
sed -n '1176,1535p' /home/draco/go/pkg/mod/github.com/tree-sitter/go-tree-sitter@v0.25.0/src/parser.c
nl -ba /home/draco/go/pkg/mod/github.com/tree-sitter/go-tree-sitter@v0.25.0/src/parser.c | sed -n '1268,1308p'
nl -ba /home/draco/go/pkg/mod/github.com/tree-sitter/go-tree-sitter@v0.25.0/src/parser.c | sed -n '1490,1530p'
```

Correctness/parity:

```sh
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label c-rec-same-position-reject-cuda-20260625 --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/c-rec-same-position-reject-20260625 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cuda REPRO_FILE=/workspace/corpus_sources/cuda/cpp/0_Introduction/UnifiedMemoryStreams/UnifiedMemoryStreams.cu REPRO_DIR=/workspace/corpus_sources GOT_PARSE_PROGRESS=1 GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=8480:8810 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v 2>&1 | tee harness_out/c-rec-same-position-reject-20260625/cuda-unified-memory-streams.log"
bash cgo_harness/docker/run_parity_in_docker.sh --repo-root /home/draco/work/gotreesitter-build-baseline --mount /home/draco/work/gotreesitter-corpora/corpus_sources:/workspace/corpus_sources:ro --label c-rec-same-position-reject-cpp-20260625 --memory 8g --cpus 4 --no-build -- "set -o pipefail; cd /workspace/cgo_harness && mkdir -p harness_out/c-rec-same-position-reject-20260625 && /usr/bin/time -v timeout --kill-after=10s 90s env CGO_ENABLED=1 REPRO_LANG=cpp REPRO_FILE=/workspace/corpus_sources/cpp/include/fmt/args.h REPRO_DIR=/workspace/corpus_sources GOT_PARSE_PROGRESS=1 GOT_C_RECOVERY=all GOT_C_RECOVERY_TRACE_WINDOW=400:7200 go test . -tags 'cgo treesitter_c_parity' -run '^TestFirstDiffDiag$' -count=1 -v > harness_out/c-rec-same-position-reject-20260625/cpp-args-h.log 2>&1"
```

Results:

- CUDA: exit `0`, `oom_killed=false`, max RSS `1480312 kB`.
- C++: exit `0`, `oom_killed=false`, max RSS `1194684 kB`.

Pre-commit checks:

```sh
git diff --check
```
