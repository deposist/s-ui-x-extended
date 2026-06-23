# Release notes: v1.5.9-beta3

This release adds web updates and outbound failover. No database, API, or configuration changes are required.

## 1. Update the panel from the web UI

A **Panel updates** card is available in **Settings -> Maintenance**.

* Choose a channel. *Main* tracks stable releases. *Beta* tracks the newest release, including pre-releases. On the Beta channel, a newly published stable release replaces the matching beta; once `1.5.9` is out, it is offered instead of `1.5.9-beta2`.
* Click **Check updates** to see the available version, release type, and release notes.
* Click **Update** to download, verify, replace, and restart the panel. Only newer versions are offered, so the web updater does not downgrade the panel.
* The web update does not ask to change the admin login, password, or settings. Existing credentials and configuration stay in place, and required database migrations run on restart.

## 2. Update safety

Applying an update runs downloaded code with the panel's privileges, so the update path has explicit checks.

* Only a signed-in administrator can update, and the action requires password re-entry. The UI warns that the panel and proxying will restart briefly.
* The release artifact is downloaded over verified HTTPS and checked against the release SHA-256 before replacement. A checksum mismatch aborts the update and keeps the running version.
* If an update fails, the previous version keeps running. The replaced binary is backed up, and a newly installed version that repeatedly fails to start is rolled back automatically.
* Every check and update attempt, including rejected attempts, is written to the audit log. The password is never logged.

## 3. Automatic outbound failover

A new **Failover group** outbound switches traffic when an upstream stops responding.

* Create a *Failover group* and add outbounds in priority order. The first member is primary; the rest are backups.
* The panel probes each member over HTTPS. You choose the target host or IP and the interval; the default interval is 30 seconds. When the active member stops responding, traffic moves to the next working backup.
* When the primary is healthy again, traffic returns to it after a short confirmation delay. This avoids repeated switching on an unstable connection.
* If every member is down, the group uses a direct connection when one exists. Otherwise it holds the top member and shows an *all members down* state.
* Select the group as **Routing -> Default Outbound** or in any routing rule. It behaves like any other outbound, and the Outbounds list shows the active member.

No database, API, or configuration changes are required. These features apply to the standard Linux service install.

# Заметки о релизе v1.5.9-beta3

В этом релизе добавлены обновление панели из веб-интерфейса и автоматическое переключение Outbounds. Изменения базы данных, API или конфигурации не требуются.

## 1. Обновление панели из веб-интерфейса

В разделе **Настройки -> Обслуживание** появилась карточка **Обновления панели**.

* Выберите канал. *Main* отслеживает стабильные релизы. *Beta* отслеживает новейший релиз, включая предварительные. На канале Beta новая стабильная версия заменяет соответствующую бету; после выхода `1.5.9` будет предложена она, а не `1.5.9-beta2`.
* Нажмите **Check updates**, чтобы увидеть доступную версию, тип релиза и описание изменений.
* Нажмите **Update**, чтобы скачать, проверить, заменить и перезапустить панель. Веб-обновление предлагает только более новые версии и не выполняет понижение.
* Веб-обновление не предлагает менять логин администратора, пароль или настройки. Текущие учётные данные и конфигурация сохраняются, а нужные миграции базы данных выполняются при перезапуске.

## 2. Проверки при обновлении

Обновление запускает скачанный код с правами панели, поэтому путь обновления проверяется явно.

* Обновление доступно только вошедшему администратору и требует повторного ввода пароля. Интерфейс предупреждает, что панель и проксирование кратковременно перезапустятся.
* Артефакт релиза скачивается по проверенному HTTPS и сверяется с опубликованной SHA-256 суммой до замены. Несовпадение отменяет обновление и оставляет текущую версию.
* Если обновление не завершилось, продолжает работать прежняя версия. Заменяемый бинарь сохраняется в резервную копию, а новая версия, которая повторно не стартует, откатывается автоматически.
* Каждая попытка проверки и обновления, включая отклонённые, записывается в журнал аудита. Пароль не логируется.

## 3. Автоматическое переключение Outbounds

Новый Outbound **Группа отказоустойчивости** переключает трафик, когда внешний канал перестаёт отвечать.

* Создайте *группу отказоустойчивости* и добавьте Outbounds в порядке приоритета. Первый участник основной, остальные резервные.
* Панель проверяет каждого участника по HTTPS. Вы выбираете целевой домен или IP и интервал; по умолчанию это 30 секунд. Когда активный участник перестаёт отвечать, трафик переходит на следующий рабочий резерв.
* Когда основной участник снова стабилен, трафик возвращается к нему после короткой задержки подтверждения. Это защищает от постоянных переключений на нестабильном канале.
* Если недоступны все участники, группа использует direct, если он есть. Иначе она удерживает старшего участника и показывает состояние *все участники недоступны*.
* Выберите группу в **Маршрутизация -> Default Outbound** или в любом правиле. Она ведёт себя как обычный Outbound, а список Outbounds показывает активного участника.

Изменения базы данных, API или конфигурации не требуются. Возможности применимы к стандартной установке как Linux-служба.
