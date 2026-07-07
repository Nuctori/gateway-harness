#!/usr/bin/env sh
set -eu

mkdir -p dist
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -o dist/harness-linux-armv7 ./cmd/harness
