package gotreesitter_test

import "testing"

// TestParseGoBlankImportIsLeaf covers gt-RUNTIME bug go-2: blank_identifier
// (the `_` in `import _ "pkg"`) must be a leaf node (0 children), matching
// the C oracle, instead of wrapping a spurious anonymous "_" child. See the
// forestGapCollapse override in shouldKeepVisibleAnonymousTokenChild
// (parser_reduce.go), which shares the C-oracle-seeded allowlist already
// used by the GLR-forest tree-construction path (glr_forest.go).
func TestParseGoBlankImportIsLeaf(t *testing.T) {
	tree, lang := parseGo(t, "package main\n\nimport _ \"fmt\"\n\nfunc main() {}\n")
	root := tree.RootNode()
	if root.HasError() {
		t.Fatalf("root has error flag set: %s", root.SExpr(lang))
	}

	spec := findNamedChild(lang, root, "import_spec")
	if spec == nil {
		t.Fatal("no import_spec found")
	}
	blank := findNamedChild(lang, spec, "blank_identifier")
	if blank == nil {
		t.Fatal("no blank_identifier found in import_spec")
	}
	if got, want := blank.ChildCount(), 0; got != want {
		t.Fatalf("blank_identifier.ChildCount() = %d, want %d (leaf, matching oracle)", got, want)
	}
	if got, want := blank.Text(tree.Source()), "_"; got != want {
		t.Fatalf("blank_identifier text = %q, want %q", got, want)
	}
}
