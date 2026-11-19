# S-UI-X Extended v1.1.2-beta10

## Manage paired nodes

Copy an invitation on Master and paste it on Slave. It carries the full HTTPS address, including the panel path. The code expires after 10 minutes and can be used once.

The node card shows the last request from Slave, the confirmed access deadline and why a subscription link is missing. On updated panels, managed connections close when the five-minute lease expires without renewal. Preview bulk additions and removals from the node’s client list before applying them.

To check a VLESS TCP inbound, set `SUI_NODE_PROBE_URL` on Master to a trusted HTTPS endpoint that returns a 2xx response. Then select the inbound in **Nodes** and choose **Check connection**. The check temporarily reloads that inbound when adding and removing the test credential.

Overview lists node incidents. You can turn on node alerts in Telegram settings. Client diagnostics shows traffic used and quota remaining on each node, along with the time of its last report.

Paired nodes support more protocols and transports, delayed start and automatic quota resets. Availability depends on the panel versions, builds and subscription format.

## Upgrade

Back up the panel database, then update both Master and Slave. To roll back, install the previous version and restore that backup. If you roll a Slave back to beta9 after it used confirmed leases with Master, pair it again to restore subscription links.

```sh
curl -fLsS https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.2-beta10/install.sh \
  | sudo bash -s -- v1.1.2-beta10
```

On Windows, extract the ZIP for your architecture and run `install-windows.bat` as Administrator. The Docker image is `ghcr.io/deposist/s-ui-x-extended:v1.1.2-beta10`.

## По-русски

Скопируйте приглашение на Master и вставьте его на Slave. В приглашении уже есть полный HTTPS-адрес, включая путь к панели. Код одноразовый и действует 10 минут.

Карточка ноды показывает, когда Slave обращался к Master, до какого времени подтверждён допуск и почему в подписке нет ссылки. Если допуск не продлён в течение пяти минут, обновлённый Slave закрывает управляемые подключения. Перед массовым назначением или удалением клиентов можно посмотреть, какие ссылки изменятся.

Для пробы VLESS TCP задайте на Master `SUI_NODE_PROBE_URL` с адресом доверенного HTTPS-ресурса, который возвращает код 2xx. В «Нодах» выберите входящее подключение и нажмите «Проверить подключение». Панель перезагрузит выбранное входящее подключение при добавлении и удалении временной учётной записи.

Сбои нод видны в «Обзоре». Оповещения включаются в настройках Telegram. Диагностика клиента показывает расход, остаток квоты и время последнего замера на каждом сервере.

Для связанных нод добавлены протоколы и транспорты. Панели также могут применять отложенный старт и автосброс квоты. Доступность зависит от версий панелей, сборок и формата подписки.

Перед обновлением сохраните копию базы и обновите обе панели. Для отката установите предыдущую версию и восстановите копию. Если вернули на beta9 Slave, который уже использовал подтверждённый допуск, привяжите его заново, чтобы вернуть ссылки в подписки.
