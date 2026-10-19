#!/usr/bin/env bash
# Developer tasks (stand-in for a Makefile; run from anywhere).
#   scripts/dev.sh run        start the server (migrate + seed on boot)
#   scripts/dev.sh test       go test ./...
#   scripts/dev.sh swag       regenerate docs/ from the swag annotations
#   scripts/dev.sh check      swag + fail if docs/ changed, then vet + test
#   scripts/dev.sh migrate    show applied/pending migrations, then apply
#   scripts/dev.sh reset-db   delete data/app.db (recreated on next run)
set -euo pipefail
cd "$(dirname "${BASH_SOURCE[0]}")/.."

swag_bin() { command -v swag || echo "$(go env GOPATH)/bin/swag"; }

case "${1:-}" in
  run)      go run . serve ;;
  test)     go test ./... ;;
  swag)     "$(swag_bin)" fmt && "$(swag_bin)" init -g main.go -o docs --parseInternal --quiet ;;
  check)
    "$(swag_bin)" fmt && "$(swag_bin)" init -g main.go -o docs --parseInternal --quiet
    if git rev-parse --git-dir >/dev/null 2>&1 && ! git diff --quiet -- docs/; then
      echo "docs/ was stale: commit the regenerated Swagger files" >&2; exit 1
    fi
    go vet ./... && go test ./... ;;
  migrate)  go run . migrate --dry-run && go run . migrate ;;
  reset-db) rm -f data/app.db data/app.db-wal data/app.db-shm && echo "removed data/app.db" ;;
  *)        sed -n '2,9p' "$0" | sed 's/^# \{0,1\}//'; exit 1 ;;
esac
