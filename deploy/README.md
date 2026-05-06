# Deployment

Ansible-based provisioning untuk multi-tenant deployment. Satu tenant = satu kecamatan,
masing-masing dapat instance app + database terpisah, bisa shared VPS atau dedicated VPS.

Konfigurasi per-tenant (inventory + secrets) tinggal di repo terpisah:
**[idtazkia/aplikasi-surat-kecamatan-deploy](https://github.com/idtazkia/aplikasi-surat-kecamatan-deploy)** (private).

Repo ini hanya berisi playbook + role generic, tidak ada nama tenant atau secret apa pun.

## Struktur

```
deploy/ansible/
├── ansible.cfg
├── requirements.yml                # collections deps (community.postgresql, ansible.posix)
├── inventory.ini.example           # template inventory
├── group_vars/all.yml.example      # template per-tenant config (semua var REQUIRED)
├── site.yml                        # full provision satu tenant
├── deploy.yml                      # update binary + dist (rolling, fast)
├── setup-ssl.yml                   # issue Let's Encrypt cert via certbot
├── backup.yml                      # on-demand backup
├── teardown.yml                    # hapus tenant (DESTRUCTIVE, butuh -e confirm=YES)
└── roles/
    ├── validate           # pre_tasks gate semua var WAJIB terisi (no fallback)
    ├── host-bootstrap     # apt: nginx, postgresql-client, certbot, rclone, goose
    ├── tenant-user        # Linux user surat-<tenant>, /opt/surat/<tenant>/ skeleton
    ├── postgres-tenant    # CREATE ROLE + DATABASE di cluster yang sudah jalan
    ├── surat-app          # build binary di laptop, sync, env, systemd, goose migrate
    ├── surat-frontend     # build dist di laptop, sync, render config.json
    ├── nginx-vhost        # vhost template (auto HTTP→HTTPS sesuai cert state)
    ├── surat-tls          # certbot --nginx (certonly), re-render vhost ke HTTPS
    └── surat-backup       # pg_dump + rclone, daily cron + opsional weekly verify
```

## Asumsi host

- Ubuntu/Debian dengan systemd, sudo untuk `ansible_user`
- PostgreSQL **server cluster** sudah jalan di host (Ansible tidak menginstall server,
  hanya client + role/db tenant). Cluster version harus match `postgresql_client_version`.
- nginx + cron sudah ada (kalau belum, `host-bootstrap` akan apt-install)
- Untuk tenant TLS: certbot bisa reach `app_domain` lewat HTTP-01 (DNS A-record sudah
  point ke `ansible_host`, port 80 reachable)

## Konvensi tenant

- `tenant_id`: slug `[a-z0-9-]+`, ≤24 char (mis. `kec-bogor-tengah`, `stmik-tazkia-test`)
- `app_user`: Linux system user `surat-<tenant_id>`, no shell login, ≤31 char total
- `app_dir`: `/opt/surat/<tenant_id>/` berisi `bin/`, `web/`, `etc/`, `data/`,
  `migrations/`, `backup/`, `log/`
- `systemd_unit`: `surat-<tenant_id>.service`
- `app_port`: TCP loopback unik per tenant kalau shared VPS
- nginx vhost: `surat-<tenant_id>.conf`, log `/var/log/nginx/surat-<tenant_id>.{access,error}.log`
- DB: `db_name`/`db_user` ditentukan operator (recommended pattern: `surat_<tenant_id_underscore>`)

## Quick start

```sh
# 1. Clone deploy repo (sibling dari app repo)
git clone git@github.com:idtazkia/aplikasi-surat-kecamatan-deploy.git
cd aplikasi-surat-kecamatan-deploy

# 2. Install Ansible deps (sekali per laptop)
ansible-galaxy install -r ../aplikasi-surat-kecamatan/deploy/ansible/requirements.yml

# 3. Provision tenant (full setup)
./deploy.sh stmik-tazkia-test site.yml

# 4. Update binary + dist (fast path, setelah ada perubahan code)
./deploy.sh stmik-tazkia-test deploy.yml

# 5. Issue/renew TLS cert
./deploy.sh stmik-tazkia-test setup-ssl.yml

# 6. Manual backup
./deploy.sh stmik-tazkia-test backup.yml

# 7. Teardown (DESTRUCTIVE)
./deploy.sh stmik-tazkia-test teardown.yml -e confirm=YES
```

## Validation gates

Semua var di `group_vars/all.yml` WAJIB terisi. Tidak ada default value. Playbook akan
fail di `pre_tasks` kalau:

- ada placeholder `CHANGE_THIS_*` yang belum diganti
- `tenant_id` tidak match `^[a-z0-9-]+$` atau >24 char
- `app_user` >31 char
- `jwt_secret` <32 char
- `app_port` di luar 1024–65535
- `tls_email`/`backup_rclone_remote`/`backup_verify_staging_db_url` placeholder saat
  fitur masing-masing di-enable

## Initial admin user

Schema migration tidak seed admin. Setelah `site.yml` selesai, buat admin pertama:

```sh
# Register via API
curl -X POST https://<app_domain>/api/auth/register \
    -H 'Content-Type: application/json' \
    -d '{"email":"admin@<tenant>","password":"<strong_password>","full_name":"<name>"}'

# Grant role admin via SQL
PGPASSWORD='<db_password>' psql -h localhost -U <db_user> -d <db_name> <<SQL
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u, roles r
WHERE u.email = 'admin@<tenant>' AND r.name = 'admin';
SQL
```

## Catatan: runtime config (issue #1)

Role `surat-frontend` me-render `{{ app_dir }}/web/config.json` dari `brand_*` vars.
Sebelum [#1](https://github.com/idtazkia/aplikasi-surat-kecamatan/issues/1) merge,
file ini di-render tapi belum dibaca aplikasi — semua tenant tampil dengan branding
default Vue dist/. Setelah #1 selesai, tenant branding aktif tanpa rebuild dist/.
