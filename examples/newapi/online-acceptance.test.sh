#!/bin/sh
set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

BIN_DIR="$TMP_DIR/bin"
mkdir -p "$BIN_DIR"

HOST_PYTHON=""
for candidate in python3 python; do
  if command -v "$candidate" >/dev/null 2>&1 && "$candidate" -c 'import sqlite3' >/dev/null 2>&1; then
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

DB_PATH="$TMP_DIR/one-api.db"
POLICY_OUT="$TMP_DIR/exported-policy.json"
TOKEN_FILE="$TMP_DIR/newapi-token"
STDOUT_FILE="$TMP_DIR/stdout"
STDERR_FILE="$TMP_DIR/stderr"

cat >"$TOKEN_FILE" <<'EOF'
sk-test-token
EOF

"$BIN_DIR/python3" - "$DB_PATH" <<'PY'
import json
import sqlite3
import sys

db_path = sys.argv[1]
policy = {
    "programs": [
        {
            "name": "ci-ledger-sentinel",
            "models": ["*"],
            "steps": [
                {
                    "hooks": [
                        "chat.before_upstream",
                        "responses.before_upstream",
                        "responses.compact.before_upstream",
                    ],
                    "do": [
                        {
                            "action": "context.inject_ledger_summary",
                            "source": "ledger.summary",
                            "ledger_ref": "ledger://ci/newapi/current",
                            "text": "CI ledger sentinel.",
                        }
                    ],
                }
            ],
        }
    ]
}
rules = [
    {"name": "primary", "trigger_status_codes": "403,429,500-599", "fallback_models": ["fallback"]},
    {"name": "fallback", "trigger_status_codes": "403", "fallback_models": ["last"]},
]
conn = sqlite3.connect(db_path)
conn.execute("create table options (key text primary key, value text)")
conn.executemany(
    "insert into options(key, value) values (?, ?)",
    [
        ("context_harness.enabled", "true"),
        ("context_harness.policy", json.dumps(policy)),
        ("model_failover_setting.enabled", "true"),
        ("model_failover_setting.rules", json.dumps(rules)),
    ],
)
conn.commit()
PY

cat >"$BIN_DIR/gateway-harness" <<'EOF'
#!/bin/sh
set -eu
if [ "$1" != "validate" ]; then
  echo "unexpected gateway-harness command: $*" >&2
  exit 1
fi
python3 - "$2" <<'PY'
import json
import sys
json.load(open(sys.argv[1], encoding="utf-8"))
PY
EOF
chmod +x "$BIN_DIR/gateway-harness"

cat >"$BIN_DIR/docker" <<'EOF'
#!/bin/sh
set -eu
cmd="$1"
shift
case "$cmd" in
  ps)
    echo '0.0.0.0:3000->3000/tcp, :::3000->3000/tcp'
    ;;
  logs)
    cat <<'LOG'
{"admin_info":{"context_harness":{"operations":[{"source":"ledger.summary"}],"summary":{"content_mode":"redacted"}}},"request_path":"/v1/responses"}
{"admin_info":{"context_harness":{"operations":[{"source":"ledger.summary"}],"summary":{"content_mode":"redacted"}}},"request_path":"/v1/responses/compact"}
LOG
    ;;
  *)
    echo "unexpected docker command: $cmd $*" >&2
    exit 1
    ;;
esac
EOF
chmod +x "$BIN_DIR/docker"

cat >"$BIN_DIR/curl" <<'EOF'
#!/bin/sh
set -eu
out_file=""
write_format=""
url=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o)
      out_file="$2"
      shift 2
      ;;
    -w)
      write_format="$2"
      shift 2
      ;;
    -H|-d)
      shift 2
      ;;
    -sS|-fsS)
      shift
      ;;
    *)
      url="$1"
      shift
      ;;
  esac
done

if [ -n "$out_file" ] && [ "$out_file" != "/dev/null" ]; then
  if [ "${MOCK_CURL_FAIL_RESPONSES:-0}" = "1" ] && echo "$url" | grep -q '/v1/responses$'; then
    printf '%s\n' 'SECRET_RESPONSE_BODY_SHOULD_NOT_PRINT' >"$out_file"
    printf '500'
    exit 0
  fi
  printf '%s\n' '{"output_text":"ok"}' >"$out_file"
fi

case "$write_format" in
  *http_code*size_download*) printf 'http_code=200 size=16\n' ;;
  *http_code*) printf '200' ;;
esac
EOF
chmod +x "$BIN_DIR/curl"

run_acceptance() {
  PATH="$BIN_DIR:$PATH" \
  DB_PATH="$DB_PATH" \
  POLICY_OUT="$POLICY_OUT" \
  NEWAPI_URL="http://127.0.0.1:3000/" \
  CONTAINER_NAME="new-api" \
  sh "$SCRIPT_DIR/online-acceptance.sh"
}

NEWAPI_API_KEY="" NEWAPI_API_KEY_FILE="" LIVE_SMOKE=0 COMPACT_SMOKE=0 PRINT_ERROR_BODY=0 \
  run_acceptance >"$STDOUT_FILE" 2>"$STDERR_FILE"
grep -F 'gateway-harness newapi online acceptance ok' "$STDOUT_FILE" >/dev/null
grep -F 'skipping live /v1/responses smoke' "$STDERR_FILE" >/dev/null

NEWAPI_API_KEY="" NEWAPI_API_KEY_FILE="$TOKEN_FILE" LIVE_SMOKE=0 COMPACT_SMOKE=0 PRINT_ERROR_BODY=0 \
  run_acceptance >"$STDOUT_FILE" 2>"$STDERR_FILE"
grep -F 'ok: live /v1/responses smoke' "$STDOUT_FILE" >/dev/null
grep -F 'ok: redacted /v1/responses harness trace' "$STDOUT_FILE" >/dev/null

NEWAPI_API_KEY="" NEWAPI_API_KEY_FILE="$TOKEN_FILE" LIVE_SMOKE=0 COMPACT_SMOKE=1 PRINT_ERROR_BODY=0 \
  run_acceptance >"$STDOUT_FILE" 2>"$STDERR_FILE"
grep -F 'ok: live /v1/responses smoke' "$STDOUT_FILE" >/dev/null
grep -F 'ok: live /v1/responses/compact smoke' "$STDOUT_FILE" >/dev/null
grep -F 'ok: redacted /v1/responses/compact harness trace' "$STDOUT_FILE" >/dev/null

if NEWAPI_API_KEY="" NEWAPI_API_KEY_FILE="$TOKEN_FILE" LIVE_SMOKE=0 COMPACT_SMOKE=0 PRINT_ERROR_BODY=0 MOCK_CURL_FAIL_RESPONSES=1 \
  run_acceptance >"$STDOUT_FILE" 2>"$STDERR_FILE"; then
  echo "expected live smoke failure" >&2
  exit 1
fi
grep -F 'response body suppressed' "$STDERR_FILE" >/dev/null
if grep -F 'SECRET_RESPONSE_BODY_SHOULD_NOT_PRINT' "$STDERR_FILE" >/dev/null; then
  echo "live-smoke failure body leaked despite default suppression" >&2
  exit 1
fi

echo "newapi online acceptance mock test ok"
