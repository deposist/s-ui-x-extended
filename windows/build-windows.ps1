# PowerShell script for building S-UI on Windows
[CmdletBinding()]
param(
    [ValidateSet("amd64", "386", "arm64")]
    [string]$Architecture = "amd64",
    [switch]$NoCGO,
    [switch]$Help
)

$ErrorActionPreference = "Stop"

if ($Help) {
    Write-Host "Usage: .\build-windows.ps1 [-Architecture <arch>] [-NoCGO] [-Help]"
    Write-Host "Architectures: amd64, 386, arm64"
    Write-Host "Examples:"
    Write-Host "  .\build-windows.ps1                    # Build amd64 with CGO"
    Write-Host "  .\build-windows.ps1 -Architecture 386 # Build 32-bit Windows with CGO"
    Write-Host "  .\build-windows.ps1 -Architecture arm64 -NoCGO"
    exit 0
}
$Architecture = $Architecture.ToLowerInvariant()

$repoRoot = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
$frontendDir = Join-Path $repoRoot "frontend"
$frontendDist = Join-Path $frontendDir "dist"
$webDir = Join-Path $repoRoot "web"
$webHtml = Join-Path $webDir "html"
$outputPath = Join-Path $repoRoot "sui.exe"
$stageDir = $null
$backupDir = $null
$locationPushed = $false

function Assert-Command {
    param([Parameter(Mandatory = $true)][string]$Name)

    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "$Name is not installed or not in PATH"
    }
}

function Invoke-CheckedCommand {
    param(
        [Parameter(Mandatory = $true)][string]$Command,
        [Parameter(Mandatory = $true)][string[]]$Arguments,
        [Parameter(Mandatory = $true)][string]$FailureMessage
    )

    & $Command @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw $FailureMessage
    }
}

function Restore-PreviousWebAssets {
    if ($script:backupDir -and (Test-Path -LiteralPath (Join-Path $script:backupDir "html")) -and -not (Test-Path -LiteralPath $script:webHtml)) {
        Move-Item -LiteralPath (Join-Path $script:backupDir "html") -Destination $script:webHtml
    }
}

try {
    Assert-Command "go"
    Assert-Command "node"
    Assert-Command "npm"

    Write-Host "Building S-UI for Windows ($Architecture)..." -ForegroundColor Green
    Invoke-CheckedCommand -Command "go" -Arguments @("version") -FailureMessage "Failed to query the Go toolchain"
    Invoke-CheckedCommand -Command "node" -Arguments @("--version") -FailureMessage "Failed to query Node.js"

    Write-Host "Building frontend..." -ForegroundColor Yellow
    Push-Location -LiteralPath $frontendDir
    $locationPushed = $true
    try {
        Invoke-CheckedCommand -Command "npm" -Arguments @("ci") -FailureMessage "Failed to install frontend dependencies"
        Invoke-CheckedCommand -Command "npm" -Arguments @("run", "lint", "--", "--max-warnings=0") -FailureMessage "Frontend lint failed"
        Invoke-CheckedCommand -Command "npm" -Arguments @("run", "test") -FailureMessage "Frontend unit tests failed"
        Invoke-CheckedCommand -Command "npm" -Arguments @("run", "build") -FailureMessage "Frontend production build failed"
        Invoke-CheckedCommand -Command "npm" -Arguments @("run", "verify:dist") -FailureMessage "Frontend production dist verification failed"
    }
    finally {
        Pop-Location
        $locationPushed = $false
    }

    # Copy verified output into a sibling staging directory. Existing embedded
    # assets stay intact until all frontend gates and the staging copy succeed.
    New-Item -ItemType Directory -Path $webDir -Force | Out-Null
    $stageDir = Join-Path $webDir (".html-stage." + [Guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Path $stageDir | Out-Null
    Copy-Item -Path (Join-Path $frontendDist "*") -Destination $stageDir -Recurse -Force

    if (Test-Path -LiteralPath $webHtml) {
        $backupDir = Join-Path $webDir (".html-backup." + [Guid]::NewGuid().ToString("N"))
        New-Item -ItemType Directory -Path $backupDir | Out-Null
        Move-Item -LiteralPath $webHtml -Destination (Join-Path $backupDir "html")
    }

    try {
        Move-Item -LiteralPath $stageDir -Destination $webHtml
        $stageDir = $null
    }
    catch {
        Restore-PreviousWebAssets
        throw "Failed to replace embedded web assets: $($_.Exception.Message)"
    }

    if ($backupDir) {
        Remove-Item -LiteralPath $backupDir -Recurse -Force
        $backupDir = $null
    }

    $env:GOOS = "windows"
    $env:GOARCH = $Architecture
    $env:CGO_ENABLED = if ($NoCGO) { "0" } else { "1" }

    Write-Host "Building backend with CGO_ENABLED=$env:CGO_ENABLED..." -ForegroundColor Yellow
    $buildTags = "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,badlinkname,tfogo_checklinkname0,with_tailscale,with_dhcp,with_wireguard,with_masque,with_mtproxy,with_openvpn,with_sudoku,with_trusttunnel,with_ccm,with_ocm,with_oomkiller"
    $artifactPlatformFlag = "github.com/deposist/s-ui-x-extended/config.ArtifactPlatform=$Architecture"
    $ldFlags = "-w -s -checklinkname=0 -X $artifactPlatformFlag"
    $buildArguments = @("-C", $repoRoot, "build", "-ldflags", $ldFlags, "-tags", $buildTags, "-o", $outputPath, "main.go")
    Invoke-CheckedCommand -Command "go" -Arguments $buildArguments -FailureMessage "Backend build failed for Windows $Architecture with CGO_ENABLED=$env:CGO_ENABLED; no fallback was attempted"

    $fileInfo = Get-Item -LiteralPath $outputPath
    Write-Host "Build completed successfully!" -ForegroundColor Green
    Write-Host "Output: $outputPath" -ForegroundColor Green
    Write-Host "File size: $([math]::Round($fileInfo.Length / 1MB, 2)) MB" -ForegroundColor Cyan
}
catch {
    [Console]::Error.WriteLine("Error: $($_.Exception.Message)")
    exit 1
}
finally {
    if ($locationPushed) {
        Pop-Location
    }
    if ($stageDir -and (Test-Path -LiteralPath $stageDir)) {
        Remove-Item -LiteralPath $stageDir -Recurse -Force -ErrorAction SilentlyContinue
    }
    try {
        Restore-PreviousWebAssets
    }
    catch {
        [Console]::Error.WriteLine("Error: Failed to restore prior web assets from $backupDir: $($_.Exception.Message)")
    }
    if ($backupDir -and (Test-Path -LiteralPath $backupDir)) {
        $savedHtml = Join-Path $backupDir "html"
        if (-not (Test-Path -LiteralPath $savedHtml)) {
            Remove-Item -LiteralPath $backupDir -Recurse -Force -ErrorAction SilentlyContinue
        }
        else {
            Write-Warning "Prior web assets remain preserved at $savedHtml"
        }
    }
}
