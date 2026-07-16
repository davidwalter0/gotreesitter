package gotreesitter

// normalizeCppSizeofDecltypeScope repairs blob-parity cpp-4: the shape of
// `sizeof(decltype(x)::Member)`. gt reads the parenthesized operand as an
// expression, producing
//
//	sizeof_expression(sizeof, parenthesized_expression((, qualified_identifier(
//	  decltype, ::, identifier), )))
//
// whereas the oracle reads it as a type — `decltype(x)::Member` is a
// type-dependent name that can only be a type in this position —
//
//	sizeof_expression(sizeof, (, type_descriptor(type: qualified_identifier(
//	  scope: decltype, ::, name: type_identifier)), ))
//
// Root cause is the same shared-GLR type-vs-expression dynamic-precedence
// resolution that gt gets wrong for decltype-rooted qualified names in
// sizeof's operand. The rewrite is tightly gated: it fires only on a
// `sizeof_expression(sizeof, parenthesized_expression)` whose sole parenthesized
// content is a `qualified_identifier` rooted at a `decltype` and ending in a
// plain identifier, so ordinary `sizeof(expr)` operands are untouched.
func normalizeCppSizeofDecltypeScope(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "cpp" || len(source) == 0 {
		return
	}
	typeDescSym, ok := symbolByName(lang, "type_descriptor")
	if !ok {
		return
	}
	typeIdentSym, ok := symbolByName(lang, "type_identifier")
	if !ok {
		return
	}
	qualSym, ok := symbolByName(lang, "qualified_identifier")
	if !ok {
		return
	}
	typeFID, ok := lang.FieldByName("type")
	if !ok {
		return
	}
	scopeFID, ok := lang.FieldByName("scope")
	if !ok {
		return
	}
	nameFID, ok := lang.FieldByName("name")
	if !ok {
		return
	}
	typeDescNamed := symbolIsNamed(lang, typeDescSym)
	typeIdentNamed := symbolIsNamed(lang, typeIdentSym)
	qualNamed := symbolIsNamed(lang, qualSym)

	walkResultTree(root, func(n *Node) {
		if n.Type(lang) != "sizeof_expression" || len(n.children) != 2 {
			return
		}
		sizeofKw := n.children[0]
		paren := n.children[1]
		if sizeofKw == nil || paren == nil || paren.Type(lang) != "parenthesized_expression" || len(paren.children) != 3 {
			return
		}
		lparen := paren.children[0]
		qi := paren.children[1]
		rparen := paren.children[2]
		if lparen == nil || qi == nil || rparen == nil ||
			lparen.Type(lang) != "(" || rparen.Type(lang) != ")" ||
			qi.Type(lang) != "qualified_identifier" || len(qi.children) != 3 {
			return
		}
		scope := qi.children[0]
		sep := qi.children[1]
		name := qi.children[2]
		if scope == nil || sep == nil || name == nil ||
			scope.Type(lang) != "decltype" || sep.Type(lang) != "::" || name.Type(lang) != "identifier" {
			return
		}

		arena := n.ownerArena
		nameType := newLeafNodeInArena(arena, typeIdentSym, typeIdentNamed,
			name.startByte, name.endByte, name.startPoint, name.endPoint)
		newQual := newParentNodeInArena(arena, qualSym, qualNamed,
			cloneNodeSliceIfArena(arena, []*Node{scope, sep, nameType}),
			cloneFieldIDSliceInArena(arena, []FieldID{scopeFID, 0, nameFID}), 0)
		typeDesc := newParentNodeInArena(arena, typeDescSym, typeDescNamed,
			cloneNodeSliceIfArena(arena, []*Node{newQual}),
			cloneFieldIDSliceInArena(arena, []FieldID{typeFID}), 0)

		newChildren := cloneNodeSliceIfArena(arena, []*Node{sizeofKw, lparen, typeDesc, rparen})
		n.children = newChildren
		n.clearFieldMetadata()
		if arena != nil {
			arena.clearFinalChildRefs(n)
		}
		n.productionID = 0
		populateParentNode(n, n.children)
		setNodeChildFieldDirect(n, 2, typeFID)
	})
}
