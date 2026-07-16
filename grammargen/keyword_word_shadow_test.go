package grammargen

import "testing"

// TestKeywordWordShadowSuppression is the targeted unit regression for the
// bash ${...} parameter-expansion defect: the keyword-capture `word` token is
// injected into a state's lex mode whenever any keyword is valid there
// (including reduce-follow / missing-recovery WIDENED keywords), but on a
// scanner-heavy LALR-merged grammar that injection can land in a state whose
// real job is to lex a specific pattern terminal (bash's
// _simple_variable_name = /\w+/ inside `${...}`). The broad word then wins
// longest-match and swallows the following operators. With
// SuppressKeywordWordShadow enabled, the injected word is dropped from exactly
// that shape of state (word not a real action, no keyword a real action, a
// real-action pattern terminal present); without it, word stays.
//
// Symbols: 1 = keyword ("if"), 2 = variable_name (/\w+/, pattern terminal),
// 3 = word (keyword-capture). State 1 directly shifts only variable_name(2);
// the keyword(1) is only reduce-follow-widened in, which is what triggers the
// word(3) injection.
func TestKeywordWordShadowSuppression(t *testing.T) {
	const (
		tokenCount = 4
		symKeyword = 1
		symVarName = 2
		symWord    = 3
	)
	actionLookup := func(state, sym int) bool {
		return state == 1 && sym == symVarName // only variable_name is a real action
	}
	followTokens := func(state int) []int {
		if state == 1 {
			return []int{symKeyword} // keyword valid only via reduce-follow widening
		}
		return nil
	}
	run := func(suppress bool) map[int]bool {
		modes, stateToMode, _ := computeLexModes(
			2,          // stateCount
			tokenCount, // tokenCount
			actionLookup,
			nil, // stringPrefixExtensions
			nil, // extraSymbols
			-1,  // extraChainStateStart
			nil, // immediateTokens
			nil, // externalSymbols
			symWord,
			map[int]bool{symKeyword: true}, // keywordSymbols
			map[int]bool{symVarName: true}, // terminalPatternSyms
			followTokens,
			nil,                            // missingRecoveryTokens
			nil,                            // suppressAfterWhitespaceSyms
			map[int]bool{symVarName: true}, // patternTerminals: variable_name is /\w+/
			nil,                            // zeroWidthTerminals
			suppress,
		)
		return modes[stateToMode[1]].validSymbols
	}

	base := run(false)
	if !base[symWord] {
		t.Fatalf("precondition: without suppression, word must be injected into state 1 lex mode; got %v", base)
	}
	if !base[symVarName] {
		t.Fatalf("precondition: variable_name must be valid in state 1; got %v", base)
	}

	fixed := run(true)
	if fixed[symWord] {
		t.Fatalf("with SuppressKeywordWordShadow, keyword-capture word must be dropped from state 1 lex mode; got %v", fixed)
	}
	if !fixed[symVarName] {
		t.Fatalf("with SuppressKeywordWordShadow, the shadowed pattern terminal variable_name must remain valid; got %v", fixed)
	}
}

// TestKeywordWordShadowKeepsWordWithRealAction ensures the suppression does not
// fire when word is a genuine parse action in the state (normal command
// context), so ordinary word/keyword lexing is unaffected.
func TestKeywordWordShadowKeepsRealWordAction(t *testing.T) {
	const (
		tokenCount = 4
		symKeyword = 1
		symVarName = 2
		symWord    = 3
	)
	actionLookup := func(state, sym int) bool {
		// state 1 directly shifts BOTH word and variable_name (command context).
		return state == 1 && (sym == symWord || sym == symVarName)
	}
	modes, stateToMode, _ := computeLexModes(
		2, tokenCount, actionLookup,
		nil, nil, -1, nil, nil, symWord,
		map[int]bool{symKeyword: true},
		map[int]bool{symVarName: true},
		nil, nil, nil,
		map[int]bool{symVarName: true},
		nil,
		true, // suppression enabled
	)
	vs := modes[stateToMode[1]].validSymbols
	if !vs[symWord] {
		t.Fatalf("word with a real parse action must NOT be suppressed; got %v", vs)
	}
}
