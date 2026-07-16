package gotreesitter

// normalizeCPointerAssignmentInversion repairs the operator-precedence
// inversion that gt's LALR parse tables produce for writes through a pointer
// such as `*p = q`.
//
// Root cause (parse-table layer, not fixable here): the bare-identifier parser
// state reachable from a prefix `*`/`&` operand is LALR-merged with the state
// reached from a statement / assignment-left context. The merged state can
// only pick one action for the `=` lookahead, and it shifts — nesting the
// assignment INSIDE the pointer_expression (`*(p = q)`) instead of making the
// dereference the assignment's left operand (`(*p) = q`). The merge also
// prevents the precise LR(1) state that tree-sitter's own generator produces,
// so the fix lives here, in the established post-parse normalization layer.
//
// Detection is unambiguous: canonical C never places an assignment_expression
// *directly* under a pointer_expression. A dereferenced assignment is always
// parenthesized — `*(p = q)` parses as
// pointer_expression(parenthesized_expression(assignment_expression)) — so a
// pointer_expression whose `argument` child is directly an
// assignment_expression is only ever the inverted shape. The rotation makes
// the pointer_expression wrap just its operator plus the assignment's former
// left operand, and lifts the assignment to the top; byte ranges are
// recomputed from the children (the pointer_expression shrinks to span only
// `*p`, the assignment spans the whole `*p = q`) and field metadata is set to
// the canonical operator/argument and left/operator/right layout.
//
// A postorder walk resolves nested writes bottom-up, so chained and
// multi-level forms normalize correctly:
//   - `**pp = q`  → assignment(pointer(pointer(pp)), q)
//   - `*p = *q = r` → assignment(pointer(p), assignment(pointer(q), r))
func normalizeCPointerAssignmentInversion(root *Node, lang *Language) {
	if root == nil || lang == nil || lang.Name != "c" {
		return
	}
	pointerSym, ok := lang.symbolByNamePreferNamed("pointer_expression")
	if !ok {
		return
	}
	assignmentSym, ok := lang.symbolByNamePreferNamed("assignment_expression")
	if !ok {
		return
	}
	operatorFID, ok := lang.FieldByName("operator")
	if !ok {
		return
	}
	argumentFID, ok := lang.FieldByName("argument")
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
	pointerNamed := symbolIsNamed(lang, pointerSym)
	assignmentNamed := symbolIsNamed(lang, assignmentSym)

	walkResultTreePostorder(root, func(n *Node) {
		// Signature: pointer_expression(operator, argument=assignment_expression).
		if !cResultSymbolMatches(lang, n, pointerSym) || resultChildCount(n) != 2 {
			return
		}
		operator := resultChildAt(n, 0)
		argument := resultChildAt(n, 1)
		if operator == nil || argument == nil {
			return
		}
		if !cResultSymbolMatches(lang, argument, assignmentSym) || resultChildCount(argument) != 3 {
			return
		}
		assignLeft := resultChildAt(argument, 0)
		assignOp := resultChildAt(argument, 1)
		assignRight := resultChildAt(argument, 2)
		if assignLeft == nil || assignOp == nil || assignRight == nil {
			return
		}

		// Canonical dereference of the assignment's former left operand.
		innerPointer := newParentNodeInArena(
			n.ownerArena,
			pointerSym,
			pointerNamed,
			cloneNodeSliceIfArena(n.ownerArena, []*Node{operator, assignLeft}),
			cloneFieldIDSliceInArena(n.ownerArena, []FieldID{operatorFID, argumentFID}),
			0,
		)

		// Retag n in place as the outer assignment; populateParentNode (via
		// setCRewriteChildren) recomputes n's byte range from the rotated
		// children, so it keeps spanning the whole `*p = q`.
		children := cloneNodeSliceIfArena(n.ownerArena, []*Node{innerPointer, assignOp, assignRight})
		fieldIDs := cloneFieldIDSliceInArena(n.ownerArena, []FieldID{leftFID, operatorFID, rightFID})
		setCRewriteChildren(n, assignmentSym, assignmentNamed, children, fieldIDs, []int{0, 1, 2})
	})
}
