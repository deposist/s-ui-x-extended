# s-ui-x-extended v1.0.0-hotfix4

Hotfix for the first stable release. No manual database migration is required.

## Fixes

- Fixed the client config editor not showing the MTProxy `secret` field. The generic config loop only rendered `password`, `uuid`, and `auth_str` fields, so MTProxy credentials were invisible and uneditable in the UI.

## Verification

Checked locally with:

- `cd frontend && npm run test`
- `cd frontend && npm run lint`
- `cd frontend && npm run build`
