package gotreesitter_test

import (
	"testing"

	gts "github.com/davidwalter0/gotreesitter"
	"github.com/davidwalter0/gotreesitter/grammars"
)

// TestCppOutOfLineDefaultedEmptyParamMember covers blob-parity cpp-3: an
// out-of-line defaulted special member with an empty parameter list
// (`A::~A() = default;`, `A::A() = default;`) must match tree-sitter-cpp's
// expression_statement(assignment_expression(call_expression, identifier))
// shape, while `= delete`, non-empty params, and function bodies must remain
// function_definition (unchanged).
func TestCppOutOfLineDefaultedEmptyParamMember(t *testing.T) {
	lang := grammars.CppLanguage()

	rewritten := []struct {
		name string
		src  string
	}{
		{"dtor", "A::~A() = default;\n"},
		{"ctor", "A::A() = default;\n"},
		{"nested_qualified_dtor", "ns::A::~A() = default;\n"},
	}
	for _, tc := range rewritten {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.src)
			tree, err := gts.NewParser(lang).Parse(src)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			defer tree.Release()
			root := tree.RootNode()
			if root.HasError() {
				t.Fatalf("root.HasError = true, want false\n%s", root.SExpr(lang))
			}
			stmt := root.Child(0)
			if got, want := stmt.Type(lang), "expression_statement"; got != want {
				t.Fatalf("child[0] = %q, want %q\n%s", got, want, root.SExpr(lang))
			}
			assign := stmt.Child(0)
			if got, want := assign.Type(lang), "assignment_expression"; got != want {
				t.Fatalf("assign = %q, want %q\n%s", got, want, stmt.SExpr(lang))
			}
			left := assign.ChildByFieldName("left", lang)
			right := assign.ChildByFieldName("right", lang)
			if left == nil || left.Type(lang) != "call_expression" {
				t.Fatalf("assign.left = %v, want call_expression\n%s", left, assign.SExpr(lang))
			}
			if right == nil || right.Type(lang) != "identifier" || right.Text(src) != "default" {
				t.Fatalf("assign.right = %v, want identifier 'default'\n%s", right, assign.SExpr(lang))
			}
			fn := left.ChildByFieldName("function", lang)
			args := left.ChildByFieldName("arguments", lang)
			if fn == nil || fn.Type(lang) != "qualified_identifier" {
				t.Fatalf("call.function = %v, want qualified_identifier\n%s", fn, left.SExpr(lang))
			}
			if args == nil || args.Type(lang) != "argument_list" {
				t.Fatalf("call.arguments = %v, want argument_list\n%s", args, left.SExpr(lang))
			}

			// Regression: the rewritten assign's middle child (the `=` token)
			// must carry the "operator" field, matching every other
			// assignment_expression shape (`a = b;`, `a += b;`, `arr[0] = b;`,
			// ...). This was hardcoded to field ID 0 (no field) when this
			// rewrite was first added, so gt's `=` node exposed no field tag
			// here even though the real tree-sitter-cpp oracle always assigns
			// "operator" to an assignment_expression's middle token.
			eq := assign.ChildByFieldName("operator", lang)
			if eq == nil || eq.Type(lang) != "=" {
				t.Fatalf("assign.operator = %v, want \"=\" token\n%s", eq, assign.SExpr(lang))
			}
			if got, want := int(assign.ChildCount()), 3; got != want {
				t.Fatalf("assign.ChildCount() = %d, want %d\n%s", got, want, assign.SExpr(lang))
			}
			if got := assign.FieldNameForChild(1, lang); got != "operator" {
				t.Fatalf("assign child[1] field name = %q, want \"operator\"\n%s", got, assign.SExpr(lang))
			}
		})
	}

	unchanged := []struct {
		name string
		src  string
	}{
		{"delete", "A::~A() = delete;\n"},
		{"nonempty_params", "A::A(const A& o) = default;\n"},
		{"body", "A::~A() {}\n"},
	}
	for _, tc := range unchanged {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.src)
			tree, err := gts.NewParser(lang).Parse(src)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			defer tree.Release()
			root := tree.RootNode()
			if got, want := root.Child(0).Type(lang), "function_definition"; got != want {
				t.Fatalf("child[0] = %q, want %q (must stay function_definition)\n%s", got, want, root.SExpr(lang))
			}
		})
	}
}
