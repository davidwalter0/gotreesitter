//go:build !grammar_subset || grammar_subset_authzed || grammar_subset_c || grammar_subset_cpp || grammar_subset_go || grammar_subset_html || grammar_subset_java || grammar_subset_json || grammar_subset_lua || grammar_subset_toml

package grammars

import "testing"

// TestSourceCursorAdvanceRuneUsesByteBasedColumns pins sourceCursor.advanceRune
// to tree-sitter's byte-based Point.Column semantics. advanceRune advances the
// byte offset by a decoded rune's full UTF-8 width (correct) but must also
// advance the column by that same byte width, not by 1 (a rune count). This
// cursor backs every hand-rolled comment/text scanner that shares
// token_source_common.go (c/cpp's CTokenSource.commentToken and
// consumeBlockComment, java's commentToken, html's commentToken, authzed's
// lineComment/blockComment, plus generic_lexer/lua_lexer/toml_lexer), so a
// rune-count here undercounts the column of a COMMENT node's endPoint (and
// every later node on the same line) whenever the comment contains a
// multi-byte UTF-8 character. Confirmed against a real corpus file
// (afl.c, an AFL++ source with em-dash comments): 2 range-only divergences
// against the tree-sitter-c oracle, both on comment endPoints, both
// eliminated by this fix with byte ranges unchanged.
func TestSourceCursorAdvanceRuneUsesByteBasedColumns(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantRow uint32
		wantCol uint32
	}{
		{
			// "café": c(1)+a(1)+f(1)+é(2) = 5 bytes, single line.
			// byte-based col = 5; rune-based (buggy) = 4.
			name:    "single line multibyte",
			source:  "café",
			wantRow: 0,
			wantCol: 5,
		},
		{
			// em-dash "—" (U+2014) is 3 bytes. "a—\nb—c": line 0 "a—" (4 bytes),
			// newline, line 1 "b—c" (5 bytes).
			// byte-based end = (row 1, col 5); rune-based (buggy) = (row 1, col 3).
			name:    "multi line multibyte with em-dash",
			source:  "a—\nb—c",
			wantRow: 1,
			wantCol: 5,
		},
		{
			// 4-byte astral char (😀 = U+1F600). byte-based col = 4, rune = 1.
			name:    "astral plane char",
			source:  "\U0001F600",
			wantRow: 0,
			wantCol: 4,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cur := newSourceCursor([]byte(tc.source))
			for !cur.eof() {
				cur.advanceRune()
			}
			if cur.row != tc.wantRow || cur.col != tc.wantCol {
				t.Fatalf("end point = (%d,%d), want (%d,%d) [byte-based columns]",
					cur.row, cur.col, tc.wantRow, tc.wantCol)
			}
		})
	}
}

func TestSourceCursorSkipsEscapedNewlineExtras(t *testing.T) {
	src := []byte(" \t\\\r\n  \\\nnext")
	cur := newSourceCursor(src)

	cur.skipSpacesTabsAndEscapedNewlines()

	if got, want := cur.offset, len(src)-len("next"); got != want {
		t.Fatalf("offset = %d, want %d", got, want)
	}
	if got, want := cur.row, uint32(2); got != want {
		t.Fatalf("row = %d, want %d", got, want)
	}
	if got, want := cur.col, uint32(0); got != want {
		t.Fatalf("column = %d, want %d", got, want)
	}
}
