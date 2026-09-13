package grammargen

import "testing"

func TestPrecOrderComparesRepeatedSymbolInSharedLevel(t *testing.T) {
	table := buildPrecOrderTable([][]PrecEntry{
		{
			{IsSymbol: true, Name: "type_query"},
			{IsSymbol: true, Name: "subscript_expression"},
		},
		{
			{IsSymbol: true, Name: "type_query"},
			{IsSymbol: true, Name: "_type_query_subscript_expression"},
		},
	}, nil)

	if got := table.resolveSymbolVsSymbol("_type_query_subscript_expression", "type_query"); got >= 0 {
		t.Fatalf("_type_query_subscript_expression vs type_query = %d, want type_query to outrank in shared level", got)
	}
	if got := table.resolveSymbolVsSymbol("type_query", "_type_query_subscript_expression"); got <= 0 {
		t.Fatalf("type_query vs _type_query_subscript_expression = %d, want type_query to outrank in shared level", got)
	}
}

// TestPrecOrderComparesNamedPrecInSharedLevel pins the TypeScript unary/call
// precedence fix: "call" and "unary_void" co-occur in the grammar's main
// expression-precedence level (member, template_call, call,
// update_expression, unary_void, binary_exp, ...), with "call" listed
// earlier — i.e. higher precedence — so a shift toward call_expression must
// win a conflict against a reduce tagged PrecLeft("unary_void").
func TestPrecOrderComparesNamedPrecInSharedLevel(t *testing.T) {
	namedPrecs := map[string]int{"call": 22, "unary_void": 21}
	table := buildPrecOrderTable([][]PrecEntry{
		{
			{Name: "member"},
			{Name: "template_call"},
			{Name: "call"},
			{IsSymbol: true, Name: "update_expression"},
			{Name: "unary_void"},
			{Name: "binary_exp"},
		},
	}, namedPrecs)

	if got := table.resolveNamedPrecVsNamedPrec(namedPrecs["call"], namedPrecs["unary_void"]); got <= 0 {
		t.Fatalf("call vs unary_void = %d, want call to outrank (earlier in shared level)", got)
	}
	if got := table.resolveNamedPrecVsNamedPrec(namedPrecs["unary_void"], namedPrecs["call"]); got >= 0 {
		t.Fatalf("unary_void vs call = %d, want call to outrank (earlier in shared level)", got)
	}
}

// TestPrecOrderDoesNotCompareNamedPrecAcrossUnrelatedLevels guards against
// the regression this fix specifically had to avoid: two named precedence
// values that never co-occur in any single declared precedence level (e.g.
// TypeScript's short dedicated [call, instantiation, unary, binary, ...]
// level for one declared conflicts group, versus the main expression-
// precedence level a specific binary operator's name lives in) must stay
// incomparable, even though buildPrecOrderTable assigns every named entry
// some global position. A naive raw-position comparison would silently rank
// "instantiation" against an unrelated binary operator's precedence name and
// wrongly force TypeScript's `a<T>() < b` generic-vs-comparison ambiguity out
// of genuine GLR.
func TestPrecOrderDoesNotCompareNamedPrecAcrossUnrelatedLevels(t *testing.T) {
	namedPrecs := map[string]int{"binary_relation": 100, "instantiation": 5}
	table := buildPrecOrderTable([][]PrecEntry{
		{
			{Name: "binary_relation"},
		},
		{
			{Name: "call"},
			{Name: "instantiation"},
			{Name: "unary"},
			{Name: "binary"},
		},
	}, namedPrecs)

	if got := table.resolveNamedPrecVsNamedPrec(namedPrecs["instantiation"], namedPrecs["binary_relation"]); got != 0 {
		t.Fatalf("instantiation vs binary_relation = %d, want 0 (no shared level, not comparable)", got)
	}
	if got := table.resolveNamedPrecVsNamedPrec(namedPrecs["binary_relation"], namedPrecs["instantiation"]); got != 0 {
		t.Fatalf("binary_relation vs instantiation = %d, want 0 (no shared level, not comparable)", got)
	}
}

func TestPrecOrderNamedPrecEqualValueIsNotComparable(t *testing.T) {
	table := buildPrecOrderTable([][]PrecEntry{
		{{Name: "call"}, {Name: "unary_void"}},
	}, map[string]int{"call": 1, "unary_void": 0})

	if got := table.resolveNamedPrecVsNamedPrec(1, 1); got != 0 {
		t.Fatalf("equal named prec values = %d, want 0", got)
	}
	if got := (*precOrderTable)(nil).resolveNamedPrecVsNamedPrec(1, 0); got != 0 {
		t.Fatalf("nil table = %d, want 0", got)
	}
}
