package gotreesitter

// normalizeCppOutOfLineDefaultedEmptyParamMember rewrites an out-of-line
// defaulted special-member definition with an EMPTY parameter list — e.g.
// `A::~A() = default;` or `A::A() = default;` — from gt's
// `function_definition(function_declarator(qualified_identifier, parameter_list),
// default_method_clause)` shape into the tree-sitter-cpp oracle's shape:
//
//	expression_statement(
//	  assignment_expression(
//	    left:  call_expression(function: qualified_identifier, arguments: argument_list),
//	    right: identifier "default"))
//	  ;
//
// Root cause (parse-table / GLR layer, not fixable here): with an empty `()`
// the oracle's dynamic-precedence resolution prefers the expression reading —
// `Qualified::Name()` is a valid call_expression and `= default` a valid
// assignment to the identifier `default` — over the (arguably more correct)
// out-of-line defaulted member definition. gt commits to the declaration
// reading. This is blob-parity cpp-3 (confirmed on flutter-engine
// task_runners.cc).
//
// The rewrite is tightly gated so it fires ONLY where the oracle actually
// diverges:
//   - the parameter list must be EMPTY (a non-empty `(const A&)` is not a valid
//     argument_list, so the oracle keeps those as function_definition — matches
//     gt already, must be left alone);
//   - the trailing clause must be `= default` (default_method_clause). `= delete`
//     is a delete_method_clause and the oracle keeps it a function_definition
//     (delete is a keyword, not an assignable identifier) — left alone;
//   - the definition must be exactly declarator + clause (no leading specifiers).
func normalizeCppOutOfLineDefaultedEmptyParamMember(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "cpp" || len(source) == 0 {
		return
	}
	exprStmtSym, ok := symbolByName(lang, "expression_statement")
	if !ok {
		return
	}
	assignSym, ok := symbolByName(lang, "assignment_expression")
	if !ok {
		return
	}
	callSym, ok := symbolByName(lang, "call_expression")
	if !ok {
		return
	}
	argListSym, ok := symbolByName(lang, "argument_list")
	if !ok {
		return
	}
	identSym, ok := symbolByName(lang, "identifier")
	if !ok {
		return
	}
	functionFID, ok := lang.FieldByName("function")
	if !ok {
		return
	}
	argumentsFID, ok := lang.FieldByName("arguments")
	if !ok {
		return
	}
	leftFID, ok := lang.FieldByName("left")
	if !ok {
		return
	}
	rightFID, ok := lang.FieldByName("right")
	if !ok {
		return
	}
	exprStmtNamed := symbolIsNamed(lang, exprStmtSym)
	assignNamed := symbolIsNamed(lang, assignSym)
	callNamed := symbolIsNamed(lang, callSym)
	argListNamed := symbolIsNamed(lang, argListSym)
	identNamed := symbolIsNamed(lang, identSym)

	walkResultTree(root, func(n *Node) {
		if n.Type(lang) != "function_definition" || len(n.children) != 2 {
			return
		}
		fd := n.children[0]
		dmc := n.children[1]
		if fd == nil || dmc == nil ||
			fd.Type(lang) != "function_declarator" ||
			dmc.Type(lang) != "default_method_clause" ||
			len(fd.children) != 2 || len(dmc.children) != 3 {
			return
		}
		qi := fd.children[0]
		pl := fd.children[1]
		if qi == nil || pl == nil ||
			qi.Type(lang) != "qualified_identifier" ||
			pl.Type(lang) != "parameter_list" ||
			!cppParameterListIsEmpty(pl, lang) {
			return
		}
		eq := dmc.children[0]
		defaultTok := dmc.children[1]
		semi := dmc.children[2]
		if eq == nil || defaultTok == nil || semi == nil ||
			eq.Type(lang) != "=" || defaultTok.Type(lang) != "default" || semi.Type(lang) != ";" {
			return
		}

		arena := n.ownerArena
		// parameter_list `()` -> argument_list `()` (same `(` `)` children).
		argList := newParentNodeInArena(arena, argListSym, argListNamed,
			cloneNodeSliceIfArena(arena, pl.children), nil, 0)
		callExpr := newParentNodeInArena(arena, callSym, callNamed,
			cloneNodeSliceIfArena(arena, []*Node{qi, argList}),
			cloneFieldIDSliceInArena(arena, []FieldID{functionFID, argumentsFID}), 0)
		identDefault := newLeafNodeInArena(arena, identSym, identNamed,
			defaultTok.startByte, defaultTok.endByte, defaultTok.startPoint, defaultTok.endPoint)
		assign := newParentNodeInArena(arena, assignSym, assignNamed,
			cloneNodeSliceIfArena(arena, []*Node{callExpr, eq, identDefault}),
			cloneFieldIDSliceInArena(arena, []FieldID{leftFID, 0, rightFID}), 0)

		// Retag n (the former function_definition) in place as the
		// expression_statement wrapping the assignment plus the trailing `;`.
		n.symbol = exprStmtSym
		n.setNamed(exprStmtNamed)
		n.clearFieldMetadata()
		n.productionID = 0
		replaceNodeChildrenUnfielded(n, cloneNodeSliceIfArena(arena, []*Node{assign, semi}))
	})
}

// cppParameterListIsEmpty reports whether a parameter_list has no parameter
// declarations (only its `(` and `)` punctuation).
func cppParameterListIsEmpty(pl *Node, lang *Language) bool {
	for _, child := range pl.children {
		if child == nil {
			continue
		}
		if child.isNamed() {
			return false
		}
	}
	return true
}
