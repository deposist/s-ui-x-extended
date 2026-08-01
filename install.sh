#!/bin/bash
# S-UI installer with multilingual UI (English / Russian / Chinese).
# Language choice can be supplied non-interactively via env:
#   SUI_LANG=en|ru|zh  bash install.sh ...
# A version tag (e.g. "v1.4.2-beta") may be provided as the only positional
set -Eeo pipefail
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
        saved=$(tr -d '[:space:]' < "${LANG_FILE}" 2>/dev/null)
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
            awg_key_generated) echo "已生成 AWG 设备密钥加密密钥。该值只显示一次："; return ;;
            awg_key_label) echo "AWG_KEY_ENC：$2"; return ;;
            awg_key_keep) echo "请妥善保存该密钥。如果丢失，已创建设备的配置将无法解密。"; return ;;
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
        en:awg_key_generated) echo "Generated the AWG device key-encryption key. It is shown once:";;
        ru:awg_key_generated) echo "Сгенерирован ключ шифрования ключей устройств AWG. Он показывается один раз:";;
        en:awg_key_label) echo "AWG_KEY_ENC: $2";;
        ru:awg_key_label) echo "AWG_KEY_ENC: $2";;
        en:awg_key_keep) echo "Keep this key. If it is lost, the configs of devices already created cannot be decrypted.";;
        ru:awg_key_keep) echo "Сохраните этот ключ. Если он потеряется, конфиги уже созданных устройств нельзя будет расшифровать.";;
        *) echo "${key}";;
    esac
}

ask_language

[[ $EUID -ne 0 ]] && echo -e "${red}$(t run_as_root)${plain}\n" && exit 1


if [[ -f /etc/os-release ]]; then
    # Standard system file, present only on target Linux hosts.
    # shellcheck disable=SC1091
    source /etc/os-release
    release=$ID
elif [[ -f /usr/lib/os-release ]]; then
    # Standard system file, present only on target Linux hosts.
    # shellcheck disable=SC1091
    source /usr/lib/os-release
    release=$ID
else
    t detect_failed >&2
    exit 1
fi
t current_release "${release}"

arch() {
    case "$(uname -m)" in
    x86_64 | x64 | amd64) echo 'amd64' ;;
    i*86 | x86) echo '386' ;;
    armv8* | arm64 | aarch64) echo 'arm64' ;;
    armv7* | arm) echo 'armv7' ;;
    armv6*) echo 'armv6' ;;
    armv5*) echo 'armv5' ;;
    s390x) echo 's390x' ;;
    *) echo -e "${green}$(t arch_unsupported)${plain}" && rm -f install.sh && exit 1 ;;
    esac
}

t arch_label "$(arch)"

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

# Generates the AWG device key-encryption key once. It seals the per-device
# WireGuard private and preshared keys at rest (AWG_KEY_ENC in service/awg_crypto.go).
# An existing key is never regenerated: rotating it after devices exist makes
# their stored key material undecryptable, so re-running the installer for
# updates is a no-op here.
prepare_awg_key() {
    local awg_key

    if awg_key=$(read_env_key_file AWG_KEY_ENC); then
        chmod 600 "${SECRETBOX_ENV_FILE}"
        return 0
    fi

    if [[ -n "${AWG_KEY_ENC:-}" ]]; then
        awg_key="${AWG_KEY_ENC}"
        append_env_key_file AWG_KEY_ENC "${awg_key}"
        return 0
    fi

    awg_key=$(head -c 32 /dev/urandom | base64 | tr -d '\r\n')
    append_env_key_file AWG_KEY_ENC "${awg_key}"

    echo -e "###############################################"
    echo -e "${yellow}$(t awg_key_generated)${plain}"
    echo -e "${green}$(t awg_key_label "${awg_key}")${plain}"
    echo -e "$(t secretbox_key_file "${SECRETBOX_ENV_FILE}")"
    echo -e "${red}$(t awg_key_keep)${plain}"
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
        params=()
        [ -z "$config_port" ] || params+=("-port" "$config_port")
        [ -z "$config_path" ] || params+=("-path" "$config_path")
        [ -z "$config_subPort" ] || params+=("-subPort" "$config_subPort")
        [ -z "$config_subPath" ] || params+=("-subPath" "$config_subPath")
        /usr/local/s-ui/sui setting "${params[@]}"

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
    # Legacy runtime assets are preserved with the live tree.  Do not remove
    # them here: any cleanup which is not part of the transaction would make a
    # later rollback incomplete.
    if [[ -e "/usr/local/s-ui/bin" ]]; then
        echo -e "###############################################################"
        echo -e "${red}$(t bin_dir_exists)${plain}"
        echo -e "###############################################################"
    fi
}

download_file() {
    local url="$1"
    local destination="$2"
    local temporary="${destination}.part"
    local attempt

    case "${url}" in
        https://*) ;;
        *) return 1 ;;
    esac

    rm -f -- "${temporary}" "${destination}"
    for attempt in 1 2 3 4 5; do
        if curl --proto '=https' --proto-redir '=https' --tlsv1.2 --fail --location \
            --silent --show-error --connect-timeout 20 \
            --speed-limit 1024 --speed-time 60 \
            --output "${temporary}" "${url}"; then
            mv -f -- "${temporary}" "${destination}"
            return 0
        fi
        rm -f -- "${temporary}" "${destination}"
        [[ ${attempt} -eq 5 ]] || sleep 2
    done
    return 1
}

verify_download_checksum() {
    local artifact_path="$1"
    local checksum_path="$2"
    local artifact_name="$3"
    local line digest suffix actual

    [[ -f "${artifact_path}" && ! -L "${artifact_path}" && -s "${artifact_path}" ]] || return 1
    [[ -f "${checksum_path}" && ! -L "${checksum_path}" && -s "${checksum_path}" ]] || return 1
    # Release checksums are one newline-terminated sha256sum record.  Requiring
    # the exact basename prevents a valid digest from authorizing another file.
    [[ $(wc -l <"${checksum_path}") -eq 1 ]] || return 1
    IFS= read -r line <"${checksum_path}" || return 1
    [[ ${#line} -eq $((66 + ${#artifact_name})) ]] || return 1
    digest=${line:0:64}
    suffix=${line:64}
    [[ "${digest}" =~ ^[[:xdigit:]]{64}$ ]] || return 1
    [[ "${suffix}" == "  ${artifact_name}" || "${suffix}" == " *${artifact_name}" ]] || return 1

    actual=$(sha256sum -- "${artifact_path}") || return 1
    actual=${actual%% *}
    [[ "${actual,,}" == "${digest,,}" ]]
}

validate_archive_paths() {
    local archive_path="$1"
    local list_path="$2"
    local verbose_path="$3"
    local member line type duplicates

    TAR_OPTIONS= LC_ALL=C tar --quoting-style=escape --list --gzip \
        --file "${archive_path}" >"${list_path}" || return 1
    [[ -s "${list_path}" ]] || return 1

    while IFS= read -r member; do
        [[ -n "${member}" ]] || return 1
        # This deliberately accepts a small, portable path alphabet.  In
        # particular, escaped controls/backslashes, absolute paths, dot
        # components and paths outside the single s-ui root are rejected.
        [[ "${member}" != *'\\'* ]] || return 1
        [[ "${member}" == "s-ui" || "${member}" == "s-ui/" || "${member}" == s-ui/* ]] || return 1
        [[ "${member}" =~ ^s-ui(/[-+._A-Za-z0-9]+)*/?$ ]] || return 1
        [[ "${member}" != */. && "${member}" != */.. ]] || return 1
        [[ "/${member}/" != *'/../'* && "/${member}/" != *'/./'* && "${member}" != *'//'* ]] || return 1
    done <"${list_path}"

    duplicates=$(LC_ALL=C sort "${list_path}" | uniq -d) || return 1
    [[ -z "${duplicates}" ]] || return 1
    [[ $(grep -Fxc 's-ui/sui' "${list_path}") -eq 1 ]] || return 1
    [[ $(grep -Fxc 's-ui/s-ui.sh' "${list_path}") -eq 1 ]] || return 1
    [[ $(grep -Fxc 's-ui/s-ui.service' "${list_path}") -eq 1 ]] || return 1

    TAR_OPTIONS= LC_ALL=C tar --quoting-style=escape --list --verbose --gzip \
        --file "${archive_path}" >"${verbose_path}" || return 1
    while IFS= read -r line; do
        [[ -n "${line}" ]] || return 1
        type=${line:0:1}
        [[ "${type}" == '-' || "${type}" == 'd' ]] || return 1
    done <"${verbose_path}"
}

verify_staged_tree() {
    local tree="$1"

    [[ -d "${tree}" && ! -L "${tree}" ]] || return 1
    [[ -f "${tree}/sui" && ! -L "${tree}/sui" && -s "${tree}/sui" ]] || return 1
    [[ -f "${tree}/s-ui.sh" && ! -L "${tree}/s-ui.sh" && -s "${tree}/s-ui.sh" ]] || return 1
    [[ -f "${tree}/s-ui.service" && ! -L "${tree}/s-ui.service" && -s "${tree}/s-ui.service" ]] || return 1
    chmod 755 "${tree}/sui" "${tree}/s-ui.sh" || return 1
    bash -n "${tree}/s-ui.sh" || return 1
    grep -Fqx 'WorkingDirectory=/usr/local/s-ui/' "${tree}/s-ui.service" || return 1
    grep -Fqx 'ExecStart=/usr/local/s-ui/sui' "${tree}/s-ui.service" || return 1
    "${tree}/sui" -v >/dev/null 2>&1
}

copy_preserved_tree() {
    local live="$1" candidate="$2" entry

    [[ -d "${live}" && ! -L "${live}" ]] || return 1
    for entry in "${live}"/* "${live}"/.[!.]* "${live}"/..?*; do
        path_exists "${entry}" || continue
        case "${entry##*/}" in
            sui|s-ui.sh|s-ui.service) continue ;;
        esac
        cp -a -- "${entry}" "${candidate}/"
    done
}
path_exists() {
    [[ -e "$1" || -L "$1" ]]
}

install_transaction_cleanup() {
    local path
    set +e
    for path in "${MENU_NEW:-}" "${UNIT_NEW:-}" "${CONFIG_NEW:-}" "${DROPIN_NEW:-}"; do
        [[ -n "${path}" ]] && rm -rf -- "${path}"
    done
    [[ -n "${INSTALL_STAGE:-}" ]] && rm -rf -- "${INSTALL_STAGE}"
}

restore_promoted_path() {
    local live="$1" backup="$2" failed="$3" source="$4" had_old="$5" state="$6"

    if path_exists "${backup}"; then
        if path_exists "${live}"; then
            rm -rf -- "${failed}"
            mv -- "${live}" "${failed}" || return 1
        fi
        mv -- "${backup}" "${live}" || return 1
        rm -rf -- "${failed}"
        return 0
    fi

    # No backup means either this was a fresh path or the old rename never
    # happened.  Only remove a fresh promoted object when its source vanished.
    if [[ "${had_old}" == 0 && "${state}" != untouched ]] && \
        ! path_exists "${source}" && path_exists "${live}"; then
        rm -rf -- "${failed}"
        mv -- "${live}" "${failed}" || return 1
        rm -rf -- "${failed}"
    fi
}

restore_service_state() {
    local failed=0

    systemctl daemon-reload || failed=1
    systemctl disable s-ui >/dev/null 2>&1 || failed=1
    if [[ "${SERVICE_WAS_ENABLED:-0}" == 1 ]]; then
        systemctl enable "${SERVICE_ENABLED_UNIT:-s-ui.service}" || failed=1
    fi
    if [[ "${SERVICE_WAS_ACTIVE:-0}" == 1 ]]; then
        systemctl start s-ui || failed=1
        systemctl is-active --quiet s-ui || failed=1
    else
        systemctl stop s-ui >/dev/null 2>&1 || failed=1
    fi
    return "${failed}"
}

rollback_install_transaction() {
    local failed=0
    set +e

    systemctl stop s-ui >/dev/null 2>&1 || true
    restore_promoted_path "${DROPIN_PATH}" "${DROPIN_BACKUP}" "${DROPIN_FAILED}" \
        "${DROPIN_NEW}" "${DROPIN_HAD_OLD}" "${DROPIN_STATE}" || failed=1
    restore_promoted_path "${UNIT_PATH}" "${UNIT_BACKUP}" "${UNIT_FAILED}" \
        "${UNIT_NEW}" "${UNIT_HAD_OLD}" "${UNIT_STATE}" || failed=1
    restore_promoted_path "${CONFIG_PATH}" "${CONFIG_BACKUP}" "${CONFIG_FAILED}" \
        "${CONFIG_NEW}" "${CONFIG_HAD_OLD}" "${CONFIG_STATE}" || failed=1
    restore_promoted_path "${MENU_PATH}" "${MENU_BACKUP}" "${MENU_FAILED}" \
        "${MENU_NEW}" "${MENU_HAD_OLD}" "${MENU_STATE}" || failed=1
    restore_promoted_path "${LIVE_PATH}" "${LIVE_BACKUP}" "${LIVE_FAILED}" \
        "${TREE_NEW}" "${LIVE_HAD_OLD}" "${LIVE_STATE}" || failed=1
    restore_service_state || failed=1
    if [[ ${failed} -ne 0 ]]; then
        echo "s-ui rollback encountered an error; backups were left in place" >&2
    fi
    return "${failed}"
}

abort_install_transaction() {
    local status="$1"
    trap - ERR INT TERM
    set +e
    if [[ "${TRANSACTION_ACTIVE:-0}" == 1 ]]; then
        rollback_install_transaction
    fi
    install_transaction_cleanup
    exit "${status}"
}

promote_path() {
    local source="$1" live="$2" backup="$3" had_var="$4" state_var="$5"

    printf -v "${state_var}" '%s' pending
    if path_exists "${live}"; then
        printf -v "${had_var}" '%s' 1
        mv -- "${live}" "${backup}"
    fi
    mv -- "${source}" "${live}"
    printf -v "${state_var}" '%s' promoted
}

stage_config_and_service_files() {
    local extracted_tree="$1"

    MENU_NEW=$(mktemp '/usr/bin/.s-ui.install.XXXXXXXXXX')
    UNIT_NEW=$(mktemp '/etc/systemd/system/.s-ui.service.install.XXXXXXXXXX')
    CONFIG_NEW=$(mktemp -d '/etc/.s-ui.install.XXXXXXXXXX')
    DROPIN_NEW=$(mktemp -d '/etc/systemd/system/.s-ui.service.d.install.XXXXXXXXXX')

    cp -a -- "${extracted_tree}/s-ui.sh" "${MENU_NEW}"
    cp -a -- "${extracted_tree}/s-ui.service" "${UNIT_NEW}"
    chmod 755 "${MENU_NEW}"
    chmod 644 "${UNIT_NEW}"
    if path_exists "${CONFIG_PATH}"; then
        [[ -d "${CONFIG_PATH}" && ! -L "${CONFIG_PATH}" ]] || return 1
        cp -a -- "${CONFIG_PATH}/." "${CONFIG_NEW}/"
        chown --reference="${CONFIG_PATH}" "${CONFIG_NEW}"
        chmod --reference="${CONFIG_PATH}" "${CONFIG_NEW}"
    fi
    if path_exists "${DROPIN_PATH}"; then
        [[ -d "${DROPIN_PATH}" && ! -L "${DROPIN_PATH}" ]] || return 1
        cp -a -- "${DROPIN_PATH}/." "${DROPIN_NEW}/"
        chown --reference="${DROPIN_PATH}" "${DROPIN_NEW}"
        chmod --reference="${DROPIN_PATH}" "${DROPIN_NEW}"
    fi
    [[ ! -L "${CONFIG_NEW}/secretbox.env" && ! -L "${CONFIG_NEW}/lang" ]] || return 1

    # Point the existing key preparation helpers at private candidates.  The
    # live /etc tree is not changed until all release files have been verified.
    LANG_FILE="${CONFIG_NEW}/lang"
    SECRETBOX_ENV_DIR="${CONFIG_NEW}"
    SECRETBOX_ENV_FILE="${CONFIG_NEW}/secretbox.env"
    SECRETBOX_DROPIN_DIR="${DROPIN_NEW}"
    SECRETBOX_DROPIN_FILE="${DROPIN_NEW}/10-secretbox-env.conf"
    printf '%s\n' "${lang}" >"${LANG_FILE}"
    prepare_secretbox_key
    prepare_cookie_key
    prepare_awg_key

    LANG_FILE="/etc/s-ui/lang"
    SECRETBOX_ENV_DIR="/etc/s-ui"
    SECRETBOX_ENV_FILE="/etc/s-ui/secretbox.env"
    SECRETBOX_DROPIN_DIR="/etc/systemd/system/s-ui.service.d"
    SECRETBOX_DROPIN_FILE="${SECRETBOX_DROPIN_DIR}/10-secretbox-env.conf"

    [[ -s "${MENU_NEW}" && -s "${UNIT_NEW}" ]] || return 1
    [[ -f "${CONFIG_NEW}/secretbox.env" && ! -L "${CONFIG_NEW}/secretbox.env" ]] || return 1
    [[ -f "${DROPIN_NEW}/10-secretbox-env.conf" && ! -L "${DROPIN_NEW}/10-secretbox-env.conf" ]] || return 1
    bash -n "${MENU_NEW}"
}

record_service_state() {
    SERVICE_WAS_ACTIVE=0
    SERVICE_WAS_ENABLED=0
    SERVICE_ENABLED_UNIT='s-ui.service'
    systemctl is-active --quiet s-ui 2>/dev/null && SERVICE_WAS_ACTIVE=1 || true
    if SERVICE_ENABLED_UNIT=$(systemctl is-enabled s-ui 2>/dev/null); then
        SERVICE_WAS_ENABLED=1
        case "${SERVICE_ENABLED_UNIT}" in
            *.service) ;;
            *) SERVICE_ENABLED_UNIT='s-ui.service' ;;
        esac
    else
        SERVICE_ENABLED_UNIT='s-ui.service'
    fi
}

install_s-ui() {
    local artifact_name version url latest_file archive_path checksum_path
    local archive_list archive_verbose extracted_tree token desired_active desired_enabled

    umask 077
    mkdir -p /usr/local /usr/bin /etc/systemd/system
    tar --version 2>/dev/null | grep -q 'GNU tar' || {
        echo 'GNU tar is required for secure archive validation' >&2
        return 1
    }
    INSTALL_STAGE=$(mktemp -d '/usr/local/.s-ui-install.XXXXXXXXXX')
    chmod 700 "${INSTALL_STAGE}"
    trap install_transaction_cleanup EXIT
    trap 'abort_install_transaction $?' ERR
    trap 'abort_install_transaction 130' INT
    trap 'abort_install_transaction 143' TERM

    LIVE_PATH='/usr/local/s-ui'
    MENU_PATH='/usr/bin/s-ui'
    UNIT_PATH='/etc/systemd/system/s-ui.service'
    CONFIG_PATH='/etc/s-ui'
    DROPIN_PATH='/etc/systemd/system/s-ui.service.d'
    token=${INSTALL_STAGE##*/}
    LIVE_BACKUP="/usr/local/.s-ui.backup.${token}"
    MENU_BACKUP="/usr/bin/.s-ui.backup.${token}"
    UNIT_BACKUP="/etc/systemd/system/.s-ui.service.backup.${token}"
    CONFIG_BACKUP="/etc/.s-ui.backup.${token}"
    DROPIN_BACKUP="/etc/systemd/system/.s-ui.service.d.backup.${token}"
    LIVE_FAILED="/usr/local/.s-ui.failed.${token}"
    MENU_FAILED="/usr/bin/.s-ui.failed.${token}"
    UNIT_FAILED="/etc/systemd/system/.s-ui.service.failed.${token}"
    CONFIG_FAILED="/etc/.s-ui.failed.${token}"
    DROPIN_FAILED="/etc/systemd/system/.s-ui.service.d.failed.${token}"
    LIVE_HAD_OLD=0 MENU_HAD_OLD=0 UNIT_HAD_OLD=0 CONFIG_HAD_OLD=0 DROPIN_HAD_OLD=0
    LIVE_STATE=untouched MENU_STATE=untouched UNIT_STATE=untouched CONFIG_STATE=untouched DROPIN_STATE=untouched
    TRANSACTION_ACTIVE=0

    artifact_name="s-ui-linux-$(arch).tar.gz"
    latest_file="${INSTALL_STAGE}/latest.json"
    if [[ $# -eq 0 || -z "${1:-}" ]]; then
        download_file 'https://api.github.com/repos/deposist/s-ui-x-extended/releases/latest' "${latest_file}" || {
            echo -e "${red}$(t rate_limited)${plain}"
            return 1
        }
        version=$(sed -n -E 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/p' "${latest_file}" | sed -n '1p')
        [[ -n "${version}" ]] || return 1
        echo -e "$(t fetching_latest "${version}")"
    else
        version=$1
        [[ "${version}" == v* ]] || version="v${version}"
        echo -e "$(t installing_specific "${version}")"
    fi
    [[ "${version}" =~ ^v[0-9]+[.][0-9]+[.][0-9]+(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$ ]] || return 1

    url="https://github.com/deposist/s-ui-x-extended/releases/download/${version}/${artifact_name}"
    archive_path="${INSTALL_STAGE}/${artifact_name}"
    checksum_path="${INSTALL_STAGE}/${artifact_name}.sha256"
    if ! download_file "${url}" "${archive_path}"; then
        echo -e "${red}$(t download_failed_specific "${version}")${plain}"
        return 1
    fi
    if ! download_file "${url}.sha256" "${checksum_path}" || \
        ! verify_download_checksum "${archive_path}" "${checksum_path}" "${artifact_name}"; then
        echo -e "${red}$(t checksum_failed)${plain}"
        return 1
    fi

    archive_list="${INSTALL_STAGE}/archive.list"
    archive_verbose="${INSTALL_STAGE}/archive.verbose"
    validate_archive_paths "${archive_path}" "${archive_list}" "${archive_verbose}" || {
        echo -e "${red}$(t checksum_failed)${plain}"
        return 1
    }
    mkdir "${INSTALL_STAGE}/extract"
    TAR_OPTIONS= tar --extract --gzip --file "${archive_path}" \
        --directory "${INSTALL_STAGE}/extract" --no-same-owner --no-same-permissions
    extracted_tree="${INSTALL_STAGE}/extract/s-ui"
    verify_staged_tree "${extracted_tree}"

    # Only after the complete release is runnable and structurally verified do
    # we inspect/stop the service or prepare any live-path promotion.
    record_service_state
    if path_exists "${LIVE_PATH}"; then
        [[ -d "${LIVE_PATH}" && ! -L "${LIVE_PATH}" ]] || return 1
        desired_active=${SERVICE_WAS_ACTIVE}
        desired_enabled=${SERVICE_WAS_ENABLED}
    else
        # A fresh install starts/enables on commit, but rollback must restore
        # the pre-install (absent, stopped, disabled) service state.
        desired_active=1
        desired_enabled=1
        SERVICE_WAS_ACTIVE=0
        SERVICE_WAS_ENABLED=0
        SERVICE_ENABLED_UNIT='s-ui.service'
    fi
    stage_config_and_service_files "${extracted_tree}"

    TREE_NEW="${INSTALL_STAGE}/live-tree"
    mkdir "${TREE_NEW}"
    if path_exists "${LIVE_PATH}"; then
        copy_preserved_tree "${LIVE_PATH}" "${TREE_NEW}"
    fi
    cp -a -- "${extracted_tree}/." "${TREE_NEW}/"
    verify_staged_tree "${TREE_NEW}"

    TRANSACTION_ACTIVE=1
    if [[ "${SERVICE_WAS_ACTIVE}" == 1 ]]; then
        systemctl stop s-ui
    fi
    prepare_services

    promote_path "${TREE_NEW}" "${LIVE_PATH}" "${LIVE_BACKUP}" LIVE_HAD_OLD LIVE_STATE
    promote_path "${MENU_NEW}" "${MENU_PATH}" "${MENU_BACKUP}" MENU_HAD_OLD MENU_STATE
    promote_path "${CONFIG_NEW}" "${CONFIG_PATH}" "${CONFIG_BACKUP}" CONFIG_HAD_OLD CONFIG_STATE
    promote_path "${UNIT_NEW}" "${UNIT_PATH}" "${UNIT_BACKUP}" UNIT_HAD_OLD UNIT_STATE
    promote_path "${DROPIN_NEW}" "${DROPIN_PATH}" "${DROPIN_BACKUP}" DROPIN_HAD_OLD DROPIN_STATE

    systemctl daemon-reload
    config_after_install

    systemctl disable s-ui >/dev/null 2>&1 || true
    if [[ "${desired_enabled}" == 1 ]]; then
        systemctl enable "${SERVICE_ENABLED_UNIT:-s-ui.service}"
    fi
    if [[ "${desired_active}" == 1 ]]; then
        systemctl start s-ui
        systemctl is-active --quiet s-ui
    fi

    # Backups are retained until after the transaction is marked committed.
    # Cleanup is best effort and must never re-enter rollback after success.
    rm -rf -- "${LIVE_BACKUP}" "${MENU_BACKUP}" "${CONFIG_BACKUP}" \
        "${UNIT_BACKUP}" "${DROPIN_BACKUP}" || true
    TRANSACTION_ACTIVE=0
    trap - ERR INT TERM

    echo -e "${green}$(t installed_running "${version}")${plain}"
    if [[ "${desired_active}" == 1 ]]; then
        echo -e "$(t panel_url)${green}"
        /usr/local/s-ui/sui uri || true
        echo -e "${plain}"
    fi
    /usr/bin/s-ui help || true
}

echo -e "${green}$(t running)${plain}"
install_base
install_s-ui "$@"
