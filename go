#!/usr/bin/env bash
# TopBuck update u-newsplit-20261107: separate News and Insights pages, News narrowed to household-money stories (Botswana first).
# Replaces only the files in the pack plus the feeds script. Does not touch keys or MyCraft.
set -euo pipefail
main() {
  [ "$(id -u)" = 0 ] || { echo "Run with sudo"; exit 1; }
  local SRC=https://raw.githubusercontent.com/cosmictech-dev/topbuck-updates/main
  local PKG=u-newsplit-20261107
  local SHA=f240db5e60c438e744855b800650b083143d5bd6ecf852cecd4a232f94b5f94a
  local APP=/opt/topbuck WWW=/opt/topbuck/www
  [ -d "$WWW" ] && [ -f "$APP/feeds/fetch_feeds.py" ] || { echo "UPDATE FAILED: TopBuck folders not found. Nothing was changed."; exit 1; }
  TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
  curl -fsSL "$SRC/$PKG.tgz" -o "$TMP/u.tgz"
  [ "$(sha256sum "$TMP/u.tgz" | cut -d' ' -f1)" = "$SHA" ] || { echo "UPDATE FAILED: download damaged. Nothing was changed."; exit 1; }
  mkdir "$TMP/p"; tar xzf "$TMP/u.tgz" -C "$TMP/p" --no-same-owner
  [ -f "$TMP/p/insights.html" ] && [ -f "$TMP/p/fetcher/fetch_feeds.py" ] || { echo "UPDATE FAILED: pack incomplete. Nothing was changed."; exit 1; }
  /usr/bin/python3 -m py_compile "$TMP/p/fetcher/fetch_feeds.py" || { echo "UPDATE FAILED: feeds script check failed. Nothing was changed."; exit 1; }
  local BK=/root/topbuck-before-$(date +%F-%H%M).tgz
  tar czf "$BK" -C /opt/topbuck www feeds
  echo "Backup saved to $BK"
  cp "$TMP/p/fetcher/fetch_feeds.py" "$APP/feeds/fetch_feeds.py"; chmod 755 "$APP/feeds/fetch_feeds.py"
  rm -rf "$TMP/p/fetcher"
  cp -a "$TMP/p/." "$WWW/"
  find "$WWW" -type d -exec chmod 755 {} +
  find "$WWW" -type f -exec chmod 644 {} +
  OUT_DIR="$WWW/data" STATE_DIR="$APP/feeds-state" TOPBUCK_FEEDS_ENV="$APP/feeds.env" /usr/bin/python3 "$APP/feeds/fetch_feeds.py" >/dev/null 2>&1 || echo "Note: news refresh needs working keys (run the keys command)."
  echo "Undo: sudo rm -rf $APP/www $APP/feeds && sudo tar xzf $BK -C /opt/topbuck"
  echo "TOPBUCK NEWS + INSIGHTS UPDATE DONE"
}
main "$@"
