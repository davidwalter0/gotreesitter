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

// TestCBlobMacroCallingConventionMultiParamLeftUntouched asserts the
// conservative fallback: parameter-list shapes outside the empirically
// verified set (here: 2 parameters) are left completely unmodified rather
// than guessed, since the oracle's GLR error-recovery shape for arbitrary
// multi-parameter lists was not validated.
func TestCBlobMacroCallingConventionMultiParamLeftUntouched(t *testing.T) {
	src := []byte("typedef GLint (GL_APIENTRYP PFNGLGETATTRIBLOCATIONPROC) (GLuint program, const GLchar *name);\n")
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
