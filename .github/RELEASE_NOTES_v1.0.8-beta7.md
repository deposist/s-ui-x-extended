# S-UI-X Extended v1.0.8-beta7

This beta fixes the Russia regional routing and DNS preset.

The preset previously downloaded SagerNet `geosite-geolocation-ru.srs`. That file was removed upstream, so the panel received HTTP 404 and correctly left the configuration unchanged. Beta7 switches the preset to the maintained runetfreedom `geosite-category-ru.srs` binary rule-set.

The panel still downloads and validates the file before writing the local rule-set path into the sing-box configuration. No database migration is required. After upgrading, re-apply the Russia regional preset.

This is a beta. Test the regional preset on a non-critical server before upgrading production.

Full release notes: [`docs/releases/v1.0.8-beta7.md`](../docs/releases/v1.0.8-beta7.md).
