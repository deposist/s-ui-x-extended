param()

$ErrorActionPreference = "Continue"
$exitCode = 0

Write-Output "## staticcheck ./..."
& staticcheck ./...
if ($LASTEXITCODE -ne 0) {
    $exitCode = $LASTEXITCODE
}

Write-Output ""
Write-Output "## golangci-lint run"
& golangci-lint run
if ($LASTEXITCODE -ne 0 -and $exitCode -eq 0) {
    $exitCode = $LASTEXITCODE
}

exit $exitCode
