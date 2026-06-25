package gotreesitter

import "sort"

type ForestBlockCommentRepeatConflictHitForTest struct {
	Lang       string
	State      StateID
	Lookahead  string
	Reductions []string
	ShiftState StateID
}

func ForestBlockCommentRepeatConflictHitsForTest(lang *Language) []ForestBlockCommentRepeatConflictHitForTest {
	if lang == nil {
		return nil
	}
	p := NewParser(lang)
	stateCount := int(lang.StateCount)
	if stateCount == 0 {
		stateCount = len(lang.ParseTable) + len(lang.SmallParseTableMap)
	}

	var hits []ForestBlockCommentRepeatConflictHitForTest
	for state := 0; state < stateCount; state++ {
		p.forEachActionIndexInState(StateID(state), func(sym Symbol, idx uint16) bool {
			if int(idx) >= len(lang.ParseActions) {
				return true
			}
			actions := p.actionsForParseState(StateID(state), sym, lang.ParseActions)
			chosen, ok := forestBlockCommentRepeatShiftConflictChoice(lang, Token{Symbol: sym}, actions)
			if !ok {
				return true
			}

			var reductions []string
			for _, action := range actions {
				if action.Type == ParseActionReduce {
					reductions = append(reductions, symName(lang, action.Symbol))
				}
			}
			sort.Strings(reductions)
			hits = append(hits, ForestBlockCommentRepeatConflictHitForTest{
				Lang:       lang.Name,
				State:      StateID(state),
				Lookahead:  symName(lang, sym),
				Reductions: reductions,
				ShiftState: chosen.State,
			})
			return true
		})
	}
	return hits
}
