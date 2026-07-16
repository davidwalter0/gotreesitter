package gotreesitter

// normalizeTypeScriptRecoveredNestedGenericCall repairs blob-parity ts-1: a
// generic (type-argumented) call in an expression position whose type argument
// list is itself a nested generic — e.g.
//
//	response = await axios.get<ServiceResponse<CredentialCreation>>(path);
//	return    axios.post<OptionalDataServiceResponse<any>>(a, b);
//
// mis-parsed as a `<`/`>` comparison chain (a hasError binary_expression)
// instead of a call_expression(function, type_arguments, arguments).
//
// Root cause (shared GLR core, NOT cpp/ts-local): the `<`
// template-argument-list-open vs. less-than disambiguation is a GLR dynamic-
// precedence choice. gt resolves it correctly for the SAME construct in
// isolation, but mis-resolves it when accumulated GLR state from surrounding
// context (here: two identical calls in an if/else, or a preceding nested
// generic type annotation) biases the less-than reading. The oracle always
// prefers the call reading. The engine-level fix would touch the forest scoring
// every language shares (python/js/… at 100% parity), so this stays a
// post-parse repair.
//
// Strategy — RE-PARSE, don't hand-reconstruct. Because the identical span
// parses correctly in isolation (the bug is purely a context-sensitive GLR
// mis-resolution), the offending binary_expression is simply re-parsed on its
// own via the recovery parser and, when that yields a single clean
// call_expression covering the exact byte span, spliced back with an offset
// clone. It fires ONLY when the isolated re-parse is error-free AND a
// call_expression — a genuine comparison such as `a < b >> c` re-parses to a
// binary_expression (or stays errored) and is left untouched.
func normalizeTypeScriptRecoveredNestedGenericCall(root *Node, source []byte, parser *Parser, lang *Language) {
	if root == nil || parser == nil || parser.skipRecoveryReparse || lang == nil || root.ownerArena == nil || len(source) == 0 {
		return
	}
	if lang.Name != "typescript" && lang.Name != "tsx" {
		return
	}
	if !root.HasError() {
		return
	}
	binExprSym, ok := symbolByName(lang, "binary_expression")
	if !ok {
		return
	}
	callSym, ok := symbolByName(lang, "call_expression")
	if !ok {
		return
	}
	exprStmtSym, ok := symbolByName(lang, "expression_statement")
	if !ok {
		return
	}
	arena := root.ownerArena
	replaced := false
	walkResultTree(root, func(n *Node) {
		children := n.children
		if n.ownerArena != nil && n.childIndex <= finalChildSidecarIndexBase {
			children = resultDenseChildrenFallbackForMutation(n)
		}
		for i := 0; i < len(children); i++ {
			child := children[i]
			if child == nil || child.symbol != binExprSym || !child.HasError() {
				continue
			}
			if !typeScriptSpanLooksLikeGenericCall(source, child.startByte, child.endByte) {
				continue
			}
			repl := reparseTypeScriptGenericCallSpan(parser, source, child.startByte, child.endByte, callSym, exprStmtSym, arena, lang)
			if repl == nil {
				continue
			}
			children[i] = repl
			repl.parent = n
			repl.childIndex = int32(i)
			replaced = true
		}
	})
	if replaced {
		refreshTypeScriptCompatibilityHasErrorSubtree(root)
	}
}

// typeScriptSpanLooksLikeGenericCall is a cheap pre-filter: the trimmed span
// must open a `<` before its first `(` and close on `)`, the shape of a
// (possibly await-prefixed) generic call. The authoritative check is the
// isolated re-parse in reparseTypeScriptGenericCallSpan; this only avoids
// re-parsing binary_expressions that cannot possibly be generic calls.
func typeScriptSpanLooksLikeGenericCall(source []byte, start, end uint32) bool {
	if start >= end || end > uint32(len(source)) {
		return false
	}
	s := start
	for s < end && typeScriptIsSpaceByte(source[s]) {
		s++
	}
	e := end
	for e > s && typeScriptIsSpaceByte(source[e-1]) {
		e--
	}
	if e == s || source[e-1] != ')' {
		return false
	}
	sawAngle := false
	for i := s; i < e; i++ {
		switch source[i] {
		case '<':
			sawAngle = true
		case '(':
			return sawAngle
		}
	}
	return false
}

func typeScriptIsSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}

// reparseTypeScriptGenericCallSpan re-parses source[start:end) in isolation and
// returns an offset-cloned copy of the resulting call_expression when the
// re-parse is a single, error-free expression_statement whose expression is a
// call_expression spanning the entire span. Returns nil (leaving the tree
// untouched) in every other case.
func reparseTypeScriptGenericCallSpan(parser *Parser, source []byte, start, end uint32, callSym, exprStmtSym Symbol, arena *nodeArena, lang *Language) *Node {
	span := source[start:end]
	tree, err := parser.parseForRecovery(span)
	if err != nil || tree == nil {
		if tree != nil {
			tree.Release()
		}
		return nil
	}
	defer tree.Release()
	program := tree.RootNode()
	if program == nil || program.HasError() {
		return nil
	}
	// program must have exactly one named, non-extra child: an
	// expression_statement.
	var stmt *Node
	for i := 0; i < resultChildCount(program); i++ {
		c := resultChildAt(program, i)
		if c == nil || c.isExtra() || !c.isNamed() {
			continue
		}
		if stmt != nil {
			return nil
		}
		stmt = c
	}
	if stmt == nil || stmt.symbol != exprStmtSym {
		return nil
	}
	// the expression_statement's first named non-extra child is the expression.
	var expr *Node
	for i := 0; i < resultChildCount(stmt); i++ {
		c := resultChildAt(stmt, i)
		if c == nil || c.isExtra() || !c.isNamed() {
			continue
		}
		expr = c
		break
	}
	if expr == nil || expr.symbol != callSym || expr.HasError() {
		return nil
	}
	if expr.startByte != 0 || expr.endByte != uint32(len(span)) {
		return nil
	}
	offset := &cloneOffset{
		byteDelta: start,
		point:     advancePointByBytes(Point{}, source[:start]),
		baseRow:   program.startPoint.Row,
	}
	return cloneTreeNodesIntoArenaWithOffset(expr, arena, offset)
}

// normalizeTypeScriptRecoveredCurriedGenericCallDeclaration repairs the second
// TypeScript nested-generic-call sub-case — a curried generic call in a
// variable declaration, e.g. the zustand store idiom
//
//	export const useSession = create<SessionState>()(persist(cfg));
//
// Here the WHOLE declaration collapses into a statement-level ERROR node
//
//	ERROR[ export? const IDENT = binary_expression(create < SessionState) `>` `(`
//	       ERROR(`)`) parenthesized_expression((curry args)) ]  + empty_statement(`;`)
//
// because `create<SessionState>()` at the head of a curried call trips the same
// less-than mis-resolution (which even an isolated re-parse of the full RHS does
// not clear). The callee `create<SessionState>()` DOES re-parse cleanly in
// isolation, so it is re-parsed and re-composed with the trailing curry
// (lifted from the parenthesized_expression) into
// call_expression(function: call_expression(create, type_arguments, arguments),
// arguments: arguments(curry)), then wrapped back into the
// export_statement/lexical_declaration/variable_declarator the oracle produces.
func normalizeTypeScriptRecoveredCurriedGenericCallDeclaration(root *Node, source []byte, parser *Parser, lang *Language) {
	if root == nil || parser == nil || parser.skipRecoveryReparse || lang == nil || root.ownerArena == nil || len(source) == 0 {
		return
	}
	if lang.Name != "typescript" && lang.Name != "tsx" {
		return
	}
	if !root.HasError() {
		return
	}
	r, ok := newTypeScriptCurriedDeclReconstructor(source, parser, lang, root.ownerArena)
	if !ok {
		return
	}
	changed := false
	walkResultTree(root, func(n *Node) {
		children := n.children
		if n.ownerArena != nil && n.childIndex <= finalChildSidecarIndexBase {
			children = resultDenseChildrenFallbackForMutation(n)
		}
		out := make([]*Node, 0, len(children))
		hit := false
		for i := 0; i < len(children); i++ {
			child := children[i]
			if child == nil || child.symbol != errorSymbol {
				out = append(out, child)
				continue
			}
			// A following empty_statement carries the declaration's `;`.
			var semi *Node
			consumeNext := false
			if i+1 < len(children) {
				semi = r.trailingSemicolon(children[i+1])
				consumeNext = semi != nil
			}
			repl := r.rebuild(child, semi)
			if repl == nil {
				out = append(out, child)
				continue
			}
			out = append(out, repl)
			if consumeNext {
				i++ // consume the empty_statement sibling
			}
			hit = true
		}
		if hit {
			replaceNodeChildrenUnfielded(n, cloneNodeSliceIfArena(n.ownerArena, out))
			changed = true
		}
	})
	if changed {
		refreshTypeScriptCompatibilityHasErrorSubtree(root)
		// The parse left the program root ending at the reconstructed
		// statement (the trailing declaration was an ERROR, so the native
		// trailing-whitespace extension never ran); re-extend it over any
		// remaining trailing whitespace to match the oracle's root range.
		extendNodeToTrailingWhitespace(root, source)
	}
}

type typeScriptCurriedDeclReconstructor struct {
	src    []byte
	parser *Parser
	lang   *Language
	arena  *nodeArena

	exportStmtSym Symbol
	lexDeclSym    Symbol
	varDeclSym    Symbol
	callSym       Symbol
	argsSym       Symbol
	exprStmtSym   Symbol
	parenExprSym  Symbol
	emptyStmtSym  Symbol
	semiSym       Symbol

	exportStmtNamed bool
	lexDeclNamed    bool
	varDeclNamed    bool
	argsNamed       bool

	declarationFID FieldID
	kindFID        FieldID
	nameFID        FieldID
	valueFID       FieldID
	functionFID    FieldID
	argumentsFID   FieldID
}

func newTypeScriptCurriedDeclReconstructor(source []byte, parser *Parser, lang *Language, arena *nodeArena) (*typeScriptCurriedDeclReconstructor, bool) {
	if arena == nil {
		return nil, false
	}
	r := &typeScriptCurriedDeclReconstructor{src: source, parser: parser, lang: lang, arena: arena}
	syms := []struct {
		name string
		dst  *Symbol
	}{
		{"export_statement", &r.exportStmtSym},
		{"lexical_declaration", &r.lexDeclSym},
		{"variable_declarator", &r.varDeclSym},
		{"call_expression", &r.callSym},
		{"arguments", &r.argsSym},
		{"expression_statement", &r.exprStmtSym},
		{"parenthesized_expression", &r.parenExprSym},
		{"empty_statement", &r.emptyStmtSym},
		{";", &r.semiSym},
	}
	for _, s := range syms {
		sym, ok := symbolByName(lang, s.name)
		if !ok {
			return nil, false
		}
		*s.dst = sym
	}
	fields := []struct {
		name string
		dst  *FieldID
	}{
		{"declaration", &r.declarationFID},
		{"kind", &r.kindFID},
		{"name", &r.nameFID},
		{"value", &r.valueFID},
		{"function", &r.functionFID},
		{"arguments", &r.argumentsFID},
	}
	for _, f := range fields {
		fid, ok := lang.FieldByName(f.name)
		if !ok {
			return nil, false
		}
		*f.dst = fid
	}
	r.exportStmtNamed = symbolIsNamed(lang, r.exportStmtSym)
	r.lexDeclNamed = symbolIsNamed(lang, r.lexDeclSym)
	r.varDeclNamed = symbolIsNamed(lang, r.varDeclSym)
	r.argsNamed = symbolIsNamed(lang, r.argsSym)
	return r, true
}

// trailingSemicolon returns the `;` leaf of an empty_statement sibling, or nil.
func (r *typeScriptCurriedDeclReconstructor) trailingSemicolon(n *Node) *Node {
	if n == nil || n.symbol != r.emptyStmtSym || resultChildCount(n) != 1 {
		return nil
	}
	c := resultChildAt(n, 0)
	if c == nil || c.symbol != r.semiSym {
		return nil
	}
	return c
}

// reparseCurryAsArguments re-parses the curry `( … )` as the arguments of a
// synthetic one-character callee (`x( … )`) and returns an offset-clone of the
// resulting arguments node positioned back over the original curry span. This
// yields the oracle's argument_list even when the curry carries a trailing
// comma. Returns nil unless the re-parse is clean and the arguments span the
// whole curry.
func (r *typeScriptCurriedDeclReconstructor) reparseCurryAsArguments(curry *Node) *Node {
	if curry == nil || curry.startByte >= curry.endByte || curry.startPoint.Column == 0 {
		return nil
	}
	// The single-char prefix sits on the curry's first row; a multi-line curry
	// that begins on row 0 would misplace column offsets (offsetPoint applies
	// the column delta to every row when there is no row delta), so skip it.
	if curry.startPoint.Row == 0 && curry.endPoint.Row != 0 {
		return nil
	}
	span := r.src[curry.startByte:curry.endByte]
	synth := make([]byte, 0, len(span)+1)
	synth = append(synth, 'x')
	synth = append(synth, span...)
	tree, err := r.parser.parseForRecovery(synth)
	if err != nil || tree == nil {
		if tree != nil {
			tree.Release()
		}
		return nil
	}
	defer tree.Release()
	program := tree.RootNode()
	if program == nil || program.HasError() {
		return nil
	}
	var stmt *Node
	for i := 0; i < resultChildCount(program); i++ {
		c := resultChildAt(program, i)
		if c == nil || c.isExtra() || !c.isNamed() {
			continue
		}
		if stmt != nil {
			return nil
		}
		stmt = c
	}
	if stmt == nil || stmt.symbol != r.exprStmtSym {
		return nil
	}
	var call *Node
	for i := 0; i < resultChildCount(stmt); i++ {
		c := resultChildAt(stmt, i)
		if c == nil || c.isExtra() || !c.isNamed() {
			continue
		}
		call = c
		break
	}
	if call == nil || call.symbol != r.callSym {
		return nil
	}
	args := call.ChildByFieldName("arguments", r.lang)
	if args == nil || args.symbol != r.argsSym || args.HasError() {
		return nil
	}
	if args.startByte != 1 || args.endByte != uint32(len(synth)) {
		return nil
	}
	offset := &cloneOffset{
		byteDelta: curry.startByte - 1,
		point:     Point{Row: curry.startPoint.Row, Column: curry.startPoint.Column - 1},
		baseRow:   0,
	}
	return cloneTreeNodesIntoArenaWithOffset(args, r.arena, offset)
}

// rebuild attempts to turn a curried-generic-call-declaration ERROR node into
// the oracle's export_statement/lexical_declaration shape. Returns nil when the
// node does not match the signature.
func (r *typeScriptCurriedDeclReconstructor) rebuild(e *Node, semi *Node) *Node {
	cc := resultChildCount(e)
	if cc < 5 {
		return nil
	}
	// Identify: [export?] <kind> <identifier> `=` … parenthesized_expression(curry).
	idx := 0
	var exportTok *Node
	if c := resultChildAt(e, idx); c != nil && c.Type(r.lang) == "export" {
		exportTok = c
		idx++
	}
	kindTok := resultChildAt(e, idx)
	if kindTok == nil {
		return nil
	}
	switch kindTok.Type(r.lang) {
	case "const", "let", "var":
	default:
		return nil
	}
	varIdent := resultChildAt(e, idx+1)
	eq := resultChildAt(e, idx+2)
	if varIdent == nil || eq == nil || varIdent.Type(r.lang) != "identifier" || eq.Type(r.lang) != "=" {
		return nil
	}
	// The curry is the last child and must be a parenthesized_expression ending
	// where the declaration's value ends.
	curry := resultChildAt(e, cc-1)
	if curry == nil || curry.symbol != r.parenExprSym || curry.endByte != e.endByte {
		return nil
	}
	// Re-parse the callee `create<SessionState>()` from the byte after `=` up to
	// the curry; it must yield a clean call_expression spanning that span.
	calleeStart := eq.endByte
	for calleeStart < curry.startByte && typeScriptIsSpaceByte(r.src[calleeStart]) {
		calleeStart++
	}
	inner := reparseTypeScriptGenericCallSpan(r.parser, r.src, calleeStart, curry.startByte, r.callSym, r.exprStmtSym, r.arena, r.lang)
	if inner == nil {
		return nil
	}
	// Turn the curry `( … )` into the oracle's argument list by re-parsing it as
	// the arguments of a synthetic call. The real parser then handles argument
	// separators — including a trailing comma, which gt otherwise mis-flags as
	// an ERROR inside the mis-parsed parenthesized_expression — correctly.
	outerArgs := r.reparseCurryAsArguments(curry)
	if outerArgs == nil {
		return nil
	}

	outerCall := newParentNodeInArena(r.arena, r.callSym, symbolIsNamed(r.lang, r.callSym),
		cloneNodeSliceIfArena(r.arena, []*Node{inner, outerArgs}),
		[]FieldID{r.functionFID, r.argumentsFID}, 0)
	outerCall.startByte, outerCall.startPoint = inner.startByte, inner.startPoint
	outerCall.endByte, outerCall.endPoint = outerArgs.endByte, outerArgs.endPoint

	varDecl := newParentNodeInArena(r.arena, r.varDeclSym, r.varDeclNamed,
		cloneNodeSliceIfArena(r.arena, []*Node{varIdent, eq, outerCall}),
		[]FieldID{r.nameFID, 0, r.valueFID}, 0)
	varDecl.startByte, varDecl.startPoint = varIdent.startByte, varIdent.startPoint
	varDecl.endByte, varDecl.endPoint = outerCall.endByte, outerCall.endPoint

	lexChildren := []*Node{kindTok, varDecl}
	lexFields := []FieldID{r.kindFID, 0}
	if semi != nil {
		lexChildren = append(lexChildren, semi)
		lexFields = append(lexFields, 0)
	}
	lexDecl := newParentNodeInArena(r.arena, r.lexDeclSym, r.lexDeclNamed,
		cloneNodeSliceIfArena(r.arena, lexChildren), cloneFieldIDSliceInArena(r.arena, lexFields), 0)
	lexDecl.startByte, lexDecl.startPoint = kindTok.startByte, kindTok.startPoint
	last := lexChildren[len(lexChildren)-1]
	lexDecl.endByte, lexDecl.endPoint = last.endByte, last.endPoint

	if exportTok == nil {
		return lexDecl
	}
	exportStmt := newParentNodeInArena(r.arena, r.exportStmtSym, r.exportStmtNamed,
		cloneNodeSliceIfArena(r.arena, []*Node{exportTok, lexDecl}),
		[]FieldID{0, r.declarationFID}, 0)
	exportStmt.startByte, exportStmt.startPoint = exportTok.startByte, exportTok.startPoint
	exportStmt.endByte, exportStmt.endPoint = lexDecl.endByte, lexDecl.endPoint
	return exportStmt
}
