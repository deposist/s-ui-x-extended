#!/bin/bash
# S-UI management menu with multilingual UI (English / Russian / Chinese).
# Language is persisted in /etc/s-ui/lang and can be switched from
# menu item 22.
set -eo pipefail

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
plain='\033[0m'

LANG_FILE="/etc/s-ui/lang"
SECRETBOX_ENV_FILE="/etc/s-ui/secretbox.env"
SECRETBOX_DROPIN_DIR="/etc/systemd/system/s-ui.service.d"
SECRETBOX_DROPIN_FILE="${SECRETBOX_DROPIN_DIR}/10-secretbox-env.conf"

load_language() {
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
    lang="en"
}

save_language() {
    mkdir -p "$(dirname "${LANG_FILE}")"
    printf '%s\n' "${lang}" >"${LANG_FILE}" 2>/dev/null || true
}

t() {
    local key="$1"
    if [[ "${lang}" == "zh" ]]; then
        case "${key}" in
            run_as_root)         echo "错误：必须使用 root 权限运行此脚本！"; return ;;
            detect_failed)       echo "检测系统失败，请联系作者！"; return ;;
            current_release)     echo "当前系统发行版为：$2"; return ;;
            debug_tag)           echo "[调试]"; return ;;
            error_tag)           echo "[错误]"; return ;;
            info_tag)            echo "[信息]"; return ;;
            default_n)           echo "默认$2"; return ;;
            press_enter_main)    echo "按回车返回主菜单："; return ;;
            restart_service_q)   echo "重启 $2 服务"; return ;;
            install_force_q)     echo "此功能将强制重装最新版本，数据不会丢失。是否继续？"; return ;;
            cancelled)           echo "已取消"; return ;;
            update_done)         echo "更新完成，面板已自动重启"; return ;;
            enter_panel_version) echo "请输入面板版本（例如 v1.4.1）："; return ;;
            version_required)    echo "面板版本不能为空。正在退出。"; return ;;
            invalid_panel_version) echo "无效的面板版本：$2。请使用类似 v1.4.1 或 v1.0.0-beta1 的版本。"; return ;;
            downloading_version) echo "正在下载并安装面板版本 $2..."; return ;;
            uninstall_q)         echo "确定要卸载面板吗？"; return ;;
            uninstall_done)      echo "卸载成功。如果要删除此脚本，请在退出脚本后运行 rm /usr/local/s-ui -f。"; return ;;
            reset_admin_warn)    echo "不建议将管理员账号密码设置为默认值！"; return ;;
            reset_admin_q)       echo "确定要将管理员账号密码重置为默认值吗？"; return ;;
            set_admin_warn)      echo "不建议将管理员账号密码设置为过于复杂的文本。"; return ;;
            set_username_p)      echo "请设置用户名："; return ;;
            set_password_p)      echo "请设置密码："; return ;;
            reset_settings_q)    echo "确定要将设置重置为默认值吗？"; return ;;
            clear_domain_q)      echo "确定要清除面板的域名、监听地址、Web URI，并停止使用面板和订阅证书吗？"; return ;;
            enter_panel_port)    echo "请输入面板端口（留空则使用现有/默认值）："; return ;;
            enter_panel_path)    echo "请输入面板路径（留空则使用现有/默认值）："; return ;;
            enter_sub_port)      echo "请输入订阅端口（留空则使用现有/默认值）："; return ;;
            enter_sub_path)      echo "请输入订阅路径（留空则使用现有/默认值）："; return ;;
            initializing)        echo "正在初始化，请稍候..."; return ;;
            could_not_get_uri)   echo "获取当前 URI 失败"; return ;;
            panel_url)           echo "你可以通过以下 URL 访问面板："; return ;;
            already_running)     echo "$2 正在运行，无需再次启动；如果需要重启，请选择重启"; return ;;
            start_ok)            echo "$2 启动成功"; return ;;
            start_fail)          echo "启动 $2 失败，可能是启动时间超过两秒，请稍后查看日志信息"; return ;;
            already_stopped)     echo "$2 已停止，无需再次停止！"; return ;;
            stop_ok)             echo "$2 停止成功"; return ;;
            stop_fail)           echo "停止 $2 失败，可能是停止时间超过两秒，请稍后查看日志信息"; return ;;
            restart_ok)          echo "$2 重启成功"; return ;;
            restart_fail)        echo "重启 $2 失败，可能是启动时间超过两秒，请稍后查看日志信息"; return ;;
            enable_ok)           echo "已成功设置 $2 开机自启"; return ;;
            enable_fail)         echo "设置 $2 开机自启失败"; return ;;
            disable_ok)          echo "已成功取消 $2 开机自启"; return ;;
            disable_fail)        echo "取消 $2 开机自启失败"; return ;;
            download_fail)       echo "下载脚本失败，请检查当前机器是否可以连接 Github"; return ;;
            script_updated)      echo "脚本升级成功，请重新运行脚本"; return ;;
            already_installed)   echo "面板已安装，请勿重复安装"; return ;;
            install_first)       echo "请先安装面板"; return ;;
            status_running)      echo "$2 状态：运行中"; return ;;
            status_stopped)      echo "$2 状态：未运行"; return ;;
            status_missing)      echo "$2 状态：未安装"; return ;;
            autostart_yes)       echo "$2 开机自启：是"; return ;;
            autostart_no)        echo "$2 开机自启：否"; return ;;
            invalid_choice)      echo "无效选择"; return ;;

            enable_bbr)          echo "启用 BBR"; return ;;
            disable_bbr)         echo "禁用 BBR"; return ;;
            back_main)           echo "返回主菜单"; return ;;
            select_option)       echo "请选择一个选项："; return ;;
            bbr_already_off)     echo "当前未启用 BBR。"; return ;;
            bbr_to_cubic_ok)     echo "已成功将 BBR 替换为 CUBIC。"; return ;;
            bbr_to_cubic_fail)   echo "将 BBR 替换为 CUBIC 失败。请检查系统配置。"; return ;;
            bbr_already_on)      echo "BBR 已启用！"; return ;;
            bbr_enabled)         echo "BBR 启用成功。"; return ;;
            bbr_enable_fail)     echo "启用 BBR 失败。请检查系统配置。"; return ;;
            os_not_supported)    echo "不支持的操作系统。请检查脚本并手动安装必要的软件包。"; return ;;
            installing_acme)     echo "正在安装 acme..."; return ;;
            acme_install_fail)   echo "安装 acme 失败"; return ;;
            acme_install_ok)     echo "安装 acme 成功"; return ;;
            ssl_get)             echo "获取 SSL"; return ;;
            ssl_revoke)          echo "吊销证书"; return ;;
            ssl_force_renew)     echo "强制续签"; return ;;
            ssl_self_signed)     echo "自签名证书"; return ;;
            ssl_ip)              echo "为 IP 地址签发证书 (Let's Encrypt)"; return ;;

            menu_title)          echo "S-UI 管理脚本"; return ;;
            menu_exit)           echo "退出"; return ;;
            menu_install)        echo "安装"; return ;;
            menu_update)         echo "更新"; return ;;
            menu_custom_version) echo "自定义版本"; return ;;
            menu_uninstall)      echo "卸载"; return ;;
            menu_reset_admin)    echo "将管理员账号密码重置为默认值"; return ;;
            menu_set_admin)      echo "设置管理员账号密码"; return ;;
            menu_view_admin)     echo "查看管理员账号密码"; return ;;
            menu_reset_settings) echo "重置面板设置"; return ;;
            menu_set_settings)   echo "设置面板设置"; return ;;
            menu_view_settings)  echo "查看面板设置"; return ;;
            menu_clear_domain)   echo "清除面板域名和地址"; return ;;
            menu_start)          echo "启动 S-UI"; return ;;
            menu_stop)           echo "停止 S-UI"; return ;;
            menu_restart)        echo "重启 S-UI"; return ;;
            menu_status)         echo "查看 S-UI 状态"; return ;;
            menu_log)            echo "查看 S-UI 日志"; return ;;
            menu_enable_auto)    echo "启用 S-UI 开机自启"; return ;;
            menu_disable_auto)   echo "禁用 S-UI 开机自启"; return ;;
            menu_bbr)            echo "启用或禁用 BBR"; return ;;
            menu_ssl)            echo "SSL 证书管理"; return ;;
            menu_ssl_cf)         echo "Cloudflare SSL 证书"; return ;;
            menu_language)       echo "语言"; return ;;
            menu_env_keys)       echo "生成环境密钥"; return ;;
            env_keys_title)      echo "生成环境密钥"; return ;;
            env_keys_opt_cookie) echo "生成会话 Cookie 密钥（SUI_COOKIE_KEY）"; return ;;
            env_keys_opt_awg)    echo "生成 AmneziaWG 设备密钥（AWG_KEY_ENC）"; return ;;
            env_keys_choice)     echo "请输入你的选择 [0-2]："; return ;;
            awg_key_menu_exists) echo "已存在 AWG_KEY_ENC（当前：$2）"; return ;;
            awg_key_rotate_q)    echo "轮换密钥？警告：如果已创建 AWG 设备，轮换后其已存储的密钥材料将无法解密"; return ;;
            awg_key_none)        echo "尚未设置 AWG_KEY_ENC，将生成新密钥。"; return ;;
            awg_key_generated)   echo "已生成 AWG_KEY_ENC（仅显示一次）："; return ;;
            awg_key_keep)        echo "请妥善保存该密钥。如果丢失，已创建设备的配置将无法解密。"; return ;;
            awg_key_restart)     echo "密钥在服务重启后生效。"; return ;;
            cookie_key_exists)   echo "已存在 SUI_COOKIE_KEY（当前：$2）"; return ;;
            cookie_key_rotate_q) echo "轮换密钥？新密钥将签发新会话，旧密钥保留用于平滑过渡，现有会话不会被注销"; return ;;
            cookie_key_none)     echo "尚未设置 SUI_COOKIE_KEY，将生成新密钥。"; return ;;
            cookie_key_generated) echo "已生成 SUI_COOKIE_KEY（仅显示一次）："; return ;;
            cookie_key_file)     echo "密钥文件：$2"; return ;;
            cookie_key_keep)     echo "请保持该文件私密，并在更新与恢复时保留同一个值。"; return ;;
            cookie_key_restart_rollover) echo "密钥在服务重启后生效。旧密钥已保留用于平滑过渡，现有会话保持有效。"; return ;;
            cookie_key_restart_fresh) echo "密钥在服务重启后生效。使用旧后备密钥签发的会话将被注销一次，需要重新登录。"; return ;;
            menu_back)           echo "返回"; return ;;
            enter_choice_range)  echo "请输入你的选择 [0-23]："; return ;;
            enter_valid_number)  echo "请输入正确的数字 [0-23]"; return ;;
            lang_select)         echo "Select language / Выберите язык / 请选择语言"; return ;;
            lang_set_to)         echo "语言已设置为：$2"; return ;;

            usage_title)         echo "S-UI 控制菜单用法"; return ;;
            usage_main)          echo "s-ui              - 管理员管理脚本"; return ;;
            usage_start)         echo "s-ui start        - 启动 s-ui"; return ;;
            usage_stop)          echo "s-ui stop         - 停止 s-ui"; return ;;
            usage_restart)       echo "s-ui restart      - 重启 s-ui"; return ;;
            usage_status)        echo "s-ui status       - 查看当前 s-ui 状态"; return ;;
            usage_enable)        echo "s-ui enable       - 启用开机自启"; return ;;
            usage_disable)       echo "s-ui disable      - 禁用开机自启"; return ;;
            usage_log)           echo "s-ui log          - 查看 s-ui 日志"; return ;;
            usage_update)        echo "s-ui update       - 更新"; return ;;
            usage_install)       echo "s-ui install      - 安装"; return ;;
            usage_uninstall)     echo "s-ui uninstall    - 卸载"; return ;;
            usage_help)          echo "s-ui help         - 管理菜单帮助"; return ;;
        esac
    fi
    case "${lang}:${key}" in
        en:run_as_root)         echo "Error: this script must be run as root!";;
        ru:run_as_root)         echo "Ошибка: этот скрипт нужно запускать с правами root!";;
        en:detect_failed)       echo "Could not detect the system, please contact the maintainer!";;
        ru:detect_failed)       echo "Не удалось определить систему, обратитесь к автору!";;
        en:current_release)     echo "Detected distribution: $2";;
        ru:current_release)     echo "Текущий дистрибутив: $2";;
        en:debug_tag)           echo "[Debug]";;
        ru:debug_tag)           echo "[Отладка]";;
        en:error_tag)           echo "[Error]";;
        ru:error_tag)           echo "[Ошибка]";;
        en:info_tag)            echo "[Info]";;
        ru:info_tag)            echo "[Инфо]";;
        en:default_n)           echo "default $2";;
        ru:default_n)           echo "по умолчанию $2";;
        en:press_enter_main)    echo "Press Enter to return to the main menu: ";;
        ru:press_enter_main)    echo "Нажмите Enter, чтобы вернуться в главное меню: ";;
        en:restart_service_q)   echo "Restart the service $2?";;
        ru:restart_service_q)   echo "Перезапустить службу $2?";;
        en:install_force_q)     echo "This will force-reinstall the latest version. Data is preserved. Continue?";;
        ru:install_force_q)     echo "Эта функция принудительно переустановит последнюю версию. Данные не будут потеряны. Продолжить?";;
        en:cancelled)           echo "Cancelled";;
        ru:cancelled)           echo "Отменено";;
        en:update_done)         echo "Update complete; the panel has been restarted automatically";;
        ru:update_done)         echo "Обновление завершено, панель автоматически перезапущена";;
        en:enter_panel_version) echo "Enter the panel version (for example v1.4.1):";;
        ru:enter_panel_version) echo "Введите версию панели (например, v1.4.1):";;
        en:version_required)    echo "Panel version cannot be empty. Exiting.";;
        ru:version_required)    echo "Версия панели не может быть пустой. Выход.";;
        en:invalid_panel_version) echo "Invalid panel version: $2. Use a version like v1.4.1 or v1.0.0-beta1.";;
        ru:invalid_panel_version) echo "Недопустимая версия панели: $2. Используйте версию вида v1.4.1 или v1.0.0-beta1.";;
        en:downloading_version) echo "Downloading and installing panel $2...";;
        ru:downloading_version) echo "Скачивание и установка версии панели $2...";;
        en:uninstall_q)         echo "Are you sure you want to uninstall the panel?";;
        ru:uninstall_q)         echo "Вы уверены, что хотите удалить панель?";;
        en:uninstall_done)      echo "Uninstall complete. To remove this script run rm /usr/local/s-ui -f after exiting.";;
        ru:uninstall_done)      echo "Удаление завершено. Если нужно удалить этот скрипт, после выхода выполните rm /usr/local/s-ui -f.";;
        en:reset_admin_warn)    echo "Setting default admin credentials is not recommended!";;
        ru:reset_admin_warn)    echo "Не рекомендуется устанавливать учетные данные администратора по умолчанию!";;
        en:reset_admin_q)       echo "Reset admin credentials to default?";;
        ru:reset_admin_q)       echo "Сбросить учетные данные администратора к значениям по умолчанию?";;
        en:set_admin_warn)      echo "Avoid using overly complex strings for the admin credentials.";;
        ru:set_admin_warn)      echo "Не рекомендуется использовать слишком сложный текст для учетных данных администратора.";;
        en:set_username_p)      echo "Username: ";;
        ru:set_username_p)      echo "Имя пользователя: ";;
        en:set_password_p)      echo "Password: ";;
        ru:set_password_p)      echo "Пароль: ";;
        en:reset_settings_q)    echo "Reset settings to default values?";;
        ru:reset_settings_q)    echo "Сбросить настройки к значениям по умолчанию?";;
        en:clear_domain_q)      echo "Clear the panel domain, listen address and web URI, and stop using panel and subscription certificates?";;
        ru:clear_domain_q)      echo "Очистить домен, адрес и Web URI панели и отключить использование сертификатов панелью и подписками?";;
        en:enter_panel_port)    echo "Enter panel port (leave empty to keep current/default):";;
        ru:enter_panel_port)    echo "Введите порт панели (оставьте пустым, чтобы использовать текущее/стандартное значение):";;
        en:enter_panel_path)    echo "Enter panel path (leave empty to keep current/default):";;
        ru:enter_panel_path)    echo "Введите путь панели (оставьте пустым, чтобы использовать текущее/стандартное значение):";;
        en:enter_sub_port)      echo "Enter subscription port (leave empty to keep current/default):";;
        ru:enter_sub_port)      echo "Введите порт подписки (оставьте пустым, чтобы использовать текущее/стандартное значение):";;
        en:enter_sub_path)      echo "Enter subscription path (leave empty to keep current/default):";;
        ru:enter_sub_path)      echo "Введите путь подписки (оставьте пустым, чтобы использовать текущее/стандартное значение):";;
        en:initializing)        echo "Initializing, please wait...";;
        ru:initializing)        echo "Инициализация, подождите...";;
        en:could_not_get_uri)   echo "Could not retrieve the current URI";;
        ru:could_not_get_uri)   echo "Не удалось получить текущий URI";;
        en:panel_url)           echo "Panel is available at:";;
        ru:panel_url)           echo "Панель доступна по адресу:";;
        en:already_running)     echo "$2 is already running; restart it from the menu if needed.";;
        ru:already_running)     echo "$2 уже работает, повторный запуск не нужен; если нужно, выберите перезапуск.";;
        en:start_ok)            echo "$2 started successfully";;
        ru:start_ok)            echo "$2 успешно запущен";;
        en:start_fail)          echo "Could not start $2; the start may take more than two seconds, check the log later.";;
        ru:start_fail)          echo "Не удалось запустить $2; возможно, запуск занимает больше двух секунд. Проверьте журнал позже.";;
        en:already_stopped)     echo "$2 is already stopped.";;
        ru:already_stopped)     echo "$2 уже остановлен, повторная остановка не нужна.";;
        en:stop_ok)             echo "$2 stopped successfully";;
        ru:stop_ok)             echo "$2 успешно остановлен";;
        en:stop_fail)           echo "Could not stop $2; check the log later.";;
        ru:stop_fail)           echo "Не удалось остановить $2; проверьте журнал позже.";;
        en:restart_ok)          echo "$2 restarted successfully";;
        ru:restart_ok)          echo "$2 успешно перезапущен";;
        en:restart_fail)        echo "Could not restart $2; check the log later.";;
        ru:restart_fail)        echo "Не удалось перезапустить $2; проверьте журнал позже.";;
        en:enable_ok)           echo "Autostart for $2 enabled";;
        ru:enable_ok)           echo "Автозапуск $2 успешно включен";;
        en:enable_fail)         echo "Could not enable autostart for $2";;
        ru:enable_fail)         echo "Не удалось включить автозапуск $2";;
        en:disable_ok)          echo "Autostart for $2 disabled";;
        ru:disable_ok)          echo "Автозапуск $2 успешно отключен";;
        en:disable_fail)        echo "Could not disable autostart for $2";;
        ru:disable_fail)        echo "Не удалось отключить автозапуск $2";;
        en:download_fail)       echo "Could not download the script. Check that the server has GitHub access.";;
        ru:download_fail)       echo "Не удалось скачать скрипт. Проверьте, есть ли на сервере доступ к GitHub.";;
        en:script_updated)      echo "Script updated successfully, run it again";;
        ru:script_updated)      echo "Скрипт успешно обновлен, запустите его заново";;
        en:already_installed)   echo "Panel already installed; reinstallation is not required.";;
        ru:already_installed)   echo "Панель уже установлена, повторная установка не нужна.";;
        en:install_first)       echo "Install the panel first.";;
        ru:install_first)       echo "Сначала установите панель.";;
        en:status_running)      echo "$2 status: running";;
        ru:status_running)      echo "$2 статус: работает";;
        en:status_stopped)      echo "$2 status: stopped";;
        ru:status_stopped)      echo "$2 статус: не работает";;
        en:status_missing)      echo "$2 status: not installed";;
        ru:status_missing)      echo "$2 статус: не установлен";;
        en:autostart_yes)       echo "$2 autostart: yes";;
        ru:autostart_yes)       echo "$2 автозапуск: да";;
        en:autostart_no)        echo "$2 autostart: no";;
        ru:autostart_no)        echo "$2 автозапуск: нет";;
        en:invalid_choice)      echo "Invalid choice";;
        ru:invalid_choice)      echo "Недопустимый выбор";;

        en:enable_bbr)          echo "Enable BBR";;
        ru:enable_bbr)          echo "Включить BBR";;
        en:disable_bbr)         echo "Disable BBR";;
        ru:disable_bbr)         echo "Отключить BBR";;
        en:back_main)           echo "Return to the main menu";;
        ru:back_main)           echo "Вернуться в главное меню";;
        en:select_option)       echo "Select an option: ";;
        ru:select_option)       echo "Выберите пункт: ";;
        en:bbr_already_off)     echo "BBR is currently disabled.";;
        ru:bbr_already_off)     echo "BBR сейчас не включен.";;
        en:bbr_to_cubic_ok)     echo "BBR replaced with CUBIC successfully.";;
        ru:bbr_to_cubic_ok)     echo "BBR успешно заменен на CUBIC.";;
        en:bbr_to_cubic_fail)   echo "Could not replace BBR with CUBIC. Check the system configuration.";;
        ru:bbr_to_cubic_fail)   echo "Не удалось заменить BBR на CUBIC. Проверьте системную конфигурацию.";;
        en:bbr_already_on)      echo "BBR is already enabled!";;
        ru:bbr_already_on)      echo "BBR уже включен!";;
        en:bbr_enabled)         echo "BBR enabled successfully.";;
        ru:bbr_enabled)         echo "BBR успешно включен.";;
        en:bbr_enable_fail)     echo "Could not enable BBR. Check the system configuration.";;
        ru:bbr_enable_fail)     echo "Не удалось включить BBR. Проверьте системную конфигурацию.";;
        en:os_not_supported)    echo "Operating system is not supported. Inspect the script and install the required packages manually.";;
        ru:os_not_supported)    echo "Операционная система не поддерживается. Проверьте скрипт и установите нужные пакеты вручную.";;
        en:installing_acme)     echo "Installing acme...";;
        ru:installing_acme)     echo "Установка acme...";;
        en:acme_install_fail)   echo "Could not install acme";;
        ru:acme_install_fail)   echo "Не удалось установить acme";;
        en:acme_install_ok)     echo "acme installed successfully";;
        ru:acme_install_ok)     echo "acme успешно установлен";;
        en:ssl_get)             echo "Issue an SSL certificate";;
        ru:ssl_get)             echo "Получить SSL";;
        en:ssl_revoke)          echo "Revoke a certificate";;
        ru:ssl_revoke)          echo "Отозвать сертификат";;
        en:ssl_force_renew)     echo "Force renew";;
        ru:ssl_force_renew)     echo "Принудительно продлить";;
        en:ssl_self_signed)     echo "Self-signed certificate";;
        ru:ssl_self_signed)     echo "Самоподписанный сертификат";;
        en:ssl_ip)              echo "Issue a certificate for an IP address (Let's Encrypt)";;
        ru:ssl_ip)              echo "Выпустить сертификат для IP-адреса (Let's Encrypt)";;

        en:menu_title)          echo "S-UI management script";;
        ru:menu_title)          echo "Скрипт управления S-UI";;
        en:menu_exit)           echo "Exit";;
        ru:menu_exit)           echo "Выход";;
        en:menu_install)        echo "Install";;
        ru:menu_install)        echo "Установить";;
        en:menu_update)         echo "Update";;
        ru:menu_update)         echo "Обновить";;
        en:menu_custom_version) echo "Custom version";;
        ru:menu_custom_version) echo "Пользовательская версия";;
        en:menu_uninstall)      echo "Uninstall";;
        ru:menu_uninstall)      echo "Удалить";;
        en:menu_reset_admin)    echo "Reset admin credentials to default";;
        ru:menu_reset_admin)    echo "Сбросить учетные данные администратора по умолчанию";;
        en:menu_set_admin)      echo "Set admin credentials";;
        ru:menu_set_admin)      echo "Задать учетные данные администратора";;
        en:menu_view_admin)     echo "Show admin credentials";;
        ru:menu_view_admin)     echo "Показать учетные данные администратора";;
        en:menu_reset_settings) echo "Reset panel settings";;
        ru:menu_reset_settings) echo "Сбросить настройки панели";;
        en:menu_set_settings)   echo "Configure panel";;
        ru:menu_set_settings)   echo "Настроить панель";;
        en:menu_view_settings)  echo "Show panel settings";;
        ru:menu_view_settings)  echo "Показать настройки панели";;
        en:menu_clear_domain)   echo "Clear panel domain and address";;
        ru:menu_clear_domain)   echo "Очистить домен и адрес панели";;
        en:menu_start)          echo "Start S-UI";;
        ru:menu_start)          echo "Запустить S-UI";;
        en:menu_stop)           echo "Stop S-UI";;
        ru:menu_stop)           echo "Остановить S-UI";;
        en:menu_restart)        echo "Restart S-UI";;
        ru:menu_restart)        echo "Перезапустить S-UI";;
        en:menu_status)         echo "Show S-UI status";;
        ru:menu_status)         echo "Показать статус S-UI";;
        en:menu_log)            echo "Show S-UI log";;
        ru:menu_log)            echo "Показать журнал S-UI";;
        en:menu_enable_auto)    echo "Enable S-UI autostart";;
        ru:menu_enable_auto)    echo "Включить автозапуск S-UI";;
        en:menu_disable_auto)   echo "Disable S-UI autostart";;
        ru:menu_disable_auto)   echo "Отключить автозапуск S-UI";;
        en:menu_bbr)            echo "Enable or disable BBR";;
        ru:menu_bbr)            echo "Включить или отключить BBR";;
        en:menu_ssl)            echo "Manage SSL certificates";;
        ru:menu_ssl)            echo "Управление SSL-сертификатами";;
        en:menu_ssl_cf)         echo "Cloudflare SSL certificate";;
        ru:menu_ssl_cf)         echo "SSL-сертификат Cloudflare";;
        en:menu_language)       echo "Language";;
        ru:menu_language)       echo "Язык";;
        en:menu_env_keys)       echo "Generate env keys";;
        ru:menu_env_keys)       echo "Сгенерировать ключи окружения";;
        en:env_keys_title)      echo "Generate env keys";;
        ru:env_keys_title)      echo "Генерация ключей окружения";;
        en:env_keys_opt_cookie) echo "Generate session cookie key (SUI_COOKIE_KEY)";;
        ru:env_keys_opt_cookie) echo "Сгенерировать ключ сессионных cookie (SUI_COOKIE_KEY)";;
        en:env_keys_opt_awg)    echo "Generate AmneziaWG device key (AWG_KEY_ENC)";;
        ru:env_keys_opt_awg)    echo "Сгенерировать ключ устройств AmneziaWG (AWG_KEY_ENC)";;
        en:env_keys_choice)     echo "Enter your choice [0-2]: ";;
        ru:env_keys_choice)     echo "Введите ваш выбор [0-2]: ";;
        en:awg_key_menu_exists) echo "AWG_KEY_ENC already exists (current: $2)";;
        ru:awg_key_menu_exists) echo "AWG_KEY_ENC уже существует (текущий: $2)";;
        en:awg_key_rotate_q)    echo "Rotate it? WARNING: if AWG devices already exist, rotating this key makes their stored key material undecryptable";;
        ru:awg_key_rotate_q)    echo "Ротировать? ВНИМАНИЕ: если устройства AWG уже созданы, смена этого ключа сделает их сохранённый ключевой материал нечитаемым";;
        en:awg_key_none)        echo "AWG_KEY_ENC is not set yet; a new key will be generated.";;
        ru:awg_key_none)        echo "AWG_KEY_ENC ещё не задан; будет сгенерирован новый ключ.";;
        en:awg_key_generated)   echo "Generated AWG_KEY_ENC (shown once):";;
        ru:awg_key_generated)   echo "Сгенерирован AWG_KEY_ENC (показывается один раз):";;
        en:awg_key_keep)        echo "Keep this key. If it is lost, the configs of devices already created cannot be decrypted.";;
        ru:awg_key_keep)        echo "Сохраните этот ключ. Если он потеряется, конфиги уже созданных устройств нельзя будет расшифровать.";;
        en:awg_key_restart)     echo "The key takes effect after a service restart.";;
        ru:awg_key_restart)     echo "Ключ вступит в силу после перезапуска службы.";;
        en:cookie_key_exists)   echo "SUI_COOKIE_KEY already exists (current: $2)";;
        ru:cookie_key_exists)   echo "SUI_COOKIE_KEY уже существует (текущий: $2)";;
        en:cookie_key_rotate_q) echo "Rotate it? A new key will sign new sessions; the previous key is kept for rollover, so active sessions are not signed out";;
        ru:cookie_key_rotate_q) echo "Ротировать его? Новый ключ будет подписывать новые сессии; прежний ключ сохранится для плавного перехода, активные сессии не разлогинятся";;
        en:cookie_key_none)     echo "SUI_COOKIE_KEY is not set yet; a new key will be generated.";;
        ru:cookie_key_none)     echo "SUI_COOKIE_KEY еще не задан; будет сгенерирован новый ключ.";;
        en:cookie_key_generated) echo "Generated SUI_COOKIE_KEY (shown once):";;
        ru:cookie_key_generated) echo "Сгенерирован SUI_COOKIE_KEY (показывается один раз):";;
        en:cookie_key_file)     echo "Key file: $2";;
        ru:cookie_key_file)     echo "Файл ключа: $2";;
        en:cookie_key_keep)     echo "Keep this file private and preserve the same value across updates and restores.";;
        ru:cookie_key_keep)     echo "Держите этот файл в секрете и сохраняйте то же значение при обновлениях и восстановлении.";;
        en:cookie_key_restart_rollover) echo "The key takes effect after a service restart. The previous key is kept for rollover, so active sessions stay signed in.";;
        ru:cookie_key_restart_rollover) echo "Ключ вступит в силу после перезапуска службы. Прежний ключ сохранен для плавной ротации. Активные сессии останутся в силе.";;
        en:cookie_key_restart_fresh) echo "The key takes effect after a service restart. Sessions signed with the previous fallback key will be signed out once; log in again afterwards.";;
        ru:cookie_key_restart_fresh) echo "Ключ вступит в силу после перезапуска службы. Сессии, подписанные прежним fallback-ключом, будут разлогинены один раз. Потребуется повторный вход.";;
        en:menu_back)           echo "Back";;
        ru:menu_back)           echo "Назад";;
        en:enter_choice_range)  echo "Enter your choice [0-23]: ";;
        ru:enter_choice_range)  echo "Введите ваш выбор [0-23]: ";;
        en:enter_valid_number)  echo "Enter a valid number [0-23]";;
        ru:enter_valid_number)  echo "Введите корректное число [0-23]";;
        en:lang_select)         echo "Select language / Выберите язык / 请选择语言";;
        ru:lang_select)         echo "Select language / Выберите язык / 请选择语言";;
        en:lang_set_to)         echo "Language set to: $2";;
        ru:lang_set_to)         echo "Язык установлен: $2";;
        en:lang_save_failed)    echo "Could not write language file $2; check permissions.";;
        ru:lang_save_failed)    echo "Не удалось сохранить файл языка $2; проверьте права.";;

        en:usage_title)         echo "S-UI management menu usage";;
        ru:usage_title)         echo "Использование меню управления S-UI";;
        en:usage_main)          echo "s-ui              - admin management script";;
        ru:usage_main)          echo "s-ui              - скрипт управления администратора";;
        en:usage_start)         echo "s-ui start        - start s-ui";;
        ru:usage_start)         echo "s-ui start        - запустить s-ui";;
        en:usage_stop)          echo "s-ui stop         - stop s-ui";;
        ru:usage_stop)          echo "s-ui stop         - остановить s-ui";;
        en:usage_restart)       echo "s-ui restart      - restart s-ui";;
        ru:usage_restart)       echo "s-ui restart      - перезапустить s-ui";;
        en:usage_status)        echo "s-ui status       - show current s-ui status";;
        ru:usage_status)        echo "s-ui status       - показать текущий статус s-ui";;
        en:usage_enable)        echo "s-ui enable       - enable autostart";;
        ru:usage_enable)        echo "s-ui enable       - включить автозапуск";;
        en:usage_disable)       echo "s-ui disable      - disable autostart";;
        ru:usage_disable)       echo "s-ui disable      - отключить автозапуск";;
        en:usage_log)           echo "s-ui log          - show s-ui log";;
        ru:usage_log)           echo "s-ui log          - показать журнал s-ui";;
        en:usage_update)        echo "s-ui update       - update";;
        ru:usage_update)        echo "s-ui update       - обновить";;
        en:usage_install)       echo "s-ui install      - install";;
        ru:usage_install)       echo "s-ui install      - установить";;
        en:usage_uninstall)     echo "s-ui uninstall    - uninstall";;
        ru:usage_uninstall)     echo "s-ui uninstall    - удалить";;
        en:usage_help)          echo "s-ui help         - management menu help";;
        ru:usage_help)          echo "s-ui help         - справка по меню управления";;

        *) echo "${key}";;
    esac
}

load_language

function LOGD() { echo -e "${yellow}$(t debug_tag) $* ${plain}"; }
function LOGE() { echo -e "${red}$(t error_tag) $* ${plain}"; }
function LOGI() { echo -e "${green}$(t info_tag) $* ${plain}"; }

[[ $EUID -ne 0 ]] && LOGE "$(t run_as_root)\n" && exit 1

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

confirm() {
    if [[ $# -gt 1 ]]; then
        echo && read -rp "$1 [$(t default_n "$2")]: " temp
        if [[ "${temp}" == "" ]]; then
            temp=$2
        fi
    else
read -rp "$1 [y/n]: " temp
    fi
    if [[ "${temp}" == "y" || "${temp}" == "Y" ]]; then
        return 0
    else
        return 1
    fi
}

confirm_restart() {
    if confirm "$(t restart_service_q "${1}")" "y"; then
        restart
    else
        show_menu
    fi
}

before_show_menu() {
    echo && echo -n -e "${yellow}$(t press_enter_main)${plain}" && read -r temp
    show_menu
}

install() {
    if bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/install.sh); then
        if [[ $# == 0 ]]; then
            start
        else
            start 0
        fi
    fi
}

update() {
    if ! confirm "$(t install_force_q)" "n"; then
        LOGE "$(t cancelled)"
        if [[ $# == 0 ]]; then
            before_show_menu
        fi
        return 0
    fi
    if bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/install.sh); then
        LOGI "$(t update_done)"
        exit 0
    fi
}

custom_version() {
    t enter_panel_version
    read -r panel_version

    if [ -z "$panel_version" ]; then
        t version_required
        exit 1
    fi

    [[ "${panel_version}" != v* ]] && panel_version="v${panel_version}"

    if [[ ! "${panel_version}" =~ ^v[0-9]+[.][0-9]+[.][0-9]+(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$ ]]; then
        t invalid_panel_version "${panel_version}"
        exit 1
    fi

    download_link="https://raw.githubusercontent.com/deposist/s-ui-x-extended/main/install.sh"

    t downloading_version "${panel_version}"
    bash <(curl -Ls "${download_link}") "${panel_version}"
}

uninstall() {
    if ! confirm "$(t uninstall_q)" "n"; then
        if [[ $# == 0 ]]; then
            show_menu
        fi
        return 0
    fi
    systemctl stop s-ui
    systemctl disable s-ui
    rm /etc/systemd/system/s-ui.service -f
    rm "${SECRETBOX_DROPIN_FILE}" -f
    rmdir "${SECRETBOX_DROPIN_DIR}" 2>/dev/null || true
    systemctl daemon-reload
    systemctl reset-failed
    rm /etc/s-ui/ -rf
    rm /usr/local/s-ui/ -rf

    echo ""
    echo -e "$(t uninstall_done)"
    echo ""

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

reset_admin() {
    t reset_admin_warn
    if confirm "$(t reset_admin_q)" "n"; then
        /usr/local/s-ui/sui admin -reset
    fi
    before_show_menu
}

set_admin() {
    t set_admin_warn
read -rp "$(t set_username_p)" config_account
read -rp "$(t set_password_p)" config_password
    /usr/local/s-ui/sui admin -username "${config_account}" -password "${config_password}"
    before_show_menu
}

view_admin() {
    /usr/local/s-ui/sui admin -show
    before_show_menu
}

reset_setting() {
    if confirm "$(t reset_settings_q)" "n"; then
        /usr/local/s-ui/sui setting -reset
    fi
    before_show_menu
}

set_setting() {
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
    before_show_menu
}

view_setting() {
    /usr/local/s-ui/sui setting -show
    view_uri
    before_show_menu
}

clear_domain() {
    if confirm "$(t clear_domain_q)" "n"; then
        /usr/local/s-ui/sui setting -clearDomain
    fi
    before_show_menu
}

read_env_value() {
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

write_env_value() {
    local var="$1" value="$2"
    mkdir -p "$(dirname "${SECRETBOX_ENV_FILE}")"
    if [[ -f "${SECRETBOX_ENV_FILE}" ]]; then
        # Note: an `exit 0` inside the main block still runs END, so the detector
        # must record the match in a flag and let END decide the exit code -
        # a bare `exit 0` here would be overwritten by `END { exit 1 }` and the
        # key would be appended (duplicated) instead of replaced in place.
        if awk -v var="${var}" 'index($0, var"=") == 1 { found = 1 } END { exit(found ? 0 : 1) }' "${SECRETBOX_ENV_FILE}"; then
            local tmp="${SECRETBOX_ENV_FILE}.tmp"
            (umask 077 && awk -v var="${var}" -v val="${value}" \
                'index($0, var"=") == 1 { print var "=" val; next } { print }' \
                "${SECRETBOX_ENV_FILE}" >"${tmp}") && mv "${tmp}" "${SECRETBOX_ENV_FILE}"
        else
            printf '%s=%s\n' "${var}" "${value}" >>"${SECRETBOX_ENV_FILE}"
        fi
    else
        (umask 077 && printf '%s=%s\n' "${var}" "${value}" >"${SECRETBOX_ENV_FILE}")
    fi
    chmod 600 "${SECRETBOX_ENV_FILE}"
}

ensure_env_dropin() {
    mkdir -p "${SECRETBOX_DROPIN_DIR}"
    if [[ ! -f "${SECRETBOX_DROPIN_FILE}" ]]; then
        printf '[Service]\nEnvironmentFile=-%s\n' "${SECRETBOX_ENV_FILE}" >"${SECRETBOX_DROPIN_FILE}"
        chmod 644 "${SECRETBOX_DROPIN_FILE}"
        systemctl daemon-reload
    fi
}

generate_cookie_key() {
    local existing=""
    existing=$(read_env_value SUI_COOKIE_KEY) || existing=""

    local new_key
    new_key=$(head -c 32 /dev/urandom | base64 | tr -d '\r\n')
    local value="${new_key}"
    local restart_note="cookie_key_restart_fresh"

    if [[ -n "${existing}" ]]; then
        LOGI "$(t cookie_key_exists "${existing:0:8}...")"
        if ! confirm "$(t cookie_key_rotate_q)" "n"; then
            LOGI "$(t cancelled)"
            before_show_menu
            return 0
        fi
        # Rollover: the new key goes first (it signs new session cookies) and up
        # to two previous keys stay accepted, so rotating the key does not sign
        # out active sessions. The backend reads SUI_COOKIE_KEY as a
        # comma-separated key list.
        local kept
        kept=$(printf '%s' "${existing}" | awk -F'[,;]' \
            '{ out=""; for (i = 1; i <= NF && i <= 2; i++) { gsub(/^[ \t]+|[ \t]+$/, "", $i); if ($i != "") out = out (out == "" ? "" : ",") $i }; print out }')
        if [[ -n "${kept}" ]]; then
            value="${new_key},${kept}"
            restart_note="cookie_key_restart_rollover"
        fi
    else
        LOGI "$(t cookie_key_none)"
    fi

    write_env_value SUI_COOKIE_KEY "${value}"
    ensure_env_dropin

    echo -e "###############################################"
    echo -e "${yellow}$(t cookie_key_generated)${plain}"
    echo -e "${green}SUI_COOKIE_KEY: ${value}${plain}"
    echo -e "$(t cookie_key_file "${SECRETBOX_ENV_FILE}")"
    echo -e "${red}$(t cookie_key_keep)${plain}"
    echo -e "###############################################"
    LOGI "$(t "${restart_note}")"

    if confirm "$(t restart_service_q "s-ui")" "y"; then
        restart s-ui
    else
        before_show_menu
    fi
}

# Generates the AmneziaWG device key-encryption key (AWG_KEY_ENC). It seals the
# per-device WireGuard private and preshared keys at rest. Unlike the cookie
# key, there is no rollover: rotating it after devices exist makes their stored
# key material undecryptable, so an existing key is kept unless the operator
# explicitly confirms the destructive rotation.
generate_awg_key() {
    local existing=""
    existing=$(read_env_value AWG_KEY_ENC) || existing=""

    if [[ -n "${existing}" ]]; then
        LOGI "$(t awg_key_menu_exists "${existing:0:8}...")"
        if ! confirm "$(t awg_key_rotate_q)" "n"; then
            LOGI "$(t cancelled)"
            before_show_menu
            return 0
        fi
    else
        LOGI "$(t awg_key_none)"
    fi

    local value
    value=$(head -c 32 /dev/urandom | base64 | tr -d '\r\n')
    write_env_value AWG_KEY_ENC "${value}"
    ensure_env_dropin

    echo -e "###############################################"
    echo -e "${yellow}$(t awg_key_generated)${plain}"
    echo -e "${green}AWG_KEY_ENC: ${value}${plain}"
    echo -e "$(t cookie_key_file "${SECRETBOX_ENV_FILE}")"
    echo -e "${red}$(t awg_key_keep)${plain}"
    echo -e "###############################################"
    LOGI "$(t awg_key_restart)"

    if confirm "$(t restart_service_q "s-ui")" "y"; then
        restart s-ui
    else
        before_show_menu
    fi
}

# Submenu for the environment key generators (menu item 23).
generate_env_keys() {
    echo -e "
  ${green}$(t env_keys_title)${plain}
---------------------------------------------------------------
  ${green}1.${plain} $(t env_keys_opt_cookie)
  ${green}2.${plain} $(t env_keys_opt_awg)
---------------------------------------------------------------
  ${green}0.${plain} $(t menu_back)
"
    local choice
    read -rp "$(t env_keys_choice)" choice
    case "${choice}" in
        1) generate_cookie_key ;;
        2) generate_awg_key ;;
        0) show_menu ;;
        *) LOGE "$(t enter_valid_number)"; before_show_menu ;;
    esac
}

view_uri() {
    if ! info=$(/usr/local/s-ui/sui uri); then
        LOGE "$(t could_not_get_uri)"
        before_show_menu
        return
    fi
    LOGI "$(t panel_url)"
    echo -e "${green}${info}${plain}"
}

start() {
    if check_status "$1"; then
        echo ""
        LOGI "$(t already_running "${1}")"
    else
        systemctl start "$1"
        sleep 2
        if check_status "$1"; then
            LOGI "$(t start_ok "${1}")"
        else
            LOGE "$(t start_fail "${1}")"
        fi
    fi

    if [[ $# == 1 ]]; then
        before_show_menu
    fi
}

stop() {
    check_status "$1"
    if [[ $? == 1 ]]; then
        echo ""
        LOGI "$(t already_stopped "${1}")"
    else
        systemctl stop "$1"
        sleep 2
        check_status "$1"
        if [[ $? == 1 ]]; then
            LOGI "$(t stop_ok "${1}")"
        else
            LOGE "$(t stop_fail "${1}")"
        fi
    fi

    if [[ $# == 1 ]]; then
        before_show_menu
    fi
}

restart() {
    systemctl restart "$1"
    local waited=0
    while [[ $waited -lt 10 ]]; do
        if check_status "$1"; then
            break
        fi
        sleep 1
        waited=$((waited + 1))
    done
    if check_status "$1"; then
        LOGI "$(t restart_ok "${1}")"
    else
        LOGE "$(t restart_fail "${1}")"
    fi
    if [[ $# == 1 ]]; then
        before_show_menu
    fi
}

status() {
    systemctl status s-ui -l
    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

enable() {
    if systemctl enable "$1"; then
        LOGI "$(t enable_ok "${1}")"
    else
        LOGE "$(t enable_fail "${1}")"
    fi

    if [[ $# == 1 ]]; then
        before_show_menu
    fi
}

disable() {
    if systemctl disable "$1"; then
        LOGI "$(t disable_ok "${1}")"
    else
        LOGE "$(t disable_fail "${1}")"
    fi

    if [[ $# == 1 ]]; then
        before_show_menu
    fi
}

show_log() {
    journalctl -u "$1".service -e --no-pager -f
    if [[ $# == 1 ]]; then
        before_show_menu
    fi
}

update_shell() {
    local tmp_script
    tmp_script="$(mktemp)"
    # Keep TLS validation ON (github.com presents a valid certificate, so the
    # transport is the integrity anchor) and download to a temp file first, then
    # swap it into the root-executed path atomically only after a fully successful
    # fetch. A failed/partial transfer must never leave a broken root script in
    # /usr/bin/s-ui.
    wget --timeout=20 --tries=5 --retry-connrefused -O "${tmp_script}" https://github.com/deposist/s-ui-x-extended/raw/main/s-ui.sh
    if [[ $? != 0 || ! -s "${tmp_script}" ]]; then
        rm -f "${tmp_script}"
        echo ""
        LOGE "$(t download_fail)"
        before_show_menu
    else
        chmod +x "${tmp_script}"
        mv -f "${tmp_script}" /usr/bin/s-ui
        LOGI "$(t script_updated)" && exit 0
    fi
}

check_status() {
    if [[ ! -f "/etc/systemd/system/$1.service" ]]; then
        return 2
    fi
    temp=$(systemctl is-active "$1" 2>/dev/null || true)
    if [[ "${temp}" == "active" || "${temp}" == "activating" ]]; then
        return 0
    else
        return 1
    fi
}

check_enabled() {
    temp=$(systemctl is-enabled "$1")
    if [[ "${temp}" == "enabled" ]]; then
        return 0
    else
        return 1
    fi
}

check_uninstall() {
    local status_code=0
    check_status s-ui || status_code=$?
    if [[ ${status_code} != 2 ]]; then
        echo ""
        LOGE "$(t already_installed)"
        if [[ $# == 0 ]]; then
            before_show_menu
        fi
        return 1
    else
        return 0
    fi
}

check_install() {
    local status_code=0
    check_status s-ui || status_code=$?
    if [[ ${status_code} == 2 ]]; then
        echo ""
        LOGE "$(t install_first)"
        if [[ $# == 0 ]]; then
            before_show_menu
        fi
        return 1
    else
        return 0
    fi
}

show_status() {
    local status_code=0
    check_status "$1" || status_code=$?
    case ${status_code} in
    0)
        echo -e "${green}$(t status_running "${1}")${plain}"
        show_enable_status "$1"
        ;;
    1)
        echo -e "${yellow}$(t status_stopped "${1}")${plain}"
        show_enable_status "$1"
        ;;
    2)
        echo -e "${red}$(t status_missing "${1}")${plain}"
        ;;
    esac
}

show_enable_status() {
    if check_enabled "$1"; then
        echo -e "${green}$(t autostart_yes "${1}")${plain}"
    else
        echo -e "${red}$(t autostart_no "${1}")${plain}"
    fi
}

bbr_menu() {
    echo -e "${green}\t1.${plain} $(t enable_bbr)"
    echo -e "${green}\t2.${plain} $(t disable_bbr)"
    echo -e "${green}\t0.${plain} $(t back_main)"
    read -rp "$(t select_option)" choice
    case "$choice" in
    0) show_menu ;;
    1) enable_bbr ;;
    2) disable_bbr ;;
    *) t invalid_choice ;;
    esac
}

disable_bbr() {
    if ! grep -q "net.core.default_qdisc=fq" /etc/sysctl.conf || ! grep -q "net.ipv4.tcp_congestion_control=bbr" /etc/sysctl.conf; then
        echo -e "${yellow}$(t bbr_already_off)${plain}"
        exit 0
    fi
    sed -i 's/net.core.default_qdisc=fq/net.core.default_qdisc=pfifo_fast/' /etc/sysctl.conf
    sed -i 's/net.ipv4.tcp_congestion_control=bbr/net.ipv4.tcp_congestion_control=cubic/' /etc/sysctl.conf
    sysctl -p
    if [[ $(sysctl net.ipv4.tcp_congestion_control | awk '{print $3}') == "cubic" ]]; then
        echo -e "${green}$(t bbr_to_cubic_ok)${plain}"
    else
        echo -e "${red}$(t bbr_to_cubic_fail)${plain}"
    fi
}

enable_bbr() {
    if grep -q "net.core.default_qdisc=fq" /etc/sysctl.conf && grep -q "net.ipv4.tcp_congestion_control=bbr" /etc/sysctl.conf; then
        echo -e "${green}$(t bbr_already_on)${plain}"
        exit 0
    fi
    case "${release}" in
    ubuntu | debian | armbian)
        apt-get update && apt-get install -yqq --no-install-recommends ca-certificates ;;
    centos | almalinux | rocky | oracle)
        yum -y update && yum -y install ca-certificates ;;
    fedora)
        dnf -y update && dnf -y install ca-certificates ;;
    arch | manjaro | parch)
        pacman -Sy --noconfirm ca-certificates ;;
    *)
        echo -e "${red}$(t os_not_supported)${plain}\n"
        exit 1 ;;
    esac
    echo "net.core.default_qdisc=fq" | tee -a /etc/sysctl.conf
    echo "net.ipv4.tcp_congestion_control=bbr" | tee -a /etc/sysctl.conf
    sysctl -p
    if [[ $(sysctl net.ipv4.tcp_congestion_control | awk '{print $3}') == "bbr" ]]; then
        echo -e "${green}$(t bbr_enabled)${plain}"
    else
        echo -e "${red}$(t bbr_enable_fail)${plain}"
    fi
}

install_acme() {
    cd ~ || return 1
    LOGI "$(t installing_acme)"
    # Fail closed: -f rejects HTTP error bodies (a 404/partial page must never be
    # piped into a root shell), --proto '=https' forbids a downgrade/redirect to
    # plain http, and --tlsv1.2 sets a TLS floor. get.acme.sh is the vendor's
    # canonical installer; this only hardens how it is fetched.
    if ! curl -fsS --proto '=https' --tlsv1.2 https://get.acme.sh | sh; then
        LOGE "$(t acme_install_fail)"
        return 1
    else
        LOGI "$(t acme_install_ok)"
    fi
    return 0
}

ssl_cert_issue_main() {
    echo -e "${green}\t1.${plain} $(t ssl_get)"
    echo -e "${green}\t2.${plain} $(t ssl_revoke)"
    echo -e "${green}\t3.${plain} $(t ssl_force_renew)"
    echo -e "${green}\t4.${plain} $(t ssl_self_signed)"
    echo -e "${green}\t5.${plain} $(t ssl_ip)"
    read -rp "$(t select_option)" choice
    case "$choice" in
        1) ssl_cert_issue ;;
        2)
            local domain=""
            read -rp "Domain to revoke / Введите домен сертификата для отзыва: " domain
            "${HOME}/.acme.sh/acme.sh" --revoke -d "${domain}"
            LOGI "Certificate revoked / Сертификат отозван"
            ;;
        3)
            local domain=""
            read -rp "Domain to force-renew / Введите домен SSL-сертификата для принудительного продления: " domain
            "${HOME}/.acme.sh/acme.sh" --renew -d "${domain}" --force ;;
        4) generate_self_signed_cert ;;
        5) ssl_cert_issue_ip ;;
        *) t invalid_choice ;;
    esac
}

# ssl_cert_issue_ip drives the in-process IP-address certificate issuance exposed
# by the panel binary (`sui ip-cert`). Unlike the acme.sh flows above this needs
# no external tooling: lego is embedded in the binary. The panel is stopped so
# the HTTP-01 challenge port is free and so the binary has exclusive DB access,
# then restarted to load the new certificate into the web listener.
ssl_cert_issue_ip() {
    local bin="/usr/local/s-ui/sui"
    if [[ ! -x "${bin}" ]]; then
        LOGE "S-UI binary not found at ${bin}; install S-UI first / Бинарник S-UI не найден, сначала установите S-UI."
        before_show_menu
        return 1
    fi

    local ip=""
read -rp "IP address / IP-адрес: " ip
    ip="$(echo -n "${ip}" | tr -d '[:space:]')"
    if [[ -z "${ip}" ]]; then
        LOGE "No IP address entered / IP-адрес не введён."
        before_show_menu
        return 1
    fi

    local email=""
read -rp "ACME account email (optional) / Email для ACME-аккаунта (необязательно): " email

    local WebPort=80
read -rp "HTTP-01 challenge port (default 80) / Порт проверки HTTP-01 (по умолчанию 80): " WebPort
    [[ -z "${WebPort}" ]] && WebPort=80
    if [[ ! "${WebPort}" =~ ^[0-9]+$ ]] || (( WebPort < 1 || WebPort > 65535 )); then
        LOGE "Invalid port; using 80 / Некорректный порт, используется 80."
        WebPort=80
    fi

    echo -e "${yellow}The panel is stopped to free port ${WebPort} during validation, then restarted."
    echo -e "Панель будет остановлена для освобождения порта ${WebPort} на время проверки, затем перезапущена.${plain}"

    # Load the panel's secretbox key so the CLI reads/writes the encrypted ACME
    # account exactly as the running panel does.
    if [[ -f "${SECRETBOX_ENV_FILE}" ]]; then
        set -a
        # shellcheck disable=SC1090
        . "${SECRETBOX_ENV_FILE}"
        set +a
    fi

    # Restore the panel to its prior run state afterwards.
    local was_running=1
    if check_status s-ui; then
        was_running=0
    fi

    stop s-ui 0

    "${bin}" ip-cert issue -ip "${ip}" -email "${email}" -port "${WebPort}"
    local rc=$?

    if [[ ${was_running} == 0 ]]; then
        start s-ui 0
    fi

    if [[ ${rc} == 0 ]]; then
        LOGI "IP certificate issued and applied to the panel HTTPS listener / Сертификат для IP выпущен и применён к HTTPS-панели."
    else
        LOGE "IP certificate issuance failed (exit ${rc}) / Не удалось выпустить сертификат для IP (код ${rc})."
    fi
    before_show_menu
}

ssl_cert_issue() {
    if ! command -v "${HOME}/.acme.sh/acme.sh" &>/dev/null; then
        echo "acme.sh not found, installing / acme.sh не найден, будет выполнена установка"
        if ! install_acme; then
            LOGE "Could not install acme / Не удалось установить acme"
            exit 1
        fi
    fi
    case "${release}" in
    ubuntu | debian | armbian) apt update && apt install socat -y ;;
    centos | almalinux | rocky | oracle) yum -y update && yum -y install socat ;;
    fedora) dnf -y update && dnf -y install socat ;;
    arch | manjaro | parch) pacman -Sy --noconfirm socat ;;
    *) echo -e "${red}$(t os_not_supported)${plain}\n"; exit 1 ;;
    esac

    local domain=""
read -rp "Domain / Домен: " domain
    LOGD "Domain: ${domain}"
    # Detect an existing acme.sh cert for this exact domain. Scan every row
    # (column 1) rather than only the last line, so the check stays correct
    # when several certificates are present.
    local force_flag=""
    local existing
    existing=$("${HOME}/.acme.sh/acme.sh" --list | awk -v d="${domain}" 'NR>1 && $1==d {print $1; exit}')

    if [ "${existing}" == "${domain}" ]; then
        LOGI "$("${HOME}/.acme.sh/acme.sh" --list)"
        echo -e "${yellow}Certificate already exists / Сертификат уже существует.${plain}"
        echo -e "${yellow}Re-issuing overwrites it and counts against the Let's Encrypt"
        echo -e "duplicate-certificate limit (5 per week per identical domain set)."
        echo -e "Перевыпуск перезапишет его и расходует лимит Let's Encrypt"
        echo -e "(5 дубликатов в неделю на одинаковый набор доменов).${plain}"
        local reissue=""
read -rp "Force re-issue? / Перевыпустить принудительно? [y/N]: " reissue
        if [[ "${reissue}" =~ ^[Yy]$ ]]; then
            force_flag="--force"
        else
            LOGI "Cancelled; existing certificate left unchanged / Отменено, сертификат не изменён."
            return 0
        fi
    fi

    certPath="/root/cert/${domain}"
    rm -rf "$certPath"
    mkdir -p "$certPath"

    local WebPort=80
read -rp "Port (default 80) / Порт (по умолчанию 80): " WebPort
    if [[ ${WebPort} -gt 65535 || ${WebPort} -lt 1 ]]; then
        LOGE "Invalid port; using default."
        WebPort=80
    fi
    "${HOME}/.acme.sh/acme.sh" --set-default-ca --server letsencrypt
    if ! "${HOME}/.acme.sh/acme.sh" --issue -d "${domain}" --standalone --httpport "${WebPort}" $force_flag; then
        LOGE "Issue failed; aborting."
        rm -rf "${HOME}/.acme.sh/${domain}"
        exit 1
    fi
    "${HOME}/.acme.sh/acme.sh" --installcert -d "${domain}" \
        --key-file "/root/cert/${domain}/privkey.pem" \
        --fullchain-file "/root/cert/${domain}/fullchain.pem"
    "${HOME}/.acme.sh/acme.sh" --upgrade
    chmod 755 "$certPath"/*
    ls -lah "$certPath"/*
}

ssl_cert_issue_CF() {
    LOGD "******Cloudflare SSL******"
    echo "1) Issue / Выпустить новый сертификат через Cloudflare"
    echo "2) Force renew / Принудительно продлить существующий сертификат"
    echo "3) Back / Вернуться"
read -rp "Choice [1-3]: " choice

    certPath="/root/cert-CF"
    case $choice in
        1|2)
            force_flag=""
            [ "$choice" -eq 2 ] && force_flag="--force"

            if ! command -v "${HOME}/.acme.sh/acme.sh" &>/dev/null; then
                install_acme || exit 1
            fi

            CF_Domain=""
            CF_GlobalKey=""
            CF_AccountEmail=""

read -rp "Domain / Домен: " CF_Domain
read -rp "Cloudflare Global API key / API key: " CF_GlobalKey
read -rp "Cloudflare account email / Email: " CF_AccountEmail

            rm -rf "$certPath" && mkdir -p "$certPath"

            "${HOME}/.acme.sh/acme.sh" --set-default-ca --server letsencrypt || exit 1
            export CF_Key="${CF_GlobalKey}"
            export CF_Email="${CF_AccountEmail}"

            "${HOME}/.acme.sh/acme.sh" --issue --dns dns_cf -d "${CF_Domain}" -d "*.${CF_Domain}" $force_flag --log || exit 1

            mkdir -p "${certPath}/${CF_Domain}"
            "${HOME}/.acme.sh/acme.sh" --installcert -d "${CF_Domain}" -d "*.${CF_Domain}" \
                --fullchain-file "${certPath}/${CF_Domain}/fullchain.pem" \
                --key-file "${certPath}/${CF_Domain}/privkey.pem"
            "${HOME}/.acme.sh/acme.sh" --upgrade
            chmod 755 "${certPath}/${CF_Domain}"
            ls -lah "${certPath}/${CF_Domain}"
            show_menu
            ;;
        3) show_menu ;;
        *) t invalid_choice; show_menu ;;
    esac
}

generate_self_signed_cert() {
    cert_dir="/etc/sing-box"
    mkdir -p "$cert_dir"
    LOGI "Choose certificate type / Выберите тип сертификата:"
    echo -e "${green}\t1.${plain} Ed25519 (recommended / рекомендуется)"
    echo -e "${green}\t2.${plain} RSA 2048"
    echo -e "${green}\t3.${plain} RSA 4096"
    echo -e "${green}\t4.${plain} ECDSA prime256v1"
    echo -e "${green}\t5.${plain} ECDSA secp384r1"
read -rp "Choice [1-5, default 1]: " cert_type
    cert_type=${cert_type:-1}

    case "$cert_type" in
        1) algo="ed25519"; key_opt="-newkey ed25519" ;;
        2) algo="rsa"; key_opt="-newkey rsa:2048" ;;
        3) algo="rsa"; key_opt="-newkey rsa:4096" ;;
        4) algo="ecdsa"; key_opt="-newkey ec -pkeyopt ec_paramgen_curve:prime256v1" ;;
        5) algo="ecdsa"; key_opt="-newkey ec -pkeyopt ec_paramgen_curve:secp384r1" ;;
        *) algo="ed25519"; key_opt="-newkey ed25519" ;;
    esac

    LOGI "Generating self-signed certificate ($algo)..."
    # The script already requires root (see the EUID check), so call openssl
    # directly: prefixing sudo breaks on minimal root-only systems where sudo is
    # not installed ("sudo: command not found").
    if openssl req -x509 -nodes -days 3650 "${key_opt}" \
        -keyout "${cert_dir}/self.key" \
        -out "${cert_dir}/self.crt" \
        -subj "/CN=myserver"; then
        chmod 600 "${cert_dir}/self."*
        LOGI "Self-signed certificate created."
        LOGI "Path: ${cert_dir}/self.crt"
        LOGI "Key:  ${cert_dir}/self.key"
    else
        LOGE "Could not create self-signed certificate."
    fi
    before_show_menu
}

choose_language() {
    echo "$(t lang_select):"
    echo "  1) English"
    echo "  2) Русский"
    echo "  3) 中文"
    read -rp "[1-3]: " lang_choice
    case "${lang_choice}" in
        1|en|EN|English) lang="en" ;;
        2|ru|RU|Russian|Русский) lang="ru" ;;
        3|zh|ZH|Chinese|中文|简体中文) lang="zh" ;;
        *) ;;
    esac
    save_language
    LOGI "$(t lang_set_to "${lang}")"
    before_show_menu
}

show_usage() {
    t usage_title
    echo "------------------------------------------"
    t usage_main
    t usage_start
    t usage_stop
    t usage_restart
    t usage_status
    t usage_enable
    t usage_disable
    t usage_log
    t usage_update
    t usage_install
    t usage_uninstall
    t usage_help
    echo "------------------------------------------"
}

show_menu() {
  echo -e "
  ${green}$(t menu_title) ${plain}
---------------------------------------------------------------
  ${green}0.${plain} $(t menu_exit)
---------------------------------------------------------------
  ${green}1.${plain} $(t menu_install)
  ${green}2.${plain} $(t menu_update)
  ${green}3.${plain} $(t menu_custom_version)
  ${green}4.${plain} $(t menu_uninstall)
---------------------------------------------------------------
  ${green}5.${plain} $(t menu_reset_admin)
  ${green}6.${plain} $(t menu_set_admin)
  ${green}7.${plain} $(t menu_view_admin)
---------------------------------------------------------------
  ${green}8.${plain} $(t menu_reset_settings)
  ${green}9.${plain} $(t menu_set_settings)
  ${green}10.${plain} $(t menu_view_settings)
  ${green}11.${plain} $(t menu_clear_domain)
---------------------------------------------------------------
  ${green}12.${plain} $(t menu_start)
  ${green}13.${plain} $(t menu_stop)
  ${green}14.${plain} $(t menu_restart)
  ${green}15.${plain} $(t menu_status)
  ${green}16.${plain} $(t menu_log)
  ${green}17.${plain} $(t menu_enable_auto)
  ${green}18.${plain} $(t menu_disable_auto)
---------------------------------------------------------------
  ${green}19.${plain} $(t menu_bbr)
  ${green}20.${plain} $(t menu_ssl)
  ${green}21.${plain} $(t menu_ssl_cf)
  ${green}22.${plain} $(t menu_language)
  ${green}23.${plain} $(t menu_env_keys)
---------------------------------------------------------------
 "
    show_status s-ui
    echo && read -rp "$(t enter_choice_range)" num

    case "${num}" in
    0) exit 0 ;;
    1)  check_uninstall && install ;;
    2)  check_install && update ;;
    3)  check_install && custom_version ;;
    4)  check_install && uninstall ;;
    5)  check_install && reset_admin ;;
    6)  check_install && set_admin ;;
    7)  check_install && view_admin ;;
    8)  check_install && reset_setting ;;
    9)  check_install && set_setting ;;
    10) check_install && view_setting ;;
    11) check_install && clear_domain ;;
    12) check_install && start s-ui ;;
    13) check_install && stop s-ui ;;
    14) check_install && restart s-ui ;;
    15) check_install && status s-ui ;;
    16) check_install && show_log s-ui ;;
    17) check_install && enable s-ui ;;
    18) check_install && disable s-ui ;;
    19) bbr_menu ;;
    20) ssl_cert_issue_main ;;
    21) ssl_cert_issue_CF ;;
    22) choose_language ;;
    23) check_install && generate_env_keys ;;
    *) LOGE "$(t enter_valid_number)" ;;
    esac
}

if [[ $# -gt 0 ]]; then
    case $1 in
    "start")     check_install 0 && start s-ui 0 ;;
    "stop")      check_install 0 && stop s-ui 0 ;;
    "restart")   check_install 0 && restart s-ui 0 ;;
    "status")    check_install 0 && status 0 ;;
    "enable")    check_install 0 && enable s-ui 0 ;;
    "disable")   check_install 0 && disable s-ui 0 ;;
    "log")       check_install 0 && show_log s-ui 0 ;;
    "update")    check_install 0 && update 0 ;;
    "install")   check_uninstall 0 && install 0 ;;
    "uninstall") check_install 0 && uninstall 0 ;;
    *) show_usage ;;
    esac
else
    show_menu
fi
