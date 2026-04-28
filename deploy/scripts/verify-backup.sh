#!/usr/bin/env bash
#
# Verify backup integrity dengan restore ke staging DB + smoke test.
# Jalankan via cron weekly (mis. Senin pagi setelah backup Minggu).
#
# Pre-req:
#   - rclone remote 'backup' configured
#   - staging DB accessible: env STAGING_DATABASE_URL
#   - env BACKUP_REMOTE
#
# Exit non-zero kalau gagal, supaya cron alert via MAILTO.

set -euo pipefail

BACKUP_REMOTE="${BACKUP_REMOTE:-backup:aplikasi-surat-backup}"
STAGING_DB="${STAGING_DATABASE_URL:?STAGING_DATABASE_URL required}"
TMPDIR="$(mktemp -d)"

cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT

echo ">>> [$(date)] Verify start"

# 1. Get latest daily backup
LATEST=$(rclone lsf "$BACKUP_REMOTE/daily/" | sort -r | head -1)
if [ -z "$LATEST" ]; then
    echo "FAIL: tidak ada backup di $BACKUP_REMOTE/daily/"
    exit 1
fi
echo ">>> Latest backup: $LATEST"

# 2. Download
rclone copy "$BACKUP_REMOTE/daily/$LATEST" "$TMPDIR/"
DUMP_PATH="$TMPDIR/$LATEST"
echo ">>> Downloaded: $(du -h "$DUMP_PATH" | awk '{print $1}')"

# 3. Restore ke staging (drop & recreate DB nama)
STAGING_DBNAME=$(echo "$STAGING_DB" | sed 's|.*/||' | sed 's|?.*||')
echo ">>> Drop & recreate $STAGING_DBNAME"
psql "${STAGING_DB%/$STAGING_DBNAME*}/postgres" -c "DROP DATABASE IF EXISTS $STAGING_DBNAME;"
psql "${STAGING_DB%/$STAGING_DBNAME*}/postgres" -c "CREATE DATABASE $STAGING_DBNAME;"

echo ">>> Restoring..."
gunzip < "$DUMP_PATH" | pg_restore --dbname="$STAGING_DB" --no-owner --no-privileges

# 4. Smoke test: query critical tables
echo ">>> Smoke test queries"
USERS_COUNT=$(psql "$STAGING_DB" -tAc "SELECT COUNT(*) FROM users WHERE is_active")
SURAT_COUNT=$(psql "$STAGING_DB" -tAc "SELECT COUNT(*) FROM surat WHERE NOT is_deleted")
AUDIT_COUNT=$(psql "$STAGING_DB" -tAc "SELECT COUNT(*) FROM audit_log")

echo "    users (aktif): $USERS_COUNT"
echo "    surat (active): $SURAT_COUNT"
echo "    audit_log: $AUDIT_COUNT"

if [ "$USERS_COUNT" -lt 1 ]; then
    echo "FAIL: tidak ada user aktif di restore"
    exit 1
fi

echo ">>> [$(date)] Verify OK"
