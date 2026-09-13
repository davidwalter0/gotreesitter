package grammars

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/davidwalter0/gotreesitter"
)

// TestLooksLikeCppHeader exercises the marker heuristic directly against
// synthetic snippets covering each marker, plus the deliberately-conservative
// negative cases the task calls out explicitly (bare "class"/"new" used as C
// identifiers, and a C header wrapped in the standard
// "#ifdef __cplusplus / extern \"C\" {" idiom).
func TestLooksLikeCppHeader(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name: "template declaration",
			content: `
#ifndef FOO_H
#define FOO_H
template <typename T>
T max_of(T a, T b) { return a > b ? a : b; }
#endif
`,
			want: true,
		},
		{
			name: "namespace with body",
			content: `
#ifndef FOO_H
#define FOO_H
namespace foo {
int bar(int x);
}
#endif
`,
			want: true,
		},
		{
			name: "std:: usage",
			content: `
#ifndef FOO_H
#define FOO_H
void greet(std::string name);
#endif
`,
			want: true,
		},
		{
			name: "class with body",
			content: `
#ifndef FOO_H
#define FOO_H
class Foo {
public:
    Foo();
    ~Foo();
};
#endif
`,
			want: true,
		},
		{
			name: "plain C header — typedef struct, define, prototypes",
			content: `
#ifndef FOO_H
#define FOO_H

typedef struct {
    int x;
    int y;
} point_t;

#define MAX_POINTS 64

int point_distance(const point_t *a, const point_t *b);
void point_init(point_t *p, int x, int y);

#endif /* FOO_H */
`,
			want: false,
		},
		{
			name: "extern C with c++ body — unconditional, no __cplusplus guard",
			content: `
#ifndef FOO_H
#define FOO_H
extern "C" {
class FooHandle;
FooHandle *foo_create(void);
void foo_destroy(FooHandle *h);
}
#endif
`,
			want: true,
		},
		{
			name: "class and new used as plain C identifiers, no other markers",
			content: `
#ifndef FOO_H
#define FOO_H

/* "class" and "new" are not reserved words in C and are legal identifiers */
int class;
int new;

int compute(int class, int new);

#endif
`,
			want: false,
		},
		{
			name: "ifdef __cplusplus / extern C wrapper idiom stays C",
			content: `
#ifndef FOO_H
#define FOO_H

#ifdef __cplusplus
extern "C" {
#endif

int foo_init(void);
void foo_shutdown(void);

#ifdef __cplusplus
}
#endif

#endif /* FOO_H */
`,
			want: false,
		},
		{
			name: "cplusplus guard with real namespace/using shim stays C (libintl.h pattern)",
			content: `
#ifndef FOO_H
#define FOO_H

extern int foo_fprintf(void *stream, const char *format, ...);

#if defined __cplusplus
namespace std {
using ::foo_fprintf;
}
#endif

#endif
`,
			want: false,
		},
		{
			name: "cpp-only stdlib include with no extension",
			content: `
#ifndef FOO_H
#define FOO_H
#include <vector>
void store(int x);
#endif
`,
			want: true,
		},
		{
			name: "scope resolution operator outside comments",
			content: `
#ifndef FOO_H
#define FOO_H
int Foo::bar(int x);
#endif
`,
			want: true,
		},
		{
			name: "scope resolution mentioned only in a comment stays C",
			content: `
#ifndef FOO_H
#define FOO_H
/* Whether struct sockaddr::__ss_family exists */
typedef struct {
    int family;
} sockaddr_shim_t;
#endif
`,
			want: false,
		},
		{
			name: "using namespace",
			content: `
#ifndef FOO_H
#define FOO_H
using namespace std;
void greet(string name);
#endif
`,
			want: true,
		},
		{
			name: "access specifier without class keyword nearby",
			content: `
#ifndef FOO_H
#define FOO_H
struct Widget {
protected:
    int state;
};
#endif
`,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeCppHeader([]byte(tt.content))
			if got != tt.want {
				t.Errorf("looksLikeCppHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDetectLanguageWithContent checks the wiring: a bare ".h" file resolves
// to "c" or "cpp" depending on content, while every other extension (and
// DetectLanguage without content) is unaffected.
func TestDetectLanguageWithContent(t *testing.T) {
	cppHeader := []byte(`
#ifndef FOO_H
#define FOO_H
namespace foo {
class Widget {
public:
    Widget();
};
}
#endif
`)
	cHeader := []byte(`
#ifndef FOO_H
#define FOO_H
typedef struct { int x; } point_t;
void point_init(point_t *p);
#endif
`)

	tests := []struct {
		name     string
		filename string
		content  []byte
		want     string // expected LangEntry.Name, "" for nil
	}{
		{"ambiguous .h with cpp content", "widget.h", cppHeader, "cpp"},
		{"ambiguous .h with c content", "point.h", cHeader, "c"},
		{"ambiguous .h with no markers at all", "empty.h", []byte("\n"), "c"},
		{".c is never reclassified even with cpp-looking content", "widget.c", cppHeader, "c"},
		{".hpp is unaffected (already cpp via extension)", "widget.hpp", cHeader, "cpp"},
		{".hh is unaffected (already cpp via extension)", "widget.hh", cHeader, "cpp"},
		{".hxx is unaffected (already cpp via extension)", "widget.hxx", cHeader, "cpp"},
		{"unknown extension stays nil", "widget.qqzz", cppHeader, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectLanguageWithContent(tt.filename, tt.content)
			if tt.want == "" {
				if got != nil {
					t.Fatalf("DetectLanguageWithContent(%q) = %+v, want nil", tt.filename, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("DetectLanguageWithContent(%q) = nil, want %q", tt.filename, tt.want)
			}
			if got.Name != tt.want {
				t.Errorf("DetectLanguageWithContent(%q).Name = %q, want %q", tt.filename, got.Name, tt.want)
			}
		})
	}
}

// TestDetectLanguageUnaffectedByContent confirms the plain, content-free
// DetectLanguage(filename) still resolves ".h" to "c" unconditionally —
// content-based detection is strictly additive via DetectLanguageWithContent.
func TestDetectLanguageUnaffectedByContent(t *testing.T) {
	entry := DetectLanguage("widget.h")
	if entry == nil || entry.Name != "c" {
		t.Fatalf("DetectLanguage(%q) = %+v, want c (back-compat, no content available)", "widget.h", entry)
	}
}

// TestParseFileRealCorpusCppHeaderParsesWithCppGrammar exercises the full
// ParseFile pipeline (not just detection) against a real ambiguous C++
// header, confirming the content-aware detection actually changes which
// grammar parses the file, not merely which LangEntry gets returned.
func TestParseFileRealCorpusCppHeaderParsesWithCppGrammar(t *testing.T) {
	const corpusDir = "/home/david/.claude/jobs/f84fc485/tmp/blob-parity/corpus/c"
	name := "home__david__src__ghostty__src__simd__index_of.h"
	data, err := os.ReadFile(filepath.Join(corpusDir, name))
	if err != nil {
		t.Skipf("contamination corpus not present at %s: %v", corpusDir, err)
	}

	entry := DetectLanguageWithContent(name, data)
	if entry == nil || entry.Name != "cpp" {
		t.Fatalf("DetectLanguageWithContent(%s) = %+v, want cpp", name, entry)
	}

	tree, err := ParseFile(name, data)
	if err != nil {
		t.Fatalf("ParseFile(%s) with cpp grammar: %v", name, err)
	}
	defer tree.Release()
	root := tree.RootNode()
	if root == nil {
		t.Fatalf("ParseFile(%s) returned a bound tree with nil root node", name)
	}

	// Confirm the C++ grammar (not C) actually parsed the file: walk the
	// tree looking for node types that only the C++ grammar produces (the
	// C grammar has no equivalent node kinds at all). index_of.h's
	// "template <class D, ...>" functions and "namespace ghostty { ... }"
	// blocks surface as template_function / qualified_identifier /
	// namespace_alias_definition nodes; any one of them proves the C++
	// grammar ran. (The file's file-guard `#if defined(X) == defined(Y)`
	// preprocessor idiom trips up this grammar's error recovery — see the
	// top-level ERROR node — but that's a pre-existing parser-quality gap
	// unrelated to language detection, which is what this test verifies.)
	cppOnlyNodeTypes := map[string]bool{
		"template_function":          true,
		"qualified_identifier":       true,
		"namespace_alias_definition": true,
		"type_parameter_declaration": true,
	}
	var found string
	stack := []*gotreesitter.Node{root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n == nil {
			continue
		}
		if ty := tree.NodeType(n); cppOnlyNodeTypes[ty] {
			found = ty
			break
		}
		for i := 0; i < n.ChildCount(); i++ {
			stack = append(stack, n.Child(i))
		}
	}
	if found == "" {
		t.Errorf("ParseFile(%s): expected a C++-only node type (template_function/qualified_identifier/namespace_alias_definition/type_parameter_declaration), none found — root type %q", name, tree.NodeType(root))
	}
}

// TestDetectLanguageWithContentRealCorpus replays the heuristic against real
// contamination-corpus files gathered from ghostty's vendored C/C++
// dependency tree (ImageMagick's Magick++ vs. magick/wand C API, OpenEXR's
// Imath, ghostty's own SIMD index_of.h, and system EGL/GLES/GLX headers).
// Skips gracefully if the corpus directory isn't present on this machine —
// it is scratch data outside the repo, not a build dependency.
func TestDetectLanguageWithContentRealCorpus(t *testing.T) {
	const corpusDir = "/home/david/.claude/jobs/f84fc485/tmp/blob-parity/corpus/c"
	if _, err := os.Stat(corpusDir); err != nil {
		t.Skipf("contamination corpus not present at %s: %v", corpusDir, err)
	}

	// Real C++ headers that must now detect as cpp despite the .h extension.
	wantCPP := []string{
		"home__david__src__ghostty__src__simd__index_of.h",
		"usr__include__Imath__ImathBox.h",
		"usr__include__ImageMagick-6__Magick++__Blob.h",
		"usr__include__ImageMagick-6__Magick++__Color.h",
		"usr__include__ImageMagick-6__Magick++__Exception.h",
		"usr__include__ImageMagick-6__Magick++__Geometry.h",
		"usr__include__ImageMagick-6__Magick++__Montage.h",
		"usr__include__ImageMagick-6__Magick++__Pixels.h",
	}

	// Genuine C headers that must stay c, including ones that lean on the
	// "#ifdef __cplusplus / extern \"C\"" idiom (EGL/GLES/GLX, ImageMagick's
	// C-facing magick/wand headers) or that contain guarded, conditional
	// C++ shims without being C++ headers themselves (glibc's libintl.h).
	wantC := []string{
		"usr__include__EGL__egl.h",
		"usr__include__GLES2__gl2.h",
		"usr__include__GL__glx.h",
		"home__david__src__ghostty__pkg__fontconfig__override__include__fcobjshash.h",
		"home__david__src__ghostty__pkg__libpng__pnglibconf.h",
		"home__david__src__ghostty__pkg__libintl__libintl.h",
		"home__david__src__ghostty__pkg__libxml2__override__include__libxml__xmlversion.h",
		"home__david__src__ghostty__pkg__libxml2__override__config__posix__config.h",
		"usr__include__ImageMagick-6__magick__animate.h",
		"usr__include__ImageMagick-6__magick__color.h",
		"usr__include__ImageMagick-6__magick__MagickCore.h",
		"usr__include__ImageMagick-6__wand__magick-image.h",
	}

	readCorpusFile := func(t *testing.T, name string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(corpusDir, name))
		if err != nil {
			t.Fatalf("read corpus file %s: %v", name, err)
		}
		return data
	}

	for _, name := range wantCPP {
		t.Run("cpp/"+name, func(t *testing.T) {
			data := readCorpusFile(t, name)
			entry := DetectLanguageWithContent(name, data)
			if entry == nil || entry.Name != "cpp" {
				got := "nil"
				if entry != nil {
					got = entry.Name
				}
				t.Errorf("DetectLanguageWithContent(%s) = %s, want cpp", name, got)
			}
		})
	}

	for _, name := range wantC {
		t.Run("c/"+name, func(t *testing.T) {
			data := readCorpusFile(t, name)
			entry := DetectLanguageWithContent(name, data)
			if entry == nil || entry.Name != "c" {
				got := "nil"
				if entry != nil {
					got = entry.Name
				}
				t.Errorf("DetectLanguageWithContent(%s) = %s, want c", name, got)
			}
		})
	}
}
