package gotreesitter

// normalizeCSizeofCastMergedSignedLiteral repairs a companion shape of the
// sizeof/cast ambiguity that normalizeCSizeofUnknownTypeIdentifiers (above)
// already handles for the "clean" `sizeof(unknown_identifier)` case.
//
// Root cause (parse-table layer, not fixable here): tree-sitter-c's
// number_literal token optionally swallows a leading `+`/`-` sign
// (`token(seq(optional(/[-\+]/), ...))`), so `+1` written with no
// intervening whitespace lexes as a *single* signed number_literal token
// rather than a `+` operator followed by `1`. When that signed literal
// immediately follows `sizeof(IDENT)` with no space (`sizeof(a)+1`), gt's
// merged parse-table state resolves the ambiguity the same way it does for
// `sizeof(TYPE)+1` with a real type keyword: `IDENT` is kept as a
// type_descriptor's type_identifier and the whole thing becomes
// `sizeof_expression(value=cast_expression(type=type_descriptor(IDENT),
// value=<signed number_literal>))` — i.e. `sizeof((IDENT)(+1))`. That
// reading is only correct when IDENT is a real keyword type (verified
// against the oracle: `sizeof(int)+1` — a real primitive keyword —
// legitimately parses as a cast on both sides, and is left untouched by this
// pass). tree-sitter-c has no semantic scope tracking, so a plain identifier
// never keeps the cast reading here regardless of whether it happens to name
// a real local typedef: `sizeof(myint)+1` resolves identically to
// `sizeof(unknownName)+1` against the pinned oracle (empirically verified —
// unlike normalizeCSizeofUnknownTypeIdentifiers above, this pass
// deliberately does *not* special-case known local typedef names, since
// doing so would diverge from the oracle for this signature). The merged
// sign detaches back into a standalone `+`/`-` binary operator against a
// plain (unsigned) number_literal —
// `binary_expression(sizeof_expression(parenthesized_expression(IDENT)),
// +/-, number_literal)`.
//
// A further complication (why the earlier c-fixer pass deferred this):
// when the fixed sizeof_expression is itself the right-hand operand of an
// enclosing same-precedence (`+`/`-`) binary_expression — the
// `sizeof(a)+sizeof(b)+1` chain — a naive per-node fix produces a
// *right*-associative tree (`sizeof(a) + (sizeof(b) + 1)`), but tree-sitter
// (and this grammar's own left-associative chain construction elsewhere)
// builds `(sizeof(a) + sizeof(b)) + 1`. This pass walks top-down (parent
// before child) specifically so the enclosing binary_expression is always
// visited before its buggy right-hand sizeof_expression child, letting it
// detect that shape directly and perform the equivalent single-step
// rotation in place: the enclosing binary_expression is retagged in place
// to become the *outer* node of the corrected left-associative chain (its
// own left/operator survive unchanged, only its right operand — the buggy
// sizeof_expression — is replaced by the rotated structure), keeping the
// object identity so no grandparent needs to change. Chains of more than
// two sizeof terms are unaffected by this rotation (only the immediate
// parent is inspected), but since the bug only ever originates at the
// single rightmost element of a chain (the only position that can be
// textually adjacent to the trailing signed literal), one rotation is
// always sufficient regardless of chain length — the elements to the left
// were already correctly left-associated before this pass runs.
//
// (Node.Parent() is not used to find the enclosing binary_expression: parent
// links are wired lazily on first read and are not reliably available yet
// during this mid-normalization pass, so the rotation is detected by
// walking top-down instead of looking upward from the sizeof_expression.)
//
// Detection is conservative: only fires when (a) the sizeof's argument is a
// *lone* type_identifier (never primitive_type — a real type keyword's cast
// reading is correct and must not be touched), and (b) the cast's value is a
// number_literal whose first byte is literally '+' or '-'.
func normalizeCSizeofCastMergedSignedLiteral(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "c" || len(source) == 0 {
		return
	}
	syms, ok := cSizeofSignedTrailingSymbolsForLanguage(lang)
	if !ok {
		return
	}
	walkCSizeofCastMergedSignedLiteral(root, source, lang, syms)
}

type cSizeofSignedTrailingSymbols struct {
	sizeofExpr     Symbol
	castExpr       Symbol
	typeDescriptor Symbol
	typeIdentifier Symbol
	identifier     Symbol
	parenthesized  Symbol
	binaryExpr     Symbol
	numberLiteral  Symbol
	plusSym        Symbol
	minusSym       Symbol

	valueFID    FieldID
	typeFID     FieldID
	leftFID     FieldID
	operatorFID FieldID
	rightFID    FieldID

	identifierNamed    bool
	parenthesizedNamed bool
	binaryExprNamed    bool
	numberLiteralNamed bool
}

func cSizeofSignedTrailingSymbolsForLanguage(lang *Language) (cSizeofSignedTrailingSymbols, bool) {
	var s cSizeofSignedTrailingSymbols
	var ok bool
	if s.sizeofExpr, ok = symbolByName(lang, "sizeof_expression"); !ok {
		return s, false
	}
	if s.castExpr, ok = symbolByName(lang, "cast_expression"); !ok {
		return s, false
	}
	if s.typeDescriptor, ok = symbolByName(lang, "type_descriptor"); !ok {
		return s, false
	}
	if s.typeIdentifier, ok = symbolByName(lang, "type_identifier"); !ok {
		return s, false
	}
	if s.identifier, ok = symbolByName(lang, "identifier"); !ok {
		return s, false
	}
	if s.parenthesized, ok = symbolByName(lang, "parenthesized_expression"); !ok {
		return s, false
	}
	if s.binaryExpr, ok = symbolByName(lang, "binary_expression"); !ok {
		return s, false
	}
	if s.numberLiteral, ok = symbolByName(lang, "number_literal"); !ok {
		return s, false
	}
	if s.plusSym, ok = symbolByName(lang, "+"); !ok {
		return s, false
	}
	if s.minusSym, ok = symbolByName(lang, "-"); !ok {
		return s, false
	}
	if s.valueFID, ok = lang.FieldByName("value"); !ok {
		return s, false
	}
	if s.typeFID, ok = lang.FieldByName("type"); !ok {
		return s, false
	}
	if s.leftFID, ok = lang.FieldByName("left"); !ok {
		return s, false
	}
	if s.operatorFID, ok = lang.FieldByName("operator"); !ok {
		return s, false
	}
	if s.rightFID, ok = lang.FieldByName("right"); !ok {
		return s, false
	}
	s.identifierNamed = symbolIsNamed(lang, s.identifier)
	s.parenthesizedNamed = symbolIsNamed(lang, s.parenthesized)
	s.binaryExprNamed = symbolIsNamed(lang, s.binaryExpr)
	s.numberLiteralNamed = symbolIsNamed(lang, s.numberLiteral)
	return s, true
}

// cSizeofSignedLiteralFix carries the pieces extracted from a buggy
// `sizeof_expression(cast_expression(...))` node, built but not yet spliced
// into the tree.
type cSizeofSignedLiteralFix struct {
	fixedSizeof  *Node
	operatorLeaf *Node
	shrunkNumber *Node
}

// walkCSizeofCastMergedSignedLiteral walks top-down (parent before child) so
// that an enclosing binary_expression is always inspected before its own
// right-hand sizeof_expression child, letting the chain-rotation case (see
// the package-level doc comment) be detected directly from the parent
// without relying on Node.Parent().
func walkCSizeofCastMergedSignedLiteral(n *Node, source []byte, lang *Language, syms cSizeofSignedTrailingSymbols) {
	if n == nil {
		return
	}
	childCount := resultChildCount(n)

	if cResultSymbolMatches(lang, n, syms.binaryExpr) && childCount == 3 {
		left := resultChildAt(n, 0)
		operator := resultChildAt(n, 1)
		right := resultChildAt(n, 2)
		if left != nil && operator != nil && right != nil {
			opText := operator.Text(source)
			if opText == "+" || opText == "-" {
				if fix, ok := extractCSizeofCastMergedSignedLiteral(right, source, lang, syms); ok {
					arena := n.ownerArena
					// The left subtree may itself contain further,
					// independent instances of this bug (e.g. nested
					// parenthesized sizeof arithmetic) — fix it first.
					walkCSizeofCastMergedSignedLiteral(left, source, lang, syms)
					rotatedInner := newParentNodeInArena(arena, syms.binaryExpr, syms.binaryExprNamed,
						cloneNodeSliceIfArena(arena, []*Node{left, operator, fix.fixedSizeof}),
						cloneFieldIDSliceInArena(arena, []FieldID{syms.leftFID, syms.operatorFID, syms.rightFID}),
						0)
					setCRewriteChildren(n, syms.binaryExpr, syms.binaryExprNamed,
						cloneNodeSliceIfArena(arena, []*Node{rotatedInner, fix.operatorLeaf, fix.shrunkNumber}),
						cloneFieldIDSliceInArena(arena, []FieldID{syms.leftFID, syms.operatorFID, syms.rightFID}),
						[]int{0, 1, 2})
					return
				}
			}
		}
	}

	if cResultSymbolMatches(lang, n, syms.sizeofExpr) {
		if fix, ok := extractCSizeofCastMergedSignedLiteral(n, source, lang, syms); ok {
			arena := n.ownerArena
			setCRewriteChildren(n, syms.binaryExpr, syms.binaryExprNamed,
				cloneNodeSliceIfArena(arena, []*Node{fix.fixedSizeof, fix.operatorLeaf, fix.shrunkNumber}),
				cloneFieldIDSliceInArena(arena, []FieldID{syms.leftFID, syms.operatorFID, syms.rightFID}),
				[]int{0, 1, 2})
			return
		}
	}

	for i := 0; i < childCount; i++ {
		walkCSizeofCastMergedSignedLiteral(resultChildAt(n, i), source, lang, syms)
	}
}

// extractCSizeofCastMergedSignedLiteral checks whether n is a
// `sizeof_expression(cast_expression(type_descriptor(type_identifier),
// <signed number_literal>))` matching the buggy signature and, if so, builds
// (without mutating anything) the corrected inner sizeof_expression plus the
// detached operator/number-literal leaves. Returns ok=false for any other
// shape (most importantly a real primitive_type cast), in which case the
// caller must leave the node untouched.
func extractCSizeofCastMergedSignedLiteral(n *Node, source []byte, lang *Language, syms cSizeofSignedTrailingSymbols) (cSizeofSignedLiteralFix, bool) {
	var fix cSizeofSignedLiteralFix
	if n == nil || !cResultSymbolMatches(lang, n, syms.sizeofExpr) || resultChildCount(n) != 2 {
		return fix, false
	}
	sizeofKeyword := resultChildAt(n, 0)
	castExpr := resultChildAt(n, 1)
	if sizeofKeyword == nil || castExpr == nil || !cResultSymbolMatches(lang, castExpr, syms.castExpr) || resultChildCount(castExpr) != 4 {
		return fix, false
	}
	openParen := resultChildAt(castExpr, 0)
	typeDescriptor := resultChildAt(castExpr, 1)
	closeParen := resultChildAt(castExpr, 2)
	numLit := resultChildAt(castExpr, 3)
	if openParen == nil || typeDescriptor == nil || closeParen == nil || numLit == nil {
		return fix, false
	}
	if !cResultSymbolMatches(lang, typeDescriptor, syms.typeDescriptor) || resultChildCount(typeDescriptor) != 1 {
		return fix, false
	}
	typeIdent := resultChildAt(typeDescriptor, 0)
	if typeIdent == nil || !cResultSymbolMatches(lang, typeIdent, syms.typeIdentifier) {
		// Deliberately excludes primitive_type: a real type keyword's cast
		// reading is the oracle's own correct behavior (verified against
		// `sizeof(int)+1`) and must not be touched.
		return fix, false
	}
	if !cResultSymbolMatches(lang, numLit, syms.numberLiteral) {
		return fix, false
	}
	text := numLit.Text(source)
	if len(text) < 2 || (text[0] != '+' && text[0] != '-') {
		return fix, false
	}

	arena := n.ownerArena
	sign := text[0]
	signSym := syms.plusSym
	if sign == '-' {
		signSym = syms.minusSym
	}
	signEndByte := numLit.startByte + 1
	signEndPoint := advancePointByBytes(numLit.startPoint, source[numLit.startByte:signEndByte])
	fix.operatorLeaf = newLeafNodeInArena(arena, signSym, false, numLit.startByte, signEndByte, numLit.startPoint, signEndPoint)
	fix.shrunkNumber = newLeafNodeInArena(arena, syms.numberLiteral, syms.numberLiteralNamed, signEndByte, numLit.endByte, signEndPoint, numLit.endPoint)

	ident := newLeafNodeInArena(arena, syms.identifier, syms.identifierNamed, typeIdent.startByte, typeIdent.endByte, typeIdent.startPoint, typeIdent.endPoint)
	parenExpr := newParentNodeInArena(arena, syms.parenthesized, syms.parenthesizedNamed,
		cloneNodeSliceIfArena(arena, []*Node{openParen, ident, closeParen}), nil, 0)
	fix.fixedSizeof = newParentNodeInArena(arena, syms.sizeofExpr, symbolIsNamed(lang, syms.sizeofExpr),
		cloneNodeSliceIfArena(arena, []*Node{sizeofKeyword, parenExpr}),
		cloneFieldIDSliceInArena(arena, []FieldID{0, syms.valueFID}), 0)
	return fix, true
}
