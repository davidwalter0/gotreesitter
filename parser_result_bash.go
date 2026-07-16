package gotreesitter

func normalizeBashProgramVariableAssignments(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "bash" || root.Type(lang) != "program" || len(root.children) == 0 {
		return
	}
	normalizeBashVariableAssignmentsInNode(root, source, lang)
}

func normalizeBashVariableAssignmentsInNode(node *Node, source []byte, lang *Language) {
	if node == nil || lang == nil || len(node.children) == 0 {
		return
	}
	for _, child := range node.children {
		if child != nil {
			normalizeBashVariableAssignmentsInNode(child, source, lang)
		}
	}
	out := make([]*Node, 0, len(node.children))
	changed := false
	for _, child := range node.children {
		if child == nil {
			continue
		}
		if child.Type(lang) == "variable_assignments" && bashAllVariableAssignments(child, lang) && bashShouldSplitVariableAssignments(node.Type(lang), child, source) {
			out = append(out, child.children...)
			changed = true
			continue
		}
		out = append(out, child)
	}
	if !changed {
		assignBashIfConditionField(node, lang)
		return
	}
	if node.ownerArena != nil {
		buf := node.ownerArena.allocNodeSlice(len(out))
		copy(buf, out)
		out = buf
	}
	node.children = out
	node.clearFieldMetadata()
	assignBashIfConditionField(node, lang)
}

func normalizeBashGeneratedCommandAssignments(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || !lang.GeneratedByGrammargen || lang.Name != "bash" || len(source) == 0 {
		return
	}
	ctx, ok := newBashGeneratedAssignmentContext(lang)
	if !ok {
		return
	}
	normalizeBashGeneratedCommandAssignmentsInNode(root, source, lang, ctx)
}

func normalizeBashCommandNameArguments(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "bash" || root.IsError() || root.hasError() {
		return
	}
	commandSym, ok := symbolByName(lang, "command")
	if !ok {
		return
	}
	commandNameSym, ok := symbolByName(lang, "command_name")
	if !ok {
		return
	}
	concatenationSym, ok := symbolByName(lang, "concatenation")
	if !ok {
		return
	}
	normalizeBashCommandNameArgumentsInNode(root, commandSym, commandNameSym, concatenationSym)
}

func normalizeBashCommandNameArgumentsInNode(node *Node, commandSym, commandNameSym, concatenationSym Symbol) {
	if node == nil || node.IsError() || node.hasError() {
		return
	}
	for _, child := range node.children {
		normalizeBashCommandNameArgumentsInNode(child, commandSym, commandNameSym, concatenationSym)
	}
	mergeBashCommandNameArguments(node, commandSym, commandNameSym, concatenationSym)
}

func mergeBashCommandNameArguments(node *Node, commandSym, commandNameSym, concatenationSym Symbol) bool {
	if node == nil || node.symbol != commandSym || len(node.children) < 2 || node.IsError() || node.hasError() {
		return false
	}
	commandName := node.children[0]
	if commandName == nil || commandName.symbol != commandNameSym || len(commandName.children) == 0 || commandName.IsError() || commandName.hasError() {
		return false
	}
	if len(commandName.children) == 1 && commandName.children[0] != nil && commandName.children[0].symbol == concatenationSym {
		return false
	}

	arena := node.ownerArena
	mergeEnd := 1
	lastEnd := commandName.endByte
	for mergeEnd < len(node.children) {
		child := node.children[mergeEnd]
		if child == nil || child.IsError() || child.hasError() || child.startByte != lastEnd {
			break
		}
		lastEnd = child.endByte
		mergeEnd++
	}
	if mergeEnd == 1 {
		return false
	}

	parts := make([]*Node, 0, len(commandName.children)+mergeEnd-1)
	parts = append(parts, commandName.children...)
	parts = append(parts, node.children[1:mergeEnd]...)
	concat := newParentNodeInArena(arena, concatenationSym, true, cloneNodeSliceInArena(arena, parts), nil, 0)

	commandName.endByte = concat.endByte
	commandName.endPoint = concat.endPoint
	replaceNodeChildrenUnfielded(commandName, []*Node{concat})

	commandChildren := make([]*Node, 0, len(node.children)-mergeEnd+1)
	commandChildren = append(commandChildren, commandName)
	commandChildren = append(commandChildren, node.children[mergeEnd:]...)
	if arena != nil {
		commandChildren = cloneNodeSliceInArena(arena, commandChildren)
	}
	replaceNodeChildrenUnfielded(node, commandChildren)
	return true
}

type bashGeneratedAssignmentContext struct {
	commandSym            Symbol
	commandNameSym        Symbol
	concatenationSym      Symbol
	variableAssignmentSym Symbol
	variableNameSym       Symbol
	nameFieldID           FieldID
	valueFieldID          FieldID
}

func newBashGeneratedAssignmentContext(lang *Language) (bashGeneratedAssignmentContext, bool) {
	var ctx bashGeneratedAssignmentContext
	var ok bool
	if ctx.commandSym, ok = symbolByName(lang, "command"); !ok {
		return ctx, false
	}
	if ctx.commandNameSym, ok = symbolByName(lang, "command_name"); !ok {
		return ctx, false
	}
	if ctx.concatenationSym, ok = symbolByName(lang, "concatenation"); !ok {
		return ctx, false
	}
	if ctx.variableAssignmentSym, ok = symbolByName(lang, "variable_assignment"); !ok {
		return ctx, false
	}
	if ctx.variableNameSym, ok = symbolByName(lang, "variable_name"); !ok {
		return ctx, false
	}
	ctx.nameFieldID, _ = lang.FieldByName("name")
	ctx.valueFieldID, _ = lang.FieldByName("value")
	return ctx, true
}

func normalizeBashGeneratedCommandAssignmentsInNode(node *Node, source []byte, lang *Language, ctx bashGeneratedAssignmentContext) {
	if node == nil {
		return
	}
	for _, child := range node.children {
		normalizeBashGeneratedCommandAssignmentsInNode(child, source, lang, ctx)
	}
	rewriteBashGeneratedCommandAssignment(node, source, lang, ctx)
}

func rewriteBashGeneratedCommandAssignment(node *Node, source []byte, lang *Language, ctx bashGeneratedAssignmentContext) bool {
	if node == nil || node.symbol != ctx.commandSym || len(node.children) != 1 {
		return false
	}
	commandName := node.children[0]
	if commandName == nil || commandName.symbol != ctx.commandNameSym {
		return false
	}
	parts := bashGeneratedAssignmentParts(commandName, ctx)
	if len(parts) == 0 {
		return false
	}
	first := parts[0]
	if first == nil || int(first.startByte) >= len(source) || first.endByte > uint32(len(source)) {
		return false
	}
	nameEnd, valueStart, ok := bashSimpleAssignmentSplit(source, first.startByte, first.endByte)
	if !ok || valueStart > commandName.endByte {
		return false
	}

	arena := node.ownerArena
	name := newLeafNodeInArena(arena, ctx.variableNameSym, true, first.startByte, nameEnd, first.startPoint, advancePointByBytes(first.startPoint, source[first.startByte:nameEnd]))
	valueChildren := make([]*Node, 0, len(parts))
	if valueStart < first.endByte {
		valueStartPoint := advancePointByBytes(first.startPoint, source[first.startByte:valueStart])
		valueChildren = append(valueChildren, newLeafNodeInArena(arena, first.symbol, first.isNamed(), valueStart, first.endByte, valueStartPoint, first.endPoint))
	}
	valueChildren = append(valueChildren, parts[1:]...)
	if len(valueChildren) == 0 {
		return false
	}

	value := valueChildren[0]
	if len(valueChildren) > 1 {
		value = newParentNodeInArena(arena, ctx.concatenationSym, true, cloneNodeSliceInArena(arena, valueChildren), nil, 0)
	}

	children := []*Node{name, value}
	if arena != nil {
		children = cloneNodeSliceInArena(arena, children)
	}
	node.symbol = ctx.variableAssignmentSym
	node.setNamed(true)
	node.children = children
	node.setFieldMetadata(bashGeneratedAssignmentFieldIDs(arena, ctx), bashGeneratedAssignmentFieldSources(ctx))
	node.setHasError(false)
	populateParentNode(node, children)
	return true
}

func bashGeneratedAssignmentParts(commandName *Node, ctx bashGeneratedAssignmentContext) []*Node {
	if commandName == nil {
		return nil
	}
	if len(commandName.children) != 1 {
		return nil
	}
	child := commandName.children[0]
	if child == nil {
		return nil
	}
	if child.symbol == ctx.concatenationSym {
		return child.children
	}
	return []*Node{child}
}

func bashSimpleAssignmentSplit(source []byte, startByte, endByte uint32) (uint32, uint32, bool) {
	if startByte >= endByte || endByte > uint32(len(source)) {
		return 0, 0, false
	}
	start := int(startByte)
	end := int(endByte)
	if !bashAssignmentNameStart(source[start]) {
		return 0, 0, false
	}
	for i := start + 1; i < end; i++ {
		c := source[i]
		if c == '=' {
			nameEnd := i
			if i > start && source[i-1] == '+' {
				nameEnd = i - 1
			}
			if nameEnd <= start {
				return 0, 0, false
			}
			return uint32(nameEnd), uint32(i + 1), true
		}
		if c == '+' && i+1 < end && source[i+1] == '=' {
			continue
		}
		if !bashAssignmentNameContinue(c) {
			return 0, 0, false
		}
	}
	return 0, 0, false
}

func bashAssignmentNameStart(c byte) bool {
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

func bashAssignmentNameContinue(c byte) bool {
	return bashAssignmentNameStart(c) || (c >= '0' && c <= '9')
}

func bashGeneratedAssignmentFieldIDs(arena *nodeArena, ctx bashGeneratedAssignmentContext) []FieldID {
	if ctx.nameFieldID == 0 && ctx.valueFieldID == 0 {
		return nil
	}
	fields := []FieldID{ctx.nameFieldID, ctx.valueFieldID}
	if arena == nil {
		return fields
	}
	out := arena.allocFieldIDSlice(len(fields))
	copy(out, fields)
	return out
}

func bashGeneratedAssignmentFieldSources(ctx bashGeneratedAssignmentContext) []uint8 {
	if ctx.nameFieldID == 0 && ctx.valueFieldID == 0 {
		return nil
	}
	return []uint8{fieldSourceDirect, fieldSourceDirect}
}

func bashAllVariableAssignments(node *Node, lang *Language) bool {
	if node == nil || lang == nil || len(node.children) < 2 {
		return false
	}
	for _, child := range node.children {
		if child == nil || child.Type(lang) != "variable_assignment" {
			return false
		}
	}
	return true
}

// bashShouldSplitVariableAssignments decides whether a raw "variable_assignments"
// group (upstream rule: variable_assignments: $ => seq($.variable_assignment,
// repeat1($.variable_assignment)) -- grammar.js) found as a direct child of
// parentType should be exploded back into individual variable_assignment
// siblings.
//
// $.variable_assignments is a completely ordinary member of
// $._statement_not_subshell (grammar.js), so it is grammatically valid --
// and must be left GROUPED -- as a direct statement child of program,
// if/while/for bodies, subshell, list, etc. whenever it represents one
// genuine same-statement multi-assignment, e.g. `VERBOSE= QUIET= OPTS=` or
// `[[ cond ]] || A=1 B=2 C=3`. Splitting those (as an earlier, parent-type-only
// heuristic did for program/if_statement/subshell/list) diverges from the
// pinned oracle -- confirmed byte-exact against tree-sitter-bash@a06c2e44 for
// both shapes.
//
// The one case that legitimately needs splitting is a merge artifact: gt's
// GLR/LALR statement-boundary handling occasionally accumulates what should
// have been >=2 separate _terminated_statement reductions (separate lines)
// into a single variable_assignments production instead of treating the
// intervening newline as a statement terminator. bashVariableAssignmentsSpansNewline
// distinguishes the two cases directly from the source bytes: a newline
// between two consecutive assignment children is only possible if a
// terminator got swallowed, since bash's own newline-as-terminator rule
// would otherwise have closed the statement there.
func bashShouldSplitVariableAssignments(parentType string, group *Node, source []byte) bool {
	switch parentType {
	case "command", "redirected_statement", "declaration_command", "unset_command":
		// variable_assignments never appears here in the upstream grammar
		// (command's own repeat(choice($.variable_assignment, ...)) already
		// yields individual assignment children); a distinct, more specific
		// pass (normalizeBashGeneratedCommandAssignments) owns any repair
		// needed for these parent shapes. Leave them alone here.
		return false
	default:
		return bashVariableAssignmentsSpansNewline(group, source)
	}
}

// bashVariableAssignmentsSpansNewline reports whether any gap between
// consecutive children of a "variable_assignments" group node contains a
// newline byte, which is only possible if the group is a cross-statement
// merge artifact rather than a genuine single-line multi-assignment.
func bashVariableAssignmentsSpansNewline(group *Node, source []byte) bool {
	if group == nil || len(source) == 0 || len(group.children) < 2 {
		return false
	}
	for i := 1; i < len(group.children); i++ {
		prev := group.children[i-1]
		cur := group.children[i]
		if prev == nil || cur == nil {
			continue
		}
		start, end := int(prev.endByte), int(cur.startByte)
		if start < 0 || end > len(source) || start > end {
			continue
		}
		for _, b := range source[start:end] {
			if b == '\n' {
				return true
			}
		}
	}
	return false
}

func assignBashIfConditionField(node *Node, lang *Language) {
	if node == nil || lang == nil || node.Type(lang) != "if_statement" || len(node.children) <= 1 {
		return
	}
	fid, ok := lang.FieldByName("condition")
	if !ok {
		return
	}
	ensureNodeFieldStorage(node, len(node.children))
	thenIndex := -1
	for i, child := range node.children {
		if child != nil && child.Type(lang) == "then" {
			thenIndex = i
			break
		}
	}
	if thenIndex < 0 {
		thenIndex = len(node.children)
	}
	fieldIDs := node.fieldIDs()
	fieldSources := node.fieldSources()
	for i := 1; i < thenIndex; i++ {
		if node.children[i] == nil {
			continue
		}
		fieldIDs[i] = fid
		fieldSources[i] = fieldSourceDirect
	}
}
