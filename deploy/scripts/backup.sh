#!/usr/bin/env bash
#
# Backup PostgreSQL ke object storage (Biznet Gio NEO Object Storage atau ekuivalen).
# Jalankan via cron daily.
#
# Retention:
#   - daily/   7 file (7 hari terakhir)
#   - weekly/  4 file (4 minggu terakhir, hari Minggu)
#   - monthly/ 12 file (12 bulan terakhir, tanggal 1)
#
# Pre-req:
#   - rclone configured dengan remote name 'backup' (lihat deploy/scripts/setup-backup.sh)
#   - PostgreSQL pg_dump accessible
#   - env BACKUP_REMOTE (default 'backup:aplikasi-surat-backup')
#
# Usage (cron):
#   0 2 * * * /opt/aplikasi-surat-kecamatan/deploy/scripts/backup.sh

set -euo pipefail

DB_NAME="${DB_NAME:-surat}"
BACKUP_REMOTE="${BACKUP_REMOTE:-backup:aplikasi-surat-backup}"
TMPDIR="$(mktemp -d)"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
DAY_OF_WEEK="$(date -u +%u)"  # 1=Senin, 7=Minggu
DAY_OF_MONTH="$(date -u +%d)"

cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT

DUMP_FILE="$TMPDIR/${DB_NAME}-${TIMESTAMP}.sql.gz"

echo ">>> [$(date)] Backup start: $DB_NAME -> $DUMP_FILE"
pg_dump --format=custom --compress=9 "$DB_NAME" | gzip > "$DUMP_FILE"
SIZE=$(du -h "$DUMP_FILE" | awk '{print $1}')
echo ">>> Dump size: $SIZE"

# Always upload ke daily/
rclone copyto "$DUMP_FILE" "$BACKUP_REMOTE/daily/$(basename "$DUMP_FILE")"
echo ">>> Uploaded ke daily/"

# Hari Minggu juga upload ke weekly/
if [ "$DAY_OF_WEEK" = "7" ]; then
    rclone copyto "$DUMP_FILE" "$BACKUP_REMOTE/weekly/$(basename "$DUMP_FILE")"
    echo ">>> Uploaded ke weekly/ (Minggu)"
fi

# Tanggal 1 juga upload ke monthly/
if [ "$DAY_OF_MONTH" = "01" ]; then
    rclone copyto "$DUMP_FILE" "$BACKUP_REMOTE/monthly/$(basename "$DUMP_FILE")"
    echo ">>> Uploaded ke monthly/ (tanggal 1)"
fi

# Retention rotation: hapus dump yang lebih lama dari N file di tiap tier
prune() {
    local tier="$1"
    local keep="$2"
    rclone lsf "$BACKUP_REMOTE/$tier/" | sort -r | tail -n +$((keep + 1)) | while read -r f; do
        echo ">>> Prune $tier/$f"
        rclone delete "$BACKUP_REMOTE/$tier/$f"
    done
}
prune daily 7
prune weekly 4
prune monthly 12

echo ">>> [$(date)] Backup selesai"
