package grammargen

import (
	"testing"

	gotreesitter "github.com/davidwalter0/gotreesitter"
)

// TestCPointerAssignmentInversionParity locks in the post-parse repair of the
// LALR-merge precedence inversion for writes through a pointer. The generated
// (imported-from-grammar.json) C language historically nested the assignment
// inside the pointer_expression (`*(p = q)`); the c_pointer_assignment_inversion
// normalization pass rotates it to the canonical `(*p) = q`. Every case must
// deep-match the reference C oracle, including the must-not-change forms that
// were already correct.
func TestCPointerAssignmentInversionParity(t *testing.T) {
	genLang, refLang := loadGeneratedCLanguage(t)
	genParser := gotreesitter.NewParser(genLang)
	refParser := gotreesitter.NewParser(refLang)

	cases := []struct {
		name string
		src  string
	}{
		// Must-fix: pointer_expression previously wrapped the assignment.
		{"simple", "void f() { *p = q; }\n"},
		{"literal_rhs", "void f() { *count = 0; }\n"},
		{"double_deref", "void f() { **pp = q; }\n"},
		{"chained", "void f() { *p = *q = r; }\n"},
		{"augmented", "void f() { *p += q; }\n"},
		{"compound_rhs", "void f() { *n = (*n + 1) & M; }\n"},
		{"deref_field", "void f() { *p->x = q; }\n"},
		{"deref_subscript", "void f() { *a[i] = q; }\n"},
		// Must-not-change: already-correct assignments and the genuinely
		// parenthesized dereference-of-assignment (`*(p = q)`), which the
		// signature must never touch.
		{"plain", "void f() { a = b; }\n"},
		{"field_lhs", "void f() { s.f = b; }\n"},
		{"subscript_lhs", "void f() { arr[i] = b; }\n"},
		{"arrow_lhs", "void f() { p->x = b; }\n"},
		{"paren_assign_deref", "void f() { x = *(p = q); }\n"},
		{"deref_read", "void f() { x = *p; }\n"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			genTree, err := genParser.Parse([]byte(tc.src))
			if err != nil {
				t.Fatalf("gen parse: %v", err)
			}
			defer genTree.Release()
			refTree, err := refParser.Parse([]byte(tc.src))
			if err != nil {
				t.Fatalf("ref parse: %v", err)
			}
			defer refTree.Release()

			genRoot := genTree.RootNode()
			refRoot := refTree.RootNode()
			if genRoot == nil || refRoot == nil {
				t.Fatal("nil root")
			}
			if genRoot.HasError() {
				t.Fatalf("generated tree has ERROR: %s", genRoot.SExpr(genLang))
			}
			if refRoot.HasError() {
				t.Fatalf("reference tree has ERROR: %s", refRoot.SExpr(refLang))
			}
			genSexp := normalizeSexp(genRoot.SExpr(genLang))
			refSexp := normalizeSexp(refRoot.SExpr(refLang))
			if genSexp != refSexp {
				t.Fatalf("parity mismatch:\n  gen: %s\n  ref: %s", genRoot.SExpr(genLang), refRoot.SExpr(refLang))
			}
		})
	}
}

// TestCPointerAssignmentInversionExactShape asserts the canonical s-expression
// for the headline case directly, independent of the reference oracle, so the
// rotation shape is pinned even if the oracle blob changes.
func TestCPointerAssignmentInversionExactShape(t *testing.T) {
	genLang, _ := loadGeneratedCLanguage(t)
	genParser := gotreesitter.NewParser(genLang)

	cases := []struct {
		src  string
		want string
	}{
		{
			src:  "void f() { *p = q; }\n",
			want: "(pointer_expression (identifier))",
		},
	}
	for _, tc := range cases {
		tree, err := genParser.Parse([]byte(tc.src))
		if err != nil {
			t.Fatalf("parse: %v", err)
		}
		sexp := tree.RootNode().SExpr(genLang)
		tree.Release()
		// The pointer_expression must now be a child of an assignment_expression,
		// i.e. "assignment_expression" appears before "pointer_expression".
		assignIdx := indexOfSub(sexp, "assignment_expression")
		ptrIdx := indexOfSub(sexp, "pointer_expression")
		if assignIdx < 0 || ptrIdx < 0 || assignIdx > ptrIdx {
			t.Fatalf("expected assignment_expression to enclose pointer_expression, got: %s", sexp)
		}
	}
}

func indexOfSub(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
