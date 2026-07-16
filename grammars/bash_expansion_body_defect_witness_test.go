package grammars

import (
	"testing"

	ts "github.com/davidwalter0/gotreesitter"
)

// Round-9 (2026-07-16) negative-result witness for the bash BLOB path
// (BashLanguage()). The round-9 task hypothesized that bash's residual
// blob-parity failures (43/55) were the same GLR merge-selection class the
// round-8 cpp fix cleared with a retry-rung merge-per-key widening — i.e. the
// steady-state merge-per-key cap pruning the alternative the tree-sitter-bash
// oracle selects for `conflicts: [$.command, $.variable_assignments]`.
//
// That hypothesis was DISPROVEN empirically:
//
//   - bash's steady-state merge-per-key cap is already 6 (the global
//     maxStacksPerMergeKey default), NOT the tight 1 cpp uses. There is no
//     tight cap to relax.
//   - corpus_parity (blob vs pinned tree-sitter-bash @ a06c2e44) holds at
//     exactly 43/55 across every merge-per-key cap tested (GOT_GLR_MAX_MERGE_PER_KEY
//     = 1, 2, 4, 16, 32) AND every stack cap (GOT_GLR_MAX_STACKS = 2, 48, 256).
//     The pass count does not move by one file: no ERROR cleared, no shape
//     flipped.
//   - A cpp-identical bash retry-rung (temporary experiment: return
//     cppFullParseRetryMaxMergePerKey for a bash complete accept-with-error
//     tree) fired its widened retry pass (mergePerKey=16) and still produced
//     hasErr=true. preferRetryTree would keep the original — a pure no-op with
//     a wasted second parse. It was reverted.
//
// The five blob accept-with-error files are NOT merge-selection defects. Their
// triggers reduce to the parameter-expansion-body table defect (the
// `${...:+...}` / substring `${var:off:len}` operator/suffix alternatives have
// no valid parse path once the replacement carries a `$`-expansion + `;`) and a
// zsh anonymous-function construct — the same class as the grammargen-lane
// witness TestBashParameterExpansionOperatorReducer and
// hypha://m31labs/gotreesitter/object/spec.bash-expansion-table-defect.rca.
// No survivor budget can preserve a branch the LR automaton never generates, so
// this belongs to the d6444430 GLR/table-core effort, not the retry-rung lever.
//
// The controls below must stay clean (regression guards). The known-defect
// reducers use the repo's self-healing skip pattern: they assert the CORRECT
// (error-free) outcome and t.Skip today; when the underlying table defect is
// fixed they start passing with no edit.
func TestBashExpansionBodyBlobDefectWitness(t *testing.T) {
	lang := BashLanguage()
	entry := DetectLanguage("witness.sh")
	if entry == nil {
		t.Fatal("bash entry not detected")
	}

	parse := func(src string) *ts.Node {
		p := ts.NewParser(lang)
		var tree *ts.Tree
		var err error
		if entry.TokenSourceFactory != nil {
			tree, err = p.ParseWithTokenSource([]byte(src), entry.TokenSourceFactory([]byte(src), lang))
		} else {
			tree, err = p.Parse([]byte(src))
		}
		if err != nil {
			t.Fatalf("parse %q: %v", src, err)
		}
		t.Cleanup(tree.Release)
		return tree.RootNode()
	}

	// Controls: these must NOT regress. They are the isolated
	// command-vs-variable_assignments shapes the round-9 hypothesis blamed;
	// each parses clean, so the residual is not that conflict.
	controls := map[string]string{
		"assign_then_keyword":  "migrated=\"a b c\"\nif foo; then :; fi\n",
		"assign_then_function": "migrated=\"a b c\"\n\nis_migrated() {\n  echo x\n}\n",
		"dblbracket_gt":        "if [[ \"$a\" > \"$b\" ]]; then :; fi\n",
		"redirect_then_pipe":   "echo \"$m\" < /dev/null | grep -q x\n",
		"expansion_plus_lit":   "x=\"${y:+z}\"\n",
	}
	for name, src := range controls {
		t.Run("control_"+name, func(t *testing.T) {
			if root := parse(src); root == nil || root.HasError() {
				var sx string
				if root != nil {
					sx = root.SExpr(lang)
				}
				t.Fatalf("control regressed to ERROR for %q: %s", src, sx)
			}
		})
	}

	// Known-defect reducers (self-healing): the parameter-expansion-body table
	// defect on the blob lane. Confirmed NOT cleared by any merge-per-key or
	// stack cap. Assert the correct error-free outcome; skip while it holds.
	defects := map[string]string{
		"expansion_plus_expansion_semicolon": "x=\"${y:+$y; }\"\n",
		"substring_arith_expansion":          "echo ${sp:i++%${#sp}:1}\n",
	}
	for name, src := range defects {
		t.Run("defect_"+name, func(t *testing.T) {
			root := parse(src)
			if root != nil && !root.HasError() {
				return // table defect fixed — pin now green.
			}
			t.Skipf("KNOWN DIVERGENCE (bash expansion-body table defect, blob lane): %q parses to ERROR; "+
				"NOT a merge-selection defect (merge-per-key/stack cap sweep is inert). See "+
				"spec.bash-expansion-table-defect.rca / d6444430.", src)
		})
	}
}
