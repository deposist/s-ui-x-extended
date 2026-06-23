# v1.5.7-beta7 Summary

## English

Maintenance release. No new features and no manual migration.

### Changed

* **Scheduled 3x-ui sync and remote import removed.** 3x-ui import is now a one-time local `.db` upload through the UI wizard, API, or `import-xui --src`. Dry-run, conflict policy, plan/apply, and rollback remain available.

### Fixes

* Per-client traffic charts now sum each time bucket instead of showing only the first sample.
* The navigation drawer no longer toggles itself on resize or re-render.
* The 1.3 data migration now runs in a transaction and checks every write.
* If sing-box fails to start, the panel reports the failure and stays available so the configuration can be fixed from the UI.

### Security and performance

* Login timing no longer reveals whether a username exists.
* URL credentials are masked in logs.
* Session cookies are `Secure` by default.
* IDN panel domains such as `панель.рф` work.
* Paid Subscriptions order history uses indexed lookups.
* Unused frontend dependencies were removed.

## Русский

Сервисный релиз. Новых функций нет, ручная миграция не требуется.

### Изменения

* **Удалены плановая синхронизация 3x-ui и удалённый импорт.** Импорт 3x-ui теперь выполняется как разовая локальная загрузка `.db` через UI-мастер, API или `import-xui --src`. Dry-run, политика конфликтов, plan/apply и откат сохранены.

### Исправления

* Графики трафика клиента теперь суммируют каждый временной интервал, а не показывают только первый отсчёт.
* Навигационный drawer больше не переключается сам при resize или повторном render.
* Миграция данных 1.3 теперь выполняется в транзакции и проверяет каждую запись.
* Если sing-box не запускается, панель сообщает об ошибке и остаётся доступной, чтобы конфиг можно было исправить через UI.

### Безопасность и производительность

* Тайминг логина больше не показывает, существует ли имя пользователя.
* Учётные данные в URL маскируются в логах.
* Cookie сессии по умолчанию имеют флаг `Secure`.
* IDN-домены панели, например `панель.рф`, работают.
* История заказов «Платных подписок» использует индексированные запросы.
* Неиспользуемые frontend-зависимости удалены.

## 简体中文

维护版本。没有新功能，也无需手动迁移。

### 变化

* **移除 3x-ui 定时同步和远程导入。** 3x-ui 导入现在是一次性本地 `.db` 上传，可通过 UI 向导、API 或 `import-xui --src` 执行。Dry-run、冲突策略、plan/apply 和回滚仍然保留。

### 修复

* 客户端流量图现在按时间桶求和，不再只显示第一个样本。
* 导航 drawer 不再在 resize 或重新 render 时自行切换。
* 1.3 数据迁移现在在事务中运行，并检查每次写入。
* 如果 sing-box 启动失败，面板会报告错误并保持可用，方便从 UI 修复配置。

### 安全与性能

* 登录耗时不再泄露用户名是否存在。
* 日志中的 URL 凭据会被脱敏。
* 会话 cookie 默认带 `Secure` 标志。
* 支持 `панель.рф` 等 IDN 面板域名。
* Paid Subscriptions 订单历史使用索引查询。
* 已移除未使用的 frontend 依赖。
