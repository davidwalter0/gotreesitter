package grammars

import (
	"testing"

	"github.com/davidwalter0/gotreesitter"
)

// TestPythonListSplatBindingScope is a regression test for the list_splat scope
// mis-binding. In a tuple / list display, a splat element whose operand carries
// a postfix chain (subscript / attribute / call) was parsed with "*" bound only
// to the primary, so the trailing postfix operators wrapped the list_splat —
// e.g. *params[-1].__args__ became (*params[-1]).__args__ instead of
// *(params[-1].__args__). Real corpus site: typing.py:1682
// `(*params[:-1], *params[-1].__args__)`.
//
// The "must stay unchanged" cases cover: a correct first-position splat, a
// splat whose operand is a call (list_splat sits at the top of the chain), a
// plain postfix chain with no splat, and — critically — the argument_list
// `*sys.version.split()` sub-case, where gt already matches CPython's AST
// (Starred wrapping the Call) while canonical tree-sitter-python has a
// pre-existing quirk. gt must NOT be "corrected" to that quirk.
func TestPythonListSplatBindingScope(t *testing.T) {
	lang := PythonLanguage()

	cases := []struct {
		name string
		src  string
		want string
	}{
		// --- mis-bindings that must now bind the whole postfix chain ---
		{
			name: "subscript_then_attribute_second_splat",
			src:  "x = (*params[:-1], *params[-1].__args__)\n",
			want: "(module (assignment (identifier) (tuple (list_splat (subscript (identifier) (slice (unary_operator (integer))))) (list_splat (attribute (subscript (identifier) (unary_operator (integer))) (identifier))))))",
		},
		{
			name: "second_splat_subscript_attribute",
			src:  "x = (*a, *b[-1].c)\n",
			want: "(module (assignment (identifier) (tuple (list_splat (identifier)) (list_splat (attribute (subscript (identifier) (unary_operator (integer))) (identifier))))))",
		},
		{
			name: "both_splats_subscript",
			src:  "x = (*a[-1], *b[-1])\n",
			want: "(module (assignment (identifier) (tuple (list_splat (subscript (identifier) (unary_operator (integer)))) (list_splat (subscript (identifier) (unary_operator (integer)))))))",
		},
		{
			name: "both_splats_attribute",
			src:  "x = (*a.b, *c.d)\n",
			want: "(module (assignment (identifier) (tuple (list_splat (attribute (identifier) (identifier))) (list_splat (attribute (identifier) (identifier))))))",
		},
		{
			name: "list_display_splat_call",
			src:  "x = [*a, *b.c()]\n",
			want: "(module (assignment (identifier) (list (list_splat (identifier)) (list_splat (call (attribute (identifier) (identifier)) (argument_list))))))",
		},

		// --- correct cases that must stay unchanged ---
		{
			name: "first_position_splat_stays_correct",
			src:  "x = (*a[-1].c, *b)\n",
			want: "(module (assignment (identifier) (tuple (list_splat (attribute (subscript (identifier) (unary_operator (integer))) (identifier))) (list_splat (identifier)))))",
		},
		{
			name: "single_splat_stays_correct",
			src:  "x = (*a[-1].c,)\n",
			want: "(module (assignment (identifier) (tuple (list_splat (attribute (subscript (identifier) (unary_operator (integer))) (identifier))))))",
		},
		{
			name: "argument_list_splat_stays_correct",
			src:  "x = f(*a, *b[-1].c)\n",
			want: "(module (assignment (identifier) (call (identifier) (argument_list (list_splat (identifier)) (list_splat (attribute (subscript (identifier) (unary_operator (integer))) (identifier)))))))",
		},
		{
			name: "plain_postfix_chain_no_splat",
			src:  "x = a[-1].b\n",
			want: "(module (assignment (identifier) (attribute (subscript (identifier) (unary_operator (integer))) (identifier))))",
		},

		// --- SKIP case: gt matches CPython, canonical has a quirk. Do NOT rotate. ---
		{
			name: "arg_list_star_dotted_call_stays_cpython_correct",
			src:  "print(\"==\", platform.python_implementation(), *sys.version.split())\n",
			want: "(module (call (identifier) (argument_list (string (string_start) (string_content) (string_end)) (call (attribute (identifier) (identifier)) (argument_list)) (list_splat (call (attribute (attribute (identifier) (identifier)) (identifier)) (argument_list))))))",
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
			if root.HasError() {
				t.Fatalf("unexpected parse error: %s", root.SExpr(lang))
			}
			if got := root.SExpr(lang); got != tc.want {
				t.Fatalf("list_splat S-expression mismatch\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}
