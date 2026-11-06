# TopBuck updates

Public one-line installer for topbuck.app (Cosmic Technologies).

```bash
curl -fsSL https://raw.githubusercontent.com/cosmictech-dev/topbuck-updates/main/go | sudo bash
```

Installs the live-feeds pack (site + fetcher + 2-hour systemd timer).  
API keys are **not** in this repo — the installer prompts on the server via `/dev/tty` if `/opt/topbuck/feeds.env` is empty (hidden input). Re-runs with keys present skip the prompt.

Rollback: installer writes `/root/topbuck-www-before-YYYY-MM-DD-HHMM.tgz` and keeps `/opt/topbuck/www.prev`.

Previous site-only pack kept for reference: `u-f647638c.tgz`.
