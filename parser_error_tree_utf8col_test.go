package gotreesitter

import "testing"

// TestErrorTreeUsesByteColumnsForUTF8 pins the whole-source error-tree fallback
// (parseErrorTreeWithArena) to tree-sitter's byte-based column semantics.
// The end Point.Column must be the BYTE offset from the last newline, not a
// codepoint (rune) count. This path is the language-agnostic full-parse-failure
// fallback shared by every grammar, so a rune-count here undercounts columns for
// any multi-byte UTF-8 content in a document that collapses to an error tree.
func TestErrorTreeUsesByteColumnsForUTF8(t *testing.T) {
	// U+2717 (✗) is 3 UTF-8 bytes. rune-count would report each as 1 column.
	tests := []struct {
		name    string
		source  string
		wantRow uint32
		wantCol uint32
	}{
		{
			// "x✗z": x(1) + ✗(3) + z(1) = 5 bytes, single line.
			// byte-based end col = 5; rune-based (buggy) = 3.
			name:    "single line multibyte",
			source:  "x✗z",
			wantRow: 0,
			wantCol: 5,
		},
		{
			// "a✗\nb✗c": line 0 "a✗" (4 bytes), newline, line 1 "b✗c" (5 bytes).
			// byte-based end = (row 1, col 5); rune-based (buggy) = (row 1, col 3).
			name:    "multi line multibyte",
			source:  "a✗\nb✗c",
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
			tree := parseErrorTreeWithArena([]byte(tc.source), nil, nil)
			root := tree.RootNode()
			if got := root.StartByte(); got != 0 {
				t.Fatalf("start byte = %d, want 0", got)
			}
			if got := root.EndByte(); got != uint32(len(tc.source)) {
				t.Fatalf("end byte = %d, want %d", got, len(tc.source))
			}
			end := root.EndPoint()
			if end.Row != tc.wantRow || end.Column != tc.wantCol {
				t.Fatalf("end point = (%d,%d), want (%d,%d) [byte-based columns]",
					end.Row, end.Column, tc.wantRow, tc.wantCol)
			}
		})
	}
}
