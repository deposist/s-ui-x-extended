# S-UI v1.5.5-beta4-hotfix1

## English

Hotfix release for `v1.5.5-beta4`. It keeps the beta4 feature set and ships the
frontend dependency lockfile and Playwright e2e stabilization fixes that landed
after the original beta4 tag.

### Fixed

* Synchronized `frontend/package-lock.json` with the current npm resolver so
  `npm ci` succeeds in GitHub Actions and fresh installs include the required
  `@emnapi/core` and `@emnapi/runtime` packages.
* Stabilized the frontend e2e dev server by disabling Vite HMR in e2e mode and
  excluding Vuetify from dependency pre-optimization. This prevents missing
  `.vite/deps/*.js` optimized chunks and unexpected page reloads during
  Playwright runs.
* Hardened the websocket reconnect chaos test so browser offline transitions do
  not destroy the page execution context before test-side state is updated.
* Increased the accessibility baseline timeout for the multi-page axe pass.
* Restored bilingual release notes for `v1.5.5-beta4`; the original release now
  has English and Russian notes again.

### Validation

* `cd frontend && npx playwright test` - PASS
* `cd frontend && npm run lint` - PASS
* `cd frontend && npm run build` - PASS
* GitHub Actions on `main`: CI, Audit Frontend, and Audit Go - PASS

### Install

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.5-beta4-hotfix1
```

## Русский

Hotfix-релиз для `v1.5.5-beta4`. Он сохраняет набор изменений beta4 и добавляет
исправления lockfile фронтенда и стабилизацию Playwright e2e, которые были
добавлены после исходного тега beta4.

### Исправлено

* `frontend/package-lock.json` синхронизирован с текущим npm resolver: `npm ci`
  снова проходит в GitHub Actions, а чистая установка получает нужные пакеты
  `@emnapi/core` и `@emnapi/runtime`.
* e2e dev-server фронтенда стабилизирован: в e2e-режиме отключается Vite HMR, а
  Vuetify исключён из dependency pre-optimization. Это убирает пропадающие
  `.vite/deps/*.js` chunks и неожиданные reload во время Playwright-прогонов.
* Websocket reconnect chaos test больше не переводит браузер в offline до
  обновления test-side состояния страницы, поэтому `page.evaluate` не теряет
  execution context.
* Для accessibility baseline увеличен timeout на multi-page axe-прогон.
* Для `v1.5.5-beta4` восстановлены bilingual release notes: исходный релиз снова
  содержит английский и русский текст.

### Валидация

* `cd frontend && npx playwright test` - PASS
* `cd frontend && npm run lint` - PASS
* `cd frontend && npm run build` - PASS
* GitHub Actions на `main`: CI, Audit Frontend и Audit Go - PASS

### Установка

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.5-beta4-hotfix1
```
