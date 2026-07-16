package grammars

import (
	"testing"

	"github.com/davidwalter0/gotreesitter"
)

// TestCBlobMacroCallingConventionTypedefNamedReturn guards the GL/EGL/GLES
// calling-convention-macro idiom against the c-1 regression: a bare (non-
// primitive) return type immediately followed by a parenthesized
// "(CALLING_CONV_MACRO name)" pair must resolve to a macro_type_specifier
// type field, matching the pinned tree-sitter-c oracle
// (github.com/tree-sitter/tree-sitter-c @ ae19b676) exactly, including the
// trailing "(unsigned int target)" parameter list's re-parse as a generic
// declarator (byte-for-byte verified against `tree-sitter parse --cst`).
func TestCBlobMacroCallingConventionTypedefNamedReturn(t *testing.T) {
	src := []byte("typedef GLenum (GL_APIENTRYP PFNGLCHECKFRAMEBUFFERSTATUSPROC) (unsigned int target);\n")
	root, lang := cBlobMustParse(t, src)

	typeDef := root.NamedChild(0)
	if typeDef == nil || typeDef.Type(lang) != "type_definition" {
		t.Fatalf("expected type_definition, got %s", root.SExpr(lang))
	}

	macroType := typeDef.ChildByFieldName("type", lang)
	assertMacroTypeSpecifier(t, src, lang, macroType, "GLenum", "GL_APIENTRYP", "PFNGLCHECKFRAMEBUFFERSTATUSPROC")

	declarator := typeDef.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "parenthesized_declarator" {
		t.Fatalf("expected declarator field to be parenthesized_declarator, got %s", root.SExpr(lang))
	}
	if got, want := declarator.Text(src), "(unsigned int target)"; got != want {
		t.Fatalf("declarator text = %q, want %q", got, want)
	}
	if got, want := declarator.ChildCount(), 4; got != want {
		t.Fatalf("declarator child count = %d, want %d (full tree: %s)", got, want, root.SExpr(lang))
	}
	primType := declarator.Child(1)
	if primType == nil || primType.Type(lang) != "primitive_type" || primType.Text(src) != "unsigned" {
		t.Fatalf("declarator.Child(1) = %v, want primitive_type `unsigned`", primType)
	}
	errNode := declarator.Child(2)
	if errNode == nil || errNode.Type(lang) != "ERROR" || errNode.ChildCount() != 2 {
		t.Fatalf("declarator.Child(2) = %v, want ERROR with 2 children", errNode)
	}
	if got, want := errNode.Child(0).Text(src), "int"; got != want {
		t.Fatalf("ERROR.Child(0) text = %q, want %q", got, want)
	}
	if got, want := errNode.Child(1).Text(src), "target"; got != want {
		t.Fatalf("ERROR.Child(1) text = %q, want %q", got, want)
	}
}

// TestCBlobMacroCallingConventionPlainDeclarationNamedReturn covers the
// same idiom in non-typedef declaration form (q3 in the round5-c bug
// report), which additionally exercises the identifier/type_identifier
// retagging that differs from the typedef form's already-matching shape.
func TestCBlobMacroCallingConventionPlainDeclarationNamedReturn(t *testing.T) {
	src := []byte("GLenum (GL_APIENTRYP glCheckFramebufferStatus) (unsigned int target);\n")
	root, lang := cBlobMustParse(t, src)

	decl := root.NamedChild(0)
	if decl == nil || decl.Type(lang) != "declaration" {
		t.Fatalf("expected declaration, got %s", root.SExpr(lang))
	}
	macroType := decl.ChildByFieldName("type", lang)
	assertMacroTypeSpecifier(t, src, lang, macroType, "GLenum", "GL_APIENTRYP", "glCheckFramebufferStatus")

	declarator := decl.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "parenthesized_declarator" {
		t.Fatalf("expected declarator field to be parenthesized_declarator, got %s", root.SExpr(lang))
	}
	if got, want := declarator.Text(src), "(unsigned int target)"; got != want {
		t.Fatalf("declarator text = %q, want %q", got, want)
	}
}

// TestCBlobMacroCallingConventionSingleIdentifierParam covers the dominant
// real-world GL/EGL header shape: a single Khronos-typedef'd scalar
// parameter (e.g. `GLenum target`), which the vast majority of non-void
// PFNGL*PROC typedefs use. `GLenum`(6) and `target`(6) are byte-length-tied,
// which the oracle resolves toward wrapping the *type* side in ERROR (see
// TestCBlobMacroCallingConventionShortParamNameWrapsInsteadOfType for the
// opposite case).
func TestCBlobMacroCallingConventionSingleIdentifierParam(t *testing.T) {
	src := []byte("typedef GLenum (GL_APIENTRYP PFNGLCHECKFRAMEBUFFERSTATUSPROC) (GLenum target);\n")
	root, lang := cBlobMustParse(t, src)

	typeDef := root.NamedChild(0)
	macroType := typeDef.ChildByFieldName("type", lang)
	assertMacroTypeSpecifier(t, src, lang, macroType, "GLenum", "GL_APIENTRYP", "PFNGLCHECKFRAMEBUFFERSTATUSPROC")

	declarator := typeDef.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "parenthesized_declarator" {
		t.Fatalf("expected parenthesized_declarator, got %s", root.SExpr(lang))
	}
	if got, want := declarator.ChildCount(), 4; got != want {
		t.Fatalf("declarator child count = %d, want %d (full tree: %s)", got, want, root.SExpr(lang))
	}
	errNode := declarator.Child(1)
	if errNode == nil || errNode.Type(lang) != "ERROR" || errNode.ChildCount() != 1 {
		t.Fatalf("declarator.Child(1) = %v, want ERROR with 1 child", errNode)
	}
	if got, want := errNode.Child(0).Type(lang), "type_identifier"; got != want || errNode.Child(0).Text(src) != "GLenum" {
		t.Fatalf("ERROR.Child(0) = %s %q, want type_identifier \"GLenum\"", got, errNode.Child(0).Text(src))
	}
	name := declarator.Child(2)
	if name == nil || name.Type(lang) != "type_identifier" || name.Text(src) != "target" {
		t.Fatalf("declarator.Child(2) = %v, want type_identifier \"target\"", name)
	}
}

// TestCBlobMacroCallingConventionShortParamNameWrapsInsteadOfType guards a
// regression found via the real corpus (usr/include/GLES2/gl2.h,
// PFNGLCREATESHADERPROC): when the declarator name is *shorter* than the
// parameter's type identifier (here `type`(4) < `GLenum`(6)), the oracle's
// error-cost minimization wraps the *shorter* side — the declarator name,
// not the type — reversing the wrapping seen in
// TestCBlobMacroCallingConventionSingleIdentifierParam's tied-length case.
// Byte-for-byte verified against the pinned oracle
// (github.com/tree-sitter/tree-sitter-c @ ae19b676) via `tree-sitter parse
// --cst`.
func TestCBlobMacroCallingConventionShortParamNameWrapsInsteadOfType(t *testing.T) {
	src := []byte("typedef GLuint (GL_APIENTRYP PFNGLCREATESHADERPROC) (GLenum type);\n")
	root, lang := cBlobMustParse(t, src)

	typeDef := root.NamedChild(0)
	macroType := typeDef.ChildByFieldName("type", lang)
	assertMacroTypeSpecifier(t, src, lang, macroType, "GLuint", "GL_APIENTRYP", "PFNGLCREATESHADERPROC")

	declarator := typeDef.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "parenthesized_declarator" {
		t.Fatalf("expected parenthesized_declarator, got %s", root.SExpr(lang))
	}
	if got, want := declarator.ChildCount(), 4; got != want {
		t.Fatalf("declarator child count = %d, want %d (full tree: %s)", got, want, root.SExpr(lang))
	}
	bareType := declarator.Child(1)
	if bareType == nil || bareType.Type(lang) != "type_identifier" || bareType.Text(src) != "GLenum" {
		t.Fatalf("declarator.Child(1) = %v, want bare type_identifier \"GLenum\" (not wrapped in ERROR)", bareType)
	}
	errNode := declarator.Child(2)
	if errNode == nil || errNode.Type(lang) != "ERROR" || errNode.ChildCount() != 1 {
		t.Fatalf("declarator.Child(2) = %v, want ERROR with 1 child", errNode)
	}
	if got := errNode.Child(0); got == nil || got.Type(lang) != "identifier" || got.Text(src) != "type" {
		t.Fatalf("ERROR.Child(0) = %v, want identifier \"type\"", got)
	}
}

// TestCBlobMacroCallingConventionSizedTypeLongParamNameFlipsWrapping guards
// the two-word (`unsigned int`) analog of the same length-driven boundary:
// when the declarator name is longer than the modifier word ("unsigned",
// 8 bytes), the oracle wraps (word1+word2) in ERROR and leaves the name
// bare instead of the reverse (see
// TestCBlobMacroCallingConventionTypedefNamedReturn for the short-name
// case where `target`(6) <= `unsigned`(8) keeps the original wrapping).
// Byte-for-byte verified against the pinned oracle.
func TestCBlobMacroCallingConventionSizedTypeLongParamNameFlipsWrapping(t *testing.T) {
	src := []byte("typedef GLenum (GL_APIENTRYP PFNGLTESTPROC) (unsigned int targetlongname);\n")
	root, lang := cBlobMustParse(t, src)

	typeDef := root.NamedChild(0)
	macroType := typeDef.ChildByFieldName("type", lang)
	assertMacroTypeSpecifier(t, src, lang, macroType, "GLenum", "GL_APIENTRYP", "PFNGLTESTPROC")

	declarator := typeDef.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "parenthesized_declarator" {
		t.Fatalf("expected parenthesized_declarator, got %s", root.SExpr(lang))
	}
	if got, want := declarator.ChildCount(), 4; got != want {
		t.Fatalf("declarator child count = %d, want %d (full tree: %s)", got, want, root.SExpr(lang))
	}
	errNode := declarator.Child(1)
	if errNode == nil || errNode.Type(lang) != "ERROR" || errNode.ChildCount() != 2 {
		t.Fatalf("declarator.Child(1) = %v, want ERROR with 2 children", errNode)
	}
	if got := errNode.Child(0); got == nil || got.Type(lang) != "primitive_type" || got.Text(src) != "unsigned" {
		t.Fatalf("ERROR.Child(0) = %v, want primitive_type \"unsigned\"", got)
	}
	if got := errNode.Child(1); got == nil || got.Type(lang) != "identifier" || got.Text(src) != "int" {
		t.Fatalf("ERROR.Child(1) = %v, want identifier \"int\"", got)
	}
	name := declarator.Child(2)
	if name == nil || name.Type(lang) != "type_identifier" || name.Text(src) != "targetlongname" {
		t.Fatalf("declarator.Child(2) = %v, want bare type_identifier \"targetlongname\"", name)
	}
}

// TestCBlobMacroCallingConventionZeroArgVoid covers the zero-parameter
// "(void)" form (e.g. PFNGLCREATEPROGRAMPROC, PFNGLGETERRORPROC).
func TestCBlobMacroCallingConventionZeroArgVoid(t *testing.T) {
	src := []byte("typedef GLuint (GL_APIENTRYP PFNGLCREATEPROGRAMPROC) (void);\n")
	root, lang := cBlobMustParse(t, src)

	typeDef := root.NamedChild(0)
	macroType := typeDef.ChildByFieldName("type", lang)
	assertMacroTypeSpecifier(t, src, lang, macroType, "GLuint", "GL_APIENTRYP", "PFNGLCREATEPROGRAMPROC")

	declarator := typeDef.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "parenthesized_declarator" {
		t.Fatalf("expected parenthesized_declarator, got %s", root.SExpr(lang))
	}
	if got, want := declarator.Text(src), "(void)"; got != want {
		t.Fatalf("declarator text = %q, want %q", got, want)
	}
	if got, want := declarator.ChildCount(), 3; got != want {
		t.Fatalf("declarator child count = %d, want %d", got, want)
	}
	sole := declarator.Child(1)
	if sole == nil || sole.Type(lang) != "primitive_type" || sole.Text(src) != "void" {
		t.Fatalf("declarator.Child(1) = %v, want primitive_type \"void\"", sole)
	}
}

// TestCBlobMacroCallingConventionVoidReturnUnaffected asserts the pass never
// fires when the return type is the `void` keyword (primitive_type, not an
// identifier) -- the oracle and gt already agree there, so this signature
// must not match and the original function_declarator+parameter_list shape
// must survive untouched.
func TestCBlobMacroCallingConventionVoidReturnUnaffected(t *testing.T) {
	src := []byte("typedef void (GL_APIENTRYP PFNGLCLEARPROC) (unsigned int mask);\n")
	root, lang := cBlobMustParse(t, src)

	typeDef := root.NamedChild(0)
	if typeDef == nil || typeDef.Type(lang) != "type_definition" {
		t.Fatalf("expected type_definition, got %s", root.SExpr(lang))
	}
	typeField := typeDef.ChildByFieldName("type", lang)
	if typeField == nil || typeField.Type(lang) != "primitive_type" {
		t.Fatalf("expected type field to stay primitive_type, got %s", root.SExpr(lang))
	}
	declarator := typeDef.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "function_declarator" {
		t.Fatalf("expected declarator field to stay function_declarator, got %s", root.SExpr(lang))
	}
}

// TestCBlobMacroCallingConventionTwoParamPointerLastConstQualified covers
// the dominant real-world GL/EGL two-parameter shape (a receiver plus a
// `const`-qualified pointer output/input, e.g. gl2.h's
// PFNGLGETATTRIBLOCATIONPROC / PFNGLGETUNIFORMLOCATIONPROC): the final
// parameter's pointer declarator is unconditionally kept bare, and
// everything else (first parameter, `,`, `const`, the pointer's base type)
// is merged into one ERROR node — byte-for-byte verified against the
// pinned oracle (github.com/tree-sitter/tree-sitter-c @ ae19b676). This
// exact input previously asserted the conservative "left untouched"
// fallback (pre-multi-param-extension); the fallback assertion for
// genuinely out-of-scope shapes now lives in
// TestCBlobMacroCallingConventionLeadingPointerParamLeftUntouched and
// TestCBlobMacroCallingConventionFourParamLeftUntouched below.
func TestCBlobMacroCallingConventionTwoParamPointerLastConstQualified(t *testing.T) {
	src := []byte("typedef GLint (GL_APIENTRYP PFNGLGETATTRIBLOCATIONPROC) (GLuint program, const GLchar *name);\n")
	root, lang := cBlobMustParse(t, src)

	typeDef := root.NamedChild(0)
	macroType := typeDef.ChildByFieldName("type", lang)
	assertMacroTypeSpecifier(t, src, lang, macroType, "GLint", "GL_APIENTRYP", "PFNGLGETATTRIBLOCATIONPROC")

	declarator := typeDef.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "parenthesized_declarator" {
		t.Fatalf("expected parenthesized_declarator, got %s", root.SExpr(lang))
	}
	if got, want := declarator.ChildCount(), 4; got != want {
		t.Fatalf("declarator child count = %d, want %d (full tree: %s)", got, want, root.SExpr(lang))
	}
	// ChildCount() (unlike SExpr, which hides anonymous nodes) counts every
	// child including the `,` separator and the bare, unnamed `const`
	// token that was unwrapped from gt's own type_qualifier node — both
	// verified present (in this exact position) in the pinned oracle via
	// the corpus_parity dump.v1 JSON, not just the `tree-sitter parse`
	// CLI's default (named-only) rendering.
	errNode := declarator.Child(1)
	if errNode == nil || errNode.Type(lang) != "ERROR" || errNode.ChildCount() != 5 {
		t.Fatalf("declarator.Child(1) = %v, want ERROR with 5 children", errNode)
	}
	if got := errNode.Child(0); got == nil || got.Type(lang) != "type_identifier" || got.Text(src) != "GLuint" {
		t.Fatalf("ERROR.Child(0) = %v, want type_identifier \"GLuint\"", got)
	}
	if got := errNode.Child(1); got == nil || got.Type(lang) != "identifier" || got.Text(src) != "program" {
		t.Fatalf("ERROR.Child(1) = %v, want identifier \"program\"", got)
	}
	if got := errNode.Child(2); got == nil || got.IsNamed() || got.Type(lang) != "," {
		t.Fatalf("ERROR.Child(2) = %v, want bare unnamed \",\" separator", got)
	}
	if got := errNode.Child(3); got == nil || got.IsNamed() || got.Type(lang) != "const" {
		t.Fatalf("ERROR.Child(3) = %v, want bare unnamed \"const\" token (unwrapped from type_qualifier)", got)
	}
	if got := errNode.Child(4); got == nil || got.Type(lang) != "identifier" || got.Text(src) != "GLchar" {
		t.Fatalf("ERROR.Child(4) = %v, want identifier \"GLchar\" (demoted)", got)
	}
	ptr := declarator.Child(2)
	if ptr == nil || ptr.Type(lang) != "pointer_declarator" || ptr.ChildCount() != 2 {
		t.Fatalf("declarator.Child(2) = %v, want pointer_declarator with 2 children", ptr)
	}
	if got := ptr.Child(1); got == nil || got.Type(lang) != "type_identifier" || got.Text(src) != "name" || ptr.FieldNameForChild(1, lang) != "declarator" {
		t.Fatalf("pointer_declarator's inner declarator = %v (field %q), want type_identifier \"name\" field \"declarator\"", got, ptr.FieldNameForChild(1, lang))
	}
}

// TestCBlobMacroCallingConventionTwoParamPlainFrontSplit covers the
// two-plain-parameter shape where the first parameter's type is at least
// as long as the last parameter's type (byte length) — the oracle
// front-splits: only the first type stays bare, the rest (including the
// last parameter, entirely) is merged into one ERROR. Matches the real
// egl.h entry PFNEGLDESTROYCONTEXTPROC (`EGLDisplay`=10 vs `EGLContext`=10,
// a tie, which also resolves to front-split).
func TestCBlobMacroCallingConventionTwoParamPlainFrontSplit(t *testing.T) {
	src := []byte("typedef EGLBoolean (EGLAPIENTRYP PFNEGLDESTROYCONTEXTPROC) (EGLDisplay dpy, EGLContext ctx);\n")
	root, lang := cBlobMustParse(t, src)

	typeDef := root.NamedChild(0)
	macroType := typeDef.ChildByFieldName("type", lang)
	assertMacroTypeSpecifier(t, src, lang, macroType, "EGLBoolean", "EGLAPIENTRYP", "PFNEGLDESTROYCONTEXTPROC")

	declarator := typeDef.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "parenthesized_declarator" || declarator.ChildCount() != 4 {
		t.Fatalf("expected 4-child parenthesized_declarator, got %s", root.SExpr(lang))
	}
	bare := declarator.Child(1)
	if bare == nil || bare.Type(lang) != "type_identifier" || bare.Text(src) != "EGLDisplay" {
		t.Fatalf("declarator.Child(1) = %v, want bare type_identifier \"EGLDisplay\"", bare)
	}
	errNode := declarator.Child(2)
	if errNode == nil || errNode.Type(lang) != "ERROR" || errNode.ChildCount() != 4 {
		t.Fatalf("declarator.Child(2) = %v, want ERROR with 4 children", errNode)
	}
	wantErr := []struct {
		typ, text string
		named     bool
	}{
		{"identifier", "dpy", true}, {",", ",", false},
		{"identifier", "EGLContext", true}, {"identifier", "ctx", true},
	}
	for i, w := range wantErr {
		got := errNode.Child(i)
		if got == nil || got.Type(lang) != w.typ || got.Text(src) != w.text || got.IsNamed() != w.named {
			t.Fatalf("ERROR.Child(%d) = %v, want %s %q (named=%v)", i, got, w.typ, w.text, w.named)
		}
	}
}

// TestCBlobMacroCallingConventionTwoParamPlainBackSplit covers the mirror
// case: when the *last* parameter's type is strictly longer than the
// first's, the oracle back-splits instead — everything up to and
// including the last type stays in one ERROR (with the last type
// exceptionally retaining type_identifier, since it is adjacent to the
// bare boundary), and only the last parameter's declarator name (promoted
// to type_identifier) is bare. Synthetic (`GLbyte`=6 < `GLenumLongType`=13)
// since no real corpus two-parameter entry happens to hit this side of the
// split; confirmed byte-exact against the pinned oracle via a one-off
// corpus_parity Docker run (same code path as the real-corpus sweep, not
// the `tree-sitter parse` CLI, whose default rendering hides `,` tokens
// and misled an earlier draft of this fix into dropping them).
func TestCBlobMacroCallingConventionTwoParamPlainBackSplit(t *testing.T) {
	src := []byte("typedef GLenum (GL_APIENTRYP PFNGLTESTPROC) (GLbyte a, GLenumLongType b);\n")
	root, lang := cBlobMustParse(t, src)

	typeDef := root.NamedChild(0)
	macroType := typeDef.ChildByFieldName("type", lang)
	assertMacroTypeSpecifier(t, src, lang, macroType, "GLenum", "GL_APIENTRYP", "PFNGLTESTPROC")

	declarator := typeDef.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "parenthesized_declarator" || declarator.ChildCount() != 4 {
		t.Fatalf("expected 4-child parenthesized_declarator, got %s", root.SExpr(lang))
	}
	errNode := declarator.Child(1)
	if errNode == nil || errNode.Type(lang) != "ERROR" || errNode.ChildCount() != 4 {
		t.Fatalf("declarator.Child(1) = %v, want ERROR with 4 children", errNode)
	}
	wantErr := []struct {
		typ, text string
		named     bool
	}{
		{"type_identifier", "GLbyte", true}, {"identifier", "a", true},
		{",", ",", false}, {"type_identifier", "GLenumLongType", true},
	}
	for i, w := range wantErr {
		got := errNode.Child(i)
		if got == nil || got.Type(lang) != w.typ || got.Text(src) != w.text || got.IsNamed() != w.named {
			t.Fatalf("ERROR.Child(%d) = %v, want %s %q (named=%v)", i, got, w.typ, w.text, w.named)
		}
	}
	bare := declarator.Child(2)
	if bare == nil || bare.Type(lang) != "type_identifier" || bare.Text(src) != "b" {
		t.Fatalf("declarator.Child(2) = %v, want bare type_identifier \"b\" (promoted)", bare)
	}
}

// TestCBlobMacroCallingConventionThreeParamPointerLastConstQualified
// covers the real egl.h entry PFNEGLCREATEPBUFFERSURFACEPROC: three
// parameters, the last `const`-qualified and pointer-shaped. Unlike the
// plain-last three-parameter cases below, a pointer-shaped last parameter
// always back-splits regardless of byte lengths.
func TestCBlobMacroCallingConventionThreeParamPointerLastConstQualified(t *testing.T) {
	src := []byte("typedef EGLSurface (EGLAPIENTRYP PFNEGLCREATEPBUFFERSURFACEPROC) (EGLDisplay dpy, EGLConfig config, const EGLint *attrib_list);\n")
	root, lang := cBlobMustParse(t, src)

	typeDef := root.NamedChild(0)
	macroType := typeDef.ChildByFieldName("type", lang)
	assertMacroTypeSpecifier(t, src, lang, macroType, "EGLSurface", "EGLAPIENTRYP", "PFNEGLCREATEPBUFFERSURFACEPROC")

	declarator := typeDef.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "parenthesized_declarator" || declarator.ChildCount() != 4 {
		t.Fatalf("expected 4-child parenthesized_declarator, got %s", root.SExpr(lang))
	}
	// ChildCount() counts the `,` separators and the bare, unnamed `const`
	// token too (see the two-parameter const-qualified test above for
	// why): one comma after each leading parameter, then `const`, then
	// the demoted last-parameter type.
	errNode := declarator.Child(1)
	if errNode == nil || errNode.Type(lang) != "ERROR" || errNode.ChildCount() != 8 {
		t.Fatalf("declarator.Child(1) = %v, want ERROR with 8 children", errNode)
	}
	wantErr := []struct {
		idx       int
		typ, text string
		named     bool
	}{
		{0, "type_identifier", "EGLDisplay", true}, {1, "identifier", "dpy", true},
		{2, ",", ",", false},
		{3, "identifier", "EGLConfig", true}, {4, "identifier", "config", true},
		{5, ",", ",", false},
		{6, "const", "const", false},
		{7, "identifier", "EGLint", true},
	}
	for _, w := range wantErr {
		got := errNode.Child(w.idx)
		if got == nil || got.Type(lang) != w.typ || got.Text(src) != w.text || got.IsNamed() != w.named {
			t.Fatalf("ERROR.Child(%d) = %v, want %s %q (named=%v)", w.idx, got, w.typ, w.text, w.named)
		}
	}
	ptr := declarator.Child(2)
	if ptr == nil || ptr.Type(lang) != "pointer_declarator" || ptr.ChildCount() != 2 {
		t.Fatalf("declarator.Child(2) = %v, want pointer_declarator with 2 children", ptr)
	}
	if got := ptr.Child(1); got == nil || got.Type(lang) != "type_identifier" || got.Text(src) != "attrib_list" {
		t.Fatalf("pointer_declarator's inner declarator = %v, want type_identifier \"attrib_list\"", got)
	}
}

// TestCBlobMacroCallingConventionThreeParamPointerLastUnqualified covers
// the unqualified-pointer variant (no `const`) at three parameters, same
// unconditional back-split rule, no qualifier token to fold into the
// ERROR node. Synthetic (no real corpus three-parameter, unqualified-
// pointer-last entry), confirmed byte-exact against the pinned oracle via
// a one-off corpus_parity Docker run (see the two-parameter back-split
// test above for why the CLI alone isn't trusted here).
func TestCBlobMacroCallingConventionThreeParamPointerLastUnqualified(t *testing.T) {
	src := []byte("typedef GLenum (GL_APIENTRYP PFNGLTESTPROC) (GLuint program, GLuint index, GLint *value);\n")
	root, lang := cBlobMustParse(t, src)

	typeDef := root.NamedChild(0)
	macroType := typeDef.ChildByFieldName("type", lang)
	assertMacroTypeSpecifier(t, src, lang, macroType, "GLenum", "GL_APIENTRYP", "PFNGLTESTPROC")

	declarator := typeDef.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "parenthesized_declarator" || declarator.ChildCount() != 4 {
		t.Fatalf("expected 4-child parenthesized_declarator, got %s", root.SExpr(lang))
	}
	errNode := declarator.Child(1)
	if errNode == nil || errNode.Type(lang) != "ERROR" || errNode.ChildCount() != 7 {
		t.Fatalf("declarator.Child(1) = %v, want ERROR with 7 children", errNode)
	}
	wantErr := []struct {
		typ, text string
		named     bool
	}{
		{"type_identifier", "GLuint", true}, {"identifier", "program", true},
		{",", ",", false},
		{"identifier", "GLuint", true}, {"identifier", "index", true},
		{",", ",", false},
		{"identifier", "GLint", true},
	}
	for i, w := range wantErr {
		got := errNode.Child(i)
		if got == nil || got.Type(lang) != w.typ || got.Text(src) != w.text || got.IsNamed() != w.named {
			t.Fatalf("ERROR.Child(%d) = %v, want %s %q (named=%v)", i, got, w.typ, w.text, w.named)
		}
	}
	ptr := declarator.Child(2)
	if ptr == nil || ptr.Type(lang) != "pointer_declarator" {
		t.Fatalf("declarator.Child(2) = %v, want pointer_declarator", ptr)
	}
	if got := ptr.Child(1); got == nil || got.Type(lang) != "type_identifier" || got.Text(src) != "value" {
		t.Fatalf("pointer_declarator's inner declarator = %v, want type_identifier \"value\"", got)
	}
}

// TestCBlobMacroCallingConventionThreeParamPlainFrontSplit covers the real
// egl.h entry PFNEGLBINDTEXIMAGEPROC: three plain parameters where the
// first type (`EGLDisplay`=10) is longer than the last (`EGLint`=6), so
// the oracle front-splits — only the first type stays bare.
func TestCBlobMacroCallingConventionThreeParamPlainFrontSplit(t *testing.T) {
	src := []byte("typedef EGLBoolean (EGLAPIENTRYP PFNEGLBINDTEXIMAGEPROC) (EGLDisplay dpy, EGLSurface surface, EGLint buffer);\n")
	root, lang := cBlobMustParse(t, src)

	typeDef := root.NamedChild(0)
	macroType := typeDef.ChildByFieldName("type", lang)
	assertMacroTypeSpecifier(t, src, lang, macroType, "EGLBoolean", "EGLAPIENTRYP", "PFNEGLBINDTEXIMAGEPROC")

	declarator := typeDef.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "parenthesized_declarator" || declarator.ChildCount() != 4 {
		t.Fatalf("expected 4-child parenthesized_declarator, got %s", root.SExpr(lang))
	}
	bare := declarator.Child(1)
	if bare == nil || bare.Type(lang) != "type_identifier" || bare.Text(src) != "EGLDisplay" {
		t.Fatalf("declarator.Child(1) = %v, want bare type_identifier \"EGLDisplay\"", bare)
	}
	errNode := declarator.Child(2)
	if errNode == nil || errNode.Type(lang) != "ERROR" || errNode.ChildCount() != 7 {
		t.Fatalf("declarator.Child(2) = %v, want ERROR with 7 children", errNode)
	}
	wantErr := []struct {
		typ, text string
		named     bool
	}{
		{"identifier", "dpy", true},
		{",", ",", false},
		{"identifier", "EGLSurface", true}, {"identifier", "surface", true},
		{",", ",", false},
		{"identifier", "EGLint", true}, {"identifier", "buffer", true},
	}
	for i, w := range wantErr {
		got := errNode.Child(i)
		if got == nil || got.Type(lang) != w.typ || got.Text(src) != w.text || got.IsNamed() != w.named {
			t.Fatalf("ERROR.Child(%d) = %v, want %s %q (named=%v)", i, got, w.typ, w.text, w.named)
		}
	}
}

// TestCBlobMacroCallingConventionThreeParamPlainBackSplit covers the real
// egl.h entry PFNEGLCOPYBUFFERSPROC: structurally identical to
// PFNEGLBINDTEXIMAGEPROC above (three plain parameters) but the last
// type (`EGLNativePixmapType`=20) is longer than the first
// (`EGLDisplay`=10), flipping the split direction to back-split — proof
// the rule is a length comparison, not a fixed structural shape.
func TestCBlobMacroCallingConventionThreeParamPlainBackSplit(t *testing.T) {
	src := []byte("typedef EGLBoolean (EGLAPIENTRYP PFNEGLCOPYBUFFERSPROC) (EGLDisplay dpy, EGLSurface surface, EGLNativePixmapType target);\n")
	root, lang := cBlobMustParse(t, src)

	typeDef := root.NamedChild(0)
	macroType := typeDef.ChildByFieldName("type", lang)
	assertMacroTypeSpecifier(t, src, lang, macroType, "EGLBoolean", "EGLAPIENTRYP", "PFNEGLCOPYBUFFERSPROC")

	declarator := typeDef.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "parenthesized_declarator" || declarator.ChildCount() != 4 {
		t.Fatalf("expected 4-child parenthesized_declarator, got %s", root.SExpr(lang))
	}
	errNode := declarator.Child(1)
	if errNode == nil || errNode.Type(lang) != "ERROR" || errNode.ChildCount() != 7 {
		t.Fatalf("declarator.Child(1) = %v, want ERROR with 7 children", errNode)
	}
	wantErr := []struct {
		typ, text string
		named     bool
	}{
		{"type_identifier", "EGLDisplay", true}, {"identifier", "dpy", true},
		{",", ",", false},
		{"identifier", "EGLSurface", true}, {"identifier", "surface", true},
		{",", ",", false},
		{"type_identifier", "EGLNativePixmapType", true},
	}
	for i, w := range wantErr {
		got := errNode.Child(i)
		if got == nil || got.Type(lang) != w.typ || got.Text(src) != w.text || got.IsNamed() != w.named {
			t.Fatalf("ERROR.Child(%d) = %v, want %s %q (named=%v)", i, got, w.typ, w.text, w.named)
		}
	}
	bare := declarator.Child(2)
	if bare == nil || bare.Type(lang) != "type_identifier" || bare.Text(src) != "target" {
		t.Fatalf("declarator.Child(2) = %v, want bare type_identifier \"target\" (promoted)", bare)
	}
}

// TestCBlobMacroCallingConventionLeadingPointerParamLeftUntouched asserts
// the conservative fallback still holds for a shape outside the verified
// set: a *leading* (non-final) parameter with a pointer declarator (real
// corpus entry PFNGLQUERYMATRIXXOESPROC, GLES/glext.h). Only the final
// parameter is allowed to be pointer-shaped; a pointer anywhere else bails
// to the untouched no-op, since that combination was not validated against
// the pinned oracle.
func TestCBlobMacroCallingConventionLeadingPointerParamLeftUntouched(t *testing.T) {
	src := []byte("typedef GLbitfield (GL_APIENTRYP PFNGLQUERYMATRIXXOESPROC) (GLfixed *mantissa, GLint *exponent);\n")
	root, lang := cBlobMustParse(t, src)

	typeDef := root.NamedChild(0)
	if typeDef == nil || typeDef.Type(lang) != "type_definition" {
		t.Fatalf("expected type_definition, got %s", root.SExpr(lang))
	}
	typeField := typeDef.ChildByFieldName("type", lang)
	if typeField == nil || typeField.Type(lang) != "type_identifier" {
		t.Fatalf("expected type field to stay a bare type_identifier (no-op fallback), got %s", root.SExpr(lang))
	}
	declarator := typeDef.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "function_declarator" {
		t.Fatalf("expected declarator field to stay function_declarator (no-op fallback), got %s", root.SExpr(lang))
	}
}

// TestCBlobMacroCallingConventionFourParamPointerLastBackSplit covers the
// real four-parameter pointer-last shape (eglmesaext.h's
// PFNEGLSWAPBUFFERSREGIONNOK): three plain leading parameters plus a
// `const`-qualified pointer last. The pointer-last recovery back-splits
// *unconditionally* — its split point does not depend on any token length —
// so the exact same reconstruction that is byte-exact at two/three
// parameters stays byte-exact at four, reproducing the pinned oracle's clean
// macro_type_specifier + parenthesized_declarator recovery
// (github.com/tree-sitter/tree-sitter-c @ ae19b676). This flips eglmesaext.h
// to byte-exact parity on the blob path.
func TestCBlobMacroCallingConventionFourParamPointerLastBackSplit(t *testing.T) {
	src := []byte("typedef EGLBoolean (EGLAPIENTRYP PFNEGLSWAPBUFFERSREGIONNOK) (EGLDisplay dpy, EGLSurface surface, EGLint numRects, const EGLint* rects);\n")
	root, lang := cBlobMustParse(t, src)

	typeDef := root.NamedChild(0)
	macroType := typeDef.ChildByFieldName("type", lang)
	assertMacroTypeSpecifier(t, src, lang, macroType, "EGLBoolean", "EGLAPIENTRYP", "PFNEGLSWAPBUFFERSREGIONNOK")

	declarator := typeDef.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "parenthesized_declarator" || declarator.ChildCount() != 4 {
		t.Fatalf("expected 4-child parenthesized_declarator, got %s", root.SExpr(lang))
	}
	// Every leading parameter plus the final parameter's `const` qualifier and
	// its base type collapse into one ERROR (the first parameter's type stays
	// type_identifier, all others demote to identifier); only the pointer
	// declarator survives bare. Child indices count the `,` separators and the
	// unwrapped bare `const` token too.
	errNode := declarator.Child(1)
	if errNode == nil || errNode.Type(lang) != "ERROR" || errNode.ChildCount() != 11 {
		t.Fatalf("declarator.Child(1) = %v, want ERROR with 11 children (full tree: %s)", errNode, root.SExpr(lang))
	}
	wantErr := []struct {
		typ, text string
		named     bool
	}{
		{"type_identifier", "EGLDisplay", true}, {"identifier", "dpy", true},
		{",", ",", false},
		{"identifier", "EGLSurface", true}, {"identifier", "surface", true},
		{",", ",", false},
		{"identifier", "EGLint", true}, {"identifier", "numRects", true},
		{",", ",", false},
		{"const", "const", false},
		{"identifier", "EGLint", true},
	}
	for i, w := range wantErr {
		got := errNode.Child(i)
		if got == nil || got.Type(lang) != w.typ || got.Text(src) != w.text || got.IsNamed() != w.named {
			t.Fatalf("ERROR.Child(%d) = %v, want %s %q (named=%v)", i, got, w.typ, w.text, w.named)
		}
	}
	ptr := declarator.Child(2)
	if ptr == nil || ptr.Type(lang) != "pointer_declarator" || ptr.ChildCount() != 2 {
		t.Fatalf("declarator.Child(2) = %v, want pointer_declarator with 2 children", ptr)
	}
	if got := ptr.Child(1); got == nil || got.Type(lang) != "type_identifier" || got.Text(src) != "rects" {
		t.Fatalf("pointer_declarator's inner declarator = %v, want type_identifier \"rects\"", got)
	}
}

// TestCBlobMacroCallingConventionFourParamPlainLastLeftUntouched asserts the
// four-plus-parameter cap survives for the *plain-last* (non-pointer) shape.
// The plain-last front/back split is decided by a byte-length comparison
// between the first and last parameter types, and that length-driven rule is
// only empirically verified against the pinned oracle at two and three
// parameters. At four-plus plain parameters the same shape family can cascade
// into a structurally different recovery (MISSING tokens, a declaration split
// off the typedef) for some length combinations, with no cheap structural
// signal to tell a clean case apart from a cascading one — so the plain-last
// four-plus case is left untouched entirely rather than emit a guessed tree.
// (This is the real egl.h entry PFNEGLCLIENTWAITSYNCPROC; the pointer-last
// path above is length-independent and therefore safe to extend, but this
// plain-last one is not.)
func TestCBlobMacroCallingConventionFourParamPlainLastLeftUntouched(t *testing.T) {
	src := []byte("typedef EGLint (EGLAPIENTRYP PFNEGLCLIENTWAITSYNCPROC) (EGLDisplay dpy, EGLSync sync, EGLint flags, EGLTime timeout);\n")
	root, lang := cBlobMustParse(t, src)

	typeDef := root.NamedChild(0)
	if typeDef == nil || typeDef.Type(lang) != "type_definition" {
		t.Fatalf("expected type_definition, got %s", root.SExpr(lang))
	}
	typeField := typeDef.ChildByFieldName("type", lang)
	if typeField == nil || typeField.Type(lang) != "type_identifier" {
		t.Fatalf("expected type field to stay a bare type_identifier (no-op fallback), got %s", root.SExpr(lang))
	}
	declarator := typeDef.ChildByFieldName("declarator", lang)
	if declarator == nil || declarator.Type(lang) != "function_declarator" {
		t.Fatalf("expected declarator field to stay function_declarator (no-op fallback), got %s", root.SExpr(lang))
	}
}

func assertMacroTypeSpecifier(t *testing.T, src []byte, lang *gotreesitter.Language, macroType *gotreesitter.Node, wantName, wantMacro, wantInner string) {
	t.Helper()
	if macroType == nil || macroType.Type(lang) != "macro_type_specifier" {
		t.Fatalf("expected type field to be macro_type_specifier, got %v", macroType)
	}
	if got, want := macroType.ChildCount(), 5; got != want {
		t.Fatalf("macro_type_specifier child count = %d, want %d", got, want)
	}
	name := macroType.ChildByFieldName("name", lang)
	if name == nil || name.Type(lang) != "identifier" || name.Text(src) != wantName {
		t.Fatalf("macro_type_specifier name field = %v, want identifier %q", name, wantName)
	}
	errNode := macroType.Child(2)
	if errNode == nil || errNode.Type(lang) != "ERROR" || errNode.ChildCount() != 1 {
		t.Fatalf("macro_type_specifier.Child(2) = %v, want ERROR with 1 child", errNode)
	}
	if got := errNode.Child(0); got == nil || got.Type(lang) != "type_identifier" || got.Text(src) != wantMacro {
		t.Fatalf("macro_type_specifier ERROR child = %v, want type_identifier %q", got, wantMacro)
	}
	typeDescriptor := macroType.ChildByFieldName("type", lang)
	if typeDescriptor == nil || typeDescriptor.Type(lang) != "type_descriptor" {
		t.Fatalf("macro_type_specifier type field = %v, want type_descriptor", typeDescriptor)
	}
	inner := typeDescriptor.ChildByFieldName("type", lang)
	if inner == nil || inner.Type(lang) != "type_identifier" || inner.Text(src) != wantInner {
		t.Fatalf("type_descriptor type field = %v, want type_identifier %q", inner, wantInner)
	}
}

func cBlobMustParse(t *testing.T, src []byte) (*gotreesitter.Node, *gotreesitter.Language) {
	t.Helper()
	lang := CLanguage()
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	t.Cleanup(tree.Release)
	return tree.RootNode(), lang
}
