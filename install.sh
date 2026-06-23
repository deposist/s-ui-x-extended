#!/bin/bash
# S-UI installer with multilingual UI (English / Russian / Chinese).
# Language choice can be supplied non-interactively via env:
#   SUI_LANG=en|ru|zh  bash install.sh ...
# A version tag (e.g. "v1.4.2-beta") may be provided as the only positional
# argument to install a specific release.

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
plain='\033[0m'

LANG_FILE="/etc/s-ui/lang"
SECRETBOX_ENV_DIR="/etc/s-ui"
SECRETBOX_ENV_FILE="${SECRETBOX_ENV_DIR}/secretbox.env"
SECRETBOX_DROPIN_DIR="/etc/systemd/system/s-ui.service.d"
SECRETBOX_DROPIN_FILE="${SECRETBOX_DROPIN_DIR}/10-secretbox-env.conf"

ask_language() {
    if [[ -n "${SUI_LANG}" ]]; then
        case "${SUI_LANG}" in
            en|ru|zh) lang="${SUI_LANG}"; return ;;
        esac
    fi
    if [[ -f "${LANG_FILE}" ]]; then
        local saved
        saved=$(cat "${LANG_FILE}" 2>/dev/null | tr -d '[:space:]')
        case "${saved}" in
            en|ru|zh) lang="${saved}"; return ;;
        esac
    fi
    if [[ ! -t 0 ]]; then
        # Non-interactive (piped from curl) and no env: default to English.
        lang="en"
        return
    fi
    echo
    echo "Select language / Выберите язык / 请选择语言:"
    echo "  1) English"
    echo "  2) Русский"
    echo "  3) 中文"
    read -rp "[1-3, default 1]: " lang_choice
    case "${lang_choice}" in
        2|ru|RU|Russian|Русский) lang="ru" ;;
        3|zh|ZH|Chinese|中文|简体中文) lang="zh" ;;
        *) lang="en" ;;
    esac
}

t() {
    local key="$1"
    if [[ "${lang}" == "zh" ]]; then
        case "${key}" in
            run_as_root)        echo "致命错误：请使用 root 权限运行此脚本"; return ;;
            detect_failed)      echo "检测系统失败，请联系作者！"; return ;;
            current_release)    echo "当前系统发行版为：$2"; return ;;
            arch_label)         echo "架构：$2"; return ;;
            arch_unsupported)   echo "不支持的 CPU 架构！"; return ;;
            running)            echo "正在执行..."; return ;;
            migrate)            echo "正在迁移..."; return ;;
            install_done)       echo "安装/更新完成！出于安全考虑，建议修改面板设置"; return ;;
            continue_settings)  echo "是否继续修改设置 [y/n]？"; return ;;
            enter_panel_port)   echo "请输入面板端口（留空则使用现有/默认值）："; return ;;
            enter_panel_path)   echo "请输入面板路径（留空则使用现有/默认值）："; return ;;
            enter_sub_port)     echo "请输入订阅端口（留空则使用现有/默认值）："; return ;;
            enter_sub_path)     echo "请输入订阅路径（留空则使用现有/默认值）："; return ;;
            initializing)       echo "正在初始化，请稍候..."; return ;;
            change_admin)       echo "是否修改管理员账号密码 [y/n]？"; return ;;
            set_username)       echo "请设置用户名："; return ;;
            set_password)       echo "请设置密码："; return ;;
            current_admin)      echo "当前管理员账号密码："; return ;;
            cancelled)          echo "已取消..."; return ;;
            fresh_install_creds) echo "这是全新安装，出于安全考虑将生成随机登录信息："; return ;;
            username_label)     echo "用户名：$2"; return ;;
            password_label)     echo "密码：$2"; return ;;
            lost_creds)         echo "如果忘记登录信息，可以输入 s-ui 打开配置菜单"; return ;;
            upgrade_keep_settings) echo "这是升级安装，将保留旧设置；如果忘记登录信息，可以输入 s-ui 打开配置菜单"; return ;;
            stop_singbox)       echo "正在停止 sing-box 服务..."; return ;;
            bin_dir_exists)     echo "/usr/local/s-ui/bin 目录已存在！请检查其中内容，并在迁移后手动删除"; return ;;
            fetching_latest)    echo "已获取 s-ui 最新版本：$2，开始安装..."; return ;;
            rate_limited)       echo "获取 s-ui 版本失败，可能是 Github API 限制导致，请稍后重试"; return ;;
            download_failed)    echo "下载 s-ui 失败，请确认服务器可以访问 Github"; return ;;
            checksum_failed)    echo "s-ui 校验和验证失败，请稍后重试或检查发布文件"; return ;;
            installing_specific) echo "开始安装 s-ui $2"; return ;;
            download_failed_specific) echo "下载 s-ui $2 失败，请检查该版本是否存在"; return ;;
            installed_running)  echo "s-ui $2 安装完成，现已启动并运行..."; return ;;
            panel_url)          echo "你可以通过以下 URL 访问面板："; return ;;
            secretbox_key_generated) echo "已生成加密设置的 S-UI secretbox 密钥。该值只显示一次："; return ;;
            secretbox_key_label) echo "SUI_SECRETBOX_KEY：$2"; return ;;
            secretbox_key_file) echo "密钥文件：$2"; return ;;
            secretbox_key_keep) echo "请保持该文件和密钥私密，并在更新、恢复时保留同一个值。"; return ;;
            cookie_key_generated) echo "已生成 S-UI 会话 Cookie 密钥。该值只显示一次："; return ;;
            cookie_key_label) echo "SUI_COOKIE_KEY：$2"; return ;;
            cookie_key_relogin) echo "现有浏览器会话需要重新登录一次。"; return ;;
        esac
    fi
    case "${lang}:${key}" in
        # generic
        en:run_as_root)        echo "Critical error: run this script as root";;
        ru:run_as_root)        echo "Критическая ошибка: запустите этот скрипт с правами root";;
        en:detect_failed)      echo "Could not detect the system, please contact the maintainer.";;
        ru:detect_failed)      echo "Не удалось определить систему, обратитесь к автору.";;
        en:current_release)    echo "Detected distribution: $2";;
        ru:current_release)    echo "Текущий дистрибутив: $2";;
        en:arch_label)         echo "Architecture: $2";;
        ru:arch_label)         echo "Архитектура: $2";;
        en:arch_unsupported)   echo "CPU architecture is not supported.";;
        ru:arch_unsupported)   echo "Архитектура CPU не поддерживается.";;

        en:running)            echo "Running...";;
        ru:running)            echo "Выполняется...";;
        en:migrate)            echo "Running migration...";;
        ru:migrate)            echo "Выполняется миграция...";;
        en:install_done)       echo "Install/upgrade complete. For security reasons it is recommended to change panel settings.";;
        ru:install_done)       echo "Установка/обновление завершены. Из соображений безопасности рекомендуется изменить настройки панели.";;
        en:continue_settings)  echo "Continue editing settings? [y/n] ";;
        ru:continue_settings)  echo "Продолжить изменение настроек? [y/n] ";;
        en:enter_panel_port)   echo "Enter panel port (leave empty to keep current/default):";;
        ru:enter_panel_port)   echo "Введите порт панели (оставьте пустым, чтобы использовать текущее/стандартное значение):";;
        en:enter_panel_path)   echo "Enter panel path (leave empty to keep current/default):";;
        ru:enter_panel_path)   echo "Введите путь панели (оставьте пустым, чтобы использовать текущее/стандартное значение):";;
        en:enter_sub_port)     echo "Enter subscription port (leave empty to keep current/default):";;
        ru:enter_sub_port)     echo "Введите порт подписки (оставьте пустым, чтобы использовать текущее/стандартное значение):";;
        en:enter_sub_path)     echo "Enter subscription path (leave empty to keep current/default):";;
        ru:enter_sub_path)     echo "Введите путь подписки (оставьте пустым, чтобы использовать текущее/стандартное значение):";;
        en:initializing)       echo "Initializing, please wait...";;
        ru:initializing)       echo "Инициализация, подождите...";;
        en:change_admin)       echo "Change admin credentials? [y/n] ";;
        ru:change_admin)       echo "Изменить логин и пароль администратора? [y/n] ";;
        en:set_username)       echo "Username: ";;
        ru:set_username)       echo "Имя пользователя: ";;
        en:set_password)       echo "Password: ";;
        ru:set_password)       echo "Пароль: ";;
        en:current_admin)      echo "Current admin credentials:";;
        ru:current_admin)      echo "Текущие учетные данные администратора:";;
        en:cancelled)          echo "Cancelled.";;
        ru:cancelled)          echo "Отменено.";;
        en:fresh_install_creds) echo "Fresh install detected. For security a random username/password were generated:";;
        ru:fresh_install_creds) echo "Это новая установка. Из соображений безопасности будут сгенерированы случайные данные для входа:";;
        en:username_label)     echo "Username: $2";;
        ru:username_label)     echo "Имя пользователя: $2";;
        en:password_label)     echo "Password: $2";;
        ru:password_label)     echo "Пароль: $2";;
        en:lost_creds)         echo "If you forget the credentials, run 's-ui' to open the management menu.";;
        ru:lost_creds)         echo "Если вы забыли данные для входа, введите s-ui для открытия меню настроек.";;
        en:upgrade_keep_settings) echo "Upgrade detected; existing settings are preserved. Use 's-ui' menu to recover credentials if needed.";;
        ru:upgrade_keep_settings) echo "Это обновление; старые настройки сохраняются. Откройте меню s-ui для восстановления данных входа.";;

        en:stop_singbox)       echo "Stopping legacy sing-box service...";;
        ru:stop_singbox)       echo "Останавливается служба sing-box...";;
        en:bin_dir_exists)     echo "Directory /usr/local/s-ui/bin already exists; please review and remove it manually after the migration.";;
        ru:bin_dir_exists)     echo "Каталог /usr/local/s-ui/bin уже существует. Проверьте его содержимое и удалите вручную после миграции.";;

        en:fetching_latest)    echo "Got the latest s-ui version: $2. Starting installation...";;
        ru:fetching_latest)    echo "Получена последняя версия s-ui: $2. Начинается установка...";;
        en:rate_limited)       echo "Could not retrieve s-ui version. GitHub API rate limit may apply, please retry later.";;
        ru:rate_limited)       echo "Не удалось получить версию s-ui. Возможно, сработало ограничение GitHub API. Повторите попытку позже.";;
        en:download_failed)    echo "Could not download s-ui. Verify the server has access to GitHub.";;
        ru:download_failed)    echo "Не удалось скачать s-ui. Убедитесь, что сервер имеет доступ к GitHub.";;
        en:checksum_failed)    echo "s-ui checksum verification failed. Retry later or check the release files.";;
        ru:checksum_failed)    echo "Проверка checksum s-ui не прошла. Повторите позже или проверьте файлы релиза.";;
        en:installing_specific) echo "Installing s-ui $2";;
        ru:installing_specific) echo "Начинается установка s-ui $2";;
        en:download_failed_specific) echo "Could not download s-ui $2. Make sure this version exists.";;
        ru:download_failed_specific) echo "Не удалось скачать s-ui $2. Проверьте, существует ли эта версия.";;
        en:installed_running)  echo "s-ui $2 is installed, started and running...";;
        ru:installed_running)  echo "s-ui $2 установлен, запущен и работает...";;
        en:panel_url)          echo "Panel is available at:";;
        ru:panel_url)          echo "Панель доступна по адресу:";;
        en:secretbox_key_generated) echo "Generated the S-UI secretbox key for encrypted settings. It is shown once:";;
        ru:secretbox_key_generated) echo "Сгенерирован S-UI secretbox key для зашифрованных настроек. Он показывается один раз:";;
        en:secretbox_key_label) echo "SUI_SECRETBOX_KEY: $2";;
        ru:secretbox_key_label) echo "SUI_SECRETBOX_KEY: $2";;
        en:secretbox_key_file) echo "Key file: $2";;
        ru:secretbox_key_file) echo "Файл ключа: $2";;
        en:secretbox_key_keep) echo "Keep this file and key private, and preserve the same value across updates and restores.";;
        ru:secretbox_key_keep) echo "Держите этот файл и ключ в секрете и сохраняйте то же значение при обновлениях и восстановлении.";;
        en:cookie_key_generated) echo "Generated the S-UI session cookie key. It is shown once:";;
        ru:cookie_key_generated) echo "Сгенерирован ключ сессионных cookie S-UI. Он показывается один раз:";;
        en:cookie_key_label) echo "SUI_COOKIE_KEY: $2";;
        ru:cookie_key_label) echo "SUI_COOKIE_KEY: $2";;
        en:cookie_key_relogin) echo "Existing browser sessions will require one re-login.";;
        ru:cookie_key_relogin) echo "Существующие сессии браузера потребуют одного повторного входа.";;
        *) echo "${key}";;
    esac
}

cur_dir=$(pwd)

ask_language

[[ $EUID -ne 0 ]] && echo -e "${red}$(t run_as_root)${plain}\n" && exit 1

# Persist selected language so the management menu picks it up.
mkdir -p "$(dirname "${LANG_FILE}")"
printf '%s\n' "${lang}" >"${LANG_FILE}" 2>/dev/null || true

if [[ -f /etc/os-release ]]; then
    source /etc/os-release
    release=$ID
elif [[ -f /usr/lib/os-release ]]; then
    source /usr/lib/os-release
    release=$ID
else
    echo "$(t detect_failed)" >&2
    exit 1
fi
echo "$(t current_release "${release}")"

arch() {
    case "$(uname -m)" in
    x86_64 | x64 | amd64) echo 'amd64' ;;
    i*86 | x86) echo '386' ;;
    armv8* | armv8 | arm64 | aarch64) echo 'arm64' ;;
    armv7* | armv7 | arm) echo 'armv7' ;;
    armv6* | armv6) echo 'armv6' ;;
    armv5* | armv5) echo 'armv5' ;;
    s390x) echo 's390x' ;;
    *) echo -e "${green}$(t arch_unsupported)${plain}" && rm -f install.sh && exit 1 ;;
    esac
}

echo "$(t arch_label "$(arch)")"

install_base() {
    case "${release}" in
    centos | almalinux | rocky | oracle)
        yum -y update && yum install -y -q wget curl tar tzdata
        ;;
    fedora)
        dnf -y update && dnf install -y -q wget curl tar tzdata
        ;;
    arch | manjaro | parch)
        pacman -Syu && pacman -Syu --noconfirm wget curl tar tzdata
        ;;
    opensuse-tumbleweed)
        zypper refresh && zypper -q install -y wget curl tar timezone
        ;;
    *)
        apt-get update && apt-get install -y -q wget curl tar tzdata
        ;;
    esac
}

read_secretbox_key_file() {
    [[ -f "${SECRETBOX_ENV_FILE}" ]] || return 1

    local line
    local secretbox_key
    while IFS= read -r line; do
        case "${line}" in
            SUI_SECRETBOX_KEY=*)
                secretbox_key="${line#SUI_SECRETBOX_KEY=}"
                if [[ -n "${secretbox_key}" ]]; then
                    printf '%s' "${secretbox_key}"
                    return 0
                fi
                ;;
        esac
    done <"${SECRETBOX_ENV_FILE}"
    return 1
}

write_secretbox_key_file() {
    local secretbox_key="$1"

    mkdir -p "${SECRETBOX_ENV_DIR}"
    if [[ -f "${SECRETBOX_ENV_FILE}" ]]; then
        printf '\nSUI_SECRETBOX_KEY=%s\n' "${secretbox_key}" >>"${SECRETBOX_ENV_FILE}"
    else
        (umask 077 && printf 'SUI_SECRETBOX_KEY=%s\n' "${secretbox_key}" >"${SECRETBOX_ENV_FILE}")
    fi
    chmod 600 "${SECRETBOX_ENV_FILE}"
}

prepare_secretbox_key() {
    local secretbox_key
    local generated_key="false"

    if secretbox_key=$(read_secretbox_key_file); then
        chmod 600 "${SECRETBOX_ENV_FILE}"
        export SUI_SECRETBOX_KEY="${secretbox_key}"
    else
        if [[ -n "${SUI_SECRETBOX_KEY:-}" ]]; then
            secretbox_key="${SUI_SECRETBOX_KEY}"
        else
            secretbox_key=$(head -c 32 /dev/urandom | base64 | tr -d '\r\n')
            generated_key="true"
        fi
        write_secretbox_key_file "${secretbox_key}"
        export SUI_SECRETBOX_KEY="${secretbox_key}"

        if [[ "${generated_key}" == "true" ]]; then
            echo -e "###############################################"
            echo -e "${yellow}$(t secretbox_key_generated)${plain}"
            echo -e "${green}$(t secretbox_key_label "${secretbox_key}")${plain}"
            echo -e "$(t secretbox_key_file "${SECRETBOX_ENV_FILE}")"
            echo -e "${red}$(t secretbox_key_keep)${plain}"
            echo -e "###############################################"
        fi
    fi

    mkdir -p "${SECRETBOX_DROPIN_DIR}"
    printf '[Service]\nEnvironmentFile=-%s\n' "${SECRETBOX_ENV_FILE}" >"${SECRETBOX_DROPIN_FILE}"
    chmod 644 "${SECRETBOX_DROPIN_FILE}"
}

read_env_key_file() {
    local var="$1"
    [[ -f "${SECRETBOX_ENV_FILE}" ]] || return 1

    local line value
    while IFS= read -r line; do
        case "${line}" in
            "${var}="*)
                value="${line#"${var}"=}"
                if [[ -n "${value}" ]]; then
                    printf '%s' "${value}"
                    return 0
                fi
                ;;
        esac
    done <"${SECRETBOX_ENV_FILE}"
    return 1
}

append_env_key_file() {
    local var="$1" value="$2"

    mkdir -p "${SECRETBOX_ENV_DIR}"
    if [[ -f "${SECRETBOX_ENV_FILE}" ]]; then
        printf '\n%s=%s\n' "${var}" "${value}" >>"${SECRETBOX_ENV_FILE}"
    else
        (umask 077 && printf '%s=%s\n' "${var}" "${value}" >"${SECRETBOX_ENV_FILE}")
    fi
    chmod 600 "${SECRETBOX_ENV_FILE}"
}

# Generates the session-cookie signing key once. An existing SUI_COOKIE_KEY is
# never touched (key rotation is an explicit operator action — s-ui menu
# item 23), so re-running the installer for updates is a no-op here and never
# prompts, which keeps non-interactive installs working.
prepare_cookie_key() {
    local cookie_key

    if cookie_key=$(read_env_key_file SUI_COOKIE_KEY); then
        chmod 600 "${SECRETBOX_ENV_FILE}"
        return 0
    fi

    if [[ -n "${SUI_COOKIE_KEY:-}" ]]; then
        cookie_key="${SUI_COOKIE_KEY}"
        append_env_key_file SUI_COOKIE_KEY "${cookie_key}"
        return 0
    fi

    cookie_key=$(head -c 32 /dev/urandom | base64 | tr -d '\r\n')
    append_env_key_file SUI_COOKIE_KEY "${cookie_key}"

    echo -e "###############################################"
    echo -e "${yellow}$(t cookie_key_generated)${plain}"
    echo -e "${green}$(t cookie_key_label "${cookie_key}")${plain}"
    echo -e "$(t secretbox_key_file "${SECRETBOX_ENV_FILE}")"
    echo -e "${red}$(t secretbox_key_keep)${plain}"
    echo -e "${yellow}$(t cookie_key_relogin)${plain}"
    echo -e "###############################################"
}

config_after_install() {
    echo -e "${yellow}$(t migrate)${plain}"
    /usr/local/s-ui/sui migrate

    echo -e "${yellow}$(t install_done)${plain}"
    read -rp "$(t continue_settings)" config_confirm
    if [[ "${config_confirm}" == "y" || "${config_confirm}" == "Y" ]]; then
        echo -e "$(t enter_panel_port)"
        read -r config_port
        echo -e "$(t enter_panel_path)"
        read -r config_path

        echo -e "$(t enter_sub_port)"
        read -r config_subPort
        echo -e "$(t enter_sub_path)"
        read -r config_subPath

        echo -e "${yellow}$(t initializing)${plain}"
        params=""
        [ -z "$config_port" ] || params="$params -port $config_port"
        [ -z "$config_path" ] || params="$params -path $config_path"
        [ -z "$config_subPort" ] || params="$params -subPort $config_subPort"
        [ -z "$config_subPath" ] || params="$params -subPath $config_subPath"
        /usr/local/s-ui/sui setting ${params}

        read -rp "$(t change_admin)" admin_confirm
        if [[ "${admin_confirm}" == "y" || "${admin_confirm}" == "Y" ]]; then
            read -rp "$(t set_username)" config_account
            read -rp "$(t set_password)" config_password

            echo -e "${yellow}$(t initializing)${plain}"
            /usr/local/s-ui/sui admin -username "${config_account}" -password "${config_password}"
        else
            echo -e "${yellow}$(t current_admin)${plain}"
            /usr/local/s-ui/sui admin -show
        fi
    else
        echo -e "${red}$(t cancelled)${plain}"
        if [[ ! -f "/usr/local/s-ui/db/s-ui.db" ]]; then
            local usernameTemp
            local passwordTemp
            usernameTemp=$(head -c 6 /dev/urandom | base64)
            passwordTemp=$(head -c 6 /dev/urandom | base64)
            echo -e "$(t fresh_install_creds)"
            echo -e "###############################################"
            echo -e "${green}$(t username_label "${usernameTemp}")${plain}"
            echo -e "${green}$(t password_label "${passwordTemp}")${plain}"
            echo -e "###############################################"
            echo -e "${red}$(t lost_creds)${plain}"
            /usr/local/s-ui/sui admin -username "${usernameTemp}" -password "${passwordTemp}"
        else
            echo -e "${red}$(t upgrade_keep_settings)${plain}"
        fi
    fi
}

prepare_services() {
    if [[ -f "/etc/systemd/system/sing-box.service" ]]; then
        echo -e "${yellow}$(t stop_singbox)${plain}"
        systemctl stop sing-box
        rm -f /usr/local/s-ui/bin/sing-box /usr/local/s-ui/bin/runSingbox.sh /usr/local/s-ui/bin/signal
    fi
    if [[ -e "/usr/local/s-ui/bin" ]]; then
        echo -e "###############################################################"
        echo -e "${red}$(t bin_dir_exists)${plain}"
        echo -e "###############################################################"
    fi
    systemctl daemon-reload
}

verify_download_checksum() {
    local artifact_name="$1"
    local checksum_url="$2"
    local checksum_name="${artifact_name}.sha256"

    wget -N --timeout=20 --tries=5 --retry-connrefused -O "/tmp/${checksum_name}" "${checksum_url}"
    if [[ $? -ne 0 ]]; then
        echo -e "${red}$(t checksum_failed)${plain}"
        exit 1
    fi
    if ! (cd /tmp/ && sha256sum -c "${checksum_name}"); then
        echo -e "${red}$(t checksum_failed)${plain}"
        exit 1
    fi
}

install_s-ui() {
    cd /tmp/
    artifact_name="s-ui-linux-$(arch).tar.gz"

    if [[ $# -eq 0 || -z "${1:-}" ]]; then
        last_version=$(curl -Ls "https://api.github.com/repos/deposist/s-ui-x-extended/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [[ ! -n "$last_version" ]]; then
            echo -e "${red}$(t rate_limited)${plain}"
            exit 1
        fi
        echo -e "$(t fetching_latest "${last_version}")"
        url="https://github.com/deposist/s-ui-x-extended/releases/download/${last_version}/${artifact_name}"
        wget -N --timeout=20 --tries=5 --retry-connrefused -O "/tmp/${artifact_name}" "${url}"
        if [[ $? -ne 0 ]]; then
            echo -e "${red}$(t download_failed)${plain}"
            exit 1
        fi
        verify_download_checksum "${artifact_name}" "${url}.sha256"
    else
        last_version=$1
        [[ "${last_version}" != v* ]] && last_version="v${last_version}"
        url="https://github.com/deposist/s-ui-x-extended/releases/download/${last_version}/${artifact_name}"
        echo -e "$(t installing_specific "${last_version}")"
        wget -N --timeout=20 --tries=5 --retry-connrefused -O "/tmp/${artifact_name}" "${url}"
        if [[ $? -ne 0 ]]; then
            echo -e "${red}$(t download_failed_specific "${last_version}")${plain}"
            exit 1
        fi
        verify_download_checksum "${artifact_name}" "${url}.sha256"
    fi

    if [[ -e /usr/local/s-ui/ ]]; then
        systemctl stop s-ui
    fi

    tar zxvf "${artifact_name}"
    rm "${artifact_name}" "${artifact_name}.sha256" -f

    chmod +x s-ui/sui s-ui/s-ui.sh
    cp s-ui/s-ui.sh /usr/bin/s-ui
    cp -rf s-ui /usr/local/
    cp -f s-ui/*.service /etc/systemd/system/
    rm -rf s-ui

    prepare_secretbox_key
    prepare_cookie_key
    config_after_install
    prepare_services

    systemctl enable s-ui --now

    echo -e "${green}$(t installed_running "${last_version}")${plain}"
    echo -e "$(t panel_url)${green}"
    /usr/local/s-ui/sui uri
    echo -e "${plain}"
    echo -e ""
    s-ui help
}

echo -e "${green}$(t running)${plain}"
install_base
install_s-ui "$@"
