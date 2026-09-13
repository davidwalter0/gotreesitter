package gotreesitter_test

import (
	"strings"
	"testing"

	gts "github.com/davidwalter0/gotreesitter"
	"github.com/davidwalter0/gotreesitter/grammars"
)

// TestCppNestedTemplateDeclaration covers blob-parity cpp-1: a variable
// declaration whose type is a >=2-level template closed by `>>` (or `> >`)
// must recover to a declaration, not a binary_expression chain.
func TestCppNestedTemplateDeclaration(t *testing.T) {
	lang := grammars.CppLanguage()

	// Each must produce a `declaration` with a nested template_type type and no
	// residual ERROR / binary_expression at the statement position.
	decls := []struct {
		name string
		src  string
	}{
		{"local_std", "void f() {\n  std::deque<std::unique_ptr<int>> resolvers_;\n}\n"},
		{"local_custom", "void f() {\n  Deque<UniquePtr<int>> resolvers_;\n}\n"},
		{"global", "std::deque<std::unique_ptr<int>> resolvers_;\n"},
		{"triple_nested", "void f() {\n  std::vector<std::vector<std::vector<int>>> cube;\n}\n"},
		{"space_separated", "void f() {\n  std::deque<std::unique_ptr<int> > resolvers_;\n}\n"},
		{"namespace_wrapped", "namespace flutter {\nvoid f() {\n  std::deque<std::unique_ptr<int>> r;\n}\n}\n"},
	}
	for _, tc := range decls {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.src)
			tree, err := gts.NewParser(lang).Parse(src)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			defer tree.Release()
			root := tree.RootNode()
			sx := root.SExpr(lang)
			if root.HasError() {
				t.Fatalf("root.HasError = true, want false\n%s", sx)
			}
			// Locate the declaration node and verify it wraps a nested template.
			decl := findCppNodeByType(root, lang, "declaration")
			if decl == nil {
				t.Fatalf("no declaration node produced\n%s", sx)
			}
			typeChild := decl.ChildByFieldName("type", lang)
			if typeChild == nil {
				t.Fatalf("declaration has no type field\n%s", decl.SExpr(lang))
			}
			// The type must contain at least two template_argument_list levels.
			if n := strings.Count(decl.SExpr(lang), "template_argument_list"); n < 2 {
				t.Fatalf("expected nested template (>=2 template_argument_list), got %d\n%s", n, decl.SExpr(lang))
			}
			if strings.Contains(decl.SExpr(lang), "binary_expression") {
				t.Fatalf("declaration still contains a binary_expression chain\n%s", decl.SExpr(lang))
			}
		})
	}

	// Genuine comparison/shift statements must NOT be rewritten into
	// declarations (their trailing token is not a declarator).
	expressions := []struct {
		name string
		src  string
	}{
		{"compare_shift", "void f() {\n  a < b >> c;\n}\n"},
		{"assign_compare_shift", "void f() {\n  x = a < b >> c;\n}\n"},
		{"if_compare_shift", "void f() {\n  if (a < b >> c) {}\n}\n"},
		{"return_compare_shift", "void f() {\n  return a < b >> c;\n}\n"},
	}
	for _, tc := range expressions {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.src)
			tree, err := gts.NewParser(lang).Parse(src)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			defer tree.Release()
			root := tree.RootNode()
			if d := findCppNodeByType(root, lang, "declaration"); d != nil {
				t.Fatalf("genuine expression wrongly rewritten to a declaration\n%s", root.SExpr(lang))
			}
		})
	}
}

func findCppNodeByType(n *gts.Node, lang *gts.Language, typ string) *gts.Node {
	if n == nil {
		return nil
	}
	if n.Type(lang) == typ {
		return n
	}
	for i := 0; i < int(n.ChildCount()); i++ {
		if got := findCppNodeByType(n.Child(i), lang, typ); got != nil {
			return got
		}
	}
	return nil
}
