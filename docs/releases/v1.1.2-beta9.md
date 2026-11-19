# S-UI-X Extended v1.1.2-beta9

## Pair servers from the panel

Open **Nodes** in the sidebar. On the server that manages clients, choose **Master** and create a one-time invitation. On the other server, enter the full HTTPS address of the Master panel and the invitation code. The code expires after 10 minutes.

On the Slave, configure an inbound and set the address clients will connect to. On the Master, assign a client to that inbound. The subscription shows the Slave link after the Slave applies the assignment. Existing installations keep their current role when upgraded.

Paired nodes support TCP VLESS, VMess and Trojan without REALITY. Each server applies its own IP, connection, speed and traffic limits and fixed expiry, including while the Master is offline. Assignments that require delayed start or automatic quota reset are blocked.

Use **Disconnect node** while the Slave is reachable so it can remove access and confirm the cleanup. Forcing a disconnect removes subscription links immediately. An unreachable Slave can still accept previously issued credentials until you unlink it locally.

## Upgrade

Back up the panel database before upgrading. To roll back, reinstall the previous version and restore that backup. Update both panels before pairing them.

```sh
curl -fLsS https://raw.githubusercontent.com/deposist/s-ui-x-extended/v1.1.2-beta9/install.sh \
  | sudo bash -s -- v1.1.2-beta9
```

On Windows, extract the ZIP for your architecture and run `install-windows.bat` as Administrator. The Docker image is `ghcr.io/deposist/s-ui-x-extended:v1.1.2-beta9`.

## По-русски

Пункт **«Ноды»** появился в боковом меню. На сервере, где вы ведёте клиентов, выберите роль **Master** и создайте код привязки. На втором сервере введите полный HTTPS-адрес панели Master и этот код. Код одноразовый и действует 10 минут.

На Slave настройте входящее и укажите адрес для клиентов. Затем на Master назначьте клиента на это входящее. Ссылка появится в подписке после того, как Slave применит назначение. При обновлении панель сохраняет свою роль.

Связанные ноды принимают VLESS, VMess и Trojan по TCP без REALITY. Каждый сервер отдельно применяет лимиты IP, соединений, скорости и трафика и проверяет срок доступа. Эти ограничения действуют и при потере связи с Master. Панель блокирует назначения с отложенным стартом или автосбросом квоты.

Пока Slave на связи, используйте **«Отключить ноду»**. Master дождётся, пока Slave отзовёт доступ. Принудительное отключение сразу убирает ссылки из подписок. Если Slave недоступен, он может принимать подключения по ранее выданным данным до локальной отвязки.

Перед обновлением сохраните копию базы. Обновите обе панели перед привязкой. Команда для Linux приведена выше. Для Windows распакуйте ZIP для архитектуры своего компьютера и запустите `install-windows.bat` от имени администратора. Образ Docker указан в разделе Upgrade. Для отката установите предыдущую версию и восстановите копию базы.
