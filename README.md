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

| Tool | Versi | Cara install |
|---|---|---|
| Go | 1.22+ | `brew install go` (macOS) / [go.dev/dl](https://go.dev/dl) |
| Node.js | 20+ | `brew install node` / [nodejs.org](https://nodejs.org) |
| PostgreSQL | 14+ (lokal) | `brew install postgresql` — opsional, lihat catatan testcontainers di bawah |
| Docker | 20+ | [docs.docker.com](https://docs.docker.com/get-docker/) — wajib untuk testcontainers (E2E test) |
| mdBook | 0.4.40+ | `cargo install mdbook` — opsional, hanya untuk preview concept catalog lokal (CI build otomatis) |

Tooling Go dipasang via `make install-tools`:

```sh
make install-tools  # install goose, sqlc (perlu network)
```

### Setup Awal (Sekali)

1. Clone repo:
   ```sh
   git clone https://github.com/idtazkia/aplikasi-surat-kecamatan.git
   cd aplikasi-surat-kecamatan
   ```

2. Setup PostgreSQL lokal (untuk dev backend):
   ```sh
   createdb surat_dev
   createuser surat
   psql -d surat_dev -c "ALTER USER surat WITH PASSWORD 'surat';"
   ```

3. Buat file `.env`:
   ```sh
   cp .env.example .env
   # edit .env, set JWT_SECRET ke random base64 min 32 byte
   # generate dengan: openssl rand -base64 32
   ```

4. Apply schema migration + demo seed:
   ```sh
   export DATABASE_URL="postgres://surat:surat@localhost:5432/surat_dev?sslmode=disable"
   make migrate-up
   make seed-demo
   ```

5. Install frontend dependencies:
   ```sh
   make web-install
   ```

### Run Backend (Development)

```sh
# Terminal 1
export $(grep -v '^#' .env | xargs)
make dev
# Server start di http://localhost:8080
# Health check: curl http://localhost:8080/healthz
```

### Run Frontend (Development)

```sh
# Terminal 2
make web-dev
# Vite dev server di http://localhost:5173
# /api/* otomatis di-proxy ke localhost:8080 (lihat web/vite.config.ts)
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

> **Catatan**: kredensial ini hanya untuk demo seed. Production deployment apply schema saja, tidak demo-seed.

### Build Production Binary

```sh
make build       # output: bin/server (Go binary)
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

### Reset Demo Data

```sh
make reset-demo   # rollback semua seed + re-apply (schema tidak disentuh)
```

### Troubleshooting

| Problem | Solusi |
|---|---|
| `goose: command not found` | Jalankan `make install-tools`. Goose dipasang ke `$GOPATH/bin/goose` — pastikan ada di `PATH` |
| `pgx: connect: connection refused` | PostgreSQL tidak running. `pg_isready` untuk check, `brew services start postgresql` untuk start |
| `STUDENT_MODE_ENABLED must be 'true' or 'false'` | Set di `.env`. Default ke `false` aman untuk dev |
| `npm install` gagal di node_modules existing | `rm -rf web/node_modules web/package-lock.json && make web-install` |
| Playwright timeout di global setup | Cek Docker running. Image PostgreSQL ~80MB akan di-download saat pertama run |
| Concept-links lint gagal `[orphan-page]` | Halaman markdown punya `id:` tapi tidak ada marker `// concept:<id>:start` di source. Tambah marker, atau set `pending: true` di frontmatter untuk page intro |

### Deployment Production

Setup di VPS Biznet Gio: lihat `deploy/README.md` untuk instruksi step-by-step.

## Status

**Fase 0 — Fondasi**: scaffold selesai. Schema, auth, login HTTP, frontend Vue + PWA + Naive UI, concept catalog seed, CI workflows, deployment templates.

**Fase 1 — MVP Online**: belum dimulai. Lihat `ROADMAP.md` untuk detail per fase.

## Lisensi

Apache License 2.0 — lihat `LICENSE`.
