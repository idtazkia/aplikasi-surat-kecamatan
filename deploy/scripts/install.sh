#!/usr/bin/env bash
#
# Initial setup script untuk VPS Biznet Gio.
# Jalankan sebagai root sekali saat provisioning.

set -euo pipefail

INSTALL_DIR="/opt/aplikasi-surat-kecamatan"
DATA_DIR="/var/lib/aplikasi-surat-kecamatan"
ENV_DIR="/etc/aplikasi-surat-kecamatan"
SERVICE_USER="surat"

echo ">>> Membuat user sistem '$SERVICE_USER'..."
if ! id -u "$SERVICE_USER" >/dev/null 2>&1; then
    useradd --system --no-create-home --shell /usr/sbin/nologin "$SERVICE_USER"
fi

echo ">>> Membuat direktori..."
mkdir -p "$INSTALL_DIR" "$DATA_DIR" "$ENV_DIR"
chown -R "$SERVICE_USER:$SERVICE_USER" "$DATA_DIR"
chmod 750 "$DATA_DIR"
chmod 750 "$ENV_DIR"

echo ">>> Menyiapkan env template..."
if [ ! -f "$ENV_DIR/env" ]; then
    cat > "$ENV_DIR/env" <<'EOF'
DATABASE_URL=postgres://surat:GANTI_DENGAN_PASSWORD@localhost:5432/surat?sslmode=disable
JWT_SECRET=GANTI_DENGAN_RANDOM_BASE64_MIN_32_BYTE
LISTEN_ADDR=127.0.0.1:8080
LOG_LEVEL=info
STUDENT_MODE_ENABLED=false
ATTACHMENT_STORAGE_PATH=/var/lib/aplikasi-surat-kecamatan/attachments
EOF
    chmod 640 "$ENV_DIR/env"
    chown root:"$SERVICE_USER" "$ENV_DIR/env"
    echo "    Edit $ENV_DIR/env dan ganti placeholder."
fi

echo ">>> Install systemd unit..."
cp "$(dirname "$0")/../systemd/aplikasi-surat-kecamatan.service" /etc/systemd/system/
systemctl daemon-reload
systemctl enable aplikasi-surat-kecamatan.service

echo ""
echo ">>> Setup selesai. Langkah selanjutnya:"
echo "    1. Edit $ENV_DIR/env dengan credential PostgreSQL dan JWT_SECRET asli"
echo "    2. Setup PostgreSQL: createdb surat; createuser surat; grant"
echo "    3. Apply schema migration: lihat deploy/scripts/migrate.sh"
echo "    4. Copy server binary ke $INSTALL_DIR/server (chown $SERVICE_USER)"
echo "    5. Copy frontend dist/ ke /var/www/aplikasi-surat-kecamatan/"
echo "    6. Setup TLS: bash deploy/scripts/setup-tls.sh <domain>"
echo "    7. Apply nginx config: cp deploy/nginx/*.conf /etc/nginx/sites-available/ && ln -s ... && nginx -t && systemctl reload nginx"
echo "    8. Start service: systemctl start aplikasi-surat-kecamatan"
