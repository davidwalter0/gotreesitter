package gotreesitter

// normalizeCMacroCallingConventionTypeSpecifier repairs gt's mis-resolution
// of the GL/EGL/GLES calling-convention-macro idiom:
//
//	RetType (CALLING_CONV_MACRO name) (params);
//
// e.g. `typedef GLenum (GL_APIENTRYP PFNGLCHECKFRAMEBUFFERSTATUSPROC)
// (GLenum target);` — the standard way GL/EGL/GLES headers declare or
// typedef a calling-convention-annotated function pointer (GL_APIENTRYP
// itself expands, post-preprocessing, to something like `* APIENTRY`; since
// tree-sitter parses pre-preprocessor, it sees a bare identifier there,
// which the grammar's own `macro_type_specifier` rule was written to
// tolerate as an ambiguous "macro-decorated declarator").
//
// Root cause (parse-table layer, not fixable here): tree-sitter-c's grammar
// declares an explicit conflict between `type_specifier` and
// `macro_type_specifier` (`identifier '(' type_descriptor ')'`, dynamic
// precedence -1). When the return type is a bare, non-primitive identifier
// (e.g. `GLenum`), the oracle's LALR/GLR automaton resolves the ambiguity by
// committing to `macro_type_specifier` — the *only* production that can make
// forward progress without an immediate hard error, despite its lower
// dynamic precedence — and recovers from the inner "two identifiers where
// one type_descriptor is expected" mismatch by wrapping the first identifier
// (the calling-convention macro) in an ERROR node and treating the second
// (the real declarator name) as the macro_type_specifier's type_descriptor.
// gt's merged parse-table state instead keeps the bare type_identifier and
// falls through to its own (separately correct) parenthesized_declarator
// recovery for the "(MACRO name)" pair — never building the
// macro_type_specifier wrapper the oracle does. When the return type is a
// keyword primitive (`void`), there is no such ambiguity (`primitive_type`
// cannot be confused with `macro_type_specifier`'s identifier name field),
// so both sides already agree — the bug is specific to non-keyword return
// types.
//
// Detection is conservative and narrowly signature-gated: a
// type_definition/declaration whose type field is a bare type_identifier
// immediately followed (as the declarator field) by a function_declarator
// whose own declarator is a parenthesized_declarator of the exact shape
// `( ERROR(identifier) identifier )` — precisely what gt already produces,
// correctly, for the "(MACRO name)" pair on its own; this pass only relocates
// and retags it under a macro_type_specifier the way the oracle does.
//
// The trailing parameter list is reshaped too — the oracle does not keep it
// as a function_declarator+parameter_list once the return type takes the
// macro_type_specifier branch; it re-parses `(params)` from scratch under
// generic declarator rules. For the one-parameter cases, which of the type
// vs. declarator-name tokens ends up wrapped in the synthetic ERROR node is
// *not* fixed — tree-sitter's GLR error recovery picks whichever candidate
// minimizes the total byte span marked as an error, so it wraps whichever
// side is shorter (see cBuildMacroCallingConventionSingleParamDeclarator's
// doc comment for the reverse-engineered rule and its oracle evidence).
// Replicating this faithfully for arbitrary (2+)-parameter lists is not
// tractable as a conservative post-parse rewrite — the cost search has more
// viable partitions than a simple two/three-token comparison — so only the
// small set of shapes empirically verified against the pinned oracle
// (github.com/tree-sitter/tree-sitter-c @ ae19b676) are rebuilt:
//   - zero parameters, `(void)`;
//   - one parameter whose type is a single bare identifier (the dominant
//     real-world case: GL/EGL typedef'd scalar types like GLenum/GLuint);
//   - one parameter whose type is a two-token modifier+primitive combo
//     (e.g. `unsigned int`).
//
// Any other trailing shape (2+ parameters, pointer/array declarators,
// qualified types, …) is left completely untouched — this pass makes no
// change at all to that declaration rather than emit a guessed, unverified
// tree.
func normalizeCMacroCallingConventionTypeSpecifier(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "c" || len(source) == 0 {
		return
	}
	syms, ok := cMacroCallingConventionSymbolsForLanguage(lang)
	if !ok {
		return
	}
	walkResultTreePostorder(root, func(n *Node) {
		normalizeCMacroCallingConventionNode(n, source, lang, syms)
	})
}

type cMacroCallingConventionSymbols struct {
	typeDefinition          Symbol
	declaration             Symbol
	typeIdentifier          Symbol
	identifier              Symbol
	functionDeclarator      Symbol
	parenthesizedDeclarator Symbol
	parameterList           Symbol
	parameterDeclaration    Symbol
	sizedTypeSpecifier      Symbol
	primitiveType           Symbol
	macroTypeSpecifier      Symbol
	typeDescriptor          Symbol

	nameFID       FieldID
	typeFID       FieldID
	declaratorFID FieldID
	parametersFID FieldID

	identifierNamed        bool
	typeIdentifierNamed    bool
	primitiveTypeNamed     bool
	macroTypeSpecNamed     bool
	typeDescriptorNamed    bool
	parenthesizedDeclNamed bool
}

func cMacroCallingConventionSymbolsForLanguage(lang *Language) (cMacroCallingConventionSymbols, bool) {
	var s cMacroCallingConventionSymbols
	var ok bool
	if s.typeDefinition, ok = symbolByName(lang, "type_definition"); !ok {
		return s, false
	}
	if s.declaration, ok = symbolByName(lang, "declaration"); !ok {
		return s, false
	}
	if s.typeIdentifier, ok = symbolByName(lang, "type_identifier"); !ok {
		return s, false
	}
	if s.identifier, ok = symbolByName(lang, "identifier"); !ok {
		return s, false
	}
	if s.functionDeclarator, ok = symbolByName(lang, "function_declarator"); !ok {
		return s, false
	}
	if s.parenthesizedDeclarator, ok = symbolByName(lang, "parenthesized_declarator"); !ok {
		return s, false
	}
	if s.parameterList, ok = symbolByName(lang, "parameter_list"); !ok {
		return s, false
	}
	if s.parameterDeclaration, ok = symbolByName(lang, "parameter_declaration"); !ok {
		return s, false
	}
	if s.sizedTypeSpecifier, ok = symbolByName(lang, "sized_type_specifier"); !ok {
		return s, false
	}
	if s.primitiveType, ok = symbolByName(lang, "primitive_type"); !ok {
		return s, false
	}
	if s.macroTypeSpecifier, ok = symbolByName(lang, "macro_type_specifier"); !ok {
		return s, false
	}
	if s.typeDescriptor, ok = symbolByName(lang, "type_descriptor"); !ok {
		return s, false
	}
	if s.nameFID, ok = lang.FieldByName("name"); !ok {
		return s, false
	}
	if s.typeFID, ok = lang.FieldByName("type"); !ok {
		return s, false
	}
	if s.declaratorFID, ok = lang.FieldByName("declarator"); !ok {
		return s, false
	}
	if s.parametersFID, ok = lang.FieldByName("parameters"); !ok {
		return s, false
	}
	s.identifierNamed = symbolIsNamed(lang, s.identifier)
	s.typeIdentifierNamed = symbolIsNamed(lang, s.typeIdentifier)
	s.primitiveTypeNamed = symbolIsNamed(lang, s.primitiveType)
	s.macroTypeSpecNamed = symbolIsNamed(lang, s.macroTypeSpecifier)
	s.typeDescriptorNamed = symbolIsNamed(lang, s.typeDescriptor)
	s.parenthesizedDeclNamed = symbolIsNamed(lang, s.parenthesizedDeclarator)
	return s, true
}

func normalizeCMacroCallingConventionNode(n *Node, source []byte, lang *Language, syms cMacroCallingConventionSymbols) {
	if n == nil {
		return
	}
	if !cResultSymbolMatches(lang, n, syms.typeDefinition) && !cResultSymbolMatches(lang, n, syms.declaration) {
		return
	}

	childCount := resultChildCount(n)
	typeIdx, fdIdx := -1, -1
	var typeNode, fd *Node
	for i := 0; i < childCount; i++ {
		child := resultChildAt(n, i)
		if child == nil {
			continue
		}
		switch n.FieldNameForChild(i, lang) {
		case "type":
			typeNode, typeIdx = child, i
		case "declarator":
			fd, fdIdx = child, i
		}
	}
	if typeNode == nil || fd == nil || typeIdx < 0 || fdIdx < 0 {
		return
	}
	if !cResultSymbolMatches(lang, typeNode, syms.typeIdentifier) {
		return
	}
	if !cResultSymbolMatches(lang, fd, syms.functionDeclarator) || resultChildCount(fd) != 2 {
		return
	}
	if fd.FieldNameForChild(0, lang) != "declarator" || fd.FieldNameForChild(1, lang) != "parameters" {
		return
	}
	pd := resultChildAt(fd, 0)
	pl := resultChildAt(fd, 1)
	if pd == nil || pl == nil ||
		!cResultSymbolMatches(lang, pd, syms.parenthesizedDeclarator) ||
		!cResultSymbolMatches(lang, pl, syms.parameterList) {
		return
	}
	if resultChildCount(pd) != 4 {
		return
	}
	openParen := resultChildAt(pd, 0)
	errNode := resultChildAt(pd, 1)
	nameNode := resultChildAt(pd, 2)
	closeParen := resultChildAt(pd, 3)
	if openParen == nil || errNode == nil || nameNode == nil || closeParen == nil {
		return
	}
	if openParen.Type(lang) != "(" || closeParen.Type(lang) != ")" {
		return
	}
	if !errNode.IsError() || resultChildCount(errNode) != 1 {
		return
	}
	macroNode := resultChildAt(errNode, 0)
	if macroNode == nil ||
		(!cResultSymbolMatches(lang, macroNode, syms.typeIdentifier) && !cResultSymbolMatches(lang, macroNode, syms.identifier)) ||
		(!cResultSymbolMatches(lang, nameNode, syms.typeIdentifier) && !cResultSymbolMatches(lang, nameNode, syms.identifier)) {
		return
	}

	newDeclarator := cBuildMacroCallingConventionDeclarator(pl, source, lang, syms)
	if newDeclarator == nil {
		// Trailing parameter shape isn't one of the empirically-verified
		// forms; leave the whole declaration untouched rather than guess.
		return
	}

	arena := n.ownerArena

	// Retag in place: T (type_identifier) -> identifier (macro name field);
	// M (ERROR's child) and N (declarator name) -> type_identifier. Oracle
	// uses these exact node kinds regardless of typedef vs. plain
	// declaration form.
	typeNode.symbol = syms.identifier
	typeNode.setNamed(syms.identifierNamed)
	macroNode.symbol = syms.typeIdentifier
	macroNode.setNamed(syms.typeIdentifierNamed)
	nameNode.symbol = syms.typeIdentifier
	nameNode.setNamed(syms.typeIdentifierNamed)

	typeDescriptor := newParentNodeInArena(arena, syms.typeDescriptor, syms.typeDescriptorNamed,
		cloneNodeSliceIfArena(arena, []*Node{nameNode}),
		cloneFieldIDSliceInArena(arena, []FieldID{syms.typeFID}), 0)

	macroChildren := cloneNodeSliceIfArena(arena, []*Node{typeNode, openParen, errNode, typeDescriptor, closeParen})
	macroFieldIDs := cloneFieldIDSliceInArena(arena, []FieldID{syms.nameFID, 0, 0, syms.typeFID, 0})
	macroTypeSpec := newParentNodeInArena(arena, syms.macroTypeSpecifier, syms.macroTypeSpecNamed, macroChildren, macroFieldIDs, 0)

	children := resultChildSliceForMutation(n)
	out := make([]*Node, len(children))
	copy(out, children)
	out[typeIdx] = macroTypeSpec
	out[fdIdx] = newDeclarator
	replaceCMacroCallingConventionOuterChildren(n, cloneNodeSliceIfArena(arena, out))
}

// replaceCMacroCallingConventionOuterChildren swaps in the rewritten
// type/declarator children of a type_definition/declaration node while
// preserving its own symbol and its existing field-ID-per-index assignment
// (only the child *values* at the type/declarator slots changed, not which
// slot carries which field).
func replaceCMacroCallingConventionOuterChildren(n *Node, children []*Node) {
	fieldIDs := append([]FieldID(nil), n.fieldIDs()...)
	fieldSources := append([]uint8(nil), n.fieldSources()...)
	n.children = children
	if len(fieldIDs) == len(children) {
		n.setFieldMetadata(fieldIDs, fieldSources)
	}
	if n.ownerArena != nil {
		n.ownerArena.clearFinalChildRefs(n)
	}
	n.productionID = 0
	populateParentNode(n, n.children)
}

// cBuildMacroCallingConventionDeclarator rebuilds the trailing "(params)" as
// the oracle's own re-parsed generic declarator, for the handful of
// parameter-list shapes verified against the pinned tree-sitter-c oracle.
// Returns nil for any other shape (caller leaves the declaration untouched).
func cBuildMacroCallingConventionDeclarator(pl *Node, source []byte, lang *Language, syms cMacroCallingConventionSymbols) *Node {
	if pl == nil || resultChildCount(pl) != 3 {
		return nil
	}
	openParen := resultChildAt(pl, 0)
	paramDecl := resultChildAt(pl, 1)
	closeParen := resultChildAt(pl, 2)
	if openParen == nil || paramDecl == nil || closeParen == nil ||
		openParen.Type(lang) != "(" || closeParen.Type(lang) != ")" ||
		!cResultSymbolMatches(lang, paramDecl, syms.parameterDeclaration) {
		return nil
	}

	switch resultChildCount(paramDecl) {
	case 1:
		return cBuildMacroCallingConventionVoidDeclarator(pl, paramDecl, source, lang, syms)
	case 2:
		return cBuildMacroCallingConventionSingleParamDeclarator(pl, paramDecl, lang, syms)
	default:
		return nil
	}
}

// cBuildMacroCallingConventionVoidDeclarator handles the zero-parameter
// "(void)" case. Oracle: parenthesized_declarator(primitive_type).
func cBuildMacroCallingConventionVoidDeclarator(pl, paramDecl *Node, source []byte, lang *Language, syms cMacroCallingConventionSymbols) *Node {
	sole := resultChildAt(paramDecl, 0)
	if sole == nil || !cResultSymbolMatches(lang, sole, syms.primitiveType) || string(sole.Text(source)) != "void" {
		return nil
	}
	arena := pl.ownerArena
	children := cloneNodeSliceIfArena(arena, []*Node{resultChildAt(pl, 0), sole, resultChildAt(pl, 2)})
	return newParentNodeInArena(arena, syms.parenthesizedDeclarator, syms.parenthesizedDeclNamed, children, nil, 0)
}

// cBuildMacroCallingConventionSingleParamDeclarator handles the one-parameter
// cases: a single-token identifier type (e.g. `GLenum target`) or a
// two-token modifier+primitive type (e.g. `unsigned int target`).
//
// Which of the two tokens (type vs. declarator name) ends up wrapped in the
// synthetic ERROR node is *not* fixed — it depends on their relative byte
// lengths. tree-sitter's GLR error recovery picks whichever candidate
// parse minimizes the total byte span marked as an error, so it wraps
// whichever side is *shorter* (ties go to keeping the type/first-word side
// bare). Empirically reverse-engineered against the pinned oracle
// (github.com/tree-sitter/tree-sitter-c @ ae19b676) by sweeping declarator
// name lengths against fixed type spellings:
//   - `GLenum`(6) vs `target`(6): tie -> type wrapped, name bare.
//   - `GLenum`(6) vs `type`/`name`/`id`(<6): name wrapped, type bare.
//   - `GLenum`(6) vs `length`(6): tie -> type wrapped (confirms the rule
//     isn't about specific spellings, only byte length).
//   - `Foo`(3) vs 1-7 byte names: shorter side wrapped, tie (len 3) wraps
//     type.
//
// The two-word `unsigned int NAME` case follows the identical
// minimize-total-error-span principle across the two viable partitions —
// wrap (int+NAME) vs wrap (unsigned+int) — verified by sweeping NAME's
// length against the fixed "unsigned"(8)+"int"(3) prefix: the boundary
// lands exactly at NAME length == len("unsigned"), matching
// len("int")+1+len(NAME) compared against len("unsigned")+1+len("int").
func cBuildMacroCallingConventionSingleParamDeclarator(pl, paramDecl *Node, lang *Language, syms cMacroCallingConventionSymbols) *Node {
	typeChild := resultChildAt(paramDecl, 0)
	declChild := resultChildAt(paramDecl, 1)
	if typeChild == nil || declChild == nil || !cResultSymbolMatches(lang, declChild, syms.identifier) {
		return nil
	}
	arena := pl.ownerArena
	nameLen := declChild.endByte - declChild.startByte

	switch {
	case cResultSymbolMatches(lang, typeChild, syms.typeIdentifier):
		typeLen := typeChild.endByte - typeChild.startByte
		if nameLen < typeLen {
			// Oracle: parenthesized_declarator(type_identifier<TYPE> (bare), ERROR(identifier<NAME>)).
			errNode := newParentNodeInArena(arena, errorSymbol, true, cloneNodeSliceIfArena(arena, []*Node{declChild}), nil, 0)
			errNode.setHasError(true)
			errNode.setExtra(true)
			children := cloneNodeSliceIfArena(arena, []*Node{resultChildAt(pl, 0), typeChild, errNode, resultChildAt(pl, 2)})
			return newParentNodeInArena(arena, syms.parenthesizedDeclarator, syms.parenthesizedDeclNamed, children, nil, 0)
		}
		// nameLen >= typeLen (tie included): parenthesized_declarator(ERROR(type_identifier<TYPE>), type_identifier<NAME>).
		declChild.symbol = syms.typeIdentifier
		declChild.setNamed(syms.typeIdentifierNamed)
		errNode := newParentNodeInArena(arena, errorSymbol, true, cloneNodeSliceIfArena(arena, []*Node{typeChild}), nil, 0)
		errNode.setHasError(true)
		errNode.setExtra(true)
		children := cloneNodeSliceIfArena(arena, []*Node{resultChildAt(pl, 0), errNode, declChild, resultChildAt(pl, 2)})
		return newParentNodeInArena(arena, syms.parenthesizedDeclarator, syms.parenthesizedDeclNamed, children, nil, 0)

	case cResultSymbolMatches(lang, typeChild, syms.sizedTypeSpecifier):
		if resultChildCount(typeChild) != 2 {
			return nil
		}
		word1 := resultChildAt(typeChild, 0)
		word2 := resultChildAt(typeChild, 1)
		if word1 == nil || word2 == nil || word1.IsNamed() || !cResultSymbolMatches(lang, word2, syms.primitiveType) {
			return nil
		}
		word1Len := word1.endByte - word1.startByte

		if nameLen <= word1Len {
			// Oracle: parenthesized_declarator(primitive_type<word1>, ERROR(identifier<word2>, identifier<NAME>)).
			word1.symbol = syms.primitiveType
			word1.setNamed(syms.primitiveTypeNamed)
			word2.symbol = syms.identifier
			word2.setNamed(syms.identifierNamed)
			errNode := newParentNodeInArena(arena, errorSymbol, true, cloneNodeSliceIfArena(arena, []*Node{word2, declChild}), nil, 0)
			errNode.setHasError(true)
			errNode.setExtra(true)
			children := cloneNodeSliceIfArena(arena, []*Node{resultChildAt(pl, 0), word1, errNode, resultChildAt(pl, 2)})
			return newParentNodeInArena(arena, syms.parenthesizedDeclarator, syms.parenthesizedDeclNamed, children, nil, 0)
		}
		// nameLen > word1Len: parenthesized_declarator(ERROR(primitive_type<word1>, identifier<word2>), type_identifier<NAME>).
		word1.symbol = syms.primitiveType
		word1.setNamed(syms.primitiveTypeNamed)
		word2.symbol = syms.identifier
		word2.setNamed(syms.identifierNamed)
		declChild.symbol = syms.typeIdentifier
		declChild.setNamed(syms.typeIdentifierNamed)
		errNode := newParentNodeInArena(arena, errorSymbol, true, cloneNodeSliceIfArena(arena, []*Node{word1, word2}), nil, 0)
		errNode.setHasError(true)
		errNode.setExtra(true)
		children := cloneNodeSliceIfArena(arena, []*Node{resultChildAt(pl, 0), errNode, declChild, resultChildAt(pl, 2)})
		return newParentNodeInArena(arena, syms.parenthesizedDeclarator, syms.parenthesizedDeclNamed, children, nil, 0)

	default:
		return nil
	}
}
