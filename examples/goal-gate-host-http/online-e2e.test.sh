#!/bin/sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)"
cd "$ROOT_DIR"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

BIN_DIR="$TMP_DIR/bin"
mkdir -p "$BIN_DIR"

HOST_PYTHON=""
for candidate in python3 python; do
  if command -v "$candidate" >/dev/null 2>&1 && "$candidate" -c 'import json' >/dev/null 2>&1; then
    HOST_PYTHON="$(command -v "$candidate")"
    break
  fi
done
if [ -z "$HOST_PYTHON" ]; then
  echo "missing required command: python3 or python" >&2
  exit 1
fi
cat >"$BIN_DIR/python3" <<EOF
#!/bin/sh
exec "$HOST_PYTHON" "\$@"
EOF
chmod +x "$BIN_DIR/python3"

BIN="$TMP_DIR/goal-gate-host-http"
PORT="${PORT:-4070}"
BASE_URL="http://127.0.0.1:${PORT}"
APPROVE_PROPOSAL='{"version":"0.1","id":"proposal_goal_review_approve_http_e2e","steward":"goal-completion-reviewer","hook":"goal.before_complete","outputs":[{"action":"goal.approve_complete","reason":"integration test approved"}]}'
REJECT_PROPOSAL='{"version":"0.1","id":"proposal_goal_review_reject_http_e2e","steward":"goal-completion-reviewer","hook":"goal.before_complete","outputs":[{"action":"goal.reject_complete","reason":"deployment verification is still missing"},{"action":"goal.request_continue","reason":"continue with verification","instruction":"Deploy the image and run the smoke test."}]}'

go build -o "$BIN" ./examples/goal-gate-host-http

server_pid=""
start_server() {
  server_pid=""
  if [ -n "${1:-}" ]; then
    GATEWAY_HARNESS_TEST_GOAL_PROPOSAL="$1" "$BIN" -addr "127.0.0.1:${PORT}" >"$TMP_DIR/server.log" 2>&1 &
  else
    "$BIN" -addr "127.0.0.1:${PORT}" >"$TMP_DIR/server.log" 2>&1 &
  fi
  server_pid=$!
  for _ in $(seq 1 200); do
    if curl -fsS "$BASE_URL/api/healthz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.1
  done
  cat "$TMP_DIR/server.log" >&2
  return 1
}

stop_server() {
  if [ -n "$server_pid" ]; then
    kill "$server_pid" >/dev/null 2>&1 || true
    wait "$server_pid" >/dev/null 2>&1 || true
    server_pid=""
  fi
}

trap 'stop_server; rm -rf "$TMP_DIR"' EXIT INT TERM

fetch_json() {
  url="$1"
  out="$2"
  curl -fsS "$url" -o "$out"
}

assert_json_field() {
  file="$1"
  expr="$2"
  expected="$3"
  python3 - "$file" "$expr" "$expected" <<'PY'
import json
import sys

path = sys.argv[2].split(".")
expected = sys.argv[3]
obj = json.load(open(sys.argv[1], encoding="utf-8"))
cur = obj
for part in path:
    cur = cur[part]
if str(cur).lower() != expected.lower():
    raise SystemExit(f"unexpected value for {sys.argv[2]}: {cur!r} != {expected!r}")
PY
}

start_server ""

fetch_json "$BASE_URL/ui-demo.html" "$TMP_DIR/ui-demo.html"
grep -F "config-assistant" "$TMP_DIR/ui-demo.html" >/dev/null
grep -F "goal-gate-host.bundle.example.json" "$TMP_DIR/ui-demo.html" >/dev/null

fetch_json "$BASE_URL/goal-gate-host.bundle.example.json" "$TMP_DIR/bundle.json"
"$BIN_DIR/python3" - "$TMP_DIR/bundle.json" <<'PY'
import json
import sys

bundle = json.load(open(sys.argv[1], encoding="utf-8"))
if "interaction_model" not in bundle:
    raise SystemExit("bundle missing interaction_model")
if bundle["interaction_model"].get("primary_surface") != "chat_assistant":
    raise SystemExit(f"unexpected primary surface: {bundle['interaction_model']}")
PY

fetch_json "$BASE_URL/api/goal-gate/bundle" "$TMP_DIR/api-bundle.json"
"$BIN_DIR/python3" - "$TMP_DIR/api-bundle.json" <<'PY'
import json
import sys

bundle = json.load(open(sys.argv[1], encoding="utf-8"))
if bundle["interaction_model"]["primary_surface"] != "chat_assistant":
    raise SystemExit(f"unexpected api bundle surface: {bundle['interaction_model']}")
PY

fetch_json "$BASE_URL/api/goal-gate/example-request" "$TMP_DIR/example-request.json"
curl -fsS -H 'Content-Type: application/json' -d @"$TMP_DIR/example-request.json" "$BASE_URL/api/goal-gate/execute" -o "$TMP_DIR/approve-result.json"
"$BIN_DIR/python3" - "$TMP_DIR/approve-result.json" <<'PY'
import json
import sys

result = json.load(open(sys.argv[1], encoding="utf-8"))
if not result["enabled"] or not result["triggered"]:
    raise SystemExit(f"expected enabled triggered result: {result}")
if not result["sidecar"]["outcome"]["allow_complete"]:
    raise SystemExit(f"expected allow_complete: {result}")
if result["append_record"]["event"]["action"] != "goal.approve_complete":
    raise SystemExit(f"unexpected approve append record: {result['append_record']}")
PY

stop_server
start_server "$REJECT_PROPOSAL"

curl -fsS -H 'Content-Type: application/json' -d @"$TMP_DIR/example-request.json" "$BASE_URL/api/goal-gate/execute" -o "$TMP_DIR/reject-result.json"
"$BIN_DIR/python3" - "$TMP_DIR/reject-result.json" <<'PY'
import json
import sys

result = json.load(open(sys.argv[1], encoding="utf-8"))
if not result["sidecar"]["outcome"]["continue_work"]:
    raise SystemExit(f"expected continue_work: {result}")
instruction = result["sidecar"]["outcome"]["continue_instruction"]
if "smoke test" not in instruction:
    raise SystemExit(f"unexpected continue instruction: {instruction!r}")
if result["append_record"]["event"]["action"] != "goal.request_continue":
    raise SystemExit(f"unexpected reject append record: {result['append_record']}")
PY

stop_server
echo "goal-gate-host-http e2e ok"
