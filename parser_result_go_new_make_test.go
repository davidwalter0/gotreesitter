package gotreesitter_test

import (
	"testing"

	"github.com/davidwalter0/gotreesitter"
)

// findFirstNamedChild returns the first *direct* named child of n (skipping
// anonymous tokens like "(" / "," / ")"), without recursing — unlike
// findNamedChild (parser_go_test.go), which searches the whole subtree and
// would find a nested named node inside a retagged qualified_type instead of
// the qualified_type itself.
func findFirstNamedChild(lang *gotreesitter.Language, n *gotreesitter.Node) *gotreesitter.Node {
	if n == nil {
		return nil
	}
	for i := 0; i < n.ChildCount(); i++ {
		if c := n.Child(i); c != nil && c.IsNamed() {
			return c
		}
	}
	return nil
}

// TestParseGoNewPlainTypeArgument covers gt-RUNTIME bug go-1: new(T)'s sole
// argument must be a type_identifier (matching the C oracle), not a plain
// expression identifier. See normalizeGoNewMakeTypeArgument.
func TestParseGoNewPlainTypeArgument(t *testing.T) {
	tree, lang := parseGo(t, "package main\n\nfunc f() *int {\n\tret := new(int)\n\treturn ret\n}\n")
	root := tree.RootNode()
	if root.HasError() {
		t.Fatalf("root has error flag set: %s", root.SExpr(lang))
	}

	call := findNamedChild(lang, root, "call_expression")
	if call == nil {
		t.Fatal("no call_expression found")
	}
	fn := call.ChildByFieldName("function", lang)
	if fn == nil || fn.Text(tree.Source()) != "new" {
		t.Fatalf("call_expression function = %v, want identifier \"new\"", fn)
	}

	args := call.ChildByFieldName("arguments", lang)
	if args == nil {
		t.Fatal("call_expression has no arguments")
	}
	arg := findFirstNamedChild(lang, args)
	if arg == nil {
		t.Fatal("argument_list has no named argument")
	}
	if got, want := arg.Type(lang), "type_identifier"; got != want {
		t.Fatalf("new(int) argument type = %q, want %q; tree=%s", got, want, root.SExpr(lang))
	}
	if got, want := arg.Text(tree.Source()), "int"; got != want {
		t.Fatalf("new(int) argument text = %q, want %q", got, want)
	}
}

// TestParseGoNewQualifiedTypeArgument covers the package-qualified variant
// of go-1: new(pkg.Type) must produce a qualified_type(package_identifier,
// type_identifier), matching the oracle's qualified_type shape instead of a
// selector_expression.
func TestParseGoNewQualifiedTypeArgument(t *testing.T) {
	tree, lang := parseGo(t, "package main\n\nimport \"math/big\"\n\nfunc f() *big.Int {\n\tret := new(big.Int)\n\treturn ret\n}\n")
	root := tree.RootNode()
	if root.HasError() {
		t.Fatalf("root has error flag set: %s", root.SExpr(lang))
	}

	call := findNamedChild(lang, root, "call_expression")
	if call == nil {
		t.Fatal("no call_expression found")
	}
	args := call.ChildByFieldName("arguments", lang)
	if args == nil {
		t.Fatal("call_expression has no arguments")
	}
	arg := findFirstNamedChild(lang, args)
	if arg == nil {
		t.Fatal("argument_list has no named argument")
	}
	if got, want := arg.Type(lang), "qualified_type"; got != want {
		t.Fatalf("new(big.Int) argument type = %q, want %q; tree=%s", got, want, root.SExpr(lang))
	}
	if got, want := arg.Text(tree.Source()), "big.Int"; got != want {
		t.Fatalf("new(big.Int) argument text = %q, want %q", got, want)
	}

	pkg := arg.ChildByFieldName("package", lang)
	if pkg == nil || pkg.Type(lang) != "package_identifier" || pkg.Text(tree.Source()) != "big" {
		t.Fatalf("qualified_type package field = %v, want package_identifier \"big\"", pkg)
	}
	name := arg.ChildByFieldName("name", lang)
	if name == nil || name.Type(lang) != "type_identifier" || name.Text(tree.Source()) != "Int" {
		t.Fatalf("qualified_type name field = %v, want type_identifier \"Int\"", name)
	}
}

// TestParseGoMakeNamedTypeArgument confirms the same new/make normalization
// applies to make(T, ...): only the leading type argument is retagged, and
// trailing len/cap expression arguments are left untouched.
func TestParseGoMakeNamedTypeArgument(t *testing.T) {
	tree, lang := parseGo(t, "package main\n\ntype T []int\n\nfunc f() T {\n\treturn make(T, 0)\n}\n")
	root := tree.RootNode()
	if root.HasError() {
		t.Fatalf("root has error flag set: %s", root.SExpr(lang))
	}

	call := findNamedChild(lang, root, "call_expression")
	if call == nil {
		t.Fatal("no call_expression found")
	}
	fn := call.ChildByFieldName("function", lang)
	if fn == nil || fn.Text(tree.Source()) != "make" {
		t.Fatalf("call_expression function = %v, want identifier \"make\"", fn)
	}

	args := call.ChildByFieldName("arguments", lang)
	if args == nil {
		t.Fatal("call_expression has no arguments")
	}

	type namedNode struct{ typ, text string }
	var named []namedNode
	for i := 0; i < args.ChildCount(); i++ {
		c := args.Child(i)
		if c != nil && c.IsNamed() {
			named = append(named, namedNode{typ: c.Type(lang), text: c.Text(tree.Source())})
		}
	}
	if len(named) != 2 {
		t.Fatalf("argument_list named children = %v, want 2 (type, int_literal)", named)
	}
	if named[0].typ != "type_identifier" || named[0].text != "T" {
		t.Fatalf("make(T, 0) first argument = %+v, want type_identifier \"T\"", named[0])
	}
	if named[1].typ != "int_literal" || named[1].text != "0" {
		t.Fatalf("make(T, 0) second argument = %+v, want unchanged int_literal \"0\"", named[1])
	}
}

// TestParseGoRegularCallIdentifierArgumentUnaffected is the negative-case
// sanity check for go-1: an ordinary call whose function is not literally
// "new"/"make" must keep a bare-identifier argument as a plain identifier
// (expression), never retagged to type_identifier.
func TestParseGoRegularCallIdentifierArgumentUnaffected(t *testing.T) {
	tree, lang := parseGo(t, "package main\n\nfunc g(x int) {}\n\nfunc f() {\n\tvar T int\n\tg(T)\n}\n")
	root := tree.RootNode()
	if root.HasError() {
		t.Fatalf("root has error flag set: %s", root.SExpr(lang))
	}

	call := findNamedChild(lang, root, "call_expression")
	if call == nil {
		t.Fatal("no call_expression found")
	}
	fn := call.ChildByFieldName("function", lang)
	if fn == nil || fn.Text(tree.Source()) != "g" {
		t.Fatalf("call_expression function = %v, want identifier \"g\"", fn)
	}
	args := call.ChildByFieldName("arguments", lang)
	if args == nil {
		t.Fatal("call_expression has no arguments")
	}
	arg := findFirstNamedChild(lang, args)
	if arg == nil {
		t.Fatal("argument_list has no named argument")
	}
	if got, want := arg.Type(lang), "identifier"; got != want {
		t.Fatalf("g(T) argument type = %q, want %q (must not be retagged)", got, want)
	}
}

// TestParseGoNewShadowedLocalUnaffected is a further go-1 sanity check: when
// "new" is used as a plain local (not called), it must stay a normal
// identifier/expression — the new/make normalization only fires on an
// actual call_expression whose function text is "new"/"make".
func TestParseGoNewShadowedLocalUnaffected(t *testing.T) {
	tree, lang := parseGo(t, "package main\n\nfunc f() {\n\tnew := 1\n\t_ = new\n}\n")
	root := tree.RootNode()
	if root.HasError() {
		t.Fatalf("root has error flag set: %s", root.SExpr(lang))
	}
	if call := findNamedChild(lang, root, "call_expression"); call != nil {
		t.Fatalf("unexpected call_expression found in non-call source: %s", root.SExpr(lang))
	}
}

