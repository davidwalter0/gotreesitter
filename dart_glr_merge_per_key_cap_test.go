package gotreesitter_test

import (
	"fmt"
	"strings"
	"testing"

	gts "github.com/davidwalter0/gotreesitter"
	"github.com/davidwalter0/gotreesitter/grammars"
)

// This file pins the fix for the Dart full-parse GLR merge-per-key cap
// narrowing removed from effectiveParseMergePerKeyCap's "dart" case
// (parser_retry.go). That narrowing hard-capped Dart's same-key merge
// survivors at 3, half the global default (maxStacksPerMergeKey = 6,
// glr.go). Some real Dart inputs need all 6 same-key survivors to select
// the correct GLR branch; at cap=3 the correct branch is culled and the
// ENTIRE FILE collapses to a single root ERROR node, silently deleting
// every declaration from the parse result (mcp-ast todo 1d758046).
//
// Trigger shape (valid Dart, verified against Dart SDK 3.12.2 `dart
// format`): 5+ nested constructor calls where an inner call carries a
// `const` NAMED ARGUMENT, written across multiple lines. Single-line
// formatting of the identical expression does not trigger it.
//
// dartNestedConstTrigger builds that shape parametrized by nesting depth,
// followed by 5 canary classes (G0..G4) so a collapse is visible as lost
// declarations rather than just a flipped error flag. depth must be >= 1.
func dartNestedConstTrigger(depth int) []byte {
	var b strings.Builder
	b.WriteString("class Bad {\n  Widget build() {\n    return ")
	for i := 0; i < depth; i++ {
		fmt.Fprintf(&b, "Padding(\n      padding: const EdgeInsets.all(%d),\n      child: ", depth-1-i)
	}
	b.WriteString("const Text('leaf')")
	for i := 0; i < depth; i++ {
		b.WriteString(",\n    )")
	}
	b.WriteString(";\n  }\n}\n\n")
	for i := 0; i < 5; i++ {
		fmt.Fprintf(&b, "class G%d { int f%d() => %d; }\n", i, i, i)
	}
	return []byte(b.String())
}

// dartRootCollapsedToWholeFileError reports the exact collapse signature
// from the mcp-ast investigation: the root has exactly one child, and that
// child is a single ERROR node spanning (to within trailing whitespace) the
// entire source. Trailing "\n"/"\r"/"\t"/" " is excluded from the byte
// comparison because ordinary trailing-trivia handling legitimately leaves
// it outside the ERROR node's span even in a genuine whole-file collapse
// (observed: a 623-byte source's collapse ERROR ends at byte 622, the exact
// width of its trailing newline).
func dartRootCollapsedToWholeFileError(root *gts.Node, src []byte) bool {
	if root == nil || root.ChildCount() != 1 {
		return false
	}
	child := root.Child(0)
	if child == nil || !child.IsError() || child.StartByte() != 0 {
		return false
	}
	trimmedLen := len(strings.TrimRight(string(src), "\n\r\t "))
	return int(child.EndByte()) >= trimmedLen
}

// TestDartNestedConstTriggerParsesCleanlyAcrossNestingDepths is the
// regression pin: the minimal trigger at nesting depths 1..8, asserting the
// depth at which parsing currently collapses. Measured against the fixed
// build (cap=6, the unmodified global default): depths 1..5 parse cleanly
// with all 6 declarations reachable, matching the originally-diagnosed
// minimal trigger (depth 5) and the "6 same-key survivors needed" finding.
// Depths 6..8 STILL collapse post-fix -- each added nesting level appears to
// need one more same-key survivor than the last (depth needs depth+1
// survivors), so depth >= 6 exceeds even the widened cap=6 default. This is
// not swept under the rug: it is the same residual gap called out for
// settings_screen.dart (needs far wider caps than this fix provides, and
// cap behavior is non-monotone globally -- see the commit message). Do not
// "fix" this sub-test by raising the global caps; that path is independently
// known to break six other tests (see the commit message).
//
// UPDATE 2026-09-13, after syncing onto upstream main (21e4312b): depths 6..8
// now parse cleanly too, with no cap change on the fork side. Upstream's own
// Dart GLR work landed in between (0d1cd052 starts Dart's full-parse retry at
// six GLR stacks, the same "six survivors" finding reached independently), so
// the residual collapse is gone at every depth pinned here. The rows below
// assert clean at all eight depths; the collapse-shape branch is kept so a
// future regression that reintroduces it reports the exact signature.
//
// On stock v0.38.2 (the old dart-specific cap=3 narrowing) this test fails
// starting at depth 1, not depth 6 -- see the LOAD-BEARING verification
// recorded in the commit message for the exact stock-vs-fixed comparison.
func TestDartNestedConstTriggerParsesCleanlyAcrossNestingDepths(t *testing.T) {
	lang := grammars.DartLanguage()
	const wantDecls = 6 // 1 "Bad" class + 5 canary classes (G0..G4)

	tests := []struct {
		depth     int
		wantClean bool
	}{
		{depth: 1, wantClean: true},
		{depth: 2, wantClean: true},
		{depth: 3, wantClean: true},
		{depth: 4, wantClean: true},
		{depth: 5, wantClean: true},
		{depth: 6, wantClean: true},
		{depth: 7, wantClean: true},
		{depth: 8, wantClean: true},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("depth=%d", tc.depth), func(t *testing.T) {
			src := dartNestedConstTrigger(tc.depth)
			parser := gts.NewParser(lang)
			tree, err := parser.Parse(src)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			defer tree.Release()

			root := tree.RootNode()
			if root == nil {
				t.Fatal("missing root node")
			}

			if !tc.wantClean {
				// Pin the known-remaining collapse rather than ignore it:
				// still a whole-file ERROR at this depth, post-fix.
				if !root.HasError() {
					t.Fatalf("depth %d: expected this depth to still collapse post-fix (see doc comment), but the parse was clean; tree=%s", tc.depth, root.SExpr(lang))
				}
				if !dartRootCollapsedToWholeFileError(root, src) {
					t.Fatalf("depth %d: expected the known whole-file-ERROR collapse shape, got a different error shape; tree=%s", tc.depth, root.SExpr(lang))
				}
				return
			}

			if root.HasError() {
				t.Fatalf("depth %d: expected a clean parse, root has error; tree=%s", tc.depth, root.SExpr(lang))
			}
			if got := root.NamedChildCount(); got != wantDecls {
				t.Fatalf("depth %d: NamedChildCount() = %d, want %d; tree=%s", tc.depth, got, wantDecls, root.SExpr(lang))
			}
			// The canonical symptom is that everything AFTER the trigger
			// disappears, so explicitly confirm the last declaration
			// (the 5th canary, G4) is present and correctly typed rather
			// than trusting the aggregate count alone.
			last := root.NamedChild(wantDecls - 1)
			if last == nil || last.Type(lang) != "class_definition" {
				t.Fatalf("depth %d: last named child = %v, want class_definition (G4 must survive)", tc.depth, last)
			}
		})
	}
}

// TestDartNestedConstTriggerRootDoesNotDegenerateToWholeFileError is the
// PARSE-SHAPE test: on the fixed build, the canonical 5-deep trigger must
// not collapse the root to a single whole-file ERROR child, and every
// declaration after the trigger must remain reachable.
func TestDartNestedConstTriggerRootDoesNotDegenerateToWholeFileError(t *testing.T) {
	const depth = 5 // matches the originally-diagnosed minimal trigger exactly
	const wantDecls = 6

	src := dartNestedConstTrigger(depth)
	lang := grammars.DartLanguage()
	parser := gts.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	defer tree.Release()

	root := tree.RootNode()
	if root == nil {
		t.Fatal("missing root node")
	}
	if dartRootCollapsedToWholeFileError(root, src) {
		t.Fatalf("root degenerated to a single whole-file ERROR child; tree=%s", root.SExpr(lang))
	}
	if got := root.NamedChildCount(); got != wantDecls {
		t.Fatalf("NamedChildCount() = %d, want %d (declarations after the trigger must survive); tree=%s", got, wantDecls, root.SExpr(lang))
	}
	for i := 0; i < wantDecls; i++ {
		decl := root.NamedChild(i)
		if decl == nil || decl.Type(lang) != "class_definition" {
			t.Fatalf("named child %d = %v, want class_definition; tree=%s", i, decl, root.SExpr(lang))
		}
	}
}

// TestDartPlainSyntaxErrorStaysLocalized is the CONTRAST test: an ordinary,
// unambiguous syntax error must trigger normal, LOCALIZED error recovery,
// not the whole-file collapse mode exercised above. This protects the
// distinction between the two failure classes -- a fix that accidentally
// broke ordinary recovery while helping the cap-exhaustion case would
// otherwise go unnoticed, since both symptoms are "some declarations
// missing".
func TestDartPlainSyntaxErrorStaysLocalized(t *testing.T) {
	var b strings.Builder
	const before = 10
	const after = 10
	for i := 0; i < before; i++ {
		fmt.Fprintf(&b, "class G%d { int f%d() => %d; }\n", i, i, i)
	}
	b.WriteString("class Broken { void f() { @@@ } }\n")
	for i := 0; i < after; i++ {
		fmt.Fprintf(&b, "class H%d { int g%d() => %d; }\n", i, i, i)
	}
	src := []byte(b.String())
	wantDecls := before + 1 + after // all canaries plus the broken class itself

	lang := grammars.DartLanguage()
	parser := gts.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	defer tree.Release()

	root := tree.RootNode()
	if root == nil {
		t.Fatal("missing root node")
	}
	if !root.HasError() {
		t.Fatalf("expected the deliberate @@@ syntax error to be flagged; tree=%s", root.SExpr(lang))
	}
	if dartRootCollapsedToWholeFileError(root, src) {
		t.Fatalf("a single localized syntax error must not collapse the whole file to one ERROR child; tree=%s", root.SExpr(lang))
	}
	if got := root.NamedChildCount(); got != wantDecls {
		t.Fatalf("NamedChildCount() = %d, want %d: ordinary recovery must keep every declaration reachable; tree=%s", got, wantDecls, root.SExpr(lang))
	}
	// Explicitly confirm the LAST canary (H9, well past the broken class)
	// is present and correctly typed -- the exact declaration a
	// whole-file collapse would have destroyed.
	last := root.NamedChild(wantDecls - 1)
	if last == nil || last.Type(lang) != "class_definition" {
		t.Fatalf("last named child = %v, want class_definition (H9 must survive a localized error)", last)
	}
}

// TestDartMergePerKeyCapOverflowObservableUnderNarrowCap is the
// LOGGING/METRICS requirement: cap exhaustion must be observable rather
// than silent. It uses the pre-existing GOT_GLR_MAX_MERGE_PER_KEY override
// (parser_config.go) to force the SAME narrow cap the removed dart-specific
// case used to hard-code, independent of language, and checks that the new
// always-on counter (ParseRuntime.MergePerKeyCapOverflow /
// Tree.MergePerKeyCapExceeded) fires exactly when the cap is too tight and
// is consistent with a clean parse at the language default.
func TestDartMergePerKeyCapOverflowObservableUnderNarrowCap(t *testing.T) {
	gts.ResetParseEnvConfigCacheForTests()
	defer gts.ResetParseEnvConfigCacheForTests()

	const depth = 5
	src := dartNestedConstTrigger(depth)
	lang := grammars.DartLanguage()

	t.Run("forced_cap_3_collapses_and_is_observed", func(t *testing.T) {
		t.Setenv("GOT_GLR_MAX_MERGE_PER_KEY", "3")
		gts.ResetParseEnvConfigCacheForTests()

		parser := gts.NewParser(lang)
		tree, err := parser.Parse(src)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		defer tree.Release()

		root := tree.RootNode()
		if root == nil || !root.HasError() {
			t.Fatalf("expected a forced cap=3 parse of the 5-deep trigger to error; tree=%v", root)
		}
		if got := tree.ParseRuntime().MergePerKeyCapOverflow; got == 0 {
			t.Fatalf("ParseRuntime().MergePerKeyCapOverflow = 0 under forced cap=3, want > 0: cap exhaustion did not get recorded for a parse known to need it")
		}
		if !tree.MergePerKeyCapExceeded() {
			t.Fatalf("MergePerKeyCapExceeded() = false under forced cap=3, want true")
		}
	})

	t.Run("language_default_cap_is_clean_post_fix", func(t *testing.T) {
		t.Setenv("GOT_GLR_MAX_MERGE_PER_KEY", "")
		gts.ResetParseEnvConfigCacheForTests()

		parser := gts.NewParser(lang)
		tree, err := parser.Parse(src)
		if err != nil {
			t.Fatalf("Parse() error = %v", err)
		}
		defer tree.Release()

		root := tree.RootNode()
		if root == nil || root.HasError() {
			t.Fatalf("expected the post-fix language-default-cap parse to be clean; root=%v", root)
		}
	})
}
