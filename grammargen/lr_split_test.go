package grammargen

import (
	"context"
	"errors"
	"testing"
)

func TestSplitKernelLookaheadsForTransitionUsesRetainedFollowSet(t *testing.T) {
	follow := newBitset(8)
	follow.add(3)
	ctx := &lrContext{
		tokenCount: 2,
		lalrFollowByTransition: map[[2]int]bitset{
			{7, 4}: follow,
		},
	}

	got := ctx.splitKernelLookaheadsForTransition(7, 4, newBitset(8))
	if !got.contains(3) || got.count() != 1 {
		t.Fatalf("splitKernelLookaheadsForTransition() = %v, want retained follow lookahead 3", got.words)
	}

	got.add(5)
	stored := ctx.lalrFollowByTransition[[2]int{7, 4}]
	if stored.contains(5) {
		t.Fatal("splitKernelLookaheadsForTransition() should clone retained follow sets")
	}
}

func TestSplitKernelLookaheadsForTransitionKeepsInheritedLookaheads(t *testing.T) {
	inherited := newBitset(8)
	inherited.add(1)
	follow := newBitset(8)
	follow.add(3)
	ctx := &lrContext{
		tokenCount: 2,
		lalrFollowByTransition: map[[2]int]bitset{
			{7, 4}: follow,
		},
	}

	got := ctx.splitKernelLookaheadsForTransition(7, 4, inherited)
	if !got.contains(1) || got.count() != 1 {
		t.Fatalf("splitKernelLookaheadsForTransition() = %v, want inherited lookahead 1", got.words)
	}
}

func TestLocalLR1Rebuild(t *testing.T) {
	// Create a grammar known to have LALR merge pathology.
	// Two rules that share a common prefix but diverge:
	//   A → a b c d
	//   B → a b c e
	// In LALR, the states for "a b c ." merge. But the reduce on
	// lookahead {d} vs {e} creates a conflict if both are viable.
	g := NewGrammar("split_test")
	g.Define("start", Choice(Sym("a_rule"), Sym("b_rule")))
	g.Define("a_rule", Seq(Str("a"), Str("b"), Str("c"), Str("d")))
	g.Define("b_rule", Seq(Str("a"), Str("b"), Str("c"), Str("e")))

	ng, err := Normalize(g)
	if err != nil {
		t.Fatal(err)
	}

	tables, ctx, err := buildLRTablesWithProvenance(ng)
	if err != nil {
		t.Fatal(err)
	}
	prov := ctx.provenance

	// Run conflict resolution with diagnostics.
	diags, _, err := resolveConflictsWithDiag(context.Background(), tables, ng, prov)
	if err != nil {
		t.Fatal(err)
	}

	oracle := newSplitOracle(diags, prov)
	candidates := oracle.candidates()

	t.Logf("states=%d, conflicts=%d, candidates=%d",
		tables.StateCount, len(diags), len(candidates))

	if len(candidates) == 0 {
		t.Skip("no split candidates — grammar may be too simple for LALR pathology")
	}

	// Apply local rebuild.
	splitCount, err := localLR1Rebuild(tables, ng, ctx, candidates, 100)
	if err != nil {
		t.Fatalf("localLR1Rebuild failed: %v", err)
	}

	t.Logf("split %d states", splitCount)

	// After splitting, re-resolve conflicts — should have fewer GLR entries.
	diagsAfter, _, err := resolveConflictsWithDiag(context.Background(), tables, ng, prov)
	if err != nil {
		t.Fatal(err)
	}

	glrBefore := 0
	for _, d := range diags {
		if d.Resolution == "GLR (multiple actions kept)" {
			glrBefore++
		}
	}
	glrAfter := 0
	for _, d := range diagsAfter {
		if d.Resolution == "GLR (multiple actions kept)" {
			glrAfter++
		}
	}

	t.Logf("GLR conflicts: before=%d, after=%d", glrBefore, glrAfter)
	if glrAfter > glrBefore {
		t.Errorf("splitting should not increase GLR conflicts")
	}
}

func TestGenerateWithReportSplitting(t *testing.T) {
	g := NewGrammar("split_gen_test")
	g.Define("start", Choice(Sym("a_rule"), Sym("b_rule")))
	g.Define("a_rule", Seq(Str("a"), Str("b"), Str("c"), Str("d")))
	g.Define("b_rule", Seq(Str("a"), Str("b"), Str("c"), Str("e")))

	// Enable splitting.
	g.EnableLRSplitting = true

	report, err := GenerateWithReport(g)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("states=%d, conflicts=%d, candidates=%d, splitResult=%v",
		report.StateCount, len(report.Conflicts),
		len(report.SplitCandidates), report.SplitResult)
}

// TestCountGLRConflictsSurfacesResolutionError is the P3 #8 regression test:
// a resolveActionConflict failure must abort countGLRConflicts with an error
// instead of being masked as "not a GLR conflict" (the previous `err == nil
// && len(resolved) > 1` pattern). A masked error would silently undercount
// GLR conflicts and could make localLR1Rebuild's accept/rollback decision
// treat an unverifiable split as an improvement.
func TestCountGLRConflictsSurfacesResolutionError(t *testing.T) {
	multiAction := []lrAction{{kind: lrShift, state: 1}, {kind: lrReduce, prodIdx: 0}}
	actionTable := map[int][]lrAction{5: multiAction}

	wantErr := errors.New("boom")
	failingResolve := func(lookaheadSym int, actions []lrAction, ng *NormalizedGrammar) ([]lrAction, error) {
		return nil, wantErr
	}

	glr, err := countGLRConflicts(actionTable, nil, failingResolve)
	if err == nil {
		t.Fatal("countGLRConflicts() error = nil, want a surfaced resolution error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("countGLRConflicts() error = %v, want it to wrap %v", err, wantErr)
	}
	if glr != 0 {
		t.Fatalf("countGLRConflicts() count = %d on error, want 0 (must not silently count a candidate whose resolution failed)", glr)
	}
}

// TestCountGLRConflictsHappyPath proves the P3 #8 fix does not change
// behavior for well-formed input: entries that resolve cleanly are still
// counted exactly as before.
func TestCountGLRConflictsHappyPath(t *testing.T) {
	tests := []struct {
		name        string
		actionTable map[int][]lrAction
		resolve     resolveConflictFunc
		wantGLR     int
	}{
		{
			name:        "no conflicts",
			actionTable: map[int][]lrAction{5: {{kind: lrShift, state: 1}}},
			resolve: func(int, []lrAction, *NormalizedGrammar) ([]lrAction, error) {
				t.Fatal("resolve should not be called for a single-action entry")
				return nil, nil
			},
			wantGLR: 0,
		},
		{
			name:        "conflict resolves to a single action",
			actionTable: map[int][]lrAction{5: {{kind: lrShift, state: 1}, {kind: lrReduce, prodIdx: 0}}},
			resolve: func(sym int, actions []lrAction, ng *NormalizedGrammar) ([]lrAction, error) {
				return actions[:1], nil
			},
			wantGLR: 0,
		},
		{
			name:        "conflict stays GLR",
			actionTable: map[int][]lrAction{5: {{kind: lrShift, state: 1}, {kind: lrReduce, prodIdx: 0}}},
			resolve: func(sym int, actions []lrAction, ng *NormalizedGrammar) ([]lrAction, error) {
				return actions, nil
			},
			wantGLR: 1,
		},
		{
			name: "mixed table: one resolved, one GLR",
			actionTable: map[int][]lrAction{
				5: {{kind: lrShift, state: 1}, {kind: lrReduce, prodIdx: 0}},
				6: {{kind: lrShift, state: 2}, {kind: lrReduce, prodIdx: 1}},
			},
			resolve: func(sym int, actions []lrAction, ng *NormalizedGrammar) ([]lrAction, error) {
				if sym == 5 {
					return actions[:1], nil
				}
				return actions, nil
			},
			wantGLR: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			glr, err := countGLRConflicts(tt.actionTable, nil, tt.resolve)
			if err != nil {
				t.Fatalf("countGLRConflicts() error = %v, want nil", err)
			}
			if glr != tt.wantGLR {
				t.Fatalf("countGLRConflicts() = %d, want %d", glr, tt.wantGLR)
			}
		})
	}
}

// TestCountGLRConflictsAcrossStatesStopsOnFirstError proves
// countGLRConflictsAcrossStates propagates a per-state resolution failure
// instead of summing past it (which would silently understate the total).
func TestCountGLRConflictsAcrossStatesStopsOnFirstError(t *testing.T) {
	tables := &LRTables{
		ActionTable: map[int]map[int][]lrAction{
			1: {5: {{kind: lrShift, state: 1}, {kind: lrReduce, prodIdx: 0}}},
			2: {6: {{kind: lrShift, state: 2}, {kind: lrReduce, prodIdx: 1}}},
		},
	}
	wantErr := errors.New("boom")
	resolve := func(sym int, actions []lrAction, ng *NormalizedGrammar) ([]lrAction, error) {
		if sym == 6 {
			return nil, wantErr
		}
		return actions, nil
	}

	total, err := countGLRConflictsAcrossStates([]int{1, 2}, tables, nil, resolve)
	if err == nil {
		t.Fatal("countGLRConflictsAcrossStates() error = nil, want a surfaced resolution error")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("countGLRConflictsAcrossStates() error = %v, want it to wrap %v", err, wantErr)
	}
	if total != 0 {
		t.Fatalf("countGLRConflictsAcrossStates() total = %d on error, want 0", total)
	}
}
