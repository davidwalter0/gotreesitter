package gotreesitter

import (
	"strings"
	"testing"
)

// TestMatchAnchorAfterZeroMatchedQuantifier pins the distinction between two
// situations that an anchor must NOT conflate:
//
//	(parent . (x))            an anchor with no preceding step -- (x) must be
//	                          the FIRST named child.
//	(parent (y)* . (x))       an anchor whose preceding quantified step matched
//	                          ZERO nodes -- there is nothing to be adjacent to,
//	                          so the anchor constrains nothing.
//
// Conflating them makes `(y)*` and `(y)?` behave as if they required at least
// one occurrence, because the anchor silently falls back to "first named
// child". The failure is silent: the query compiles, nothing errors, and the
// pattern simply stops matching wherever the optional part is absent.
//
// This is the shape every upstream tags.scm uses to attach doc comments to a
// definition:
//
//	((comment)* @doc . (function_declaration name: (identifier) @name) @definition.function)
//
// Under the conflation, that finds a function only when the function is the
// first named child of the file -- so in Go, where `package main` comes first,
// it finds nothing on ordinary uncommented code. Measured against real
// grammars before this was fixed: upstream Go's tags.scm returned 5 of 114
// functions.
//
// The cases below are a discriminating pair by construction: the tree is
// program -> identifier "a", identifier "b", number "1", and (string) never
// occurs in it, so `(string)*` always matches zero. Only the target's child
// POSITION varies between the first two cases and the last two, which is
// exactly what a first-child constraint keys on and an adjacency constraint
// does not.
func TestMatchAnchorAfterZeroMatchedQuantifier(t *testing.T) {
	lang := queryTestLanguage()
	tree := buildProgramTreeWithIdentifiers(lang)

	cases := []struct {
		name  string
		query string
		want  []string
	}{
		{
			// (number) is the THIRD named child. With the quantifier matching
			// zero there is no preceding match, so the anchor must not
			// constrain position.
			name:  "zero-matched star leaves the anchor unconstrained",
			query: `(program (string)* @doc . (number) @name)`,
			want:  []string{"1"},
		},
		{
			// Same, with ? rather than * -- both admit zero.
			name:  "zero-matched option leaves the anchor unconstrained",
			query: `(program (string)? @doc . (number) @name)`,
			want:  []string{"1"},
		},
		{
			// REGRESSION GUARD. A pattern-initial anchor has no preceding step
			// at all, and must still mean "first named child". A fix that
			// simply drops the boundary test would wrongly match here.
			name:  "pattern-initial anchor still rejects a non-first child",
			query: `(program . (number) @name)`,
			want:  nil,
		},
		{
			// The other half of the guard: it must still ACCEPT the real first
			// named child, so the test cannot pass by rejecting everything.
			name:  "pattern-initial anchor still accepts the first child",
			query: `(program . (identifier) @name)`,
			want:  []string{"a"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q, err := NewQuery(tc.query, lang)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			var got []string
			for _, m := range q.Execute(tree) {
				for _, c := range m.Captures {
					if c.Name == "name" {
						got = append(got, c.Node.Text(tree.Source()))
					}
				}
			}

			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Errorf("query %s\n  @name captures = %v, want %v",
					tc.query, got, tc.want)
			}
		})
	}
}
