# SSRF Validator Tests

The SSRF validator is intentionally a small, pure validation package. Its unit
tests exercise URL parsing, scheme allow-listing, and public-address filtering.

## Windows race runs

`go test -race ./util/ssrf` requires CGO and creates a temporary
`ssrf.test.exe`. On some Windows hosts antivirus heuristics can block that
temporary executable before the Go test runner starts it. That is an environment
failure, not a validator failure.

For race coverage, run the package on Linux/CI:

```powershell
$env:CGO_ENABLED = "1"
go test -race ./util/ssrf
```

If Windows race coverage is required, install a working CGO C compiler and
allow the Go build/test temp directory in the local antivirus policy. Do not
weaken the SSRF tests or the outbound URL validation rules to work around the
Windows temp-executable block.
