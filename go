#!/usr/bin/env bash
# TopBuck update u-tools2-20261108c: adds My Optimized Balance Sheet and My Monthly Budget (first on Tools), handwritten font, calculator title tidy-up.
# Replaces only site files in the pack. Does not touch keys, feeds or MyCraft.
set -euo pipefail
main() {
  [ "$(id -u)" = 0 ] || { echo "Run with sudo"; exit 1; }
  local SRC=https://raw.githubusercontent.com/cosmictech-dev/topbuck-updates/main
  local PKG=u-tools2-20261108c
  local SHA=da807ccb5538cd01f96ee0f5667a403ac98e57ffb7e77b5d30549b0af2dc9dbe
  local WWW=/opt/topbuck/www
  [ -d "$WWW" ] || { echo "UPDATE FAILED: TopBuck folder not found. Nothing was changed."; exit 1; }
  TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
  curl -fsSL "$SRC/$PKG.tgz" -o "$TMP/u.tgz"
  [ "$(sha256sum "$TMP/u.tgz" | cut -d' ' -f1)" = "$SHA" ] || { echo "UPDATE FAILED: download damaged. Nothing was changed."; exit 1; }
  mkdir "$TMP/p"; tar xzf "$TMP/u.tgz" -C "$TMP/p" --no-same-owner
  [ -f "$TMP/p/tools.html" ] && [ -f "$TMP/p/balance-sheet.html" ] && [ -f "$TMP/p/js/balance.js" ] && [ -f "$TMP/p/js/budget.js" ] || { echo "UPDATE FAILED: pack incomplete. Nothing was changed."; exit 1; }
  local BK=/root/topbuck-before-$(date +%F-%H%M).tgz
  tar czf "$BK" -C /opt/topbuck www
  echo "Backup saved to $BK"
  cp -a "$TMP/p/." "$WWW/"
  find "$WWW" -type d -exec chmod 755 {} +
  find "$WWW" -type f -exec chmod 644 {} +
  echo "Undo: sudo rm -rf $WWW && sudo tar xzf $BK -C /opt/topbuck"
  echo "TOPBUCK BALANCE SHEET AND BUDGET UPDATE DONE"
}
main "$@"
