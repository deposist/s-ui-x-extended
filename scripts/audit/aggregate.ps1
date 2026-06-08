param(
    [string]$Root = ".",
    [string]$OutDir = "tests/baseline/phase8"
)

$ErrorActionPreference = "Stop"

$rootPath = (Resolve-Path -LiteralPath $Root).Path
$baselineDir = Join-Path $rootPath "tests/baseline"
$outputDir = Join-Path $rootPath $OutDir
New-Item -ItemType Directory -Force -Path $outputDir | Out-Null

function Count-Matches {
    param(
        [string]$Text,
        [string]$Pattern
    )
    if ([string]::IsNullOrEmpty($Text)) {
        return 0
    }
    return ([regex]::Matches($Text, $Pattern, [System.Text.RegularExpressions.RegexOptions]::IgnoreCase)).Count
}

function Get-RelativePathCompat {
    param(
        [string]$BasePath,
        [string]$TargetPath
    )
    $baseFull = [System.IO.Path]::GetFullPath($BasePath)
    if (-not $baseFull.EndsWith([System.IO.Path]::DirectorySeparatorChar)) {
        $baseFull += [System.IO.Path]::DirectorySeparatorChar
    }
    $targetFull = [System.IO.Path]::GetFullPath($TargetPath)
    $baseUri = New-Object System.Uri($baseFull)
    $targetUri = New-Object System.Uri($targetFull)
    return [System.Uri]::UnescapeDataString($baseUri.MakeRelativeUri($targetUri).ToString()).Replace('/', [System.IO.Path]::DirectorySeparatorChar)
}

function Get-XmlIntAttr {
    param(
        $Node,
        [string]$Name
    )
    if ($null -eq $Node -or $null -eq $Node.Attributes -or $null -eq $Node.Attributes[$Name]) {
        return 0
    }
    $value = 0
    if ([int]::TryParse($Node.Attributes[$Name].Value, [ref]$value)) {
        return $value
    }
    return 0
}

function Get-Phase {
    param([string]$Path)
    $relative = Get-RelativePathCompat $baselineDir $Path
    foreach ($part in ($relative -split '[\\/]')) {
        if ($part -match '^phase\d+') {
            return $part
        }
    }
    return "unknown"
}

$phaseMap = @{}
$files = @()
$totals = [ordered]@{ tests = 0; green = 0; red = 0; skipped = 0; xfail = 0 }

$junitFiles = @()
if (Test-Path -LiteralPath $baselineDir) {
    $junitFiles = Get-ChildItem -LiteralPath $baselineDir -Recurse -Filter "*.junit.xml" | Sort-Object FullName
}

foreach ($file in $junitFiles) {
    $xmlText = Get-Content -LiteralPath $file.FullName -Raw
    $tests = 0
    $failures = 0
    $errors = 0
    $skipped = 0

    try {
        [xml]$doc = $xmlText
        $suites = $doc.SelectNodes("//testsuite")
        foreach ($suite in $suites) {
            $tests += Get-XmlIntAttr $suite "tests"
            $failures += Get-XmlIntAttr $suite "failures"
            $errors += Get-XmlIntAttr $suite "errors"
            $skipped += Get-XmlIntAttr $suite "skipped"
        }
    } catch {
        $tests = Count-Matches $xmlText "<testcase\b"
        $failures = Count-Matches $xmlText "<failure\b"
        $errors = Count-Matches $xmlText "<error\b"
        $skipped = Count-Matches $xmlText "<skipped\b"
    }

    $txtPath = $file.FullName -replace '\.junit\.xml$', '.txt'
    $txtText = ""
    if (Test-Path -LiteralPath $txtPath) {
        $txtText = Get-Content -LiteralPath $txtPath -Raw
    }

    $xfail = Count-Matches ($xmlText + "`n" + $txtText) "\b(XFAIL|expected[- ]fail|fixme)\b"
    $red = $failures + $errors
    $green = [Math]::Max(0, $tests - $red - $skipped)
    $phase = Get-Phase $file.FullName

    if (-not $phaseMap.ContainsKey($phase)) {
        $phaseMap[$phase] = [ordered]@{ phase = $phase; tests = 0; green = 0; red = 0; skipped = 0; xfail = 0; files = 0 }
    }

    $bucket = $phaseMap[$phase]
    $bucket.tests += $tests
    $bucket.green += $green
    $bucket.red += $red
    $bucket.skipped += $skipped
    $bucket.xfail += $xfail
    $bucket.files += 1

    $totals.tests += $tests
    $totals.green += $green
    $totals.red += $red
    $totals.skipped += $skipped
    $totals.xfail += $xfail

    $files += [ordered]@{
        phase = $phase
        file = (Get-RelativePathCompat $rootPath $file.FullName).Replace("\", "/")
        tests = $tests
        green = $green
        red = $red
        skipped = $skipped
        xfail = $xfail
    }
}

$summaryPath = Join-Path $baselineDir "SUMMARY.md"
$summaryText = ""
if (Test-Path -LiteralPath $summaryPath) {
    $summaryText = Get-Content -LiteralPath $summaryPath -Raw
}

$baselineMarkers = [ordered]@{
    green = Count-Matches $summaryText "\bgreen\b"
    red = Count-Matches $summaryText "\bred\b"
    skipped = Count-Matches $summaryText "\bskip(?:ped)?\b"
    xfail = Count-Matches $summaryText "\bXFAIL\b"
}

$phases = $phaseMap.Values | Sort-Object phase
$result = [ordered]@{
    generatedAt = (Get-Date).ToUniversalTime().ToString("o")
    root = $rootPath
    summaryPath = (Get-RelativePathCompat $rootPath $summaryPath).Replace("\", "/")
    totals = $totals
    baselineMarkers = $baselineMarkers
    deltaVsBaselineMarkers = [ordered]@{
        green = $totals.green - $baselineMarkers.green
        red = $totals.red - $baselineMarkers.red
        skipped = $totals.skipped - $baselineMarkers.skipped
        xfail = $totals.xfail - $baselineMarkers.xfail
    }
    phases = @($phases)
    files = @($files)
}

$jsonPath = Join-Path $outputDir "summary.json"
$htmlPath = Join-Path $outputDir "summary.html"
$junitPath = Join-Path $outputDir "aggregate.junit.xml"

$result | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $jsonPath -Encoding UTF8

$rows = foreach ($phase in $phases) {
    "<tr><td>$($phase.phase)</td><td>$($phase.files)</td><td>$($phase.tests)</td><td class=`"green`">$($phase.green)</td><td class=`"red`">$($phase.red)</td><td>$($phase.skipped)</td><td>$($phase.xfail)</td></tr>"
}

$html = @"
<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <title>s-ui-x audit dashboard</title>
  <style>
    body { font-family: system-ui, -apple-system, Segoe UI, sans-serif; margin: 32px; color: #17202a; }
    table { border-collapse: collapse; width: 100%; margin-top: 16px; }
    th, td { border: 1px solid #d8dee4; padding: 8px 10px; text-align: left; }
    th { background: #f6f8fa; }
    .green { color: #1a7f37; font-weight: 600; }
    .red { color: #cf222e; font-weight: 600; }
    .meta { color: #57606a; }
  </style>
</head>
<body>
  <h1>s-ui-x audit dashboard</h1>
  <p class="meta">Generated at $($result.generatedAt)</p>
  <p>Totals: <span class="green">$($totals.green) green</span>, <span class="red">$($totals.red) red</span>, $($totals.skipped) skipped, $($totals.xfail) XFAIL markers.</p>
  <table>
    <thead><tr><th>Phase</th><th>JUnit files</th><th>Tests</th><th>Green</th><th>Red</th><th>Skipped</th><th>XFAIL markers</th></tr></thead>
    <tbody>$($rows -join "`n")</tbody>
  </table>
</body>
</html>
"@

Set-Content -LiteralPath $htmlPath -Value $html -Encoding UTF8

$junit = @"
<?xml version="1.0" encoding="UTF-8"?>
<testsuite name="audit-aggregate" tests="1" failures="0" errors="0" skipped="0">
  <testcase classname="audit.phase8" name="aggregate" />
</testsuite>
"@
Set-Content -LiteralPath $junitPath -Value $junit -Encoding UTF8

Write-Output "Audit dashboard written to $((Get-RelativePathCompat $rootPath $htmlPath).Replace('\', '/'))"
Write-Output "Totals: green=$($totals.green) red=$($totals.red) skipped=$($totals.skipped) xfail=$($totals.xfail)"
