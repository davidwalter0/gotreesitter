package gotreesitter

import (
	"strings"
	"testing"
)

func TestTransientReduceLanguageDefaultsToDisabled(t *testing.T) {
	t.Setenv("GOT_TRANSIENT_REDUCE_CHILDREN", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_PARENTS", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_LANGS", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_CHILDREN_LANGS", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_PARENTS_LANGS", "")

	if parseTransientReduceChildrenLanguageEnabled(&Language{Name: "python"}) {
		t.Fatal("python transient reduce children enabled by default")
	}
	if parseTransientReduceChildrenLanguageEnabled(&Language{Name: "java"}) {
		t.Fatal("java transient reduce children enabled by default")
	}
	if parseTransientReduceChildrenLanguageEnabled(&Language{Name: "go"}) {
		t.Fatal("go transient reduce children enabled without source-gated default")
	}
	if parseTransientReduceParentsLanguageEnabled(&Language{Name: "go"}) {
		t.Fatal("go transient reduce parents enabled without source-gated default")
	}
}

func TestTransientReduceGoDefaultLargeSourceOnly(t *testing.T) {
	t.Setenv("GOT_TRANSIENT_REDUCE_CHILDREN", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_PARENTS", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_LANGS", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_CHILDREN_LANGS", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_PARENTS_LANGS", "")

	p := &Parser{language: &Language{Name: "go"}}
	small := make([]byte, defaultTransientReduceGoMinSourceLen-1)
	large := make([]byte, defaultTransientReduceGoMinSourceLen)
	if p.shouldUseTransientReduceChildren(small, nil, nil, arenaClassFull) {
		t.Fatal("go transient reduce children enabled below large-source threshold")
	}
	if p.shouldUseTransientReduceParents(small, nil, nil, arenaClassFull) {
		t.Fatal("go transient reduce parents enabled below large-source threshold")
	}
	if !p.shouldUseTransientReduceChildren(large, nil, nil, arenaClassFull) {
		t.Fatal("go transient reduce children disabled at large-source threshold")
	}
	if !p.shouldUseTransientReduceParents(large, nil, nil, arenaClassFull) {
		t.Fatal("go transient reduce parents disabled at large-source threshold")
	}

	p.noResultCompatibilityBenchmarkOnly = true
	if p.shouldUseTransientReduceChildren(large, nil, nil, arenaClassFull) {
		t.Fatal("go transient reduce children enabled in no-result-compat benchmark mode")
	}
	if p.shouldUseTransientReduceParents(large, nil, nil, arenaClassFull) {
		t.Fatal("go transient reduce parents enabled in no-result-compat benchmark mode")
	}
}

func TestTransientReduceLanguageAllowlist(t *testing.T) {
	t.Setenv("GOT_TRANSIENT_REDUCE_CHILDREN", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_PARENTS", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_LANGS", "java, typescript")
	t.Setenv("GOT_TRANSIENT_REDUCE_CHILDREN_LANGS", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_PARENTS_LANGS", "")

	if !parseTransientReduceChildrenLanguageEnabled(&Language{Name: "java"}) {
		t.Fatal("java transient reduce children disabled by allowlist")
	}
	if !parseTransientReduceParentsLanguageEnabled(&Language{Name: "typescript"}) {
		t.Fatal("typescript transient reduce parents disabled by allowlist")
	}
	if parseTransientReduceChildrenLanguageEnabled(&Language{Name: "python"}) {
		t.Fatal("python transient reduce children enabled outside allowlist")
	}
}

func TestTransientReduceGoAllowlistBypassesLargeSourceDefault(t *testing.T) {
	t.Setenv("GOT_TRANSIENT_REDUCE_CHILDREN", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_PARENTS", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_LANGS", "go")
	t.Setenv("GOT_TRANSIENT_REDUCE_CHILDREN_LANGS", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_PARENTS_LANGS", "")

	p := &Parser{language: &Language{Name: "go"}}
	src := []byte("package p\n")
	if !p.shouldUseTransientReduceChildren(src, nil, nil, arenaClassFull) {
		t.Fatal("go transient reduce children explicit allowlist ignored below threshold")
	}
	if !p.shouldUseTransientReduceParents(src, nil, nil, arenaClassFull) {
		t.Fatal("go transient reduce parents explicit allowlist ignored below threshold")
	}
}

func TestTransientReduceLanguageSpecificOverride(t *testing.T) {
	t.Setenv("GOT_TRANSIENT_REDUCE_CHILDREN", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_PARENTS", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_LANGS", "all")
	t.Setenv("GOT_TRANSIENT_REDUCE_CHILDREN_LANGS", "kotlin")
	t.Setenv("GOT_TRANSIENT_REDUCE_PARENTS_LANGS", "none")

	if !parseTransientReduceChildrenLanguageEnabled(&Language{Name: "kotlin"}) {
		t.Fatal("kotlin transient reduce children disabled by specific allowlist")
	}
	if parseTransientReduceChildrenLanguageEnabled(&Language{Name: "java"}) {
		t.Fatal("java transient reduce children enabled despite specific override")
	}
	if parseTransientReduceParentsLanguageEnabled(&Language{Name: "kotlin"}) {
		t.Fatal("kotlin transient reduce parents enabled despite specific none override")
	}
}

func TestTransientReduceLegacyDisable(t *testing.T) {
	t.Setenv("GOT_TRANSIENT_REDUCE_CHILDREN", "")
	t.Setenv("GOT_TRANSIENT_REDUCE_PARENTS", "")
	t.Setenv("GOT_PYTHON_TRANSIENT_REDUCE_CHILDREN", "false")

	if parseTransientReduceChildrenEnabled() {
		t.Fatal("legacy transient reduce disable ignored")
	}
	if parseTransientReduceParentsEnabled() {
		t.Fatal("legacy transient reduce parent disable ignored")
	}
}

func TestTransientReducePathDisable(t *testing.T) {
	t.Setenv("GOT_TRANSIENT_REDUCE_CHILDREN", "0")
	t.Setenv("GOT_TRANSIENT_REDUCE_PARENTS", "false")
	t.Setenv("GOT_PYTHON_TRANSIENT_REDUCE_CHILDREN", "")

	if parseTransientReduceChildrenEnabled() {
		t.Fatal("transient reduce children disable ignored")
	}
	if parseTransientReduceParentsEnabled() {
		t.Fatal("transient reduce parent disable ignored")
	}
}

func TestTransientReduceScratchNoAliasLargeOnly(t *testing.T) {
	if parseShouldUseTransientReduceScratchNoAlias(50 * 1024) {
		t.Fatal("scratch no-alias transient reduce enabled for 50KB input")
	}
	if !parseShouldUseTransientReduceScratchNoAlias(256 * 1024) {
		t.Fatal("scratch no-alias transient reduce disabled at large-file threshold")
	}
}

func TestTransientReduceCheckpointBytes(t *testing.T) {
	t.Setenv("GOT_TRANSIENT_REDUCE_CHECKPOINT_MB", "128")
	if got, want := parseTransientReduceCheckpointBytes(), int64(128<<20); got != want {
		t.Fatalf("parseTransientReduceCheckpointBytes() = %d, want %d", got, want)
	}
	t.Setenv("GOT_TRANSIENT_REDUCE_CHECKPOINT_MB", "off")
	if got := parseTransientReduceCheckpointBytes(); got != 0 {
		t.Fatalf("parseTransientReduceCheckpointBytes() for invalid value = %d, want 0", got)
	}
}

// TestEnvConfigWarningsSurfaceMalformedValues is the P3 #6 regression test:
// a malformed GOT_GLR_MAX_STACKS / GOT_GLR_MAX_MERGE_PER_KEY /
// GOT_PARSE_MEMORY_BUDGET_MB value must no longer vanish silently — it must
// be discoverable through envConfigWarnings().
func TestEnvConfigWarningsSurfaceMalformedValues(t *testing.T) {
	tests := []struct {
		name       string
		envVar     string
		rawValue   string
		wantSubstr string
	}{
		{
			name:       "non-numeric GOT_GLR_MAX_STACKS",
			envVar:     "GOT_GLR_MAX_STACKS",
			rawValue:   "not-a-number",
			wantSubstr: `GOT_GLR_MAX_STACKS="not-a-number"`,
		},
		{
			name:       "non-positive GOT_GLR_MAX_STACKS",
			envVar:     "GOT_GLR_MAX_STACKS",
			rawValue:   "0",
			wantSubstr: `GOT_GLR_MAX_STACKS="0"`,
		},
		{
			name:       "non-numeric GOT_GLR_MAX_MERGE_PER_KEY",
			envVar:     "GOT_GLR_MAX_MERGE_PER_KEY",
			rawValue:   "abc",
			wantSubstr: `GOT_GLR_MAX_MERGE_PER_KEY="abc"`,
		},
		{
			name:       "non-numeric GOT_PARSE_MEMORY_BUDGET_MB",
			envVar:     "GOT_PARSE_MEMORY_BUDGET_MB",
			rawValue:   "not-a-number",
			wantSubstr: `GOT_PARSE_MEMORY_BUDGET_MB="not-a-number"`,
		},
		{
			name:       "negative GOT_PARSE_MEMORY_BUDGET_MB",
			envVar:     "GOT_PARSE_MEMORY_BUDGET_MB",
			rawValue:   "-1",
			wantSubstr: `GOT_PARSE_MEMORY_BUDGET_MB="-1"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ResetParseEnvConfigCacheForTests()
			t.Cleanup(ResetParseEnvConfigCacheForTests)
			t.Setenv("GOT_GLR_MAX_STACKS", "")
			t.Setenv("GOT_GLR_MAX_MERGE_PER_KEY", "")
			t.Setenv("GOT_PARSE_MEMORY_BUDGET_MB", "")
			t.Setenv(tt.envVar, tt.rawValue)

			warnings := envConfigWarnings()
			found := false
			for _, w := range warnings {
				if strings.Contains(w, tt.wantSubstr) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("envConfigWarnings() = %v, want a warning containing %q", warnings, tt.wantSubstr)
			}
		})
	}
}

func TestEnvConfigWarningsEmptyForWellFormedOrUnsetValues(t *testing.T) {
	t.Run("well-formed", func(t *testing.T) {
		ResetParseEnvConfigCacheForTests()
		t.Cleanup(ResetParseEnvConfigCacheForTests)
		t.Setenv("GOT_GLR_MAX_STACKS", "12")
		t.Setenv("GOT_GLR_MAX_MERGE_PER_KEY", "4")
		t.Setenv("GOT_PARSE_MEMORY_BUDGET_MB", "256")

		if warnings := envConfigWarnings(); len(warnings) != 0 {
			t.Fatalf("envConfigWarnings() = %v, want none for well-formed values", warnings)
		}
		if got := parseMaxGLRStacksValue(); got != 12 {
			t.Fatalf("parseMaxGLRStacksValue() = %d, want 12", got)
		}
		if got := parseMaxMergePerKeyValue(); got != 4 {
			t.Fatalf("parseMaxMergePerKeyValue() = %d, want 4", got)
		}
		if got := parseMemoryBudgetMB(); got != 256 {
			t.Fatalf("parseMemoryBudgetMB() = %d, want 256", got)
		}
	})

	t.Run("unset", func(t *testing.T) {
		ResetParseEnvConfigCacheForTests()
		t.Cleanup(ResetParseEnvConfigCacheForTests)
		t.Setenv("GOT_GLR_MAX_STACKS", "")
		t.Setenv("GOT_GLR_MAX_MERGE_PER_KEY", "")
		t.Setenv("GOT_PARSE_MEMORY_BUDGET_MB", "")

		if warnings := envConfigWarnings(); len(warnings) != 0 {
			t.Fatalf("envConfigWarnings() = %v, want none when unset", warnings)
		}
	})
}

// TestMalformedEnvConfigFallsBackToDefault proves the diagnostic is purely
// additive: a malformed override never changes the effective config, only
// whether it's discoverable.
func TestMalformedEnvConfigFallsBackToDefault(t *testing.T) {
	ResetParseEnvConfigCacheForTests()
	t.Cleanup(ResetParseEnvConfigCacheForTests)
	t.Setenv("GOT_GLR_MAX_STACKS", "garbage")
	t.Setenv("GOT_GLR_MAX_MERGE_PER_KEY", "garbage")
	t.Setenv("GOT_PARSE_MEMORY_BUDGET_MB", "garbage")

	if got := parseMaxGLRStacksValue(); got != maxGLRStacks {
		t.Fatalf("parseMaxGLRStacksValue() = %d, want default %d", got, maxGLRStacks)
	}
	if got := parseMaxMergePerKeyValue(); got != maxStacksPerMergeKey {
		t.Fatalf("parseMaxMergePerKeyValue() = %d, want default %d", got, maxStacksPerMergeKey)
	}
	if got := parseMemoryBudgetMB(); got != 512 {
		t.Fatalf("parseMemoryBudgetMB() = %d, want default 512", got)
	}
}

// TestParserLoggerSurfacesMalformedEnvConfig is an end-to-end proof that a
// Parser with a logger attached observes the malformed-env-var diagnostic
// during an ordinary Parse call — the P3 fix for finding #6 (silent env-var
// misconfiguration).
func TestParserLoggerSurfacesMalformedEnvConfig(t *testing.T) {
	ResetParseEnvConfigCacheForTests()
	t.Cleanup(ResetParseEnvConfigCacheForTests)
	t.Setenv("GOT_GLR_MAX_STACKS", "not-a-number")
	t.Setenv("GOT_GLR_MAX_MERGE_PER_KEY", "")
	t.Setenv("GOT_PARSE_MEMORY_BUDGET_MB", "")

	lang := buildArithmeticLanguage()
	parser := NewParser(lang)

	var configEvents []string
	parser.SetLogger(func(kind ParserLogType, msg string) {
		if kind == ParserLogConfig {
			configEvents = append(configEvents, msg)
		}
	})

	if _, err := parser.Parse([]byte("1+2")); err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(configEvents) == 0 {
		t.Fatal("expected at least one ParserLogConfig event for malformed GOT_GLR_MAX_STACKS")
	}
	found := false
	for _, msg := range configEvents {
		if strings.Contains(msg, "GOT_GLR_MAX_STACKS") && strings.Contains(msg, "not-a-number") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("config events = %v, want one naming GOT_GLR_MAX_STACKS=\"not-a-number\"", configEvents)
	}

	// The malformed value must not change the effective config: the default
	// stays in force.
	if got := parseMaxGLRStacksValue(); got != maxGLRStacks {
		t.Fatalf("parseMaxGLRStacksValue() = %d, want default %d", got, maxGLRStacks)
	}
}
