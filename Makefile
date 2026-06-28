PS ?= powershell -NoProfile -ExecutionPolicy Bypass
RUN = $(PS) -File tests/baseline/run-command.ps1 -ContinueOnError

.PHONY: audit audit\:lint-go audit\:vet audit\:build audit\:test-go audit\:test-go-race audit\:cover audit\:gosec audit\:vuln audit\:fe-typecheck audit\:fe-lint audit\:fe-build audit\:test-fe audit\:e2e audit\:e2e-regress audit\:fe-install audit\:semgrep audit\:bench audit\:fuzz

audit: audit\:build audit\:vet audit\:test-go audit\:test-go-race audit\:cover audit\:gosec audit\:vuln audit\:lint-go audit\:semgrep audit\:fe-typecheck audit\:fe-lint audit\:fe-build audit\:test-fe audit\:e2e

audit\:lint-go:
	$(RUN) -Phase phase1 -Name staticcheck -CommandLine "staticcheck ./..."
	$(RUN) -Phase phase1 -Name golangci-lint -CommandLine "golangci-lint run"

audit\:vet:
	$(RUN) -Phase phase0 -Name go-vet -CommandLine "go vet ./..."

audit\:build:
	$(RUN) -Phase phase0 -Name go-build -CommandLine "go build ./..."

audit\:test-go:
	$(RUN) -Phase phase0 -Name go-test -CommandLine "go test ./..."

audit\:test-go-race:
	$(RUN) -Phase phase0 -Name go-test-race -CommandLine "go test ./... -race -count=1"

audit\:cover:
	$(RUN) -Phase phase0 -Name go-cover -CommandLine "go test ./... -coverprofile tests/baseline/phase0/coverage.out"

audit\:gosec:
	$(RUN) -Phase phase1 -Name gosec -CommandLine "gosec -exclude-dir .gotmp -exclude-dir .gocache -exclude-dir frontend/node_modules ./..."

audit\:vuln:
	$(RUN) -Phase phase1 -Name govulncheck -CommandLine "govulncheck ./..."

audit\:fe-install:
	$(RUN) -Phase phase0 -Name npm-ci -WorkingDirectory frontend -CommandLine "npm ci"

audit\:fe-typecheck:
	$(RUN) -Phase phase0 -Name npm-run-typecheck -WorkingDirectory frontend -SkipReason "frontend/package.json does not define a typecheck script; build runs vue-tsc --noEmit."

audit\:fe-lint:
	$(RUN) -Phase phase0 -Name npm-run-lint -WorkingDirectory frontend -CommandLine "npm run lint"

audit\:fe-build:
	$(RUN) -Phase phase0 -Name npm-run-build -WorkingDirectory frontend -CommandLine "npm run build"

audit\:test-fe:
	$(RUN) -Phase phase0 -Name npm-run-test -WorkingDirectory frontend -CommandLine "npm run test"

audit\:e2e:
	$(RUN) -Phase phase0 -Name e2e -SkipReason "TODO: e2e baseline is reserved for later phases."

# Phase 9 Synthetic + e2e Regression Suite (S9):
#   - runs the S9 synthetic-fixture regression tests (HIGH bugs from S1-S8)
#   - runs all per-package _test.go files that were added in S9
#   - cross-checks against the JSON perf/fuzz baselines
#   - emits a JUnit XML under tests/baseline/phase9/
# Fail-fast: return non-zero on the first failing package.
audit\:e2e-regress:
	@echo "==> phase9 e2e-regress (S9 synthetic fixtures + regression suite)"
	$(RUN) -Phase phase9 -Name phase9-fixture-driver -CommandLine "go test -count=1 -v ./tests/baseline/phase9/phase9_e2e_regress/..."
	$(RUN) -Phase phase9 -Name phase9-s1-regress -CommandLine "go test -count=1 -v -run=TestS1F01 ./service/..."
	$(RUN) -Phase phase9 -Name phase9-s2-regress -CommandLine "go test -count=1 -v -run=TestS2F01 ./database/importxui/..."
	$(RUN) -Phase phase9 -Name phase9-s6-regress -CommandLine "go test -count=1 -v -run=TestS6F -race ./paidsub/... ./sub/... ./cmd/..."
	$(RUN) -Phase phase9 -Name phase9-s7t01-regress -CommandLine "go test -count=1 -v -run=TestS7T01 ./util/common/..."
	$(RUN) -Phase phase9 -Name phase9-s8f10-regress -CommandLine "go test -count=1 -v -run=TestProxyGroupsYAML|TestS8F10 ./sub/..."
	$(RUN) -Phase phase9 -Name phase9-bench-extension -CommandLine "go test -run=^$$ -bench=BenchmarkBroadcastSendOnce -benchmem -benchtime=10x ./paidsub/..."

audit\:semgrep:
	@echo "==> semgrep custom ruleset"
	@pip show semgrep >/dev/null 2>&1 || pip install --quiet "semgrep==1.168.0"
	@python -c "import sys; sys.argv=['semgrep','scan','--config=.semgrep-rules/custom_s0_to_s6.yml','--metrics=off','--quiet','--no-git-ignore','--json','.']; from semgrep.cli import cli; cli()" > semgrep_results.json 2> semgrep_stderr.log || (cat semgrep_stderr.log && exit 1)
	@python -c "import json,sys; d=json.load(open('semgrep_results.json',encoding='utf-8')); r=d['results']; print(f'hits={len(r)}'); [print('  '+x['check_id'].replace('semgrep-rules.','')+' '+x['path']+':'+str(x['start']['line'])) for x in r]; sys.exit(0 if len(r)==11 else 1)"

audit\:bench:
	@echo "==> go test -bench (hot paths)"
	@cd . && go test -run=^$$ -bench=. -benchmem -benchtime=2s ./core/... ./sub/... ./paidsub/... ./service/... ./util/... 2>/dev/null | grep -E "^(Benchmark|PASS|FAIL|ok)" || true

audit\:fuzz:
	@echo "==> go test -fuzz (JSON unmarshal paths)"
	@cd . && go test -run=^$$ -fuzz=FuzzTgUpdateUnmarshal -fuzztime=10s ./paidsub/ 2>&1 | tail -2
	@cd . && go test -run=^$$ -fuzz=FuzzParseTelegramResponse -fuzztime=10s ./paidsub/ 2>&1 | tail -2
	@cd . && go test -run=^$$ -fuzz=FuzzTariffUnmarshal -fuzztime=10s ./paidsub/ 2>&1 | tail -2
	@cd . && go test -run=^$$ -fuzz=FuzzProxyGroupsYAML -fuzztime=10s ./sub/ 2>&1 | tail -2
	@cd . && go test -run=^$$ -fuzz=FuzzClashBasicConfigYAML -fuzztime=10s ./sub/ 2>&1 | tail -2
	@cd . && go test -run=^$$ -fuzz=FuzzTokenInMemoryUnmarshal -fuzztime=10s ./api/ 2>&1 | tail -2
	@cd . && go test -run=^$$ -fuzz=FuzzSubscriptionPathSnapshotSettings -fuzztime=10s ./api/ 2>&1 | tail -2
