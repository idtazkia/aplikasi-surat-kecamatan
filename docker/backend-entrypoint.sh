#!/bin/sh
#
# Entrypoint backend container:
# 1. Apply schema migration (selalu, idempotent)
# 2. Apply demo seed kalau APPLY_DEMO_SEED=true
# 3. Start server

set -e

if [ -z "$DATABASE_URL" ]; then
    echo "FATAL: DATABASE_URL env var required"
    exit 1
fi

echo ">>> Applying schema migration..."
goose -dir /app/migrations/schema postgres "$DATABASE_URL" up

if [ "$APPLY_DEMO_SEED" = "true" ]; then
    echo ">>> Applying demo seed migration..."
    goose -dir /app/migrations/demo-seed -table goose_demo_seed_version postgres "$DATABASE_URL" up
fi

echo ">>> Starting server..."
exec /app/server
