//go:build cgo && treesitter_c_parity

package cgoharness

import (
	"os"
	"testing"

	gts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// TestGoFailWhere: for REPRO_LANG+REPRO_FILE, parse with Go and print the first
// ERROR/MISSING node + a source snippet — pinpoints where Go's parser breaks on
// valid (C-parseable) input. Phase-1 goFail diagnosis tool.
func TestGoFailWhere(t *testing.T) {
	name := os.Getenv("REPRO_LANG")
	fp := os.Getenv("REPRO_FILE")
	if name == "" || fp == "" {
		t.Skip("set REPRO_LANG + REPRO_FILE")
	}
	var goLang *gts.Language
	for _, e := range grammars.AllLanguages() {
		if e.Name == name {
			goLang = e.Language()
			break
		}
	}
	if goLang == nil {
		t.Fatalf("%s not in AllLanguages", name)
	}
	src, err := os.ReadFile(fp)
	if err != nil {
		t.Fatal(err)
	}
	p := gts.NewParser(goLang)
	tree, _ := p.Parse(src)
	if tree == nil || tree.RootNode() == nil {
		t.Fatalf("nil tree")
	}
	root := tree.RootNode()
	t.Logf("root=%s err=%v span=[%d-%d] childCount=%d", root.Type(goLang), root.HasError(), root.StartByte(), root.EndByte(), root.ChildCount())

	var find func(n *gts.Node) *gts.Node
	find = func(n *gts.Node) *gts.Node {
		if n.IsError() || n.IsMissing() {
			return n
		}
		for i := 0; i < n.ChildCount(); i++ {
			c := n.Child(i)
			if c == nil {
				continue
			}
			if r := find(c); r != nil {
				return r
			}
		}
		return nil
	}
	e := find(root)
	if e == nil {
		t.Logf("NO error/missing node (root err=%v) — divergence is shape, not ERROR", root.HasError())
		return
	}
	s, en := e.StartByte(), e.EndByte()
	var lo uint32
	if s > 50 {
		lo = s - 50
	}
	hi := en + 50
	if hi > uint32(len(src)) {
		hi = uint32(len(src))
	}
	kind := "ERROR"
	if e.IsMissing() {
		kind = "MISSING"
	}
	t.Logf("FIRST-%s type=%s span=[%d-%d]\n  CONTEXT: %q", kind, e.Type(goLang), s, en, string(src[lo:hi]))
}
