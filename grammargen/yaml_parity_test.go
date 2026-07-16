package grammargen

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/davidwalter0/gotreesitter"
	"github.com/davidwalter0/gotreesitter/grammars"
)

func TestYAMLSimpleMappingParity(t *testing.T) {
	genLang, refLang := loadGeneratedYAMLLanguagesForParity(t)
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, "key: value\n")
}

func TestYAMLTagDirectiveSequenceParity(t *testing.T) {
	genLang, refLang := loadGeneratedYAMLLanguagesForParity(t)
	src := "%TAG ! tag:clarkevans.com,2002:\n" +
		"--- !shape\n" +
		"- !circle\n"
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, src)
}

func TestYAMLFoldedBlockScalarParity(t *testing.T) {
	genLang, refLang := loadGeneratedYAMLLanguagesForParity(t)
	src := ">\n" +
		" Sammy Sosa completed another\n" +
		" fine season with great stats.\n" +
		"\n" +
		"   63 Home Runs\n" +
		"   0.288 Batting Average\n" +
		"\n" +
		" What a year!\n"
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, src)
}

func TestYAMLNestedSequenceMappingParity(t *testing.T) {
	genLang, refLang := loadGeneratedYAMLLanguagesForParity(t)
	src := "american:\n" +
		"  - Boston Red Sox\n" +
		"  - Detroit Tigers\n" +
		"  - New York Yankees\n" +
		"national:\n" +
		"  - New York Mets\n" +
		"  - Chicago Cubs\n" +
		"  - Atlanta Braves\n"
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, src)
}

func TestYAMLSequenceOfMappingsParity(t *testing.T) {
	genLang, refLang := loadGeneratedYAMLLanguagesForParity(t)
	src := "-\n" +
		"  name: Mark McGwire\n" +
		"  hr:   65\n" +
		"  avg:  0.278\n" +
		"-\n" +
		"  name: Sammy Sosa\n" +
		"  hr:   63\n" +
		"  avg:  0.288\n"
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, src)
}

func TestYAMLExplicitKeyBlockScalarParity(t *testing.T) {
	genLang, refLang := loadGeneratedYAMLLanguagesForParity(t)
	src := "? explicit key # Empty value\n" +
		"\n" +
		"? |\n" +
		"  block key\n" +
		"\n" +
		": - one # Explicit compact\n" +
		"  - two # block value\n"
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, src)
}

func TestYAMLFlowMappingParity(t *testing.T) {
	genLang, refLang := loadGeneratedYAMLLanguagesForParity(t)
	src := "point: { x: 89, y: 102 }\n"
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, src)
}

func TestYAMLFlowSequenceParity(t *testing.T) {
	genLang, refLang := loadGeneratedYAMLLanguagesForParity(t)
	src := "vals: [ true, false ]\n"
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, src)
}

func TestYAMLQuotedScalarParity(t *testing.T) {
	genLang, refLang := loadGeneratedYAMLLanguagesForParity(t)
	src := "double: \"Quoted\\t\"\n" +
		"single: 'Howdy'\n"
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, src)
}

func TestYAMLExplicitDocumentCommentRangeParity(t *testing.T) {
	genLang, refLang := loadGeneratedYAMLLanguagesForParity(t)
	src := "# Ordered maps are represented as\n" +
		"# A sequence of mappings, with\n" +
		"# each mapping having one key\n" +
		"--- !!omap\n" +
		"- Mark McGwire: 65\n" +
		"- Sammy Sosa: 63\n" +
		"- Ken Griffy: 58\n"
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, src)
}

func TestYAMLLeadingCommentSequenceParity(t *testing.T) {
	genLang, refLang := loadGeneratedYAMLLanguagesForParity(t)
	src := "# Outside flow collection:\n" +
		"- ::vector\n" +
		"- \": - ()\"\n" +
		"- Up, up, and away!\n" +
		"- -123\n" +
		"- http://example.com/foo#bar\n" +
		"# Inside flow collection:\n" +
		"- [ ::vector,\n" +
		"  \": - ()\",\n" +
		"  \"Up, up and away!\",\n" +
		"  -123,\n" +
		"  http://example.com/foo#bar ]\n"
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, src)
}

func TestYAMLBlockScalarSequenceParity(t *testing.T) {
	genLang, refLang := loadGeneratedYAMLLanguagesForParity(t)
	src := "- | # Empty header\n" +
		"\n" +
		" literal\n" +
		"- >1 # Indentation indicator\n" +
		"\n" +
		"  folded\n" +
		"- |+ # Chomping indicator\n" +
		"\n" +
		" keep\n" +
		"\n" +
		"- >1- # Both indicators\n" +
		"\n" +
		"  strip\n"
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, src)
}

// TestYAMLMultiDocumentStreamRangeParity guards against a table-generation
// regression where a freshly imported/generated yaml Language (as opposed to
// the checked-in ts2go blob) mis-lexed the "---" directives-end marker right
// after an implicit top-level document's block_mapping closed. The external
// scanner was never offered the directives-end token as valid at that state
// (yaml carries 100+ external tokens, past the >=24-external heuristic in
// lr.go that otherwise routes table generation to the less context-precise
// LALR-merged builder), so it mis-lexed the remainder of the stream as one
// run-on token. Recovery then collapsed every following document into the
// first document's span: root/document's range ballooned to end-of-file even
// though only the first document's children survived as its own children.
// Fixed via g.PreferPreciseExternalLexStates = true for yaml in
// applyImportGrammarShapeHints (import_grammarjson.go).
func TestYAMLMultiDocumentStreamRangeParity(t *testing.T) {
	genLang, refLang := loadGeneratedYAMLLanguagesForParity(t)
	src := "a: 1\n---\nb: 2\n"
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, src)
}

// TestYAMLMultiDocumentStreamRangeParityLogFile mirrors the upstream
// tree-sitter-yaml examples/log-file.yaml fixture (three "---"-separated
// documents, the first with no leading directives-end marker of its own)
// that surfaced the divergence in TestYAMLMultiDocumentStreamRangeParity
// against a real-world multi-document stream.
func TestYAMLMultiDocumentStreamRangeParityLogFile(t *testing.T) {
	genLang, refLang := loadGeneratedYAMLLanguagesForParity(t)
	src := "---\n" +
		"Time: 2001-11-23 15:01:42 -5\n" +
		"User: ed\n" +
		"Warning:\n" +
		"  This is an error message\n" +
		"  for the log file\n" +
		"---\n" +
		"Time: 2001-11-23 15:02:31 -5\n" +
		"User: ed\n" +
		"Warning:\n" +
		"  A slightly different error\n" +
		"  message.\n" +
		"---\n" +
		"Date: 2001-11-23 15:03:17 -5\n" +
		"User: ed\n" +
		"Fatal:\n" +
		"  Unknown variable \"bar\"\n" +
		"Stack:\n" +
		"  - file: TopClass.py\n" +
		"    line: 23\n" +
		"    code: |\n" +
		"      x = MoreObject(\"345\\n\")\n" +
		"  - file: MoreClass.py\n" +
		"    line: 58\n" +
		"    code: |-\n" +
		"      foo = bar\n"
	assertGeneratedAndReferenceDeepParity(t, genLang, refLang, src)
}

func loadGeneratedYAMLLanguagesForParity(t *testing.T) (*gotreesitter.Language, *gotreesitter.Language) {
	t.Helper()

	source, err := os.ReadFile(yamlGrammarJSONPathForTest(t))
	if err != nil {
		t.Fatalf("read yaml grammar.json: %v", err)
	}
	gram, err := ImportGrammarJSON(source)
	if err != nil {
		t.Fatalf("import yaml grammar: %v", err)
	}
	genLang, err := generateWithTimeout(gram, 90*time.Second)
	if err != nil {
		t.Fatalf("generate yaml language: %v", err)
	}
	refLang := grammars.YamlLanguage()
	adaptExternalScanner(refLang, genLang)
	return genLang, refLang
}

func yamlGrammarJSONPathForTest(t *testing.T) string {
	t.Helper()

	candidates := []string{
		"/tmp/grammar_parity/yaml/src/grammar.json",
		"/tmp/tree-sitter-yaml/src/grammar.json",
		".parity_seed/yaml/src/grammar.json",
		"../.parity_seed/yaml/src/grammar.json",
	}
	globs := []string{
		"/tmp/gotreesitter-parity-*/repos/yaml/src/grammar.json",
	}
	for _, pattern := range globs {
		matches, err := filepath.Glob(pattern)
		if err == nil && len(matches) > 0 {
			candidates = append(candidates, matches...)
		}
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	t.Skip("YAML grammar.json not available")
	return ""
}
