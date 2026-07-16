package gotreesitter_test

import (
	"testing"

	gts "github.com/davidwalter0/gotreesitter"
	"github.com/davidwalter0/gotreesitter/grammars"
)

// TestCppQualifiedIdentifierCascadeRetry covers the C++ whole-file ERROR cascade
// that C++'s steady-state merge-per-key cap of 1 (effectiveParseMergePerKeyCap)
// produced for two qualified-identifier ambiguities the tree-sitter-cpp oracle
// resolves the other way:
//
//   - a call at statement scope whose arguments are all namespace-qualified
//     identifiers — `EXPECT_EQ(A::x, B::y)` — which cap=1 mis-reads as a local
//     function declaration (`A::x`/`B::y` look like qualified type names in a
//     typed, unnamed parameter list), and
//   - an out-of-line method with a namespace-qualified template return type whose
//     template argument is also qualified — `fml::RefPtr<fml::TaskRunner> C::M()`
//     — which cap=1 mis-reads as a `<`/`>` comparison chain.
//
// A single such construct cascaded the enclosing function (and, via error
// recovery, the whole file) to a root ERROR. The fix is a cpp-scoped retry-rung
// merge-per-key widening (fullParseRetryMergePerKeyOverride): a fresh cap=1 parse
// that accepts with an error retries at cppFullParseRetryMaxMergePerKey survivors,
// which keeps the C-oracle branch alive. Clean files never retry, so the tight
// cap=1 fast path is untouched.
//
// These cases parse to a root ERROR at cap=1 and must recover to an error-free
// tree through the retry. Real corpus witnesses (task_runners.cc,
// dl_matrix_color_filter.cc, dl_paint.cc, the flutter-engine unittest set) are
// covered by the cpp blob-parity corpus; these self-contained cases lock the
// retry behavior into a fast unit test.
func TestCppQualifiedIdentifierCascadeRetry(t *testing.T) {
	lang := grammars.CppLanguage()

	cases := []struct {
		name string
		src  string
	}{
		{
			// GoogleTest-shaped: EXPECT_EQ comparing two qualified enum values.
			// The exemplar inside dl_paint_unittests.cc's ConstructorDefaults test.
			"gtest_qualified_arg_call",
			"namespace flutter {\n" +
				"namespace testing {\n" +
				"void Body() {\n" +
				"  EXPECT_EQ(DlBlendMode::kDefaultMode, DlBlendMode::kSrcOver);\n" +
				"  EXPECT_EQ(DlDrawStyle::kDefaultStyle, DlDrawStyle::kFill);\n" +
				"}\n" +
				"}\n" +
				"}\n",
		},
		{
			// The minimal statement-scope form: a bare function body with one
			// all-qualified-argument call. Cascades at cap=1 in isolation.
			"bare_function_qualified_arg_call",
			"void fn() {\n  EXPECT_EQ(A::kOne, A::kTwo);\n}\n",
		},
		{
			// Out-of-line methods whose types mix a const-qualified reference
			// return and a namespace-qualified template return with a qualified
			// template argument — the task_runners.cc / GetPlatformTaskRunner
			// shape.
			"qualified_template_return_methods",
			"namespace flutter {\n" +
				"const std::string& A::GetLabel() const { return label_; }\n" +
				"fml::RefPtr<fml::TaskRunner> A::GetPlatform() const { return platform_; }\n" +
				"}\n",
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
			if root.Type(lang) == "ERROR" {
				t.Fatalf("root cascaded to ERROR, want translation_unit\n%s", root.SExpr(lang))
			}
			if root.HasError() {
				t.Fatalf("root.HasError = true, want false (cascade not cleared)\n%s", root.SExpr(lang))
			}
		})
	}

	// A genuinely clean C++ file with no qualified-argument cascade trigger must
	// still parse clean and must not be perturbed by the retry mechanism (it never
	// fires — the cap=1 parse is already error-free).
	t.Run("clean_file_untouched", func(t *testing.T) {
		src := "namespace n {\n" +
			"int add(int a, int b) { return a + b; }\n" +
			"struct S { int x; };\n" +
			"}\n"
		tree, err := gts.NewParser(lang).Parse([]byte(src))
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		defer tree.Release()
		if tree.RootNode().HasError() {
			t.Fatalf("clean file reported an error\n%s", tree.RootNode().SExpr(lang))
		}
	})
}
