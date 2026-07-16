package gotreesitter_test

import (
	"strings"
	"testing"

	gts "github.com/davidwalter0/gotreesitter"
	"github.com/davidwalter0/gotreesitter/grammars"
)

// TestCppNestedTemplateCall covers the CALL/expression-context sibling of the
// nested-template declaration bug: a call whose callee is a >=2-level template
// closed by `>>` must recover to a call_expression(template_function, …,
// argument_list), not a binary_expression comparison chain. The expected
// S-expressions are the tree-sitter-cpp oracle shapes (confirmed against the
// control variants gt already parses correctly, e.g. foo<int, bar<int>>(a)).
func TestCppNestedTemplateCall(t *testing.T) {
	lang := grammars.CppLanguage()

	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			"plain_leading_nested_multi_arg",
			"void f() { foo<bar<int>>(a, b); }",
			"(translation_unit (function_definition (primitive_type) (function_declarator (identifier) (parameter_list)) (compound_statement (expression_statement (call_expression (template_function (identifier) (template_argument_list (type_descriptor (template_type (type_identifier) (template_argument_list (type_descriptor (primitive_type))))))) (argument_list (identifier) (identifier)))))))",
		},
		{
			"qualified_make_shared_empty_args",
			"void f() { auto x = std::make_shared<std::vector<uint8_t>>(); }",
			"(translation_unit (function_definition (primitive_type) (function_declarator (identifier) (parameter_list)) (compound_statement (declaration (placeholder_type_specifier (auto)) (init_declarator (identifier) (call_expression (qualified_identifier (namespace_identifier) (template_function (identifier) (template_argument_list (type_descriptor (qualified_identifier (namespace_identifier) (template_type (type_identifier) (template_argument_list (type_descriptor (primitive_type))))))))) (argument_list)))))))",
		},
		{
			"qualified_return_nested_qualified_inner",
			"void f() { return std::vector<std::unique_ptr<fml::Mapping>>(); }",
			"(translation_unit (function_definition (primitive_type) (function_declarator (identifier) (parameter_list)) (compound_statement (return_statement (call_expression (qualified_identifier (namespace_identifier) (template_function (identifier) (template_argument_list (type_descriptor (qualified_identifier (namespace_identifier) (template_type (type_identifier) (template_argument_list (type_descriptor (qualified_identifier (namespace_identifier) (type_identifier)))))))))) (argument_list))))))",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tree, err := gts.NewParser(lang).Parse([]byte(tc.src))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			defer tree.Release()
			root := tree.RootNode()
			sx := root.SExpr(lang)
			if root.HasError() {
				t.Fatalf("root.HasError = true, want false\n%s", sx)
			}
			if sx != tc.want {
				t.Fatalf("S-expression mismatch\n got: %s\nwant: %s", sx, tc.want)
			}
			if findCppNodeByType(root, lang, "call_expression") == nil {
				t.Fatalf("no call_expression produced\n%s", sx)
			}
		})
	}

	// Genuine comparison / shift expressions must NOT be rewritten into a
	// call_expression by the reconstruction: none has a nested template in a
	// callee position, so no template_function may appear.
	negatives := []struct {
		name string
		src  string
	}{
		{"compare_shift", "void f() { a < b >> c; }"},
		{"assign_compare_shift", "void f() { x = a < b >> c; }"},
		{"compare_shift_call_tail", "void f() { a < b >> c(d); }"},
	}
	for _, tc := range negatives {
		t.Run(tc.name, func(t *testing.T) {
			tree, err := gts.NewParser(lang).Parse([]byte(tc.src))
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			defer tree.Release()
			root := tree.RootNode()
			if strings.Contains(root.SExpr(lang), "template_function") {
				t.Fatalf("genuine comparison wrongly rewritten (template_function present)\n%s", root.SExpr(lang))
			}
		})
	}

	// A single-level generic call is handled natively by gt and must be left
	// untouched by the reconstruction (which only fires on a nested callee): it
	// stays a clean call_expression.
	t.Run("single_level_generic_call_untouched", func(t *testing.T) {
		tree, err := gts.NewParser(lang).Parse([]byte("void f() { foo<int>(a); }"))
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		defer tree.Release()
		root := tree.RootNode()
		if root.HasError() {
			t.Fatalf("root.HasError = true, want false\n%s", root.SExpr(lang))
		}
		if findCppNodeByType(root, lang, "call_expression") == nil {
			t.Fatalf("single-level generic call should stay a call_expression\n%s", root.SExpr(lang))
		}
	})
}
