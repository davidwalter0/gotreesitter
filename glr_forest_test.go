package gotreesitter

import (
	"fmt"
	"strings"
	"testing"
)

type forestZeroWidthRetryScanner struct{}

func (forestZeroWidthRetryScanner) Create() any               { return nil }
func (forestZeroWidthRetryScanner) Destroy(any)               {}
func (forestZeroWidthRetryScanner) Serialize(any, []byte) int { return 0 }
func (forestZeroWidthRetryScanner) Deserialize(any, []byte)   {}
func (forestZeroWidthRetryScanner) Scan(_ any, lexer *ExternalLexer, valid []bool) bool {
	if len(valid) == 0 || !valid[0] || lexer.Lookahead() != 'x' {
		return false
	}
	lexer.SetResultSymbol(1)
	return true
}

// pathsOf reduces childCount children over node and returns each visited path as
// "child-states|popToState" so order and fan-out are easy to assert.
func pathsOf(node *gssForestNode, childCount int) []string {
	var out []string
	reduceOverForest(node, childCount, func(children []stackEntry, _ int, popTo *gssForestNode) {
		states := make([]uint32, len(children))
		for i, c := range children {
			states[i] = uint32(c.state)
		}
		out = append(out, fmt.Sprintf("%v|%d", states, popTo.state))
	})
	return out
}

func TestReduceOverForestLinearChain(t *testing.T) {
	// n0 <-(a:10)- n1 <-(b:11)- n2 <-(c:12)- n3
	n0 := &gssForestNode{state: 0, byteOffset: 0}
	n1 := &gssForestNode{state: 1, links: []gssLink{{prev: n0, subtree: stackEntry{state: 10}}}}
	n2 := &gssForestNode{state: 2, links: []gssLink{{prev: n1, subtree: stackEntry{state: 11}}}}
	n3 := &gssForestNode{state: 3, links: []gssLink{{prev: n2, subtree: stackEntry{state: 12}}}}

	// reduce 2 children over n3 → [b,c] = [11 12], pop back to n1.
	got := pathsOf(n3, 2)
	want := []string{"[11 12]|1"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("childCount=2: got %v want %v", got, want)
	}
	// reduce 0 children → empty, pop to n3 itself.
	if got := pathsOf(n3, 0); fmt.Sprint(got) != "[[]|3]" {
		t.Fatalf("childCount=0: got %v", got)
	}
	// reduce all 3 → [a b c] = [10 11 12], pop to n0.
	if got := pathsOf(n3, 3); fmt.Sprint(got) != "[[10 11 12]|0]" {
		t.Fatalf("childCount=3: got %v", got)
	}
}

func TestParseForestNoLookaheadReductionContinuesToRealToken(t *testing.T) {
	lang := &Language{
		Name:               "forest_no_lookahead",
		SymbolCount:        5,
		TokenCount:         3,
		StateCount:         6,
		LargeStateCount:    6,
		InitialState:       1,
		ProductionIDCount:  2,
		ExternalTokenCount: 0,
		SymbolNames:        []string{"end", "a", "b", "source_file", "prefix"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "end", Visible: false, Named: false},
			{Name: "a", Visible: true, Named: false},
			{Name: "b", Visible: true, Named: false},
			{Name: "source_file", Visible: true, Named: true},
			{Name: "prefix", Visible: true, Named: true},
		},
		FieldNames: []string{""},
		ParseActions: []ParseActionEntry{
			{Actions: nil},
			{Actions: []ParseAction{{Type: ParseActionShift, State: 2}}},
			{Actions: []ParseAction{{Type: ParseActionReduce, Symbol: 4, ChildCount: 1, ProductionID: 0}}},
			{Actions: []ParseAction{{Type: ParseActionShift, State: 4}}},
			{Actions: []ParseAction{{Type: ParseActionReduce, Symbol: 3, ChildCount: 2, ProductionID: 1}}},
			{Actions: []ParseAction{{Type: ParseActionAccept}}},
		},
		ParseTable: [][]uint16{
			{0, 0, 0, 0, 0},
			{0, 1, 0, 5, 3},
			{2, 0, 0, 0, 0},
			{0, 0, 3, 0, 0},
			{4, 0, 0, 0, 0},
			{5, 0, 0, 0, 0},
		},
		LexModes: []LexMode{
			{LexState: 0},
			{LexState: 0},
			{LexStateID: noLookaheadLexState},
			{LexState: 0},
			{LexState: 0},
			{LexState: 0},
		},
		LexStates: []LexState{
			{
				Default: -1,
				EOF:     -1,
				Transitions: []LexTransition{
					{Lo: 'a', Hi: 'a', NextState: 1},
					{Lo: 'b', Hi: 'b', NextState: 2},
				},
			},
			{AcceptToken: 1, Default: -1, EOF: -1},
			{AcceptToken: 2, Default: -1, EOF: -1},
		},
	}

	tree, ok := NewParser(lang).ParseForestExperimental([]byte("ab"))
	if !ok || tree == nil {
		t.Fatal("ParseForestExperimental failed after no-lookahead reduction")
	}
	defer tree.Release()
	root := tree.RootNode()
	if root == nil {
		t.Fatal("root is nil")
	}
	if got, want := root.Type(lang), "source_file"; got != want {
		t.Fatalf("root type = %q, want %q", got, want)
	}
	if got, want := root.EndByte(), uint32(2); got != want {
		t.Fatalf("root end = %d, want %d", got, want)
	}
	if got, want := root.ChildCount(), 2; got != want {
		t.Fatalf("root child count = %d, want %d", got, want)
	}
	if got, want := root.Child(0).Type(lang), "prefix"; got != want {
		t.Fatalf("first child type = %q, want %q", got, want)
	}
	if got, want := root.Child(1).Type(lang), "b"; got != want {
		t.Fatalf("second child type = %q, want %q", got, want)
	}
}

func TestParseForestNoLookaheadSameStateReductionMaterializesExtra(t *testing.T) {
	lang := &Language{
		Name:              "forest_no_lookahead_same_state_extra",
		SymbolCount:       6,
		TokenCount:        4,
		StateCount:        6,
		LargeStateCount:   6,
		InitialState:      1,
		ProductionIDCount: 2,
		SymbolNames:       []string{"end", "x", "a", "b", "source_file", "ghost"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "end"},
			{Name: "x", Visible: true, Named: true},
			{Name: "a", Visible: true, Named: true},
			{Name: "b", Visible: true, Named: true},
			{Name: "source_file", Visible: true, Named: true},
			{Name: "ghost", Visible: true, Named: true},
		},
		FieldNames: []string{""},
		ParseActions: []ParseActionEntry{
			{Actions: nil},
			{Actions: []ParseAction{{Type: ParseActionShift, State: 2}}},
			{Actions: []ParseAction{{Type: ParseActionShift, State: 3}}},
			{Actions: []ParseAction{{Type: ParseActionShift, State: 4}}},
			{Actions: []ParseAction{{Type: ParseActionReduce, Symbol: 5, ChildCount: 1, ProductionID: 0}}},
			{Actions: []ParseAction{{Type: ParseActionReduce, Symbol: 4, ChildCount: 2, ProductionID: 1}}},
			{Actions: []ParseAction{{Type: ParseActionAccept}}},
		},
		ParseTable: [][]uint16{
			{0, 0, 0, 0, 0, 0},
			{0, 1, 0, 0, 5, 0},
			{0, 0, 2, 3, 0, 2},
			{4, 0, 0, 0, 0, 0},
			{5, 0, 0, 0, 0, 0},
			{6, 0, 0, 0, 0, 0},
		},
		LexModes: []LexMode{
			{LexState: 0},
			{LexState: 0},
			{LexState: 0},
			{LexStateID: noLookaheadLexState},
			{LexState: 0},
			{LexState: 0},
		},
		LexStates: []LexState{
			{
				Default: -1,
				EOF:     -1,
				Transitions: []LexTransition{
					{Lo: 'x', Hi: 'x', NextState: 1},
					{Lo: 'a', Hi: 'a', NextState: 2},
					{Lo: 'b', Hi: 'b', NextState: 3},
				},
			},
			{AcceptToken: 1, Default: -1, EOF: -1},
			{AcceptToken: 2, Default: -1, EOF: -1},
			{AcceptToken: 3, Default: -1, EOF: -1},
		},
	}

	tree, ok := NewParser(lang).ParseForestExperimental([]byte("xab"))
	if !ok || tree == nil {
		t.Fatal("ParseForestExperimental failed for same-state no-lookahead reduction")
	}
	defer tree.Release()
	root := tree.RootNode()
	if root == nil {
		t.Fatal("root is nil")
	}
	if got, want := root.SExpr(lang), "(source_file (x) (ghost (a)) (b))"; got != want {
		t.Fatalf("root SExpr = %s, want %s", got, want)
	}
	if got, want := root.ChildCount(), 3; got != want {
		t.Fatalf("root child count = %d, want %d", got, want)
	}
	ghost := root.Child(1)
	if ghost == nil {
		t.Fatal("root child[1] is nil")
	}
	if got, want := ghost.Type(lang), "ghost"; got != want {
		t.Fatalf("root child[1] type = %q, want %q", got, want)
	}
	if !ghost.IsExtra() {
		t.Fatal("same-state no-lookahead reduction was not materialized as an extra")
	}
	if got, want := root.Child(0).Type(lang), "x"; got != want {
		t.Fatalf("root child[0] type = %q, want %q", got, want)
	}
	if got, want := root.Child(2).Type(lang), "b"; got != want {
		t.Fatalf("root child[2] type = %q, want %q", got, want)
	}
}

func TestParseForestRetriesUnshiftableZeroWidthExternalToken(t *testing.T) {
	lang := &Language{
		Name:               "forest_zero_width_external_retry",
		SymbolCount:        5,
		TokenCount:         3,
		StateCount:         4,
		LargeStateCount:    4,
		InitialState:       1,
		ProductionIDCount:  2,
		ExternalTokenCount: 1,
		ExternalSymbols:    []Symbol{1},
		ExternalScanner:    forestZeroWidthRetryScanner{},
		SymbolNames:        []string{"end", "_zero", "x", "source_file", "dead_reduce"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "end", Visible: false, Named: false},
			{Name: "_zero", Visible: false, Named: false},
			{Name: "x", Visible: true, Named: false},
			{Name: "source_file", Visible: true, Named: true},
			{Name: "dead_reduce", Visible: true, Named: true},
		},
		FieldNames: []string{""},
		ExternalLexStates: [][]bool{
			{false},
			{true},
		},
		ParseActions: []ParseActionEntry{
			{Actions: nil},
			{Actions: []ParseAction{{Type: ParseActionShift, State: 2}}},
			{Actions: []ParseAction{{Type: ParseActionReduce, Symbol: 4, ChildCount: 0, ProductionID: 0}}},
			{Actions: []ParseAction{{Type: ParseActionReduce, Symbol: 3, ChildCount: 1, ProductionID: 1}}},
			{Actions: []ParseAction{{Type: ParseActionAccept}}},
		},
		ParseTable: [][]uint16{
			{0, 0, 0, 0, 0},
			{0, 2, 1, 3, 0},
			{3, 0, 0, 0, 0},
			{4, 0, 0, 0, 0},
		},
		LexModes: []LexMode{
			{LexState: 0},
			{LexState: 0, ExternalLexState: 1},
			{LexState: 0},
			{LexState: 0},
		},
		LexStates: []LexState{
			{
				Default: -1,
				EOF:     -1,
				Transitions: []LexTransition{
					{Lo: 'x', Hi: 'x', NextState: 1},
				},
			},
			{AcceptToken: 2, Default: -1, EOF: -1},
		},
	}

	tree, ok := NewParser(lang).ParseForestExperimental([]byte("x"))
	if !ok || tree == nil {
		t.Fatal("ParseForestExperimental failed after unshiftable zero-width external")
	}
	defer tree.Release()
	root := tree.RootNode()
	if root == nil {
		t.Fatal("root is nil")
	}
	if got, want := root.Type(lang), "source_file"; got != want {
		t.Fatalf("root type = %q, want %q", got, want)
	}
	if got, want := root.EndByte(), uint32(1); got != want {
		t.Fatalf("root end = %d, want %d", got, want)
	}
	if got, want := root.ChildCount(), 1; got != want {
		t.Fatalf("root child count = %d, want %d", got, want)
	}
	if got, want := root.Child(0).Type(lang), "x"; got != want {
		t.Fatalf("child type = %q, want %q", got, want)
	}
}

func TestForestResolveConflictPrefersBlockCommentRepetitionShift(t *testing.T) {
	p := NewParser(&Language{
		Name:        "forest_repetition_conflict",
		SymbolNames: []string{"end", "block_comment_token1", "x", "block_comment_repeat1"},
	})
	actions := []ParseAction{
		{Type: ParseActionReduce, Symbol: 3, ChildCount: 1},
		{Type: ParseActionShift, State: 2, Repetition: true},
	}

	got := p.forestResolveConflict(actions, Token{Symbol: 1})
	if len(got) != 1 {
		t.Fatalf("resolved actions len = %d, want 1", len(got))
	}
	if got[0].Type != ParseActionShift || !got[0].Repetition || got[0].State != 2 {
		t.Fatalf("resolved action = %+v, want repetition shift to state 2", got[0])
	}

	ordinary := p.forestResolveConflict(actions, Token{Symbol: 2})
	if len(ordinary) != len(actions) {
		t.Fatalf("ordinary repetition conflict resolved len = %d, want %d", len(ordinary), len(actions))
	}
}

func TestForestTraceTransitionGateRequiresFlagAndWindow(t *testing.T) {
	oldTransitions := glrForestTraceTransitions
	oldWindow := glrForestTraceWindow
	t.Cleanup(func() {
		glrForestTraceTransitions = oldTransitions
		glrForestTraceWindow = oldWindow
	})

	glrForestTraceTransitions = false
	glrForestTraceWindow = glrForestTraceWindowConfig{enabled: true, start: 1, end: 2}
	if forestTraceTransitionEnabled() {
		t.Fatal("transition trace enabled without flag")
	}

	glrForestTraceTransitions = true
	glrForestTraceWindow = glrForestTraceWindowConfig{}
	if forestTraceTransitionEnabled() {
		t.Fatal("transition trace enabled without window")
	}

	glrForestTraceTransitions = true
	glrForestTraceWindow = glrForestTraceWindowConfig{enabled: true, start: 1, end: 2}
	if !forestTraceTransitionEnabled() {
		t.Fatal("transition trace disabled with flag and window")
	}
}

func TestForestTraceActionsIncludeConflictRelevantFields(t *testing.T) {
	lang := &Language{SymbolNames: []string{"end", "identifier"}}
	got := forestTraceActions(lang, []ParseAction{
		{Type: ParseActionReduce, Symbol: 1, ChildCount: 1, DynamicPrecedence: 2, ProductionID: 7},
		{Type: ParseActionShift, State: 42, Extra: true, Repetition: true},
	})
	for _, want := range []string{
		"reduce(sym=1(identifier) cc=1 dyn=2 prod=7)",
		"shift(state=42 extra,repeat)",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("forestTraceActions() = %q, missing %q", got, want)
		}
	}
}

func TestReduceOverForestLinearChainWithExtra(t *testing.T) {
	// n0 <-(a:10)- n1 <-(b:11)- n2 <-(extra:90)- n3 <-(c:12)- n4
	extra := &Node{}
	extra.setExtra(true)
	n0 := &gssForestNode{state: 0, byteOffset: 0}
	n1 := &gssForestNode{state: 1, links: []gssLink{{prev: n0, subtree: stackEntry{state: 10}}}}
	n2 := &gssForestNode{state: 2, links: []gssLink{{prev: n1, subtree: stackEntry{state: 11}}}}
	n3 := &gssForestNode{state: 3, links: []gssLink{{prev: n2, subtree: newStackEntryNode(90, extra)}}}
	n4 := &gssForestNode{state: 4, links: []gssLink{{prev: n3, subtree: stackEntry{state: 12}}}}

	// Extras are included in the reduce window but do not count toward childCount.
	got := pathsOf(n4, 2)
	want := []string{"[11 90 12]|1"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("childCount=2 with extra: got %v want %v", got, want)
	}
}

func TestReduceOverForestForkedNode(t *testing.T) {
	// Shared base n0 <-(a:10)- n1, then two alternatives reaching a coalesced n3:
	//   path A: n1 <-(b:11)- n2  <-(c:12)- n3
	//   path B: n1 <-(x:21)- n2a <-(y:22)- n3
	n0 := &gssForestNode{state: 0}
	n1 := &gssForestNode{state: 1, links: []gssLink{{prev: n0, subtree: stackEntry{state: 10}}}}
	n2 := &gssForestNode{state: 2, links: []gssLink{{prev: n1, subtree: stackEntry{state: 11}}}}
	n2a := &gssForestNode{state: 20, links: []gssLink{{prev: n1, subtree: stackEntry{state: 21}}}}
	n3 := &gssForestNode{state: 3, links: []gssLink{
		{prev: n2, subtree: stackEntry{state: 12}},
		{prev: n2a, subtree: stackEntry{state: 22}},
	}}

	// reduce 2 children over the coalesced n3 → BOTH alternatives, each popping to n1.
	got := pathsOf(n3, 2)
	want := []string{"[11 12]|1", "[21 22]|1"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("forked childCount=2: got %v want %v", got, want)
	}
	if got, want := pathsOf(n3, 1), []string{"[12]|2", "[22]|20"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("forked childCount=1: got %v want %v", got, want)
	}
}

func TestReduceOverForestNestedForkNoExtras(t *testing.T) {
	// Two no-extra fork levels:
	//   n0 <-(a:10 or b:20)- n1 <-(c:11 or d:21)- n2
	// The reducer should enumerate the cartesian product in left-to-right order.
	n0 := &gssForestNode{state: 0}
	n1 := &gssForestNode{state: 1, links: []gssLink{
		{prev: n0, subtree: stackEntry{state: 10}},
		{prev: n0, subtree: stackEntry{state: 20}},
	}, noExtraDepth: 1}
	n2 := &gssForestNode{state: 2, links: []gssLink{
		{prev: n1, subtree: stackEntry{state: 11}},
		{prev: n1, subtree: stackEntry{state: 21}},
	}, noExtraDepth: 2}

	got := pathsOf(n2, 2)
	want := []string{"[10 11]|0", "[20 11]|0", "[10 21]|0", "[20 21]|0"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("nested fork no-extra childCount=2: got %v want %v", got, want)
	}
}

func TestReduceOverForestNestedForkNoExtrasChildCount4(t *testing.T) {
	n0 := &gssForestNode{state: 0}
	n1 := &gssForestNode{state: 1, links: []gssLink{
		{prev: n0, subtree: stackEntry{state: 10}},
		{prev: n0, subtree: stackEntry{state: 20}},
	}, noExtraDepth: 1}
	n2 := &gssForestNode{state: 2, links: []gssLink{
		{prev: n1, subtree: stackEntry{state: 11}},
		{prev: n1, subtree: stackEntry{state: 21}},
	}, noExtraDepth: 2}
	n3 := &gssForestNode{state: 3, links: []gssLink{
		{prev: n2, subtree: stackEntry{state: 12}},
		{prev: n2, subtree: stackEntry{state: 22}},
	}, noExtraDepth: 3}
	n4 := &gssForestNode{state: 4, links: []gssLink{
		{prev: n3, subtree: stackEntry{state: 13}},
		{prev: n3, subtree: stackEntry{state: 23}},
	}, noExtraDepth: 4}

	got := pathsOf(n4, 4)
	want := []string{
		"[10 11 12 13]|0", "[20 11 12 13]|0",
		"[10 21 12 13]|0", "[20 21 12 13]|0",
		"[10 11 22 13]|0", "[20 11 22 13]|0",
		"[10 21 22 13]|0", "[20 21 22 13]|0",
		"[10 11 12 23]|0", "[20 11 12 23]|0",
		"[10 21 12 23]|0", "[20 21 12 23]|0",
		"[10 11 22 23]|0", "[20 11 22 23]|0",
		"[10 21 22 23]|0", "[20 21 22 23]|0",
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("nested fork no-extra childCount=4: got %v want %v", got, want)
	}
}

func TestReduceOverForestForkedLinearWithExtra(t *testing.T) {
	extra := &Node{}
	extra.setExtra(true)
	n0 := &gssForestNode{state: 0}
	n1 := &gssForestNode{state: 1, links: []gssLink{{prev: n0, subtree: stackEntry{state: 10}}}}
	n2 := &gssForestNode{state: 2, links: []gssLink{{prev: n1, subtree: stackEntry{state: 11}}}}
	n3 := &gssForestNode{state: 3, links: []gssLink{{prev: n2, subtree: newStackEntryNode(90, extra)}}}
	n2a := &gssForestNode{state: 20, links: []gssLink{{prev: n1, subtree: stackEntry{state: 21}}}}
	n4 := &gssForestNode{state: 4, links: []gssLink{
		{prev: n3, subtree: stackEntry{state: 12}},
		{prev: n2a, subtree: stackEntry{state: 22}},
	}}

	got := pathsOf(n4, 2)
	want := []string{"[11 90 12]|1", "[21 22]|1"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("forked linear with extra: got %v want %v", got, want)
	}
}

func TestReduceOverForestLinearPrefixForkedWithExtra(t *testing.T) {
	extra := &Node{}
	extra.setExtra(true)
	n0 := &gssForestNode{state: 0}
	n1 := &gssForestNode{state: 1, links: []gssLink{{prev: n0, subtree: stackEntry{state: 11}}}}
	n2 := &gssForestNode{state: 2, links: []gssLink{
		{prev: n1, subtree: newStackEntryNode(90, extra)},
		{prev: n0, subtree: stackEntry{state: 21}},
	}}
	n3 := &gssForestNode{state: 3, links: []gssLink{{prev: n2, subtree: stackEntry{state: 31}}}}

	got := pathsOf(n3, 2)
	want := []string{"[11 90 31]|0", "[21 31]|0"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("linear prefix forked with extra: got %v want %v", got, want)
	}
}

// TestCoalesceForestSharesNode proves coalesceForest dedups by (state,byteOffset)
// into one node with multiple links — the O(1), no-deep-compare mechanism.
func TestCoalesceForestSharesNode(t *testing.T) {
	idx := newGSSForestIndex(0)
	slab := &gssForestNodeSlab{}
	base := &gssForestNode{state: 0}
	// Two distinct parses reach (state=5, byteOffset=42).
	a := coalesceForest(&idx, slab, 5, 42, base, stackEntry{state: 100}, 3, 0)
	b := coalesceForest(&idx, slab, 5, 42, base, stackEntry{state: 101}, 7, 0)
	if a != b {
		t.Fatal("coalesceForest created two nodes for the same (state,byteOffset)")
	}
	if len(a.links) != 2 {
		t.Fatalf("want 2 links on the coalesced node, got %d", len(a.links))
	}
	if a.noExtraDepth != 1 {
		t.Fatalf("want no-extra depth 1, got %d", a.noExtraDepth)
	}
	if a.minLinkScore != 3 {
		t.Fatalf("want min link score 3, got %d", a.minLinkScore)
	}
	if best := a.bestLink(); best == nil || best.score != 7 {
		t.Fatalf("want best link score 7, got %v", best)
	}
	// A different (state,byteOffset) is a separate node.
	c := coalesceForest(&idx, slab, 6, 42, base, stackEntry{state: 102}, 1, 0)
	if c == a {
		t.Fatal("distinct (state,byteOffset) coalesced into the same node")
	}
}

func TestCoalesceForestRefreshesMinLinkScoreOnReplacement(t *testing.T) {
	idx := newGSSForestIndex(0)
	slab := &gssForestNodeSlab{}
	base := &gssForestNode{state: 0}
	loser := newStackEntryNode(100, &Node{symbol: 100, startByte: 1, endByte: 4})
	winner := newStackEntryNode(100, &Node{symbol: 100, startByte: 1, endByte: 4})

	node := coalesceForest(&idx, slab, 5, 4, base, loser, 3, 0)
	initialDirty := node.dirty
	again := coalesceForest(&idx, slab, 5, 4, base, winner, 7, 0)
	if again != node {
		t.Fatal("replacement reached a different coalesced node")
	}
	if len(node.links) != 1 {
		t.Fatalf("replacement appended duplicate link: got %d links, want 1", len(node.links))
	}
	if node.minLinkScore != 7 {
		t.Fatalf("want refreshed min link score 7, got %d", node.minLinkScore)
	}
	if best := node.bestLink(); best == nil || best.score != 7 {
		t.Fatalf("want best link score 7 after replacement, got %v", best)
	}
	if node.dirty <= initialDirty {
		t.Fatalf("replacement dirty=%d, want > %d", node.dirty, initialDirty)
	}
}

func TestCoalesceForestMarksDirtyWhenPredecessorChanges(t *testing.T) {
	idx := newGSSForestIndex(0)
	slab := &gssForestNodeSlab{}
	prev := &gssForestNode{state: 1, dirty: 1}
	entry := newStackEntryNode(2, &Node{symbol: 7, startByte: 10, endByte: 20})

	top := coalesceForest(&idx, slab, 5, 20, prev, entry, 0, 0)
	initialDirty := top.dirty
	initialLinks := len(top.links)

	prev.dirty++
	again := coalesceForest(&idx, slab, 5, 20, prev, entry, 0, 0)
	if again != top {
		t.Fatal("same link reached a different coalesced node")
	}
	if len(top.links) != initialLinks {
		t.Fatalf("duplicate link appended: got %d links, want %d", len(top.links), initialLinks)
	}
	if top.dirty <= initialDirty {
		t.Fatalf("coalesced node dirty=%d, want > %d after predecessor changed", top.dirty, initialDirty)
	}
}

func TestCoalesceForestCapKeepsWiderGeneratedRepeatAux(t *testing.T) {
	const repeatSym Symbol = 3
	meta := []SymbolMetadata{
		{Name: "end"},
		{Name: "close", Visible: true},
		{Name: "wrapper", Visible: true, Named: true},
		{Name: "wrapper_repeat1", GeneratedRepeatAux: true},
	}
	idx := newGSSForestIndex(0)
	slab := &gssForestNodeSlab{}
	for i := 0; i < forestMaxLinksPerNode; i++ {
		prev := &gssForestNode{state: StateID(20 + i), byteOffset: uint32(90 + i)}
		entry := newStackEntryNode(10, &Node{symbol: repeatSym, startByte: uint32(90 + i), endByte: 100})
		coalesceForestWithMetadata(&idx, slab, meta, 10, 100, prev, entry, 0, 0)
	}

	closeCapablePrev := &gssForestNode{state: 99, byteOffset: 0}
	wide := newStackEntryNode(10, &Node{symbol: repeatSym, startByte: 0, endByte: 100})
	node := coalesceForestWithMetadata(&idx, slab, meta, 10, 100, closeCapablePrev, wide, 0, 0)

	if len(node.links) != forestMaxLinksPerNode {
		t.Fatalf("links = %d, want capped %d", len(node.links), forestMaxLinksPerNode)
	}
	for _, link := range node.links {
		if link.prev == closeCapablePrev {
			return
		}
	}
	t.Fatalf("wide generated-repeat aux link was dropped: links=%+v", node.links)
}

func TestForestCoalescePreCapKeepsGeneratedRepeatAuxCandidate(t *testing.T) {
	const repeatSym Symbol = 3
	meta := []SymbolMetadata{
		{Name: "end"},
		{Name: "close", Visible: true},
		{Name: "wrapper", Visible: true, Named: true},
		{Name: "wrapper_repeat1", GeneratedRepeatAux: true},
	}
	idx := newGSSForestIndex(0)
	slab := &gssForestNodeSlab{}
	for i := 0; i < forestMaxLinksPerNode; i++ {
		prev := &gssForestNode{state: StateID(20 + i), byteOffset: uint32(90 + i)}
		entry := newStackEntryNode(10, &Node{symbol: repeatSym, startByte: uint32(90 + i), endByte: 100})
		coalesceForestWithMetadata(&idx, slab, meta, 10, 100, prev, entry, 0, 0)
	}

	if forestCoalesceWouldDropForCap(&idx, meta, repeatSym, 10, 100, 0, 0) {
		t.Fatal("generated repeat aux candidate was pre-dropped before retention could compare span width")
	}
	if !forestCoalesceWouldDropForCap(&idx, meta, 2, 10, 100, 0, 0) {
		t.Fatal("ordinary equal-score candidate was not pre-dropped")
	}
}

func TestParseGLRForestTraceWindow(t *testing.T) {
	disabled := parseGLRForestTraceWindow("")
	if disabled.enabled {
		t.Fatal("empty trace window enabled diagnostics")
	}

	single := parseGLRForestTraceWindow("5594")
	if !single.enabled || single.start != 5594 || single.end != 5594 {
		t.Fatalf("single trace window = %+v, want enabled 5594..5594", single)
	}

	span := parseGLRForestTraceWindow("5600:5588")
	if !span.enabled || span.start != 5588 || span.end != 5600 {
		t.Fatalf("reversed trace window = %+v, want enabled 5588..5600", span)
	}

	invalid := parseGLRForestTraceWindow("not-a-window")
	if invalid.enabled {
		t.Fatalf("invalid trace window enabled diagnostics: %+v", invalid)
	}
}

func TestGSSForestIndexLookupCacheClearsOnReset(t *testing.T) {
	idx := newGSSForestIndex(0)
	key := gssForestKey{state: 7, byteOffset: 11}
	node := &gssForestNode{state: 7, byteOffset: 11}

	idx.set(key, node)
	if got := idx.lookup(key); got != node {
		t.Fatalf("cached lookup returned %p, want %p", got, node)
	}
	idx.reset()
	if got := idx.lookup(key); got != nil {
		t.Fatalf("lookup after reset returned stale node %p", got)
	}
}

func TestGSSForestNodeSlabReleaseClearsPointers(t *testing.T) {
	slab := &gssForestNodeSlab{}
	base := slab.alloc(1, 0, 0, 0)
	node := slab.alloc(2, 1, 0, 0)
	nodeLinkStart := slab.linkIdx - forestMaxLinksPerNode
	node.links = append(node.links, gssLink{
		prev:    base,
		subtree: stackEntry{state: 3},
	})

	if len(slab.nodeBatches) == 0 || len(slab.linkBatches) == 0 {
		t.Fatal("expected slab batches to be allocated")
	}
	if got := slab.linkBatches[0][nodeLinkStart].prev; got != base {
		t.Fatalf("test setup failed: link batch prev = %p, want %p", got, base)
	}
	slab.resetForRelease()
	if got := slab.nodeBatches[0][0].links; got != nil {
		t.Fatalf("node batch retained stale links slice: %v", got)
	}
	if got := slab.linkBatches[0][nodeLinkStart].prev; got != nil {
		t.Fatalf("link batch retained stale prev pointer: %p", got)
	}
}
