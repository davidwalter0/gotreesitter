package gotreesitter

import "testing"

// ---------------------------------------------------------------------------
// General predicate passthrough tests (M4).
//
// Before this change, an unrecognized predicate name (anything not in
// gotreesitter's built-in set: #eq?, #match?, #set!, #count?, etc.) was a
// hard compile error. Upstream tree-sitter instead surfaces such predicates
// to the caller as "general predicates" (official cgo binding:
// Query.GeneralPredicates() / QueryPredicateGeneral) for the caller to
// evaluate. These tests cover the same model in gotreesitter:
//   - unknown predicates compile and are retrievable via
//     Query.GeneralPredicates, with the name and parsed args intact;
//   - known/standard predicates keep their existing auto-applied,
//     in-engine behavior, and are never misclassified as general;
//   - a malformed KNOWN predicate (e.g. wrong arity) still errors — the
//     passthrough only applies to predicate *names* gotreesitter doesn't
//     recognize, not to malformed calls of ones it does.
// ---------------------------------------------------------------------------

func TestParsePredicateGeneralPassthroughTable(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantName string
		wantArgs []GeneralPredicateArg
	}{
		{
			name:     "capture and string literal",
			query:    `(identifier) @a (#custom? @a "x")`,
			wantName: "#custom?",
			wantArgs: []GeneralPredicateArg{
				{IsCapture: true, Capture: "a"},
				{Literal: "x"},
			},
		},
		{
			name:     "capture only",
			query:    `(identifier) @a (#is-cool? @a)`,
			wantName: "#is-cool?",
			wantArgs: []GeneralPredicateArg{
				{IsCapture: true, Capture: "a"},
			},
		},
		{
			name:     "no arguments",
			query:    `(identifier) @a (#always-true?)`,
			wantName: "#always-true?",
			wantArgs: nil,
		},
		{
			name:     "multiple mixed args including a bare atom",
			query:    `(identifier) @a (#weird! @a "one" two @a)`,
			wantName: "#weird!",
			wantArgs: []GeneralPredicateArg{
				{IsCapture: true, Capture: "a"},
				{Literal: "one"},
				{Literal: "two"},
				{IsCapture: true, Capture: "a"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lang := queryTestLanguage()
			q, err := NewQuery(tt.query, lang)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			general, ok := q.GeneralPredicates(0)
			if !ok {
				t.Fatalf("GeneralPredicates: ok=false")
			}
			if len(general) != 1 {
				t.Fatalf("general predicates: got %d, want 1", len(general))
			}
			if general[0].Name != tt.wantName {
				t.Fatalf("name: got %q, want %q", general[0].Name, tt.wantName)
			}
			if len(general[0].Args) != len(tt.wantArgs) {
				t.Fatalf("args len: got %d, want %d (%+v)", len(general[0].Args), len(tt.wantArgs), general[0].Args)
			}
			for i, want := range tt.wantArgs {
				if got := general[0].Args[i]; got != want {
					t.Fatalf("arg[%d]: got %+v, want %+v", i, got, want)
				}
			}

			// A general-only query must not leak into PredicatesForPattern
			// as anything other than predicateGeneral.
			preds, ok := q.PredicatesForPattern(0)
			if !ok || len(preds) != 1 {
				t.Fatalf("PredicatesForPattern: got %d (ok=%v), want 1", len(preds), ok)
			}
			if preds[0].kind != predicateGeneral {
				t.Fatalf("predicate kind: got %d, want predicateGeneral", preds[0].kind)
			}
		})
	}
}

// TestMatchPredicateGeneralDoesNotAffectMatching verifies that a general
// predicate is never evaluated in-engine: it neither filters matches nor
// panics during execution. The caller is solely responsible for applying
// it (e.g. the way cgo_harness/cmd/parse_gap_report post-filters matches
// against the official binding's GeneralPredicates for comparison).
func TestMatchPredicateGeneralDoesNotAffectMatching(t *testing.T) {
	lang := queryTestLanguage()
	tree := buildSimpleTree(lang)

	q, err := NewQuery(`(identifier) @name (#custom? @name "whatever")`, lang)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	matches := q.Execute(tree)
	if len(matches) != 1 {
		t.Fatalf("matches: got %d, want 1 (general predicate must not filter matches in-engine)", len(matches))
	}
}

// TestParsePredicateMixedStandardAndGeneral covers a pattern with both a
// standard predicate (auto-applied, in-engine) and a general predicate
// (passthrough) attached: the standard predicate keeps filtering matches,
// and the general predicate is retrievable independently.
func TestParsePredicateMixedStandardAndGeneral(t *testing.T) {
	lang := queryTestLanguage()
	q, err := NewQuery(`(identifier) @name (#eq? @name "main") (#custom? @name "x")`, lang)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	preds, ok := q.PredicatesForPattern(0)
	if !ok || len(preds) != 2 {
		t.Fatalf("predicates: got %d, want 2 (ok=%v)", len(preds), ok)
	}
	if preds[0].kind != predicateEq {
		t.Fatalf("preds[0].kind: got %d, want predicateEq", preds[0].kind)
	}
	if preds[1].kind != predicateGeneral {
		t.Fatalf("preds[1].kind: got %d, want predicateGeneral", preds[1].kind)
	}

	general, ok := q.GeneralPredicates(0)
	if !ok || len(general) != 1 {
		t.Fatalf("general predicates: got %d, want 1 (ok=%v)", len(general), ok)
	}
	if general[0].Name != "#custom?" {
		t.Fatalf("name: got %q, want %q", general[0].Name, "#custom?")
	}

	// The standard #eq? predicate still filters matches: buildSimpleTree's
	// identifier text is "main".
	tree := buildSimpleTree(lang)
	matches := q.Execute(tree)
	if len(matches) != 1 {
		t.Fatalf("matches: got %d, want 1", len(matches))
	}
}

// TestParsePredicateStandardOnlyUnchanged is a regression check: a query
// using only standard predicates behaves exactly as before this change —
// same predicate classification, same match filtering, and no general
// predicates reported.
func TestParsePredicateStandardOnlyUnchanged(t *testing.T) {
	lang := queryTestLanguage()

	q, err := NewQuery(`(identifier) @name (#eq? @name "main")`, lang)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	preds, ok := q.PredicatesForPattern(0)
	if !ok || len(preds) != 1 || preds[0].kind != predicateEq {
		t.Fatalf("predicates: got %+v (ok=%v), want single predicateEq", preds, ok)
	}

	general, ok := q.GeneralPredicates(0)
	if !ok {
		t.Fatalf("GeneralPredicates: ok=false")
	}
	if len(general) != 0 {
		t.Fatalf("general predicates: got %d, want 0", len(general))
	}

	tree := buildSimpleTree(lang)
	matches := q.Execute(tree)
	if len(matches) != 1 {
		t.Fatalf("matches: got %d, want 1 (standard #eq? behavior unchanged)", len(matches))
	}

	qNoMatch, err := NewQuery(`(identifier) @name (#eq? @name "nope")`, lang)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if matches := qNoMatch.Execute(tree); len(matches) != 0 {
		t.Fatalf("matches: got %d, want 0 (standard #eq? behavior unchanged)", len(matches))
	}
}

// TestParsePredicateKnownStillErrorsOnBadArity proves the passthrough only
// applies to predicate *names* gotreesitter doesn't recognize — a
// recognized predicate called with the wrong number of arguments must
// still be a compile error, not silently accepted as "general".
func TestParsePredicateKnownStillErrorsOnBadArity(t *testing.T) {
	lang := queryTestLanguage()
	// #eq? requires exactly two arguments (a capture, then a capture or
	// string literal); giving it only one must still fail to compile.
	if _, err := NewQuery(`(identifier) @name (#eq? @name)`, lang); err == nil {
		t.Fatal("expected error for #eq? with wrong arity, got nil")
	}
}
