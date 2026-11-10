#!/usr/bin/env bash
# TopBuck update u-track-20261110: SONA 2026 news article, short tracking links (/sona, /sona-fb, /sona-x) and a cookie-free visit counter.
# Logs keep no IP addresses or cookies. Aggregate counts only, behind a password at /stats/summary.json.
# Touches only the TOPBUCK block in the Caddyfile; MyCraft's own config is not changed. Rolls back on any failure.
set -euo pipefail
main() {
  [ "$(id -u)" = 0 ] || { echo "Run with sudo"; exit 1; }
  local MYCRAFT=/opt/mycraft APP=/opt/topbuck
  local MARK_BEGIN='# BEGIN TOPBUCK (managed installer — do not edit between markers)'
  local MARK_END='# END TOPBUCK'
  [ -f "$MYCRAFT/Caddyfile" ] && [ -d "$APP/www" ] || { echo "UPDATE FAILED: TopBuck or Caddy files not found. Nothing was changed."; exit 1; }
  grep -qF "$MARK_BEGIN" "$MYCRAFT/Caddyfile" || { echo "UPDATE FAILED: TopBuck block not found in Caddyfile. Nothing was changed."; exit 1; }
  # --- site files: SONA 2026 article + News card ---
  local SRC=https://raw.githubusercontent.com/cosmictech-dev/topbuck-updates/main PKG=u-track-20261110 SHA=ab6f3a43b03a6f4cce845442398263e830729899bfa2b77d1086e951cf4a51c9
  TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
  curl -fsSL "$SRC/$PKG.tgz" -o "$TMP/u.tgz"
  [ "$(sha256sum "$TMP/u.tgz" | cut -d' ' -f1)" = "$SHA" ] || { echo "UPDATE FAILED: download damaged. Nothing was changed."; exit 1; }
  mkdir "$TMP/p"; tar xzf "$TMP/u.tgz" -C "$TMP/p" --no-same-owner
  [ -f "$TMP/p/sona-2026-in-numbers.html" ] && [ -f "$TMP/p/news.html" ] || { echo "UPDATE FAILED: pack incomplete. Nothing was changed."; exit 1; }
  cd "$MYCRAFT"
  local BK=/root/topbuck-track-backup-$(date +%F-%H%M); mkdir -p "$BK"
  cp -p Caddyfile "$BK/Caddyfile"
  tar czf "$BK/www-before.tgz" -C /opt/topbuck www
  echo "Backup saved to $BK"
  cp -a "$TMP/p/." "$APP/www/"
  find "$APP/www" -type d -exec chmod 755 {} +
  find "$APP/www" -type f -exec chmod 644 {} +
  echo "SONA article: installed"
  local NEW; NEW=$(mktemp)
  cat > "$NEW.block" <<'BLOCK'
topbuck.app, www.topbuck.app {
	log topbuck {
		output file /data/topbuck-logs/access.log {
			roll_size 20MiB
			roll_keep 12
		}
		format filter {
			wrap json
			fields {
				request>remote_ip delete
				request>client_ip delete
				request>remote_port delete
				request>headers>Cookie delete
				request>headers>Authorization delete
				request>headers>X-Forwarded-For delete
				request>headers>Cf-Connecting-Ip delete
				resp_headers delete
			}
		}
	}
	redir /sona /sona-2026-in-numbers.html?utm_source=slide&utm_medium=qr&utm_campaign=sona2026 302
	redir /sona-fb /sona-2026-in-numbers.html?utm_source=facebook&utm_medium=social&utm_campaign=sona2026 302
	redir /sona-x /sona-2026-in-numbers.html?utm_source=x&utm_medium=social&utm_campaign=sona2026 302
	basicauth /stats/* {
		tbstats $2a$14$2bfUa0SLVMkl5Mvv3tIUouvIZ/EKkCQLEX1NeRC4ViHPviiOvsHsS
	}
	root * /srv/topbuck
	encode zstd gzip
	file_server
	header {
		X-Content-Type-Options nosniff
		Referrer-Policy strict-origin-when-cross-origin
		X-Frame-Options SAMEORIGIN
	}
}
http://topbuck.app, http://www.topbuck.app {
	redir https://topbuck.app{uri} permanent
}
BLOCK
  python3 - Caddyfile "$NEW.block" "$MARK_BEGIN" "$MARK_END" <<'PY2'
import sys, re, pathlib
p = pathlib.Path(sys.argv[1]); blk = pathlib.Path(sys.argv[2]).read_text().strip()
b, e = sys.argv[3], sys.argv[4]
t = p.read_text()
t2, n = re.subn(re.escape(b) + r"[\s\S]*?" + re.escape(e), lambda m: b + "\n" + blk + "\n" + e, t, count=1)
if n != 1: sys.exit("could not find TopBuck markers")
p.write_text(t2)
PY2
  rm -f "$NEW" "$NEW.block"
  rollback() { echo "UPDATE FAILED: $1. Putting the old settings back."; cp -p "$BK/Caddyfile" Caddyfile; docker compose exec -T caddy caddy reload --config /etc/caddy/Caddyfile </dev/null >/dev/null 2>&1 || docker compose restart caddy </dev/null >/dev/null; exit 1; }
  docker compose exec -T caddy caddy validate --config /etc/caddy/Caddyfile </dev/null >/dev/null 2>&1 || rollback "new settings did not validate"
  if ! docker compose exec -T caddy caddy reload --config /etc/caddy/Caddyfile </dev/null >/dev/null 2>&1; then
    docker compose restart caddy </dev/null >/dev/null || rollback "Caddy would not restart"
  fi
  sleep 3
  local ok=0
  for _ in $(seq 1 15); do
    curl -fsS -o /dev/null --max-time 5 https://mycraftglobal.com/ 2>/dev/null && { ok=1; break; }; sleep 2
  done
  [ "$ok" = 1 ] || rollback "MyCraft did not answer after the change"
  local loc; loc=$(curl -sk -o /dev/null -w '%{redirect_url}' --resolve topbuck.app:443:127.0.0.1 https://topbuck.app/sona-fb || true)
  case "$loc" in *sona-2026-in-numbers.html?utm_source=facebook*) echo "Tracking links: working";; *) rollback "tracking link test failed";; esac
  mkdir -p "$APP/bin" "$APP/www/stats"
  cat > "$APP/bin/topbuck-stats.py" <<'STATS'
#!/usr/bin/env python3
"""TopBuck cookie-free visit counter. Reads Caddy's TopBuck access log (no IPs stored)
and writes aggregate counts only to /opt/topbuck/www/stats/summary.json (password protected)."""
import json, sys, subprocess, gzip, io, os, re, datetime as dt
from collections import Counter, defaultdict
from urllib.parse import urlsplit, parse_qs

OUT = sys.argv[1] if len(sys.argv) > 1 else "/opt/topbuck/www/stats/summary.json"
SRC = os.environ.get("TB_LOG_SRC")  # local folder for testing
TZ = dt.timezone(dt.timedelta(hours=2))
BOT = re.compile(r"bot|crawl|spider|slurp|preview|facebookexternalhit|curl|wget|python|monitor|headless|lighthouse", re.I)
SHORT = {"/sona", "/sona-fb", "/sona-x"}

def read_logs():
    if SRC:
        data = b""
        for f in sorted(os.listdir(SRC)):
            p = os.path.join(SRC, f)
            raw = open(p, "rb").read()
            data += gzip.decompress(raw) if f.endswith(".gz") else raw
        return data.decode("utf-8", "replace")
    cmd = ["docker", "compose", "exec", "-T", "caddy", "sh", "-c",
           "cd /data/topbuck-logs 2>/dev/null || exit 0; for f in *.gz; do [ -f \"$f\" ] && zcat \"$f\"; done; for f in *.log; do [ -f \"$f\" ] && cat \"$f\"; done"]
    r = subprocess.run(cmd, cwd="/opt/mycraft", capture_output=True, stdin=subprocess.DEVNULL, timeout=120)
    return r.stdout.decode("utf-8", "replace")

def is_page(path):
    if path.startswith("/stats"): return False
    last = path.rsplit("/", 1)[-1]
    return path.endswith("/") or last.endswith(".html") or "." not in last

def main():
    now = dt.datetime.now(TZ)
    since30 = (now - dt.timedelta(days=30)).date()
    since7 = (now - dt.timedelta(days=7)).date()
    daily = Counter(); pages7 = Counter(); refs7 = Counter(); short7 = Counter(); short_all = Counter()
    camp = defaultdict(Counter)  # campaign -> source -> landings (all time)
    camp7 = defaultdict(Counter)
    for line in read_logs().splitlines():
        try: e = json.loads(line)
        except Exception: continue
        q = e.get("request") or {}
        if q.get("method") != "GET": continue
        ua = " ".join((q.get("headers") or {}).get("User-Agent") or [])
        if not ua or BOT.search(ua): continue
        d = dt.datetime.fromtimestamp(e.get("ts", 0), TZ).date()
        u = urlsplit(q.get("uri", "/")); path = u.path or "/"
        st = e.get("status", 0)
        if path in SHORT and st in (301, 302):
            short_all[path] += 1
            if d >= since7: short7[path] += 1
            continue
        if st != 200 or not is_page(path): continue
        if d >= since30: daily[d.isoformat()] += 1
        qs = parse_qs(u.query)
        if "utm_source" in qs:
            c = qs.get("utm_campaign", ["(none)"])[0][:40]; s = qs["utm_source"][0][:40]
            camp[c][s] += 1
            if d >= since7: camp7[c][s] += 1
        if d >= since7:
            pages7[path] += 1
            ref = " ".join((q.get("headers") or {}).get("Referer") or [])
            host = urlsplit(ref).hostname or ""
            if host and not host.endswith("topbuck.app"):
                refs7[re.sub(r"^(www|m|l|lm|mobile)\.", "", host)] += 1
            elif not host:
                refs7["(typed in or app)"] += 1
    out = {
        "generated": now.strftime("%Y-%m-%d %H:%M") + " Gaborone time",
        "note": "Page views by real browsers only; no IPs, cookies or personal data are stored.",
        "page_views_by_day_30d": dict(sorted(daily.items())),
        "page_views_7d": sum(pages7.values()),
        "campaign_landings_all_time": {c: dict(v) for c, v in camp.items()},
        "campaign_landings_7d": {c: dict(v) for c, v in camp7.items()},
        "short_link_clicks_7d": dict(short7), "short_link_clicks_all_time": dict(short_all),
        "top_pages_7d": dict(pages7.most_common(15)),
        "arrived_from_7d": dict(refs7.most_common(15)),
    }
    os.makedirs(os.path.dirname(OUT), exist_ok=True)
    tmp = OUT + ".tmp"
    with open(tmp, "w") as f: json.dump(out, f, indent=1)
    os.chmod(tmp, 0o644); os.replace(tmp, OUT)

if __name__ == "__main__":
    main()
STATS
  chmod 755 "$APP/bin/topbuck-stats.py"
  printf '%s\n' 'SHELL=/bin/bash' '17 * * * * root /usr/bin/python3 /opt/topbuck/bin/topbuck-stats.py >/dev/null 2>&1' > /etc/cron.d/topbuck-stats
  chmod 644 /etc/cron.d/topbuck-stats
  python3 "$APP/bin/topbuck-stats.py" && echo "Visit counter: running (updates hourly)"
  echo "Undo: sudo cp $BK/Caddyfile $MYCRAFT/Caddyfile && sudo rm -rf $APP/www && sudo tar xzf $BK/www-before.tgz -C /opt/topbuck && cd $MYCRAFT && sudo docker compose restart caddy"
  echo "TOPBUCK TRACKING UPDATE DONE"
}
main "$@"
