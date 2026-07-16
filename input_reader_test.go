package gotreesitter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
)

// arithmeticSourceForTest returns a source string accepted by
// buildArithmeticLanguage's grammar (expression -> NUMBER | expression '+'
// NUMBER), long and irregular enough that small chunk sizes are guaranteed
// to split at least one multi-digit NUMBER token across a chunk boundary.
func arithmeticSourceForTest() string {
	return "123 + 4567 + 89 + 12345 + 6 + 78901234 + 5 +\n6789 + 1234567890 + 42"
}

// nodeRecord captures the parts of a Node relevant to a structural
// equivalence check: symbol identity, byte span, and point span. Text is
// intentionally re-derived from source at comparison time (not stored here)
// so the check also catches a tree whose spans are right but whose backing
// source bytes differ.
type nodeRecord struct {
	symbol     Symbol
	named      bool
	startByte  uint32
	endByte    uint32
	startPoint Point
	endPoint   Point
}

// walkAllNodes returns a pre-order flattening of every node (named and
// anonymous) under root, including root itself.
func walkAllNodes(root *Node) []nodeRecord {
	if root == nil {
		return nil
	}
	var out []nodeRecord
	var walk func(n *Node)
	walk = func(n *Node) {
		if n == nil {
			return
		}
		out = append(out, nodeRecord{
			symbol:     n.Symbol(),
			named:      n.IsNamed(),
			startByte:  n.StartByte(),
			endByte:    n.EndByte(),
			startPoint: n.StartPoint(),
			endPoint:   n.EndPoint(),
		})
		for i := 0; i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}
	walk(root)
	return out
}

// assertTreesEquivalent fails t if got and want do not represent the exact
// same parse: same source bytes, same stop reason, and the same sequence of
// node symbols/spans (named and anonymous) in pre-order, and the same
// S-expression over named nodes.
func assertTreesEquivalent(t *testing.T, lang *Language, source []byte, got, want *Tree) {
	t.Helper()
	if (got == nil) != (want == nil) {
		t.Fatalf("tree nilness mismatch: got=%v want=%v", got == nil, want == nil)
	}
	if got == nil {
		return
	}
	if got.ParseStopReason() != want.ParseStopReason() {
		t.Fatalf("ParseStopReason mismatch: got %q, want %q", got.ParseStopReason(), want.ParseStopReason())
	}
	gotRoot, wantRoot := got.RootNode(), want.RootNode()
	if (gotRoot == nil) != (wantRoot == nil) {
		t.Fatalf("root nilness mismatch: got=%v want=%v", gotRoot == nil, wantRoot == nil)
	}
	if gotRoot == nil {
		return
	}
	gotSExpr := gotRoot.SExpr(lang)
	wantSExpr := wantRoot.SExpr(lang)
	if gotSExpr != wantSExpr {
		t.Fatalf("SExpr mismatch:\n got: %s\nwant: %s", gotSExpr, wantSExpr)
	}
	gotNodes := walkAllNodes(gotRoot)
	wantNodes := walkAllNodes(wantRoot)
	if len(gotNodes) != len(wantNodes) {
		t.Fatalf("node count mismatch: got %d, want %d", len(gotNodes), len(wantNodes))
	}
	for i := range wantNodes {
		if gotNodes[i] != wantNodes[i] {
			t.Fatalf("node %d mismatch: got %+v, want %+v", i, gotNodes[i], wantNodes[i])
		}
	}
}

func TestReadAllInputReconstructsSourceAcrossChunkSizes(t *testing.T) {
	source := []byte(arithmeticSourceForTest())
	chunkSizes := []int{-1, 0, 1, 2, 3, 7, 16, len(source), len(source) * 2}
	for _, chunkSize := range chunkSizes {
		t.Run(fmt.Sprintf("chunk=%d", chunkSize), func(t *testing.T) {
			ir := NewReaderInputReader(bytes.NewReader(source), chunkSize)
			got, err := ReadAllInput(ir)
			if err != nil {
				t.Fatalf("ReadAllInput failed: %v", err)
			}
			if !bytes.Equal(got, source) {
				t.Fatalf("ReadAllInput = %q, want %q", got, source)
			}
		})
	}
}

func TestReadAllInputEmptyReaderReturnsNilNoError(t *testing.T) {
	ir := NewReaderInputReader(bytes.NewReader(nil), 4)
	got, err := ReadAllInput(ir)
	if err != nil {
		t.Fatalf("ReadAllInput failed: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("ReadAllInput of empty reader = %q, want empty", got)
	}
}

func TestReadAllInputNilReaderReturnsNilNoError(t *testing.T) {
	got, err := ReadAllInput(nil)
	if err != nil {
		t.Fatalf("ReadAllInput(nil) failed: %v", err)
	}
	if got != nil {
		t.Fatalf("ReadAllInput(nil) = %q, want nil", got)
	}
}

func TestParseReaderEquivalenceAcrossChunkBoundaries(t *testing.T) {
	lang := buildArithmeticLanguage()
	source := []byte(arithmeticSourceForTest())

	// chunk=1 forces every multi-digit NUMBER token (e.g. "78901234") to be
	// split across many chunk boundaries; chunk=3 and chunk=7 split some
	// tokens but not others; len(source) and larger deliver the whole input
	// in a single ReadAt call, matching the non-streaming baseline.
	chunkSizes := []int{1, 2, 3, 7, 16, len(source), len(source) * 2}

	want, err := NewParser(lang).Parse(source)
	if err != nil {
		t.Fatalf("baseline Parse failed: %v", err)
	}

	for _, chunkSize := range chunkSizes {
		t.Run(fmt.Sprintf("chunk=%d", chunkSize), func(t *testing.T) {
			p := NewParser(lang)
			got, err := p.ParseReader(bytes.NewReader(source), chunkSize)
			if err != nil {
				t.Fatalf("ParseReader failed: %v", err)
			}
			assertTreesEquivalent(t, lang, source, got, want)
		})
	}
}

func TestParseReaderAtEquivalenceAcrossChunkSizes(t *testing.T) {
	lang := buildArithmeticLanguage()
	source := []byte(arithmeticSourceForTest())
	chunkSizes := []int{1, 2, 5, 11, len(source), len(source) * 2}

	want, err := NewParser(lang).Parse(source)
	if err != nil {
		t.Fatalf("baseline Parse failed: %v", err)
	}

	for _, chunkSize := range chunkSizes {
		t.Run(fmt.Sprintf("chunk=%d", chunkSize), func(t *testing.T) {
			p := NewParser(lang)
			got, err := p.ParseReaderAt(bytes.NewReader(source), chunkSize)
			if err != nil {
				t.Fatalf("ParseReaderAt failed: %v", err)
			}
			assertTreesEquivalent(t, lang, source, got, want)
		})
	}
}

func TestParseInputReaderEquivalenceWithCustomCallback(t *testing.T) {
	lang := buildArithmeticLanguage()
	source := []byte(arithmeticSourceForTest())

	want, err := NewParser(lang).Parse(source)
	if err != nil {
		t.Fatalf("baseline Parse failed: %v", err)
	}

	// A hand-rolled InputReader (no io.Reader/io.ReaderAt involved) that
	// hands back exactly one byte per call, exercising the InputReaderFunc
	// adapter and confirming the position hint tracks StartPoint/EndPoint
	// correctly one byte at a time.
	var offset int
	oneByteAtATime := InputReaderFunc(func(byteOffset uint32, position Point) ([]byte, error) {
		if int(byteOffset) != offset {
			return nil, fmt.Errorf("unexpected byte offset %d, want %d", byteOffset, offset)
		}
		if offset >= len(source) {
			return nil, nil
		}
		b := source[offset]
		offset++
		return []byte{b}, nil
	})

	p := NewParser(lang)
	got, err := p.ParseInputReader(oneByteAtATime)
	if err != nil {
		t.Fatalf("ParseInputReader failed: %v", err)
	}
	assertTreesEquivalent(t, lang, source, got, want)
}

func TestParseReaderEmptyInputMatchesDirectParse(t *testing.T) {
	lang := buildArithmeticLanguage()

	want, wantErr := NewParser(lang).Parse(nil)
	p := NewParser(lang)
	got, err := p.ParseReader(strings.NewReader(""), 4)

	if (err == nil) != (wantErr == nil) {
		t.Fatalf("error mismatch: got %v, want %v", err, wantErr)
	}
	assertTreesEquivalent(t, lang, nil, got, want)
}

func TestReaderInputReaderRejectsOutOfOrderReads(t *testing.T) {
	ir := NewReaderInputReader(strings.NewReader("hello world"), 4)
	if _, err := ir.ReadAt(0, Point{}); err != nil {
		t.Fatalf("first sequential ReadAt failed: %v", err)
	}
	if _, err := ir.ReadAt(0, Point{}); err == nil {
		t.Fatal("re-reading offset 0 after it was already consumed should error, got nil")
	}
	if _, err := ir.ReadAt(100, Point{}); err == nil {
		t.Fatal("jumping ahead to offset 100 should error, got nil")
	}
}

// erroringReader returns a non-EOF error after emitting n bytes.
type erroringReader struct {
	data []byte
	pos  int
	fail error
}

func (r *erroringReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, r.fail
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func TestParseReaderPropagatesUnderlyingReaderError(t *testing.T) {
	lang := buildArithmeticLanguage()
	wantErr := errors.New("boom")
	r := &erroringReader{data: []byte("12"), fail: wantErr}

	p := NewParser(lang)
	tree, err := p.ParseReader(r, 4)
	if tree != nil {
		t.Fatalf("expected nil tree on reader error, got %v", tree)
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("ParseReader error = %v, want wrapping %v", err, wantErr)
	}
}

func TestParseReaderAtToleratesOutOfOrderReads(t *testing.T) {
	source := []byte(arithmeticSourceForTest())
	ir := NewReaderAtInputReader(bytes.NewReader(source), 4)

	// io.ReaderAt-backed InputReader supports arbitrary offsets, unlike the
	// io.Reader-backed one: read the tail before the head.
	tail, err := ir.ReadAt(uint32(len(source)-4), Point{})
	if err != nil {
		t.Fatalf("ReadAt(tail) failed: %v", err)
	}
	if !bytes.Equal(tail, source[len(source)-4:]) {
		t.Fatalf("ReadAt(tail) = %q, want %q", tail, source[len(source)-4:])
	}
	head, err := ir.ReadAt(0, Point{})
	if err != nil {
		t.Fatalf("ReadAt(head) failed: %v", err)
	}
	if !bytes.Equal(head, source[:4]) {
		t.Fatalf("ReadAt(head) = %q, want %q", head, source[:4])
	}
}

var _ io.Reader = (*erroringReader)(nil)
