# Release Notes: s-ui-x-extended v1.1.2-beta13

**Date:** November 26, 2026

## What's New

Beta release focused on **stability improvements** and **code quality**.

## Why Upgrade

### More Stable Server
API handlers now log errors instead of crashing. The server stays online even when handling invalid X-UI import requests.

### No Memory Leaks
Fixed event listener cleanup in the browser router. Pages use less memory during long sessions.

### Faster Database
Optimized payment migration queries. Batch updates replace N+1 individual UPDATE statements.

## How to Update

### Docker
```bash
docker pull ghcr.io/deposist/s-ui-x-extended:v1.1.2-beta13
docker stop s-ui-x-extended
docker rm s-ui-x-extended
docker run -d --name s-ui-x-extended \
  -p 54321:54321 \
  -v /etc/s-ui-x:/app/db \
  ghcr.io/deposist/s-ui-x-extended:v1.1.2-beta13
```

### Install Script
```bash
bash <(curl -Ls https://raw.githubusercontent.com/deposist/s-ui-x-extended/master/install.sh) v1.1.2-beta13
```

## Stats
- **7** files changed
- **52** lines added
- **20** lines removed

## Known Issues
None.

---

_Next: stable v1.1.2 after beta13 testing._
