package gotreesitter

// normalizeCppInlineErrorReturnType repairs the parse divergence where a
// `function_definition` whose return type is a user/typedef'd (non-primitive)
// identifier AND which is preceded by a leading specifier (`inline`, a
// `type_qualifier`, an attribute, …) leaves the return-type identifier as a
// bare `ERROR` leaf instead of a `type_identifier`.
//
// Root cause (parse-table / GLR layer, not fixable here): with a leading
// storage-class specifier the parser reaches the declaration-specifier state
// via a path where a following bare identifier is no longer a valid
// type-specifier shift, so the identifier is dropped into error recovery. The
// oracle (tree-sitter-cpp) recovers it as the `function_definition`'s `type`
// child; gt leaves it an unclassified ERROR leaf. `inline int Foo(...)`
// (primitive return type) and `DlScalar Foo(...)` (custom type, no leading
// specifier) both parse cleanly — only the specifier + custom-type combination
// trips it.
//
// Detection is tight: the ERROR must be a childless leaf whose text is a single
// C++ identifier, it must sit immediately before the `function_declarator`, and
// at least one earlier sibling must be a leading declaration specifier. That
// signature only ever arises from this recovery gap, so retagging the ERROR to
// `type_identifier` (with the canonical `type` field) restores oracle parity
// without touching genuine multi-token error regions.
func normalizeCppInlineErrorReturnType(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "cpp" || len(source) == 0 {
		return
	}
	typeIdentSym, ok := symbolByName(lang, "type_identifier")
	if !ok {
		return
	}
	typeFID, hasTypeField := lang.FieldByName("type")
	typeIdentNamed := symbolIsNamed(lang, typeIdentSym)

	changed := false
	walkResultTree(root, func(n *Node) {
		if n.Type(lang) != "function_definition" || len(n.children) < 3 {
			return
		}
		sawLeadingSpecifier := false
		for i := 0; i < len(n.children); i++ {
			child := n.children[i]
			if child == nil {
				continue
			}
			if child.Type(lang) == "ERROR" &&
				resultChildCount(child) == 0 &&
				i > 0 && sawLeadingSpecifier &&
				i+1 < len(n.children) &&
				n.children[i+1] != nil &&
				n.children[i+1].Type(lang) == "function_declarator" &&
				isCppSingleIdentifier(child.Text(source)) {
				typeIdent := newLeafNodeInArena(
					n.ownerArena,
					typeIdentSym,
					typeIdentNamed,
					child.startByte,
					child.endByte,
					child.startPoint,
					child.endPoint,
				)
				n.children[i] = typeIdent
				setNodeParentLink(typeIdent, n, i)
				if hasTypeField {
					setNodeChildFieldDirect(n, i, typeFID)
				}
				changed = true
				return
			}
			if cppIsLeadingDeclarationSpecifier(child.Type(lang)) {
				sawLeadingSpecifier = true
			}
		}
	})
	if changed {
		recomputeCppSubtreeHasError(root)
	}
}

// cppIsLeadingDeclarationSpecifier reports whether a node type is a
// declaration specifier that can legally precede a function's return type
// (and thereby trigger the inline-return-type recovery gap above).
func cppIsLeadingDeclarationSpecifier(nodeType string) bool {
	switch nodeType {
	case "storage_class_specifier",
		"type_qualifier",
		"attribute_specifier",
		"attribute_declaration",
		"virtual_function_specifier",
		"explicit_function_specifier",
		"ms_declspec_modifier":
		return true
	default:
		return false
	}
}

// isCppSingleIdentifier reports whether s is exactly one C++ identifier token
// (no embedded whitespace, operators, or qualifiers).
func isCppSingleIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') {
			continue
		}
		if b >= '0' && b <= '9' {
			if i == 0 {
				return false
			}
			continue
		}
		return false
	}
	return true
}

// recomputeCppSubtreeHasError recomputes the hasError flag bottom-up for every
// interior node under (and including) n, so that retagging an ERROR leaf away
// clears the stale error bit off all its ancestors up to the root. Leaf flags
// are authoritative from the parse and left untouched; an interior node has an
// error iff it is itself an ERROR/MISSING node or any descendant does.
func recomputeCppSubtreeHasError(n *Node) bool {
	if n == nil {
		return false
	}
	cc := resultChildCount(n)
	if cc == 0 {
		return n.HasError()
	}
	hasErr := n.symbol == errorSymbol || n.isMissing()
	for i := 0; i < cc; i++ {
		if recomputeCppSubtreeHasError(resultChildAt(n, i)) {
			hasErr = true
		}
	}
	n.setHasError(hasErr)
	return hasErr
}
