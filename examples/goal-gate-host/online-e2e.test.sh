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

BIN="$TMP_DIR/goal-gate-host"
APPROVE_LEDGER="$TMP_DIR/approve.ledger.json"
REJECT_LEDGER="$TMP_DIR/reject.ledger.json"
DISABLED_LEDGER="$TMP_DIR/disabled.ledger.json"
APPROVE_PROPOSAL='{"version":"0.1","id":"proposal_goal_review_approve_e2e","steward":"goal-completion-reviewer","hook":"goal.before_complete","outputs":[{"action":"goal.approve_complete","reason":"integration test approved"}]}'
REJECT_PROPOSAL='{"version":"0.1","id":"proposal_goal_review_reject_e2e","steward":"goal-completion-reviewer","hook":"goal.before_complete","outputs":[{"action":"goal.reject_complete","reason":"deployment verification is still missing"},{"action":"goal.request_continue","reason":"continue with verification","instruction":"Deploy the image and run the smoke test."}]}'

go build -o "$BIN" ./examples/goal-gate-host

GATEWAY_HARNESS_TEST_GOAL_PROPOSAL="$APPROVE_PROPOSAL" "$BIN" -ledger "$APPROVE_LEDGER" >/dev/null
"$BIN_DIR/python3" - "$APPROVE_LEDGER" <<'PY'
import json
import sys

ledger = json.load(open(sys.argv[1], encoding="utf-8"))
event = ledger["projects"][0]["sessions"][0]["events"][0]
if event["action"] != "goal.approve_complete":
    raise SystemExit(f"unexpected approve action: {event}")
if event["hook"] != "goal.before_complete":
    raise SystemExit(f"unexpected approve hook: {event}")
PY

GATEWAY_HARNESS_TEST_GOAL_PROPOSAL="$REJECT_PROPOSAL" "$BIN" -ledger "$REJECT_LEDGER" >"$TMP_DIR/reject.out"
"$BIN_DIR/python3" - "$REJECT_LEDGER" "$TMP_DIR/reject.out" <<'PY'
import json
import sys

ledger = json.load(open(sys.argv[1], encoding="utf-8"))
event = ledger["projects"][0]["sessions"][0]["events"][0]
if event["action"] != "goal.request_continue":
    raise SystemExit(f"unexpected reject action: {event}")
if event["type"] != "harness_action":
    raise SystemExit(f"unexpected reject event type: {event}")

output = json.load(open(sys.argv[2], encoding="utf-8"))
if not output["continue_work"]:
    raise SystemExit(f"expected continue_work in output: {output}")
if "smoke test" not in output["continue_instruction"]:
    raise SystemExit(f"unexpected continue instruction: {output}")
PY

"$BIN" -config "$ROOT_DIR/examples/newapi/goal-gate.config.json" -ledger "$DISABLED_LEDGER" >/dev/null
if [ -e "$DISABLED_LEDGER" ]; then
  echo "disabled config unexpectedly wrote a ledger file" >&2
  exit 1
fi

echo "goal-gate-host e2e ok"
