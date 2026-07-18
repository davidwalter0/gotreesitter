package grammars

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/davidwalter0/gotreesitter"
)

// tagStrings renders tags as sorted "Kind:Name" strings so assertions are
// independent of match-emission order (which tracks source position, not
// pattern-declaration order, and is not part of the Tagger's documented
// contract).
func tagStrings(tags []gotreesitter.Tag) []string {
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		out = append(out, fmt.Sprintf("%s:%s", tag.Kind, tag.Name))
	}
	sort.Strings(out)
	return out
}

func resolveTaggerForTest(t *testing.T, langName string) *gotreesitter.Tagger {
	t.Helper()
	entry := DetectLanguageByName(langName)
	if entry == nil {
		t.Fatalf("DetectLanguageByName(%q) returned nil", langName)
	}
	query := ResolveTagsQuery(*entry)
	if strings.TrimSpace(query) == "" {
		t.Fatalf("ResolveTagsQuery(%q) returned empty", langName)
	}
	tagger, err := gotreesitter.NewTagger(entry.Language(), query)
	if err != nil {
		t.Fatalf("NewTagger(%q): %v", langName, err)
	}
	return tagger
}

func assertTagsEqual(t *testing.T, tags []gotreesitter.Tag, want []string) {
	t.Helper()
	sortedWant := append([]string(nil), want...)
	sort.Strings(sortedWant)
	got := tagStrings(tags)
	if len(got) != len(sortedWant) {
		t.Fatalf("tag count = %d, want %d\n got=%v\nwant=%v", len(got), len(sortedWant), got, sortedWant)
	}
	for i := range got {
		if got[i] != sortedWant[i] {
			t.Fatalf("tags mismatch at %d: got %q, want %q\n full got=%v\nfull want=%v", i, got[i], sortedWant[i], got, sortedWant)
		}
	}
}

func assertTagAbsent(t *testing.T, tags []gotreesitter.Tag, kindColonName string) {
	t.Helper()
	for _, s := range tagStrings(tags) {
		if s == kindColonName {
			t.Fatalf("expected %q to be absent, but it was captured; full tags=%v", kindColonName, tagStrings(tags))
		}
	}
}

func countTag(tags []gotreesitter.Tag, kindColonName string) int {
	n := 0
	for _, s := range tagStrings(tags) {
		if s == kindColonName {
			n++
		}
	}
	return n
}

// TestTagsQueryOverridePython closes the gap where module-level assignments
// (constants like `PREFIX = "hi"` or `enabled = True`) were invisible to the
// tags query -- only function/class definitions and call references were
// captured before. The query anchors to `(module (assignment ...))`, so a
// same-shaped assignment nested inside a function body must NOT be captured.
func TestTagsQueryOverridePython(t *testing.T) {
	tagger := resolveTaggerForTest(t, "python")

	src := `PREFIX = "hi"

enabled = True

def foo():
    x = 1
    return x
`
	tags := tagger.Tag([]byte(src))

	assertTagsEqual(t, tags, []string{
		"definition.constant:PREFIX",
		"definition.constant:enabled",
		"definition.function:foo",
	})
	assertTagAbsent(t, tags, "definition.constant:x")
	assertTagAbsent(t, tags, "definition.variable:x")
}

// TestTagsQueryOverrideJavaScript closes the gap where top-level
// const/let bindings -- including arrow-function and function-expression
// assignments -- were invisible to the tags query. It also guards against
// the over-capture risk documented on the "javascript" override: the
// Tagger (gotreesitter.Tagger.tagTree) applies no match precedence or
// dedupe, so a naive catch-all constant pattern would double-tag every
// function-valued declarator as BOTH definition.function AND
// definition.constant. The value-type allowlist on the constant pattern
// must keep the two kinds mutually exclusive.
func TestTagsQueryOverrideJavaScript(t *testing.T) {
	tagger := resolveTaggerForTest(t, "javascript")

	src := `const add = (a, b) => a + b;
const MAX = 5;
const greet = function(name) { return name; };
function foo() {}
let enabled = true;
`
	tags := tagger.Tag([]byte(src))

	assertTagsEqual(t, tags, []string{
		"definition.function:add",
		"definition.constant:MAX",
		"definition.function:greet",
		"definition.function:foo",
		"definition.constant:enabled",
	})

	// The central over-capture guard: "add" and "greet" must each produce
	// exactly one tag (definition.function), never also
	// definition.constant.
	if n := countTag(tags, "definition.function:add"); n != 1 {
		t.Fatalf("definition.function:add count = %d, want 1 (tags=%v)", n, tagStrings(tags))
	}
	assertTagAbsent(t, tags, "definition.constant:add")
	if n := countTag(tags, "definition.function:greet"); n != 1 {
		t.Fatalf("definition.function:greet count = %d, want 1 (tags=%v)", n, tagStrings(tags))
	}
	assertTagAbsent(t, tags, "definition.constant:greet")
}

// TestTagsQueryOverrideRust closes the gap where const/static items were
// invisible to the tags query. struct/enum/trait keep resolving to
// definition.type, unchanged from before this override.
func TestTagsQueryOverrideRust(t *testing.T) {
	tagger := resolveTaggerForTest(t, "rust")

	src := `const MAX: i32 = 5;
static NAME: &str = "hi";
struct Foo {}
`
	tags := tagger.Tag([]byte(src))

	assertTagsEqual(t, tags, []string{
		"definition.constant:MAX",
		"definition.constant:NAME",
		"definition.type:Foo",
	})
}

// TestTagsQueryOverrideC closes two gaps: #define macros were invisible,
// and file-scope variable declarations were invisible. The file-scope
// variable patterns are anchored to `(translation_unit (declaration ...))`
// -- declaration must be a DIRECT child of the file root -- which is what
// excludes function-local declarations (nested under
// function_definition -> compound_statement, never a direct child of
// translation_unit). The patterns also use the `declarator:` field
// explicitly so that, for `int a = b;`, only the declared name "a" is
// captured and not the initializer-side reference "b" (also typed
// `identifier` by the grammar, but under the `value:` field).
func TestTagsQueryOverrideC(t *testing.T) {
	tagger := resolveTaggerForTest(t, "c")

	src := `#define MAX 5

int global_var = 10;
int b = 3;
int a = b;
int x;

int add(int a, int b) {
    int local_var = a + b;
    return local_var;
}
`
	tags := tagger.Tag([]byte(src))

	assertTagsEqual(t, tags, []string{
		"definition.variable:global_var",
		"definition.variable:b",
		"definition.variable:a",
		"definition.variable:x",
		"definition.constant:MAX",
		"definition.function:add",
	})

	// Scope-anchoring guard: the local variable inside add() must not leak.
	assertTagAbsent(t, tags, "definition.variable:local_var")

	// RHS-reference guard: "int a = b;" must not spuriously produce a
	// second definition.variable:b (the value-side identifier "b" is a
	// reference, not a second declaration of "b").
	if n := countTag(tags, "definition.variable:b"); n != 1 {
		t.Fatalf("definition.variable:b count = %d, want 1 (tags=%v)", n, tagStrings(tags))
	}
	if n := countTag(tags, "definition.variable:a"); n != 1 {
		t.Fatalf("definition.variable:a count = %d, want 1 (tags=%v)", n, tagStrings(tags))
	}

	// The parameter identifiers named "a"/"b" inside add(int a, int b) must
	// not be mistaken for file-scope declarations either -- already implied
	// by the exact-set assertion above (which fixes the total count), but
	// asserted directly here for clarity.
	if n := countTag(tags, "definition.function:add"); n != 1 {
		t.Fatalf("definition.function:add count = %d, want 1 (tags=%v)", n, tagStrings(tags))
	}
}

// TestTagsQueryOverridesCompile is a narrow, override-specific companion to
// the broader TestAllTagsQueriesCompile in highlight_validation_test.go --
// it pins the exact four languages this task touched so a future change to
// any of them fails fast and close to the cause.
func TestTagsQueryOverridesCompile(t *testing.T) {
	for _, name := range []string{"python", "javascript", "rust", "c"} {
		name := name
		t.Run(name, func(t *testing.T) {
			entry := DetectLanguageByName(name)
			if entry == nil {
				t.Fatalf("DetectLanguageByName(%q) returned nil", name)
			}
			query := ResolveTagsQuery(*entry)
			if _, err := gotreesitter.NewTagger(entry.Language(), query); err != nil {
				t.Fatalf("compile tags query for %q: %v", name, err)
			}
		})
	}
}
