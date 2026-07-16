package gotreesitter

// normalizeCppNestedTemplateDeclaration repairs blob-parity cpp-1: a variable
// declaration whose type is a two-or-more-level-deep template closed by `>>`
// (or `> >`) mis-parsed as an expression instead of a declaration.
//
// Root cause (shared GLR core, NOT safely fixable in a cpp-local way): the
// disambiguation of `<` as template-argument-list-open vs. less-than is a GLR
// conflict resolved by tree-sitter's dynamic precedence. gt resolves it
// correctly for a single template level and when the outer list's leading
// argument is simple (`a<x, b<int>>` parses fine), but fails specifically when
// the OUTER argument list's FIRST argument is itself a template (`a<b<int>> d`):
// on the `identifier <` lookahead gt commits to the less-than reading, so the
// whole construct collapses into a binary_expression chain (or a root ERROR).
// The oracle always prefers the template reading. Since the engine-level fix
// would touch the forest scoring every language shares (python/js/… at 100%
// parity), this stays a cpp-scoped post-parse repair.
//
// Strategy: this repair is deliberately limited to the DECLARATION context —
// the dominant real-world shape (`Type<Nested<…>> name;` at statement, global,
// or namespace scope). The offending statement is re-parsed from source with a
// small recursive type-id recognizer that rebuilds the oracle's
// declaration(type, declarator) shape. It only fires when the source cleanly
// tokenizes as `<type-id with a nested template> <declarator-identifier> ;` with
// no leftover — a genuine comparison/shift such as `a < b >> c;` leaves a
// non-declarator remainder (`> c`) and is left untouched. Contexts other than a
// plain declaration (call expressions, `if`/`return`/assignment operands) are
// out of scope and unchanged.
func normalizeCppNestedTemplateDeclaration(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "cpp" || len(source) == 0 {
		return
	}
	r, ok := newCppTemplateReconstructor(source, lang, root.ownerArena)
	if !ok {
		return
	}

	walkResultTree(root, func(n *Node) {
		// Case A: a single expression_statement that spans the whole
		// `Type<…> name ;` (statement / compound-statement context).
		if n.Type(lang) == "expression_statement" && resultChildCount(n) > 0 {
			if children, ok := r.buildDeclarationChildren(n.startByte, n.endByte); ok {
				r.retagAsDeclaration(n, children)
				return
			}
		}
		// Case B: at container level (translation_unit / declaration_list) the
		// mis-parse splits into an `expression` sibling holding `Type<…> name`
		// followed by an `expression_statement` holding just `;`. Merge the pair
		// into one declaration.
		r.mergeContainerExpressionSemicolonPairs(n)
	})
}

type cppTemplateReconstructor struct {
	src   []byte
	lang  *Language
	arena *nodeArena

	declSym        Symbol
	qualSym        Symbol
	nsIdentSym     Symbol
	tmplTypeSym    Symbol
	typeIdentSym   Symbol
	tmplArgListSym Symbol
	typeDescSym    Symbol
	primSym        Symbol
	identSym       Symbol
	exprSym        Symbol
	exprStmtSym    Symbol
	ltSym          Symbol
	gtSym          Symbol
	commaSym       Symbol
	semiSym        Symbol
	colonColonSym  Symbol

	declNamed     bool
	qualNamed     bool
	nsIdentNamed  bool
	tmplTypeNamed bool
	typeIdentN    bool
	tmplArgNamed  bool
	typeDescNamed bool
	primNamed     bool
	identNamed    bool

	typeFID       FieldID
	declaratorFID FieldID
	scopeFID      FieldID
	nameFID       FieldID
	argumentsFID  FieldID
}

func newCppTemplateReconstructor(source []byte, lang *Language, arena *nodeArena) (*cppTemplateReconstructor, bool) {
	if arena == nil {
		return nil, false
	}
	r := &cppTemplateReconstructor{src: source, lang: lang, arena: arena}
	syms := []struct {
		name string
		dst  *Symbol
	}{
		{"declaration", &r.declSym},
		{"qualified_identifier", &r.qualSym},
		{"namespace_identifier", &r.nsIdentSym},
		{"template_type", &r.tmplTypeSym},
		{"type_identifier", &r.typeIdentSym},
		{"template_argument_list", &r.tmplArgListSym},
		{"type_descriptor", &r.typeDescSym},
		{"primitive_type", &r.primSym},
		{"identifier", &r.identSym},
		{"expression", &r.exprSym},
		{"expression_statement", &r.exprStmtSym},
		{"<", &r.ltSym},
		{">", &r.gtSym},
		{",", &r.commaSym},
		{";", &r.semiSym},
		{"::", &r.colonColonSym},
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
		{"type", &r.typeFID},
		{"declarator", &r.declaratorFID},
		{"scope", &r.scopeFID},
		{"name", &r.nameFID},
		{"arguments", &r.argumentsFID},
	}
	for _, f := range fields {
		fid, ok := lang.FieldByName(f.name)
		if !ok {
			return nil, false
		}
		*f.dst = fid
	}
	r.declNamed = symbolIsNamed(lang, r.declSym)
	r.qualNamed = symbolIsNamed(lang, r.qualSym)
	r.nsIdentNamed = symbolIsNamed(lang, r.nsIdentSym)
	r.tmplTypeNamed = symbolIsNamed(lang, r.tmplTypeSym)
	r.typeIdentN = symbolIsNamed(lang, r.typeIdentSym)
	r.tmplArgNamed = symbolIsNamed(lang, r.tmplArgListSym)
	r.typeDescNamed = symbolIsNamed(lang, r.typeDescSym)
	r.primNamed = symbolIsNamed(lang, r.primSym)
	r.identNamed = symbolIsNamed(lang, r.identSym)
	return r, true
}

func (r *cppTemplateReconstructor) leaf(sym Symbol, named bool, start, end uint32) *Node {
	return newLeafNodeInArena(r.arena, sym, named, start, end,
		advancePointByBytes(Point{}, r.src[:start]),
		advancePointByBytes(Point{}, r.src[:end]))
}

func (r *cppTemplateReconstructor) skipWS(i uint32, end uint32) uint32 {
	for i < end {
		switch r.src[i] {
		case ' ', '\t', '\r', '\n':
			i++
		default:
			return i
		}
	}
	return i
}

func cppIsIdentStart(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func cppIsIdentByte(b byte) bool {
	return cppIsIdentStart(b) || (b >= '0' && b <= '9')
}

// readIdent returns the exclusive end of an identifier starting at i, or i if
// there is no identifier there.
func (r *cppTemplateReconstructor) readIdent(i, end uint32) uint32 {
	if i >= end || !cppIsIdentStart(r.src[i]) {
		return i
	}
	j := i + 1
	for j < end && cppIsIdentByte(r.src[j]) {
		j++
	}
	return j
}

// parseTypeId parses a C++ type-id at [i, end): a (possibly namespace-qualified)
// name optionally ending in a template argument list, a primitive keyword, or a
// bare custom type identifier. Returns the built node and the position after it.
func (r *cppTemplateReconstructor) parseTypeId(i, end uint32) (*Node, uint32, bool) {
	i = r.skipWS(i, end)
	idEnd := r.readIdent(i, end)
	if idEnd == i {
		return nil, i, false
	}
	j := r.skipWS(idEnd, end)
	// Qualified: `<ns> :: <rest-type-id>`.
	if j+1 < end && r.src[j] == ':' && r.src[j+1] == ':' {
		ns := r.leaf(r.nsIdentSym, r.nsIdentNamed, i, idEnd)
		colons := r.leaf(r.colonColonSym, symbolIsNamed(r.lang, r.colonColonSym), j, j+2)
		rest, k, ok := r.parseTypeId(j+2, end)
		if !ok {
			return nil, i, false
		}
		qual := newParentNodeInArena(r.arena, r.qualSym, r.qualNamed,
			[]*Node{ns, colons, rest},
			[]FieldID{r.scopeFID, 0, r.nameFID}, 0)
		return qual, k, true
	}
	// Template: `<name> < args >`.
	if j < end && r.src[j] == '<' {
		argList, k, ok := r.parseTemplateArgList(j, end)
		if !ok {
			return nil, i, false
		}
		ti := r.leaf(r.typeIdentSym, r.typeIdentN, i, idEnd)
		tt := newParentNodeInArena(r.arena, r.tmplTypeSym, r.tmplTypeNamed,
			[]*Node{ti, argList},
			[]FieldID{r.nameFID, r.argumentsFID}, 0)
		return tt, k, true
	}
	// Leaf: primitive keyword or custom type identifier.
	if cppIsPrimitiveTypeName(string(r.src[i:idEnd])) {
		return r.leaf(r.primSym, r.primNamed, i, idEnd), idEnd, true
	}
	return r.leaf(r.typeIdentSym, r.typeIdentN, i, idEnd), idEnd, true
}

// parseTemplateArgList parses `< type-descriptor (, type-descriptor)* >` where
// r.src[i] == '<'. It consumes exactly one `>` char to close (so a `>>` closing
// two levels is handled by the caller consuming the next `>`).
func (r *cppTemplateReconstructor) parseTemplateArgList(i, end uint32) (*Node, uint32, bool) {
	if i >= end || r.src[i] != '<' {
		return nil, i, false
	}
	children := make([]*Node, 0, 4)
	children = append(children, r.leaf(r.ltSym, symbolIsNamed(r.lang, r.ltSym), i, i+1))
	p := i + 1
	for {
		p = r.skipWS(p, end)
		if p >= end {
			return nil, i, false
		}
		if r.src[p] == '>' {
			// Empty `<>` — not a shape we reconstruct.
			return nil, i, false
		}
		td, k, ok := r.parseTypeDescriptor(p, end)
		if !ok {
			return nil, i, false
		}
		children = append(children, td)
		p = r.skipWS(k, end)
		if p >= end {
			return nil, i, false
		}
		switch r.src[p] {
		case ',':
			children = append(children, r.leaf(r.commaSym, symbolIsNamed(r.lang, r.commaSym), p, p+1))
			p++
		case '>':
			children = append(children, r.leaf(r.gtSym, symbolIsNamed(r.lang, r.gtSym), p, p+1))
			node := newParentNodeInArena(r.arena, r.tmplArgListSym, r.tmplArgNamed, children, nil, 0)
			return node, p + 1, true
		default:
			return nil, i, false
		}
	}
}

func (r *cppTemplateReconstructor) parseTypeDescriptor(i, end uint32) (*Node, uint32, bool) {
	ti, k, ok := r.parseTypeId(i, end)
	if !ok {
		return nil, i, false
	}
	td := newParentNodeInArena(r.arena, r.typeDescSym, r.typeDescNamed,
		[]*Node{ti}, []FieldID{r.typeFID}, 0)
	return td, k, true
}

// buildDeclarationChildren re-parses [start, end) as `Type<…> declarator ;` and
// returns the declaration's children ([type, declarator, ;]) when the whole
// span is consumed by that shape AND the type carries a nested template (>=2
// template levels). Returns false otherwise, leaving the caller's tree intact.
func (r *cppTemplateReconstructor) buildDeclarationChildren(start, end uint32) ([]*Node, bool) {
	if start >= end || end > uint32(len(r.src)) {
		return nil, false
	}
	typeNode, k, ok := r.parseTypeId(start, end)
	if !ok || cppCountTemplateTypes(typeNode, r.tmplTypeSym) < 2 {
		return nil, false
	}
	p := r.skipWS(k, end)
	declStart := p
	declEnd := r.readIdent(p, end)
	if declEnd == declStart {
		return nil, false
	}
	p = r.skipWS(declEnd, end)
	if p >= end || r.src[p] != ';' {
		return nil, false
	}
	semiPos := p
	if q := r.skipWS(semiPos+1, end); q < end {
		return nil, false // leftover content after `;`
	}
	declarator := r.leaf(r.identSym, r.identNamed, declStart, declEnd)
	semi := r.leaf(r.semiSym, symbolIsNamed(r.lang, r.semiSym), semiPos, semiPos+1)
	return []*Node{typeNode, declarator, semi}, true
}

// retagAsDeclaration rewrites n in place as a declaration with the given
// [type, declarator, ;] children.
func (r *cppTemplateReconstructor) retagAsDeclaration(n *Node, children []*Node) {
	n.symbol = r.declSym
	n.setNamed(r.declNamed)
	n.productionID = 0
	replaceNodeChildrenUnfielded(n, cloneNodeSliceIfArena(r.arena, children))
	setNodeChildFieldDirect(n, 0, r.typeFID)
	setNodeChildFieldDirect(n, 1, r.declaratorFID)
}

// mergeContainerExpressionSemicolonPairs scans n's children for an `expression`
// node directly followed by an `expression_statement` that is just `;`, and
// merges any that reconstruct into a single declaration node.
func (r *cppTemplateReconstructor) mergeContainerExpressionSemicolonPairs(n *Node) {
	if resultChildCount(n) < 2 {
		return
	}
	changed := false
	out := make([]*Node, 0, len(n.children))
	for i := 0; i < len(n.children); i++ {
		child := n.children[i]
		if child != nil && child.symbol == r.exprSym && i+1 < len(n.children) {
			next := n.children[i+1]
			if next != nil && next.symbol == r.exprStmtSym && cppNodeIsBareSemicolon(next, r.semiSym) {
				if declChildren, ok := r.buildDeclarationChildren(child.startByte, next.endByte); ok {
					decl := newParentNodeInArena(r.arena, r.declSym, r.declNamed,
						cloneNodeSliceIfArena(r.arena, declChildren),
						[]FieldID{r.typeFID, r.declaratorFID, 0}, 0)
					out = append(out, decl)
					i++ // consume the `;` sibling
					changed = true
					continue
				}
			}
		}
		out = append(out, child)
	}
	if changed {
		replaceNodeChildrenUnfielded(n, cloneNodeSliceIfArena(r.arena, out))
	}
}

func cppNodeIsBareSemicolon(n *Node, semiSym Symbol) bool {
	if resultChildCount(n) != 1 {
		return false
	}
	c := resultChildAt(n, 0)
	return c != nil && c.symbol == semiSym
}

// cppCountTemplateTypes counts template_type nodes in a subtree, used to require
// that a reconstructed type actually carries a nested template (>= 2 levels).
func cppCountTemplateTypes(n *Node, tmplTypeSym Symbol) int {
	if n == nil {
		return 0
	}
	count := 0
	if n.symbol == tmplTypeSym {
		count = 1
	}
	for _, child := range n.children {
		count += cppCountTemplateTypes(child, tmplTypeSym)
	}
	return count
}

// cppIsPrimitiveTypeName reports whether s is one of tree-sitter-cpp's
// primitive_type keyword tokens (grammar src/grammar.json primitive_type rule).
func cppIsPrimitiveTypeName(s string) bool {
	switch s {
	case "bool", "char", "int", "float", "double", "void",
		"size_t", "ssize_t", "ptrdiff_t", "intptr_t", "uintptr_t",
		"charptr_t", "nullptr_t", "max_align_t",
		"int8_t", "int16_t", "int32_t", "int64_t",
		"uint8_t", "uint16_t", "uint32_t", "uint64_t",
		"char8_t", "char16_t", "char32_t", "char64_t":
		return true
	default:
		return false
	}
}
