param(
    [Parameter(Mandatory = $true)]
    [string]$Phase,

    [Parameter(Mandatory = $true)]
    [string]$Name,

    [string]$CommandLine = "",

    [string]$WorkingDirectory = ".",

    [string]$SkipReason = "",

    [switch]$ContinueOnError
)

function Escape-Xml {
    param([string]$Value)

    if ($null -eq $Value) {
        return ""
    }

    return $Value.
        Replace("&", "&amp;").
        Replace("<", "&lt;").
        Replace(">", "&gt;").
        Replace('"', "&quot;").
        Replace("'", "&apos;")
}

$phaseDir = Join-Path "tests/baseline" $Phase
New-Item -ItemType Directory -Force -Path $phaseDir | Out-Null

$base = Join-Path $phaseDir $Name
$txtPath = "$base.txt"
$xmlPath = "$base.junit.xml"
$start = Get-Date
$exitCode = 0
$output = ""
$status = "passed"

if ($SkipReason -ne "") {
    $status = "skipped"
    $output = $SkipReason
} else {
    Push-Location $WorkingDirectory
    try {
        $output = Invoke-Expression $CommandLine 2>&1 | Out-String
        $exitCode = $LASTEXITCODE
        if ($null -eq $exitCode) {
            $exitCode = 0
        }
        if ($exitCode -ne 0) {
            $status = "failed"
        }
    } catch {
        $exitCode = 127
        $status = "failed"
        $output = $_ | Out-String
    } finally {
        Pop-Location
    }
}

$end = Get-Date
$duration = [Math]::Max(0, ($end - $start).TotalSeconds)
$header = @(
    "# Command: $CommandLine",
    "# WorkingDirectory: $WorkingDirectory",
    "# Status: $status",
    "# ExitCode: $exitCode",
    "# Started: $($start.ToString('o'))",
    "# Finished: $($end.ToString('o'))",
    ""
) -join "`r`n"

Set-Content -LiteralPath $txtPath -Value ($header + $output) -Encoding UTF8

$escapedOutput = Escape-Xml $output
if ($status -eq "skipped") {
    $junit = "<?xml version=`"1.0`" encoding=`"UTF-8`"?>`n<testsuite name=`"$Name`" tests=`"1`" failures=`"0`" errors=`"0`" skipped=`"1`" time=`"$duration`">`n  <testcase classname=`"baseline.$Phase`" name=`"$Name`" time=`"$duration`">`n    <skipped message=`"skipped`">$escapedOutput</skipped>`n  </testcase>`n</testsuite>`n"
} elseif ($exitCode -eq 0) {
    $junit = "<?xml version=`"1.0`" encoding=`"UTF-8`"?>`n<testsuite name=`"$Name`" tests=`"1`" failures=`"0`" errors=`"0`" skipped=`"0`" time=`"$duration`">`n  <testcase classname=`"baseline.$Phase`" name=`"$Name`" time=`"$duration`">`n    <system-out>$escapedOutput</system-out>`n  </testcase>`n</testsuite>`n"
} else {
    $junit = "<?xml version=`"1.0`" encoding=`"UTF-8`"?>`n<testsuite name=`"$Name`" tests=`"1`" failures=`"1`" errors=`"0`" skipped=`"0`" time=`"$duration`">`n  <testcase classname=`"baseline.$Phase`" name=`"$Name`" time=`"$duration`">`n    <failure message=`"exit code $exitCode`">$escapedOutput</failure>`n    <system-out>$escapedOutput</system-out>`n  </testcase>`n</testsuite>`n"
}

Set-Content -LiteralPath $xmlPath -Value $junit -Encoding UTF8

if ($exitCode -ne 0 -and -not $ContinueOnError) {
    exit $exitCode
}

exit 0
