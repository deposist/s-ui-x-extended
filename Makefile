PS ?= powershell -NoProfile -ExecutionPolicy Bypass
RUN = $(PS) -File tests/baseline/run-command.ps1 -ContinueOnError

.PHONY: audit audit\:lint-go audit\:vet audit\:build audit\:test-go audit\:test-go-race audit\:cover audit\:gosec audit\:vuln audit\:fe-typecheck audit\:fe-lint audit\:fe-build audit\:test-fe audit\:e2e audit\:fe-install

audit: audit\:build audit\:vet audit\:test-go audit\:test-go-race audit\:cover audit\:gosec audit\:vuln audit\:lint-go audit\:fe-typecheck audit\:fe-lint audit\:fe-build audit\:test-fe audit\:e2e

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
