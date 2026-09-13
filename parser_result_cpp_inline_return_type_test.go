package gotreesitter_test

import (
	"testing"

	gts "github.com/davidwalter0/gotreesitter"
	"github.com/davidwalter0/gotreesitter/grammars"
)

// TestCppInlineErrorReturnType exercises the inline + custom-return-type
// recovery gap (blob-parity cpp-2). `inline DlScalar Foo(...)` used to leave
// the return-type identifier as a bare ERROR leaf; it must recover to a
// type_identifier child on the function_definition, matching tree-sitter-cpp.
func TestCppInlineErrorReturnType(t *testing.T) {
	lang := grammars.CppLanguage()
	cases := []struct {
		name string
		src  string
	}{
		{"inline_custom_ret_primitive_param", "inline DlScalar Foo(int height) {\n  return height;\n}\n"},
		{"inline_custom_ret_custom_param", "inline DlScalar AmbientBlurRadius(DlScalar height) {\n  return height;\n}\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := []byte(tc.src)
			tree, err := gts.NewParser(lang).Parse(src)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			defer tree.Release()
			root := tree.RootNode()
			if got, want := root.Type(lang), "translation_unit"; got != want {
				t.Fatalf("root type = %q, want %q\n%s", got, want, root.SExpr(lang))
			}
			if root.HasError() {
				t.Fatalf("root.HasError = true, want false\n%s", root.SExpr(lang))
			}
			fn := root.Child(0)
			if got, want := fn.Type(lang), "function_definition"; got != want {
				t.Fatalf("child[0] = %q, want %q\n%s", got, want, root.SExpr(lang))
			}
			// Locate the return type child (field "type"); it must be a
			// type_identifier with the custom return type text, never an ERROR.
			var typeChild *gts.Node
			for i := 0; i < int(fn.ChildCount()); i++ {
				c := fn.Child(i)
				if c.Type(lang) == "ERROR" {
					t.Fatalf("function_definition still carries an ERROR child\n%s", fn.SExpr(lang))
				}
				if fn.FieldNameForChild(i, lang) == "type" {
					typeChild = c
				}
			}
			if typeChild == nil {
				t.Fatalf("no type field on function_definition\n%s", fn.SExpr(lang))
			}
			if got, want := typeChild.Type(lang), "type_identifier"; got != want {
				t.Fatalf("return type = %q, want %q\n%s", got, want, fn.SExpr(lang))
			}
			if got, want := typeChild.Text(src), "DlScalar"; got != want {
				t.Fatalf("return type text = %q, want %q", got, want)
			}
		})
	}
}
