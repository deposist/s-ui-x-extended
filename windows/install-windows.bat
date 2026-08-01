@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

echo ========================================
echo Установщик S-UI для Windows
echo ========================================

REM Проверка запуска от имени администратора
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo Ошибка: этот скрипт нужно запускать от имени администратора
    echo Щелкните файл правой кнопкой мыши и выберите "Запуск от имени администратора"
    pause
    exit /b 1
)

cd /d "%~dp0"
REM Каталог установки
set "INSTALL_DIR=C:\Program Files\s-ui"
set "SERVICE_NAME=s-ui"

REM WinSW v2.12.0 hashes are pinned from ScoopInstaller/Main commit
REM 5ea81de9099ade91e1f3a69330532b4ed2b71678 bucket/winsw.json.
REM The upstream v2.12.0 PE files are not Authenticode-signed, so there is no
REM trusted publisher constant to pin today. Keep this empty and fail closed;
REM service delivery can be enabled only when both this publisher and the hash
REM are updated from an independently trusted source.
set "WINSW_VERSION=v2.12.0"
set "WINSW_PUBLISHER="
set "NATIVE_ARCH=%PROCESSOR_ARCHITECTURE%"
if defined PROCESSOR_ARCHITEW6432 set "NATIVE_ARCH=%PROCESSOR_ARCHITEW6432%"
if /I "%NATIVE_ARCH%"=="AMD64" (
    set "WINSW_ASSET=WinSW-x64.exe"
    set "WINSW_SHA256=05b82d46ad331cc16bdc00de5c6332c1ef818df8ceefcd49c726553209b3a0da"
) else if /I "%NATIVE_ARCH%"=="x86" (
    set "WINSW_ASSET=WinSW-x86.exe"
    set "WINSW_SHA256=0c21327463a43a61f2efb227ec4afd2467fde91618cc725148c1099001ca91ae"
) else (
    echo Ошибка: для архитектуры %NATIVE_ARCH% нет доверенного WinSW service wrapper.
    exit /b 1
)
if not defined WINSW_PUBLISHER (
    echo Ошибка: WinSW %WINSW_VERSION% не имеет проверяемой подписи Authenticode.
    echo Установка службы остановлена: использование wrapper без доверенного издателя запрещено.
    exit /b 1
)
set "WINSW_VERIFIED=%TEMP%\s-ui-winsw-%RANDOM%-%RANDOM%.exe"
set "WINSW_URL=https://github.com/winsw/winsw/releases/download/%WINSW_VERSION%/%WINSW_ASSET%"
powershell -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -Command "$ErrorActionPreference='Stop'; $ProgressPreference='SilentlyContinue'; $path=$env:WINSW_VERIFIED; try { Invoke-WebRequest -UseBasicParsing -Uri $env:WINSW_URL -OutFile $path; $actual=(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant(); if ($actual -cne $env:WINSW_SHA256) { throw ('WinSW SHA-256 mismatch: expected {0}, got {1}' -f $env:WINSW_SHA256,$actual) }; $signature=Get-AuthenticodeSignature -LiteralPath $path; if ($signature.Status -ne [System.Management.Automation.SignatureStatus]::Valid -or $null -eq $signature.SignerCertificate) { throw ('WinSW Authenticode validation failed: {0}' -f $signature.Status) }; if ($signature.SignerCertificate.Subject -cne $env:WINSW_PUBLISHER) { throw ('WinSW publisher mismatch: {0}' -f $signature.SignerCertificate.Subject) } } catch { Remove-Item -LiteralPath $path -Force -ErrorAction SilentlyContinue; throw }"
if errorlevel 1 (
    echo Ошибка: проверка WinSW не пройдена; непроверенный wrapper удален.
    exit /b 1
)
if not exist "%WINSW_VERIFIED%" (
    echo Ошибка: проверенный WinSW отсутствует.
    exit /b 1
)

echo Установка S-UI в каталог: %INSTALL_DIR%

REM Создание каталога установки
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"
if not exist "%INSTALL_DIR%\db" mkdir "%INSTALL_DIR%\db"
if not exist "%INSTALL_DIR%\logs" mkdir "%INSTALL_DIR%\logs"
if not exist "%INSTALL_DIR%\cert" mkdir "%INSTALL_DIR%\cert"

REM Копирование файлов
echo Копирование файлов...
copy "sui.exe" "%INSTALL_DIR%\" >nul
if errorlevel 1 (
    echo Ошибка: не удалось скопировать sui.exe
    exit /b 1
)
copy "libcronet.dll" "%INSTALL_DIR%\" >nul
if errorlevel 1 (
    echo Ошибка: не удалось скопировать libcronet.dll
    exit /b 1
)
copy "s-ui-windows.xml" "%INSTALL_DIR%\" >nul
copy "s-ui-windows.bat" "%INSTALL_DIR%\" >nul

if errorlevel 1 (
    echo Ошибка: не удалось скопировать файлы конфигурации
    exit /b 1
)

REM Программа и wrapper доступны только для чтения/выполнения службе и обычным пользователям.
REM LocalService (S-1-5-19) получает Modify исключительно в каталогах данных.
echo Настройка прав доступа...
icacls "%INSTALL_DIR%" /inheritance:r >nul || exit /b 1
icacls "%INSTALL_DIR%" /remove:g *S-1-5-19 *S-1-5-32-545 >nul 2>&1
icacls "%INSTALL_DIR%" /grant:r "*S-1-5-18:(OI)(CI)F" "*S-1-5-32-544:(OI)(CI)F" "*S-1-5-19:(OI)(CI)RX" "*S-1-5-32-545:(OI)(CI)RX" >nul || exit /b 1
for %%D in (db logs cert) do (
    icacls "%INSTALL_DIR%\%%D" /inheritance:r >nul || exit /b 1
    icacls "%INSTALL_DIR%\%%D" /remove:g *S-1-5-19 *S-1-5-32-545 >nul 2>&1
    icacls "%INSTALL_DIR%\%%D" /grant:r "*S-1-5-18:(OI)(CI)F" "*S-1-5-32-544:(OI)(CI)F" "*S-1-5-19:(OI)(CI)M" >nul || exit /b 1
)

REM Копирование только что проверенного wrapper; ранее установленный файл не доверяется.
set "WINSW_PATH=%INSTALL_DIR%\winsw.exe"
copy /y "%WINSW_VERIFIED%" "%WINSW_PATH%" >nul
if errorlevel 1 (
    del /f /q "%WINSW_VERIFIED%" >nul 2>&1
    echo Ошибка: не удалось установить проверенный WinSW.
    exit /b 1
)
del /f /q "%WINSW_VERIFIED%" >nul 2>&1
REM Проверенный wrapper установлен.

echo Установка службы Windows...
copy /y "%WINSW_PATH%" "%INSTALL_DIR%\s-ui-service.exe" >nul || exit /b 1
copy /y "%INSTALL_DIR%\s-ui-windows.xml" "%INSTALL_DIR%\s-ui-service.xml" >nul || exit /b 1
"%INSTALL_DIR%\s-ui-service.exe" install
if errorlevel 1 (
    echo Ошибка: не удалось установить службу.
    exit /b 1
)
echo Служба успешно установлена

REM Запуск миграции
echo Запуск миграции базы данных...
cd /d "%INSTALL_DIR%"
sui.exe migrate
if %errorLevel% equ 0 (
    echo Миграция успешно завершена
) else (
    echo Предупреждение: миграция не выполнена или база данных новая
)

REM Получение сетевой конфигурации
echo.
echo ========================================
echo Сетевая конфигурация
echo ========================================

REM Получение локальных IP-адресов
echo Доступные IP-адреса:
for /f "tokens=2 delims=:" %%i in ('ipconfig ^| findstr /i "IPv4"') do (
    echo   %%i
)

REM Получение настроек панели
echo.
set /p panel_port="Введите порт панели (по умолчанию: 2095): "
if "%panel_port%"=="" set "panel_port=2095"

set /p panel_path="Введите путь панели (по умолчанию: /app/): "
if "%panel_path%"=="" set "panel_path=/app/"

set /p sub_port="Введите порт подписки (по умолчанию: 2096): "
if "%sub_port%"=="" set "sub_port=2096"

set /p sub_path="Введите путь подписки (по умолчанию: /sub/): "
if "%sub_path%"=="" set "sub_path=/sub/"

REM Применение настроек
echo.
echo Применение настроек...
cd /d "%INSTALL_DIR%"
sui.exe setting -port %panel_port% -path "%panel_path%" -subPort %sub_port% -subPath "%sub_path%"

REM Получение учетных данных администратора
echo.
echo ========================================
echo Настройка администратора
echo ========================================

set /p admin_username="Введите имя пользователя администратора (по умолчанию: admin): "
if "%admin_username%"=="" set "admin_username=admin"

set /p admin_password="Введите пароль администратора: "
if "%admin_password%"=="" (
    echo Ошибка: пароль не может быть пустым
    pause
    exit /b 1
)

REM Настройка учетных данных администратора
echo Настройка учетных данных администратора...
sui.exe admin -username "%admin_username%" -password "%admin_password%"

REM Запуск службы
echo Запуск службы S-UI...
net start %SERVICE_NAME%
if %errorLevel% equ 0 (
    echo Служба успешно запущена
) else (
    echo Предупреждение: не удалось запустить службу. Ее можно запустить вручную позже.
)

REM Создание ярлыка на рабочем столе
echo Создание ярлыка на рабочем столе...
set "DESKTOP=%USERPROFILE%\Desktop"
if exist "%DESKTOP%" (
    powershell -Command "& {$WshShell = New-Object -comObject WScript.Shell; $Shortcut = $WshShell.CreateShortcut('%DESKTOP%\S-UI.lnk'); $Shortcut.TargetPath = '%INSTALL_DIR%\s-ui-windows.bat'; $Shortcut.WorkingDirectory = '%INSTALL_DIR%'; $Shortcut.Description = 'Панель управления S-UI'; $Shortcut.Save()}"
    echo Ярлык на рабочем столе создан
)

REM Создание ярлыка в меню Пуск
echo Создание ярлыка в меню Пуск...
set "START_MENU=%APPDATA%\Microsoft\Windows\Start Menu\Programs"
if exist "%START_MENU%" (
    if not exist "%START_MENU%\S-UI" mkdir "%START_MENU%\S-UI"
    powershell -Command "& {$WshShell = New-Object -comObject WScript.Shell; $Shortcut = $WshShell.CreateShortcut('%START_MENU%\S-UI\Панель управления S-UI.lnk'); $Shortcut.TargetPath = '%INSTALL_DIR%\s-ui-windows.bat'; $Shortcut.WorkingDirectory = '%INSTALL_DIR%'; $Shortcut.Description = 'Панель управления S-UI'; $Shortcut.Save()}"
    echo Ярлык в меню Пуск создан
)


REM Создание переменной окружения
echo Настройка переменной окружения...
setx SUI_HOME "%INSTALL_DIR%" /M >nul

REM Показ итоговой конфигурации
echo.
echo ========================================
echo Установка успешно завершена!
echo ========================================
echo.
echo S-UI установлен в каталог: %INSTALL_DIR%
echo.
echo Конфигурация:
echo   Порт панели: %panel_port%
echo   Путь панели: %panel_path%
echo   Порт подписки: %sub_port%
echo   Путь подписки: %sub_path%
echo   Имя пользователя администратора: %admin_username%
echo.
echo URL для доступа:
for /f "tokens=2 delims=:" %%i in ('ipconfig ^| findstr /i "IPv4"') do (
    set "ip=%%i"
    set "ip=!ip: =!"
    echo   Панель: http://!ip!:%panel_port%%panel_path%
    echo   Подписка: http://!ip!:%sub_port%%sub_path%
)
echo.
echo Имя службы: %SERVICE_NAME%
echo.
echo Полезные команды:
echo   net start %SERVICE_NAME%    - запустить службу
echo   net stop %SERVICE_NAME%     - остановить службу
echo   sc query %SERVICE_NAME%     - проверить состояние службы
echo.
echo Также можно использовать ярлык на рабочем столе или пункт меню Пуск.
echo.
pause
