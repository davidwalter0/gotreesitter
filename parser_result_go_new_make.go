package gotreesitter

// normalizeGoNewMakeTypeArgument retags the sole/first argument of a
// new(...)/make(...) builtin call as a type node when gt's parse tables
// mis-resolve it as a plain expression.
//
// Root cause (parse-table layer, not fixable here): tree-sitter-go's own
// grammar carries a dedicated call_expression alternative for a literal
// `new`/`make` function token (see grammargen/go_grammar.go's
// special_argument_list, aliased to "argument_list"), whose first slot must
// reduce via the `_type` rule rather than `_expression`. That rule's
// `_type_identifier` alternative carries a negative dynamic precedence
// (PrecDynamic(-1)) relative to the ordinary expression `identifier`
// production, so wherever a bare identifier or two-level package-qualified
// name is syntactically ambiguous between the "special" (type-argument) and
// "generic" (expression-argument) call_expression alternatives, gt's
// resolution picks the generic argument_list + expression interpretation
// instead of the special_argument_list + type interpretation the oracle
// produces. The fix lives here, in the established post-parse normalization
// layer — mirrors normalizeCPointerAssignmentInversion's rationale for a
// parse-table-level ambiguity that is only cleanly resolved after the fact.
//
// Detection is conservative and purely syntactic, matching what the
// upstream grammar itself does (tree-sitter has no semantic/scoping
// awareness): a call_expression whose function child is the literal token
// "new" or "make" always takes the type-argument production, even when
// "new"/"make" was locally shadowed as an ordinary identifier or function —
// the oracle can't tell the difference either, so matching its (syntactic)
// behavior is the correctness bar here, not "true" semantic resolution.
// Only argument shapes that are genuinely ambiguous with a type are
// retagged:
//   - a bare identifier (e.g. `new(T)`) becomes type_identifier;
//   - a two-level selector expression (`pkg.Type`, i.e.
//     identifier "." field_identifier) becomes the equivalent qualified_type
//     (package_identifier "." type_identifier).
//
// Any other argument shape — already-unambiguous type forms like
// slice_type/map_type/pointer_type/channel_type, deeper selector chains, or
// genuine non-type expressions (make's trailing len/cap arguments, or a
// shadowed new/make's real expression argument) — is left untouched.
func normalizeGoNewMakeTypeArgument(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "go" || len(source) == 0 {
		return
	}
	syms, ok := goNewMakeSymbolsForLanguage(lang)
	if !ok {
		return
	}
	walkResultTreePostorder(root, func(n *Node) {
		normalizeGoNewMakeCallArgument(n, source, lang, syms)
	})
}

type goNewMakeSymbols struct {
	callExpression Symbol
	argumentList   Symbol
	identifier     Symbol
	typeIdentifier Symbol
	selectorExpr   Symbol
	qualifiedType  Symbol
	packageIdent   Symbol
	fieldIdent     Symbol
	packageFID     FieldID
	nameFID        FieldID
}

func goNewMakeSymbolsForLanguage(lang *Language) (goNewMakeSymbols, bool) {
	var syms goNewMakeSymbols
	var ok bool
	if syms.callExpression, ok = symbolByName(lang, "call_expression"); !ok {
		return syms, false
	}
	if syms.argumentList, ok = symbolByName(lang, "argument_list"); !ok {
		return syms, false
	}
	if syms.identifier, ok = symbolByName(lang, "identifier"); !ok {
		return syms, false
	}
	if syms.typeIdentifier, ok = symbolByName(lang, "type_identifier"); !ok {
		return syms, false
	}
	if syms.selectorExpr, ok = symbolByName(lang, "selector_expression"); !ok {
		return syms, false
	}
	if syms.qualifiedType, ok = symbolByName(lang, "qualified_type"); !ok {
		return syms, false
	}
	if syms.packageIdent, ok = symbolByName(lang, "package_identifier"); !ok {
		return syms, false
	}
	if syms.fieldIdent, ok = symbolByName(lang, "field_identifier"); !ok {
		return syms, false
	}
	// Field IDs are optional/best-effort: qualified_type is still built
	// (unfielded) even if a future grammar revision drops these field names.
	syms.packageFID, _ = lang.FieldByName("package")
	syms.nameFID, _ = lang.FieldByName("name")
	return syms, true
}

func normalizeGoNewMakeCallArgument(n *Node, source []byte, lang *Language, syms goNewMakeSymbols) {
	if !cResultSymbolMatches(lang, n, syms.callExpression) || resultChildCount(n) != 2 {
		return
	}
	fn := resultChildAt(n, 0)
	args := resultChildAt(n, 1)
	if fn == nil || args == nil ||
		!cResultSymbolMatches(lang, fn, syms.identifier) ||
		!cResultSymbolMatches(lang, args, syms.argumentList) {
		return
	}
	switch fn.Text(source) {
	case "new", "make":
	default:
		return
	}
	goRetagFirstArgumentAsType(args, lang, syms)
}

func goRetagFirstArgumentAsType(argList *Node, lang *Language, syms goNewMakeSymbols) {
	children := resultChildSliceForMutation(argList)
	if len(children) < 3 {
		return
	}
	retagged := goRetagAsTypeArgument(children[1], lang, syms)
	if retagged == nil {
		return
	}
	out := make([]*Node, len(children))
	copy(out, children)
	out[1] = retagged
	replaceNodeChildrenUnfielded(argList, cloneNodeSliceIfArena(argList.ownerArena, out))
}

func goRetagAsTypeArgument(arg *Node, lang *Language, syms goNewMakeSymbols) *Node {
	if arg == nil {
		return nil
	}
	arena := arg.ownerArena
	switch {
	case cResultSymbolMatches(lang, arg, syms.identifier):
		return newLeafNodeInArena(arena, syms.typeIdentifier, symbolIsNamed(lang, syms.typeIdentifier), arg.startByte, arg.endByte, arg.startPoint, arg.endPoint)
	case cResultSymbolMatches(lang, arg, syms.selectorExpr):
		return goRetagSelectorExpressionAsQualifiedType(arg, arena, lang, syms)
	default:
		return nil
	}
}

// goRetagSelectorExpressionAsQualifiedType rebuilds a `pkg.Name` selector
// expression as the equivalent qualified_type. Only the direct two-child
// shape (operand=identifier, field=field_identifier) is handled — a deeper
// selector chain (`a.b.c`) is never a valid Go type reference, so its
// operand is itself a selector_expression rather than a bare identifier and
// this intentionally falls through untouched.
func goRetagSelectorExpressionAsQualifiedType(sel *Node, arena *nodeArena, lang *Language, syms goNewMakeSymbols) *Node {
	if resultChildCount(sel) != 3 {
		return nil
	}
	operand := resultChildAt(sel, 0)
	dot := resultChildAt(sel, 1)
	field := resultChildAt(sel, 2)
	if operand == nil || dot == nil || field == nil ||
		!cResultSymbolMatches(lang, operand, syms.identifier) ||
		!cResultSymbolMatches(lang, field, syms.fieldIdent) {
		return nil
	}
	pkgIdent := newLeafNodeInArena(arena, syms.packageIdent, symbolIsNamed(lang, syms.packageIdent), operand.startByte, operand.endByte, operand.startPoint, operand.endPoint)
	typeIdent := newLeafNodeInArena(arena, syms.typeIdentifier, symbolIsNamed(lang, syms.typeIdentifier), field.startByte, field.endByte, field.startPoint, field.endPoint)
	dotClone := cloneNodeInArena(arena, dot)
	children := cloneNodeSliceIfArena(arena, []*Node{pkgIdent, dotClone, typeIdent})
	fieldIDs := cloneFieldIDSliceInArena(arena, []FieldID{syms.packageFID, 0, syms.nameFID})
	return newParentNodeInArena(arena, syms.qualifiedType, symbolIsNamed(lang, syms.qualifiedType), children, fieldIDs, 0)
}
