package gotreesitter_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

func TestForestBlockCommentRepeatShiftConflictChoiceEmbeddedGrammarBlastRadius(t *testing.T) {
	expected := map[string]bool{
		"ledger state=185 lookahead=block_comment_token1 reductions=[block_comment_repeat1] shift=477":    true,
		"ledger state=188 lookahead=block_comment_token1 reductions=[block_comment_repeat1] shift=491":    true,
		"racket state=223 lookahead=block_comment_token1 reductions=[block_comment_repeat1] shift=223":    true,
		"scala state=17537 lookahead=block_comment_token1 reductions=[block_comment_repeat1] shift=18979": true,
		"scheme state=129 lookahead=block_comment_token1 reductions=[block_comment_repeat1] shift=129":    true,
		"v state=4230 lookahead=block_comment_token1 reductions=[block_comment_repeat1] shift=4336":       true,
	}

	var hits []gotreesitter.ForestBlockCommentRepeatConflictHitForTest
	for _, entry := range grammars.AllLanguages() {
		lang := entry.Language()
		if lang == nil {
			t.Fatalf("%s: language loader returned nil", entry.Name)
		}
		hits = append(hits, gotreesitter.ForestBlockCommentRepeatConflictHitsForTest(lang)...)
	}

	sort.Slice(hits, func(i, j int) bool {
		if hits[i].Lang != hits[j].Lang {
			return hits[i].Lang < hits[j].Lang
		}
		if hits[i].State != hits[j].State {
			return hits[i].State < hits[j].State
		}
		return hits[i].Lookahead < hits[j].Lookahead
	})

	seen := make(map[string]bool, len(hits))
	var unexpected []string
	for _, h := range hits {
		key := fmt.Sprintf("%s state=%d lookahead=%s reductions=%v shift=%d", h.Lang, h.State, h.Lookahead, h.Reductions, h.ShiftState)
		seen[key] = true
		if !expected[key] {
			unexpected = append(unexpected, key)
		}
	}
	if len(unexpected) > 0 {
		t.Fatalf("unexpected forest block-comment repetition conflict hits:\n%s", strings.Join(unexpected, "\n"))
	}
	var missing []string
	for key := range expected {
		if !seen[key] {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("missing expected forest block-comment repetition conflict hits:\n%s", strings.Join(missing, "\n"))
	}
}
