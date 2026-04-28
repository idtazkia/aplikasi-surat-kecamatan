#!/usr/bin/env bash
#
# Issue TLS certificate via acme.sh untuk aplikasi-surat-kecamatan.
# Jalankan sekali saat setup awal di VPS Biznet Gio.
#
# Usage:
#   sudo bash setup-tls.sh <domain>
#
# Contoh:
#   sudo bash setup-tls.sh surat.example.com

set -euo pipefail

if [ "${1:-}" = "" ]; then
    echo "usage: $0 <domain>"
    exit 1
fi

DOMAIN="$1"
ACME_HOME="/root/.acme.sh"
NGINX_SSL_DIR="/etc/nginx/ssl"
ACME_WEBROOT="/var/www/acme"

# 1. Install acme.sh kalau belum
if [ ! -d "$ACME_HOME" ]; then
    echo ">>> Installing acme.sh..."
    curl https://get.acme.sh | sh -s email=admin@"$DOMAIN"
fi

# 2. Setup webroot untuk ACME challenge
mkdir -p "$ACME_WEBROOT"
chown -R www-data:www-data "$ACME_WEBROOT"

# 3. Issue certificate (Let's Encrypt)
"$ACME_HOME/acme.sh" --issue \
    -d "$DOMAIN" \
    --webroot "$ACME_WEBROOT" \
    --server letsencrypt

# 4. Install ke /etc/nginx/ssl/
mkdir -p "$NGINX_SSL_DIR"
"$ACME_HOME/acme.sh" --install-cert -d "$DOMAIN" \
    --key-file       "$NGINX_SSL_DIR/$DOMAIN.key" \
    --fullchain-file "$NGINX_SSL_DIR/$DOMAIN.crt" \
    --reloadcmd      "systemctl reload nginx"

echo ">>> Done. Cert path:"
echo "    $NGINX_SSL_DIR/$DOMAIN.crt"
echo "    $NGINX_SSL_DIR/$DOMAIN.key"
echo ""
echo "Auto-renew akan jalan via cron yang acme.sh setup otomatis."
