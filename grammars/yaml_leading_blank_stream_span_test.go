package grammars

import (
	"testing"

	"github.com/davidwalter0/gotreesitter"
)

// TestYAMLLeadingBlankLineStreamSpan guards a blob-path (gotreesitter.NewParser +
// grammars.YamlLanguage) regression where the root "stream" node's start
// point/byte was hardcoded to (0,0) in normalizeYAMLRecoveredRoot
// (parser_result_yaml.go), instead of being derived from the tree's first
// real child.
//
// tree-sitter-yaml's external scanner (scanner.c) consumes leading blank
// lines via skip-advance before the first real token, so those bytes never
// become part of any token or node — the C oracle's "stream" node starts at
// the first real content, not at byte 0. Forcing startByte/startPoint to 0
// silently re-absorbed the skipped span into the root's range whenever a
// YAML document was preceded by one or more blank lines, e.g. CMake-
// generated "CMakeConfigureLog.yaml" files that conventionally open with a
// blank line before "---". Verified against a freshly-built oracle
// (tree-sitter-grammars/tree-sitter-yaml @ 4463985dfccc640f3d6991e3396a2047
// 610cf5f8, the pinned commit in grammars/languages.lock) via `tree-sitter
// parse`.
func TestYAMLLeadingBlankLineStreamSpan(t *testing.T) {
	cases := []struct {
		name          string
		src           string
		wantStartRow  uint32
		wantStartCol  uint32
		wantStartByte uint32
		wantEndRow    uint32
		wantEndCol    uint32
	}{
		// One leading blank line, then "---" marker, then a mapping document.
		{"blank-then-marker", "\n---\nfoo: bar\n", 1, 0, 1, 3, 0},
		// No leading blank line at all — the root must still start at 0:0.
		{"no-leading-blank", "---\nfoo: bar\n", 0, 0, 0, 2, 0},
		// Two leading blank lines, no "---" marker (implicit document).
		{"two-blanks-no-marker", "\n\nfoo: bar\n", 2, 0, 2, 3, 0},
		// One leading blank line, no "---" marker — the exact CMakeConfigureLog
		// shape reported by the real-corpus parity run (13/13 corpus failures).
		{"one-blank-no-marker", "\nfoo: bar\n", 1, 0, 1, 2, 0},
		// Larger document (block scalar value) behind the same leading blank +
		// marker shape, to confirm the fix isn't limited to single-pair maps.
		{"blank-then-marker-block-scalar", "\n---\nmessage: |\n  hello\n", 1, 0, 1, 4, 0},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			entry := DetectLanguageByName("yaml")
			if entry == nil || entry.Language == nil {
				t.Skip("yaml language not registered")
			}
			lang := entry.Language()
			tree, err := gotreesitter.NewParser(lang).Parse([]byte(tc.src))
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			defer tree.Release()

			root := tree.RootNode()
			if root.Type(lang) != "stream" {
				t.Fatalf("root type = %q, want %q; tree: %s", root.Type(lang), "stream", root.SExpr(lang))
			}

			sp := root.StartPoint()
			if sp.Row != tc.wantStartRow || sp.Column != tc.wantStartCol {
				t.Errorf("root StartPoint = %d:%d, want %d:%d (leading blank line(s) must be excluded from the stream span); tree: %s",
					sp.Row, sp.Column, tc.wantStartRow, tc.wantStartCol, root.SExpr(lang))
			}
			if root.StartByte() != tc.wantStartByte {
				t.Errorf("root StartByte = %d, want %d", root.StartByte(), tc.wantStartByte)
			}

			ep := root.EndPoint()
			if ep.Row != tc.wantEndRow || ep.Column != tc.wantEndCol {
				t.Errorf("root EndPoint = %d:%d, want %d:%d", ep.Row, ep.Column, tc.wantEndRow, tc.wantEndCol)
			}
			if int(root.EndByte()) != len(tc.src) {
				t.Errorf("root EndByte = %d, want %d (full source length)", root.EndByte(), len(tc.src))
			}
		})
	}
}
