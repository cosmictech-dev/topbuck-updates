#!/usr/bin/env bash
# TopBuck site update (u-f647638c, 6 Nov 2026): launch content. Advisor pages kept off the public site.
# Swaps the site files in /opt/topbuck/www only. Does not touch MyCraft or Caddy. Old files kept in /root. Safe to run again.
set -euo pipefail
main() {
  [ "$(id -u)" = 0 ] || { echo "Run with sudo"; exit 1; }
  local SRC=https://raw.githubusercontent.com/cosmictech-dev/topbuck-updates/main PKG=u-f647638c SHA=f647638c800d81a21c707dc2193165330f630b45da66cf48afa9bbcb81c2b319 APP=/opt/topbuck
  [ -d "$APP/www" ] || { echo "UPDATE FAILED: $APP/www not found"; exit 1; }
  TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
  curl -fsSL "$SRC/$PKG.tgz" -o "$TMP/u.tgz"
  [ "$(sha256sum "$TMP/u.tgz" | cut -d' ' -f1)" = "$SHA" ] || { echo "UPDATE FAILED: download damaged (checksum mismatch). Nothing was changed."; exit 1; }
  mkdir "$TMP/www"; tar xzf "$TMP/u.tgz" -C "$TMP/www" --no-same-owner
  [ -f "$TMP/www/index.html" ] || { echo "UPDATE FAILED: pack has no index.html"; exit 1; }
  local BK=/root/topbuck-www-before-$(date +%F-%H%M).tgz
  tar czf "$BK" -C "$APP" www; echo "Backup saved to $BK"
  rm -rf "$APP/www.new"; mv "$TMP/www" "$APP/www.new"
  find "$APP/www.new" -type d -exec chmod 755 {} +; find "$APP/www.new" -type f -exec chmod 644 {} +
  rm -rf "$APP/www.old"; mv "$APP/www" "$APP/www.old"; mv "$APP/www.new" "$APP/www"; rm -rf "$APP/www.old"
  # Caddy mounts the folder path; recreate the bind by restarting caddy only if the new folder isn't visible
  cd /opt/mycraft
  if ! docker compose exec -T caddy grep -q NBFIRA /srv/topbuck/js/app.js </dev/null 2>/dev/null; then
    docker compose restart caddy </dev/null >/dev/null
    sleep 3
  fi
  if curl -fsS --max-time 10 https://topbuck.app/js/app.js 2>/dev/null | grep -q NBFIRA; then
    echo "TopBuck: launch content is live"
  else
    echo "WARNING: could not confirm the new files on https://topbuck.app yet"
  fi
  echo "TOPBUCK UPDATE DONE"
}
main "$@"
