#!/usr/bin/env bash
# TopBuck live-feeds install (feeds-20261106d). Site + fetcher + 2h timer.
# Does not touch MyCraft containers except a Caddy restart if the bind needs refresh.
# Keys are never in this script or the pack — prompted on the server if missing.
set -euo pipefail
main() {
  [ "$(id -u)" = 0 ] || { echo "Run with sudo"; exit 1; }
  local SRC=https://raw.githubusercontent.com/cosmictech-dev/topbuck-updates/main
  local PKG=feeds-20261106d
  local SHA=6cfc01a0901f82c3e9c8259d8e1314f717c60c4fc60d21731e9ff0273e947f1d
  local APP=/opt/topbuck

  [ -d "$APP/www" ] || { echo "UPDATE FAILED: $APP/www not found"; exit 1; }

  TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
  curl -fsSL "$SRC/$PKG.tgz" -o "$TMP/u.tgz"
  [ "$(sha256sum "$TMP/u.tgz" | cut -d' ' -f1)" = "$SHA" ] || {
    echo "UPDATE FAILED: download damaged (checksum mismatch). Nothing was changed."
    exit 1
  }
  mkdir "$TMP/pack"
  tar xzf "$TMP/u.tgz" -C "$TMP/pack" --no-same-owner
  [ -f "$TMP/pack/site/index.html" ] || { echo "UPDATE FAILED: pack has no site/index.html"; exit 1; }
  [ -f "$TMP/pack/fetcher/fetch_feeds.py" ] || { echo "UPDATE FAILED: pack has no fetcher"; exit 1; }

  # Backup for rollback (www + existing feeds if any)
  local BK=/root/topbuck-www-before-$(date +%F-%H%M).tgz
  if [ -d "$APP/feeds" ]; then
    tar czf "$BK" -C "$APP" www feeds
  else
    tar czf "$BK" -C "$APP" www
  fi
  echo "Backup saved to $BK"
  echo "Rollback: tar xzf $BK -C $APP  (then restore www/ from the archive)"

  # Keep previous www for one-step rollback until next successful install
  rm -rf "$APP/www.prev"
  cp -a "$APP/www" "$APP/www.prev"

  # Install fetcher
  mkdir -p "$APP/feeds" "$APP/feeds-state" "$APP/www/data"
  cp -a "$TMP/pack/fetcher/." "$APP/feeds/"
  chmod 755 "$APP/feeds/fetch_feeds.py"

  # Overlay site into www (full public site from pack)
  rm -rf "$APP/www.new"
  mkdir "$APP/www.new"
  cp -a "$TMP/pack/site/." "$APP/www.new/"
  find "$APP/www.new" -type d -exec chmod 755 {} +
  find "$APP/www.new" -type f -exec chmod 644 {} +
  # Atomic-ish swap
  rm -rf "$APP/www.old"
  mv "$APP/www" "$APP/www.old"
  mv "$APP/www.new" "$APP/www"
  rm -rf "$APP/www.old"

  # feeds.env placeholders (never ship secrets)
  if [ ! -f "$APP/feeds.env" ]; then
    cp "$APP/feeds/feeds.env.example" "$APP/feeds.env"
  fi
  chmod 600 "$APP/feeds.env"
  chown root:root "$APP/feeds.env"

  need_keys=0
  mansa_val=$(grep -E '^MANSA_API_KEY=' "$APP/feeds.env" | head -1 | cut -d= -f2- || true)
  curr_val=$(grep -E '^CURRENTS_API_KEY=' "$APP/feeds.env" | head -1 | cut -d= -f2- || true)
  if [ -z "${mansa_val// }" ] || [ -z "${curr_val// }" ]; then
    need_keys=1
  fi

  if [ "$need_keys" = 1 ]; then
    if [ ! -r /dev/tty ]; then
      echo "UPDATE FAILED: /opt/topbuck/feeds.env needs API keys, but /dev/tty is not available."
      echo "Edit $APP/feeds.env as root, then re-run this installer."
      exit 1
    fi
    echo "Enter API keys (input hidden). Keys are stored only in $APP/feeds.env (mode 600)."
    # shellcheck disable=SC2162
    read -r -s -p "MANSA_API_KEY: " MANSA_API_KEY < /dev/tty; echo >/dev/tty
    read -r -s -p "CURRENTS_API_KEY: " CURRENTS_API_KEY < /dev/tty; echo >/dev/tty
    if [ -z "${MANSA_API_KEY}" ] || [ -z "${CURRENTS_API_KEY}" ]; then
      echo "UPDATE FAILED: both keys are required."
      exit 1
    fi
    umask 077
    printf 'MANSA_API_KEY=%s\nCURRENTS_API_KEY=%s\n' "$MANSA_API_KEY" "$CURRENTS_API_KEY" > "$APP/feeds.env"
    chmod 600 "$APP/feeds.env"
    chown root:root "$APP/feeds.env"
    unset MANSA_API_KEY CURRENTS_API_KEY
    echo "Keys saved to $APP/feeds.env"
  else
    echo "Existing keys found in $APP/feeds.env — prompt skipped."
  fi

  # systemd timer (2h)
  if [ -d /etc/systemd/system ]; then
    cp "$APP/feeds/topbuck-feeds.service" /etc/systemd/system/
    cp "$APP/feeds/topbuck-feeds.timer" /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable --now topbuck-feeds.timer >/dev/null
  fi

  # First fetch now
  OUT_DIR="$APP/www/data" STATE_DIR="$APP/feeds-state" TOPBUCK_FEEDS_ENV="$APP/feeds.env" \
    /usr/bin/python3 "$APP/feeds/fetch_feeds.py" || echo "WARNING: initial fetch returned non-zero"

  # Caddy bind refresh if needed (MyCraft compose — restart caddy only)
  if [ -d /opt/mycraft ]; then
    cd /opt/mycraft
    if ! docker compose exec -T caddy test -f /srv/topbuck/js/live.js </dev/null 2>/dev/null; then
      docker compose restart caddy </dev/null >/dev/null || true
      sleep 3
    fi
  fi

  # Summary (no secrets)
  fx_st=unavailable; mk_st=unavailable; nw_st=unavailable
  fx_n=0; mk_n=0; nw_n=0
  if [ -f "$APP/www/data/fx.json" ]; then
    fx_st=$(python3 -c "import json;d=json.load(open('$APP/www/data/fx.json'));print(d.get('status','?'))" 2>/dev/null || echo '?')
    fx_n=$(python3 -c "import json;d=json.load(open('$APP/www/data/fx.json'));print(len(d.get('pairs') or []))" 2>/dev/null || echo 0)
  fi
  if [ -f "$APP/www/data/markets.json" ]; then
    mk_st=$(python3 -c "import json;d=json.load(open('$APP/www/data/markets.json'));print(d.get('status','?'))" 2>/dev/null || echo '?')
    mk_n=$(python3 -c "import json;d=json.load(open('$APP/www/data/markets.json'));print(len(d.get('indices') or []))" 2>/dev/null || echo 0)
  fi
  if [ -f "$APP/www/data/news.json" ]; then
    nw_st=$(python3 -c "import json;d=json.load(open('$APP/www/data/news.json'));print(d.get('status','?'))" 2>/dev/null || echo '?')
    nw_n=$(python3 -c "import json;d=json.load(open('$APP/www/data/news.json'));print(len(d.get('items') or []))" 2>/dev/null || echo 0)
  fi

  echo "FX $fx_st ($fx_n pairs) / markets $mk_st ($mk_n indices) / news $nw_st ($nw_n headlines)"
  echo "TopBuck live data is ON"
  echo "Backup: $BK"
  echo "TOPBUCK LIVE-FEEDS INSTALL DONE"
}
main "$@"
