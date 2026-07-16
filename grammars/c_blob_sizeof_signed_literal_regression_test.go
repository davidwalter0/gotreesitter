package grammars

import (
	"testing"

	"github.com/davidwalter0/gotreesitter"
)

// findFirstNodeOfType returns the first node (preorder) whose Type matches
// typ, or nil if none is found.
func findFirstNodeOfType(t *testing.T, n *gotreesitter.Node, lang *gotreesitter.Language, typ string) *gotreesitter.Node {
	t.Helper()
	if n == nil {
		return nil
	}
	if n.Type(lang) == typ {
		return n
	}
	for i := 0; i < n.ChildCount(); i++ {
		if found := findFirstNodeOfType(t, n.Child(i), lang, typ); found != nil {
			return found
		}
	}
	return nil
}

// TestCBlobSizeofIdentifierPlusSignedLiteral guards the c-2 regression:
// `sizeof(a)+1` (no space) must not merge the trailing signed number_literal
// into a cast of `a`. Byte-for-byte verified against the pinned
// tree-sitter-c oracle (github.com/tree-sitter/tree-sitter-c @ ae19b676):
// binary_expression(sizeof_expression(parenthesized_expression(identifier)),
// +, number_literal).
func TestCBlobSizeofIdentifierPlusSignedLiteral(t *testing.T) {
	src := []byte("void f() { int x = sizeof(a)+1; }\n")
	root, lang := cBlobMustParse(t, src)

	binExpr := findFirstNodeOfType(t, root, lang, "binary_expression")
	if binExpr == nil {
		t.Fatalf("expected a binary_expression in the tree, got %s", root.SExpr(lang))
	}
	if got, want := binExpr.Text(src), "sizeof(a)+1"; got != want {
		t.Fatalf("binary_expression text = %q, want %q (full tree: %s)", got, want, root.SExpr(lang))
	}
	left := binExpr.ChildByFieldName("left", lang)
	if left == nil || left.Type(lang) != "sizeof_expression" {
		t.Fatalf("left = %v, want sizeof_expression", left)
	}
	sizeofValue := left.ChildByFieldName("value", lang)
	if sizeofValue == nil || sizeofValue.Type(lang) != "parenthesized_expression" {
		t.Fatalf("sizeof value = %v, want parenthesized_expression (bug: stayed a cast_expression)", sizeofValue)
	}
	op := binExpr.ChildByFieldName("operator", lang)
	if op == nil || op.Text(src) != "+" {
		t.Fatalf("operator = %v, want \"+\"", op)
	}
	right := binExpr.ChildByFieldName("right", lang)
	if right == nil || right.Type(lang) != "number_literal" || right.Text(src) != "1" {
		t.Fatalf("right = %v, want number_literal \"1\" (unsigned — the merged sign must detach)", right)
	}
}

// TestCBlobSizeofChainRebalancesLeftAssociative guards the harder half of
// c-2: `sizeof(a)+sizeof(b)+1` must rebalance to the oracle's
// left-associative grouping `(sizeof(a)+sizeof(b))+1`, not the naive
// right-associative `sizeof(a)+(sizeof(b)+1)` a per-node-only fix would
// produce.
func TestCBlobSizeofChainRebalancesLeftAssociative(t *testing.T) {
	src := []byte("void f() { int x = sizeof(a)+sizeof(b)+1; }\n")
	root, lang := cBlobMustParse(t, src)

	outer := findFirstNodeOfType(t, root, lang, "binary_expression")
	if outer == nil {
		t.Fatalf("expected outer binary_expression, got %s", root.SExpr(lang))
	}
	if got, want := outer.Text(src), "sizeof(a)+sizeof(b)+1"; got != want {
		t.Fatalf("outer text = %q, want %q", got, want)
	}
	outerRight := outer.ChildByFieldName("right", lang)
	if outerRight == nil || outerRight.Type(lang) != "number_literal" || outerRight.Text(src) != "1" {
		t.Fatalf("outer right = %v, want number_literal \"1\" (left-associative chain)", outerRight)
	}
	inner := outer.ChildByFieldName("left", lang)
	if inner == nil || inner.Type(lang) != "binary_expression" {
		t.Fatalf("outer left = %v, want binary_expression (sizeof(a)+sizeof(b))", inner)
	}
	if got, want := inner.Text(src), "sizeof(a)+sizeof(b)"; got != want {
		t.Fatalf("inner text = %q, want %q", got, want)
	}
	innerLeft := inner.ChildByFieldName("left", lang)
	innerRight := inner.ChildByFieldName("right", lang)
	if innerLeft == nil || innerLeft.Type(lang) != "sizeof_expression" || innerLeft.Text(src) != "sizeof(a)" {
		t.Fatalf("inner left = %v, want sizeof_expression \"sizeof(a)\"", innerLeft)
	}
	if innerRight == nil || innerRight.Type(lang) != "sizeof_expression" || innerRight.Text(src) != "sizeof(b)" {
		t.Fatalf("inner right = %v, want sizeof_expression \"sizeof(b)\"", innerRight)
	}
	if innerRightValue := innerRight.ChildByFieldName("value", lang); innerRightValue == nil || innerRightValue.Type(lang) != "parenthesized_expression" {
		t.Fatalf("inner right sizeof value = %v, want parenthesized_expression", innerRightValue)
	}
}

// TestCBlobSizeofThreeChainRebalancesLeftAssociative extends the chain test
// to three sizeof terms, byte-for-byte verified against the oracle.
func TestCBlobSizeofThreeChainRebalancesLeftAssociative(t *testing.T) {
	src := []byte("void f() { int x = sizeof(a)+sizeof(b)+sizeof(c)+1; }\n")
	root, lang := cBlobMustParse(t, src)

	outer := findFirstNodeOfType(t, root, lang, "binary_expression")
	if outer == nil || outer.Text(src) != "sizeof(a)+sizeof(b)+sizeof(c)+1" {
		t.Fatalf("outer = %v, full tree: %s", outer, root.SExpr(lang))
	}
	if right := outer.ChildByFieldName("right", lang); right == nil || right.Text(src) != "1" {
		t.Fatalf("outer right = %v, want \"1\"", right)
	}
	mid := outer.ChildByFieldName("left", lang)
	if mid == nil || mid.Text(src) != "sizeof(a)+sizeof(b)+sizeof(c)" {
		t.Fatalf("mid = %v", mid)
	}
	if midRight := mid.ChildByFieldName("right", lang); midRight == nil || midRight.Text(src) != "sizeof(c)" {
		t.Fatalf("mid right = %v, want sizeof(c)", midRight)
	}
	inner := mid.ChildByFieldName("left", lang)
	if inner == nil || inner.Text(src) != "sizeof(a)+sizeof(b)" {
		t.Fatalf("inner = %v", inner)
	}
}

// TestCBlobSizeofPrimitiveTypeStaysCast asserts the pass never touches a
// real primitive-type cast reading: `sizeof(int)+1` legitimately parses as
// `sizeof((int)(+1))` on both gt and the oracle (the merged sign is part of
// a genuine cast-of-a-literal, matching real C's own cast/sizeof
// ambiguity), so it must be left completely alone.
func TestCBlobSizeofPrimitiveTypeStaysCast(t *testing.T) {
	src := []byte("void f() { int x = sizeof(int)+1; }\n")
	root, lang := cBlobMustParse(t, src)

	sizeofExpr := findFirstNodeOfType(t, root, lang, "sizeof_expression")
	if sizeofExpr == nil {
		t.Fatalf("expected sizeof_expression, got %s", root.SExpr(lang))
	}
	if got, want := sizeofExpr.Text(src), "sizeof(int)+1"; got != want {
		t.Fatalf("sizeof_expression text = %q, want %q", got, want)
	}
	value := sizeofExpr.ChildByFieldName("value", lang)
	if value == nil || value.Type(lang) != "cast_expression" {
		t.Fatalf("value = %v, want cast_expression (regression: primitive-type cast reading must survive)", value)
	}
}

// TestCBlobSizeofKnownLocalTypedefAlsoDetaches guards against reusing the
// "known local typedef stays a type" convention from
// normalizeCSizeofUnknownTypeIdentifiers: tree-sitter-c has no semantic
// scope tracking, so the oracle resolves `sizeof(myint)+1` identically
// whether or not `myint` was typedef'd earlier in the same translation
// unit (empirically verified against the pinned oracle) — this pass must
// not special-case known local type names.
func TestCBlobSizeofKnownLocalTypedefAlsoDetaches(t *testing.T) {
	src := []byte("typedef int myint; void f() { int x = sizeof(myint)+1; }\n")
	root, lang := cBlobMustParse(t, src)

	binExpr := findFirstNodeOfType(t, root, lang, "binary_expression")
	if binExpr == nil {
		t.Fatalf("expected binary_expression, got %s", root.SExpr(lang))
	}
	left := binExpr.ChildByFieldName("left", lang)
	if left == nil || left.Type(lang) != "sizeof_expression" {
		t.Fatalf("left = %v, want sizeof_expression", left)
	}
	if value := left.ChildByFieldName("value", lang); value == nil || value.Type(lang) != "parenthesized_expression" {
		t.Fatalf("sizeof value = %v, want parenthesized_expression even for a known local typedef", value)
	}
}

// TestCBlobSizeofNoTrailingOperatorUnaffected asserts a bare `sizeof(a)`
// with nothing following it (the existing
// normalizeCSizeofUnknownTypeIdentifiers pass's own territory, already
// correct) is left alone by this pass.
func TestCBlobSizeofNoTrailingOperatorUnaffected(t *testing.T) {
	src := []byte("void f() { int x = sizeof(a); }\n")
	root, lang := cBlobMustParse(t, src)

	sizeofExpr := findFirstNodeOfType(t, root, lang, "sizeof_expression")
	if sizeofExpr == nil || sizeofExpr.Text(src) != "sizeof(a)" {
		t.Fatalf("sizeof_expression = %v, full tree: %s", sizeofExpr, root.SExpr(lang))
	}
	value := sizeofExpr.ChildByFieldName("value", lang)
	if value == nil || value.Type(lang) != "parenthesized_expression" {
		t.Fatalf("value = %v, want parenthesized_expression", value)
	}
}
