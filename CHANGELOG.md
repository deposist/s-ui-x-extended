# Changelog

All notable changes to s-ui-x-extended.

## [1.1.2-beta13] - 2026-11-26

### Security
- **TypeScript**: replaced `any` types with strict interfaces
  - `router/index.ts`: PreloadError interface
  - `store/__tests__/ws.spec.ts`: WsEvent interface

### Stability
- **API**: replaced panic() with error logging
  - `api/apiHandler.go`
  - `api/import_xui_routes.go`
- **Browser**: fixed event listener memory leak
  - `router/index.ts`

### Performance
- **Database**: optimized N+1 query in payment migration
  - `paidsub/schema.go`

### Code Quality
- **Go**: removed TODO markers (3)
  - `core/box.go`
  - `cronjob/failoverProbe.go`

### Stats
- Files changed: 7
- Lines added: 52
- Lines removed: 20
