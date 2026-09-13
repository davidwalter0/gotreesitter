package grammars

import (
	"testing"

	gotreesitter "github.com/davidwalter0/gotreesitter"
)

// TestLeadingSiblingRecoveryBoundary pins the recovery-boundary localization
// fix: when a C / C++ file fails to parse and the engine produces a whole-file
// / top-level ERROR via error recovery (the external-scanner TokenSource /
// "blob" path), the ERROR's START must NOT over-extend backward to swallow the
// leading top-level siblings — most visibly a file's leading license comment.
// The oracle (tree-sitter-c / -cpp) keeps that leading comment as a SEPARATE
// top-level sibling BEFORE the ERROR, and starts the ERROR at the first
// genuinely-unparseable token. Regression for the round11 leading-sibling fix
// in cAcceptRootRebuild / cRecoverEOFAccept (parser_recover_c.go) and the
// panic-mode resync completed-sibling scan (parser_reduce.go).
func TestLeadingSiblingRecoveryBoundary(t *testing.T) {
	cases := []struct {
		name string
		lang string
		src  string
	}{
		// A leading comment ahead of an unparseable region. Without the fix the
		// whole file collapses into one ERROR[0-..] with the comment as its
		// first child; with the fix the comment is a separate sibling and the
		// ERROR starts at the first bad token. Each input below is verified to
		// FOLD the leading comment into the ERROR on the pre-fix engine, so the
		// assertions have teeth (they fail if the boundary fix is reverted).
		//
		// c_accept_rebuild exercises the whole-file EOF accept-root rebuild
		// (cAcceptRootRebuild): a preproc conditional whose #else body is
		// unparseable, so the file never reduces to a clean translation_unit.
		{"c_accept_rebuild", "c", "/* hdr */\n#if A\nint x;\n#else garbage @@@ )( {\n"},
		// cpp_resync exercises the panic-mode top-level resync completed-sibling
		// scan (parser_reduce.go): the leading comment must not terminate that
		// scan (extras carry no LR GOTO).
		{"cpp_resync", "cpp", "/* license */\n} ) @ int { ; )\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entry := leadingSiblingLangEntry(t, tc.lang)
			lang := entry.Language()
			parser := gotreesitter.NewParser(lang)
			ts := entry.TokenSourceFactory([]byte(tc.src), lang)
			tree, err := parser.ParseWithTokenSource([]byte(tc.src), ts)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			root := tree.RootNode()

			// The leading comment must be a genuine top-level sibling: it is the
			// first child, starts at byte 0, and is a `comment` (not an ERROR).
			if root.ChildCount() < 2 {
				t.Fatalf("root %s has %d children; want the leading comment kept as a separate sibling before the ERROR (>=2)",
					root.Type(lang), root.ChildCount())
			}
			first := root.Child(0)
			if got := first.Type(lang); got != "comment" {
				t.Fatalf("first top-level child = %q; want a separate leading `comment` sibling", got)
			}
			if first.StartByte() != 0 {
				t.Fatalf("leading comment starts at byte %d; want 0", first.StartByte())
			}
			if first.IsError() {
				t.Fatalf("leading comment was materialized as an ERROR node")
			}

			// The ERROR region must NOT begin at byte 0 (that would mean it
			// swallowed the leading comment). Find the first ERROR sibling and
			// assert it starts strictly after the comment.
			var errNode *gotreesitter.Node
			for i := 0; i < root.ChildCount(); i++ {
				if ch := root.Child(i); ch.IsError() {
					errNode = ch
					break
				}
			}
			if errNode == nil {
				t.Fatalf("expected an ERROR sibling in the recovered root %s", root.Type(lang))
			}
			if errNode.StartByte() < first.EndByte() {
				t.Fatalf("ERROR starts at byte %d, before/at the leading comment end %d — it over-extends over the leading sibling",
					errNode.StartByte(), first.EndByte())
			}
		})
	}
}

func leadingSiblingLangEntry(t *testing.T, name string) *LangEntry {
	t.Helper()
	for _, e := range AllLanguages() {
		if e.Name == name {
			ec := e
			if ec.TokenSourceFactory == nil {
				t.Fatalf("language %q has no TokenSourceFactory (blob path)", name)
			}
			return &ec
		}
	}
	t.Fatalf("language %q not registered", name)
	return nil
}
