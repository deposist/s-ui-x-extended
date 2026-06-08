@echo off
setlocal enabledelayedexpansion

echo Building S-UI for Windows...

cd /d "%~dp0"

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo Error: Go is not installed or not in PATH
    echo Please install Go from https://golang.org/dl/
    pause
    exit /b 1
)

REM Check if Node.js is installed
node --version >nul 2>&1
if errorlevel 1 (
    echo Error: Node.js is not installed or not in PATH
    echo Please install Node.js from https://nodejs.org/
    pause
    exit /b 1
)

echo Building frontend...
cd frontend
call npm install
if errorlevel 1 (
    echo Error: Failed to install frontend dependencies
    pause
    exit /b 1
)

call npm run build
if errorlevel 1 (
    echo Error: Failed to build frontend
    pause
    exit /b 1
)

cd ..

echo Creating web/html directory...
if exist "web\html" rmdir /s /q "web\html"
mkdir "web\html"

echo Copying frontend build files...
xcopy "frontend\dist\*" "web\html\" /E /Y /Q

echo Building backend...
set CGO_ENABLED=1
set GOOS=windows
set GOARCH=amd64

REM Try to build with CGO first
go build -ldflags "-w -s -checklinkname=0" -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,badlinkname,tfogo_checklinkname0,with_tailscale,with_dhcp,with_wireguard,with_masque,with_mtproxy,with_openvpn,with_sudoku,with_trusttunnel,with_ccm,with_ocm,with_oomkiller" -o sui.exe main.go
if errorlevel 1 (
    echo Warning: CGO build failed, trying without CGO...
    set CGO_ENABLED=0
    go build -ldflags "-w -s -checklinkname=0" -tags "with_quic,with_grpc,with_utls,with_acme,with_gvisor,with_naive_outbound,with_purego,badlinkname,tfogo_checklinkname0,with_tailscale,with_dhcp,with_wireguard,with_masque,with_mtproxy,with_openvpn,with_sudoku,with_trusttunnel,with_ccm,with_ocm,with_oomkiller" -o sui.exe main.go
    if errorlevel 1 (
        echo Error: Failed to build backend
        pause
        exit /b 1
    )
    echo Built without CGO (some features may be limited)
) else (
    echo Built with CGO
)

REM The Naive outbound is linked via with_purego and loads cronet at runtime, so
REM it needs libcronet.dll next to sui.exe.
echo Downloading libcronet.dll (required for the Naive outbound)...
curl -L -o libcronet.dll https://github.com/SagerNet/cronet-go/releases/latest/download/libcronet-windows-amd64.dll
if errorlevel 1 (
    echo Warning: failed to download libcronet.dll. Naive will not work until libcronet.dll is placed next to sui.exe.
)

echo Build completed successfully!
echo Output: sui.exe
pause
