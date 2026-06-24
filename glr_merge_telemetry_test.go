package gotreesitter

import (
	"bytes"
	"strings"
	"testing"
)

func TestGLRMergeTelemetryEnvAndEmission(t *testing.T) {
	t.Setenv("GOT_GLR_MERGE_TELEMETRY", "1")
	t.Setenv("GOT_GLR_MERGE_TELEMETRY_TOP_KEYS", "2")
	t.Setenv("GOT_GLR_MERGE_TELEMETRY_MIN_SEEN", "3")
	t.Setenv("GOT_GLR_MERGE_TELEMETRY_MAX_EVENTS", "0")
	ResetParseEnvConfigCacheForTests()
	defer ResetParseEnvConfigCacheForTests()

	var buf bytes.Buffer
	prevWriter := glrMergeTelemetryWriter
	glrMergeTelemetryWriter = &buf
	defer func() {
		glrMergeTelemetryWriter = prevWriter
	}()

	stacks := make([]glrStack, 0, 6)
	for i := 0; i < 6; i++ {
		stacks = append(stacks, glrStack{
			entries: []stackEntry{
				{state: StateID(10 + i)},
				{state: 42},
			},
			byteOffset:  128,
			branchOrder: uint64(i),
		})
	}
	scratch := glrMergeScratch{
		perKeyCap: 2,
		telemetry: newGLRMergeTelemetry(&Language{Name: "testlang"}, "unit"),
	}

	out := mergeStacksWithScratch(stacks, &scratch)
	if len(out) != 2 {
		t.Fatalf("mergeStacksWithScratch kept %d stacks, want 2", len(out))
	}

	log := buf.String()
	for _, want := range []string{
		"GLR-MERGE event=summary",
		"lang=testlang",
		"mode=unit",
		"in=6 alive=6 out=2 slots=1 cap=2 max_seen=6 overflow_keys=1",
		"GLR-MERGE event=key",
		"state=42 byte=128 seen=6 kept=2 cap=2",
		"overflow=4",
		"keep_best=acc0/score0/shift0/depth2/byte128",
	} {
		if !strings.Contains(log, want) {
			t.Fatalf("telemetry log missing %q:\n%s", want, log)
		}
	}
}

func TestGLRMergeTelemetryMaxEvents(t *testing.T) {
	t.Setenv("GOT_GLR_MERGE_TELEMETRY", "1")
	t.Setenv("GOT_GLR_MERGE_TELEMETRY_TOP_KEYS", "2")
	t.Setenv("GOT_GLR_MERGE_TELEMETRY_MIN_SEEN", "3")
	t.Setenv("GOT_GLR_MERGE_TELEMETRY_MAX_EVENTS", "1")
	ResetParseEnvConfigCacheForTests()
	defer ResetParseEnvConfigCacheForTests()

	var buf bytes.Buffer
	prevWriter := glrMergeTelemetryWriter
	glrMergeTelemetryWriter = &buf
	defer func() {
		glrMergeTelemetryWriter = prevWriter
	}()

	stacks := make([]glrStack, 0, 6)
	for i := 0; i < 6; i++ {
		stacks = append(stacks, glrStack{
			entries: []stackEntry{
				{state: StateID(10 + i)},
				{state: 42},
			},
			byteOffset:  128,
			branchOrder: uint64(i),
		})
	}
	scratch := glrMergeScratch{
		perKeyCap: 2,
		telemetry: newGLRMergeTelemetry(&Language{Name: "testlang"}, "unit"),
	}

	_ = mergeStacksWithScratch(stacks, &scratch)

	log := buf.String()
	if got := strings.Count(log, "GLR-MERGE "); got != 1 {
		t.Fatalf("emitted %d telemetry lines, want 1:\n%s", got, log)
	}
	if !strings.Contains(log, "event=summary") {
		t.Fatalf("bounded telemetry did not emit summary first:\n%s", log)
	}
	if strings.Contains(log, "event=key") {
		t.Fatalf("bounded telemetry emitted key line despite max events=1:\n%s", log)
	}
}

func TestGLRMergeTelemetrySuppressesSmallSummaryBelowMinSeen(t *testing.T) {
	t.Setenv("GOT_GLR_MERGE_TELEMETRY", "1")
	t.Setenv("GOT_GLR_MERGE_TELEMETRY_MIN_SEEN", "3")
	t.Setenv("GOT_GLR_MERGE_TELEMETRY_MAX_EVENTS", "0")
	ResetParseEnvConfigCacheForTests()
	defer ResetParseEnvConfigCacheForTests()

	var buf bytes.Buffer
	prevWriter := glrMergeTelemetryWriter
	glrMergeTelemetryWriter = &buf
	defer func() {
		glrMergeTelemetryWriter = prevWriter
	}()

	stacks := []glrStack{
		{entries: []stackEntry{{state: 1}}, byteOffset: 8},
		{entries: []stackEntry{{state: 2}}, byteOffset: 16},
	}
	scratch := glrMergeScratch{
		perKeyCap: 2,
		telemetry: newGLRMergeTelemetry(&Language{Name: "testlang"}, "unit"),
	}

	_ = mergeStacksWithScratch(stacks, &scratch)

	if log := buf.String(); log != "" {
		t.Fatalf("small summary below min_seen emitted unexpectedly:\n%s", log)
	}
}

func TestGLRMergeTelemetryRankComparisonIsNumeric(t *testing.T) {
	stat := glrMergeTelemetryKey{}
	lowScore := glrStack{
		entries:    []stackEntry{{state: 1}},
		byteOffset: 10,
		score:      2,
	}
	highScore := glrStack{
		entries:    []stackEntry{{state: 1}},
		byteOffset: 10,
		score:      10,
	}

	stat.considerKeep(&lowScore, 1)
	stat.considerKeep(&highScore, 2)

	best := rankOrDash(stat.keepBest)
	worst := rankOrDash(stat.keepWorst)
	if !strings.Contains(best, "score10") {
		t.Fatalf("numeric best rank = %q, want score10", best)
	}
	if !strings.Contains(worst, "score2") {
		t.Fatalf("numeric worst rank = %q, want score2", worst)
	}
}

func TestGLRMergeTelemetryDisabledByDefault(t *testing.T) {
	t.Setenv("GOT_GLR_MERGE_TELEMETRY", "")
	ResetParseEnvConfigCacheForTests()
	defer ResetParseEnvConfigCacheForTests()

	if got := newGLRMergeTelemetry(&Language{Name: "testlang"}, "unit"); got != nil {
		t.Fatalf("newGLRMergeTelemetry returned non-nil telemetry when disabled")
	}
}
