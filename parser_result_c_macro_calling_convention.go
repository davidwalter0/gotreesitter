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
// generic declarator rules. Which of the type vs. declarator-name tokens
// ends up wrapped in the synthetic ERROR node is *not* fixed — tree-sitter's
// GLR error recovery picks whichever candidate minimizes the total byte span
// marked as an error (see cBuildMacroCallingConventionSingleParamDeclarator's
// doc comment for the one-parameter rule, and
// cBuildMacroCallingConventionMultiParamDeclarator's for the two/three
// parameter extension). Only the shapes empirically verified against the
// pinned oracle (github.com/tree-sitter/tree-sitter-c @ ae19b676) are
// rebuilt:
//   - zero parameters, `(void)`;
//   - one parameter whose type is a single bare identifier (the dominant
//     real-world case: GL/EGL typedef'd scalar types like GLenum/GLuint);
//   - one parameter whose type is a two-token modifier+primitive combo
//     (e.g. `unsigned int`);
//   - two or three parameters, where every leading parameter is a plain
//     `TYPE name` pair and the final parameter is either also plain or a
//     (optionally `const`-qualified) pointer declarator — see
//     cBuildMacroCallingConventionMultiParamDeclarator.
//
// Four-or-more-parameter lists are deliberately left untouched: direct
// probing against the pinned oracle (`tree-sitter parse --cst`, cross-checked
// against the corpus_parity harness's own dump.v1 JSON — see the methodology
// note below) shows that at four parameters the same shape family can either
// reproduce cleanly (11 of 15 real four-parameter GL/EGL signatures sampled)
// or cascade into a completely different, unpredictable recovery — extra
// MISSING tokens and a second top-level declaration split off from the
// typedef (e.g. the real egl.h entry `typedef EGLContext (EGLAPIENTRYP
// PFNEGLCREATECONTEXTPROC) (EGLDisplay dpy, EGLConfig config, EGLContext
// share_context, const EGLint *attrib_list);`) — with no structural signal
// available (short of running the reference grammar) to tell the two apart
// ahead of time. Any other trailing shape (unsupported parameter counts,
// pointer/array declarators in a non-final position, non-`const`
// qualifiers, …) is likewise left completely untouched — this pass makes no
// change at all to those declarations rather than emit a guessed, unverified
// tree.
//
// Methodology note (round6-c-multiparam): the `tree-sitter parse` CLI's
// default rendering shows only *named* nodes and, in one observed case,
// silently omitted a `,` token that the corpus_parity harness's own
// dump.v1 JSON (produced by the exact cgo-bound oracle used for scoring)
// confirmed *was* present as a real child of the surrounding ERROR node.
// An early draft of the two/three-parameter extension below trusted the
// CLI's rendering and dropped those `,` tokens from its reconstructed
// ERROR nodes — every multi-parameter shape is now cross-checked against
// either a real corpus file's dump.v1 JSON or a one-off corpus_parity
// Docker run over a synthetic single-line corpus (same harness, same
// oracle, no CLI in the loop) before being treated as verified.
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
	pointerDeclarator       Symbol
	typeQualifier           Symbol

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
	if s.pointerDeclarator, ok = symbolByName(lang, "pointer_declarator"); !ok {
		return s, false
	}
	if s.typeQualifier, ok = symbolByName(lang, "type_qualifier"); !ok {
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
	if pl == nil {
		return nil
	}
	childCount := resultChildCount(pl)
	if childCount == 3 {
		// Exactly one parameter_declaration (or `void`) between the parens
		// — the original, well-tested zero/one-parameter path. Left
		// completely unchanged from before this extension.
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

	return cBuildMacroCallingConventionMultiParamDeclarator(pl, lang, syms)
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

// cBuildMacroCallingConventionMultiParamDeclarator extends the single-
// parameter rule above to two- and three-parameter lists, the dominant
// residual shape in the real GL/EGL/GLES corpus (PFNGL*PROC / PFNEGL*PROC
// typedefs taking a receiver plus one-or-more scalar/pointer arguments,
// e.g. `PFNGLGETATTRIBLOCATIONPROC (GLuint program, const GLchar *name)`).
//
// Detection is conservative: every leading parameter (all but the last)
// must be a plain `TYPE name` pair (a bare type_identifier plus a bare
// identifier declarator — no pointers, no qualifiers, no multi-word
// types). Anything else in a leading position — a pointer parameter, a
// `const`-qualified parameter, a two-word type — bails out to the
// existing no-op fallback, since those shapes were not empirically
// verified and (per the doc comment above) are known to sometimes cascade
// into a completely different, unpredictable oracle recovery.
//
// The final parameter may additionally be:
//   - a pointer declarator, optionally preceded by a single `const`
//     qualifier (e.g. `const GLchar *name`, `GLint *value`) — verified
//     against the pinned oracle for both the qualified and unqualified
//     forms, at both two and three total parameters.
//   - a plain `TYPE name` pair, same as the leading parameters.
//
// Split direction (byte-for-byte verified against the pinned oracle,
// github.com/tree-sitter/tree-sitter-c @ ae19b676, across every real
// two/three-parameter, non-void-return calling-convention typedef in the
// usr/include/{EGL,GLES,GLES2}/*.h corpus plus targeted synthetic probes):
//
//   - Final parameter has a pointer declarator: ALWAYS "back-split" —
//     every leading parameter plus the final parameter's (optional
//     qualifier and) type are merged into one ERROR node, and only the
//     pointer declarator itself is left bare (its own inner identifier
//     promoted to type_identifier, matching the oracle's
//     `pointer_declarator declarator: type_identifier` shape). This holds
//     regardless of byte lengths — e.g. PFNGLGETATTRIBLOCATIONPROC
//     (2 params) and PFNEGLCREATEPBUFFERSURFACEPROC (3 params) both
//     back-split despite very different length ratios.
//
//   - Final parameter is also plain: the split direction is decided by
//     comparing the byte length of the very first parameter's type
//     against the byte length of the final parameter's type (the two
//     "edge" tokens) — front-split (bare = first type, error = the rest)
//     wins on a tie or when the first type is longer; back-split (error =
//     everything up to the final type, which stays bare as type_identifier
//     ; bare = the final declarator, promoted to type_identifier) wins
//     when the final type is strictly longer. Confirmed against multiple
//     independent real corpus pairs with the same parameter *shape* but
//     opposite outcomes purely from this one length comparison, e.g.
//     PFNEGLBINDTEXIMAGEPROC (`EGLDisplay`=10 > `EGLint`=6 → front) versus
//     PFNEGLCOPYBUFFERSPROC (`EGLDisplay`=10 < `EGLNativePixmapType`=20 →
//     back), which are otherwise structurally identical (three plain
//     parameters). This is *not* a simple global error-span minimization
//     (a smaller-error front-split alternative existed for
//     PFNEGLCOPYBUFFERSPROC's own shape and was not what the oracle
//     picked) — it is the narrower, specifically-observed rule stated
//     above, not a general cost search.
//
// Four-or-more parameters are out of scope (see the file-level doc
// comment) and return nil unconditionally.
func cBuildMacroCallingConventionMultiParamDeclarator(pl *Node, lang *Language, syms cMacroCallingConventionSymbols) *Node {
	childCount := resultChildCount(pl)
	if childCount < 5 || childCount%2 != 1 {
		return nil
	}
	n := (childCount - 1) / 2
	if n < 2 || n > 3 {
		return nil
	}

	openParen := resultChildAt(pl, 0)
	closeParen := resultChildAt(pl, childCount-1)
	if openParen == nil || closeParen == nil || openParen.Type(lang) != "(" || closeParen.Type(lang) != ")" {
		return nil
	}

	params := make([]*Node, n)
	commas := make([]*Node, n-1)
	for i := 0; i < n; i++ {
		p := resultChildAt(pl, 1+2*i)
		if p == nil || !cResultSymbolMatches(lang, p, syms.parameterDeclaration) {
			return nil
		}
		params[i] = p
		if i < n-1 {
			comma := resultChildAt(pl, 2+2*i)
			if comma == nil || comma.Type(lang) != "," {
				return nil
			}
			commas[i] = comma
		}
	}

	leadTypes := make([]*Node, n-1)
	leadNames := make([]*Node, n-1)
	for i := 0; i < n-1; i++ {
		t, nm, ok := cMacroCallingConventionClassifyPlainParam(params[i], lang, syms)
		if !ok {
			return nil
		}
		leadTypes[i] = t
		leadNames[i] = nm
	}

	qualifierToken, lastType, lastDecl, isPointer, ok := cMacroCallingConventionClassifyLastParam(params[n-1], lang, syms)
	if !ok {
		return nil
	}

	arena := pl.ownerArena

	if isPointer {
		return cAssembleMacroCallingConventionBackSplit(arena, openParen, closeParen, leadTypes, leadNames, commas, qualifierToken, lastType, lastDecl, true, syms)
	}

	type0Len := int(leadTypes[0].endByte - leadTypes[0].startByte)
	typeLastLen := int(lastType.endByte - lastType.startByte)
	if type0Len >= typeLastLen {
		return cAssembleMacroCallingConventionFrontSplit(arena, openParen, closeParen, leadTypes, leadNames, commas, lastType, lastDecl, syms)
	}
	return cAssembleMacroCallingConventionBackSplit(arena, openParen, closeParen, leadTypes, leadNames, commas, nil, lastType, lastDecl, false, syms)
}

// cMacroCallingConventionClassifyPlainParam recognizes the conservative
// leading-parameter shape: parameter_declaration(type_identifier,
// identifier) — a bare typedef'd type name and a bare declarator name, no
// qualifiers or pointers.
func cMacroCallingConventionClassifyPlainParam(p *Node, lang *Language, syms cMacroCallingConventionSymbols) (typeNode, nameNode *Node, ok bool) {
	if p == nil || resultChildCount(p) != 2 {
		return nil, nil, false
	}
	t := resultChildAt(p, 0)
	nm := resultChildAt(p, 1)
	if t == nil || nm == nil {
		return nil, nil, false
	}
	if !cResultSymbolMatches(lang, t, syms.typeIdentifier) || !cResultSymbolMatches(lang, nm, syms.identifier) {
		return nil, nil, false
	}
	return t, nm, true
}

// cMacroCallingConventionClassifyLastParam recognizes the three supported
// shapes for the final parameter: plain (parameter_declaration(
// type_identifier, identifier)), unqualified pointer (parameter_declaration(
// type_identifier, pointer_declarator(*, identifier))), or const-qualified
// pointer (parameter_declaration(type_qualifier(const), type_identifier,
// pointer_declarator(*, identifier))). Returns the unwrapped bare `const`
// token (nil if no qualifier), the type node, the declarator node (either
// the bare identifier or the pointer_declarator), and whether it's a
// pointer shape.
func cMacroCallingConventionClassifyLastParam(p *Node, lang *Language, syms cMacroCallingConventionSymbols) (qualifierToken, typeNode, declNode *Node, isPointer, ok bool) {
	if p == nil {
		return nil, nil, nil, false, false
	}
	switch resultChildCount(p) {
	case 2:
		t := resultChildAt(p, 0)
		d := resultChildAt(p, 1)
		if t == nil || d == nil || !cResultSymbolMatches(lang, t, syms.typeIdentifier) {
			return nil, nil, nil, false, false
		}
		if cResultSymbolMatches(lang, d, syms.identifier) {
			return nil, t, d, false, true
		}
		if cMacroCallingConventionValidPointerDeclarator(d, lang, syms) {
			return nil, t, d, true, true
		}
		return nil, nil, nil, false, false

	case 3:
		q := resultChildAt(p, 0)
		t := resultChildAt(p, 1)
		d := resultChildAt(p, 2)
		if q == nil || t == nil || d == nil {
			return nil, nil, nil, false, false
		}
		if !cResultSymbolMatches(lang, q, syms.typeQualifier) || resultChildCount(q) != 1 {
			return nil, nil, nil, false, false
		}
		qtok := resultChildAt(q, 0)
		if qtok == nil || qtok.Type(lang) != "const" {
			return nil, nil, nil, false, false
		}
		if !cResultSymbolMatches(lang, t, syms.typeIdentifier) {
			return nil, nil, nil, false, false
		}
		if !cMacroCallingConventionValidPointerDeclarator(d, lang, syms) {
			return nil, nil, nil, false, false
		}
		return qtok, t, d, true, true

	default:
		return nil, nil, nil, false, false
	}
}

// cMacroCallingConventionValidPointerDeclarator checks for the shape
// pointer_declarator("*", identifier) — a single level of pointer
// indirection over a bare declarator name. Multi-level pointers or
// array/function declarators are not recognized (conservative).
func cMacroCallingConventionValidPointerDeclarator(d *Node, lang *Language, syms cMacroCallingConventionSymbols) bool {
	if d == nil || !cResultSymbolMatches(lang, d, syms.pointerDeclarator) || resultChildCount(d) != 2 {
		return false
	}
	star := resultChildAt(d, 0)
	inner := resultChildAt(d, 1)
	if star == nil || inner == nil || star.Type(lang) != "*" {
		return false
	}
	return cResultSymbolMatches(lang, inner, syms.identifier)
}

// cAssembleMacroCallingConventionFrontSplit builds the "bare first type +
// giant trailing ERROR" shape: only the first parameter's type survives
// outside the error; its own declarator name, every other leading
// parameter (type demoted to identifier, name unchanged), the final
// parameter (type demoted, name unchanged — this branch only fires when
// the final parameter is plain), and every comma separating them (the
// oracle keeps each original `,` token as a bare ERROR child in its
// original position, it is not dropped) are all merged into one ERROR
// node.
func cAssembleMacroCallingConventionFrontSplit(arena *nodeArena, openParen, closeParen *Node, leadTypes, leadNames, commas []*Node, lastType, lastName *Node, syms cMacroCallingConventionSymbols) *Node {
	bareType := leadTypes[0]

	errChildren := make([]*Node, 0, 3*len(leadTypes)+3)
	errChildren = append(errChildren, leadNames[0])
	for i := 1; i < len(leadTypes); i++ {
		errChildren = append(errChildren, commas[i-1])
		cMacroCallingConventionDemoteToIdentifier(leadTypes[i], syms)
		errChildren = append(errChildren, leadTypes[i], leadNames[i])
	}
	errChildren = append(errChildren, commas[len(leadTypes)-1])
	cMacroCallingConventionDemoteToIdentifier(lastType, syms)
	errChildren = append(errChildren, lastType, lastName)

	errNode := newParentNodeInArena(arena, errorSymbol, true, cloneNodeSliceIfArena(arena, errChildren), nil, 0)
	errNode.setHasError(true)
	errNode.setExtra(true)

	children := cloneNodeSliceIfArena(arena, []*Node{openParen, bareType, errNode, closeParen})
	return newParentNodeInArena(arena, syms.parenthesizedDeclarator, syms.parenthesizedDeclNamed, children, nil, 0)
}

// cAssembleMacroCallingConventionBackSplit builds the "giant leading ERROR
// + bare trailing declarator" shape shared by both sub-cases: pointer-last
// (unconditional) and plain-last (when the final type is strictly longer
// than the first). The first parameter's type is the sole survivor of
// type_identifier tagging *inside* the error (every other leading type is
// demoted to identifier); the final parameter's own type additionally
// stays type_identifier when the trailing declarator is plain (adjacent to
// the bare boundary) but gets demoted when it's a pointer (matching the
// oracle's two distinct observed shapes). Every comma separating the
// wrapped parameters is preserved as a bare ERROR child in its original
// position, same as the front-split case above.
func cAssembleMacroCallingConventionBackSplit(arena *nodeArena, openParen, closeParen *Node, leadTypes, leadNames, commas []*Node, qualifierToken, lastType, lastDecl *Node, isPointer bool, syms cMacroCallingConventionSymbols) *Node {
	errChildren := make([]*Node, 0, 3*len(leadTypes)+3)
	errChildren = append(errChildren, leadTypes[0], leadNames[0])
	for i := 1; i < len(leadTypes); i++ {
		errChildren = append(errChildren, commas[i-1])
		cMacroCallingConventionDemoteToIdentifier(leadTypes[i], syms)
		errChildren = append(errChildren, leadTypes[i], leadNames[i])
	}
	errChildren = append(errChildren, commas[len(leadTypes)-1])
	if qualifierToken != nil {
		errChildren = append(errChildren, qualifierToken)
	}
	if isPointer {
		cMacroCallingConventionDemoteToIdentifier(lastType, syms)
	}
	errChildren = append(errChildren, lastType)

	errNode := newParentNodeInArena(arena, errorSymbol, true, cloneNodeSliceIfArena(arena, errChildren), nil, 0)
	errNode.setHasError(true)
	errNode.setExtra(true)

	var bareDecl *Node
	if isPointer {
		inner := resultChildAt(lastDecl, 1)
		inner.symbol = syms.typeIdentifier
		inner.setNamed(syms.typeIdentifierNamed)
		bareDecl = lastDecl
	} else {
		lastDecl.symbol = syms.typeIdentifier
		lastDecl.setNamed(syms.typeIdentifierNamed)
		bareDecl = lastDecl
	}

	children := cloneNodeSliceIfArena(arena, []*Node{openParen, errNode, bareDecl, closeParen})
	return newParentNodeInArena(arena, syms.parenthesizedDeclarator, syms.parenthesizedDeclNamed, children, nil, 0)
}

// cMacroCallingConventionDemoteToIdentifier retags a type_identifier node
// (a parameter's type, once it's been absorbed into the synthetic ERROR
// node rather than left as a bare, recognizable declarator) to a plain
// identifier — matching the oracle's observed behavior that only the
// type(s) immediately adjacent to a bare boundary retain their
// type_identifier tagging inside an ERROR recovery.
func cMacroCallingConventionDemoteToIdentifier(n *Node, syms cMacroCallingConventionSymbols) {
	if n == nil {
		return
	}
	n.symbol = syms.identifier
	n.setNamed(syms.identifierNamed)
}
