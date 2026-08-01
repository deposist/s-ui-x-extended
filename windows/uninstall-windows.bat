@echo off
setlocal enabledelayedexpansion

echo ========================================
echo S-UI Windows Uninstaller
echo ========================================

REM Check if running as Administrator
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo Error: This script must be run as Administrator
    echo Right-click on this file and select "Run as administrator"
    pause
    exit /b 1
)

REM Set installation directory
set "INSTALL_DIR=C:\Program Files\s-ui"
set "SERVICE_NAME=s-ui"

echo Uninstalling S-UI from: %INSTALL_DIR%

REM Stop and remove the service through the trusted Windows SCM. Never execute
REM an installed wrapper during uninstall: the installation directory may have
REM been modified since installation.
echo Stopping and removing Windows Service...
net stop %SERVICE_NAME% >nul 2>&1
sc.exe delete %SERVICE_NAME% >nul 2>&1
if %errorLevel% equ 0 (
    echo Service deletion requested successfully
) else (
    echo Warning: Failed to delete service or service was not installed
)

REM Remove desktop shortcut
echo Removing desktop shortcut...
set "DESKTOP=%USERPROFILE%\Desktop"
if exist "%DESKTOP%\S-UI.lnk" (
    del "%DESKTOP%\S-UI.lnk" >nul 2>&1
    echo Desktop shortcut removed
)

REM Remove Start Menu shortcut
echo Removing Start Menu shortcut...
set "START_MENU=%APPDATA%\Microsoft\Windows\Start Menu\Programs\S-UI"
if exist "%START_MENU%" (
    rmdir /s /q "%START_MENU%" >nul 2>&1
    echo Start Menu shortcut removed
)

REM Remove environment variable
echo Removing environment variable...
reg delete "HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Session Manager\Environment" /v SUI_HOME /f >nul 2>&1

REM Ask user if they want to keep data
echo.
set /p keep_data="Do you want to keep your data (database, logs, certificates)? [y/n]: "
if /i "%keep_data%"=="y" (
    echo Keeping data files...
    REM Copy data outside Program Files before removing every executable,
    REM installer and unknown file. This avoids leaving a service-writable tree
    REM next to a future executable.
    set "DATA_BACKUP=%ProgramData%\s-ui"
    if exist "!DATA_BACKUP!" (
        echo Error: backup directory already exists: !DATA_BACKUP!
        echo Move or remove it before preserving another installation.
        exit /b 1
    )
    mkdir "!DATA_BACKUP!" || exit /b 1
    for %%D in (db logs cert) do if exist "%INSTALL_DIR%\%%D" (
        robocopy "%INSTALL_DIR%\%%D" "!DATA_BACKUP!\%%D" /E /COPY:DAT /DCOPY:DAT /R:2 /W:1 >nul
        if !errorLevel! GEQ 8 (
            echo Error: failed to preserve %%D; installation was not removed.
            exit /b 1
        )
    )
    rmdir /s /q "%INSTALL_DIR%" >nul 2>&1
    if exist "%INSTALL_DIR%" (
        echo Error: executable/installer files could not be removed: %INSTALL_DIR%
        exit /b 1
    )
    echo Data files preserved in: !DATA_BACKUP!
) else (
    echo Removing all files...
    REM Remove entire installation directory
    if exist "%INSTALL_DIR%" (
        rmdir /s /q "%INSTALL_DIR%" >nul 2>&1
        if exist "%INSTALL_DIR%" (
            echo Warning: Some files could not be removed. Please manually delete: %INSTALL_DIR%
        ) else (
            echo All files removed successfully
        )
    )
)

REM Remove firewall rules
echo Removing firewall rules...
netsh advfirewall firewall delete rule name="S-UI Panel" >nul 2>&1
netsh advfirewall firewall delete rule name="S-UI Subscription" >nul 2>&1

echo.
echo ========================================
echo Uninstallation completed!
echo ========================================
echo.
echo S-UI has been uninstalled from your system.
echo.
if /i "%keep_data%"=="y" (
    echo Your data has been preserved in: %ProgramData%\s-ui
    echo You can safely delete this directory if you no longer need the data.
)
echo.
echo Thank you for using S-UI!
echo.
pause
