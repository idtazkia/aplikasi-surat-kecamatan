#!/usr/bin/env bash
#
# Apply schema migration ke production DB.
# WAJIB: env DATABASE_URL terisi.
#
# Production hanya apply schema, tidak demo-seed.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
SCHEMA_DIR="$REPO_ROOT/db/migrations/schema"

if [ -z "${DATABASE_URL:-}" ]; then
    echo "error: DATABASE_URL belum di-set"
    echo "       export DATABASE_URL='postgres://...'"
    exit 1
fi

if ! command -v goose >/dev/null 2>&1; then
    echo "error: goose belum terinstall. Jalankan: go install github.com/pressly/goose/v3/cmd/goose@v3.22.1"
    exit 1
fi

echo ">>> Schema status sebelum:"
goose -dir "$SCHEMA_DIR" postgres "$DATABASE_URL" status

echo ""
echo ">>> Apply schema migration..."
goose -dir "$SCHEMA_DIR" postgres "$DATABASE_URL" up

echo ""
echo ">>> Schema status sesudah:"
goose -dir "$SCHEMA_DIR" postgres "$DATABASE_URL" status
