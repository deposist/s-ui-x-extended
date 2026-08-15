@echo off
setlocal EnableExtensions DisableDelayedExpansion

set "REPO_ROOT=%~dp0.."
for %%I in ("%REPO_ROOT%") do set "REPO_ROOT=%%~fI"
set "FRONTEND_DIR=%REPO_ROOT%\frontend"
set "FRONTEND_DIST=%FRONTEND_DIR%\dist"
set "WEB_DIR=%REPO_ROOT%\web"
set "WEB_HTML=%WEB_DIR%\html"
set "OUTPUT_PATH=%REPO_ROOT%\sui.exe"
set "ARCHITECTURE=amd64"
set "ARCHITECTURE_SET=0"
set "CGO_ENABLED=1"
set "STAGE_DIR="
set "BACKUP_DIR="

:parse_args
if "%~1"=="" goto args_done
if /I "%~1"=="--no-cgo" (
    set "CGO_ENABLED=0"
    shift
    goto parse_args
)
if /I "%~1"=="-NoCGO" (
    set "CGO_ENABLED=0"
    shift
    goto parse_args
)
if "%ARCHITECTURE_SET%"=="1" goto invalid_args
if /I "%~1"=="amd64" (
    set "ARCHITECTURE=amd64"
    set "ARCHITECTURE_SET=1"
    shift
    goto parse_args
)
if /I "%~1"=="386" (
    set "ARCHITECTURE=386"
    set "ARCHITECTURE_SET=1"
    shift
    goto parse_args
)
if /I "%~1"=="arm64" (
    set "ARCHITECTURE=arm64"
    set "ARCHITECTURE_SET=1"
    shift
    goto parse_args
)

goto invalid_args

:args_done
if not exist "%REPO_ROOT%\main.go" goto repo_error

go version >nul 2>&1
if errorlevel 1 (
    echo Error: Go is not installed or not in PATH>&2
    goto fail
)
node --version >nul 2>&1
if errorlevel 1 (
    echo Error: Node.js is not installed or not in PATH>&2
    goto fail
)
call npm --version >nul 2>&1
if errorlevel 1 (
    echo Error: npm is not installed or not in PATH>&2
    goto fail
)

echo Building S-UI for Windows %ARCHITECTURE% with CGO_ENABLED=%CGO_ENABLED%...
echo Building frontend...
pushd "%FRONTEND_DIR%" || goto frontend_directory_error
call npm ci
if errorlevel 1 (
    echo Error: Failed to install frontend dependencies>&2
    goto frontend_fail
)
call npm run lint -- --max-warnings=0
if errorlevel 1 (
    echo Error: Frontend lint failed>&2
    goto frontend_fail
)
call npm run test
if errorlevel 1 (
    echo Error: Frontend unit tests failed>&2
    goto frontend_fail
)
call npm run build
if errorlevel 1 (
    echo Error: Frontend production build failed>&2
    goto frontend_fail
)
call npm run verify:dist
if errorlevel 1 (
    echo Error: Frontend production dist verification failed>&2
    goto frontend_fail
)
popd

REM Stage the verified dist beside web\html. Existing embedded assets remain
REM untouched until every frontend gate and the complete staging copy succeed.
if not exist "%WEB_DIR%" mkdir "%WEB_DIR%"
if errorlevel 1 (
    echo Error: Failed to create web directory>&2
    goto asset_fail
)
set "STAGE_DIR=%WEB_DIR%\.html-stage.%RANDOM%.%RANDOM%"
if exist "%STAGE_DIR%" (
    echo Error: Frontend staging path already exists: %STAGE_DIR%>&2
    goto asset_fail
)
mkdir "%STAGE_DIR%"
if errorlevel 1 (
    echo Error: Failed to create frontend staging directory>&2
    goto asset_fail
)
xcopy "%FRONTEND_DIST%\*" "%STAGE_DIR%" /E /I /H /Y /Q >nul
if errorlevel 1 (
    echo Error: Failed to stage frontend production assets>&2
    goto asset_fail
)

if exist "%WEB_HTML%" goto preserve_existing_assets
goto install_staged_assets

:preserve_existing_assets
set "BACKUP_DIR=%WEB_DIR%\.html-backup.%RANDOM%.%RANDOM%"
if exist "%BACKUP_DIR%" (
    echo Error: Web asset backup path already exists: %BACKUP_DIR%>&2
    goto asset_fail
)
mkdir "%BACKUP_DIR%"
if errorlevel 1 (
    echo Error: Failed to create web asset backup directory>&2
    goto asset_fail
)
move "%WEB_HTML%" "%BACKUP_DIR%\html" >nul
if errorlevel 1 (
    echo Error: Failed to preserve prior web assets>&2
    goto asset_fail
)

:install_staged_assets
move "%STAGE_DIR%" "%WEB_HTML%" >nul
if errorlevel 1 (
    echo Error: Failed to replace embedded web assets>&2
    goto asset_fail
)
set "STAGE_DIR="
if not defined BACKUP_DIR goto assets_ready
rmdir /S /Q "%BACKUP_DIR%"
if exist "%BACKUP_DIR%" (
    echo Error: Failed to remove prior web asset backup: %BACKUP_DIR%>&2
    goto asset_fail
)
set "BACKUP_DIR="

:assets_ready
set "GOOS=windows"
set "GOARCH=%ARCHITECTURE%"
set "BUILD_TAGS=with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,badlinkname,tfogo_checklinkname0,with_tailscale,with_dhcp,with_wireguard,with_masque,with_mtproxy,with_openvpn,with_sudoku,with_trusttunnel,with_call,with_ccm,with_ocm,with_oomkiller"
set "LDFLAGS=-w -s -checklinkname=0 -X github.com/deposist/s-ui-x-extended/config.ArtifactPlatform=%ARCHITECTURE%"

echo Building backend for GOOS=%GOOS% GOARCH=%GOARCH% CGO_ENABLED=%CGO_ENABLED%...
go -C "%REPO_ROOT%" build -ldflags "%LDFLAGS%" -tags "%BUILD_TAGS%" -o "%OUTPUT_PATH%" main.go
if errorlevel 1 (
    echo Error: Backend build failed for Windows %ARCHITECTURE% with CGO_ENABLED=%CGO_ENABLED%; no fallback was attempted>&2
    goto fail
)

echo Build completed successfully!
echo Output: %OUTPUT_PATH%
exit /b 0

:frontend_fail
popd
goto fail

:frontend_directory_error
echo Error: Frontend directory not found: %FRONTEND_DIR%>&2
goto fail

:asset_fail
if defined STAGE_DIR if exist "%STAGE_DIR%" rmdir /S /Q "%STAGE_DIR%"
if not defined BACKUP_DIR goto fail
if not exist "%BACKUP_DIR%\html" goto remove_empty_backup
if exist "%WEB_HTML%" goto backup_preserved
move "%BACKUP_DIR%\html" "%WEB_HTML%" >nul
if errorlevel 1 goto backup_preserved

:remove_empty_backup
rmdir /S /Q "%BACKUP_DIR%" >nul 2>&1
goto fail

:backup_preserved
echo Warning: Prior web assets remain preserved at %BACKUP_DIR%\html>&2
goto fail

:repo_error
echo Error: Repository root not found: %REPO_ROOT%>&2
goto fail

:invalid_args
echo Error: Unsupported arguments.>&2
echo Usage: %~nx0 [amd64^|386^|arm64] [--no-cgo]>&2
exit /b 2

:fail
exit /b 1
