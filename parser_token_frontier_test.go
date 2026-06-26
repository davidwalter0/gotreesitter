package gotreesitter

import "testing"

func TestGLRTokenFrontierDispatchKeepsNarrowCandidateVersion(t *testing.T) {
	t.Setenv("GOT_GLR_TOKEN_FRONTIER_DISPATCH", "1")
	ResetParseEnvConfigCacheForTests()

	lang := buildTokenFrontierDispatchTestLanguage()
	parser := NewParser(lang)
	tree, err := parser.Parse([]byte("x>>"))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if tree == nil {
		t.Fatal("Parse returned nil tree")
	}
	if reason := tree.ParseStopReason(); reason != ParseStopAccepted {
		t.Fatalf("ParseStopReason = %s, want accepted", reason)
	}
	if root := tree.RootNode(); root == nil || root.HasError() {
		t.Fatalf("root = %v, want non-error parse", root)
	}
}

func buildTokenFrontierDispatchTestLanguage() *Language {
	return &Language{
		Name:        "token_frontier_dispatch_test",
		SymbolCount: 4,
		TokenCount:  4,
		StateCount:  6,
		SymbolNames: []string{"end", "x", ">", ">>"},
		SymbolMetadata: []SymbolMetadata{
			{Name: "end"},
			{Name: "x", Visible: true},
			{Name: ">", Visible: true},
			{Name: ">>", Visible: true},
		},
		ParseActions: []ParseActionEntry{
			{},
			{Actions: []ParseAction{
				{Type: ParseActionShift, State: 1},
				{Type: ParseActionShift, State: 2},
			}},
			{Actions: []ParseAction{{Type: ParseActionShift, State: 3}}},
			{Actions: []ParseAction{{Type: ParseActionShift, State: 4}}},
			{Actions: []ParseAction{{Type: ParseActionShift, State: 5}}},
			{Actions: []ParseAction{{Type: ParseActionAccept}}},
		},
		ParseTable: [][]uint16{
			{0, 1, 0, 0},
			{0, 0, 2, 0},
			{0, 0, 0, 3},
			{0, 0, 4, 0},
			{0, 0, 0, 0},
			{5, 0, 0, 0},
		},
		LexModes: []LexMode{
			{LexState: 0},
			{LexState: 2},
			{LexState: 4},
			{LexState: 2},
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
			{
				AcceptToken: 1,
				Default:     -1,
				EOF:         -1,
			},
			{
				Default: -1,
				EOF:     -1,
				Transitions: []LexTransition{
					{Lo: '>', Hi: '>', NextState: 3},
				},
			},
			{
				AcceptToken: 2,
				Default:     -1,
				EOF:         -1,
			},
			{
				Default: -1,
				EOF:     -1,
				Transitions: []LexTransition{
					{Lo: '>', Hi: '>', NextState: 5},
				},
			},
			{
				Default: -1,
				EOF:     -1,
				Transitions: []LexTransition{
					{Lo: '>', Hi: '>', NextState: 6},
				},
			},
			{
				AcceptToken: 3,
				Default:     -1,
				EOF:         -1,
			},
		},
	}
}
