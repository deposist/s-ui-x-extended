$ErrorActionPreference = "Stop"

$directories = @(go list -f '{{.Dir}}' ./...)
if ($LASTEXITCODE -ne 0 -or $directories.Count -eq 0) {
    throw "go list did not return package directories"
}

& gosec -terse -nosec-require-justification -nosec-require-rules @directories
exit $LASTEXITCODE
