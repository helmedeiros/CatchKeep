#!/usr/bin/env bash
# Smoke-test a built CatchKeep: start the binary against a throwaway database
# and the built frontend, then check the page, an asset, a client route and
# the API. Never touches real data.
#
# usage: scripts/smoke.sh <catchkeep-binary> <frontend-dist-dir>
set -euo pipefail

bin=${1:?binary required}
web=${2:?frontend dist dir required}
port=${SMOKE_PORT:-18080}
base="http://127.0.0.1:$port"

tmp=$(mktemp -d)
log="$tmp/serve.log"
"$bin" serve -addr "127.0.0.1:$port" -db "$tmp/smoke.db" -web-dir "$web" >"$log" 2>&1 &
pid=$!
cleanup() {
  kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
  rm -rf "$tmp"
}
trap cleanup EXIT

fail() {
  echo "smoke: FAIL: $*" >&2
  echo "--- server log" >&2
  cat "$log" >&2
  exit 1
}

for _ in $(seq 1 50); do
  curl -fsS "$base/api/v1/health" >/dev/null 2>&1 && break
  kill -0 "$pid" 2>/dev/null || fail "server exited during startup"
  sleep 0.2
done

health=$(curl -fsS "$base/api/v1/health") || fail "health endpoint unreachable"
[[ $health == *'"status":"ok"'* ]] || fail "unexpected health: $health"

index=$(curl -fsS "$base/") || fail "index unreachable"
[[ $index == *"<title>CatchKeep</title>"* ]] || fail "index has no CatchKeep title"

asset=$(grep -oE '/assets/[^"]+\.js' <<<"$index" | head -1)
[[ -n $asset ]] || fail "index references no JS asset"
curl -fsS -o /dev/null "$base$asset" || fail "asset $asset not served"

route=$(curl -fsS "$base/collection") || fail "client route not served"
[[ $route == *"<title>CatchKeep</title>"* ]] || fail "client route did not fall back to index"

echo "smoke: OK ($health)"
