package gotreesitter

import "testing"

func TestCCollectPotentialReductionsDedupeMatchesReduceActionSet(t *testing.T) {
	lang := &Language{
		TokenCount:  3,
		StateCount:  1,
		SymbolCount: 5,
		ParseTable: [][]uint16{
			{0, 1, 1},
		},
		ParseActions: []ParseActionEntry{
			{},
			{Actions: []ParseAction{
				{Type: ParseActionReduce, Symbol: 4, ChildCount: 2, ProductionID: 7, DynamicPrecedence: 0},
				{Type: ParseActionReduce, Symbol: 4, ChildCount: 2, ProductionID: 7, DynamicPrecedence: 0},
				{Type: ParseActionReduce, Symbol: 4, ChildCount: 2, ProductionID: 8, DynamicPrecedence: 0},
				{Type: ParseActionReduce, Symbol: 4, ChildCount: 2, ProductionID: 7, DynamicPrecedence: 3},
			}},
		},
	}
	parser := &Parser{language: lang, denseLimit: len(lang.ParseTable)}

	var reductions []ParseAction
	hasShift := parser.cCollectPotentialReductions(0, 1, &reductions)
	if hasShift {
		t.Fatal("hasShift = true, want false")
	}
	// Upstream reduce_action.h only compares symbol and count in
	// ts_reduce_action_set_add, so production id and dynamic precedence do not
	// keep otherwise-equivalent reductions distinct.
	if len(reductions) != 1 {
		t.Fatalf("reduction count = %d, want 1", len(reductions))
	}
	got := reductions[0]
	if got.Symbol != 4 || got.ChildCount != 2 || got.ProductionID != 7 || got.DynamicPrecedence != 0 {
		t.Fatalf("surviving reduction = %+v, want first symbol/count action", got)
	}
}

func TestCDoAllPotentialReductionsRejectsUndrainedFaithfulForks(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	lang := &Language{
		TokenCount:  3,
		StateCount:  4,
		SymbolCount: 5,
		ParseTable: [][]uint16{
			nil,
			nil,
			nil,
			{0, 1, 1},
		},
		ParseActions: []ParseActionEntry{
			{},
			{Actions: []ParseAction{{Type: ParseActionReduce, Symbol: 4, ChildCount: 2}}},
		},
		SymbolMetadata: []SymbolMetadata{
			{Name: "eof", Visible: true, Named: true},
			{Name: "a", Visible: true, Named: true},
			{Name: "b", Visible: true, Named: true},
			{Name: "unused", Visible: true, Named: true},
			{Name: "parent", Visible: true, Named: true},
		},
	}
	parser := &Parser{language: lang, denseLimit: len(lang.ParseTable)}

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	left := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	right := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	leftNode := scratch.allocNode(newStackEntryNode(2, left), base, 2)
	rightNode := scratch.allocNode(newStackEntryNode(3, right), leftNode, 3)
	altRight := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	rightNode.extraLinks = append(rightNode.extraLinks, gssMainLink{
		prev:  leftNode,
		entry: newStackEntryNode(3, altRight),
	})
	start := glrStack{gss: gssStack{head: rightNode}, byteOffset: 2}

	nodeCount := 0
	versions, canShift := parser.cDoAllPotentialReductions(start, 0, Token{}, &nodeCount, arena, nil, &scratch, nil)
	if canShift {
		t.Fatal("canShift = true, want false")
	}
	if len(parser.pendingForkStacks) != 0 {
		t.Fatalf("pending forks = %d, want 0", len(parser.pendingForkStacks))
	}
	if len(versions) != 1 {
		t.Fatalf("version count = %d, want only original version", len(versions))
	}
	if versions[0].gss.head != rightNode {
		t.Fatal("C recovery retained a forked reduction instead of the original version")
	}
}

func TestCBuildMergedGroupSummaryMatchesStackIterDelayedBranchOrder(t *testing.T) {
	parser := &Parser{}

	leaf := func(sym Symbol) *Node {
		return NewLeafNode(sym, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	}
	path := func(member int, firstState, branchState StateID, sym Symbol) []cSummaryPathEntry {
		return []cSummaryPathEntry{
			{entry: stackEntry{state: cErrorState}, posBytes: 2, memberID: member},
			{entry: newStackEntryNode(firstState, leaf(sym)), posBytes: 2, memberID: member},
			{entry: stackEntry{state: branchState}, posBytes: 0, memberID: member},
		}
	}

	// Paths 0 and 1 merge below the top ERROR link because that link is NULL
	// and their child stack nodes share state/position/error-cost. Path 2
	// stays a separate top link. C stack__iter therefore reaches depth 2 as:
	// path 0, path 2, then the delayed path 1 clone.
	got := parser.cBuildMergedGroupSummaryForPaths([][]cSummaryPathEntry{
		path(0, 10, 20, 1),
		path(1, 10, 21, 2),
		path(2, 40, 50, 3),
	})
	wantStates := []StateID{cErrorState, 10, 40, 20, 50, 21}
	wantMembers := []int{0, 0, 2, 0, 2, 1}
	if len(got) != len(wantStates) {
		t.Fatalf("summary length = %d, want %d: %+v", len(got), len(wantStates), got)
	}
	for i := range got {
		if got[i].state != wantStates[i] || got[i].memberID != wantMembers[i] {
			t.Fatalf("summary[%d] = state %d member %d, want state %d member %d; full=%+v",
				i, got[i].state, got[i].memberID, wantStates[i], wantMembers[i], got)
		}
	}
}

func TestCRecoverGroupMemberIndexSurvivesStackReorder(t *testing.T) {
	group := &cRecGroup{}
	stacks := []glrStack{
		{cRec: &cRecoverState{group: group, memberID: 42}},
		{cRec: &cRecoverState{group: group, memberID: 7}},
		{cRec: &cRecoverState{group: &cRecGroup{}, memberID: 42}},
	}
	summary := cGroupSummaryEntry{memberID: stacks[0].cRec.memberID}

	stacks[0], stacks[1] = stacks[1], stacks[0]
	if stacks[0].cRec.memberID == summary.memberID {
		t.Fatal("test setup did not move the summary owner away from its original index")
	}
	got := cRecoverGroupMemberIndex(stacks, group, summary.memberID)
	if got != 1 {
		t.Fatalf("resolved member index = %d, want 1 after reorder", got)
	}
	clone := stacks[got].cRec.clone()
	if clone == nil || clone.memberID != summary.memberID || clone.group != group {
		t.Fatalf("clone = %+v, want same group and member id %d", clone, summary.memberID)
	}
	stacks[got].dead = true
	if got := cRecoverGroupMemberIndex(stacks, group, summary.memberID); got != -1 {
		t.Fatalf("resolved dead member index = %d, want -1", got)
	}
}

func TestParseCRecoveryTraceWindow(t *testing.T) {
	disabled := parseCRecoveryTraceWindow("")
	if disabled.enabled {
		t.Fatal("empty trace window enabled diagnostics")
	}

	single := parseCRecoveryTraceWindow("5594")
	if !single.enabled || single.start != 5594 || single.end != 5594 {
		t.Fatalf("single trace window = %+v, want enabled 5594..5594", single)
	}

	span := parseCRecoveryTraceWindow("5600:5588")
	if !span.enabled || span.start != 5588 || span.end != 5600 {
		t.Fatalf("reversed trace window = %+v, want enabled 5588..5600", span)
	}

	invalid := parseCRecoveryTraceWindow("not-a-window")
	if invalid.enabled {
		t.Fatalf("invalid trace window enabled diagnostics: %+v", invalid)
	}
}
