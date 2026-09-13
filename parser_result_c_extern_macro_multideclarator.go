package gotreesitter

// normalizeCExternMacroMultiDeclarator repairs gt's split of the ImageMagick
// header idiom
//
//	extern MagickExport Image *Func1(...), *Func2(...), ..., *FuncN(...);
//
// where an export macro (`MagickExport`, expanding post-preprocessor to a
// visibility attribute) sits between `extern` and the real return type. Parsing
// pre-preprocessor, tree-sitter-c sees two adjacent type names (`MagickExport`
// then `Image`); the pinned oracle
// (github.com/tree-sitter/tree-sitter-c @ ae19b676) resolves this by committing
// to a single declaration whose type is `MagickExport`, wrapping the second
// name (`Image`) in an ERROR, and parsing every `*FuncK(...)` as a
// comma-separated function-pointer declarator:
//
//	declaration
//	  storage_class_specifier(extern)
//	  type_identifier(MagickExport)  [type]
//	  ERROR(identifier(Image))
//	  pointer_declarator(function_declarator(...))  [declarator]
//	  , pointer_declarator(...) ... ;
//
// gt's own recovery instead truncates the declaration at `Image` (with a
// MISSING `;`) and reparses the whole `*Func1(...),...;` run as an
// expression_statement (a comma_expression of pointer/multiplication
// expressions), which bears no structural resemblance to the oracle's
// declarator list.
//
// Rather than transliterate that unrecoverable expression parse, this pass
// reconstructs the declarator list the way the oracle sees it: it reparses the
// `*Func1(...),...;` byte range as a plain `TYPE *decl-list;` declaration (via
// the normal recovery reparse path, so the C token source is used), which the
// grammar parses cleanly into exactly the oracle's pointer_declarator list. A
// synthetic `extern int` prefix, padded with spaces to the exact byte offset of
// the first declarator, keeps every reparsed node at its real source offset, so
// the reconstructed subtree is byte-for-byte identical to the oracle's. The
// synthesized `extern`/`type_identifier`/`ERROR(Image)` header is reused from
// gt's own (correctly-lexed) truncated declaration.
//
// The pass is conservative: it fires only when the reparse produces a fully
// error-free declaration whose declarator run spans exactly the original byte
// range. No passing corpus file exhibits gt's split shape (it only arises where
// gt already diverges from the oracle), so a spurious fire cannot regress a
// byte-exact file; the error-free-reparse gate additionally leaves any shape it
// cannot cleanly reconstruct untouched.
func normalizeCExternMacroMultiDeclarator(root *Node, source []byte, p *Parser, lang *Language) {
	if root == nil || p == nil || lang == nil || lang.Name != "c" || p.skipRecoveryReparse || root.ownerArena == nil || len(source) == 0 {
		return
	}
	syms, ok := cExternMacroSymbolsForLanguage(lang)
	if !ok {
		return
	}
	walkResultTree(root, func(parent *Node) {
		cMergeExternMacroMultiDeclaratorChildren(parent, source, p, lang, syms)
	})
}

type cExternMacroSymbols struct {
	declaration         Symbol
	expressionStatement Symbol
	macroTypeSpecifier  Symbol
	storageClass        Symbol
	typeIdentifier      Symbol
	identifier          Symbol
	primitiveType       Symbol

	typeFID FieldID

	declarationNamed bool
}

func cExternMacroSymbolsForLanguage(lang *Language) (cExternMacroSymbols, bool) {
	var s cExternMacroSymbols
	var ok bool
	if s.declaration, ok = symbolByName(lang, "declaration"); !ok {
		return s, false
	}
	if s.expressionStatement, ok = symbolByName(lang, "expression_statement"); !ok {
		return s, false
	}
	if s.macroTypeSpecifier, ok = symbolByName(lang, "macro_type_specifier"); !ok {
		return s, false
	}
	if s.storageClass, ok = symbolByName(lang, "storage_class_specifier"); !ok {
		return s, false
	}
	if s.typeIdentifier, ok = symbolByName(lang, "type_identifier"); !ok {
		return s, false
	}
	if s.identifier, ok = symbolByName(lang, "identifier"); !ok {
		return s, false
	}
	if s.primitiveType, ok = symbolByName(lang, "primitive_type"); !ok {
		return s, false
	}
	if s.typeFID, ok = lang.FieldByName("type"); !ok {
		return s, false
	}
	s.declarationNamed = symbolIsNamed(lang, s.declaration)
	return s, true
}

func cMergeExternMacroMultiDeclaratorChildren(parent *Node, source []byte, p *Parser, lang *Language, syms cExternMacroSymbols) {
	childCount := resultChildCount(parent)
	if childCount < 2 {
		return
	}
	oldFieldIDs := parent.fieldIDs()
	newChildren := make([]*Node, 0, childCount)
	newFieldIDs := make([]FieldID, 0, childCount)
	changed := false
	i := 0
	for i < childCount {
		d := resultChildAt(parent, i)
		if consumed, merged := cTryMergeExternMacroRun(parent, i, childCount, source, p, lang, syms); merged != nil {
			newChildren = append(newChildren, merged)
			newFieldIDs = append(newFieldIDs, 0)
			i += consumed
			changed = true
			continue
		}
		newChildren = append(newChildren, d)
		fid := FieldID(0)
		if i < len(oldFieldIDs) {
			fid = oldFieldIDs[i]
		}
		newFieldIDs = append(newFieldIDs, fid)
		i++
	}
	if !changed {
		return
	}

	arena := parent.ownerArena
	origStartByte, origEndByte := parent.startByte, parent.endByte
	origStartPoint, origEndPoint := parent.startPoint, parent.endPoint
	parent.children = cloneNodeSliceIfArena(arena, newChildren)
	parent.setFieldMetadata(cloneFieldIDSliceInArena(arena, newFieldIDs), defaultFieldSourcesInArena(arena, newFieldIDs))
	if arena != nil {
		arena.clearFinalChildRefs(parent)
	}
	parent.productionID = 0
	populateParentNode(parent, parent.children)
	// A declaration_list / preproc body may extend past its last child; preserve
	// the parent's outer bounds (only children were merged, not removed from the
	// span).
	parent.startByte, parent.endByte = origStartByte, origEndByte
	parent.startPoint, parent.endPoint = origStartPoint, origEndPoint
}

// cExternMacroFragmentSymbol reports whether n is one of the fragment shapes gt
// scatters the `*decl-list;` / `decl-list;` run into after truncating the
// `extern MACRO Type` declaration: an expression_statement (variant 1, the
// pointer-function-pointer list), a macro_type_specifier or `;` (variant 2, a
// single non-pointer function), or another declaration (variant 3, a
// multi-declarator non-pointer list). Anything else (a preproc directive, a
// closing `}`, a comment) ends the run.
func cExternMacroFragmentSymbol(n *Node, lang *Language, syms cExternMacroSymbols) bool {
	if n == nil {
		return false
	}
	if cResultSymbolMatches(lang, n, syms.expressionStatement) ||
		cResultSymbolMatches(lang, n, syms.macroTypeSpecifier) ||
		cResultSymbolMatches(lang, n, syms.declaration) {
		return true
	}
	return n.Type(lang) == ";"
}

func cTryMergeExternMacroRun(parent *Node, i, childCount int, source []byte, p *Parser, lang *Language, syms cExternMacroSymbols) (int, *Node) {
	d := resultChildAt(parent, i)
	if d == nil || !cResultSymbolMatches(lang, d, syms.declaration) {
		return 0, nil
	}
	// d must be exactly: storage_class(extern), type_identifier(MACRO),
	// identifier(name), MISSING ";".
	if resultChildCount(d) != 4 {
		return 0, nil
	}
	sc := resultChildAt(d, 0)
	macroType := resultChildAt(d, 1)
	secondName := resultChildAt(d, 2)
	missingSemi := resultChildAt(d, 3)
	if sc == nil || macroType == nil || secondName == nil || missingSemi == nil {
		return 0, nil
	}
	if !cResultSymbolMatches(lang, sc, syms.storageClass) || string(sc.Text(source)) != "extern" {
		return 0, nil
	}
	if !cResultSymbolMatches(lang, macroType, syms.typeIdentifier) || !cResultSymbolMatches(lang, secondName, syms.identifier) {
		return 0, nil
	}
	if missingSemi.Type(lang) != ";" || missingSemi.startByte != missingSemi.endByte || !missingSemi.hasError() {
		return 0, nil
	}

	// Collect the following fragment siblings up to and including the one that
	// terminates the declaration with a `;`.
	firstFrag := resultChildAt(parent, i+1)
	if firstFrag == nil || !cExternMacroFragmentSymbol(firstFrag, lang, syms) {
		return 0, nil
	}
	declStart := int(firstFrag.startByte)
	lastIdx := -1
	for j := i + 1; j < childCount; j++ {
		frag := resultChildAt(parent, j)
		if frag == nil || !cExternMacroFragmentSymbol(frag, lang, syms) {
			break
		}
		end := int(frag.endByte)
		if end > 0 && end <= len(source) && source[end-1] == ';' {
			lastIdx = j
			break
		}
	}
	if lastIdx < 0 {
		return 0, nil
	}
	declEnd := int(resultChildAt(parent, lastIdx).endByte)
	consumed := lastIdx - i + 1

	if merged := cReparseExternMacroDeclarator(d, sc, macroType, secondName, declStart, declEnd, source, p, lang, syms); merged != nil {
		return consumed, merged
	}
	return 0, nil
}

func cReparseExternMacroDeclarator(d, sc, macroType, secondName *Node, declStart, declEnd int, source []byte, p *Parser, lang *Language, syms cExternMacroSymbols) *Node {
	const prefix = "extern int"
	if declEnd > len(source) || declStart >= declEnd {
		return nil
	}
	// Build a synthetic `extern int <padding> <declarators>;` buffer. The
	// padding fills bytes [len(prefix):declStart] with spaces then exactly R
	// newlines then C spaces, where (R, C) is the real source point at
	// declStart. This keeps every reparsed declarator node at BOTH its real
	// byte offset (total prefix length == declStart) AND its real row/column
	// (the padded newlines land the declarator run on its true line), so the
	// reconstructed subtree needs no offset rebase.
	realPt := advancePointByBytes(Point{}, source[:declStart])
	rows := int(realPt.Row)
	cols := int(realPt.Column)
	if declStart < len(prefix)+rows+cols {
		return nil
	}
	synth := make([]byte, declEnd)
	copy(synth, prefix)
	pos := len(prefix)
	for k := 0; k < declStart-len(prefix)-rows-cols; k++ {
		synth[pos] = ' '
		pos++
	}
	for k := 0; k < rows; k++ {
		synth[pos] = '\n'
		pos++
	}
	for k := 0; k < cols; k++ {
		synth[pos] = ' '
		pos++
	}
	copy(synth[declStart:], source[declStart:declEnd])

	tree, err := p.parseForRecovery(synth)
	if err != nil || tree == nil {
		return nil
	}
	defer tree.Release()
	reRoot := tree.RootNode()
	if reRoot == nil || reRoot.Type(lang) != "translation_unit" || reRoot.HasError() {
		return nil
	}
	if resultChildCount(reRoot) != 1 {
		return nil
	}
	reDecl := resultChildAt(reRoot, 0)
	if reDecl == nil || !cResultSymbolMatches(lang, reDecl, syms.declaration) || reDecl.HasError() {
		return nil
	}
	rdkCount := resultChildCount(reDecl)
	if rdkCount < 4 {
		return nil
	}
	reStorage := resultChildAt(reDecl, 0)
	rePrim := resultChildAt(reDecl, 1)
	if reStorage == nil || rePrim == nil ||
		!cResultSymbolMatches(lang, reStorage, syms.storageClass) ||
		!cResultSymbolMatches(lang, rePrim, syms.primitiveType) {
		return nil
	}
	// The declarator run must span exactly the original byte range (proves the
	// synthetic-prefix offset alignment held).
	firstDeclarator := resultChildAt(reDecl, 2)
	lastReChild := resultChildAt(reDecl, rdkCount-1)
	if firstDeclarator == nil || lastReChild == nil {
		return nil
	}
	if int(firstDeclarator.startByte) != declStart || int(lastReChild.endByte) != declEnd {
		return nil
	}
	if lastReChild.Type(lang) != ";" {
		return nil
	}

	arena := d.ownerArena
	if arena == nil {
		return nil
	}

	// Clone the reparsed declarator run (pointer_/function_declarators, commas,
	// ;) into the destination arena; offsets are already real thanks to the
	// padding. Reuse each child's field id from the reparsed declaration so the
	// `declarator` field lands on whichever declarator kind the grammar built.
	reFieldIDs := reDecl.fieldIDs()
	declaratorChildren := make([]*Node, 0, rdkCount-2)
	declaratorFieldIDs := make([]FieldID, 0, rdkCount-2)
	for j := 2; j < rdkCount; j++ {
		rc := resultChildAt(reDecl, j)
		if rc == nil {
			return nil
		}
		declaratorChildren = append(declaratorChildren, cloneTreeNodesIntoArenaWithOffset(rc, arena, nil))
		fid := FieldID(0)
		if j < len(reFieldIDs) {
			fid = reFieldIDs[j]
		}
		declaratorFieldIDs = append(declaratorFieldIDs, fid)
	}

	// ERROR(identifier(Image)) — named + extra, matching the oracle.
	errNode := newParentNodeInArena(arena, errorSymbol, true, cloneNodeSliceIfArena(arena, []*Node{secondName}), nil, 0)
	errNode.setHasError(true)
	errNode.setExtra(true)

	mergedChildren := make([]*Node, 0, 3+len(declaratorChildren))
	mergedFieldIDs := make([]FieldID, 0, 3+len(declaratorChildren))
	mergedChildren = append(mergedChildren, sc, macroType, errNode)
	mergedFieldIDs = append(mergedFieldIDs, 0, syms.typeFID, 0)
	for k, dc := range declaratorChildren {
		mergedChildren = append(mergedChildren, dc)
		mergedFieldIDs = append(mergedFieldIDs, declaratorFieldIDs[k])
	}

	return newParentNodeInArena(arena, syms.declaration, syms.declarationNamed,
		cloneNodeSliceIfArena(arena, mergedChildren),
		cloneFieldIDSliceInArena(arena, mergedFieldIDs), 0)
}
