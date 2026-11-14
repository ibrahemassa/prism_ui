#!/usr/bin/env bash
# Start the read-only Prism API.
# Credentials are NOT stored here — they are read at runtime from the Prism
# install/dump folder (Model.dll.config + DBList.txt) via PRISM_CONFIG_DIR,
# or from PRISM_DSN / PRISM_DB_* env vars if you prefer.
set -euo pipefail
cd "$(dirname "$0")"

# writable Go cache (the mounted drive default is not writable)
export GOCACHE="${GOCACHE:-$HOME/.cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-$HOME/go/pkg/mod}"
export GOPATH="${GOPATH:-$HOME/go}"

# Settings (DB, CRM, commissions) come from .env — see .env.example.
# Optional live CRM enrichment: set PRISM_CRM_DSN (read-only Postgres DSN);
# left unset, CRM enrichment simply stays off.
[ -f .env ] && set -a && . ./.env && set +a
export PRISM_UI_ADDR="${PRISM_UI_ADDR:-:8080}"

echo "Prism config dir: ${PRISM_CONFIG_DIR:-(unset; using PRISM_DSN / PRISM_DB_*)}"
echo "CRM enrichment  : $([ -n "${PRISM_CRM_DSN:-}" ] && echo on || echo off)"
echo "CRM sync         : $([ -n "${CRM_INGEST_URL:-}" ] && [ -n "${CRM_INGEST_KEY:-}" ] && echo "on → $CRM_INGEST_URL" || echo off)"
echo "Listening on     $PRISM_UI_ADDR  (read-only)"
go run .
