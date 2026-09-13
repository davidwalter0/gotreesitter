package grammars

import (
	"testing"

	"github.com/davidwalter0/gotreesitter"
)

// The ImageMagick headers declare exported functions as
// `extern MagickExport RetType <declarator-list>;`, where the `MagickExport`
// export macro sits between `extern` and the real return type. Parsing
// pre-preprocessor, gt sees two adjacent type names and its error recovery
// truncates the declaration at the second name, scattering the declarator list
// into follow-on fragments; the pinned tree-sitter-c oracle instead keeps a
// single declaration whose type is the macro, wraps the real return type in an
// ERROR, and parses the declarator list normally. normalizeCExternMacroMulti-
// Declarator reconstructs the oracle's shape by reparsing the declarator run.
//
// The three tests below cover the three gt fragmentation shapes (pointer
// function-pointer list, single non-pointer function, multi non-pointer list).

func assertExternMacroMergedHeader(t *testing.T, src []byte, lang *gotreesitter.Language, decl *gotreesitter.Node, wantType, wantErr string) {
	t.Helper()
	if decl == nil || decl.Type(lang) != "declaration" {
		t.Fatalf("expected a single merged declaration, got %v", decl)
	}
	sc := decl.Child(0)
	if sc == nil || sc.Type(lang) != "storage_class_specifier" || sc.Text(src) != "extern" {
		t.Fatalf("decl.Child(0) = %v, want storage_class_specifier \"extern\"", sc)
	}
	typ := decl.ChildByFieldName("type", lang)
	if typ == nil || typ.Type(lang) != "type_identifier" || typ.Text(src) != wantType {
		t.Fatalf("type field = %v, want type_identifier %q", typ, wantType)
	}
	errNode := decl.Child(2)
	if errNode == nil || errNode.Type(lang) != "ERROR" || !errNode.IsExtra() || errNode.ChildCount() != 1 {
		t.Fatalf("decl.Child(2) = %v, want extra ERROR with 1 child", errNode)
	}
	if inner := errNode.Child(0); inner == nil || inner.Type(lang) != "identifier" || inner.Text(src) != wantErr {
		t.Fatalf("ERROR child = %v, want identifier %q", inner, wantErr)
	}
}

// TestCBlobExternMacroPointerList covers the pointer function-pointer list
// (real magick/effect.h shape): gt splits it into declaration +
// expression_statement.
func TestCBlobExternMacroPointerList(t *testing.T) {
	src := []byte("extern MagickExport Image *AImage(const Image *),*BImage(const Image *);\n")
	root, lang := cBlobMustParse(t, src)
	if root.NamedChildCount() != 1 {
		t.Fatalf("expected a single merged top-level node, got %s", root.SExpr(lang))
	}
	decl := root.NamedChild(0)
	assertExternMacroMergedHeader(t, src, lang, decl, "MagickExport", "Image")

	var ptrs int
	for i := 0; i < decl.ChildCount(); i++ {
		c := decl.Child(i)
		if c != nil && c.Type(lang) == "pointer_declarator" {
			ptrs++
			if decl.FieldNameForChild(i, lang) != "declarator" {
				t.Fatalf("pointer_declarator at %d lacks 'declarator' field", i)
			}
		}
	}
	if ptrs != 2 {
		t.Fatalf("want 2 pointer_declarators, got %d (%s)", ptrs, decl.SExpr(lang))
	}
	if last := decl.Child(decl.ChildCount() - 1); last == nil || last.Type(lang) != ";" {
		t.Fatalf("declaration should end with ;, got %v", last)
	}
}

// TestCBlobExternMacroSingleFunction covers a single non-pointer function
// (real magick/animate.h shape): gt splits it into declaration +
// macro_type_specifier + ;.
func TestCBlobExternMacroSingleFunction(t *testing.T) {
	src := []byte("extern MagickExport MagickBooleanType AnimateImages(const ImageInfo *,Image *);\n")
	root, lang := cBlobMustParse(t, src)
	if root.NamedChildCount() != 1 {
		t.Fatalf("expected a single merged top-level node, got %s", root.SExpr(lang))
	}
	decl := root.NamedChild(0)
	assertExternMacroMergedHeader(t, src, lang, decl, "MagickExport", "MagickBooleanType")

	fd := decl.ChildByFieldName("declarator", lang)
	if fd == nil || fd.Type(lang) != "function_declarator" {
		t.Fatalf("declarator field = %v, want function_declarator", fd)
	}
	if name := fd.ChildByFieldName("declarator", lang); name == nil || name.Text(src) != "AnimateImages" {
		t.Fatalf("function name = %v, want \"AnimateImages\"", name)
	}
	if pl := fd.ChildByFieldName("parameters", lang); pl == nil || pl.Type(lang) != "parameter_list" {
		t.Fatalf("parameters = %v, want parameter_list", pl)
	}
}

// TestCBlobExternMacroMultiFunction covers a multi-declarator non-pointer list
// (real magick/composite.h shape): gt splits it into declaration + declaration.
func TestCBlobExternMacroMultiFunction(t *testing.T) {
	src := []byte("extern MagickExport MagickBooleanType Comp(Image *,const Image *),Text(Image *);\n")
	root, lang := cBlobMustParse(t, src)
	if root.NamedChildCount() != 1 {
		t.Fatalf("expected a single merged top-level node, got %s", root.SExpr(lang))
	}
	decl := root.NamedChild(0)
	assertExternMacroMergedHeader(t, src, lang, decl, "MagickExport", "MagickBooleanType")

	var fns int
	for i := 0; i < decl.ChildCount(); i++ {
		c := decl.Child(i)
		if c != nil && c.Type(lang) == "function_declarator" {
			fns++
		}
	}
	if fns != 2 {
		t.Fatalf("want 2 function_declarators, got %d (%s)", fns, decl.SExpr(lang))
	}
}

// TestCBlobExternMacroWellFormedUntouched asserts a normal, well-formed extern
// declaration (no export macro, no error) is left entirely alone.
func TestCBlobExternMacroWellFormedUntouched(t *testing.T) {
	src := []byte("extern int globalCounter;\n")
	root, lang := cBlobMustParse(t, src)
	if root.HasError() {
		t.Fatalf("well-formed declaration should not error: %s", root.SExpr(lang))
	}
	decl := root.NamedChild(0)
	if decl == nil || decl.Type(lang) != "declaration" || decl.HasError() {
		t.Fatalf("expected clean declaration, got %s", root.SExpr(lang))
	}
}
