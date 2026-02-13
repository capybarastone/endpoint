#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${ROOT_DIR}/bin"
APP_NAME="capyendpoint"

usage() {
  cat <<'EOF'
Usage: ./make.sh [command]

Commands:
  format   Run go fmt ./...
  vet      Run go vet ./...
  test     Run go test ./...
  build    Build the project into ./bin
  clean    Remove ./bin
  all      Run format, vet, test, then build (default)
EOF
}

cmd_format() {
  GOFLAGS="" go fmt ./...
}

cmd_vet() {
  go vet ./...
}

cmd_test() {
  # TODO: we don't have tests :c
  go test ./...
}

cmd_build() {
  mkdir -p "${BIN_DIR}"
  go build -o "${BIN_DIR}/${APP_NAME}" .
  cp config.toml bin/
}

cmd_build_win() {
    mkdir -p "${BIN_DIR}"
    GOOS=windows go build -o "${BIN_DIR}/${APP_NAME}.exe" .
    cp config.toml bin/
  }

cmd_clean() {
  rm -rf "${BIN_DIR}"
}

cmd_all() {
  cmd_format
  cmd_vet
  cmd_test
  cmd_build
}

COMMAND="${1:-all}"

case "${COMMAND}" in
  format) cmd_format ;;
  vet) cmd_vet ;;
  test) cmd_test ;;
  build) cmd_build ;;
  buildwin) cmd_build_win ;;
  clean) cmd_clean ;;
  all) cmd_all ;;
  -h|--help|help) usage ;;
  *)
    echo "Unknown command: ${COMMAND}" >&2
    usage
    exit 1
    ;;
esac
