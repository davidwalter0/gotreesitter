package gotreesitter

// normalizeOrgCompatibility realigns gotreesitter's org output with C
// tree-sitter.
//
// Two narrow, data-driven repairs are applied, both addressing the same root
// cause: gotreesitter's reduce path elides the grammar's `expr` wrapper around
// runs of visible-anonymous tokens, whereas C tree-sitter keeps it.
//
//  1. container-expr wrapping. Inside a `value` (table-cell `contents`, ...) C
//     wraps every maximal whitespace-free run of tokens in an `expr`:
//
//     C:  (value [60-65]) -> (expr [60-65]) -> (^ : str)
//     Go: (value [60-65]) -> (^ : str)                     // expr wrapper missing
//
//     Across the whole corpus (138k+ C `expr` nodes) the grouping rule is exact:
//     each C `expr` is a gap-free token run and adjacent `expr` siblings are
//     always separated by whitespace. gotreesitter only drops the wrapper when
//     such a container would consist solely of bare tokens (no expr child
//     present and no other named child), so reconstructing the `expr` runs is
//     unambiguous and never double-wraps. The affected containers are exactly
//     `value` and `contents` (verified by lockstep comparison against C).
//
//  2. empty-expr -> str. An `expr` that is a single bare word (a directive name,
//     list-item keyword, property name, cell value, ...) wraps one anonymous,
//     visible `str` token spanning the whole `expr`:
//
//     C:  (expr [2-7]) -> (str [2-7] "TITLE")        ChildCount = 1
//     Go: (expr [2-7])                               ChildCount = 0
//
//     Across the corpus every empty Go `expr` corresponds to a C `expr` carrying
//     exactly one `str` child whose span is byte-identical to the parent (no
//     `num`, no multi-child, no span mismatch). This mirrors the existing
//     collapsed-named-leaf recovery used for c_sharp `implicit_type -> var` and
//     cobol `period -> .`.
func normalizeOrgCompatibility(root *Node, source []byte, lang *Language) {
	if root == nil || lang == nil || lang.Name != "org" {
		return
	}

	// Resolve a named `expr` symbol for synthesized wrappers. Any aliased `expr`
	// production works: only the public type name and named-ness participate in
	// tree comparison.
	exprSym, exprOK := lang.symbolByNameAndNamed("expr", true)
	if !exprOK {
		exprSym, exprOK = symbolByName(lang, "expr")
	}

	// Resolve a `str` token symbol that is anonymous (named == false), matching
	// the C grammar's hidden-string token inside `expr`.
	strSym, strOK := lang.symbolByNameAndNamed("str", false)
	if !strOK {
		strSym, strOK = symbolByName(lang, "str")
	}
	strNamed := strOK && symbolIsNamed(lang, strSym)

	if exprOK {
		normalizeOrgValueExprWrapping(root, source, lang, exprSym)
	}
	if strOK {
		normalizeOrgEmptyExprStr(root, lang, strSym, strNamed)
	}
}

// orgExprWrapContainers are the named containers whose all-bare-token form must
// be re-wrapped into `expr` runs to match C tree-sitter.
var orgExprWrapContainers = map[string]struct{}{
	"value":    {},
	"contents": {},
}

// normalizeOrgValueExprWrapping reconstructs the `expr` wrappers C places around
// gap-free token runs inside a container (`value`, `contents`) whose children
// are all bare anonymous tokens.
func normalizeOrgValueExprWrapping(root *Node, source []byte, lang *Language, exprSym Symbol) {
	exprNamed := symbolIsNamed(lang, exprSym)
	walkResultTree(root, func(n *Node) {
		if n == nil {
			return
		}
		if _, ok := orgExprWrapContainers[n.Type(lang)]; !ok {
			return
		}
		count := resultChildCount(n)
		if count == 0 {
			return
		}
		// Only act when the container consists solely of bare anonymous leaf
		// tokens. (When gotreesitter already produced `expr` children, or any
		// other named child, the container matches C and must be left untouched.)
		for i := 0; i < count; i++ {
			child := resultChildAt(n, i)
			if child == nil || child.IsNamed() || resultChildCount(child) != 0 {
				return
			}
		}

		groups := groupOrgTokensByGap(n, source)
		if len(groups) == 0 {
			return
		}
		wrappers := make([]*Node, 0, len(groups))
		for _, g := range groups {
			run := cloneNodeSliceInArena(n.ownerArena, g)
			wrapper := newParentNodeInArena(n.ownerArena, exprSym, exprNamed, run, nil, 0)
			wrappers = append(wrappers, wrapper)
		}
		n.children = cloneNodeSliceInArena(n.ownerArena, wrappers)
		n.fieldIDs = nil
		n.fieldSources = nil
		populateParentNode(n, n.children)
	})
}

// groupOrgTokensByGap partitions a node's children into maximal runs such that
// consecutive children inside a run have no intervening whitespace, and a
// whitespace gap (space/tab/newline/CR in the source between two children)
// starts a new run.
func groupOrgTokensByGap(n *Node, source []byte) [][]*Node {
	count := resultChildCount(n)
	if count == 0 {
		return nil
	}
	var groups [][]*Node
	var current []*Node
	var prevEnd uint32
	for i := 0; i < count; i++ {
		child := resultChildAt(n, i)
		if child == nil {
			continue
		}
		if len(current) > 0 && gapHasWhitespace(source, prevEnd, child.startByte) {
			groups = append(groups, current)
			current = nil
		}
		current = append(current, child)
		prevEnd = child.endByte
	}
	if len(current) > 0 {
		groups = append(groups, current)
	}
	return groups
}

func gapHasWhitespace(source []byte, start, end uint32) bool {
	if start >= end || int(end) > len(source) {
		// No measurable gap (adjacent or out of range): treat as same run.
		return start < end
	}
	for i := start; i < end; i++ {
		switch source[i] {
		case ' ', '\t', '\n', '\r', '\f', '\v':
			return true
		}
	}
	return false
}

// normalizeOrgEmptyExprStr reconstructs the single anonymous `str` child that C
// keeps inside an `expr` consisting of a single bare word but gotreesitter drops.
func normalizeOrgEmptyExprStr(root *Node, lang *Language, strSym Symbol, strNamed bool) {
	walkResultTree(root, func(n *Node) {
		if n == nil || resultChildCount(n) != 0 {
			return
		}
		// Match by display name so every aliased `expr` production is covered,
		// not just one specific internal symbol id.
		if n.Type(lang) != "expr" {
			return
		}
		// A `str` token is always non-empty; never synthesize a zero-width child.
		if n.endByte <= n.startByte {
			return
		}
		child := newLeafNodeInArena(n.ownerArena, strSym, strNamed, n.startByte, n.endByte, n.startPoint, n.endPoint)
		child.parent = n
		child.childIndex = 0
		n.children = cloneNodeSliceInArena(n.ownerArena, []*Node{child})
		n.fieldIDs = cloneFieldIDSliceInArena(n.ownerArena, []FieldID{0})
		n.fieldSources = nil
	})
}
