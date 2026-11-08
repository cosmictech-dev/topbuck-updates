#!/usr/bin/env bash
# TopBuck update u-toolsdisc-20261108c: adds GM's italic calculator disclaimer under the Planning tools intro on the Tools page.
# Replaces only site files in the pack. Does not touch keys, feeds or MyCraft.
set -euo pipefail
main() {
  [ "$(id -u)" = 0 ] || { echo "Run with sudo"; exit 1; }
  local SRC=https://raw.githubusercontent.com/cosmictech-dev/topbuck-updates/main
  local PKG=u-toolsdisc-20261108c
  local SHA=cb9dce682d2868a2720e5f49d9ad57c0441b90b1061f67939f8f2d7eeefa6bb9
  local WWW=/opt/topbuck/www
  [ -d "$WWW" ] || { echo "UPDATE FAILED: TopBuck folder not found. Nothing was changed."; exit 1; }
  TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
  curl -fsSL "$SRC/$PKG.tgz" -o "$TMP/u.tgz"
  [ "$(sha256sum "$TMP/u.tgz" | cut -d' ' -f1)" = "$SHA" ] || { echo "UPDATE FAILED: download damaged. Nothing was changed."; exit 1; }
  mkdir "$TMP/p"; tar xzf "$TMP/u.tgz" -C "$TMP/p" --no-same-owner
  [ -f "$TMP/p/tools.html" ] || { echo "UPDATE FAILED: pack incomplete. Nothing was changed."; exit 1; }
  local BK=/root/topbuck-before-$(date +%F-%H%M).tgz
  tar czf "$BK" -C /opt/topbuck www
  echo "Backup saved to $BK"
  cp -a "$TMP/p/." "$WWW/"
  find "$WWW" -type d -exec chmod 755 {} +
  find "$WWW" -type f -exec chmod 644 {} +
  echo "Undo: sudo rm -rf $WWW && sudo tar xzf $BK -C /opt/topbuck"
  echo "TOPBUCK TOOLS DISCLAIMER UPDATE DONE"
}
main "$@"
