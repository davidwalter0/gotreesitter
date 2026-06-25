package gotreesitter

import "testing"

type recordingParserStateTokenSource struct {
	parserStates []StateID
	glrStates    [][]StateID
}

func (s *recordingParserStateTokenSource) Next() Token { return Token{} }

func (s *recordingParserStateTokenSource) SetParserState(state StateID) {
	s.parserStates = append(s.parserStates, state)
}

func (s *recordingParserStateTokenSource) SetGLRStates(states []StateID) {
	s.glrStates = append(s.glrStates, append([]StateID(nil), states...))
}

func TestUpdateCurrentRelexParserStateTokenSourceExcludesShiftedStacks(t *testing.T) {
	p := &Parser{}
	ts := &recordingParserStateTokenSource{}
	scratch := &parserScratch{}
	stacks := []glrStack{
		{entries: []stackEntry{{state: 10}}, shifted: true},
		{entries: []stackEntry{{state: 20}}},
		{entries: []stackEntry{{state: 30}}, shifted: true},
		{entries: []stackEntry{{state: 40}}},
	}

	if ok := p.updateCurrentRelexParserStateTokenSource(ts, stacks, scratch); !ok {
		t.Fatal("updateCurrentRelexParserStateTokenSource returned false, want true")
	}
	if got, want := len(ts.parserStates), 1; got != want {
		t.Fatalf("SetParserState calls = %d, want %d", got, want)
	}
	if got, want := ts.parserStates[0], StateID(20); got != want {
		t.Fatalf("parser state = %d, want first live unshifted state %d", got, want)
	}
	if got, want := len(ts.glrStates), 1; got != want {
		t.Fatalf("SetGLRStates calls = %d, want %d", got, want)
	}
	wantStates := []StateID{20, 40}
	if len(ts.glrStates[0]) != len(wantStates) {
		t.Fatalf("GLR states = %v, want %v", ts.glrStates[0], wantStates)
	}
	for i, want := range wantStates {
		if ts.glrStates[0][i] != want {
			t.Fatalf("GLR states = %v, want %v", ts.glrStates[0], wantStates)
		}
	}
}

func TestParseStacksShareStateAndByte(t *testing.T) {
	tests := []struct {
		name       string
		stacks     []glrStack
		state      StateID
		byteOffset uint32
		want       bool
	}{
		{
			name: "same state and byte",
			stacks: []glrStack{
				{entries: []stackEntry{{state: 10}}, byteOffset: 12},
				{entries: []stackEntry{{state: 10}}, byteOffset: 12},
			},
			state:      10,
			byteOffset: 12,
			want:       true,
		},
		{
			name: "same state mixed byte",
			stacks: []glrStack{
				{entries: []stackEntry{{state: 10}}, byteOffset: 12},
				{entries: []stackEntry{{state: 10}}, byteOffset: 16},
			},
			state:      10,
			byteOffset: 12,
			want:       false,
		},
		{
			name: "mixed state same byte",
			stacks: []glrStack{
				{entries: []stackEntry{{state: 10}}, byteOffset: 12},
				{entries: []stackEntry{{state: 11}}, byteOffset: 12},
			},
			state:      10,
			byteOffset: 12,
			want:       false,
		},
		{
			name: "dead stack ignored",
			stacks: []glrStack{
				{entries: []stackEntry{{state: 10}}, byteOffset: 12},
				{entries: []stackEntry{{state: 11}}, byteOffset: 16, dead: true},
			},
			state:      10,
			byteOffset: 12,
			want:       true,
		},
		{
			name: "shifted stack still vetoes",
			stacks: []glrStack{
				{entries: []stackEntry{{state: 10}}, byteOffset: 12},
				{entries: []stackEntry{{state: 10}}, byteOffset: 16, shifted: true},
			},
			state:      10,
			byteOffset: 12,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseStacksShareStateAndByte(tt.stacks, tt.state, tt.byteOffset)
			if got != tt.want {
				t.Fatalf("parseStacksShareStateAndByte() = %v, want %v", got, tt.want)
			}
		})
	}
}
