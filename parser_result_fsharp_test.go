package gotreesitter

import "testing"

func TestNormalizeFSharpDottedLongIdentifierOrOpWrapsLongIdentifier(t *testing.T) {
	source := []byte("config.TargetFramework")
	lang := &Language{
		Name:           "fsharp",
		SymbolNames:    []string{"EOF", "file", "long_identifier_or_op", "long_identifier", "identifier", ".", "dot_expression"},
		SymbolMetadata: []SymbolMetadata{{}, {Named: true}, {Named: true}, {Named: true}, {Named: true}, {}, {Named: true}},
	}
	arena := newNodeArena(arenaClassFull)
	identifier := func(s, e uint32) *Node {
		return newLeafNodeInArena(arena, 4, true, s, e, advancePointByBytes(Point{}, source[:s]), advancePointByBytes(Point{}, source[:e]))
	}
	dot := newLeafNodeInArena(arena, 5, false, 6, 7, Point{Column: 6}, Point{Column: 7})
	longIdentifierOrOp := newParentNodeInArena(arena, 2, true, []*Node{
		identifier(0, 6),
		dot,
		identifier(7, 22),
	}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{longIdentifierOrOp}, nil, 0)

	normalizeFSharpDottedLongIdentifierOrOps(root, lang)

	got := root.Child(0)
	if got.Type(lang) != "long_identifier_or_op" {
		t.Fatalf("root child type = %q, want long_identifier_or_op", got.Type(lang))
	}
	if got.ChildCount() != 1 {
		t.Fatalf("long_identifier_or_op child count = %d, want 1", got.ChildCount())
	}
	longIdentifier := got.Child(0)
	if longIdentifier.Type(lang) != "long_identifier" {
		t.Fatalf("wrapped child type = %q, want long_identifier", longIdentifier.Type(lang))
	}
	if longIdentifier.StartByte() != 0 || longIdentifier.EndByte() != 22 {
		t.Fatalf("long_identifier span = %d:%d, want 0:22", longIdentifier.StartByte(), longIdentifier.EndByte())
	}
	if longIdentifier.ChildCount() != 3 {
		t.Fatalf("long_identifier child count = %d, want 3", longIdentifier.ChildCount())
	}
	want := []string{"identifier", ".", "identifier"}
	for i, wantType := range want {
		if gotType := longIdentifier.Child(i).Type(lang); gotType != wantType {
			t.Fatalf("long_identifier child %d type = %q, want %q", i, gotType, wantType)
		}
	}
}

func TestNormalizeFSharpDottedLongIdentifierOrOpSplitsReceiverPrefix(t *testing.T) {
	source := []byte("proc.StandardOutput.ReadToEndAsync")
	lang := &Language{
		Name:           "fsharp",
		SymbolNames:    []string{"EOF", "file", "long_identifier_or_op", "long_identifier", "identifier", ".", "dot_expression"},
		SymbolMetadata: []SymbolMetadata{{}, {Named: true}, {Named: true}, {Named: true}, {Named: true}, {}, {Named: true}},
	}
	arena := newNodeArena(arenaClassFull)
	identifier := func(s, e uint32) *Node {
		return newLeafNodeInArena(arena, 4, true, s, e, advancePointByBytes(Point{}, source[:s]), advancePointByBytes(Point{}, source[:e]))
	}
	dot := func(s uint32) *Node {
		return newLeafNodeInArena(arena, 5, false, s, s+1, advancePointByBytes(Point{}, source[:s]), advancePointByBytes(Point{}, source[:s+1]))
	}
	receiver := newParentNodeInArena(arena, 2, true, []*Node{
		identifier(0, 4),
		dot(4),
		identifier(5, 19),
	}, nil, 0)
	method := newParentNodeInArena(arena, 2, true, []*Node{identifier(20, 34)}, nil, 0)
	dotExpression := newParentNodeInArena(arena, 6, true, []*Node{receiver, dot(19), method}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{dotExpression}, nil, 0)

	normalizeFSharpDottedLongIdentifierOrOps(root, lang)

	got := root.Child(0).Child(0)
	if got.Type(lang) != "long_identifier_or_op" {
		t.Fatalf("receiver type = %q, want long_identifier_or_op", got.Type(lang))
	}
	if got.ChildCount() != 3 {
		t.Fatalf("receiver child count = %d, want 3", got.ChildCount())
	}
	prefix := got.Child(0)
	if prefix.Type(lang) != "long_identifier" {
		t.Fatalf("receiver prefix type = %q, want long_identifier", prefix.Type(lang))
	}
	if prefix.StartByte() != 0 || prefix.EndByte() != 4 {
		t.Fatalf("receiver prefix span = %d:%d, want 0:4", prefix.StartByte(), prefix.EndByte())
	}
	if got.Child(1).Type(lang) != "." || got.Child(2).Type(lang) != "identifier" {
		t.Fatalf("receiver suffix children = %q, %q; want '.', identifier", got.Child(1).Type(lang), got.Child(2).Type(lang))
	}
	if got.Child(2).StartByte() != 5 || got.Child(2).EndByte() != 19 {
		t.Fatalf("receiver suffix span = %d:%d, want 5:19", got.Child(2).StartByte(), got.Child(2).EndByte())
	}
}

func TestNormalizeFSharpDottedLongIdentifierOrOpSplitsReceiverWrappedLongIdentifier(t *testing.T) {
	source := []byte("proc.StandardOutput.ReadToEndAsync")
	lang := &Language{
		Name:           "fsharp",
		SymbolNames:    []string{"EOF", "file", "long_identifier_or_op", "long_identifier", "identifier", ".", "dot_expression"},
		SymbolMetadata: []SymbolMetadata{{}, {Named: true}, {Named: true}, {Named: true}, {Named: true}, {}, {Named: true}},
	}
	arena := newNodeArena(arenaClassFull)
	identifier := func(s, e uint32) *Node {
		return newLeafNodeInArena(arena, 4, true, s, e, advancePointByBytes(Point{}, source[:s]), advancePointByBytes(Point{}, source[:e]))
	}
	dot := func(s uint32) *Node {
		return newLeafNodeInArena(arena, 5, false, s, s+1, advancePointByBytes(Point{}, source[:s]), advancePointByBytes(Point{}, source[:s+1]))
	}
	wrappedReceiver := newParentNodeInArena(arena, 3, true, []*Node{
		identifier(0, 4),
		dot(4),
		identifier(5, 19),
	}, nil, 0)
	receiver := newParentNodeInArena(arena, 2, true, []*Node{wrappedReceiver}, nil, 0)
	method := newParentNodeInArena(arena, 2, true, []*Node{identifier(20, 34)}, nil, 0)
	dotExpression := newParentNodeInArena(arena, 6, true, []*Node{receiver, dot(19), method}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{dotExpression}, nil, 0)

	normalizeFSharpDottedLongIdentifierOrOps(root, lang)

	got := root.Child(0).Child(0)
	if got.Type(lang) != "long_identifier_or_op" {
		t.Fatalf("receiver type = %q, want long_identifier_or_op", got.Type(lang))
	}
	if got.ChildCount() != 3 {
		t.Fatalf("receiver child count = %d, want 3", got.ChildCount())
	}
	prefix := got.Child(0)
	if prefix.Type(lang) != "long_identifier" || prefix.StartByte() != 0 || prefix.EndByte() != 4 {
		t.Fatalf("receiver prefix = %q [%d:%d], want long_identifier [0:4]", prefix.Type(lang), prefix.StartByte(), prefix.EndByte())
	}
	if got.Child(1).Type(lang) != "." || got.Child(2).Type(lang) != "identifier" {
		t.Fatalf("receiver suffix children = %q, %q; want '.', identifier", got.Child(1).Type(lang), got.Child(2).Type(lang))
	}
	if got.Child(2).StartByte() != 5 || got.Child(2).EndByte() != 19 {
		t.Fatalf("receiver suffix span = %d:%d, want 5:19", got.Child(2).StartByte(), got.Child(2).EndByte())
	}
}

func TestNormalizeFSharpDottedLongIdentifierOrOpSkipsLeadingExtraForReceiver(t *testing.T) {
	source := []byte("/*c*/proc.StandardOutput.ReadToEndAsync")
	lang := &Language{
		Name:           "fsharp",
		SymbolNames:    []string{"EOF", "file", "long_identifier_or_op", "long_identifier", "identifier", ".", "dot_expression", "comment"},
		SymbolMetadata: []SymbolMetadata{{}, {Named: true}, {Named: true}, {Named: true}, {Named: true}, {}, {Named: true}, {Named: true}},
	}
	arena := newNodeArena(arenaClassFull)
	identifier := func(s, e uint32) *Node {
		return newLeafNodeInArena(arena, 4, true, s, e, advancePointByBytes(Point{}, source[:s]), advancePointByBytes(Point{}, source[:e]))
	}
	dot := func(s uint32) *Node {
		return newLeafNodeInArena(arena, 5, false, s, s+1, advancePointByBytes(Point{}, source[:s]), advancePointByBytes(Point{}, source[:s+1]))
	}
	extra := newLeafNodeInArena(arena, 7, true, 0, 5, Point{}, Point{Column: 5})
	extra.setExtra(true)
	receiver := newParentNodeInArena(arena, 2, true, []*Node{
		identifier(5, 9),
		dot(9),
		identifier(10, 24),
	}, nil, 0)
	method := newParentNodeInArena(arena, 2, true, []*Node{identifier(25, 39)}, nil, 0)
	dotExpression := newParentNodeInArena(arena, 6, true, []*Node{extra, receiver, dot(24), method}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{dotExpression}, nil, 0)

	normalizeFSharpDottedLongIdentifierOrOps(root, lang)

	got := root.Child(0).Child(1)
	if got.Type(lang) != "long_identifier_or_op" {
		t.Fatalf("receiver type = %q, want long_identifier_or_op", got.Type(lang))
	}
	if got.ChildCount() != 3 {
		t.Fatalf("receiver child count = %d, want 3", got.ChildCount())
	}
	prefix := got.Child(0)
	if prefix.Type(lang) != "long_identifier" || prefix.StartByte() != 5 || prefix.EndByte() != 9 {
		t.Fatalf("receiver prefix = %q [%d:%d], want long_identifier [5:9]", prefix.Type(lang), prefix.StartByte(), prefix.EndByte())
	}
	if got.Child(1).Type(lang) != "." || got.Child(2).Type(lang) != "identifier" {
		t.Fatalf("receiver suffix children = %q, %q; want '.', identifier", got.Child(1).Type(lang), got.Child(2).Type(lang))
	}
}

func TestNormalizeFSharpDottedLongIdentifierOrOpPreservesReceiverPrefixSplit(t *testing.T) {
	source := []byte("proc.StandardOutput.ReadToEndAsync")
	lang := &Language{
		Name:           "fsharp",
		SymbolNames:    []string{"EOF", "file", "long_identifier_or_op", "long_identifier", "identifier", ".", "dot_expression"},
		SymbolMetadata: []SymbolMetadata{{}, {Named: true}, {Named: true}, {Named: true}, {Named: true}, {}, {Named: true}},
	}
	arena := newNodeArena(arenaClassFull)
	identifier := func(s, e uint32) *Node {
		return newLeafNodeInArena(arena, 4, true, s, e, advancePointByBytes(Point{}, source[:s]), advancePointByBytes(Point{}, source[:e]))
	}
	dot := func(s uint32) *Node {
		return newLeafNodeInArena(arena, 5, false, s, s+1, advancePointByBytes(Point{}, source[:s]), advancePointByBytes(Point{}, source[:s+1]))
	}
	prefix := newParentNodeInArena(arena, 3, true, []*Node{identifier(0, 4)}, nil, 0)
	receiver := newParentNodeInArena(arena, 2, true, []*Node{
		prefix,
		dot(4),
		identifier(5, 19),
	}, nil, 0)
	method := newParentNodeInArena(arena, 2, true, []*Node{identifier(20, 34)}, nil, 0)
	dotExpression := newParentNodeInArena(arena, 6, true, []*Node{receiver, dot(19), method}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{dotExpression}, nil, 0)

	normalizeFSharpCompatibility(root, source, lang)

	got := root.Child(0).Child(0)
	if got.ChildCount() != 3 {
		t.Fatalf("receiver child count = %d, want 3", got.ChildCount())
	}
	if got.Child(0).Type(lang) != "long_identifier" || got.Child(0).StartByte() != 0 || got.Child(0).EndByte() != 4 {
		t.Fatalf("receiver prefix = %q [%d:%d], want long_identifier [0:4]", got.Child(0).Type(lang), got.Child(0).StartByte(), got.Child(0).EndByte())
	}
	if got.Child(1).Type(lang) != "." || got.Child(2).Type(lang) != "identifier" {
		t.Fatalf("receiver suffix children = %q, %q; want '.', identifier", got.Child(1).Type(lang), got.Child(2).Type(lang))
	}
}

func TestNormalizeFSharpDottedLongIdentifierOrOpSkipsSimpleIdentifier(t *testing.T) {
	lang := &Language{
		Name:           "fsharp",
		SymbolNames:    []string{"EOF", "file", "long_identifier_or_op", "long_identifier", "identifier", ".", "dot_expression"},
		SymbolMetadata: []SymbolMetadata{{}, {Named: true}, {Named: true}, {Named: true}, {Named: true}, {}, {Named: true}},
	}
	arena := newNodeArena(arenaClassFull)
	identifier := newLeafNodeInArena(arena, 4, true, 0, 6, Point{}, Point{Column: 6})
	longIdentifierOrOp := newParentNodeInArena(arena, 2, true, []*Node{identifier}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{longIdentifierOrOp}, nil, 0)

	normalizeFSharpDottedLongIdentifierOrOps(root, lang)

	got := root.Child(0)
	if got.ChildCount() != 1 || got.Child(0).Type(lang) != "identifier" {
		t.Fatalf("simple long_identifier_or_op changed unexpectedly")
	}
}

func TestNormalizeFSharpFunctionDeclarationLeftSplitsCollapsedIdentifierPattern(t *testing.T) {
	source := []byte("let run (fsprojPath: string) (config: DtbConfig) =")
	lang := &Language{
		Name: "fsharp",
		SymbolNames: []string{
			"EOF",
			"file",
			"value_declaration_left",
			"function_declaration_left",
			"identifier_pattern",
			"identifier",
			"argument_patterns",
			"long_identifier_or_op",
			"paren_pattern",
			"(",
			"typed_pattern",
			")",
		},
		SymbolMetadata: []SymbolMetadata{
			{},
			{Named: true},
			{Named: true},
			{Named: true},
			{Named: true},
			{Named: true},
			{Named: true},
			{Named: true},
			{Named: true},
			{},
			{Named: true},
			{},
		},
	}
	arena := newNodeArena(arenaClassFull)
	end := uint32(48)
	leaf := func(sym Symbol, named bool, s, e uint32) *Node {
		return newLeafNodeInArena(arena, sym, named, s, e, advancePointByBytes(Point{}, source[:s]), advancePointByBytes(Point{}, source[:e]))
	}
	name := leaf(5, true, 4, 7)
	longIdentifierOrOp := newParentNodeInArena(arena, 7, true, []*Node{name}, nil, 0)
	firstArg := newParentNodeInArena(arena, 8, true, []*Node{
		leaf(9, false, 8, 9),
		leaf(10, true, 9, 27),
		leaf(11, false, 27, 28),
	}, nil, 0)
	secondArg := newParentNodeInArena(arena, 8, true, []*Node{
		leaf(9, false, 29, 30),
		leaf(10, true, 30, 47),
		leaf(11, false, 47, 48),
	}, nil, 0)
	pattern := newParentNodeInArena(arena, 4, true, []*Node{longIdentifierOrOp, firstArg, secondArg}, nil, 0)
	left := newParentNodeInArena(arena, 2, true, []*Node{pattern}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{left}, nil, 0)

	normalizeFSharpFunctionDeclarationLeft(root, source, lang)

	got := root.Child(0)
	if got == nil {
		t.Fatal("root child is nil")
	}
	if got.Type(lang) != "function_declaration_left" {
		t.Fatalf("left type = %q, want function_declaration_left", got.Type(lang))
	}
	if got.ChildCount() != 2 {
		t.Fatalf("left child count = %d, want 2", got.ChildCount())
	}
	gotName := got.Child(0)
	if gotName.Type(lang) != "identifier" {
		t.Fatalf("name type = %q, want identifier", gotName.Type(lang))
	}
	if gotName.StartByte() != 4 || gotName.EndByte() != 7 {
		t.Fatalf("name span = %d:%d, want 4:7", gotName.StartByte(), gotName.EndByte())
	}
	args := got.Child(1)
	if args.Type(lang) != "argument_patterns" {
		t.Fatalf("args type = %q, want argument_patterns", args.Type(lang))
	}
	if args.StartByte() != 8 || args.EndByte() != end {
		t.Fatalf("args span = %d:%d, want 8:%d", args.StartByte(), args.EndByte(), end)
	}
	if args.ChildCount() != 6 {
		t.Fatalf("args child count = %d, want 6", args.ChildCount())
	}
	wantArgChildren := []struct {
		typ        string
		start, end uint32
	}{
		{"(", 8, 9},
		{"typed_pattern", 9, 27},
		{")", 27, 28},
		{"(", 29, 30},
		{"typed_pattern", 30, 47},
		{")", 47, 48},
	}
	for i, want := range wantArgChildren {
		child := args.Child(i)
		if child.Type(lang) != want.typ || child.StartByte() != want.start || child.EndByte() != want.end {
			t.Fatalf("args child %d = %q [%d:%d], want %q [%d:%d]", i, child.Type(lang), child.StartByte(), child.EndByte(), want.typ, want.start, want.end)
		}
	}
}

func TestNormalizeFSharpFunctionDeclarationLeftClearsProductionID(t *testing.T) {
	source := []byte("let run (x) =")
	lang := &Language{
		Name: "fsharp",
		SymbolNames: []string{
			"EOF",
			"file",
			"value_declaration_left",
			"function_declaration_left",
			"identifier_pattern",
			"identifier",
			"argument_patterns",
			"long_identifier_or_op",
			"paren_pattern",
			"(",
			")",
		},
		SymbolMetadata: []SymbolMetadata{
			{},
			{Named: true},
			{Named: true},
			{Named: true},
			{Named: true},
			{Named: true},
			{Named: true},
			{Named: true},
			{Named: true},
			{},
			{},
		},
	}
	arena := newNodeArena(arenaClassFull)
	leaf := func(sym Symbol, named bool, s, e uint32) *Node {
		return newLeafNodeInArena(arena, sym, named, s, e, advancePointByBytes(Point{}, source[:s]), advancePointByBytes(Point{}, source[:e]))
	}
	name := leaf(5, true, 4, 7)
	longIdentifierOrOp := newParentNodeInArena(arena, 7, true, []*Node{name}, nil, 0)
	arg := newParentNodeInArena(arena, 8, true, []*Node{
		leaf(9, false, 8, 9),
		leaf(5, true, 9, 10),
		leaf(10, false, 10, 11),
	}, nil, 0)
	pattern := newParentNodeInArena(arena, 4, true, []*Node{longIdentifierOrOp, arg}, nil, 0)
	left := newParentNodeInArena(arena, 2, true, []*Node{pattern}, nil, 37)
	root := newParentNodeInArena(arena, 1, true, []*Node{left}, nil, 0)

	normalizeFSharpFunctionDeclarationLeft(root, source, lang)

	got := root.Child(0)
	if got.Type(lang) != "function_declaration_left" {
		t.Fatalf("left type = %q, want function_declaration_left", got.Type(lang))
	}
	if got.productionID != 0 {
		t.Fatalf("left productionID = %d, want 0", got.productionID)
	}
}

func TestNormalizeFSharpFunctionDeclarationLeftSkipsSimpleValuePattern(t *testing.T) {
	source := []byte("let value = 1")
	lang := &Language{
		Name: "fsharp",
		SymbolNames: []string{
			"EOF",
			"file",
			"value_declaration_left",
			"function_declaration_left",
			"identifier_pattern",
			"identifier",
			"argument_patterns",
			"long_identifier_or_op",
			"paren_pattern",
		},
		SymbolMetadata: []SymbolMetadata{
			{},
			{Named: true},
			{Named: true},
			{Named: true},
			{Named: true},
			{Named: true},
			{Named: true},
			{Named: true},
			{Named: true},
		},
	}
	arena := newNodeArena(arenaClassFull)
	pattern := newLeafNodeInArena(arena, 4, true, 4, 9, Point{Column: 4}, Point{Column: 9})
	left := newParentNodeInArena(arena, 2, true, []*Node{pattern}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{left}, nil, 0)

	normalizeFSharpFunctionDeclarationLeft(root, source, lang)

	got := root.Child(0)
	if got.Type(lang) != "value_declaration_left" {
		t.Fatalf("left type = %q, want value_declaration_left", got.Type(lang))
	}
	if got.ChildCount() != 1 || got.Child(0).Type(lang) != "identifier_pattern" {
		t.Fatalf("left children changed unexpectedly")
	}
}

func TestNormalizeFSharpSimpleLongIdentifierUnwrapsIdentifier(t *testing.T) {
	lang := &Language{
		Name:           "fsharp",
		SymbolNames:    []string{"EOF", "file", "long_identifier", "identifier", "long_identifier_or_op"},
		SymbolMetadata: []SymbolMetadata{{}, {Named: true}, {Named: true}, {Named: true}, {Named: true}},
	}
	arena := newNodeArena(arenaClassFull)
	identifier := newLeafNodeInArena(arena, 3, true, 4, 17, Point{Column: 4}, Point{Column: 17})
	longIdentifier := newParentNodeInArena(arena, 2, true, []*Node{identifier}, nil, 0)
	longIdentifierOrOp := newParentNodeInArena(arena, 4, true, []*Node{longIdentifier}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{longIdentifierOrOp}, nil, 0)

	normalizeFSharpSimpleLongIdentifiers(root, lang)

	got := root.Child(0).Child(0)
	if got == nil {
		t.Fatal("root child is nil")
	}
	if got.Type(lang) != "identifier" {
		t.Fatalf("root child type = %q, want identifier", got.Type(lang))
	}
	if got.ChildCount() != 0 {
		t.Fatalf("identifier child count = %d, want 0", got.ChildCount())
	}
	if got.StartByte() != 4 || got.EndByte() != 17 {
		t.Fatalf("identifier span = %d:%d, want 4:17", got.StartByte(), got.EndByte())
	}
}

func TestNormalizeFSharpSimpleLongIdentifierSkipsErrorRoot(t *testing.T) {
	lang := &Language{
		Name:           "fsharp",
		SymbolNames:    []string{"EOF", "file", "long_identifier", "identifier", "long_identifier_or_op"},
		SymbolMetadata: []SymbolMetadata{{}, {Named: true}, {Named: true}, {Named: true}, {Named: true}},
	}
	arena := newNodeArena(arenaClassFull)
	identifier := newLeafNodeInArena(arena, 3, true, 4, 17, Point{Column: 4}, Point{Column: 17})
	longIdentifier := newParentNodeInArena(arena, 2, true, []*Node{identifier}, nil, 0)
	longIdentifierOrOp := newParentNodeInArena(arena, 4, true, []*Node{longIdentifier}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{longIdentifierOrOp}, nil, 0)
	root.setHasError(true)

	normalizeFSharpSimpleLongIdentifiers(root, lang)

	got := root.Child(0).Child(0)
	if got == nil {
		t.Fatal("root child is nil")
	}
	if got.Type(lang) != "long_identifier" {
		t.Fatalf("root child type = %q, want long_identifier", got.Type(lang))
	}
}

func TestNormalizeFSharpSimpleLongIdentifierSkipsOtherContexts(t *testing.T) {
	lang := &Language{
		Name:           "fsharp",
		SymbolNames:    []string{"EOF", "file", "long_identifier", "identifier", "import_decl"},
		SymbolMetadata: []SymbolMetadata{{}, {Named: true}, {Named: true}, {Named: true}, {Named: true}},
	}
	arena := newNodeArena(arenaClassFull)
	identifier := newLeafNodeInArena(arena, 3, true, 4, 10, Point{Column: 4}, Point{Column: 10})
	longIdentifier := newParentNodeInArena(arena, 2, true, []*Node{identifier}, nil, 0)
	importDecl := newParentNodeInArena(arena, 4, true, []*Node{longIdentifier}, nil, 0)
	root := newParentNodeInArena(arena, 1, true, []*Node{importDecl}, nil, 0)

	normalizeFSharpSimpleLongIdentifiers(root, lang)

	got := root.Child(0).Child(0)
	if got == nil {
		t.Fatal("import child is nil")
	}
	if got.Type(lang) != "long_identifier" {
		t.Fatalf("import child type = %q, want long_identifier", got.Type(lang))
	}
}
