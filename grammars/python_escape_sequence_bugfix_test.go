package grammars

import (
	"testing"

	"github.com/davidwalter0/gotreesitter"
)

// TestPythonEscapeSequenceNoDoubleCountAfterBackslashNewline is a regression
// test for the escape_sequence double-count bug: an escaped backslash ("\\")
// immediately followed by a raw newline inside a string produced TWO
// escape_sequence nodes where canonical tree-sitter-python (and CPython) count
// exactly ONE. The continuation-escape normalization pass re-scanned the second
// backslash of the "\\" pair as the start of a "\<newline>" line continuation.
//
// Real corpus sites: smtplib.py (python3.11 + python3.12).
func TestPythonEscapeSequenceNoDoubleCountAfterBackslashNewline(t *testing.T) {
	lang := PythonLanguage()

	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			// "\\" + raw newline inside a triple-quoted string => ONE escape.
			name: "backslash_backslash_then_newline_triple",
			src:  "x = \"\"\"a\\\\\nb\"\"\"\n",
			want: "(module (assignment (identifier) (string (string_start) (string_content (escape_sequence)) (string_end))))",
		},
		{
			// "\\" + raw newline inside a single-quoted string => ONE escape.
			name: "backslash_backslash_then_newline_single",
			src:  "x = \"a\\\\\nb\"\n",
			want: "(module (assignment (identifier) (string (string_start) (string_content (escape_sequence)) (string_end))))",
		},
		{
			// "\\b" (escaped backslash, no newline) => ONE escape (unchanged).
			name: "backslash_backslash_no_newline",
			src:  "x = \"\"\"a\\\\b\"\"\"\n",
			want: "(module (assignment (identifier) (string (string_start) (string_content (escape_sequence)) (string_end))))",
		},
		{
			// Genuine line continuation "\<newline>" => ONE escape (must survive).
			name: "genuine_line_continuation",
			src:  "x = \"a\\\nb\"\n",
			want: "(module (assignment (identifier) (string (string_start) (string_content (escape_sequence)) (string_end))))",
		},
		{
			// Escaped backslash then a genuine continuation => TWO escapes.
			name: "escaped_backslash_then_continuation",
			src:  "x = \"a\\\\b\\\nc\"\n",
			want: "(module (assignment (identifier) (string (string_start) (string_content (escape_sequence) (escape_sequence)) (string_end))))",
		},
		{
			// Three backslashes then newline: "\\" escape + "\<newline>" => TWO.
			name: "three_backslashes_then_newline",
			src:  "x = \"\"\"p\\\\\\\nq\"\"\"\n",
			want: "(module (assignment (identifier) (string (string_start) (string_content (escape_sequence) (escape_sequence)) (string_end))))",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parser := gotreesitter.NewParser(lang)
			tree, err := parser.Parse([]byte(tc.src))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			defer tree.Release()
			root := tree.RootNode()
			if root == nil {
				t.Fatal("nil root")
			}
			if got := root.SExpr(lang); got != tc.want {
				t.Fatalf("escape S-expression mismatch\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}
