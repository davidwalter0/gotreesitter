package gotreesitter

// PureScript result normalization.
//
// The Go GLR parser produces trees that diverge from tree-sitter-c on a handful
// of systematic, faithful-to-grammar shapes. Each pass below reshapes the Go
// tree to match the C oracle exactly; all of them are pure structure rewrites
// that preserve byte spans.
//
//  1. exp_apply single-child retag.
//     The grammar's `_fexp` rule (grammar/exp.js) is documented to "get a node
//     for function application only if there is more than one expression
//     present":
//         _fexp: $ => choice($._aexp, alias($._exp_apply, $.exp_apply))
//     C reduces a bare atom (`true`, `1`, `Additive`) through `$._aexp`, so it
//     never emits a single-child `exp_apply`; it emits `exp_name`/`exp_literal`.
//     Go's GLR instead reduces through the `exp_apply` branch even for a single
//     `_aexp`, yielding `exp_apply(variable)` where C has `exp_name(variable)`.
//     Fix: a single-child `exp_apply` is retagged to `exp_name` (name/ctor
//     child) or `exp_literal` (literal child) — the child is left untouched.
//
//  2. Wildcard canonicalization.
//     C represents a pattern wildcard as `pat_wildcard(pat_wildcard(_))` and a
//     type wildcard as `type_wildcard(_)`. The Go reducer collapses the inner
//     anonymous `_` (and the inner `pat_wildcard`), leaving a childless node.
//     Fix: rebuild the canonical nested shape.
//
//  3. pat_name constructor leaf restore.
//     C wraps an upper-case constructor pattern as `pat_name(constructor)`. The
//     Go reducer drops the inner `constructor` leaf, leaving `pat_name` with no
//     children (the lower-case `pat_name(variable)` case is not collapsed).
//     Fix: restore the missing `constructor` leaf.
//
//  4. pat_parens wrapper restore.
//     C wraps a parenthesised pattern as `pat_parens('(' inner ')')`; every
//     child of a `patterns` node is a complete pattern. Go inlines the `'('
//     inner ')'` tokens directly into `patterns`. Fix: group each balanced
//     `'(' … ')'` run back into a `pat_parens` node.
//
//  5. forall re-nest. C nests a `forall` as the first child of the following
//     `type_infix` (`type_infix(forall, …)`); Go keeps `forall` as a sibling.
//     Fix: merge an adjacent `forall`+`type_infix` pair into one type_infix.
//
//  6. constraint/instance_head type_apply grouping. C folds the 2+ type
//     arguments after `class_name` into a single (left-flattened) `type_apply`;
//     Go leaves them flat. Fix: wrap the argument run and flatten nesting.
//
//  7. data constructor type_apply grouping. The same fold for the 2+ type
//     arguments after each `constructor` in a `data` declaration.
//
//  8. patterns bare-name wrap / empty-patterns restore. Every child of a
//     `patterns` node is a complete pattern node in C; Go can leave a bare
//     `variable`/`constructor`, or drop the `pat_name -> constructor|variable`
//     subtree entirely. Fix: wrap/rebuild the `pat_name` layer (leaf kind by
//     leading-character case).
//
//  9. record/array retag. Go misreduces an empty/bracketed literal through the
//     application path, tagging `{…}`/`[…]` as `exp_apply`; C tags them
//     `record_literal`/`exp_array`. Fix: retag by the leading bracket token.
//
//  10. field_value lambda wrap. C wraps a lambda field value in `exp_lambda`;
//      Go inlines the `\ … -> …` tokens into `field_value`. Fix: wrap them.
//
// These are restorations of layers the Go reducer collapses (or retags of
// reduction-choice ambiguities), in the same spirit as
// normalizeCollapsedNamedLeafChildren (used for c_sharp/cobol).
//
// Not addressed here (require parser/grammar fixes, not tree reshaping):
//   - Top-level `signature` swallowed into the following `function`: Go's type
//     parser over-extends a `type_infix` across the newline that ends a
//     signature, merging `name :: type` with the next equation's LHS into one
//     `function`. The merged subtree is structurally wrong (the type_infix
//     contains the equation LHS), so a faithful split would require re-parsing.
//   - Whole-file ERROR roots on a few inputs (the Go parse fails/recovers into
//     a single ERROR node where C parses cleanly).

type purescriptCompatSymbols struct {
	expApply   Symbol
	expName    Symbol
	expLiteral Symbol

	patWildcard   Symbol
	typeWildcard  Symbol
	underscore    Symbol
	hasUnderscore bool

	patName     Symbol
	constructor Symbol
	variable    Symbol
	hasPatName  bool

	patterns  Symbol
	patParens Symbol
	lparen    Symbol
	rparen    Symbol
	hasParens bool

	forall    Symbol
	typeInfix Symbol
	hasForall bool

	typeApply    Symbol
	className    Symbol
	constraint   Symbol
	instanceHead Symbol
	hasTypeApply bool

	recordLiteral Symbol
	expArray      Symbol

	dataDecl     Symbol
	constructor2 Symbol
	hasData      bool

	fieldValue   Symbol
	expLambda    Symbol
	backslash    Symbol
	hasFieldLamb bool

	expNameNamed       bool
	expLiteralNamed    bool
	recordLiteralNamed bool
	expArrayNamed      bool
}

func normalizePurescriptCompatibility(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "purescript" {
		return
	}
	syms := loadPurescriptCompatSymbols(lang)

	normalizePurescriptSingleChildExpApply(root, lang, syms)
	normalizePurescriptWildcards(root, lang, syms)
	normalizePurescriptPatNameConstructor(root, lang, syms)
	normalizePurescriptPatParens(root, lang, syms)
	normalizePurescriptForallInfix(root, lang, syms)
	normalizePurescriptConstraintTypeApply(root, lang, syms)
	normalizePurescriptDataConstructorArgs(root, lang, syms)
	normalizePurescriptPatternsBareName(root, lang, syms)
	normalizePurescriptEmptyPatterns(root, source, lang, syms)
	normalizePurescriptFieldValueLambda(root, lang, syms)
}

func loadPurescriptCompatSymbols(lang *Language) purescriptCompatSymbols {
	var s purescriptCompatSymbols
	s.expApply, _ = symbolByName(lang, "exp_apply")
	s.expName, _ = symbolByName(lang, "exp_name")
	s.expLiteral, _ = symbolByName(lang, "exp_literal")
	s.expNameNamed = symbolIsNamed(lang, s.expName)
	s.expLiteralNamed = symbolIsNamed(lang, s.expLiteral)

	s.patWildcard, _ = symbolByName(lang, "pat_wildcard")
	s.typeWildcard, _ = symbolByName(lang, "type_wildcard")
	// NOTE: Language.SymbolByName("_") is hard-coded to return the query
	// wildcard sentinel (symbol 0), NOT the real anonymous "_" terminal. Use
	// the token-name table to recover the actual leaf token symbol.
	s.underscore, s.hasUnderscore = purescriptAnonTokenSymbol(lang, "_")

	s.patName, _ = symbolByName(lang, "pat_name")
	s.constructor, s.hasPatName = symbolByName(lang, "constructor")
	s.variable, _ = lang.symbolByNameAndNamed("variable", true)

	s.patterns, _ = symbolByName(lang, "patterns")
	s.patParens, _ = symbolByName(lang, "pat_parens")
	s.lparen, _ = purescriptAnonTokenSymbol(lang, "(")
	s.rparen, _ = purescriptAnonTokenSymbol(lang, ")")
	s.hasParens = s.patParens != 0 && s.lparen != 0 && s.rparen != 0

	// `forall` has both an anonymous keyword token symbol and a named rule
	// symbol; the tree node is the NAMED one.
	var forallOK bool
	s.forall, forallOK = lang.symbolByNameAndNamed("forall", true)
	s.typeInfix, _ = symbolByName(lang, "type_infix")
	s.hasForall = forallOK && s.forall != 0 && s.typeInfix != 0

	s.typeApply, _ = symbolByName(lang, "type_apply")
	s.className, _ = symbolByName(lang, "class_name")
	s.constraint, _ = symbolByName(lang, "constraint")
	s.instanceHead, _ = symbolByName(lang, "instance_head")
	s.hasTypeApply = s.typeApply != 0 && s.className != 0 && (s.constraint != 0 || s.instanceHead != 0)

	s.recordLiteral, _ = symbolByName(lang, "record_literal")
	s.expArray, _ = symbolByName(lang, "exp_array")
	s.recordLiteralNamed = symbolIsNamed(lang, s.recordLiteral)
	s.expArrayNamed = symbolIsNamed(lang, s.expArray)

	var dataOK bool
	s.dataDecl, dataOK = lang.symbolByNameAndNamed("data", true)
	s.constructor2, _ = symbolByName(lang, "constructor")
	s.hasData = dataOK && s.dataDecl != 0 && s.constructor2 != 0 && s.typeApply != 0

	s.fieldValue, _ = symbolByName(lang, "field_value")
	s.expLambda, _ = symbolByName(lang, "exp_lambda")
	s.backslash, _ = purescriptAnonTokenSymbol(lang, "\\")
	s.hasFieldLamb = s.fieldValue != 0 && s.expLambda != 0 && s.backslash != 0
	return s
}

// purescriptAnonTokenSymbol returns the visible anonymous terminal token symbol
// with the given display name, bypassing the "_" wildcard special-case in
// Language.SymbolByName.
func purescriptAnonTokenSymbol(lang *Language, name string) (Symbol, bool) {
	for _, sym := range lang.TokenSymbolsByName(name) {
		if symbolIsVisible(lang, sym) && !symbolIsNamed(lang, sym) {
			return sym, true
		}
	}
	for _, sym := range lang.TokenSymbolsByName(name) {
		return sym, true
	}
	return 0, false
}

// purescriptExpNameChildKinds are the atoms that C wraps in `exp_name`.
func purescriptExpNameChildKind(t string) bool {
	switch t {
	case "variable", "qualified_variable", "constructor", "qualified_constructor":
		return true
	}
	return false
}

// purescriptExpLiteralChildKinds are the atoms that C wraps in `exp_literal`.
func purescriptExpLiteralChildKind(t string) bool {
	switch t {
	case "string", "integer", "number", "char":
		return true
	}
	return false
}

// normalizePurescriptSingleChildExpApply retags exp_apply nodes to the node C
// would produce:
//   - a single named atom child -> exp_name / exp_literal,
//   - a bracket-delimited body ({…} or […]) that Go misreduced through the
//     application path -> record_literal / exp_array.
func normalizePurescriptSingleChildExpApply(root *Node, lang *Language, syms purescriptCompatSymbols) {
	if syms.expApply == 0 {
		return
	}
	walkResultTree(root, func(n *Node) {
		if n == nil || n.symbol != syms.expApply {
			return
		}
		cc := resultChildCount(n)
		if cc == 0 {
			return
		}
		// Bracket-led exp_apply is really a record/array literal.
		if first := resultChildAt(n, 0); first != nil {
			switch first.Type(lang) {
			case "{":
				if syms.recordLiteral != 0 {
					n.symbol = syms.recordLiteral
					n.setNamed(syms.recordLiteralNamed)
				}
				return
			case "[":
				if syms.expArray != 0 {
					n.symbol = syms.expArray
					n.setNamed(syms.expArrayNamed)
				}
				return
			}
		}
		if cc != 1 {
			return
		}
		child := resultChildAt(n, 0)
		if child == nil || !child.IsNamed() {
			return
		}
		switch ct := child.Type(lang); {
		case purescriptExpNameChildKind(ct) && syms.expName != 0:
			n.symbol = syms.expName
			n.setNamed(syms.expNameNamed)
		case purescriptExpLiteralChildKind(ct) && syms.expLiteral != 0:
			n.symbol = syms.expLiteral
			n.setNamed(syms.expLiteralNamed)
		}
	})
}

// normalizePurescriptWildcards rebuilds the C-canonical wildcard subtree shape:
// pat_wildcard -> pat_wildcard -> _   and   type_wildcard -> _.
func normalizePurescriptWildcards(root *Node, lang *Language, syms purescriptCompatSymbols) {
	if !syms.hasUnderscore {
		return
	}
	if syms.patWildcard != 0 {
		walkResultTree(root, func(n *Node) {
			if n == nil || n.symbol != syms.patWildcard {
				return
			}
			// Only act on the OUTER pat_wildcard (its parent is not a
			// pat_wildcard), so we don't double-process the nested pair.
			if n.parent != nil && n.parent.symbol == syms.patWildcard {
				return
			}
			ensurePurescriptPatWildcardShape(n, syms)
		})
	}
	if syms.typeWildcard != 0 {
		walkResultTree(root, func(n *Node) {
			if n == nil || n.symbol != syms.typeWildcard {
				return
			}
			if resultChildCount(n) == 0 {
				addPurescriptLeafChild(n, syms.underscore, false)
			}
		})
	}
}

// ensurePurescriptPatWildcardShape makes outer be pat_wildcard(pat_wildcard(_)).
func ensurePurescriptPatWildcardShape(outer *Node, syms purescriptCompatSymbols) {
	cc := resultChildCount(outer)
	switch cc {
	case 0:
		// Build inner pat_wildcard(_) and attach.
		inner := newLeafNodeInArena(outer.ownerArena, syms.patWildcard, true, outer.startByte, outer.endByte, outer.startPoint, outer.endPoint)
		underscore := newLeafNodeInArena(outer.ownerArena, syms.underscore, false, outer.startByte, outer.endByte, outer.startPoint, outer.endPoint)
		underscore.parent = inner
		underscore.childIndex = 0
		inner.children = cloneNodeSliceInArena(outer.ownerArena, []*Node{underscore})
		inner.parent = outer
		inner.childIndex = 0
		outer.children = cloneNodeSliceInArena(outer.ownerArena, []*Node{inner})
		outer.fieldIDs = nil
		outer.fieldSources = nil
	case 1:
		inner := resultChildAt(outer, 0)
		if inner == nil {
			return
		}
		if inner.symbol == syms.patWildcard && resultChildCount(inner) == 0 {
			addPurescriptLeafChild(inner, syms.underscore, false)
		}
	}
}

// normalizePurescriptPatNameConstructor restores the dropped `constructor` leaf
// under a childless pat_name (the constructor pattern case).
func normalizePurescriptPatNameConstructor(root *Node, lang *Language, syms purescriptCompatSymbols) {
	if !syms.hasPatName || syms.patName == 0 || syms.constructor == 0 {
		return
	}
	walkResultTree(root, func(n *Node) {
		if n == nil || n.symbol != syms.patName || resultChildCount(n) != 0 {
			return
		}
		addPurescriptLeafChild(n, syms.constructor, true)
	})
}

// addPurescriptLeafChild gives a childless node a single leaf child spanning the
// same bytes.
func addPurescriptLeafChild(n *Node, childSym Symbol, named bool) {
	child := newLeafNodeInArena(n.ownerArena, childSym, named, n.startByte, n.endByte, n.startPoint, n.endPoint)
	child.parent = n
	child.childIndex = 0
	n.children = cloneNodeSliceInArena(n.ownerArena, []*Node{child})
	n.fieldIDs = nil
	n.fieldSources = nil
}

// normalizePurescriptPatParens groups inlined `'(' inner ')'` runs inside a
// `patterns` node back into a `pat_parens` wrapper, matching C.
func normalizePurescriptPatParens(root *Node, lang *Language, syms purescriptCompatSymbols) {
	if !syms.hasParens || syms.patterns == 0 {
		return
	}
	walkResultTree(root, func(n *Node) {
		if n == nil || n.symbol != syms.patterns {
			return
		}
		regroupPurescriptPatternsParens(n, syms)
	})
}

// normalizePurescriptForallInfix re-nests a `forall` that the Go reducer keeps
// as a sibling of the following `type_infix` back INSIDE that type_infix as its
// first child, matching C (`type_infix(forall, …)`).
func normalizePurescriptForallInfix(root *Node, lang *Language, syms purescriptCompatSymbols) {
	if !syms.hasForall {
		return
	}
	walkResultTree(root, func(n *Node) {
		if n == nil {
			return
		}
		mergePurescriptForallIntoInfix(n, syms)
	})
}

func mergePurescriptForallIntoInfix(parent *Node, syms purescriptCompatSymbols) {
	for i := 0; i+1 < resultChildCount(parent); i++ {
		a := resultChildAt(parent, i)
		b := resultChildAt(parent, i+1)
		if a == nil || b == nil {
			continue
		}
		if a.symbol != syms.forall || b.symbol != syms.typeInfix {
			continue
		}
		if a.endByte > b.startByte {
			continue
		}
		// Prepend `forall` (a) into the type_infix (b) as its first child and
		// extend b's start to cover the forall.
		infixChildren := resultChildSliceForMutation(b)
		merged := make([]*Node, 0, len(infixChildren)+1)
		merged = append(merged, a)
		merged = append(merged, infixChildren...)
		mergedClone := cloneNodeSliceInArena(b.ownerArena, merged)
		newInfix := newParentNodeInArena(b.ownerArena, syms.typeInfix, b.IsNamed(), mergedClone, nil, 0)
		newInfix.startByte = a.startByte
		newInfix.endByte = b.endByte
		newInfix.startPoint = a.startPoint
		newInfix.endPoint = b.endPoint
		newInfix.parent = parent
		replaceChildRangeWithNodes(parent, i, i+2, []*Node{newInfix})
		// The merged type_infix now sits at index i; do not advance past it so
		// any further forall directly before it (rare) could chain, but the
		// outer loop re-reads child count each iteration.
	}
}

// normalizePurescriptConstraintTypeApply groups the flat run of type-argument
// children that the Go reducer leaves after `class_name` inside a `constraint`
// or `instance_head` into a single `type_apply`, matching C (which keeps at most
// one type-argument node and folds multiple arguments into a type_apply).
func normalizePurescriptConstraintTypeApply(root *Node, lang *Language, syms purescriptCompatSymbols) {
	if !syms.hasTypeApply {
		return
	}
	walkResultTree(root, func(n *Node) {
		if n == nil {
			return
		}
		if n.symbol != syms.constraint && n.symbol != syms.instanceHead {
			return
		}
		groupPurescriptTypeArgs(n, syms)
	})
	// C's type_apply is always left-flattened; flatten any residual nesting Go
	// produced (e.g. type_apply(A type_apply(B C)) -> type_apply(A B C)).
	flattenPurescriptNestedTypeApply(root, syms)
}

func flattenPurescriptNestedTypeApply(root *Node, syms purescriptCompatSymbols) {
	walkResultTreePostorder(root, func(n *Node) {
		if n == nil || n.symbol != syms.typeApply {
			return
		}
		children := resultChildSliceForMutation(n)
		hasNested := false
		for _, c := range children {
			if c != nil && c.symbol == syms.typeApply {
				hasNested = true
				break
			}
		}
		if !hasNested {
			return
		}
		flat := flattenPurescriptTypeArgs(children, syms)
		flatClone := cloneNodeSliceInArena(n.ownerArena, flat)
		replaceChildRangeWithNodes(n, 0, resultChildCount(n), flatClone)
	})
}

func groupPurescriptTypeArgs(n *Node, syms purescriptCompatSymbols) {
	count := resultChildCount(n)
	if count < 2 {
		return
	}
	// Locate the last class_name; type arguments follow it.
	classIdx := -1
	for i := 0; i < count; i++ {
		c := resultChildAt(n, i)
		if c != nil && c.symbol == syms.className {
			classIdx = i
		}
	}
	if classIdx < 0 {
		return
	}
	start := classIdx + 1
	// Need at least two arguments after class_name to form a type_apply.
	if count-start < 2 {
		return
	}
	// All arguments to be grouped must be named type nodes (no stray anon
	// tokens / commas), so the wrap is faithful.
	for i := start; i < count; i++ {
		c := resultChildAt(n, i)
		if c == nil || !c.IsNamed() {
			return
		}
	}
	lo := resultChildAt(n, start)
	hi := resultChildAt(n, count-1)
	group := resultChildSliceRangeForMutation(n, start, count)
	// C's type_apply is fully left-flattened: `A B C` -> type_apply(A B C),
	// never type_apply(A type_apply(B C)). If the trailing argument is itself
	// a type_apply (Go's right-leaning reduction), splice in its children.
	flat := flattenPurescriptTypeArgs(group, syms)
	groupClone := cloneNodeSliceInArena(n.ownerArena, flat)
	wrapper := newParentNodeInArena(n.ownerArena, syms.typeApply, true, groupClone, nil, 0)
	wrapper.startByte = lo.startByte
	wrapper.endByte = hi.endByte
	wrapper.startPoint = lo.startPoint
	wrapper.endPoint = hi.endPoint
	wrapper.parent = n
	replaceChildRangeWithNodes(n, start, count, []*Node{wrapper})
}

// flattenPurescriptTypeArgs expands any direct type_apply argument into its
// children so the resulting type_apply argument list is flat, matching C.
func flattenPurescriptTypeArgs(args []*Node, syms purescriptCompatSymbols) []*Node {
	out := make([]*Node, 0, len(args))
	for _, a := range args {
		if a != nil && a.symbol == syms.typeApply {
			inner := resultChildSliceForMutation(a)
			out = append(out, flattenPurescriptTypeArgs(inner, syms)...)
			continue
		}
		out = append(out, a)
	}
	return out
}

// normalizePurescriptPatternsBareName wraps a bare `variable` / `constructor`
// argument that the Go reducer leaves directly under a `patterns` node in a
// `pat_name`, matching C (every `patterns` child is a complete pattern node).
func normalizePurescriptPatternsBareName(root *Node, lang *Language, syms purescriptCompatSymbols) {
	if syms.patterns == 0 || syms.patName == 0 {
		return
	}
	walkResultTree(root, func(n *Node) {
		if n == nil || n.symbol != syms.patterns {
			return
		}
		for i := 0; i < resultChildCount(n); i++ {
			c := resultChildAt(n, i)
			if c == nil {
				continue
			}
			switch c.Type(lang) {
			case "variable", "constructor", "qualified_variable", "qualified_constructor":
			default:
				continue
			}
			cClone := cloneNodeSliceInArena(n.ownerArena, []*Node{c})
			wrapper := newParentNodeInArena(n.ownerArena, syms.patName, true, cClone, nil, 0)
			wrapper.startByte = c.startByte
			wrapper.endByte = c.endByte
			wrapper.startPoint = c.startPoint
			wrapper.endPoint = c.endPoint
			wrapper.parent = n
			replaceChildRangeWithNodes(n, i, i+1, []*Node{wrapper})
		}
	})
}

// normalizePurescriptDataConstructorArgs groups the flat type-argument run that
// follows a `constructor` inside a `data` declaration into a single `type_apply`
// (when 2+ args), matching C. Each constructor's arguments end at the next `|`
// or at the end of the data node.
func normalizePurescriptDataConstructorArgs(root *Node, lang *Language, syms purescriptCompatSymbols) {
	if !syms.hasData {
		return
	}
	walkResultTree(root, func(n *Node) {
		if n == nil || n.symbol != syms.dataDecl {
			return
		}
		// Re-scan from the start after each rewrite, since indices shift.
		i := 0
		for i < resultChildCount(n) {
			c := resultChildAt(n, i)
			if c == nil || c.symbol != syms.constructor2 {
				i++
				continue
			}
			// Collect the contiguous run of named type-arg children after the
			// constructor (stopping at `|` or any anonymous token).
			start := i + 1
			end := start
			cnt := resultChildCount(n)
			for end < cnt {
				ch := resultChildAt(n, end)
				if ch == nil || !ch.IsNamed() || ch.symbol == syms.constructor2 {
					break
				}
				end++
			}
			if end-start >= 2 {
				lo := resultChildAt(n, start)
				hi := resultChildAt(n, end-1)
				group := resultChildSliceRangeForMutation(n, start, end)
				flat := flattenPurescriptTypeArgs(group, syms)
				groupClone := cloneNodeSliceInArena(n.ownerArena, flat)
				wrapper := newParentNodeInArena(n.ownerArena, syms.typeApply, true, groupClone, nil, 0)
				wrapper.startByte = lo.startByte
				wrapper.endByte = hi.endByte
				wrapper.startPoint = lo.startPoint
				wrapper.endPoint = hi.endPoint
				wrapper.parent = n
				replaceChildRangeWithNodes(n, start, end, []*Node{wrapper})
				i = start + 1
				continue
			}
			i = end
		}
	})
}

// normalizePurescriptEmptyPatterns restores the `pat_name -> constructor|variable`
// subtree that the Go reducer fully collapses when a `patterns` node is a single
// bare name pattern (e.g. the `LT` in `show LT = …`). The leaf kind is chosen by
// the leading character's case: upper -> constructor, otherwise -> variable.
func normalizePurescriptEmptyPatterns(root *Node, source []byte, lang *Language, syms purescriptCompatSymbols) {
	if syms.patterns == 0 || syms.patName == 0 || len(source) == 0 {
		return
	}
	walkResultTree(root, func(n *Node) {
		if n == nil || n.symbol != syms.patterns || resultChildCount(n) != 0 {
			return
		}
		if n.startByte >= n.endByte || int(n.endByte) > len(source) {
			return
		}
		leafSym := syms.variable
		if c := source[n.startByte]; c >= 'A' && c <= 'Z' {
			leafSym = syms.constructor
		}
		if leafSym == 0 {
			return
		}
		named := symbolIsNamed(lang, leafSym)
		leaf := newLeafNodeInArena(n.ownerArena, leafSym, named, n.startByte, n.endByte, n.startPoint, n.endPoint)
		patName := newParentNodeInArena(n.ownerArena, syms.patName, true, cloneNodeSliceInArena(n.ownerArena, []*Node{leaf}), nil, 0)
		patName.startByte = n.startByte
		patName.endByte = n.endByte
		patName.startPoint = n.startPoint
		patName.endPoint = n.endPoint
		patName.parent = n
		patName.childIndex = 0
		n.children = cloneNodeSliceInArena(n.ownerArena, []*Node{patName})
		n.fieldIDs = nil
		n.fieldSources = nil
	})
}

// normalizePurescriptFieldValueLambda wraps a lambda that the Go reducer inlines
// directly into a `field_value` (`{ k: \x -> e }`) in an `exp_lambda`, matching
// C. Fires only when the field_value's first child is the lambda backslash.
func normalizePurescriptFieldValueLambda(root *Node, lang *Language, syms purescriptCompatSymbols) {
	if !syms.hasFieldLamb {
		return
	}
	walkResultTree(root, func(n *Node) {
		if n == nil || n.symbol != syms.fieldValue {
			return
		}
		count := resultChildCount(n)
		if count < 2 {
			return
		}
		first := resultChildAt(n, 0)
		if first == nil || first.symbol != syms.backslash {
			return
		}
		lo := resultChildAt(n, 0)
		hi := resultChildAt(n, count-1)
		group := resultChildSliceRangeForMutation(n, 0, count)
		groupClone := cloneNodeSliceInArena(n.ownerArena, group)
		wrapper := newParentNodeInArena(n.ownerArena, syms.expLambda, true, groupClone, nil, 0)
		wrapper.startByte = lo.startByte
		wrapper.endByte = hi.endByte
		wrapper.startPoint = lo.startPoint
		wrapper.endPoint = hi.endPoint
		wrapper.parent = n
		replaceChildRangeWithNodes(n, 0, count, []*Node{wrapper})
	})
}

func regroupPurescriptPatternsParens(n *Node, syms purescriptCompatSymbols) {
	// Iterate by scanning the (possibly sidecar-backed) child list for a bare
	// '(' … ')' run and replacing it in-place with a pat_parens wrapper. After
	// each replacement the child indices shift, so we restart from the wrapper.
	for i := 0; i < resultChildCount(n); i++ {
		c := resultChildAt(n, i)
		if c == nil || c.symbol != syms.lparen {
			continue
		}
		// Find matching ')' with paren balancing.
		depth := 1
		j := i + 1
		count := resultChildCount(n)
		for ; j < count; j++ {
			cj := resultChildAt(n, j)
			if cj == nil {
				continue
			}
			if cj.symbol == syms.lparen {
				depth++
			} else if cj.symbol == syms.rparen {
				depth--
				if depth == 0 {
					break
				}
			}
		}
		if j >= count {
			// Unbalanced — leave as is.
			continue
		}
		lo := resultChildAt(n, i)
		hi := resultChildAt(n, j)
		group := resultChildSliceRangeForMutation(n, i, j+1)
		groupClone := cloneNodeSliceInArena(n.ownerArena, group)
		wrapper := newParentNodeInArena(n.ownerArena, syms.patParens, true, groupClone, nil, 0)
		wrapper.startByte = lo.startByte
		wrapper.endByte = hi.endByte
		wrapper.startPoint = lo.startPoint
		wrapper.endPoint = hi.endPoint
		replaceChildRangeWithNodes(n, i, j+1, []*Node{wrapper})
		// wrapper now sits at index i; continue scanning after it.
	}
}
