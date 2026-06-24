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

func TestGLRMergeTelemetryDisabledByDefault(t *testing.T) {
	t.Setenv("GOT_GLR_MERGE_TELEMETRY", "")
	ResetParseEnvConfigCacheForTests()
	defer ResetParseEnvConfigCacheForTests()

	if got := newGLRMergeTelemetry(&Language{Name: "testlang"}, "unit"); got != nil {
		t.Fatalf("newGLRMergeTelemetry returned non-nil telemetry when disabled")
	}
}
