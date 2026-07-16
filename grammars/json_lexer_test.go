//go:build !grammar_subset || grammar_subset_json

package grammars

import (
	"bytes"
	"testing"

	"github.com/davidwalter0/gotreesitter"
)

func TestNewJSONTokenSourceReturnsErrorOnMissingSymbols(t *testing.T) {
	lang := &gotreesitter.Language{
		TokenCount:  1,
		SymbolNames: []string{"end"},
	}
	if _, err := NewJSONTokenSource([]byte(`{"a":1}`), lang); err == nil {
		t.Fatal("expected error for language missing json token symbols")
	}
}

func TestNewJSONTokenSourceOrEOFFallsBack(t *testing.T) {
	lang := &gotreesitter.Language{
		TokenCount:  1,
		SymbolNames: []string{"end"},
	}
	ts := NewJSONTokenSourceOrEOF([]byte(`{"a":1}`), lang)
	tok := ts.Next()
	if tok.Symbol != 0 {
		t.Fatalf("fallback token symbol = %d, want EOF (0)", tok.Symbol)
	}
}

func TestJSONTokenSourceSplitsStringEscapes(t *testing.T) {
	lang := JsonLanguage()
	src := []byte(`{"a":"x\n\u0041"}`)
	ts, err := NewJSONTokenSource(src, lang)
	if err != nil {
		t.Fatalf("NewJSONTokenSource failed: %v", err)
	}

	var sawContent, sawEscape bool
	for i := 0; i < 64; i++ {
		tok := ts.Next()
		if tok.Symbol == 0 {
			break
		}
		typ := lang.SymbolNames[tok.Symbol]
		if typ == "string_content" {
			sawContent = true
		}
		if typ == "escape_sequence" {
			sawEscape = true
		}
	}

	if !sawContent {
		t.Fatal("expected at least one string_content token")
	}
	if !sawEscape {
		t.Fatal("expected at least one escape_sequence token")
	}
}

func TestJSONTokenSourceSkipToByte(t *testing.T) {
	lang := JsonLanguage()
	src := []byte(`{"a":1, "target": 2}`)
	ts, err := NewJSONTokenSource(src, lang)
	if err != nil {
		t.Fatalf("NewJSONTokenSource failed: %v", err)
	}

	target := uint32(8) // points near "target"
	tok := ts.SkipToByte(target)
	if tok.Symbol == 0 {
		t.Fatal("SkipToByte unexpectedly returned EOF")
	}
	if tok.StartByte < target {
		t.Fatalf("token starts before target: got %d, target %d", tok.StartByte, target)
	}
}

func TestJSONTokenSourceSkipToByteInsideStringContent(t *testing.T) {
	lang := JsonLanguage()
	src := []byte(`{"version": "0.24.8", "target": "x"}`)
	ts, err := NewJSONTokenSource(src, lang)
	if err != nil {
		t.Fatalf("NewJSONTokenSource failed: %v", err)
	}

	target := bytes.Index(src, []byte("24.8"))
	if target < 0 {
		t.Fatal("test source missing target string")
	}
	tok := ts.SkipToByte(uint32(target))
	if got := lang.SymbolNames[tok.Symbol]; got != "string_content" {
		t.Fatalf("SkipToByte token = %q, want string_content; token=%+v", got, tok)
	}
	if got, want := tok.StartByte, uint32(target); got != want {
		t.Fatalf("StartByte=%d, want %d", got, want)
	}
	if got := tok.Text; got != "24.8" {
		t.Fatalf("Text=%q, want %q", got, "24.8")
	}
	next := ts.Next()
	if got := lang.SymbolNames[next.Symbol]; got != "\"" {
		t.Fatalf("next token after clipped string content = %q, want quote; token=%+v", got, next)
	}
	if next.StartByte <= tok.StartByte {
		t.Fatalf("next token did not advance after clipped token: current=%+v next=%+v", tok, next)
	}
	if _, ok := any(ts).(gotreesitter.PointSkippableTokenSource); !ok {
		t.Fatal("JSONTokenSource should implement PointSkippableTokenSource")
	}
}

func TestParseJSONWithTokenSource(t *testing.T) {
	lang := JsonLanguage()
	parser := gotreesitter.NewParser(lang)
	src := []byte(`{"a":[1,true,null,false,"x\n"],"b":{"c":2}}`)
	ts, err := NewJSONTokenSource(src, lang)
	if err != nil {
		t.Fatalf("NewJSONTokenSource failed: %v", err)
	}

	tree, err := parser.ParseWithTokenSource(src, ts)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if tree == nil || tree.RootNode() == nil {
		t.Fatal("parse returned nil root")
	}
	if tree.RootNode().HasError() {
		t.Fatal("expected json parse without syntax errors")
	}
}

func TestParseJSONIncrementalWithTokenSourceStringEdit(t *testing.T) {
	lang := JsonLanguage()
	parser := gotreesitter.NewParser(lang)
	oldSrc := []byte(`{"version": "0.24.8", "target": "x"}`)
	newSrc := append([]byte(nil), oldSrc...)
	editAt := bytes.IndexByte(newSrc, '0')
	if editAt < 0 {
		t.Fatal("test source missing edit byte")
	}
	newSrc[editAt] = '1'

	oldTree, err := parser.ParseWithTokenSource(oldSrc, NewJSONTokenSourceOrEOF(oldSrc, lang))
	if err != nil {
		t.Fatalf("old parse failed: %v", err)
	}
	defer oldTree.Release()
	freshTree, err := parser.ParseWithTokenSource(newSrc, NewJSONTokenSourceOrEOF(newSrc, lang))
	if err != nil {
		t.Fatalf("fresh parse failed: %v", err)
	}
	defer freshTree.Release()

	edit := gotreesitter.InputEdit{
		StartByte:   uint32(editAt),
		OldEndByte:  uint32(editAt + 1),
		NewEndByte:  uint32(editAt + 1),
		StartPoint:  jsonTestPointAtOffset(oldSrc, editAt),
		OldEndPoint: jsonTestPointAtOffset(oldSrc, editAt+1),
		NewEndPoint: jsonTestPointAtOffset(newSrc, editAt+1),
	}
	oldTree.Edit(edit)

	incrTree, err := parser.ParseIncrementalWithTokenSource(newSrc, oldTree, NewJSONTokenSourceOrEOF(newSrc, lang))
	if err != nil {
		t.Fatalf("incremental parse failed: %v", err)
	}
	defer incrTree.Release()

	if incrTree.RootNode().HasError() {
		t.Fatalf("incremental parse has error: %s", incrTree.RootNode().SExpr(lang))
	}
	if got, want := incrTree.RootNode().SExpr(lang), freshTree.RootNode().SExpr(lang); got != want {
		t.Fatalf("incremental SExpr mismatch\n got: %s\nwant: %s", got, want)
	}
}

func jsonTestPointAtOffset(src []byte, offset int) gotreesitter.Point {
	var pt gotreesitter.Point
	if offset > len(src) {
		offset = len(src)
	}
	for i := 0; i < offset; i++ {
		if src[i] == '\n' {
			pt.Row++
			pt.Column = 0
		} else {
			pt.Column++
		}
	}
	return pt
}

// TestJSONTokenSourceMultibyteColumnIsByteBased is a regression test for a
// UTF-8 Point.Column undercount on the normal (non-error) parse path: JSON
// string content containing multi-byte UTF-8 runes (curly quotes, em-dash)
// caused advanceOneRune to advance the byte cursor by the rune's full byte
// width while bumping the column by only 1, undercounting every node's
// column past the multi-byte rune on that line. tree-sitter's Point.Column
// is a byte offset from the start of the line, not a codepoint count, so
// every node's Start/EndPoint must agree with a byte-for-byte scan of the
// source (jsonTestPointAtOffset), including the two real-world corpus
// patterns that exposed this: consecutive curly quotes around a word, and a
// bare em-dash later on the same line.
func TestJSONTokenSourceMultibyteColumnIsByteBased(t *testing.T) {
	lang := JsonLanguage()
	parser := gotreesitter.NewParser(lang)
	src := []byte("{\n  \"a\": \"install.packages(“languageserver”)\",\n  \"b\": \"x — y\",\n  \"c\": 3\n}\n")
	ts, err := NewJSONTokenSource(src, lang)
	if err != nil {
		t.Fatalf("NewJSONTokenSource failed: %v", err)
	}
	tree, err := parser.ParseWithTokenSource(src, ts)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if tree.RootNode().HasError() {
		t.Fatalf("expected no parse errors: %s", tree.RootNode().SExpr(lang))
	}

	checked := 0
	var walk func(n *gotreesitter.Node)
	walk = func(n *gotreesitter.Node) {
		wantStart := jsonTestPointAtOffset(src, int(n.StartByte()))
		wantEnd := jsonTestPointAtOffset(src, int(n.EndByte()))
		if got := n.StartPoint(); got != wantStart {
			t.Errorf("node %s [%d..%d]: StartPoint = %+v, want %+v (byte-based)",
				n.Type(lang), n.StartByte(), n.EndByte(), got, wantStart)
		}
		if got := n.EndPoint(); got != wantEnd {
			t.Errorf("node %s [%d..%d]: EndPoint = %+v, want %+v (byte-based)",
				n.Type(lang), n.StartByte(), n.EndByte(), got, wantEnd)
		}
		checked++
		for i := 0; i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}
	walk(tree.RootNode())
	if checked == 0 {
		t.Fatal("walked zero nodes")
	}
}

// TestJSONTokenSourceSkipToByteAfterMultibyteRuneIsByteBased is a regression
// test for advanceJSONPoint (used by clipStringContentToken when SkipToByte
// lands inside a cached string_content token): it must recompute the
// clipped token's StartPoint by byte width, not codepoint count, when the
// skipped prefix contains a multi-byte UTF-8 rune.
func TestJSONTokenSourceSkipToByteAfterMultibyteRuneIsByteBased(t *testing.T) {
	lang := JsonLanguage()
	// string_content is `pre—post` (em-dash is 3 bytes): byte offsets
	// 7..17 hold "pre" (3) + em-dash (3) + "post" (4).
	src := []byte("{\"a\": \"pre—post\"}")
	ts, err := NewJSONTokenSource(src, lang)
	if err != nil {
		t.Fatalf("NewJSONTokenSource failed: %v", err)
	}

	target := bytes.Index(src, []byte("post"))
	if target < 0 {
		t.Fatal("test source missing target string")
	}
	// Force the token cache path (clipStringContentToken) rather than the
	// streaming lexer, matching how SkipToByte is actually driven by the
	// incremental-reuse parser.
	tok := ts.SkipToByte(uint32(target))
	if got := lang.SymbolNames[tok.Symbol]; got != "string_content" {
		t.Fatalf("SkipToByte token = %q, want string_content; token=%+v", got, tok)
	}
	if got, want := tok.StartByte, uint32(target); got != want {
		t.Fatalf("StartByte=%d, want %d", got, want)
	}
	if got, want := tok.Text, "post"; got != want {
		t.Fatalf("Text=%q, want %q", got, want)
	}
	wantStart := jsonTestPointAtOffset(src, target)
	if got := tok.StartPoint; got != wantStart {
		t.Fatalf("clipped token StartPoint = %+v, want %+v (byte-based)", got, wantStart)
	}
}
