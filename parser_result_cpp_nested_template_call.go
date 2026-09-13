package gotreesitter

// normalizeCppNestedTemplateCall repairs the CALL/expression-context sibling of
// the declaration bug fixed by normalizeCppNestedTemplateDeclaration: a call
// whose callee is a two-or-more-level-deep template closed by `>>` — e.g.
//
//	foo<bar<int>>(a, b);
//	std::make_shared<std::vector<uint8_t>>();
//	return std::vector<std::unique_ptr<fml::Mapping>>();
//
// mis-parsed as a `<`/`>` comparison chain
//
//	binary_expression( binary_expression(callee, template_function(inner…)),
//	                   parenthesized_expression(args) )
//
// instead of
//
//	call_expression( function: template_function(name, template_argument_list…),
//	                 arguments: argument_list(args) ).
//
// Root cause is the same shared-GLR dynamic-precedence mis-resolution described
// on normalizeCppNestedTemplateDeclaration: gt commits to the less-than reading
// when the outer template argument list's LEADING argument is itself a template
// (`a<b<int>>` fails, `a<x, b<int>>` parses fine). The engine-level fix would
// touch the forest scoring every language shares, so this stays a cpp-scoped
// post-parse repair.
//
// Strategy — SOURCE reconstruction (mirrors the declaration repair). A candidate
// outer binary_expression whose last child is a parenthesized_expression is
// re-read from source with the same recursive type-id recognizer: the callee is
// rebuilt as a template_function (qualified when namespace-scoped) and the
// argument list is lifted verbatim from the already-parsed parenthesized_
// expression (its argument sub-trees are correct; only the enclosing shape was
// wrong). It fires ONLY when the whole span cleanly tokenizes as
// `<callee-type-id with a nested template> ( <args> )` with no leftover AND the
// callee carries a genuinely nested (>= 2 level) template — a plain comparison
// such as `a < b > (c)` (single level, no nesting) is left untouched.
func normalizeCppNestedTemplateCall(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "cpp" || len(source) == 0 {
		return
	}
	base, ok := newCppTemplateReconstructor(source, lang, root.ownerArena)
	if !ok {
		return
	}
	r, ok := newCppCallReconstructor(base, lang)
	if !ok {
		return
	}
	changed := false
	walkResultTree(root, func(n *Node) {
		// The mis-parse reads `<callee> < … > ( args )` as a comparison whose
		// rightmost leaf is the call's argument list wrapped in a
		// parenthesized_expression. Depending on how deeply the callee's nested
		// template mangled, that parenthesized_expression is either n's direct
		// last child (foo<bar<int>>(a)) or the rightmost descendant of a deeper
		// comparison chain (std::vector<std::unique_ptr<T>>()).
		if resultChildCount(n) < 2 || !cResultSymbolMatches(lang, n, r.binExprSym) {
			return
		}
		paren := r.rightmostParenthesizedAtEnd(n, n.endByte)
		if paren == nil {
			return
		}
		r.baseOff, r.basePt = n.startByte, n.startPoint
		function, argList, ok := r.buildCallChildren(n.startByte, n.endByte, paren)
		if !ok {
			return
		}
		r.retagAsCall(n, function, argList)
		changed = true
	})
	if changed {
		// Dropping the empty-args MISSING-identifier phantom can clear a
		// hasError that error-recovery propagated up the enclosing statement;
		// recompute the flag so ancestors no longer claim a spurious error.
		recomputeCppSubtreeHasError(root)
	}
}

type cppCallReconstructor struct {
	*cppTemplateReconstructor

	tmplFuncSym   Symbol
	callExprSym   Symbol
	argListSym    Symbol
	binExprSym    Symbol
	parenExprSym  Symbol
	commaExprSym  Symbol
	openParenSym  Symbol
	closeParenSym Symbol

	tmplFuncNamed bool
	callExprNamed bool
	argListNamed  bool

	functionFID FieldID
}

func newCppCallReconstructor(base *cppTemplateReconstructor, lang *Language) (*cppCallReconstructor, bool) {
	if base == nil {
		return nil, false
	}
	r := &cppCallReconstructor{cppTemplateReconstructor: base}
	syms := []struct {
		name string
		dst  *Symbol
	}{
		{"template_function", &r.tmplFuncSym},
		{"call_expression", &r.callExprSym},
		{"argument_list", &r.argListSym},
		{"binary_expression", &r.binExprSym},
		{"parenthesized_expression", &r.parenExprSym},
		{"comma_expression", &r.commaExprSym},
		{"(", &r.openParenSym},
		{")", &r.closeParenSym},
	}
	for _, s := range syms {
		sym, ok := symbolByName(lang, s.name)
		if !ok {
			return nil, false
		}
		*s.dst = sym
	}
	fid, ok := lang.FieldByName("function")
	if !ok {
		return nil, false
	}
	r.functionFID = fid
	r.tmplFuncNamed = symbolIsNamed(lang, r.tmplFuncSym)
	r.callExprNamed = symbolIsNamed(lang, r.callExprSym)
	r.argListNamed = symbolIsNamed(lang, r.argListSym)
	return r, true
}

// rightmostParenthesizedAtEnd descends the rightmost-child spine of n and
// returns the first parenthesized_expression whose end coincides with end (the
// call's trailing `( args )`). It bounds the descent to a small depth so a
// pathological tree cannot make it loop unboundedly.
func (r *cppCallReconstructor) rightmostParenthesizedAtEnd(n *Node, end uint32) *Node {
	cur := n
	for depth := 0; cur != nil && depth < 8; depth++ {
		cc := resultChildCount(cur)
		if cc == 0 {
			return nil
		}
		last := resultChildAt(cur, cc-1)
		if last == nil || last.endByte != end {
			return nil
		}
		if cResultSymbolMatches(r.lang, last, r.parenExprSym) {
			return last
		}
		cur = last
	}
	return nil
}

// buildCallChildren re-reads [start, end) as `<callee-type-id> ( <args> )` and
// returns the reconstructed (function, argument_list) pair when the whole span
// is consumed by that shape, the callee carries a nested (>= 2 level) template,
// and the trailing `( … )` matches the supplied parenthesized_expression. The
// argument list is lifted from that parenthesized_expression so already-parsed
// argument expressions are preserved byte-exact.
func (r *cppCallReconstructor) buildCallChildren(start, end uint32, paren *Node) (*Node, *Node, bool) {
	if start >= end || end > uint32(len(r.src)) || paren == nil {
		return nil, nil, false
	}
	callee, k, ok := r.parseCallCallee(start, end)
	if !ok || cppCountTemplateTypes(callee, r.tmplTypeSym) < 1 {
		// Require a genuinely nested template: the callee's own
		// template_argument_list must contain at least one template_type
		// (the second level). A single-level `a<b>(c)` has zero template_type
		// nodes and is left as a comparison.
		return nil, nil, false
	}
	p := r.skipWS(k, end)
	if p != paren.startByte || paren.endByte != end {
		return nil, nil, false
	}
	if p >= end || r.src[p] != '(' {
		return nil, nil, false
	}
	argList, ok := r.buildArgumentListFromParenthesized(paren)
	if !ok {
		return nil, nil, false
	}
	return callee, argList, true
}

// parseCallCallee parses a (possibly namespace-qualified) callee ending in a
// template argument list, building the outermost template as a template_function
// (name + arguments) — the call-context analogue of parseTypeId's template_type.
func (r *cppCallReconstructor) parseCallCallee(i, end uint32) (*Node, uint32, bool) {
	i = r.skipWS(i, end)
	idEnd := r.readIdent(i, end)
	if idEnd == i {
		return nil, i, false
	}
	j := r.skipWS(idEnd, end)
	// Qualified: `<ns> :: <rest-callee>`.
	if j+1 < end && r.src[j] == ':' && r.src[j+1] == ':' {
		ns := r.leaf(r.nsIdentSym, r.nsIdentNamed, i, idEnd)
		colons := r.leaf(r.colonColonSym, symbolIsNamed(r.lang, r.colonColonSym), j, j+2)
		rest, m, ok := r.parseCallCallee(j+2, end)
		if !ok {
			return nil, i, false
		}
		qual := newParentNodeInArena(r.arena, r.qualSym, r.qualNamed,
			[]*Node{ns, colons, rest},
			[]FieldID{r.scopeFID, 0, r.nameFID}, 0)
		return qual, m, true
	}
	// Template: `<name> < args >` → template_function.
	if j < end && r.src[j] == '<' {
		argList, m, ok := r.parseTemplateArgList(j, end)
		if !ok {
			return nil, i, false
		}
		name := r.leaf(r.identSym, r.identNamed, i, idEnd)
		tf := newParentNodeInArena(r.arena, r.tmplFuncSym, r.tmplFuncNamed,
			[]*Node{name, argList},
			[]FieldID{r.nameFID, r.argumentsFID}, 0)
		return tf, m, true
	}
	// A callee with no template argument list is not this bug.
	return nil, i, false
}

// buildArgumentListFromParenthesized converts a parenthesized_expression that
// wraps the call's arguments into an argument_list, flattening the (right-
// nested) comma_expression into positional argument children and reusing the
// already-parsed `(`, `)`, comma and argument sub-trees verbatim.
func (r *cppCallReconstructor) buildArgumentListFromParenthesized(paren *Node) (*Node, bool) {
	cc := resultChildCount(paren)
	if cc < 2 {
		return nil, false
	}
	open := resultChildAt(paren, 0)
	closeP := resultChildAt(paren, cc-1)
	if open == nil || closeP == nil ||
		!cResultSymbolMatches(r.lang, open, r.openParenSym) ||
		!cResultSymbolMatches(r.lang, closeP, r.closeParenSym) {
		return nil, false
	}
	children := make([]*Node, 0, cc+2)
	children = append(children, open)
	for i := 1; i < cc-1; i++ {
		mid := resultChildAt(paren, i)
		if mid == nil {
			return nil, false
		}
		// A zero-width child is an error-recovery phantom: for the empty-args
		// case `foo<bar<int>>()` gt fills the parenthesized_expression's
		// mandatory inner expression with a MISSING identifier. The oracle's
		// argument_list has no such node, so drop zero-width sub-nodes.
		for _, arg := range r.flattenCommaExpression(mid) {
			if arg == nil || arg.startByte == arg.endByte {
				continue
			}
			children = append(children, arg)
		}
	}
	children = append(children, closeP)
	argList := newParentNodeInArena(r.arena, r.argListSym, r.argListNamed,
		cloneNodeSliceIfArena(r.arena, children), nil, 0)
	return argList, true
}

// flattenCommaExpression turns `a, b, c` (comma_expression(a, `,`,
// comma_expression(b, `,`, c))) into the flat sequence [a, `,`, b, `,`, c],
// reusing every existing sub-node; a non-comma node is returned as a singleton.
func (r *cppCallReconstructor) flattenCommaExpression(n *Node) []*Node {
	if n == nil {
		return nil
	}
	if !cResultSymbolMatches(r.lang, n, r.commaExprSym) {
		return []*Node{n}
	}
	cc := resultChildCount(n)
	out := make([]*Node, 0, cc)
	for i := 0; i < cc; i++ {
		out = append(out, r.flattenCommaExpression(resultChildAt(n, i))...)
	}
	return out
}

// retagAsCall rewrites n in place as a call_expression with the given
// (function, argument_list) children.
func (r *cppCallReconstructor) retagAsCall(n *Node, function, argList *Node) {
	n.symbol = r.callExprSym
	n.setNamed(r.callExprNamed)
	n.productionID = 0
	replaceNodeChildrenUnfielded(n, cloneNodeSliceIfArena(r.arena, []*Node{function, argList}))
	setNodeChildFieldDirect(n, 0, r.functionFID)
	setNodeChildFieldDirect(n, 1, r.argumentsFID)
}
