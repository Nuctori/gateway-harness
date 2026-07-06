# Gateway Harness Demo Stack

This bundle is the smallest "just run it" package for trying NewAPI and Goal Gate side by side.

It starts two things:

- `NewAPI` in Docker, using the official `calciumion/new-api:latest` image on port `3000`
- `goal-gate-host-http`, the thin Goal Gate host demo on port `4070`
- `goalreviewhelper`, the bundled deterministic runner that proves the Goal Gate contract without
  external model credentials

The stack is intentionally explicit. It does not hide the fact that Goal Gate is a separate host
sidecar. That keeps the core Gateway Harness contract clean while still giving users one unpacked
bundle they can run immediately.

## Layout

- `docker-compose.yml` launches both containers
- `Dockerfile.goal-gate-host-http` builds a small image from the bundled `goal-gate-host-http`
  and `goalreviewhelper` binaries
- `goal-gate.demo.config.json` is the default Goal Gate host config for the demo

## Run

From the unpacked release bundle root:

```bash
docker compose -f demo-stack/docker-compose.yml up -d --build
```

Then open:

- NewAPI: `http://127.0.0.1:3000`
- Goal Gate demo: `http://127.0.0.1:4070/ui-demo.html`

## What to expect

- NewAPI comes up as the adapter host example.
- The Goal Gate demo exposes the chat-first config assistant and the manual settings drawer.
- The Goal Gate host uses the bundled deterministic helper by default, so the completion-review
  path works without API keys.
- Nothing auto-enables or auto-applies. You still have to click the explicit apply action in the UI.

If you want the live NewAPI acceptance checks or a real smolagents-based steward runner, run the
scripts in `examples/newapi/` and `examples/smolagents/` after the stack is up.
