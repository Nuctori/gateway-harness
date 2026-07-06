#!/bin/sh
set -eu

DB_PATH="${DB_PATH:-/opt/new-api/data/one-api.db}"
POLICY_OUT="${POLICY_OUT:-/opt/gateway-harness/current/newapi-context-harness.policy.json}"
NEWAPI_URL="${NEWAPI_URL:-http://127.0.0.1:3000/}"
NEWAPI_BASE="${NEWAPI_BASE:-${NEWAPI_URL%/}}"
CONTAINER_NAME="${CONTAINER_NAME:-new-api}"
LIVE_SMOKE="${LIVE_SMOKE:-0}"
COMPACT_SMOKE="${COMPACT_SMOKE:-0}"
TRACE_SINCE="${TRACE_SINCE:-5m}"
SMOKE_MODEL="${SMOKE_MODEL:-gpt-5.4-mini}"
COMPACT_SMOKE_MODEL="${COMPACT_SMOKE_MODEL:-$SMOKE_MODEL}"
NEWAPI_API_KEY="${NEWAPI_API_KEY:-}"
NEWAPI_API_KEY_FILE="${NEWAPI_API_KEY_FILE:-}"
PRINT_ERROR_BODY="${PRINT_ERROR_BODY:-0}"

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

pass() {
  echo "ok: $1"
}

warn() {
  echo "warn: $1" >&2
}

maybe_print_error_body() {
  file="$1"
  if [ "$PRINT_ERROR_BODY" = "1" ]; then
    cat "$file" >&2
  else
    warn "response body suppressed; set PRINT_ERROR_BODY=1 to print it"
  fi
}

need curl
need docker
need gateway-harness
need python3

if [ -z "$NEWAPI_API_KEY" ] && [ -n "$NEWAPI_API_KEY_FILE" ] && [ -f "$NEWAPI_API_KEY_FILE" ]; then
  NEWAPI_API_KEY="$(tr -d '\r\n' < "$NEWAPI_API_KEY_FILE")"
fi

mkdir -p "$(dirname "$POLICY_OUT")"

python3 - "$DB_PATH" "$POLICY_OUT" <<'PY'
import sqlite3
import sys

db_path, policy_out = sys.argv[1:3]
conn = sqlite3.connect(db_path)
row = conn.execute("select value from options where key='context_harness.policy'").fetchone()
if not row or not row[0].strip():
    raise SystemExit("missing context_harness.policy")
open(policy_out, "w", encoding="utf-8").write(row[0])
PY
pass "exported context_harness.policy"

gateway-harness validate "$POLICY_OUT" >/dev/null
pass "policy validates against Gateway Harness"

python3 - "$DB_PATH" "$POLICY_OUT" <<'PY'
import json
import sqlite3
import sys

db_path, policy_path = sys.argv[1:3]
policy = json.load(open(policy_path, encoding="utf-8"))
raw = json.dumps(policy, ensure_ascii=False)
programs = policy.get("programs") or []
if not programs:
    raise SystemExit("policy has no programs")
if '"budget"' in raw:
    raise SystemExit("policy contains hidden budget field")
if "context.truncate" in raw:
    raise SystemExit("policy contains context.truncate")
if "context.inject_ledger_summary" not in raw:
    raise SystemExit("policy does not use context.inject_ledger_summary")
hooks = set()
for program in programs:
    for step in program.get("steps") or []:
        if step.get("hook"):
            hooks.add(step["hook"])
        hooks.update(step.get("hooks") or [])
required = {"chat.before_upstream", "responses.before_upstream", "responses.compact.before_upstream"}
missing = sorted(required - hooks)
if missing:
    raise SystemExit("policy missing hooks: " + ",".join(missing))

conn = sqlite3.connect(db_path)
rows = dict(conn.execute(
    "select key,value from options where key in ("
    "'context_harness.enabled',"
    "'model_failover_setting.enabled',"
    "'model_failover_setting.rules'"
    ")"
).fetchall())
if rows.get("context_harness.enabled") != "true":
    raise SystemExit("context_harness.enabled is not true")
if rows.get("model_failover_setting.enabled") != "true":
    raise SystemExit("model_failover_setting.enabled is not true")
rules = json.loads(rows.get("model_failover_setting.rules") or "[]")
if not rules:
    raise SystemExit("model_failover_setting.rules is empty")
if not all("403" in [part.strip() for part in str(rule.get("trigger_status_codes", "")).split(",")] for rule in rules):
    raise SystemExit("not every failover rule covers 403")
print("policy_programs=%d failover_rules=%d" % (len(programs), len(rules)))
PY
pass "policy transparency and failover options"

status="$(curl -sS -o /dev/null -w '%{http_code}' "$NEWAPI_URL")"
if [ "$status" != "200" ]; then
  echo "unexpected NewAPI status from $NEWAPI_URL: $status" >&2
  exit 1
fi
pass "NewAPI HTTP 200 on $NEWAPI_URL"

ports="$(docker ps --filter "name=$CONTAINER_NAME" --format '{{.Ports}}')"
case "$ports" in
  *3000-\>*|*3000/tcp*) pass "container exposes 3000" ;;
  *) echo "container does not expose 3000: $ports" >&2; exit 1 ;;
esac
case "$ports" in
  *0.0.0.0:80-\>*|*:::80-\>*)
    echo "container appears to publish port 80, refusing: $ports" >&2
    exit 1
    ;;
esac
pass "container does not publish port 80"

if [ "$LIVE_SMOKE" = "1" ] || [ -n "$NEWAPI_API_KEY" ]; then
  if [ -z "$NEWAPI_API_KEY" ]; then
    echo "LIVE_SMOKE=1 requires NEWAPI_API_KEY or NEWAPI_API_KEY_FILE" >&2
    exit 1
  fi
  smoke_file="$(mktemp)"
  smoke_status="$(curl -sS -o "$smoke_file" -w '%{http_code}' \
    -H "Authorization: Bearer $NEWAPI_API_KEY" \
    -H 'Content-Type: application/json' \
    -d "{\"model\":\"$SMOKE_MODEL\",\"input\":\"Gateway Harness online acceptance smoke. Reply with ok.\",\"stream\":false}" \
    "$NEWAPI_BASE/v1/responses")"
  if [ "$smoke_status" -lt 200 ] || [ "$smoke_status" -ge 300 ]; then
    echo "responses smoke failed with HTTP $smoke_status" >&2
    maybe_print_error_body "$smoke_file"
    rm -f "$smoke_file"
    exit 1
  fi
  rm -f "$smoke_file"
  pass "live /v1/responses smoke"

  if ! docker logs --since "$TRACE_SINCE" "$CONTAINER_NAME" 2>&1 |
    grep -F '"context_harness"' |
    grep -F '"source":"ledger.summary"' |
    grep -F '"content_mode":"redacted"' |
    grep -F '"request_path":"/v1/responses"' >/dev/null; then
    echo "missing redacted context_harness trace for /v1/responses in docker logs since $TRACE_SINCE" >&2
    exit 1
  fi
  pass "redacted /v1/responses harness trace"
else
  warn "skipping live /v1/responses smoke; set NEWAPI_API_KEY to enable"
fi

if [ "$COMPACT_SMOKE" = "1" ]; then
  if [ -z "$NEWAPI_API_KEY" ]; then
    echo "COMPACT_SMOKE=1 requires NEWAPI_API_KEY or NEWAPI_API_KEY_FILE" >&2
    exit 1
  fi
  compact_file="$(mktemp)"
  compact_status="$(curl -sS -o "$compact_file" -w '%{http_code}' \
    -H "Authorization: Bearer $NEWAPI_API_KEY" \
    -H 'Content-Type: application/json' \
    -d "{\"model\":\"$COMPACT_SMOKE_MODEL\",\"input\":\"Gateway Harness compact online acceptance smoke.\",\"instructions\":\"Keep the explicit project ledger sentinel.\"}" \
    "$NEWAPI_BASE/v1/responses/compact")"
  if [ "$compact_status" -lt 200 ] || [ "$compact_status" -ge 300 ]; then
    echo "responses compact smoke failed with HTTP $compact_status" >&2
    maybe_print_error_body "$compact_file"
    rm -f "$compact_file"
    exit 1
  fi
  rm -f "$compact_file"
  pass "live /v1/responses/compact smoke"

  if ! docker logs --since "$TRACE_SINCE" "$CONTAINER_NAME" 2>&1 |
    grep -F '"context_harness"' |
    grep -F '"source":"ledger.summary"' |
    grep -F '"content_mode":"redacted"' |
    grep -F '"request_path":"/v1/responses/compact"' >/dev/null; then
    echo "missing redacted context_harness trace for /v1/responses/compact in docker logs since $TRACE_SINCE" >&2
    exit 1
  fi
  pass "redacted /v1/responses/compact harness trace"
fi

echo "gateway-harness newapi online acceptance ok"
