# aplikasi-surat-kecamatan

Aplikasi manajemen surat masuk dan surat keluar untuk kantor kecamatan.

Proyek PkM STMIK Tazkia. Dibangun untuk melengkapi aplikasi surat keluar yang diterbitkan pemerintah — aplikasi ini menjadi arsip lokal yang dapat dioperasikan saat aplikasi pemerintah tidak tersedia, sekaligus menjadi sistem utama untuk surat masuk.

## Mandat

1. **Operasional kantor kecamatan**: pengganti filing fisik, handle outage aplikasi pemerintah lewat PWA offline.
2. **Edukasi mahasiswa STMIK Tazkia**: aplikasi punya student-mode dan concept catalog publik yang link source code ke konsep matkul **Struktur Data**, **Algoritma**, dan **Basis Data**.

## Cakupan

- **Surat masuk**: sistem utama (belum ada aplikasi pemerintah yang menggantikan)
- **Surat keluar**: mirror lokal — nomor surat dan dokumen asli tetap dihasilkan di aplikasi pemerintah, aplikasi ini menyimpan PDF hasil unduh plus metadata untuk pencarian offline

## Daftar Fitur

### Manajemen Surat
- CRUD surat masuk & surat keluar dengan metadata (pengirim/tujuan, perihal, tanggal, sifat, klasifikasi)
- Multiple lampiran per surat (PDF utama + lampiran pendukung)
- PDF preview inline; download dengan watermark untuk surat sensitif
- Soft delete dengan retention policy (admin restore)
- Tembusan (cc list) untuk surat keluar
- Versioning PDF saat replace

### Traceability Korespondensi
- Link antar surat dengan tipe relasi: balasan, lanjutan, hasil disposisi, revisi, terkait
- Many-to-many — satu surat bisa punya banyak predecessor & successor
- External reference untuk korespondensi lama yang belum ter-input
- Thread korespondensi tervisualisasi (graph/tree)

### Direktori Master
- Direktori instansi dengan nama kanonik + alias (mengurangi inkonsistensi nama pengirim)
- Klasifikasi & sifat surat (configurable, bukan hardcoded)

### Workflow Kantor
- Disposisi: assign surat masuk ke staf, status & deadline opsional
- Komentar/catatan append-only di setiap surat
- Inbox view: surat baru / belum diproses
- Reminder & deadline tracking dengan threshold per sifat surat

### PWA Offline
- Read offline: list, search, detail metadata (PDF tetap online-only)
- Write offline: input, edit, komentar — sync otomatis saat online
- Pending-sync indicator persistent
- Login token cache untuk bertahan offline (TTL eksplisit)

### Sync & Rekonsiliasi
- UUIDv7 client-generated → no create-conflict antar-klien
- Operation log + last-write-wins per field untuk update
- Pre-save dedup check (online)
- Merge-on-sync untuk konflik dedup offline
- Antrian rekonsiliasi dengan merge UI side-by-side

### Multi-User & Permission
- Role: **staf**, **camat**, **student** (read-only demo), **admin**
- ACL per surat untuk kategori rahasia
- Audit log: setiap perubahan tercatat (siapa, kapan, apa) — dari hari pertama
- Read audit log untuk surat sensitif (siapa lihat/download)

### Reporting & Operasional
- Statistik surat per periode, klasifikasi, pengirim
- Beban kerja per staf
- Surat overdue (belum dibalas / belum didisposisi)
- Export CSV/Excel
- Bulk import historis (CSV + folder PDF)

### Student Mode (Edukasi Mahasiswa)
- Drawer panel menampilkan struktur data, algoritma, dan analisis Big-O untuk operasi yang sedang berjalan
- Concept catalog publik di GitHub Pages dengan link permalink ke source code
- Dataset demo di-tag per semester untuk reproducibility praktikum

## Arsitektur Ringkas

- **Deployment**: VPS tunggal (~1GB RAM / 1 vCPU), tanpa Docker
- **Backend**: Go (`net/http` + sqlc + pgx), PostgreSQL, JWT auth
- **Frontend**: Vue 3 + Vite, PWA (vite-plugin-pwa), Pinia, Dexie.js, TanStack Query
- **Concept catalog**: mdBook + GitHub Actions → GitHub Pages

Detail keputusan teknis: `CLAUDE.md`. Roadmap implementasi per fase: `ROADMAP.md`.

## Compile dan Run (Development Lokal)

### Prerequisites

| Tool | Versi | Wajib untuk |
|---|---|---|
| Docker | 20+ | Run app via compose (di bawah) + E2E test |
| Docker Compose | v2.x | Sudah include di Docker Desktop modern |
| Go | 1.22+ | Active coding backend (HMR), unit test |
| Node.js | 20+ | Active coding frontend (HMR), unit test |
| mdBook | 0.4.40+ | Preview concept catalog lokal (opsional, CI build otomatis) |

Untuk **just run aplikasi**, hanya butuh **Docker** + **Docker Compose**. Go dan Node.js diperlukan kalau mau active development dengan HMR.

### Run dengan Docker Compose

Quick start dari repo root:

```sh
git clone https://github.com/idtazkia/aplikasi-surat-kecamatan.git
cd aplikasi-surat-kecamatan

# Generate JWT secret (sekali, simpan di .env atau export per session)
export JWT_SECRET=$(openssl rand -base64 32)

# Build images + start semua service
docker compose up --build
```

Yang terjadi:

1. **postgres** — PostgreSQL 16 dengan persistent volume `surat-pgdata`
2. **backend** — Go binary, otomatis apply schema + demo seed migration saat startup, listen di port 8080
3. **frontend** — Vue + Naive UI ter-build, di-serve nginx di port 80, proxy `/api/*` ke backend

Setelah semua siap (~30 detik first time, ~5 detik subsequent):

- App: [http://localhost:5173](http://localhost:5173)
- Backend API direct: [http://localhost:8080](http://localhost:8080)
- Health check: `curl http://localhost:5173/healthz`

### Operasi Compose

```sh
docker compose up --build       # build + run, foreground (Ctrl+C untuk stop)
docker compose up -d --build    # detached (background)
docker compose logs -f backend  # tail log backend
docker compose ps               # list service status
docker compose stop             # stop semua, preserve container & volume
docker compose down             # stop + hapus container, preserve volume
docker compose down -v          # stop + hapus container + volume (FRESH RESET, hilangin DB data)
```

### Konfigurasi via Environment

| Env var | Default | Catatan |
|---|---|---|
| `JWT_SECRET` | (wajib) | `openssl rand -base64 32`. Compose error kalau tidak di-set |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `STUDENT_MODE_ENABLED` | `false` | `true` untuk aktifkan student-mode response (`_edu` block). **Jangan set true di production** |
| `APPLY_DEMO_SEED` | `true` | `false` untuk skip seed (simulate production fresh DB) |

Contoh dengan student mode aktif + demo seed off (untuk testing minimal startup):

```sh
JWT_SECRET=$(openssl rand -base64 32) \
STUDENT_MODE_ENABLED=true \
APPLY_DEMO_SEED=false \
docker compose up --build
```

### Login Demo

Setelah backend + frontend running, buka [http://localhost:5173](http://localhost:5173):

| Username | Password | Role |
|---|---|---|
| `staf1` | `demo123` | staf |
| `staf2` | `demo123` | staf |
| `camat` | `demo123` | camat |
| `admin` | `demo123` | admin |
| `student` | `demo123` | student (read-only demo) |

> **Catatan**: kredensial ini hanya untuk demo seed. Production deployment set `APPLY_DEMO_SEED=false` agar DB schema saja tanpa data dummy.

### Active Development dengan HMR

Compose cocok untuk run-and-test, tapi rebuild image setiap perubahan kode lambat. Untuk active coding, jalankan service berbeda di host (HMR) sambil PostgreSQL tetap di compose:

```sh
# Terminal 1: PostgreSQL only via compose
docker compose up postgres

# Terminal 2: backend Go dengan auto-reload
export DATABASE_URL="postgres://surat:surat@localhost:5432/surat?sslmode=disable"
export JWT_SECRET=$(openssl rand -base64 32)
export LISTEN_ADDR=":8080"
export LOG_LEVEL=debug
export STUDENT_MODE_ENABLED=false
make install-tools  # sekali, install goose
make migrate-up
make seed-demo
make dev            # go run ./cmd/server

# Terminal 3: Vite dev server dengan HMR
make web-install    # sekali
make web-dev        # http://localhost:5173, /api/* proxy ke localhost:8080
```

### Build Production Binary (Non-Docker)

Kalau deploy ke VPS systemd-based (lihat `deploy/README.md`), build binary langsung:

```sh
make build       # output: bin/server (Go binary, statically linked)
make web-build   # output: web/dist/ (static frontend assets)
```

### Concept Catalog (mdBook)

```sh
make concepts-inject  # scan marker di source, inject permalink ke markdown
make concepts-build   # build static site -> docs/concepts/book/
make concepts-serve   # preview lokal (butuh mdbook installed)

make concepts-lint    # gate orphan: marker tanpa page atau page tanpa marker
```

### Testing

#### Backend (Go)

```sh
make test         # unit + integration test, race detector aktif
make coverage     # buka coverage.html di browser
```

Filter pakai go test flag:

```sh
go test -run TestJWT_Issue ./internal/auth/   # satu test specific
go test -v ./internal/auth/                   # verbose
```

#### Frontend Unit Test (Vitest)

```sh
cd web
npx vitest run                  # one-shot
npx vitest                      # watch mode
npx vitest run --coverage       # dengan coverage report
```

#### End-to-End (Playwright + testcontainers)

E2E test pakai testcontainers untuk spin up PostgreSQL otomatis (image `postgres:16-alpine`). Tidak perlu setup DB manual — Docker harus running.

```sh
cd web
npx playwright install chromium       # sekali (download browser ~150MB)
npx playwright test                   # full E2E run
npx playwright test --ui              # UI mode (interactive debugging)
npx playwright test login.spec.ts     # filter by file
npx playwright show-report            # lihat report HTML setelah run
```

E2E test really submit form — pakai Playwright untuk fill input dan click button, bukan call API langsung.

### Reset Data

| Cara | Effect |
|---|---|
| `docker compose down -v` | Hapus container + volume `surat-pgdata` → DB benar-benar fresh next `up` |
| `make reset-demo` (host workflow) | Rollback semua seed migration + re-apply, schema tidak disentuh. Butuh DB connection langsung |
| Restart backend container | `docker compose restart backend` — migration idempotent, no data loss |

### Troubleshooting

| Problem | Solusi |
|---|---|
| `JWT_SECRET wajib di-set` saat compose up | Generate: `export JWT_SECRET=$(openssl rand -base64 32)` lalu compose up lagi |
| `port is already allocated` (5432, 8080, 5173) | Service host pakai port yang sama. Stop service lokal atau ubah `ports:` di docker-compose.yml |
| Backend exit dengan `db ping failed` | Postgres belum siap. Compose punya healthcheck, biasanya self-heal di restart kedua. Cek `docker compose logs postgres` |
| Frontend 502 saat akses /api/* | Backend belum siap saat nginx start. Tunggu beberapa detik atau `docker compose restart frontend` |
| `goose: command not found` (host workflow) | Jalankan `make install-tools`. Goose dipasang ke `$GOPATH/bin/goose` — pastikan ada di `PATH` |
| `npm install` gagal di node_modules existing | `rm -rf web/node_modules web/package-lock.json && make web-install` |
| Playwright timeout di global setup | Cek Docker running. Image PostgreSQL ~80MB akan di-download saat pertama run |
| Concept-links lint gagal `[orphan-page]` | Halaman markdown punya `id:` tapi tidak ada marker `// concept:<id>:start` di source. Tambah marker, atau set `pending: true` di frontmatter untuk page intro |
| Build slow (> 1 menit per layer) | `docker system prune` untuk hapus old layers. `npm ci` di-cache di image layer; perubahan `package.json` invalidate cache itu (expected) |

### Deployment Production

Setup di VPS Biznet Gio: lihat `deploy/README.md` untuk instruksi step-by-step.

## Status

**Fase 0 — Fondasi**: scaffold selesai. Schema, auth, login HTTP, frontend Vue + PWA + Naive UI, concept catalog seed, CI workflows, deployment templates.

**Fase 1 — MVP Online**: belum dimulai. Lihat `ROADMAP.md` untuk detail per fase.

## Lisensi

Apache License 2.0 — lihat `LICENSE`.
