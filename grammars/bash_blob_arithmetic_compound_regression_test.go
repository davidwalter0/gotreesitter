package grammars

import (
	"testing"

	"github.com/davidwalter0/gotreesitter"
)

// TestBashArithmeticCompoundCommandNotParsedAsNestedSubshell guards
// against a regression where the dedicated `(( expression ))` arithmetic
// compound-command syntax was parsed as two nested plain subshells
// (`( ( expression ) )`) instead of the grammar's dedicated
// compound_statement arithmetic form. Root cause: the external scanner's
// OPENING_PAREN active-match helper (bshScanOpeningParen, since removed)
// emitted a 1-byte '(' external token that pre-empted the internal DFA's
// longest-match tokenization of the 2-byte '((' arithmetic-command token.
func TestBashArithmeticCompoundCommandNotParsedAsNestedSubshell(t *testing.T) {
	src := []byte("((i++))\n")

	tree, lang := bashMustParseNoError(t, src)
	root := tree.RootNode()

	stmt := root.NamedChild(0)
	if stmt == nil || stmt.Type(lang) != "compound_statement" {
		t.Fatalf("expected compound_statement, got %s", root.SExpr(lang))
	}
	if got, want := stmt.Text(src), "((i++))"; got != want {
		t.Fatalf("compound_statement text = %q, want %q (full tree: %s)", got, want, root.SExpr(lang))
	}

	gotreesitter.Walk(root, func(node *gotreesitter.Node, depth int) gotreesitter.WalkAction {
		if node.IsNamed() && node.Type(lang) == "subshell" {
			t.Fatalf("unexpected subshell node in arithmetic compound command tree: %s", root.SExpr(lang))
		}
		return gotreesitter.WalkContinue
	})

	postfix := stmt.NamedChild(0)
	if postfix == nil || postfix.Type(lang) != "postfix_expression" {
		t.Fatalf("expected postfix_expression inside compound_statement, got %s", root.SExpr(lang))
	}
	if got, want := postfix.Text(src), "i++"; got != want {
		t.Fatalf("postfix_expression text = %q, want %q (full tree: %s)", got, want, root.SExpr(lang))
	}
}

// TestBashArithmeticCompoundCommandInWhileCondition exercises the
// `while (( cond )); do ... done` idiom cited in the parity report as
// one of the common real-world triggers for the nested-subshell bug.
func TestBashArithmeticCompoundCommandInWhileCondition(t *testing.T) {
	src := []byte("while (( offset < total )); do\n  offset=$((offset+1))\ndone\n")

	tree, lang := bashMustParseNoError(t, src)
	root := tree.RootNode()

	whileStmt := root.NamedChild(0)
	if whileStmt == nil || whileStmt.Type(lang) != "while_statement" {
		t.Fatalf("expected while_statement, got %s", root.SExpr(lang))
	}

	cond := whileStmt.ChildByFieldName("condition", lang)
	if cond == nil || cond.Type(lang) != "compound_statement" {
		t.Fatalf("expected compound_statement condition, got %s", root.SExpr(lang))
	}
	if got, want := cond.Text(src), "(( offset < total ))"; got != want {
		t.Fatalf("condition text = %q, want %q (full tree: %s)", got, want, root.SExpr(lang))
	}

	binExpr := cond.NamedChild(0)
	if binExpr == nil || binExpr.Type(lang) != "binary_expression" {
		t.Fatalf("expected binary_expression inside arithmetic condition, got %s", root.SExpr(lang))
	}
}

// TestBashArithmeticCompoundCommandInIfCondition exercises the
// `if (( cond )); then ... fi` idiom, the other common real-world
// trigger cited in the parity report.
func TestBashArithmeticCompoundCommandInIfCondition(t *testing.T) {
	src := []byte("if (( x > 0 )); then\n  echo yes\nfi\n")

	tree, lang := bashMustParseNoError(t, src)
	root := tree.RootNode()

	ifStmt := root.NamedChild(0)
	if ifStmt == nil || ifStmt.Type(lang) != "if_statement" {
		t.Fatalf("expected if_statement, got %s", root.SExpr(lang))
	}

	cond := ifStmt.ChildByFieldName("condition", lang)
	if cond == nil || cond.Type(lang) != "compound_statement" {
		t.Fatalf("expected compound_statement condition, got %s", root.SExpr(lang))
	}
	if got, want := cond.Text(src), "(( x > 0 ))"; got != want {
		t.Fatalf("condition text = %q, want %q (full tree: %s)", got, want, root.SExpr(lang))
	}
}
