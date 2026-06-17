package gotreesitter

import "testing"

type faithfulShiftGuardTokenSource struct {
	source       []byte
	tokens       []Token
	idx          int
	parserState  StateID
	glrStateSeen bool
}

func (ts *faithfulShiftGuardTokenSource) Next() Token {
	if ts.idx >= len(ts.tokens) {
		return Token{StartByte: uint32(len(ts.source)), EndByte: uint32(len(ts.source))}
	}
	tok := ts.tokens[ts.idx]
	ts.idx++
	return tok
}

func (ts *faithfulShiftGuardTokenSource) SkipToByte(offset uint32) Token {
	ts.idx = 0
	for ts.idx < len(ts.tokens) && ts.tokens[ts.idx].StartByte < offset {
		ts.idx++
	}
	return ts.Next()
}

func (ts *faithfulShiftGuardTokenSource) SkipToByteWithPoint(offset uint32, _ Point) Token {
	return ts.SkipToByte(offset)
}

func (ts *faithfulShiftGuardTokenSource) SetParserState(state StateID) {
	ts.parserState = state
}

func (ts *faithfulShiftGuardTokenSource) SetGLRStates(states []StateID) {
	ts.glrStateSeen = len(states) > 0
}

type nonSkippableFaithfulShiftGuardTokenSource struct {
	source []byte
	tokens []Token
	idx    int
}

func (ts *nonSkippableFaithfulShiftGuardTokenSource) Next() Token {
	if ts.idx >= len(ts.tokens) {
		return Token{StartByte: uint32(len(ts.source)), EndByte: uint32(len(ts.source))}
	}
	tok := ts.tokens[ts.idx]
	ts.idx++
	return tok
}

func TestFaithfulShiftGuardAllowsWhitespaceGap(t *testing.T) {
	prev := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	defer func() { glrFaithfulCapOneMerge = prev }()

	source := []byte("a   b")
	lookahead := Token{Symbol: 2, StartByte: 4, EndByte: 5}
	stack := &glrStack{
		entries:    []stackEntry{{state: 17}},
		byteOffset: 1,
	}
	parser := &Parser{
		language:    &Language{},
		parseSource: source,
		reparseFactory: func(src []byte) (TokenSource, error) {
			return &faithfulShiftGuardTokenSource{
				source: src,
				tokens: []Token{
					{Symbol: 2, StartByte: 4, EndByte: 5},
				},
			}, nil
		},
	}

	if !parser.allowFaithfulShiftAcrossGap(stack, stack.top().state, lookahead, nil) {
		t.Fatal("faithful shift guard rejected whitespace-only gap")
	}
}

func TestFaithfulShiftGuardRejectsRealTokenGap(t *testing.T) {
	prev := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	defer func() { glrFaithfulCapOneMerge = prev }()

	source := []byte("a, b")
	lookahead := Token{Symbol: 2, StartByte: 3, EndByte: 4}
	stack := &glrStack{
		entries:    []stackEntry{{state: 23}},
		byteOffset: 1,
	}
	var peek *faithfulShiftGuardTokenSource
	parser := &Parser{
		language:    &Language{},
		parseSource: source,
		reparseFactory: func(src []byte) (TokenSource, error) {
			peek = &faithfulShiftGuardTokenSource{
				source: src,
				tokens: []Token{
					{Symbol: 3, StartByte: 1, EndByte: 2},
					{Symbol: 2, StartByte: 3, EndByte: 4},
				},
			}
			return peek, nil
		},
	}

	if parser.allowFaithfulShiftAcrossGap(stack, stack.top().state, lookahead, nil) {
		t.Fatal("faithful shift guard allowed shift across real token gap")
	}
	if peek == nil {
		t.Fatal("faithful shift guard did not build a fresh token source")
	}
	if peek.parserState != 23 {
		t.Fatalf("fresh token source parser state = %d, want 23", peek.parserState)
	}
	if peek.glrStateSeen {
		t.Fatal("fresh token source kept GLR state union; want cleared states")
	}
}

func TestFaithfulShiftGuardFailsOpenWithoutFreshTokenSource(t *testing.T) {
	prev := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	defer func() { glrFaithfulCapOneMerge = prev }()

	parser := &Parser{language: &Language{}, parseSource: []byte("a,b")}
	stack := &glrStack{
		entries:    []stackEntry{{state: 3}},
		byteOffset: 1,
	}
	lookahead := Token{Symbol: 2, StartByte: 2, EndByte: 3}
	if !parser.allowFaithfulShiftAcrossGap(stack, stack.top().state, lookahead, nil) {
		t.Fatal("faithful shift guard should fail open without a fresh token source")
	}
}

func TestFaithfulShiftGuardFailsOpenWithNonSkippableFreshTokenSource(t *testing.T) {
	prev := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	defer func() { glrFaithfulCapOneMerge = prev }()

	source := []byte("a, b")
	lookahead := Token{Symbol: 2, StartByte: 3, EndByte: 4}
	stack := &glrStack{
		entries:    []stackEntry{{state: 29}},
		byteOffset: 1,
	}
	parser := &Parser{
		language:    &Language{},
		parseSource: source,
		reparseFactory: func(src []byte) (TokenSource, error) {
			return &nonSkippableFaithfulShiftGuardTokenSource{
				source: src,
				tokens: []Token{
					{Symbol: 3, StartByte: 1, EndByte: 2},
					{Symbol: 2, StartByte: 3, EndByte: 4},
				},
			}, nil
		},
	}

	if !parser.allowFaithfulShiftAcrossGap(stack, stack.top().state, lookahead, nil) {
		t.Fatal("faithful shift guard should fail open for non-skippable fresh token source")
	}
}

func TestFaithfulShiftGuardFailsOpenWhenExternalCheckpointMissing(t *testing.T) {
	prev := glrFaithfulCapOneMerge
	glrFaithfulCapOneMerge = true
	defer func() { glrFaithfulCapOneMerge = prev }()

	lang := &Language{Name: "python", ExternalScanner: byteStateExternalScanner{}}
	source := []byte("a,b")
	arena := &nodeArena{}
	leaf := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	stack := &glrStack{
		entries:    []stackEntry{{state: 1}, newStackEntryNode(7, leaf)},
		byteOffset: 1,
	}
	parser := &Parser{
		language:    lang,
		parseSource: source,
		reparseFactory: func(src []byte) (TokenSource, error) {
			return acquireDFATokenSource(NewLexer(nil, src), lang, nil, nil, nil), nil
		},
	}

	lookahead := Token{Symbol: 2, StartByte: 2, EndByte: 3}
	if !parser.allowFaithfulShiftAcrossGap(stack, stack.top().state, lookahead, arena) {
		t.Fatal("faithful shift guard should fail open when external checkpoint is missing")
	}
}

func TestFaithfulShiftGuardRestoresExternalCheckpointEndSnapshot(t *testing.T) {
	lang := &Language{Name: "python", ExternalScanner: byteStateExternalScanner{}}
	dts := acquireDFATokenSource(NewLexer(nil, nil), lang, nil, nil, nil)
	defer dts.Close()

	arena := &nodeArena{}
	leaf := newLeafNodeInArena(arena, 1, true, 0, 1, Point{}, Point{Column: 1})
	if !arena.recordExternalScannerLeafCheckpoint(leaf, []byte{3}, []byte{7}) {
		t.Fatal("failed to record external scanner checkpoint")
	}
	stack := &glrStack{
		entries:    []stackEntry{{state: 1}, newStackEntryNode(7, leaf)},
		byteOffset: 1,
	}

	*dts.externalPayload.(*byte) = 42
	if !restoreFaithfulShiftGuardExternalScannerCheckpoint(dts, arena, stack) {
		t.Fatal("restoreFaithfulShiftGuardExternalScannerCheckpoint returned false")
	}
	if got, want := *dts.externalPayload.(*byte), byte(7); got != want {
		t.Fatalf("restored external scanner state = %d, want %d", got, want)
	}
}
