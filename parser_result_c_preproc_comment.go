package gotreesitter

// normalizeCPreprocArgLeadingComment splits a leading block comment out of a
// preprocessor directive's argument, matching the pinned tree-sitter-c oracle
// (github.com/tree-sitter/tree-sitter-c @ ae19b676).
//
// For a directive line like `#  undef /**/ HAVE_MMAP`, the oracle emits the
// `/**/` as a separate *extra* `comment` child sitting between the
// preproc_directive and the `argument` preproc_arg, and starts the preproc_arg
// only at the following real token (`HAVE_MMAP`). gt's preproc lexer instead
// folds the whole `/**/ HAVE_MMAP` run into a single preproc_arg, producing a
// child-count and node-type divergence at the enclosing preproc_call/def.
//
// Only the exact, unambiguous shape is rewritten: a preproc_arg whose text
// *begins* with a `/* ... */` block comment that is followed (after
// whitespace) by more non-comment argument content. A preproc_arg with no
// leading comment, a comment-only argument (no trailing token to keep as the
// argument), or a `//` line comment is left completely untouched — the pass
// makes no change rather than guess a shape it has not verified against the
// oracle.
func normalizeCPreprocArgLeadingComment(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "c" || len(source) == 0 {
		return
	}
	argSym, ok := symbolByName(lang, "preproc_arg")
	if !ok {
		return
	}
	commentSym, commentNamed, ok := symbolMeta(lang, "comment")
	if !ok {
		return
	}
	walkResultTree(root, func(n *Node) {
		cSplitPreprocArgLeadingComment(n, source, lang, argSym, commentSym, commentNamed)
	})
}

func cSplitPreprocArgLeadingComment(n *Node, source []byte, lang *Language, argSym, commentSym Symbol, commentNamed bool) {
	childCount := resultChildCount(n)
	if childCount == 0 {
		return
	}
	argIdx := -1
	var arg *Node
	for i := 0; i < childCount; i++ {
		c := resultChildAt(n, i)
		if c != nil && c.symbol == argSym {
			argIdx, arg = i, c
			break
		}
	}
	if arg == nil || argIdx < 0 {
		return
	}

	start := int(arg.startByte)
	end := int(arg.endByte)
	if start < 0 || end > len(source) || start+3 >= end {
		return
	}
	// The argument must begin immediately with a block comment `/* ... */`.
	if source[start] != '/' || source[start+1] != '*' {
		return
	}
	commentEnd := -1
	for i := start + 2; i+1 < end; i++ {
		if source[i] == '*' && source[i+1] == '/' {
			commentEnd = i + 2
			break
		}
	}
	if commentEnd < 0 {
		return
	}
	// Skip inline whitespace (never a newline — a newline ends the directive)
	// to the start of the surviving argument token.
	newArgStart := commentEnd
	for newArgStart < end && (source[newArgStart] == ' ' || source[newArgStart] == '\t') {
		newArgStart++
	}
	// Require a non-whitespace argument remainder; a comment-only argument is
	// out of scope (the oracle would not keep a preproc_arg there).
	if newArgStart >= end {
		return
	}
	// The remainder must not itself be another comment — keep to the single
	// verified shape.
	if newArgStart+1 < end && source[newArgStart] == '/' && (source[newArgStart+1] == '*' || source[newArgStart+1] == '/') {
		return
	}

	arena := n.ownerArena
	origArgStart := arg.startPoint
	commentStartPoint := origArgStart
	commentEndPoint := advancePointByBytes(origArgStart, source[start:commentEnd])
	newArgStartPoint := advancePointByBytes(origArgStart, source[start:newArgStart])

	comment := newLeafNodeInArena(arena, commentSym, commentNamed, uint32(start), uint32(commentEnd), commentStartPoint, commentEndPoint)
	comment.setExtra(true)

	// Trim the surviving preproc_arg to begin after the comment (and its
	// trailing inline whitespace); its end is unchanged.
	arg.startByte = uint32(newArgStart)
	arg.startPoint = newArgStartPoint

	oldChildren := n.children
	oldFieldIDs := n.fieldIDs()
	newChildren := make([]*Node, 0, childCount+1)
	newFieldIDs := make([]FieldID, 0, childCount+1)
	for i := 0; i < childCount; i++ {
		if i == argIdx {
			newChildren = append(newChildren, comment)
			newFieldIDs = append(newFieldIDs, 0)
		}
		newChildren = append(newChildren, oldChildren[i])
		fid := FieldID(0)
		if i < len(oldFieldIDs) {
			fid = oldFieldIDs[i]
		}
		newFieldIDs = append(newFieldIDs, fid)
	}

	// The preproc_call's own span may extend past its last child (it absorbs
	// the trailing newline that ends the directive). populateParentNode
	// recomputes the span from the children and would drop that extension, so
	// snapshot and restore it — only a child was inserted, the outer bounds are
	// unchanged.
	origStartByte, origEndByte := n.startByte, n.endByte
	origStartPoint, origEndPoint := n.startPoint, n.endPoint

	n.children = cloneNodeSliceIfArena(arena, newChildren)
	n.setFieldMetadata(cloneFieldIDSliceInArena(arena, newFieldIDs), defaultFieldSourcesInArena(arena, newFieldIDs))
	if arena != nil {
		arena.clearFinalChildRefs(n)
	}
	n.productionID = 0
	populateParentNode(n, n.children)

	n.startByte, n.endByte = origStartByte, origEndByte
	n.startPoint, n.endPoint = origStartPoint, origEndPoint
}
