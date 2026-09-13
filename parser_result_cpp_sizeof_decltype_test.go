package gotreesitter_test

import (
	"strings"
	"testing"

	gts "github.com/davidwalter0/gotreesitter"
	"github.com/davidwalter0/gotreesitter/grammars"
)

// TestCppSizeofDecltypeScope covers blob-parity cpp-4: `sizeof(decltype(x)::M)`
// must read the operand as a type (sizeof_expression -> type_descriptor ->
// qualified_identifier(scope: decltype, name: type_identifier)), matching
// tree-sitter-cpp, not as a parenthesized_expression.
func TestCppSizeofDecltypeScope(t *testing.T) {
	lang := grammars.CppLanguage()
	src := []byte("void f() {\n  static_assert(sizeof(decltype(document)::Ch) == sizeof(unsigned char), \"\");\n}\n")
	tree, err := gts.NewParser(lang).Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	defer tree.Release()
	root := tree.RootNode()
	sz := findCppNodeByType(root, lang, "sizeof_expression")
	if sz == nil {
		t.Fatalf("no sizeof_expression\n%s", root.SExpr(lang))
	}
	sx := sz.SExpr(lang)
	if strings.Contains(sx, "parenthesized_expression") {
		t.Fatalf("sizeof operand still parenthesized_expression, want type_descriptor\n%s", sx)
	}
	typeChild := sz.ChildByFieldName("type", lang)
	if typeChild == nil || typeChild.Type(lang) != "type_descriptor" {
		t.Fatalf("sizeof type field = %v, want type_descriptor\n%s", typeChild, sx)
	}
	if !strings.Contains(sx, "type_identifier") {
		t.Fatalf("decltype member not retagged type_identifier\n%s", sx)
	}

	// An ordinary sizeof(expr) must stay a parenthesized_expression.
	src2 := []byte("void f() {\n  int a = sizeof(x);\n}\n")
	tree2, err := gts.NewParser(lang).Parse(src2)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	defer tree2.Release()
	sz2 := findCppNodeByType(tree2.RootNode(), lang, "sizeof_expression")
	if sz2 == nil || !strings.Contains(sz2.SExpr(lang), "parenthesized_expression") {
		t.Fatalf("ordinary sizeof(x) should stay parenthesized_expression\n%s", tree2.RootNode().SExpr(lang))
	}
}
