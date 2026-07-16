package grammars

import (
	"testing"

	"github.com/davidwalter0/gotreesitter"
)

// TestPythonAssignmentRHSPatternsResolveToExpressions is a regression test for
// the list/tuple vs list_pattern/tuple_pattern GLR ambiguity mis-resolution.
// On the right-hand side of "=", a multi-item value list with no unambiguous
// anchor (empty [] / (), or a bare-identifier element) was mis-resolved to the
// pattern reading (list_pattern / tuple_pattern / list_splat_pattern) where
// canonical tree-sitter-python emits the expression reading (list / tuple /
// list_splat). Real corpus site: difflib.py:1540 `fromlines,tolines=[],[]`.
//
// The "must stay a pattern" cases guard the recursion boundary: comprehension
// for-targets, chained-assignment left sides, and for-statement targets are
// legitimate pattern scopes and must NOT be converted.
func TestPythonAssignmentRHSPatternsResolveToExpressions(t *testing.T) {
	lang := PythonLanguage()

	cases := []struct {
		name string
		src  string
		want string
	}{
		// --- mis-resolutions that must now become expressions ---
		{
			name: "single_target_multi_empty_list",
			src:  "x = [], []\n",
			want: "(module (assignment (identifier) (expression_list (list) (list))))",
		},
		{
			name: "multi_target_empty_list",
			src:  "fromlines, tolines = [], []\n",
			want: "(module (assignment (pattern_list (identifier) (identifier)) (expression_list (list) (list))))",
		},
		{
			name: "multi_target_empty_tuple",
			src:  "a, b = (), ()\n",
			want: "(module (assignment (pattern_list (identifier) (identifier)) (expression_list (tuple) (tuple))))",
		},
		{
			name: "nonempty_bare_identifier_element",
			src:  "a, b = [x], []\n",
			want: "(module (assignment (pattern_list (identifier) (identifier)) (expression_list (list (identifier)) (list))))",
		},
		{
			name: "three_empty_lists",
			src:  "a, b, c = [], [], []\n",
			want: "(module (assignment (pattern_list (identifier) (identifier) (identifier)) (expression_list (list) (list) (list))))",
		},
		{
			name: "nested_empty_list",
			src:  "a, b = [[]], []\n",
			want: "(module (assignment (pattern_list (identifier) (identifier)) (expression_list (list (list)) (list))))",
		},
		{
			name: "nested_tuple_of_list",
			src:  "a, b = ([],), ()\n",
			want: "(module (assignment (pattern_list (identifier) (identifier)) (expression_list (tuple (list)) (tuple))))",
		},
		{
			name: "list_splat_element",
			src:  "a, b = *x, []\n",
			want: "(module (assignment (pattern_list (identifier) (identifier)) (expression_list (list_splat (identifier)) (list))))",
		},
		{
			name: "deeply_nested_containers",
			src:  "a, b = [([], [])], []\n",
			want: "(module (assignment (pattern_list (identifier) (identifier)) (expression_list (list (tuple (list) (list))) (list))))",
		},
		{
			name: "chained_multi_target",
			src:  "a = b, c = [], []\n",
			want: "(module (assignment (identifier) (assignment (pattern_list (identifier) (identifier)) (expression_list (list) (list)))))",
		},

		// --- anchored cases that were already correct (must stay expressions) ---
		{
			name: "integer_anchor",
			src:  "a, b = [1], []\n",
			want: "(module (assignment (pattern_list (identifier) (identifier)) (expression_list (list (integer)) (list))))",
		},
		{
			name: "empty_lists_in_list_literal",
			src:  "a = [[], []]\n",
			want: "(module (assignment (identifier) (list (list) (list))))",
		},

		// --- legitimate pattern scopes that must NOT be converted ---
		{
			name: "comprehension_for_target_stays_pattern",
			src:  "a = [i for i, j in pairs]\n",
			want: "(module (assignment (identifier) (list_comprehension (identifier) (for_in_clause (pattern_list (identifier) (identifier)) (identifier)))))",
		},
		{
			name: "chained_unpack_left_stays_pattern",
			src:  "a = [b, c] = [1, 2]\n",
			want: "(module (assignment (identifier) (assignment (list_pattern (identifier) (identifier)) (list (integer) (integer)))))",
		},
		{
			name: "for_statement_target_stays_pattern",
			src:  "for a, b in [([], [])]: pass\n",
			want: "(module (for_statement (pattern_list (identifier) (identifier)) (list (tuple (list) (list))) (block (pass_statement))))",
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
				t.Fatalf("RHS-pattern S-expression mismatch\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}
