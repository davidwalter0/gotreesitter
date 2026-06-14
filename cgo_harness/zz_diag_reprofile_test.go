//go:build cgo && treesitter_c_parity

package cgoharness

import (
	"os"
	"testing"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// TestDiagReproFile dumps the Go and C parse trees (sexp) for a single file so
// recovery divergences can be inspected node-by-node.
//
//	REPRO_LANG  grammar name
//	REPRO_FILE  path to the source file
func TestDiagReproFile(t *testing.T) {
	name := os.Getenv("REPRO_LANG")
	file := os.Getenv("REPRO_FILE")
	if name == "" || file == "" {
		t.Skip("set REPRO_LANG and REPRO_FILE")
	}
	src, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var goLang *gts.Language
	for _, e := range grammars.AllLanguages() {
		if e.Name == name {
			goLang = e.Language()
			break
		}
	}
	if goLang == nil {
		t.Fatalf("%s: not in grammars.AllLanguages", name)
	}
	cLang, err := ParityCLanguage(name)
	if err != nil {
		t.Fatalf("%s: no C reference: %v", name, err)
	}

	gp := gts.NewParser(goLang)
	if os.Getenv("GOT_DIAG_TRACE") == "1" {
		gp.SetGLRTrace(true)
	}
	gtree, _ := gp.Parse(src)
	t.Logf("source %d bytes, stopReason=%v", len(src), gtree.ParseStopReason())
	if gtree != nil && gtree.RootNode() != nil {
		r := gtree.RootNode()
		t.Logf("GO  root=%s err=%v span=%d:%d cc=%d", r.Type(goLang), r.HasError(), r.StartByte(), r.EndByte(), r.ChildCount())
		t.Logf("GO  sexp:\n%s", r.SExpr(goLang))
	}

	cp := sitter.NewParser()
	_ = cp.SetLanguage(cLang)
	ct := cp.Parse(src, nil)
	if ct != nil && ct.RootNode() != nil {
		cr := ct.RootNode()
		t.Logf("C   root=%s err=%v span=%d:%d cc=%d", cr.Kind(), cr.HasError(), cr.StartByte(), cr.EndByte(), cr.ChildCount())
		t.Logf("C   sexp:\n%s", stripFieldLabels(cr.ToSexp()))
	}
}
