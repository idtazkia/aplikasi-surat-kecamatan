# Deployment

Setup di VPS Biznet Gio.

## Prerequisites

- Ubuntu/Debian server dengan systemd
- PostgreSQL 14+ terpasang dan running
- nginx terpasang
- Go 1.22+ untuk install tooling (goose) di server
- rclone untuk backup ke object storage

## Initial Setup

```sh
# 1. Clone repo (atau scp build artifact)
sudo git clone https://github.com/idtazkia/aplikasi-surat-kecamatan.git /opt/aplikasi-surat-kecamatan-src
cd /opt/aplikasi-surat-kecamatan-src

# 2. Initial setup script (create user, dirs, env template, systemd unit)
sudo bash deploy/scripts/install.sh

# 3. Edit env dengan credentials asli
sudo nano /etc/aplikasi-surat-kecamatan/env

# 4. Setup PostgreSQL user + DB
sudo -u postgres psql <<EOF
CREATE USER surat WITH PASSWORD 'GANTI_INI';
CREATE DATABASE surat OWNER surat;
GRANT ALL PRIVILEGES ON DATABASE surat TO surat;
EOF

# 5. Install goose, apply schema migration
go install github.com/pressly/goose/v3/cmd/goose@v3.22.1
export DATABASE_URL='postgres://surat:GANTI_INI@localhost:5432/surat?sslmode=disable'
sudo -E bash deploy/scripts/migrate.sh

# 6. Build server binary
go build -o /opt/aplikasi-surat-kecamatan/server ./cmd/server
sudo chown surat:surat /opt/aplikasi-surat-kecamatan/server

# 7. Build frontend
cd web && npm ci && npm run build
sudo cp -r dist/* /var/www/aplikasi-surat-kecamatan/

# 8. Setup TLS
sudo bash deploy/scripts/setup-tls.sh surat.example.com

# 9. Apply nginx config (edit server_name dulu)
sudo cp deploy/nginx/aplikasi-surat-kecamatan.conf /etc/nginx/sites-available/
sudo ln -s /etc/nginx/sites-available/aplikasi-surat-kecamatan.conf /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

# 10. Start service
sudo systemctl start aplikasi-surat-kecamatan
sudo systemctl status aplikasi-surat-kecamatan

# 11. Health check
curl https://surat.example.com/healthz
```

## Backup Setup

```sh
# 1. Setup rclone dengan Biznet Gio NEO Object Storage (S3-compatible)
rclone config
# Pilih: New remote, name 'backup', type 's3', provider 'Other',
#         endpoint 'https://s3-jakarta.biznetcloud.com' atau region yang dipakai

# 2. Test rclone
rclone ls backup:aplikasi-surat-backup

# 3. Setup cron untuk daily backup
sudo crontab -e
# Tambah:
# 0 2 * * * /opt/aplikasi-surat-kecamatan-src/deploy/scripts/backup.sh > /var/log/aplikasi-surat-backup.log 2>&1

# 4. Setup cron untuk weekly verify (Senin pagi)
# 0 6 * * 1 STAGING_DATABASE_URL='postgres://...' /opt/aplikasi-surat-kecamatan-src/deploy/scripts/verify-backup.sh > /var/log/aplikasi-surat-verify.log 2>&1
```

## Update Deployment

```sh
cd /opt/aplikasi-surat-kecamatan-src
sudo git pull
go build -o /tmp/server ./cmd/server
sudo systemctl stop aplikasi-surat-kecamatan
sudo bash deploy/scripts/migrate.sh   # apply migration baru kalau ada
sudo cp /tmp/server /opt/aplikasi-surat-kecamatan/server
sudo chown surat:surat /opt/aplikasi-surat-kecamatan/server
sudo systemctl start aplikasi-surat-kecamatan

# Frontend
cd web && npm ci && npm run build
sudo cp -r dist/* /var/www/aplikasi-surat-kecamatan/
```

## Monitoring

- App log: `journalctl -u aplikasi-surat-kecamatan -f`
- Nginx access log: `/var/log/nginx/aplikasi-surat.access.log`
- Health endpoint: `https://surat.example.com/healthz` — daftarkan ke uptime monitor eksternal (UptimeRobot atau ekuivalen)
- Backup verify log: `/var/log/aplikasi-surat-verify.log` — alert kalau tidak ada update mingguan
