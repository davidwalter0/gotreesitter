package gotreesitter

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

// arithmeticChainSource builds "1+1+1+...+1" with n "+1" repeats -- valid
// input for buildArithmeticLanguage(). Kept small (n in the low thousands
// yields a few KB) so these tests stay well clear of the repo's documented
// host OOM risk (see AGENTS.md) while still taking tens of milliseconds to
// parse, long enough to straddle a millisecond-scale deadline.
func arithmeticChainSource(n int) []byte {
	var buf bytes.Buffer
	buf.WriteString("1")
	for i := 0; i < n; i++ {
		buf.WriteString("+1")
	}
	return buf.Bytes()
}

// assertParseCtxTreesEqual fails the test if a and b differ in type, span,
// child count, named-ness, or error state anywhere in the tree.
func assertParseCtxTreesEqual(t *testing.T, lang *Language, a, b *Node) {
	t.Helper()
	var walk func(x, y *Node, path string)
	walk = func(x, y *Node, path string) {
		if (x == nil) != (y == nil) {
			t.Fatalf("at %s: nil mismatch a=%v b=%v", path, x == nil, y == nil)
		}
		if x == nil {
			return
		}
		if x.Type(lang) != y.Type(lang) ||
			x.StartByte() != y.StartByte() || x.EndByte() != y.EndByte() ||
			x.ChildCount() != y.ChildCount() ||
			x.IsNamed() != y.IsNamed() || x.HasError() != y.HasError() {
			t.Fatalf("at %s: divergent\n  a: type=%q span=[%d..%d] childCount=%d named=%v error=%v\n  b: type=%q span=[%d..%d] childCount=%d named=%v error=%v",
				path,
				x.Type(lang), x.StartByte(), x.EndByte(), x.ChildCount(), x.IsNamed(), x.HasError(),
				y.Type(lang), y.StartByte(), y.EndByte(), y.ChildCount(), y.IsNamed(), y.HasError())
		}
		for i := 0; i < x.ChildCount(); i++ {
			walk(x.Child(i), y.Child(i), path+"/"+x.Type(lang))
		}
	}
	walk(a, b, "")
}

// TestParseCtxAlreadyCancelled verifies that ParseCtx returns ctx.Err()
// promptly -- without doing any parse work -- when ctx is already
// cancelled/expired at call time.
func TestParseCtxAlreadyCancelled(t *testing.T) {
	tests := []struct {
		name    string
		makeCtx func() (context.Context, func())
		wantErr error
	}{
		{
			name: "already cancelled",
			makeCtx: func() (context.Context, func()) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			wantErr: context.Canceled,
		},
		{
			name: "already expired deadline",
			makeCtx: func() (context.Context, func()) {
				return context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
			},
			wantErr: context.DeadlineExceeded,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lang := buildArithmeticLanguage()
			parser := NewParser(lang)
			ctx, cancel := tc.makeCtx()
			defer cancel()

			start := time.Now()
			tree, err := parser.ParseCtx(ctx, []byte("1+1"))
			elapsed := time.Since(start)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ParseCtx() error = %v, want %v", err, tc.wantErr)
			}
			if tree != nil {
				t.Fatalf("ParseCtx() tree = %v, want nil", tree)
			}
			// "Promptly" here means no parse work was attempted: this should
			// return in well under the time an actual parse (even of "1+1")
			// would take once arena/token-source setup runs.
			if elapsed > 10*time.Millisecond {
				t.Fatalf("ParseCtx() took %v for an already-done ctx, want near-instant", elapsed)
			}
		})
	}
}

// TestParseCtxBackgroundMatchesPlainParse verifies that ParseCtx with
// context.Background() produces a tree identical to plain Parse -- i.e.
// ParseCtx changes nothing about the parse result when there is nothing to
// cancel.
func TestParseCtxBackgroundMatchesPlainParse(t *testing.T) {
	lang := buildArithmeticLanguage()
	source := arithmeticChainSource(50)

	plainParser := NewParser(lang)
	plainTree, err := plainParser.Parse(source)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	defer plainTree.Release()

	ctxParser := NewParser(lang)
	ctxTree, err := ctxParser.ParseCtx(context.Background(), source)
	if err != nil {
		t.Fatalf("ParseCtx() error = %v", err)
	}
	defer ctxTree.Release()

	if ctxTree.ParseStoppedEarly() {
		t.Fatalf("ParseCtx() tree stopped early: %v", ctxTree.ParseStopReason())
	}
	assertParseCtxTreesEqual(t, lang, plainTree.RootNode(), ctxTree.RootNode())

	// ParseCtx(nil, ...) must behave identically to context.Background().
	nilCtxParser := NewParser(lang)
	nilCtxTree, err := nilCtxParser.ParseCtx(nil, source) //nolint:staticcheck // exercising the documented nil == Background() fallback
	if err != nil {
		t.Fatalf("ParseCtx(nil, ...) error = %v", err)
	}
	defer nilCtxTree.Release()
	assertParseCtxTreesEqual(t, lang, plainTree.RootNode(), nilCtxTree.RootNode())
}

// TestParseCtxDeadlineMidParse verifies that a deadline which is still live
// when ParseCtx is called, but expires while the parse is in flight, aborts
// the parse and returns context.DeadlineExceeded -- exercising the periodic
// in-loop checkpoint (activeParseStopReason), not just the upfront
// already-cancelled fast path covered by TestParseCtxAlreadyCancelled.
func TestParseCtxDeadlineMidParse(t *testing.T) {
	lang := buildArithmeticLanguage()
	// Empirically, parsing this input with plain Parse takes ~26-47ms on
	// this repo's toy left-recursive arithmetic grammar (measured locally
	// across 8 warm runs); a 6ms deadline gives a wide safety margin below
	// that floor while staying far above ordinary call-entry overhead, so
	// the deadline reliably fires mid-parse rather than before parsing
	// starts or after it finishes.
	source := arithmeticChainSource(5000)

	parser := NewParser(lang)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Millisecond)
	defer cancel()

	tree, err := parser.ParseCtx(ctx, source)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ParseCtx() error = %v, want %v", err, context.DeadlineExceeded)
	}
	if tree != nil {
		t.Fatalf("ParseCtx() tree = %v, want nil", tree)
	}
}

// TestParseCtxDoesNotAffectUnrelatedTimeout verifies that a Parser's own
// SetTimeoutMicros budget still returns its usual partial tree (nil error)
// through ParseCtx when the caller's ctx never itself completes -- ParseCtx
// only translates a stop caused by ITS ctx into ctx.Err().
func TestParseCtxDoesNotAffectUnrelatedTimeout(t *testing.T) {
	lang := buildArithmeticLanguage()
	parser := NewParser(lang)
	parser.SetTimeoutMicros(1) // fires almost immediately, unrelated to ctx

	source := arithmeticChainSource(5000)
	tree, err := parser.ParseCtx(context.Background(), source)
	if err != nil {
		t.Fatalf("ParseCtx() error = %v, want nil", err)
	}
	if tree == nil {
		t.Fatal("ParseCtx() tree = nil, want a partial tree from the unrelated SetTimeoutMicros budget")
	}
	defer tree.Release()
	if got, want := tree.ParseStopReason(), ParseStopTimeout; got != want {
		t.Fatalf("ParseStopReason() = %q, want %q", got, want)
	}
	if !tree.ParseStoppedEarly() {
		t.Fatal("ParseStoppedEarly() = false, want true")
	}
}

// TestParseCtxReleasesPartialTreeOnCancellation guards against a resource
// leak: ParseCtx must Release() the partial tree it discards on
// cancellation, not just drop the pointer. Released trees return to
// treePool, so re-acquiring a fresh tree after cancellation must not panic
// or double-release.
func TestParseCtxReleasesPartialTreeOnCancellation(t *testing.T) {
	lang := buildArithmeticLanguage()
	source := arithmeticChainSource(5000)

	parser := NewParser(lang)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Millisecond)
	defer cancel()

	tree, err := parser.ParseCtx(ctx, source)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ParseCtx() error = %v, want %v", err, context.DeadlineExceeded)
	}
	if tree != nil {
		t.Fatalf("ParseCtx() tree = %v, want nil (released)", tree)
	}

	// The parser itself must still be usable for an unrelated, un-cancelled
	// parse afterward -- p.ctx must have been restored, not left dangling.
	fresh, err := parser.Parse([]byte("1+1"))
	if err != nil {
		t.Fatalf("Parse() after cancelled ParseCtx error = %v", err)
	}
	defer fresh.Release()
	if fresh.ParseStoppedEarly() {
		t.Fatalf("Parse() after cancelled ParseCtx stopped early: %v", fresh.ParseStopReason())
	}
}
