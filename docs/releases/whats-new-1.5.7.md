# v1.5.7 Beta Summary

## English

The 1.5.7 line adds **Paid Subscriptions**, an experimental Telegram bot module. It is disabled by default. Existing installations are unchanged until an administrator enables it.

### Main changes

* **Client Telegram bot.** Bound Telegram users can get subscription links, per-protocol links, QR codes, usage, remaining days, online status, and total traffic.
* **Self-service trial.** New bot users can be auto-registered with a configurable trial, protected by a global cap and per-user rate limits.
* **Payments.** Tariffs can add days and traffic. Supported providers: Telegram Stars, YooKassa, Stripe, CryptoBot, PayMaster, and external links. Renewals are applied idempotently.
* **Refunds.** Users can request refunds from the bot. Telegram Stars refunds can run automatically; other providers send a request to the administrator.
* **Admin refund control.** Administrators can refund any order from the Orders tab and choose whether to revoke granted days and traffic.
* **Broadcasts and greeting.** Administrators can send one-time messages to bound clients and edit the `/start` greeting.
* **Telegram routing.** Bot traffic can use an HTTP/HTTPS/SOCKS5 proxy or a sing-box Outbound, separately for the client bot and admin notifications.

### Fixes

* **Duplicate saves blocked.** Client, Inbound, and Outbound saves now lock while saving, and the server rejects duplicate submissions. One action creates one record.

### Security

* Bot and payment provider tokens are encrypted at rest and masked in the UI.
* Production setups should set `SUI_SECRETBOX_KEY` so the encryption key stays outside the database.
* The bot only acts on its bound client. Sensitive payment identifiers are not sent to the browser or written to logs.

Paid Subscriptions is beta software and disabled by default. Test it on a non-critical instance first.

## Русский

Линейка 1.5.7 добавляет **«Платные подписки»**: экспериментальный модуль Telegram-бота. Он выключен по умолчанию. Существующие установки не меняются, пока администратор не включит модуль.

### Основные изменения

* **Клиентский Telegram-бот.** Привязанные Telegram-пользователи могут получать ссылки подписки, ссылки по протоколам, QR-коды, расход, остаток дней, онлайн-статус и общий трафик.
* **Саморегистрация с триалом.** Новые пользователи бота могут регистрироваться автоматически с настраиваемым пробным периодом, глобальным лимитом и per-user rate limit.
* **Оплата.** Тарифы могут добавлять дни и трафик. Поддерживаются Telegram Stars, YooKassa, Stripe, CryptoBot, PayMaster и внешние ссылки. Продления применяются идемпотентно.
* **Возвраты.** Пользователи могут запросить возврат из бота. Telegram Stars можно вернуть автоматически; по другим провайдерам администратор получает заявку.
* **Контроль возврата для администратора.** Вкладка Orders позволяет вернуть любой заказ и выбрать, отзывать ли выданные дни и трафик.
* **Рассылки и приветствие.** Администратор может отправить разовое сообщение привязанным клиентам и изменить приветствие `/start`.
* **Маршрутизация Telegram.** Трафик бота может идти через HTTP/HTTPS/SOCKS5 proxy или sing-box Outbound, отдельно для клиентского бота и админ-уведомлений.

### Исправления

* **Блокировка дублей при сохранении.** Сохранение клиентов, Inbounds и Outbounds теперь блокирует кнопку на время операции, а сервер отклоняет повторные отправки. Одно действие создаёт одну запись.

### Безопасность

* Токены бота и платёжных провайдеров шифруются на диске и маскируются в UI.
* Для production задайте `SUI_SECRETBOX_KEY`, чтобы ключ шифрования хранился вне базы.
* Бот действует только с привязанным клиентом. Чувствительные платёжные идентификаторы не отправляются в браузер и не пишутся в логи.

«Платные подписки» остаются beta-функцией и выключены по умолчанию. Сначала проверьте их на некритичном экземпляре.

## 简体中文

1.5.7 线加入 **Paid Subscriptions**：实验性的 Telegram 机器人模块。该模块默认关闭。管理员启用前，现有部署不会改变。

### 主要变化

* **客户端 Telegram 机器人。** 已绑定的 Telegram 用户可以获取订阅链接、按协议生成的链接、二维码、用量、剩余天数、在线状态和总流量。
* **自助试用注册。** 新机器人用户可按配置自动注册试用账号，并受全局上限和单用户频率限制保护。
* **支付。** 套餐可增加天数和流量。支持 Telegram Stars、YooKassa、Stripe、CryptoBot、PayMaster 和外部链接。续费会幂等应用。
* **退款。** 用户可在机器人中申请退款。Telegram Stars 可自动退款；其他渠道会把申请发送给管理员。
* **管理员退款控制。** 管理员可在 Orders 标签页退款，并选择是否撤销该订单发放的天数和流量。
* **群发和问候语。** 管理员可向已绑定客户端发送一次性消息，并修改 `/start` 问候语。
* **Telegram 路由。** 机器人流量可经由 HTTP/HTTPS/SOCKS5 代理或 sing-box Outbound，客户端机器人和管理员通知可分别配置。

### 修复

* **阻止重复保存。** 保存客户端、Inbounds 和 Outbounds 时会锁定按钮，服务端也会拒绝重复提交。一次操作只创建一条记录。

### 安全

* 机器人和支付渠道令牌会加密存储，并在 UI 中脱敏。
* 生产环境应设置 `SUI_SECRETBOX_KEY`，让加密密钥保存在数据库之外。
* 机器人只操作其绑定的客户端。敏感支付标识符不会发送到浏览器，也不会写入日志。

Paid Subscriptions 仍为 beta 功能且默认关闭。请先在非关键实例上测试。
