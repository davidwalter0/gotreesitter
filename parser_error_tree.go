package gotreesitter

import "unicode/utf8"

func parseErrorTreeWithArena(source []byte, lang *Language, arena *nodeArena) *Tree {
	// tree-sitter columns are byte offsets from the start of the line, not
	// codepoint (rune) counts. advancePointByBytes advances Row on each '\n'
	// and Column by byte width, so a multi-byte UTF-8 character correctly
	// advances the column by its full byte length instead of 1 — keeping this
	// language-agnostic error-tree fallback consistent with the lexer's
	// byte-based position tracking.
	end := advancePointByBytes(Point{}, source)

	root := NewLeafNode(errorSymbol, true, 0, uint32(len(source)), Point{}, end)
	root.setHasError(true)
	if arena != nil {
		return newTreeWithArenas(root, source, lang, arena, nil)
	}
	return NewTree(root, source, lang)
}

func isWhitespaceOnlySource(source []byte) bool {
	for i := 0; i < len(source); i++ {
		switch source[i] {
		case ' ', '\t', '\n', '\r', '\f':
		default:
			return false
		}
	}
	return true
}

func extendNodeToTrailingWhitespace(n *Node, source []byte) {
	if n == nil {
		return
	}
	sourceEnd := uint32(len(source))
	if n.endByte >= sourceEnd {
		return
	}
	tail := source[n.endByte:sourceEnd]
	for i := 0; i < len(tail); i++ {
		switch tail[i] {
		case ' ', '\t', '\n', '\r', '\f':
		default:
			return
		}
	}

	pt := n.endPoint
	for i := 0; i < len(tail); {
		if tail[i] == '\n' {
			pt.Row++
			pt.Column = 0
			i++
			continue
		}
		_, size := utf8.DecodeRune(tail[i:])
		if size <= 0 {
			size = 1
		}
		i += size
		pt.Column++
	}

	n.endByte = sourceEnd
	n.endPoint = pt
}
