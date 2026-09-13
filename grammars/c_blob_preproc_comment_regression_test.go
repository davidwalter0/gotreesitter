package grammars

import (
	"testing"

	"github.com/davidwalter0/gotreesitter"
)

// TestCBlobPreprocArgLeadingCommentSplit guards the blob-path fix for an inline
// block comment between a preprocessor directive and its argument (real corpus
// entry: libxml2's posix config.h, `#  undef /**/ HAVE_MMAP`). The pinned
// tree-sitter-c oracle (github.com/tree-sitter/tree-sitter-c @ ae19b676) emits
// the `/**/` as a separate extra `comment` child of the preproc_call and starts
// the `argument` preproc_arg at the following token; gt's preproc lexer folds
// the comment into the preproc_arg, so a post-parse pass splits it back out.
func TestCBlobPreprocArgLeadingCommentSplit(t *testing.T) {
	src := []byte("#if 1\n#  undef /**/ HAVE_MMAP\n#endif\n")
	root, lang := cBlobMustParse(t, src)

	preprocIf := root.NamedChild(0)
	if preprocIf == nil || preprocIf.Type(lang) != "preproc_if" {
		t.Fatalf("expected preproc_if, got %s", root.SExpr(lang))
	}

	var call *gotreesitter.Node
	for i := 0; i < preprocIf.ChildCount(); i++ {
		if c := preprocIf.Child(i); c != nil && c.Type(lang) == "preproc_call" {
			call = c
			break
		}
	}
	if call == nil {
		t.Fatalf("expected a preproc_call child, got %s", root.SExpr(lang))
	}
	if got, want := call.ChildCount(), 3; got != want {
		t.Fatalf("preproc_call child count = %d, want %d (full tree: %s)", got, want, root.SExpr(lang))
	}

	directive := call.Child(0)
	if directive == nil || directive.Type(lang) != "preproc_directive" || call.FieldNameForChild(0, lang) != "directive" {
		t.Fatalf("preproc_call.Child(0) = %v (field %q), want preproc_directive field \"directive\"", directive, call.FieldNameForChild(0, lang))
	}

	comment := call.Child(1)
	if comment == nil || comment.Type(lang) != "comment" || comment.Text(src) != "/**/" {
		t.Fatalf("preproc_call.Child(1) = %v, want comment \"/**/\"", comment)
	}
	if !comment.IsExtra() {
		t.Fatalf("comment node should be marked extra")
	}
	if call.FieldNameForChild(1, lang) != "" {
		t.Fatalf("comment child should carry no field, got %q", call.FieldNameForChild(1, lang))
	}

	arg := call.Child(2)
	if arg == nil || arg.Type(lang) != "preproc_arg" || arg.Text(src) != "HAVE_MMAP" || call.FieldNameForChild(2, lang) != "argument" {
		t.Fatalf("preproc_call.Child(2) = %v (field %q), want preproc_arg \"HAVE_MMAP\" field \"argument\"", arg, call.FieldNameForChild(2, lang))
	}
}

// TestCBlobPreprocArgNoLeadingCommentUntouched asserts the pass is inert for a
// plain preproc argument with no leading comment: the argument keeps its full
// text and the preproc_call keeps exactly two children.
func TestCBlobPreprocArgNoLeadingCommentUntouched(t *testing.T) {
	src := []byte("#if 1\n#  undef HAVE_MMAP\n#endif\n")
	root, lang := cBlobMustParse(t, src)

	preprocIf := root.NamedChild(0)
	var call *gotreesitter.Node
	for i := 0; i < preprocIf.ChildCount(); i++ {
		if c := preprocIf.Child(i); c != nil && c.Type(lang) == "preproc_call" {
			call = c
			break
		}
	}
	if call == nil {
		t.Fatalf("expected a preproc_call child, got %s", root.SExpr(lang))
	}
	if got, want := call.ChildCount(), 2; got != want {
		t.Fatalf("preproc_call child count = %d, want %d (no-op expected)", got, want)
	}
	arg := call.Child(1)
	if arg == nil || arg.Type(lang) != "preproc_arg" || arg.Text(src) != "HAVE_MMAP" {
		t.Fatalf("preproc_call.Child(1) = %v, want preproc_arg \"HAVE_MMAP\"", arg)
	}
}
