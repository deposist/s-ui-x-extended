# S-UI v1.5.4-beta2

Small prerelease hotfix on top of the Nexus UI beta.

## Fixed

- Nexus Overview startup can issue overlapping dashboard reads while the
  shared axios dedupe path is active.
- The older duplicate read is intentionally canceled. That cancellation
  was being converted by the shared HTTP wrapper into a visible
  `failed` notification with `CanceledError: canceled`.
- Canceled duplicate reads now stay silent; real request failures still
  use the existing failed notification path.

## Validation

- `cd frontend && npm run test` - PASS
- `cd frontend && npm run lint` - PASS
- `cd frontend && npm run build` - PASS
- External-origin primary gate over Nexus source - PASS, zero matches
- Built `dist/index.html` inline-script/event-handler gate - PASS, zero matches
- Supply-chain gate, `git diff -- package.json` dependency blocks - PASS

## Install

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.4-beta2
```

## Русский

Небольшой prerelease hotfix поверх Nexus UI beta.

- При старте Nexus Overview возможны overlapping dashboard reads.
- Общий axios dedupe намеренно отменяет более старый дублирующий read,
  но shared HTTP wrapper показывал эту штатную отмену как failed toast
  `CanceledError: canceled`.
- Отменённые duplicate reads теперь остаются тихими; настоящие ошибки
  запросов продолжают показываться.
