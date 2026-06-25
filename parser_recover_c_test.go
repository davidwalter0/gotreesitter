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

func newCRecoverySyntheticReduceParser() *Parser {
	lang := &Language{
		TokenCount:  3,
		StateCount:  4,
		SymbolCount: 7,
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
			{Name: "alternate_parent", Visible: true, Named: true},
			{Name: "extra", Visible: false, Named: false},
		},
	}
	return &Parser{language: lang, denseLimit: len(lang.ParseTable)}
}

func TestCDoAllPotentialReductionsCollapsesSamePopTargetSlices(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	parser := newCRecoverySyntheticReduceParser()

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	left := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	right := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	leftNode := scratch.allocNode(newStackEntryNode(2, left), base, 2)
	rightNode := scratch.allocNode(newStackEntryNode(3, right), leftNode, 3)
	altRight := newLeafNodeInArena(arena, 1, true, 1, 2, Point{Column: 1}, Point{Column: 2})
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
		t.Fatalf("version count = %d, want same-pop-target slices collapsed to one C version", len(versions))
	}
	if versions[0].gss.head != rightNode {
		top := stackEntryNode(versions[0].top())
		if top == nil || top.symbol != 4 {
			t.Fatalf("current version top = %+v, want reduced parent symbol 4", top)
		}
		if got := versions[0].gss.head.linkCount(); got != 1 {
			t.Fatalf("reduced top link count = %d, want same-pop-target children collapsed to one alternative", got)
		}
		if len(top.children) != 2 || top.children[1] != altRight {
			t.Fatal("same-pop collapse did not keep the C-selected child array")
		}
	} else {
		t.Fatal("current version still points at the unreduced merged head")
	}
}

func TestCDoAllPotentialReductionsCollapsesSamePopWithTrailingExtra(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	parser := newCRecoverySyntheticReduceParser()

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	left := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	right := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	altRight := newLeafNodeInArena(arena, 1, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	extra := newLeafNodeInArena(arena, 6, false, 2, 3, Point{Column: 2}, Point{Column: 3})
	extra.setExtra(true)
	altExtra := newLeafNodeInArena(arena, 6, false, 2, 3, Point{Column: 2}, Point{Column: 3})
	altExtra.setExtra(true)

	leftNode := scratch.allocNode(newStackEntryNode(2, left), base, 2)
	rightNode := scratch.allocNode(newStackEntryNode(3, right), leftNode, 3)
	altRightNode := scratch.allocNode(newStackEntryNode(3, altRight), leftNode, 3)
	head := scratch.allocNode(newStackEntryNode(3, extra), rightNode, 4)
	head.extraLinks = append(head.extraLinks, gssMainLink{
		prev:  altRightNode,
		entry: newStackEntryNode(3, altExtra),
	})
	start := glrStack{gss: gssStack{head: head}, byteOffset: 3}

	nodeCount := 0
	versions, canShift := parser.cDoAllPotentialReductions(start, 0, Token{}, &nodeCount, arena, nil, &scratch, nil)
	if canShift {
		t.Fatal("canShift = true, want false")
	}
	if len(versions) != 1 {
		t.Fatalf("version count = %d, want trailing-extra same-pop slices collapsed to one C version", len(versions))
	}
	if got := versions[0].gss.head.linkCount(); got != 1 {
		t.Fatalf("replayed extra link count = %d, want one selected path", got)
	}
	if stackEntryNode(versions[0].top()) != altExtra {
		t.Fatal("same-pop trailing-extra collapse did not replay the selected extras")
	}
	parent := stackEntryNode(versions[0].gss.head.prev.entry)
	if parent == nil || parent.symbol != 4 {
		t.Fatalf("extra predecessor = %+v, want reduced parent symbol 4", parent)
	}
	if len(parent.children) != 2 || parent.children[1] != altRight {
		t.Fatal("trailing-extra same-pop collapse did not keep the C-selected parent children")
	}
}

func TestCDoAllPotentialReductionsRetainsDistinctPopTargetSlices(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	parser := newCRecoverySyntheticReduceParser()

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	left := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	right := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	leftNode := scratch.allocNode(newStackEntryNode(2, left), base, 2)
	rightNode := scratch.allocNode(newStackEntryNode(3, right), leftNode, 3)

	altBase := scratch.allocNode(stackEntry{state: 7}, nil, 1)
	altLeft := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	altLeftNode := scratch.allocNode(newStackEntryNode(8, altLeft), altBase, 2)
	altRight := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	rightNode.extraLinks = append(rightNode.extraLinks, gssMainLink{
		prev:  altLeftNode,
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
	if len(versions) != 2 {
		t.Fatalf("version count = %d, want distinct pop targets retained separately", len(versions))
	}
	gotStates := map[StateID]bool{}
	for i := range versions {
		top := stackEntryNode(versions[i].top())
		if top == nil || top.symbol != 4 {
			t.Fatalf("version %d top = %+v, want reduced parent symbol 4", i, top)
		}
		if got := versions[i].gss.head.linkCount(); got != 1 {
			t.Fatalf("version %d reduced top link count = %d, want one alternative", i, got)
		}
		gotStates[versions[i].top().state] = true
	}
	if !gotStates[1] || !gotStates[7] {
		t.Fatalf("version top states = %v, want distinct pop target states 1 and 7", gotStates)
	}
}

func TestCDoAllPotentialReductionsMergesHeaderEquivalentDistinctPopTargets(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	parser := newCRecoverySyntheticReduceParser()

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	left := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	right := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	leftNode := scratch.allocNode(newStackEntryNode(2, left), base, 2)
	rightNode := scratch.allocNode(newStackEntryNode(3, right), leftNode, 3)

	altBase := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	altLeft := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	altLeftNode := scratch.allocNode(newStackEntryNode(2, altLeft), altBase, 2)
	altRight := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	rightNode.extraLinks = append(rightNode.extraLinks, gssMainLink{
		prev:  altLeftNode,
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
		t.Fatalf("version count = %d, want header-equivalent distinct pop targets merged into one C version", len(versions))
	}
	top := stackEntryNode(versions[0].top())
	if top == nil || top.symbol != 4 {
		t.Fatalf("current version top = %+v, want reduced parent symbol 4", top)
	}
	if got := versions[0].gss.head.linkCount(); got != 2 {
		t.Fatalf("reduced top link count = %d, want ts_stack_merge-style alternatives for distinct pop targets", got)
	}
}

func TestCAppendActionReductionsCollapseSamePopBeforeOlderMerge(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	parser := newCRecoverySyntheticReduceParser()

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	olderPop := scratch.allocNode(stackEntry{state: 8}, nil, 1)
	popTo := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	original := glrStack{gss: gssStack{head: scratch.allocNode(stackEntry{state: 3}, popTo, 2)}, byteOffset: 2}

	olderChild := newLeafNodeInArena(arena, 2, true, 0, 1, Point{}, Point{Column: 1})
	lowChild := newLeafNodeInArena(arena, 2, true, 0, 1, Point{}, Point{Column: 1})
	selectedChild := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	olderParent := newParentNodeInArena(arena, 4, true, []*Node{olderChild}, nil, 0)
	lowParent := newParentNodeInArena(arena, 4, true, []*Node{lowChild}, nil, 0)
	selectedParent := newParentNodeInArena(arena, 4, true, []*Node{selectedChild}, nil, 0)

	versions := []glrStack{
		{gss: gssStack{head: scratch.allocNode(newStackEntryNode(9, olderParent), olderPop, 2)}, byteOffset: 2},
		original,
	}
	candidates := []glrStack{
		{gss: gssStack{head: scratch.allocNode(newStackEntryNode(9, lowParent), popTo, 2)}, byteOffset: 2},
		{gss: gssStack{head: scratch.allocNode(newStackEntryNode(9, selectedParent), popTo, 2)}, byteOffset: 2},
	}

	candidates = parser.cCollapseSamePopReductionCandidates(candidates)
	if len(candidates) != 1 {
		t.Fatalf("collapsed candidates = %d, want one same-pop representative", len(candidates))
	}
	if stackEntryNode(candidates[0].top()) != selectedParent {
		t.Fatal("same-pop action-local collapse kept the wrong parent before older merge")
	}
	actionStartLen := len(versions)
	appendedAny := false
	for i := range candidates {
		var appended bool
		versions, appended = parser.cAppendReductionVersion(versions, candidates[i], 1, actionStartLen)
		appendedAny = appendedAny || appended
	}
	if appendedAny {
		t.Fatal("selected same-pop representative appended instead of merging into older header-equivalent version")
	}
	if got := versions[0].gss.head.linkCount(); got != 2 {
		t.Fatalf("older merged head link count = %d, want one selected reduction link added", got)
	}
	_, mergedEntry := versions[0].gss.head.link(1)
	if stackEntryNode(mergedEntry) != selectedParent {
		t.Fatal("older header-equivalent merge received the unselected same-pop parent")
	}
}

func TestCDoAllPotentialReductionsOverwritesReductionVersionWithLastAction(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	parser := newCRecoverySyntheticReduceParser()
	parser.language.ParseActions[1] = ParseActionEntry{Actions: []ParseAction{
		{Type: ParseActionReduce, Symbol: 4, ChildCount: 2},
		{Type: ParseActionReduce, Symbol: 5, ChildCount: 1},
	}}

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	left := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	right := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	leftNode := scratch.allocNode(newStackEntryNode(2, left), base, 2)
	rightNode := scratch.allocNode(newStackEntryNode(3, right), leftNode, 3)
	start := glrStack{gss: gssStack{head: rightNode}, byteOffset: 2}

	nodeCount := 0
	versions, canShift := parser.cDoAllPotentialReductions(start, 0, Token{}, &nodeCount, arena, nil, &scratch, nil)
	if canShift {
		t.Fatal("canShift = true, want false")
	}
	if len(versions) != 2 {
		t.Fatalf("version count = %d, want current last reduction plus earlier sibling", len(versions))
	}
	current := stackEntryNode(versions[0].top())
	if current == nil || current.symbol != 5 {
		t.Fatalf("current version top = %+v, want last reduce action symbol 5", current)
	}
	sibling := stackEntryNode(versions[1].top())
	if sibling == nil || sibling.symbol != 4 {
		t.Fatalf("sibling version top = %+v, want earlier reduce action symbol 4", sibling)
	}
}

func TestCDoAllPotentialReductionsLastNoNewReductionPreventsRenumber(t *testing.T) {
	old := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	t.Cleanup(func() { glrFaithfulCapOneMerge = old })

	parser := newCRecoverySyntheticReduceParser()
	parser.language.ParseActions[1] = ParseActionEntry{Actions: []ParseAction{
		{Type: ParseActionReduce, Symbol: 4, ChildCount: 2},
		{Type: ParseActionReduce, Symbol: 5, ChildCount: 4},
	}}

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	var scratch gssScratch
	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	left := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	right := newLeafNodeInArena(arena, 2, true, 1, 2, Point{Column: 1}, Point{Column: 2})
	leftNode := scratch.allocNode(newStackEntryNode(2, left), base, 2)
	rightNode := scratch.allocNode(newStackEntryNode(3, right), leftNode, 3)
	start := glrStack{gss: gssStack{head: rightNode}, byteOffset: 2}

	nodeCount := 0
	versions, canShift := parser.cDoAllPotentialReductions(start, 0, Token{}, &nodeCount, arena, nil, &scratch, nil)
	if canShift {
		t.Fatal("canShift = true, want false")
	}
	if len(versions) != 2 {
		t.Fatalf("version count = %d, want unreduced current plus earlier reduction sibling", len(versions))
	}
	if versions[0].gss.head != rightNode {
		top := stackEntryNode(versions[0].top())
		t.Fatalf("current version top = %+v, want last STACK_VERSION_NONE to leave original version unrenumbered", top)
	}
	sibling := stackEntryNode(versions[1].top())
	if sibling == nil || sibling.symbol != 4 {
		t.Fatalf("sibling version top = %+v, want earlier reduce action symbol 4", sibling)
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

func TestCRecoverToStateEnumeratesMergedPopSlices(t *testing.T) {
	lang := &Language{
		SymbolNames: []string{"end", "prefix", "leaf"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "end", Visible: true, Named: true},
			{Name: "prefix", Visible: true, Named: true},
			{Name: "leaf", Visible: true, Named: true},
		},
	}
	parser := &Parser{language: lang}

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	leaf := func(start, end uint32) *Node {
		return newLeafNodeInArena(arena, 2, true, start, end, Point{Column: start}, Point{Column: end})
	}

	var scratch gssScratch
	baseA := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	prefixA := scratch.allocNode(newStackEntryNode(10, leaf(0, 1)), baseA, 2)
	head := scratch.allocNode(newStackEntryNode(30, leaf(10, 11)), prefixA, 3)

	baseB := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	prefixB := scratch.allocNode(newStackEntryNode(10, leaf(2, 3)), baseB, 2)
	head.extraLinks = append(head.extraLinks, gssMainLink{
		prev:  prefixB,
		entry: newStackEntryNode(30, leaf(20, 21)),
	})

	stack := glrStack{gss: gssStack{head: head}, byteOffset: 21}
	forks := parser.cRecoverToStateForks(&stack, 1, 10, 4, arena, nil, &scratch, nil)
	if len(forks) != 2 {
		t.Fatalf("fork count = %d, want 2", len(forks))
	}
	wantStarts := []uint32{10, 20}
	wantEnds := []uint32{11, 21}
	for i := range forks {
		top := stackEntryNode(forks[i].top())
		if top == nil {
			t.Fatalf("fork %d top = nil", i)
		}
		if top.symbol != errorSymbol || !top.isExtra() || !top.hasError() {
			t.Fatalf("fork %d top = symbol %d extra=%t has_error=%t, want extra ERROR",
				i, top.symbol, top.isExtra(), top.hasError())
		}
		if forks[i].top().state != 10 || top.preGotoState != 10 || top.parseState != 10 {
			t.Fatalf("fork %d state/pre-goto/parse = %d/%d/%d, want 10/10/10",
				i, forks[i].top().state, top.preGotoState, top.parseState)
		}
		if top.startByte != wantStarts[i] || top.endByte != wantEnds[i] {
			t.Fatalf("fork %d error span = %d:%d, want %d:%d",
				i, top.startByte, top.endByte, wantStarts[i], wantEnds[i])
		}
	}
}

func TestCRecoverToStatePopsClosedErrorFromPackedTopLink(t *testing.T) {
	lang := &Language{
		SymbolNames: []string{"end", "closed_child", "payload"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "end", Visible: true, Named: true},
			{Name: "closed_child", Visible: true, Named: true},
			{Name: "payload", Visible: true, Named: true},
		},
	}
	parser := &Parser{language: lang}

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	leaf := func(sym Symbol, start, end uint32) *Node {
		return newLeafNodeInArena(arena, sym, true, start, end, Point{Column: start}, Point{Column: end})
	}
	closedChild := leaf(1, 0, 1)
	closedErr := newParentNodeInArena(arena, errorSymbol, true, []*Node{closedChild}, nil, 0)
	cSetNodeSpan(closedErr, 0, 1, Point{}, Point{Column: 1})
	closedErr.setHasError(true)

	var scratch gssScratch
	base := scratch.allocNode(stackEntry{state: 1}, nil, 1)
	goalHead := scratch.allocNode(newStackEntryNode(10, leaf(2, 4, 5)), base, 2)
	goalHead.extraLinks = append(goalHead.extraLinks, gssMainLink{
		prev:  base,
		entry: newStackEntryNode(10, closedErr),
	})
	stack := glrStack{gss: gssStack{head: goalHead}, byteOffset: 5}

	payload := leaf(2, 10, 11)
	fork, ok := parser.cRecoverToStateSlice(
		&stack,
		1,
		10,
		cRecoverPopSlice{popTo: goalHead, window: []stackEntry{newStackEntryNode(30, payload)}},
		arena,
		nil,
		&scratch,
		nil,
	)
	if !ok {
		t.Fatal("cRecoverToStateSlice failed")
	}
	var forkPrev *gssNode
	if fork.gss.head != nil {
		forkPrev = fork.gss.head.prev
	}
	if fork.gss.head == nil || forkPrev != base {
		t.Fatalf("fork did not pop the packed ERROR link to base; head=%p prev=%p base=%p", fork.gss.head, forkPrev, base)
	}
	top := stackEntryNode(fork.top())
	if top == nil || top.symbol != errorSymbol {
		t.Fatalf("top = %+v, want recovered ERROR", top)
	}
	if len(top.children) != 2 || top.children[0] != closedChild || top.children[1] != payload {
		t.Fatalf("recovered ERROR children = %+v, want closed ERROR child then payload", top.children)
	}
	if top.startByte != 0 || top.endByte != 11 {
		t.Fatalf("recovered ERROR span = %d:%d, want 0:11", top.startByte, top.endByte)
	}
}

func TestCRecoverPopSlicesDoesNotCapBeforeLaterSlices(t *testing.T) {
	lang := &Language{
		SymbolNames: []string{"end", "leaf"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "end", Visible: true, Named: true},
			{Name: "leaf", Visible: true, Named: true},
		},
	}
	parser := &Parser{language: lang}

	arena := acquireNodeArena(arenaClassFull)
	defer arena.Release()

	leaf := func(start uint32) *Node {
		return newLeafNodeInArena(arena, 1, true, start, start+1, Point{Column: start}, Point{Column: start + 1})
	}
	entry := func(i int) stackEntry {
		return newStackEntryNode(StateID(20+i), leaf(uint32(i)))
	}

	var scratch gssScratch
	goals := make([]*gssNode, cRecoverMaxVersionCount+1)
	for i := range goals {
		goals[i] = scratch.allocNode(stackEntry{state: 10}, nil, 1)
	}
	mid0 := scratch.allocNode(entry(100), goals[0], 2)
	for i := 1; i < cRecoverMaxVersionCount; i++ {
		mid0.extraLinks = append(mid0.extraLinks, gssMainLink{
			prev:  goals[i],
			entry: entry(100 + i),
		})
	}
	mid1 := scratch.allocNode(entry(200), goals[cRecoverMaxVersionCount], 2)
	head := scratch.allocNode(entry(300), mid0, 3)
	head.extraLinks = append(head.extraLinks, gssMainLink{
		prev:  mid1,
		entry: entry(301),
	})
	stack := glrStack{gss: gssStack{head: head}, byteOffset: 302}

	slices := parser.cRecoverPopSlices(&stack, 2, 10, &scratch)
	if len(slices) != cRecoverMaxVersionCount+1 {
		t.Fatalf("slice count = %d, want %d", len(slices), cRecoverMaxVersionCount+1)
	}
	foundLater := false
	for _, slice := range slices {
		if slice.popTo == goals[cRecoverMaxVersionCount] {
			foundLater = true
			break
		}
	}
	if !foundLater {
		t.Fatalf("later viable slice beyond version cap was not enumerated: %+v", slices)
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
