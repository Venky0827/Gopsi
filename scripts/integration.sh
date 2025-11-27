#!/usr/bin/env bash
set -euo pipefail
GOPSI_BIN=${GOPSI_BIN:-"$(go env GOPATH)/bin/gopsi"}
"$GOPSI_BIN" inventory -list -i examples/inventory.yml
"$GOPSI_BIN" run -i examples/inventory.yml examples/play.yml -forks 1 -check
"$GOPSI_BIN" run -i examples/inventory.yml examples/play.yml -forks 1
