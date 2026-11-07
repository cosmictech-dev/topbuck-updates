#!/usr/bin/env bash
# TopBuck update u-og-20261107: new general disclaimer (articles, footer, News, About) + Facebook/link preview images and tags.
# Adds/replaces only the files in the pack. Does not touch feeds, keys or MyCraft.
set -euo pipefail
main() {
  [ "$(id -u)" = 0 ] || { echo "Run with sudo"; exit 1; }
  local SRC=https://raw.githubusercontent.com/cosmictech-dev/topbuck-updates/main
  local PKG=u-og-20261107
  local SHA=a683e4af234b42131cb03603aa5ca332db43289ab5f83808fa329723aa0df732
  local WWW=/opt/topbuck/www
  [ -d "$WWW" ] || { echo "UPDATE FAILED: $WWW not found"; exit 1; }
  TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
  curl -fsSL "$SRC/$PKG.tgz" -o "$TMP/u.tgz"
  [ "$(sha256sum "$TMP/u.tgz" | cut -d' ' -f1)" = "$SHA" ] || { echo "UPDATE FAILED: download damaged. Nothing was changed."; exit 1; }
  mkdir "$TMP/p"; tar xzf "$TMP/u.tgz" -C "$TMP/p" --no-same-owner
  [ -f "$TMP/p/index.html" ] && [ -f "$TMP/p/assets/og/inflation.png" ] || { echo "UPDATE FAILED: pack incomplete. Nothing was changed."; exit 1; }
  local BK=/root/topbuck-www-before-$(date +%F-%H%M).tgz
  tar czf "$BK" -C /opt/topbuck www
  echo "Backup saved to $BK"
  cp -a "$TMP/p/." "$WWW/"
  find "$WWW" -type d -exec chmod 755 {} +
  find "$WWW" -type f -exec chmod 644 {} +
  echo "Undo: sudo rm -rf $WWW && sudo tar xzf $BK -C /opt/topbuck"
  echo "TOPBUCK DISCLAIMER + PREVIEW UPDATE DONE"
}
main "$@"
