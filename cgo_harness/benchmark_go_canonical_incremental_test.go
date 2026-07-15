//go:build cgo && treesitter_c_parity

package cgoharness

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	gotreesitter "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
	"github.com/odvcencio/gotreesitter/internal/benchfixtures"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

const canonicalGoIncrementalManifestSchema = "canonical-go-incremental-edits-v1"

//go:embed testdata/canonical_go_incremental_edits.json
var canonicalGoIncrementalManifestJSON []byte

type canonicalGoIncrementalManifest struct {
	Schema string                           `json:"schema"`
	Edits  []canonicalGoIncrementalEditSpec `json:"edits"`
}

type canonicalGoIncrementalEditSpec struct {
	Name         string `json:"name"`
	Fixture      string `json:"fixture"`
	StartByte    int    `json:"start_byte"`
	OldEndByte   int    `json:"old_end_byte"`
	OldText      string `json:"old_text"`
	NewText      string `json:"new_text"`
	SourceSHA256 string `json:"source_sha256"`
	EditedSHA256 string `json:"edited_sha256"`
}

type canonicalGoIncrementalCase struct {
	spec     canonicalGoIncrementalEditSpec
	fixture  benchfixtures.LoadedFixture
	edited   []byte
	forward  gotreesitter.InputEdit
	reverse  gotreesitter.InputEdit
	cForward sitter.InputEdit
	cReverse sitter.InputEdit
}

type canonicalGoIncrementalDirection struct {
	name       string
	from       []byte
	to         []byte
	goEdit     gotreesitter.InputEdit
	cEdit      sitter.InputEdit
	targetSHA  string
	targetKind string
}

func TestCanonicalGoIncrementalManifest(t *testing.T) {
	cases := loadCanonicalGoIncrementalCases(t)
	if len(cases) != 4 {
		t.Fatalf("canonical Go incremental cases=%d want=4", len(cases))
	}
}

func TestCanonicalGoIncrementalParity(t *testing.T) {
	if os.Getenv("GTS_CANONICAL_GO_INCREMENTAL") != "1" {
		t.Skip("set GTS_CANONICAL_GO_INCREMENTAL=1 to run locked four-way incremental parity")
	}
	cases := loadCanonicalGoIncrementalCases(t)
	goLang := grammars.GoLanguage()
	cLang := loadCanonicalGoCLanguage(t)
	for _, tc := range cases {
		tc := tc
		t.Run(tc.spec.Name, func(t *testing.T) {
			admitCanonicalGoIncrementalCase(t, tc, goLang, cLang)
		})
	}
}

// BenchmarkParityGoCanonicalIncremental measures only locked edits against the
// authenticated real-Go fixtures. Admission verifies fresh Go=fresh C, Go
// incremental=fresh Go, and C incremental=fresh C in both edit directions
// before either backend timer starts.
func BenchmarkParityGoCanonicalIncremental(b *testing.B) {
	cases := loadCanonicalGoIncrementalCases(b)
	goLang := grammars.GoLanguage()
	cLang := loadCanonicalGoCLanguage(b)
	for _, tc := range cases {
		tc := tc
		b.Run(tc.spec.Name, func(b *testing.B) {
			admitCanonicalGoIncrementalCase(b, tc, goLang, cLang)
			b.Run("gotreesitter", func(b *testing.B) {
				benchmarkCanonicalGoIncremental(b, tc, goLang)
			})
			b.Run("tree-sitter-c", func(b *testing.B) {
				benchmarkCanonicalCIncremental(b, tc, cLang)
			})
		})
	}
}

func loadCanonicalGoIncrementalCases(tb testing.TB) []canonicalGoIncrementalCase {
	tb.Helper()
	decoder := json.NewDecoder(bytes.NewReader(canonicalGoIncrementalManifestJSON))
	decoder.DisallowUnknownFields()
	var manifest canonicalGoIncrementalManifest
	if err := decoder.Decode(&manifest); err != nil {
		tb.Fatalf("decode canonical Go incremental manifest: %v", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		tb.Fatalf("canonical Go incremental manifest has trailing JSON: %v", err)
	}
	if manifest.Schema != canonicalGoIncrementalManifestSchema {
		tb.Fatalf("canonical Go incremental manifest schema=%q want=%q", manifest.Schema, canonicalGoIncrementalManifestSchema)
	}

	fixtures := loadCanonicalGoFixtures(tb)
	fixtureByID := make(map[string]benchfixtures.LoadedFixture, len(fixtures))
	for _, fixture := range fixtures {
		fixtureByID[fixture.Fixture.ID] = fixture
	}
	expected := []struct {
		name    string
		fixture string
	}{
		{name: "token_class_change", fixture: "query_compile"},
		{name: "same_line_length_change", fixture: "language"},
		{name: "early_newline", fixture: "grammargen_lr"},
		{name: "recovery_deletion", fixture: "rewrite"},
	}
	if len(manifest.Edits) != len(expected) {
		tb.Fatalf("canonical Go incremental manifest edits=%d want=%d", len(manifest.Edits), len(expected))
	}

	out := make([]canonicalGoIncrementalCase, 0, len(expected))
	for i, want := range expected {
		spec := manifest.Edits[i]
		if spec.Name != want.name || spec.Fixture != want.fixture {
			tb.Fatalf("canonical Go incremental edit[%d]=%s/%s want=%s/%s", i, spec.Name, spec.Fixture, want.name, want.fixture)
		}
		fixture, ok := fixtureByID[spec.Fixture]
		if !ok {
			tb.Fatalf("canonical Go incremental edit %q names unknown fixture %q", spec.Name, spec.Fixture)
		}
		out = append(out, makeCanonicalGoIncrementalCase(tb, spec, fixture))
	}
	return out
}

func makeCanonicalGoIncrementalCase(tb testing.TB, spec canonicalGoIncrementalEditSpec, fixture benchfixtures.LoadedFixture) canonicalGoIncrementalCase {
	tb.Helper()
	if err := fixture.Fixture.VerifySource(fixture.Source); err != nil {
		tb.Fatalf("canonical Go incremental edit %q source: %v", spec.Name, err)
	}
	observedSourceSHA := sha256Hex(fixture.Source)
	if spec.SourceSHA256 != fixture.Fixture.SHA256 || observedSourceSHA != spec.SourceSHA256 {
		tb.Fatalf("canonical Go incremental edit %q source sha256=%s fixture=%s observed=%s", spec.Name, spec.SourceSHA256, fixture.Fixture.SHA256, observedSourceSHA)
	}
	if spec.StartByte < 0 || spec.OldEndByte < spec.StartByte || spec.OldEndByte > len(fixture.Source) {
		tb.Fatalf("canonical Go incremental edit %q range=[%d,%d) source=%d", spec.Name, spec.StartByte, spec.OldEndByte, len(fixture.Source))
	}
	if got := string(fixture.Source[spec.StartByte:spec.OldEndByte]); got != spec.OldText {
		tb.Fatalf("canonical Go incremental edit %q old text=%q want=%q", spec.Name, got, spec.OldText)
	}
	edited := applyCanonicalGoIncrementalEdit(fixture.Source, spec)
	if got := sha256Hex(edited); got != spec.EditedSHA256 {
		tb.Fatalf("canonical Go incremental edit %q edited sha256=%s want=%s", spec.Name, got, spec.EditedSHA256)
	}
	forward := canonicalGoInputEdit(fixture.Source, edited, spec.StartByte, spec.OldEndByte, spec.StartByte+len(spec.NewText))
	reverse := canonicalGoInputEdit(edited, fixture.Source, spec.StartByte, spec.StartByte+len(spec.NewText), spec.StartByte+len(spec.OldText))
	return canonicalGoIncrementalCase{
		spec:     spec,
		fixture:  fixture,
		edited:   edited,
		forward:  forward,
		reverse:  reverse,
		cForward: realCorpusCInputEdit(forward),
		cReverse: realCorpusCInputEdit(reverse),
	}
}

func applyCanonicalGoIncrementalEdit(source []byte, spec canonicalGoIncrementalEditSpec) []byte {
	out := make([]byte, 0, len(source)-(spec.OldEndByte-spec.StartByte)+len(spec.NewText))
	out = append(out, source[:spec.StartByte]...)
	out = append(out, spec.NewText...)
	out = append(out, source[spec.OldEndByte:]...)
	return out
}

func canonicalGoInputEdit(oldSource, newSource []byte, start, oldEnd, newEnd int) gotreesitter.InputEdit {
	return gotreesitter.InputEdit{
		StartByte:   uint32(start),
		OldEndByte:  uint32(oldEnd),
		NewEndByte:  uint32(newEnd),
		StartPoint:  pointAtOffset(oldSource, start),
		OldEndPoint: pointAtOffset(oldSource, oldEnd),
		NewEndPoint: pointAtOffset(newSource, newEnd),
	}
}

func (tc canonicalGoIncrementalCase) directions() []canonicalGoIncrementalDirection {
	return []canonicalGoIncrementalDirection{
		{
			name:       "forward",
			from:       tc.fixture.Source,
			to:         tc.edited,
			goEdit:     tc.forward,
			cEdit:      tc.cForward,
			targetSHA:  tc.spec.EditedSHA256,
			targetKind: "edited",
		},
		{
			name:       "reverse",
			from:       tc.edited,
			to:         tc.fixture.Source,
			goEdit:     tc.reverse,
			cEdit:      tc.cReverse,
			targetSHA:  tc.spec.SourceSHA256,
			targetKind: "original",
		},
	}
}

func admitCanonicalGoIncrementalCase(tb testing.TB, tc canonicalGoIncrementalCase, goLang *gotreesitter.Language, cLang *sitter.Language) {
	tb.Helper()
	goParser := gotreesitter.NewParser(goLang)
	cParser := sitter.NewParser()
	defer cParser.Close()
	if err := cParser.SetLanguage(cLang); err != nil {
		tb.Fatalf("%s set pinned Go C reference: %v", tc.spec.Name, err)
	}
	for _, direction := range tc.directions() {
		admitCanonicalGoIncrementalDirection(tb, tc.spec.Name, direction, goParser, cParser, goLang)
	}
}

func admitCanonicalGoIncrementalDirection(tb testing.TB, editName string, direction canonicalGoIncrementalDirection, goParser *gotreesitter.Parser, cParser *sitter.Parser, goLang *gotreesitter.Language) {
	tb.Helper()
	label := editName + "/" + direction.name
	if got := sha256Hex(direction.to); got != direction.targetSHA {
		tb.Fatalf("%s %s target sha256=%s want=%s", label, direction.targetKind, got, direction.targetSHA)
	}

	goFresh, err := goParser.Parse(direction.to)
	requireCanonicalGoIncrementalTree(tb, goFresh, direction.to, label+" fresh Go", err)
	defer releaseCanonicalGoTree(goFresh)
	cFresh := cParser.Parse(direction.to, nil)
	requireCanonicalCIncrementalTree(tb, cFresh, direction.to, label+" fresh C")
	defer closeCanonicalCTree(cFresh)
	if diff := FirstDivergenceDumpV1(goFresh.RootNode(), goLang, cFresh.RootNode()); diff != nil {
		tb.Fatalf("%s fresh Go != fresh C: %s", label, formatRealCorpusDivergence(diff))
	}

	goFreshDigest := canonicalGoTreeDigest(tb, goFresh, goLang, label+" fresh Go")
	cFreshDigest := canonicalCTreeDigest(tb, cFresh, label+" fresh C")
	if goFreshDigest != cFreshDigest {
		tb.Fatalf("%s fresh deep digest Go=%s C=%s", label, goFreshDigest, cFreshDigest)
	}

	cOld := cParser.Parse(direction.from, nil)
	requireCanonicalCIncrementalTree(tb, cOld, direction.from, label+" old C")
	cOld.Edit(&direction.cEdit)
	cIncremental := cParser.Parse(direction.to, cOld)
	cOld.Close()
	requireCanonicalCIncrementalTree(tb, cIncremental, direction.to, label+" incremental C")
	defer cIncremental.Close()
	if got := canonicalCTreeDigest(tb, cIncremental, label+" incremental C"); got != cFreshDigest {
		tb.Fatalf("%s incremental C != fresh C: digest=%s want=%s", label, got, cFreshDigest)
	}

	goOld, err := goParser.Parse(direction.from)
	requireCanonicalGoIncrementalTree(tb, goOld, direction.from, label+" old Go", err)
	goOld.Edit(direction.goEdit)
	goIncremental, _, err := goParser.ParseIncrementalProfiled(direction.to, goOld)
	releaseCanonicalGoTree(goOld)
	requireCanonicalGoIncrementalTree(tb, goIncremental, direction.to, label+" incremental Go", err)
	defer releaseCanonicalGoTree(goIncremental)
	if got := canonicalGoTreeDigest(tb, goIncremental, goLang, label+" incremental Go"); got != goFreshDigest {
		var errs []string
		compareGoNodes(goIncremental.RootNode(), goLang, goFresh.RootNode(), "root", &errs)
		tb.Fatalf("%s incremental Go != fresh Go: digest=%s want=%s first=%s", label, got, goFreshDigest, firstRealCorpusBenchmarkError(errs))
	}
}

func benchmarkCanonicalGoIncremental(b *testing.B, tc canonicalGoIncrementalCase, lang *gotreesitter.Language) {
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(tc.fixture.Source)
	requireCanonicalGoIncrementalTree(b, tree, tc.fixture.Source, tc.spec.Name+" initial Go", err)
	defer func() { releaseCanonicalGoTree(tree) }()

	b.ReportAllocs()
	b.SetBytes(int64((len(tc.fixture.Source) + len(tc.edited)) / 2))
	b.ResetTimer()
	var totals realCorpusIncrementalProfileTotals
	edited := false
	for i := 0; i < b.N; i++ {
		target := tc.edited
		edit := tc.forward
		if edited {
			target = tc.fixture.Source
			edit = tc.reverse
		}
		editStart := time.Now()
		tree.Edit(edit)
		totals.addEdit(time.Since(editStart))
		oldTree := tree
		parseStart := time.Now()
		newTree, profile, err := parser.ParseIncrementalProfiled(target, oldTree)
		totals.addParseWall(time.Since(parseStart))
		requireCanonicalGoIncrementalTree(b, newTree, target, tc.spec.Name+" timed Go", err)
		totals.add(profile)
		if newTree != oldTree {
			oldTree.Release()
		}
		tree = newTree
		edited = !edited
	}
	b.StopTimer()
	totals.report(b, b.N)
}

func benchmarkCanonicalCIncremental(b *testing.B, tc canonicalGoIncrementalCase, lang *sitter.Language) {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(lang); err != nil {
		b.Fatalf("%s set pinned Go C reference: %v", tc.spec.Name, err)
	}
	tree := parser.Parse(tc.fixture.Source, nil)
	requireCanonicalCIncrementalTree(b, tree, tc.fixture.Source, tc.spec.Name+" initial C")
	defer func() { tree.Close() }()

	b.ReportAllocs()
	b.SetBytes(int64((len(tc.fixture.Source) + len(tc.edited)) / 2))
	b.ResetTimer()
	var editNanos int64
	var parseNanos int64
	edited := false
	for i := 0; i < b.N; i++ {
		target := tc.edited
		edit := tc.cForward
		if edited {
			target = tc.fixture.Source
			edit = tc.cReverse
		}
		editStart := time.Now()
		tree.Edit(&edit)
		editNanos += time.Since(editStart).Nanoseconds()
		oldTree := tree
		parseStart := time.Now()
		newTree := parser.Parse(target, oldTree)
		parseNanos += time.Since(parseStart).Nanoseconds()
		requireCanonicalCIncrementalTree(b, newTree, target, tc.spec.Name+" timed C")
		if newTree != oldTree {
			oldTree.Close()
		}
		tree = newTree
		edited = !edited
	}
	b.StopTimer()
	if b.N > 0 {
		b.ReportMetric(float64(editNanos)/float64(b.N), "edit_ns/op")
		b.ReportMetric(float64(parseNanos)/float64(b.N), "parse_wall_ns/op")
	}
}

func requireCanonicalGoIncrementalTree(tb testing.TB, tree *gotreesitter.Tree, source []byte, phase string, err error) {
	tb.Helper()
	if err != nil {
		releaseCanonicalGoTree(tree)
		tb.Fatalf("%s: %v", phase, err)
	}
	if tree == nil || tree.RootNode() == nil {
		tb.Fatalf("%s returned nil tree or root", phase)
	}
	root := tree.RootNode()
	if root.StartByte() != 0 || root.EndByte() != uint32(len(source)) || tree.ParseStoppedEarly() {
		tb.Fatalf("%s root=%d..%d want=0..%d stopped=%v (%s)", phase, root.StartByte(), root.EndByte(), len(source), tree.ParseStoppedEarly(), tree.ParseRuntime().Summary())
	}
}

func requireCanonicalCIncrementalTree(tb testing.TB, tree *sitter.Tree, source []byte, phase string) {
	tb.Helper()
	if tree == nil || tree.RootNode() == nil {
		closeCanonicalCTree(tree)
		tb.Fatalf("%s returned nil tree or root", phase)
	}
	root := tree.RootNode()
	if root.StartByte() != 0 || root.EndByte() != uint(len(source)) {
		closeCanonicalCTree(tree)
		tb.Fatalf("%s root=%d..%d want=0..%d", phase, root.StartByte(), root.EndByte(), len(source))
	}
}

func canonicalGoTreeDigest(tb testing.TB, tree *gotreesitter.Tree, lang *gotreesitter.Language, phase string) string {
	tb.Helper()
	inspection, err := benchfixtures.InspectGoTree(tree.RootNode(), lang)
	if err != nil {
		tb.Fatalf("%s deep digest: %v", phase, err)
	}
	return inspection.SHA256
}

func canonicalCTreeDigest(tb testing.TB, tree *sitter.Tree, phase string) string {
	tb.Helper()
	inspection, err := canonicalCTreeInspection(tree.RootNode())
	if err != nil {
		tb.Fatalf("%s deep digest: %v", phase, err)
	}
	return inspection.SHA256
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
