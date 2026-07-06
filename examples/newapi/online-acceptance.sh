#!/bin/sh
set -eu

DB_PATH="${DB_PATH:-/opt/new-api/data/one-api.db}"
POLICY_OUT="${POLICY_OUT:-/opt/gateway-harness/current/newapi-context-harness.policy.json}"
NEWAPI_URL="${NEWAPI_URL:-http://127.0.0.1:3000/}"
CONTAINER_NAME="${CONTAINER_NAME:-new-api}"

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

pass() {
  echo "ok: $1"
}

need curl
need docker
need gateway-harness
need python3

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

echo "gateway-harness newapi online acceptance ok"
