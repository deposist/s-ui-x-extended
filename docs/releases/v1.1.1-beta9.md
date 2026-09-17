# S-UI-X Extended v1.1.1-beta9

## Installer no longer rolls back a finished install

Fresh installs done through the pipe form (`curl ... | sudo bash -s -- v1.1.1-beta9`) completed and then destroyed themselves. The installer finished every step, but the final settings prompt read from a pipe instead of a terminal: `read` hit end of input, returned an error, and the error trap treated it as a failed install and ran the rollback. The rollback removed the just-installed binary and the `s-ui.service` unit, then reported a rollback error of its own, because the service it tried to stop no longer existed. On Ubuntu 24.04 this left a machine with no service, no panel and no log entries (issue #9).

The prompt now runs only when standard input is a terminal. With a pipe, the installer prints the question, takes the default answer "no" and finishes normally. A clean install leaves the service created, enabled and running.

Also fixed: `tests/install-download.sh` passed a search pattern starting with dashes to grep without a `--` separator, so grep parsed the pattern as options and the test failed before reaching its checks.

No database migration. The panel and sing-box core configuration format did not change.

## Upgrade

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta9/install.sh \
  | sudo bash -s -- v1.1.1-beta9
```

If beta8 or earlier failed the same way, the failed attempt left nothing behind (the rollback removed it), so the upgrade command is just a fresh install. Check the result with `systemctl status s-ui` and `/usr/local/s-ui/sui -v`.

## AmneziaWG managed endpoints: hot-reload no longer drops client keys

Saving a managed AWG endpoint (for example toggling the AmneziaWG 3.1 switches) recreated it inside the running core without client preshared keys. Peers in the database carry no PSKs (the plaintext keys live only encrypted in the device table), and only the full core start injected them. After every endpoint save, all clients failed handshakes until the s-ui service was restarted. The hot-reload path now injects the PSKs the same way the full start does, so toggling the 3.1 switches, or any other endpoint edit, keeps clients connected. A reconciliation pass cannot repair this by design: it compares public keys and allowed IPs, which still match, so the health snapshot stays green while handshakes fail.

Verification: an integration test starts a real core, completes a handshake, hot-reloads the endpoint through the production save path, and completes another handshake; it also checks the peer PSK and both 3.1 flags in the live device state. The release CI pipeline runs the AmneziaWG 3.1 random trailers wire-format test alongside the existing AWG 2.0 test.

## Установщик больше не откатывает завершённую установку

Чистая установка через канал (`curl ... | sudo bash -s -- v1.1.1-beta9`) проходила до конца, а потом уничтожала сама себя. Установщик выполнял все шаги, но последний вопрос о настройках читался из канала, а не из терминала: `read` упирался в конец ввода, возвращал ошибку, и ловушка ошибок принимала её за сбой установки и запускала откат. Откат удалял только что установленный бинарник и unit-файл `s-ui.service`, после чего сам сообщал об ошибке отката: служба, которую он пытался остановить, уже не существовала. На Ubuntu 24.04 это оставляло машину без службы, без панели и без записей в журнале (issue #9).

Вопрос теперь задаётся только когда стандартный ввод - терминал. При канале установщик печатает вопрос, принимает ответ по умолчанию «нет» и штатно завершается. Чистая установка оставляет службу созданной, включённой и запущенной.

Заодно исправлено: `tests/install-download.sh` передавал grep образец поиска, начинающийся с дефисов, без разделителя `--`, поэтому grep разбирал образец как опции, и тест падал, не дойдя до проверок.

Миграция базы данных не нужна. Формат конфигурации панели и ядра sing-box не менялся.

## Обновление

```sh
curl -fLsS \
  https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.1-beta9/install.sh \
  | sudo bash -s -- v1.1.1-beta9
```

Если beta8 или более ранняя версия падала так же, после неудачной попытки ничего не осталось (откат всё удалил), так что команда обновления - это просто чистая установка. Результат проверяется через `systemctl status s-ui` и `/usr/local/s-ui/sui -v`.

## AmneziaWG: горячая перезагрузка эндпоинтов больше не теряет ключи клиентов

Сохранение управляемого AWG-эндпоинта (например, переключение тумблеров AmneziaWG 3.1) пересоздавало его в работающем ядре без общих ключей клиентов. Пиры в базе PSK не содержат (открытые ключи лежат только в зашифрованном виде в таблице устройств), и подставляла их лишь полная инициализация ядра. После каждого сохранения эндпоинта рукопожатия падали у всех клиентов до перезапуска службы s-ui. Горячая перезагрузка теперь подставляет PSK так же, как полный старт, поэтому переключение тумблеров 3.1 и любое другое редактирование эндпоинта не рвёт соединения. Реконсиляция это починить не может по устройству: она сверяет публичные ключи и адреса, они совпадают, и сводка здоровья остаётся зелёной, пока рукопожатия падают.

Проверка: интеграционный тест поднимает реальное ядро, завершает рукопожатие, прогоняет горячую перезагрузку через продуктовый путь сохранения и завершает второе рукопожатие; заодно проверяет PSK пира и оба флага 3.1 в живом состоянии устройства. Релизная сборка теперь гоняет интеграционный тест формата пакетов 3.1 со случайными трейлерами вместе с существующим тестом AWG 2.0.
