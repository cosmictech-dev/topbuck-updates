#!/usr/bin/env bash
# TopBuck update u-money-20261107: Money explained (4 Botswana editorial articles), News section, Privacy notice + footer link.
# Adds/replaces only the files in the pack. Does not touch feeds, keys or MyCraft.
set -euo pipefail
main() {
  [ "$(id -u)" = 0 ] || { echo "Run with sudo"; exit 1; }
  local SRC=https://raw.githubusercontent.com/cosmictech-dev/topbuck-updates/main
  local PKG=u-money-20261107
  local SHA=989e427ce7e777da51588320dbd61ae7fb017293582aded0e80e8009e5c1be91
  local WWW=/opt/topbuck/www
  [ -d "$WWW" ] || { echo "UPDATE FAILED: $WWW not found"; exit 1; }
  TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
  curl -fsSL "$SRC/$PKG.tgz" -o "$TMP/u.tgz"
  [ "$(sha256sum "$TMP/u.tgz" | cut -d' ' -f1)" = "$SHA" ] || { echo "UPDATE FAILED: download damaged. Nothing was changed."; exit 1; }
  mkdir "$TMP/p"; tar xzf "$TMP/u.tgz" -C "$TMP/p" --no-same-owner
  [ -f "$TMP/p/news.html" ] && [ -f "$TMP/p/privacy.html" ] || { echo "UPDATE FAILED: pack incomplete. Nothing was changed."; exit 1; }
  local BK=/root/topbuck-www-before-$(date +%F-%H%M).tgz
  tar czf "$BK" -C /opt/topbuck www
  echo "Backup saved to $BK"
  cp -a "$TMP/p/." "$WWW/"
  find "$WWW" -type d -exec chmod 755 {} +
  find "$WWW" -type f -exec chmod 644 {} +
  echo "Undo: sudo rm -rf $WWW && sudo tar xzf $BK -C /opt/topbuck"
  echo "TOPBUCK MONEY EXPLAINED UPDATE DONE"
}
main "$@"
