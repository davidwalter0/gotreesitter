# gotreesitter (davidwalter0 fork) — build/test entrypoint.
#
# Fork of github.com/odvcencio/gotreesitter, evolving toward a feature-complete,
# correct, performant PURE-GO tree-sitter. Plan:
# ~/.claude/plans/gotreesitter-fork-evolution-program.md
#
# OOM DISCIPLINE (from AGENTS.md, non-negotiable): NEVER run repo-wide
# `go test ./...` or `-race` on a dev host — it OOMs. Heavy correctness/parity
# work goes through Docker isolation, one language at a time (see parity-*).
# `make test` here runs only the OOM-safe core packages, serialized.

MODULE      := github.com/davidwalter0/gotreesitter
BIN         := bin
COV_OUT     := coverage.out
# Core packages safe to test on the host (no per-language corpus blowup).
CORE_PKGS   := . ./grammargen ./taproot ./taproot/walk ./grep
GOFLAGS     ?=

.PHONY: all build vet lint test test-core cov cov-html tidy \
        parity-lang parity-docker bench eval clean help

all: vet lint test ## vet + lint + core tests

build: ## Pure-Go build of every package (CGO_ENABLED=0 is the point of this fork)
	CGO_ENABLED=0 go build $(GOFLAGS) ./...

vet: ## go vet (does not OOM)
	go vet ./...

lint: ## golangci-lint
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; \
	else echo "golangci-lint not installed; skipping"; fi

test: test-core ## alias: OOM-safe core tests only

test-core: ## Core-package tests, serialized (-p 1), no -race — host-safe
	go test $(GOFLAGS) -p 1 -count=1 $(CORE_PKGS)

# Per-language parity/corpus tests MUST be Docker-isolated, one language at a
# time (host OOM). Delegates to the upstream cgo_harness scripts we inherited.
parity-lang: ## LANG=<grammar> single-grammar parity in Docker
	@test -n "$(LANG)" || { echo "usage: make parity-lang LANG=python"; exit 2; }
	bash cgo_harness/docker/run_single_grammar_parity.sh $(LANG)

parity-docker: ## The CI cgo-parity smoke suite in Docker
	bash cgo_harness/docker/run_parity_in_docker.sh

bench: ## The AGENTS.md perf trio (GOMAXPROCS=1, stable)
	GOMAXPROCS=1 go test -run '^$$' -bench 'BenchmarkGoParse(FullDFA|IncrementalSingleByteEditDFA|IncrementalNoEditDFA)' \
		-count=10 -benchtime=750ms -benchmem .

# The 3-way conformance harness (this fork vs official vs smacker) lives in the
# mcp-agent-editor eval module (referee kept independent of the contestant):
#   mcp-agent-editor/.worktree/analysis/threeway-scanner/eval/official-treesitter-eval
# Point its gotreesitter arm at this fork and run `make eval` there.
eval: ## (see comment) run the external 3-way harness against this fork
	@echo "3-way harness is external: mcp-agent-editor eval/official-treesitter-eval (make eval there)"

cov: ## Per-package coverage table (core pkgs)
	@go test -coverprofile=$(COV_OUT) -p 1 $(CORE_PKGS) 2>&1 | \
		grep -E 'coverage:|no test files' | \
		sed 's|.*$(MODULE)/||; s/\t/ /g' | \
		awk '{ pkg=$$1; cov="0.0%"; for(i=1;i<=NF;i++) if($$i ~ /%/) cov=$$i; printf "%-45s %s\n", pkg, cov }' | \
		sort | \
		(echo ""; printf "%-45s %s\n" "Package" "Coverage"; \
		 printf "%-45s %s\n" "─────────────────────────────────────────────" "────────"; \
		 cat; \
		 printf "%-45s %s\n" "─────────────────────────────────────────────" "────────"; \
		 printf "%-45s %s\n" "TOTAL" "$$(go tool cover -func=$(COV_OUT) | tail -1 | awk '{print $$NF}')")

cov-html: ## HTML coverage
	go test -coverprofile=$(COV_OUT) -p 1 $(CORE_PKGS) && go tool cover -html=$(COV_OUT)

tidy: ## go mod tidy
	go mod tidy

clean: ## remove build/cov artifacts
	rm -rf $(BIN) $(COV_OUT)

help: ## list targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'
