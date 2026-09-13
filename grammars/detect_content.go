package grammars

import (
	"bytes"
	"path"
	"strings"
)

// cppHeaderScanLimit bounds how much of a file's content is scanned when
// resolving the ambiguous ".h" extension. This is a plain byte scan, not a
// tokenizer, so cost is linear in the bytes scanned; capping it keeps
// pathological (huge, generated) headers cheap to classify without changing
// the answer in practice — the markers we look for (an early #include, a
// namespace/class opener, an access specifier) all live near the top of a
// real header.
const cppHeaderScanLimit = 64 * 1024

// cppOnlyStdlibHeaders lists C++ standard library header names that have no
// C equivalent and, distinctively, no filename extension (e.g.
// "#include <vector>"). A real C header always spells its includes with an
// extension (<stdio.h>, <stdlib.h>, <sys/types.h>, ...); a bare, no-dot
// standard header name appearing after "#include <" is therefore a strong,
// low-noise C++ signal.
var cppOnlyStdlibHeaders = []string{
	"vector", "string", "memory", "map", "unordered_map", "unordered_set",
	"set", "algorithm", "iostream", "ostream", "istream", "sstream",
	"fstream", "iomanip", "utility", "tuple", "array", "list", "deque",
	"stack", "queue", "regex", "thread", "mutex", "atomic", "chrono",
	"type_traits", "optional", "functional", "initializer_list", "variant",
	"any", "filesystem", "numeric", "random", "bitset", "complex",
	"valarray", "exception", "stdexcept", "typeinfo", "condition_variable",
	"future", "string_view", "new",
}

// DetectLanguageWithContent resolves a LangEntry the same way DetectLanguage
// does, but additionally sniffs file content to disambiguate a plain ".h"
// extension between the C and C++ grammars.
//
// Background: gotreesitter's extension table maps both the C and the C++
// LangEntry to bare ".h" (mirroring the real world — most ".h" files are C,
// but plenty of C++ libraries ship C++-only headers under a ".h" name, e.g.
// ImageMagick's Magick++/*.h, OpenEXR's Imath/*.h). DetectLanguage resolves
// that tie in favor of C (first grammar registered for the extension; see
// buildExtIndex), which mis-parses genuine C++ headers with the C grammar.
//
// When content is available — which it always is at parse time, see
// ParseFile / ParseFilePooled — this function re-checks: if extension-based
// detection landed on the C grammar AND the extension is exactly ".h" (the
// one genuinely ambiguous case), it scans content for syntax that is valid
// C++ and never valid C (see looksLikeCppHeader) and, if found, returns the
// C++ grammar instead.
//
// Every other extension (.c, .cc, .cpp, .hpp, .hh, .hxx, and every
// non-C/C++ language) is untouched — this only refines the single ambiguous
// case. DetectLanguage(filename) itself is unchanged and remains the
// content-free entry point for callers without a content sniff.
func DetectLanguageWithContent(filename string, content []byte) *LangEntry {
	entry := DetectLanguage(filename)
	if entry == nil || entry.Name != "c" {
		return entry
	}
	if !strings.EqualFold(path.Ext(filename), ".h") {
		return entry
	}
	if !looksLikeCppHeader(content) {
		return entry
	}
	if cpp := DetectLanguageByName("cpp"); cpp != nil {
		return cpp
	}
	return entry
}

// looksLikeCppHeader scans raw ".h" file content for syntax markers that are
// valid in C++ and never valid in C. It is deliberately conservative:
// misclassifying a genuine C header as C++ is worse than leaving an obscure
// C++ header classified as C (a plain ".h" with none of these markers stays
// C), so every marker below was checked against a real contamination corpus
// (ImageMagick's Magick++/*.h and magick/*.h + wand/*.h, OpenEXR's Imath,
// ghostty's SIMD index_of.h, glibc's libintl.h, and the EGL/GLES/GLX system
// headers) before being included.
//
// Tradeoff: this is a byte/substring scan over the first cppHeaderScanLimit
// bytes, not a real tokenizer. Comments (// and /* */) and string/char
// literals are blanked out first (see stripComments / stripQuotedLiterals) so a prose
// mention like "struct sockaddr::__ss_family" in a comment doesn't trigger a
// false positive — that specific case was found in real corpus data
// (libxml2's posix config.h) during verification. What remains unhandled:
// raw string literals, digraphs/trigraphs, and multi-byte escape edge cases
// — none of which showed up in the verification corpus, but a pathological
// file could still fool the scan. That residual risk is accepted per the
// same conservative bias: the scan only ever flips a bare ".h" that
// DetectLanguage already resolved to C, never any other extension.
//
// Special case: any reference to "__cplusplus" is treated as decisive
// evidence the header's baseline language is C — that macro exists
// specifically so a header can compile as C, and add extra content that only
// a C++ compiler sees (the "#ifdef __cplusplus / extern \"C\" { / #endif"
// wrapper, or, per glibc's libintl.h, whole "namespace std { using ...; }"
// shims). Every C header in the verification corpus that references
// "__cplusplus" needed to stay C, including libintl.h despite its genuine
// (but conditional) namespace/using content; every genuine C++ header in the
// corpus (Magick++, Imath, index_of.h) references "__cplusplus" nowhere at
// all. The one exception is "template<" / "template <": C has no templates
// under any preprocessor condition, so a template marker overrides the
// __cplusplus short-circuit.
func looksLikeCppHeader(content []byte) bool {
	if len(content) > cppHeaderScanLimit {
		content = content[:cppHeaderScanLimit]
	}
	// commentsOnly keeps quote characters intact (needed for the `extern
	// "C"` check below, which IS quoted syntax); stripped additionally
	// blanks quoted literals for every other marker, none of which should
	// legitimately appear inside a string/char literal.
	commentsOnly := stripComments(content)
	stripped := stripQuotedLiterals(commentsOnly)

	if hasTemplateDecl(stripped) {
		return true
	}
	if bytes.Contains(stripped, []byte("__cplusplus")) {
		return false
	}
	if hasClassWithBody(stripped) {
		return true
	}
	if hasNamespaceWithBody(stripped) {
		return true
	}
	if bytes.Contains(stripped, []byte("std::")) {
		return true
	}
	if bytes.Contains(stripped, []byte("::")) {
		return true
	}
	if bytes.Contains(stripped, []byte("public:")) ||
		bytes.Contains(stripped, []byte("private:")) ||
		bytes.Contains(stripped, []byte("protected:")) {
		return true
	}
	if bytes.Contains(stripped, []byte("using namespace")) {
		return true
	}
	for _, hdr := range cppOnlyStdlibHeaders {
		if bytes.Contains(stripped, []byte("#include <"+hdr+">")) {
			return true
		}
	}
	// Reached only when content has no "__cplusplus" anywhere, so this is
	// an unconditional extern "C" — invalid in plain C, so the file can
	// only be C++ exposing a C-linkage API. Checked against commentsOnly
	// (not stripped) because the marker is itself the quoted string `"C"`.
	if bytes.Contains(commentsOnly, []byte(`extern "C"`)) {
		return true
	}
	return false
}

// stripComments returns a copy of content with the interior of "//" line
// comments and "/* */" block comments replaced with spaces (newlines
// preserved, so line-oriented reasoning elsewhere stays intact). Quote
// characters are left untouched — callers that also need quoted literals
// blanked should further pass the result through stripQuotedLiterals; the
// `extern "C"` check deliberately does not, since that marker's identity IS
// the quoted string `"C"`.
func stripComments(content []byte) []byte {
	out := make([]byte, len(content))
	copy(out, content)
	n := len(out)

	blank := func(start, end int) {
		for i := start; i < end; i++ {
			if out[i] != '\n' {
				out[i] = ' '
			}
		}
	}

	for i := 0; i < n; {
		switch {
		case i+1 < n && out[i] == '/' && out[i+1] == '/':
			start := i
			i += 2
			for i < n && out[i] != '\n' {
				i++
			}
			blank(start, i)
		case i+1 < n && out[i] == '/' && out[i+1] == '*':
			start := i
			i += 2
			for i+1 < n && !(out[i] == '*' && out[i+1] == '/') {
				i++
			}
			if i+1 < n {
				i += 2
			} else {
				i = n
			}
			blank(start, i)
		default:
			i++
		}
	}
	return out
}

// stripQuotedLiterals returns a copy of content with the interior of
// "..."/'...' string and char literals (including their delimiting quotes)
// replaced with spaces. Intended to run after stripComments so a quote
// character inside a comment was already blanked and isn't mistaken for the
// start of a literal.
func stripQuotedLiterals(content []byte) []byte {
	out := make([]byte, len(content))
	copy(out, content)
	n := len(out)

	blank := func(start, end int) {
		for i := start; i < end; i++ {
			if out[i] != '\n' {
				out[i] = ' '
			}
		}
	}

	for i := 0; i < n; {
		if out[i] == '"' || out[i] == '\'' {
			quote := out[i]
			start := i
			i++
			for i < n && out[i] != quote {
				if out[i] == '\\' && i+1 < n {
					i += 2
					continue
				}
				i++
			}
			if i < n {
				i++ // consume closing quote
			}
			blank(start, i)
		} else {
			i++
		}
	}
	return out
}

// isIdentStart reports whether b can start a C/C++ identifier.
func isIdentStart(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// isIdentByte reports whether b can appear inside a C/C++ identifier.
func isIdentByte(b byte) bool {
	return isIdentStart(b) || (b >= '0' && b <= '9')
}

// skipWhitespace advances i past spaces/tabs/CR/LF, up to maxSkip bytes.
func skipWhitespace(content []byte, i, maxSkip int) int {
	n := 0
	for i < len(content) && n < maxSkip {
		switch content[i] {
		case ' ', '\t', '\r', '\n':
			i++
			n++
			continue
		}
		return i
	}
	return i
}

// hasBraceBeforeSemicolon scans content[from : from+window] and reports
// whether a '{' is found before any ';'. Used to tell a declaration with a
// body ("class Foo {") from a forward declaration or unrelated statement
// ("class Foo;", or a C identifier literally named "class" used in an
// expression that ends in ';').
func hasBraceBeforeSemicolon(content []byte, from, window int) bool {
	end := from + window
	if end > len(content) {
		end = len(content)
	}
	for i := from; i < end; i++ {
		switch content[i] {
		case '{':
			return true
		case ';':
			return false
		}
	}
	return false
}

// forEachWord calls fn(start, end) for every word-boundary-matched
// occurrence of word in content, stopping early if fn returns false.
func forEachWord(content []byte, word string, fn func(start, end int) bool) {
	w := []byte(word)
	off := 0
	for {
		idx := bytes.Index(content[off:], w)
		if idx < 0 {
			return
		}
		start := off + idx
		end := start + len(w)
		var before, after byte
		if start > 0 {
			before = content[start-1]
		}
		if end < len(content) {
			after = content[end]
		}
		if !isIdentByte(before) && !isIdentByte(after) {
			if !fn(start, end) {
				return
			}
		}
		off = end
	}
}

// hasTemplateDecl reports whether content contains a C++ template
// declaration ("template<" or "template <...<"). C has no templates, under
// any preprocessor condition, so this is an unconditional C++ signal.
func hasTemplateDecl(content []byte) bool {
	found := false
	forEachWord(content, "template", func(_, end int) bool {
		i := skipWhitespace(content, end, 10)
		if i < len(content) && content[i] == '<' {
			found = true
			return false
		}
		return true
	})
	return found
}

// hasClassWithBody reports whether content contains a "class <ident> {"
// declaration (allowing an inheritance list in between, e.g.
// "class Foo : public Bar {"). Requires an identifier immediately after
// "class" so a C identifier literally named "class" (e.g. "int class = 5;")
// is never mistaken for the C++ keyword.
func hasClassWithBody(content []byte) bool {
	found := false
	forEachWord(content, "class", func(_, end int) bool {
		i := skipWhitespace(content, end, 30)
		if i >= len(content) || !isIdentStart(content[i]) {
			return true // not "class <ident>"; keep looking
		}
		if hasBraceBeforeSemicolon(content, i, 300) {
			found = true
			return false
		}
		return true
	})
	return found
}

// hasNamespaceWithBody reports whether content contains a namespace
// declaration with a body: "namespace Foo {", a nested "namespace A::B {"
// (C++17), or an anonymous "namespace {".
func hasNamespaceWithBody(content []byte) bool {
	found := false
	forEachWord(content, "namespace", func(_, end int) bool {
		i := skipWhitespace(content, end, 30)
		if i >= len(content) {
			return true
		}
		if content[i] == '{' {
			found = true
			return false
		}
		if isIdentStart(content[i]) || content[i] == ':' {
			if hasBraceBeforeSemicolon(content, i, 200) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}
