package grammars

import (
	"testing"

	"github.com/davidwalter0/gotreesitter"
)

// TestBashEmptyValueAssignmentDoesNotSwallowCommandName guards against a
// regression where a bare `VAR=` (empty-value) assignment immediately
// followed by whitespace and a command incorrectly consumed the command
// name into the assignment's value, shifting every following word one
// position to the left. This is the `while IFS= read -r line; do ...`
// idiom -- extremely common real-world Bash. Root cause: the external
// scanner's OPENING_PAREN/ESAC active-match helpers unconditionally skip
// horizontal whitespace as a side effect before deciding whether they
// match (even when they ultimately decline), and used to run before the
// EMPTY_VALUE check. That skip consumed the whitespace right after `=`,
// so EMPTY_VALUE's own whitespace-lookahead test observed a non-whitespace
// character and declined, letting the internal DFA lex "read" as the
// assignment's value instead.
func TestBashEmptyValueAssignmentDoesNotSwallowCommandName(t *testing.T) {
	src := []byte("IFS= read -r file\n")

	tree, lang := bashMustParseNoError(t, src)
	root := tree.RootNode()

	command := root.NamedChild(0)
	if command == nil || command.Type(lang) != "command" {
		t.Fatalf("expected top-level command, got %s", root.SExpr(lang))
	}

	assignment := command.NamedChild(0)
	if assignment == nil || assignment.Type(lang) != "variable_assignment" {
		t.Fatalf("expected variable_assignment as first command child, got %s", root.SExpr(lang))
	}
	if got, want := assignment.Text(src), "IFS="; got != want {
		t.Fatalf("variable_assignment text = %q, want %q (full tree: %s)", got, want, root.SExpr(lang))
	}
	if got, want := assignment.NamedChildCount(), 1; got != want {
		t.Fatalf("variable_assignment named child count = %d, want %d (only variable_name, no value): %s",
			got, want, root.SExpr(lang))
	}

	commandName := command.ChildByFieldName("name", lang)
	if commandName == nil || commandName.Type(lang) != "command_name" {
		t.Fatalf("expected command_name field on command, got %s", root.SExpr(lang))
	}
	if got, want := commandName.Text(src), "read"; got != want {
		t.Fatalf("command_name text = %q, want %q (full tree: %s)", got, want, root.SExpr(lang))
	}

	var args []string
	for i := 0; i < command.ChildCount(); i++ {
		child := command.Child(i)
		if child != nil && command.FieldNameForChild(i, lang) == "argument" {
			args = append(args, child.Text(src))
		}
	}
	if got, want := len(args), 2; got != want {
		t.Fatalf("argument count = %d, want %d, got %v (full tree: %s)", got, want, args, root.SExpr(lang))
	}
	if args[0] != "-r" || args[1] != "file" {
		t.Fatalf("arguments = %v, want [-r file] (full tree: %s)", args, root.SExpr(lang))
	}
}

// TestBashEmptyValueAssignmentInsideWhileReadIdiom exercises the exact
// idiom the bare-command case above stands in for: `while IFS= read -r
// line; do ... done`. Regression coverage for the real-world instance
// cited in the parity report (.emacs.d/30/check-lexical-binding.sh).
func TestBashEmptyValueAssignmentInsideWhileReadIdiom(t *testing.T) {
	src := []byte("while IFS= read -r file; do\n  echo \"$file\"\ndone\n")

	tree, lang := bashMustParseNoError(t, src)
	root := tree.RootNode()

	whileStmt := root.NamedChild(0)
	if whileStmt == nil || whileStmt.Type(lang) != "while_statement" {
		t.Fatalf("expected while_statement, got %s", root.SExpr(lang))
	}

	var command *gotreesitter.Node
	gotreesitter.Walk(whileStmt, func(node *gotreesitter.Node, depth int) gotreesitter.WalkAction {
		if node.IsNamed() && node.Type(lang) == "command" {
			command = node
			return gotreesitter.WalkStop
		}
		return gotreesitter.WalkContinue
	})
	if command == nil {
		t.Fatalf("expected command inside while condition, got %s", root.SExpr(lang))
	}

	assignment := command.NamedChild(0)
	if assignment == nil || assignment.Type(lang) != "variable_assignment" {
		t.Fatalf("expected variable_assignment as first command child, got %s", root.SExpr(lang))
	}
	if got, want := assignment.Text(src), "IFS="; got != want {
		t.Fatalf("variable_assignment text = %q, want %q (full tree: %s)", got, want, root.SExpr(lang))
	}

	commandName := command.ChildByFieldName("name", lang)
	if commandName == nil || commandName.Text(src) != "read" {
		text := ""
		if commandName != nil {
			text = commandName.Text(src)
		}
		t.Fatalf("command_name text = %q, want %q (full tree: %s)", text, "read", root.SExpr(lang))
	}
}
