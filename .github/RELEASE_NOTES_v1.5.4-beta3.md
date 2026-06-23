# S-UI v1.5.4-beta3

Prerelease refinement for the Nexus Overview dashboard.

## Changed

- Re-graded Nexus dark surfaces to a deeper navy palette with teal and
  violet accents while keeping classic themes unchanged.
- Removed the standalone Traffic overview panel and the duplicate Health
  KPI.
- Kept Live traffic in the KPI row and moved its spark to a compact live
  status sample window.
- Reflowed the Overview into a denser three-column primary row and
  compacted Top clients, Recent events and Protocol summaries.
- Preserved the frontend-only boundary: no backend/API/CSRF/CSP changes
  and no runtime or development dependency changes.

## Validation

- `cd frontend && npm run test` - PASS
- `cd frontend && npm run lint` - PASS
- `cd frontend && npm run build` - PASS
- `go test ./middleware/... -run TestAdminSecurityHeaders` - PASS
- External-origin primary gate over Nexus source - PASS, zero matches
- Built `dist/index.html` inline-script/event-handler shape gate - PASS
- Supply-chain gate, package dependency blocks unchanged - PASS
- Nexus viewport checks for dark LTR `en` and RTL `fa` at `1440x900`,
  `1180x800`, `834x1112` and `390x844` - PASS

## Install

```sh
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x/main/install.sh) v1.5.4-beta3
```

## Русский

Prerelease refinement для Nexus Overview dashboard.

- Dark surfaces Nexus переведены на более глубокую navy palette с
  teal/violet accents, classic themes не изменялись.
- Удалены дублирующие Traffic overview panel и Health KPI.
- Live traffic остался в KPI row, а spark теперь использует компактное
  окно live status samples.
- Top clients, Recent events и Protocol summaries уплотнены, а primary
  row на desktop стал трёхколоночным.
- Граница frontend-only сохранена: без backend/API/CSRF/CSP и
  dependency drift.
