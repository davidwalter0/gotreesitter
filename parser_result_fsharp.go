package gotreesitter

func normalizeFSharpCompatibility(root *Node, source []byte, lang *Language) {
	normalizeFSharpSimpleLongIdentifiers(root, lang)
	normalizeFSharpDottedLongIdentifierOrOps(root, lang)
	normalizeFSharpFunctionDeclarationLeft(root, source, lang)
}

func normalizeFSharpDottedLongIdentifierOrOps(root *Node, lang *Language) {
	if root == nil || lang == nil || root.HasError() {
		return
	}
	longIdentOrOpSym, ok := lang.symbolByNameAndNamed("long_identifier_or_op", true)
	if !ok {
		longIdentOrOpSym, ok = symbolByName(lang, "long_identifier_or_op")
	}
	if !ok {
		return
	}
	longIdentSym, ok := lang.symbolByNameAndNamed("long_identifier", true)
	if !ok {
		longIdentSym, ok = symbolByName(lang, "long_identifier")
	}
	if !ok {
		return
	}
	identSym, ok := lang.symbolByNameAndNamed("identifier", true)
	if !ok {
		identSym, ok = symbolByName(lang, "identifier")
	}
	if !ok {
		return
	}
	dotSym, ok := lang.symbolByNameAndNamed(".", false)
	if !ok {
		dotSym, ok = symbolByName(lang, ".")
	}
	if !ok {
		return
	}
	dotExpressionSym, ok := lang.symbolByNameAndNamed("dot_expression", true)
	if !ok {
		dotExpressionSym, ok = symbolByName(lang, "dot_expression")
	}
	if !ok {
		return
	}

	var walk func(*Node, bool)
	walk = func(n *Node, isDotExpressionReceiver bool) {
		if n == nil {
			return
		}
		if n.symbol == longIdentOrOpSym {
			normalizeFSharpDottedLongIdentifierOrOpNode(n, longIdentOrOpSym, longIdentSym, identSym, dotSym, isDotExpressionReceiver, lang)
		}
		for i := 0; i < resultChildCount(n); i++ {
			child := resultChildAt(n, i)
			walk(child, n.symbol == dotExpressionSym && fsharpIsFirstSemanticChild(n, i))
		}
	}
	walk(root, false)
}

func fsharpIsFirstSemanticChild(n *Node, childIndex int) bool {
	if n == nil || childIndex < 0 || childIndex >= resultChildCount(n) {
		return false
	}
	child := resultChildAt(n, childIndex)
	if child == nil || child.IsExtra() || child.IsMissing() {
		return false
	}
	for i := 0; i < childIndex; i++ {
		prev := resultChildAt(n, i)
		if prev != nil && !prev.IsExtra() && !prev.IsMissing() {
			return false
		}
	}
	return true
}

func normalizeFSharpDottedLongIdentifierOrOpNode(n *Node, longIdentOrOpSym, longIdentSym, identSym, dotSym Symbol, isDotExpressionReceiver bool, lang *Language) {
	if n == nil || n.symbol != longIdentOrOpSym || n.HasError() || n.IsMissing() || n.IsExtra() {
		return
	}
	if isDotExpressionReceiver {
		normalizeFSharpDottedReceiverLongIdentifierOrOpNode(n, longIdentSym, identSym, dotSym, lang)
		return
	}
	children, ok := fsharpFlattenDottedLongIdentifierChildren(n, longIdentSym, identSym, dotSym)
	if !ok {
		return
	}

	longIdent := newParentNodeInArena(
		n.ownerArena,
		longIdentSym,
		symbolIsNamed(lang, longIdentSym),
		cloneNodeSliceIfArena(n.ownerArena, children),
		nil,
		0,
	)
	replaceNodeChildrenUnfielded(n, cloneNodeSliceIfArena(n.ownerArena, []*Node{longIdent}))
}

func normalizeFSharpDottedReceiverLongIdentifierOrOpNode(n *Node, longIdentSym, identSym, dotSym Symbol, lang *Language) {
	children, ok := fsharpFlattenDottedLongIdentifierChildren(n, longIdentSym, identSym, dotSym)
	if !ok || len(children) < 3 {
		return
	}
	prefixChildren := children[:len(children)-2]
	prefix := newParentNodeInArena(
		n.ownerArena,
		longIdentSym,
		symbolIsNamed(lang, longIdentSym),
		cloneNodeSliceIfArena(n.ownerArena, prefixChildren),
		nil,
		0,
	)
	replacement := []*Node{prefix, children[len(children)-2], children[len(children)-1]}
	replaceNodeChildrenUnfielded(n, cloneNodeSliceIfArena(n.ownerArena, replacement))
}

func fsharpFlattenDottedLongIdentifierChildren(n *Node, longIdentSym, identSym, dotSym Symbol) ([]*Node, bool) {
	if n == nil {
		return nil, false
	}
	var children []*Node
	for i := 0; i < resultChildCount(n); i++ {
		child := resultChildAt(n, i)
		if child == nil || child.HasError() || child.IsMissing() || child.IsExtra() {
			return nil, false
		}
		switch child.symbol {
		case identSym, dotSym:
			children = append(children, child)
		case longIdentSym:
			for j := 0; j < resultChildCount(child); j++ {
				inner := resultChildAt(child, j)
				if inner == nil || inner.HasError() || inner.IsMissing() || inner.IsExtra() {
					return nil, false
				}
				children = append(children, inner)
			}
		default:
			return nil, false
		}
	}
	if len(children) < 3 || len(children)%2 == 0 {
		return nil, false
	}
	for i, child := range children {
		if i%2 == 0 {
			if child.symbol != identSym {
				return nil, false
			}
		} else if child.symbol != dotSym {
			return nil, false
		}
	}
	if children[0].StartByte() != n.StartByte() || children[len(children)-1].EndByte() != n.EndByte() {
		return nil, false
	}
	return children, true
}

func normalizeFSharpFunctionDeclarationLeft(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || root.HasError() {
		return
	}
	valueDeclLeftSym, ok := lang.symbolByNameAndNamed("value_declaration_left", true)
	if !ok {
		valueDeclLeftSym, ok = symbolByName(lang, "value_declaration_left")
	}
	if !ok {
		return
	}
	functionDeclLeftSym, ok := lang.symbolByNameAndNamed("function_declaration_left", true)
	if !ok {
		functionDeclLeftSym, ok = symbolByName(lang, "function_declaration_left")
	}
	if !ok {
		return
	}
	identifierPatternSym, ok := lang.symbolByNameAndNamed("identifier_pattern", true)
	if !ok {
		identifierPatternSym, ok = symbolByName(lang, "identifier_pattern")
	}
	if !ok {
		return
	}
	identifierSym, ok := lang.symbolByNameAndNamed("identifier", true)
	if !ok {
		identifierSym, ok = symbolByName(lang, "identifier")
	}
	if !ok {
		return
	}
	argumentPatternsSym, ok := lang.symbolByNameAndNamed("argument_patterns", true)
	if !ok {
		argumentPatternsSym, ok = symbolByName(lang, "argument_patterns")
	}
	if !ok {
		return
	}
	longIdentOrOpSym, ok := lang.symbolByNameAndNamed("long_identifier_or_op", true)
	if !ok {
		longIdentOrOpSym, ok = symbolByName(lang, "long_identifier_or_op")
	}
	if !ok {
		return
	}
	parenPatternSym, ok := lang.symbolByNameAndNamed("paren_pattern", true)
	if !ok {
		parenPatternSym, ok = symbolByName(lang, "paren_pattern")
	}
	if !ok {
		return
	}

	walkResultTree(root, func(n *Node) {
		normalizeFSharpFunctionDeclarationLeftNode(
			n,
			source,
			lang,
			valueDeclLeftSym,
			functionDeclLeftSym,
			identifierPatternSym,
			identifierSym,
			argumentPatternsSym,
			longIdentOrOpSym,
			parenPatternSym,
		)
	})
}

func normalizeFSharpFunctionDeclarationLeftNode(n *Node, source []byte, lang *Language, valueDeclLeftSym, functionDeclLeftSym, identifierPatternSym, identifierSym, argumentPatternsSym, longIdentOrOpSym, parenPatternSym Symbol) {
	if n == nil || n.symbol != valueDeclLeftSym || resultChildCount(n) != 1 || n.HasError() || n.IsMissing() || n.IsExtra() {
		return
	}
	pattern := resultChildAt(n, 0)
	if pattern == nil ||
		pattern.symbol != identifierPatternSym ||
		resultChildCount(pattern) < 2 ||
		pattern.HasError() ||
		pattern.IsMissing() ||
		pattern.IsExtra() ||
		pattern.StartByte() != n.StartByte() ||
		pattern.EndByte() != n.EndByte() {
		return
	}
	nameStart, nameEnd, argsStart, ok := fsharpCollapsedFunctionDeclarationLeftSplit(source, pattern.StartByte(), pattern.EndByte())
	if !ok || nameStart != pattern.StartByte() || argsStart <= nameEnd || argsStart >= pattern.EndByte() {
		return
	}
	nameNode := resultChildAt(pattern, 0)
	if nameNode == nil ||
		nameNode.symbol != longIdentOrOpSym ||
		nameNode.StartByte() != nameStart ||
		nameNode.EndByte() != nameEnd ||
		nameNode.HasError() ||
		nameNode.IsMissing() ||
		nameNode.IsExtra() {
		return
	}
	var argsChildren []*Node
	for i := 1; i < resultChildCount(pattern); i++ {
		arg := resultChildAt(pattern, i)
		if arg == nil ||
			arg.symbol != parenPatternSym ||
			arg.HasError() ||
			arg.IsMissing() ||
			arg.IsExtra() ||
			resultChildCount(arg) == 0 {
			return
		}
		for j := 0; j < resultChildCount(arg); j++ {
			argsChildren = append(argsChildren, resultChildAt(arg, j))
		}
	}
	if len(argsChildren) == 0 || argsChildren[0] == nil || argsChildren[0].StartByte() != argsStart {
		return
	}

	identifierStartPoint := advancePointByBytes(Point{}, source[:nameStart])
	identifierEndPoint := advancePointByBytes(identifierStartPoint, source[nameStart:nameEnd])
	identifier := newLeafNodeInArena(n.ownerArena, identifierSym, symbolIsNamed(lang, identifierSym), nameStart, nameEnd, identifierStartPoint, identifierEndPoint)
	arguments := newParentNodeInArena(
		n.ownerArena,
		argumentPatternsSym,
		symbolIsNamed(lang, argumentPatternsSym),
		cloneNodeSliceIfArena(n.ownerArena, argsChildren),
		nil,
		0,
	)

	n.symbol = functionDeclLeftSym
	n.productionID = 0
	n.setNamed(symbolIsNamed(lang, functionDeclLeftSym))
	replaceNodeChildrenUnfielded(n, cloneNodeSliceIfArena(n.ownerArena, []*Node{identifier, arguments}))
}

func fsharpCollapsedFunctionDeclarationLeftSplit(source []byte, start, end uint32) (uint32, uint32, uint32, bool) {
	if start >= end || int(end) > len(source) {
		return 0, 0, 0, false
	}
	i := start
	if !fsharpSimpleIdentifierStart(source[i]) {
		return 0, 0, 0, false
	}
	i++
	for i < end && fsharpSimpleIdentifierContinue(source[i]) {
		i++
	}
	nameEnd := i
	for i < end && (source[i] == ' ' || source[i] == '\t') {
		i++
	}
	if i >= end || source[i] != '(' {
		return 0, 0, 0, false
	}
	return start, nameEnd, i, true
}

func fsharpSimpleIdentifierStart(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '_'
}

func fsharpSimpleIdentifierContinue(b byte) bool {
	return fsharpSimpleIdentifierStart(b) || (b >= '0' && b <= '9') || b == '\''
}

func normalizeFSharpSimpleLongIdentifiers(root *Node, lang *Language) {
	if root == nil || lang == nil || root.HasError() {
		return
	}
	longIdentSym, ok := lang.symbolByNameAndNamed("long_identifier", true)
	if !ok {
		longIdentSym, ok = symbolByName(lang, "long_identifier")
	}
	if !ok {
		return
	}
	identSym, ok := lang.symbolByNameAndNamed("identifier", true)
	if !ok {
		identSym, ok = symbolByName(lang, "identifier")
	}
	if !ok {
		return
	}
	longIdentOrOpSym, ok := lang.symbolByNameAndNamed("long_identifier_or_op", true)
	if !ok {
		longIdentOrOpSym, ok = symbolByName(lang, "long_identifier_or_op")
	}
	if !ok {
		return
	}
	dotSym, hasDotSym := lang.symbolByNameAndNamed(".", false)
	if !hasDotSym {
		dotSym, hasDotSym = symbolByName(lang, ".")
	}

	normalizeFSharpSimpleLongIdentifiersInNode(root, longIdentSym, identSym, longIdentOrOpSym, dotSym, hasDotSym, root.symbol == longIdentOrOpSym)
}

func normalizeFSharpSimpleLongIdentifiersInNode(n *Node, longIdentSym, identSym, longIdentOrOpSym, dotSym Symbol, hasDotSym bool, inLongIdentifierOrOp bool) {
	if n == nil {
		return
	}
	childCount := resultChildCount(n)
	for i := 0; i < childCount; i++ {
		child := resultChildAt(n, i)
		childInLongIdentifierOrOp := inLongIdentifierOrOp || (child != nil && child.symbol == longIdentOrOpSym)
		if inLongIdentifierOrOp && !fsharpLongIdentifierHasDotSibling(n, i, dotSym, hasDotSym) && fsharpSimpleLongIdentifier(child, longIdentSym, identSym) {
			replacement := resultChildAt(child, 0)
			replaceChildRangeWithSingleNode(n, i, i+1, replacement)
			child = replacement
		}
		normalizeFSharpSimpleLongIdentifiersInNode(child, longIdentSym, identSym, longIdentOrOpSym, dotSym, hasDotSym, childInLongIdentifierOrOp)
	}
}

func fsharpLongIdentifierHasDotSibling(parent *Node, childIndex int, dotSym Symbol, hasDotSym bool) bool {
	if parent == nil || !hasDotSym {
		return false
	}
	var prev, next *Node
	if childIndex > 0 {
		prev = resultChildAt(parent, childIndex-1)
	}
	if childIndex+1 < resultChildCount(parent) {
		next = resultChildAt(parent, childIndex+1)
	}
	return (prev != nil && prev.symbol == dotSym) || (next != nil && next.symbol == dotSym)
}

func fsharpSimpleLongIdentifier(n *Node, longIdentSym, identSym Symbol) bool {
	if n == nil || n.symbol != longIdentSym || resultChildCount(n) != 1 || n.HasError() || n.IsMissing() || n.IsExtra() {
		return false
	}
	child := resultChildAt(n, 0)
	return child != nil &&
		child.symbol == identSym &&
		child.StartByte() == n.StartByte() &&
		child.EndByte() == n.EndByte() &&
		!child.HasError() &&
		!child.IsMissing() &&
		!child.IsExtra()
}
