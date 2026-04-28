# CLAUDE.md

Panduan untuk sesi Claude Code di repo ini. Baca sebelum melakukan perubahan.

## Konteks Proyek

Aplikasi manajemen surat masuk/keluar untuk kantor kecamatan, dibangun sebagai PkM STMIK Tazkia. Permintaan datang dari Bu Camat (nama kecamatan — TBD) karena aplikasi surat keluar pemerintah sering offline pada saat dibutuhkan.

Bu Camat bukan user data-entry utama. Data entry dilakukan oleh staf (beberapa orang). Bu Camat melakukan supervisi dan review, dengan frekuensi input sendiri sekitar 1–2x per minggu.

## Mandat Proyek

Dua mandat setara, harus dijaga keseimbangannya:

1. **Operasional**: Bu Camat dan staf pakai aplikasi sebagai pengganti filing fisik dan handle outage aplikasi pemerintah lewat PWA offline.
2. **Edukasi**: mahasiswa STMIK Tazkia belajar Struktur Data, Algoritma, dan Basis Data via aplikasi. Konsep matkul yang relevan dilink ke source code lewat student-mode dan concept catalog publik.

Implikasi edukasi terhadap pilihan teknis: hindari abstraksi yang menyembunyikan konsep dari mahasiswa (alasan tambahan kenapa stack pakai sqlc + pgx bukan ORM, dan net/http stdlib bukan framework). Pemilihan implementasi mempertimbangkan apakah representatif untuk konsep matkul.

## Keputusan Arsitektur

### Deployment
- Single VPS. Tidak ada server on-premise di kantor kecamatan.
- Alasan: menghindari kompleksitas VPN dan ketergantungan pada IT on-site yang minim. PWA menangani kebutuhan offline di PC kantor.
- Asumsi reliability: masalah aplikasi pemerintah umumnya disebabkan kapasitas planning yang buruk dan technical debt yang tidak tertangani (vendor hilang pasca-delivery), bukan infrastruktur cloud. Target coverage test otomatis ≥70% untuk menekan technical debt.

### PWA Offline
- Service worker + Cache API + IndexedDB.
- PDF **tidak** disimpan di IndexedDB — hanya metadata surat yang dicache. PDF tetap diakses dari VPS.
- Cache policy untuk menjaga ukuran storage.
- Strategi update service worker harus menangani skenario offline beberapa hari (staf bisa terjebak di kode/data stale) — eksplisit pilih semantic "update on next online visit".

### Hubungan dengan Aplikasi Pemerintah (Surat Keluar)
- Aplikasi pemerintah = source of truth. Mandatory digunakan untuk penomoran dan penerbitan.
- Workflow: buat surat keluar di aplikasi pemerintah → download PDF → input metadata + upload PDF ke aplikasi ini.
- Ekstraksi teks PDF untuk pencarian full-text bisa ditambahkan belakangan jika diperlukan.

### Identitas & Sinkronisasi
- Client-generated UUID (v7, roughly time-ordered) untuk primary key semua entitas. Menghilangkan create-conflict antar-klien.
- Operation log + last-write-wins per field untuk update. Tidak menggunakan CRDT — overkill untuk domain ini.
- Append-only untuk komentar/catatan — conflict-free by construction.
- Setiap perubahan tercatat di audit log (siapa, kapan, apa). Dibangun dari hari pertama, bukan retrofit.
- **Identifier surat hanya `nomor_surat` formal** (dari aplikasi pemerintah untuk surat keluar, dari pengirim untuk surat masuk). Tidak ada nomor agenda internal yang di-generate aplikasi. Konsekuensi: offline-first sederhana, tidak ada sequence yang harus dijaga server-side.

### Traceability Surat
- Setiap surat bisa merujuk ke surat lain via tabel `surat_references` (many-to-many bertipe).
- Skema: `from_surat_id` → `to_surat_id` dengan `relationship` enum (`balasan`, `lanjutan`, `disposisi_hasil`, `revisi`, `terkait`).
- External reference: `to_surat_id` boleh null, isi `external_ref` text untuk surat lama yang tidak ada di sistem. Saat surat tersebut akhirnya ter-input, tooling rekonsiliasi sarankan auto-link.
- Thread korespondensi traversal via recursive CTE dengan cycle detection wajib.
- Tidak menggunakan parent_id tunggal — relasi bisa multi-arah, multi-tipe, dan ke external.

### Deduplikasi Surat (dua staf input surat yang sama saat offline)
Kunci deduplikasi berbeda per jenis:
- **Surat masuk**: `(normalized_sender + sender_nomor + tanggal_terima)`. Normalisasi nama pengirim perlu perhatian ("Kemendagri" vs "Kementerian Dalam Negeri"). Strategi normalisasi: direktori instansi dengan nama kanonik + alias (Fase 2), bukan heuristik string-only.
- **Surat keluar**: `nomor_surat` (dari aplikasi pemerintah, globally unique).

Strategi:
1. **Online pre-save check**: lookup kunci deduplikasi sebelum simpan — warn "surat ini sudah ada, edit saja?". Menangkap kasus umum (kantor dengan internet normal).
2. **Offline merge-on-sync**: jika server mendeteksi duplikat saat sync, jangan tolak. Simpan kedua record terhubung ke kunci yang sama, munculkan di antrian rekonsiliasi.
3. **Merge UI**: tampilkan side-by-side, pilih salah satu atau edit jadi versi kanonik. Audit log menyimpan kedua original.

Antrian rekonsiliasi saat ini dipegang role **staf**. Bisa dipindah ke supervisor/camat nanti lewat permission matrix tanpa migrasi schema.

### Role & Permission
- Role-permission scaffolding ada dari hari pertama, walaupun semua role awalnya resolve ke permission yang sama.
- Alasan: menghindari migrasi schema saat role matrix perlu berubah. Show/hide fitur per role jadi config change, bukan DB change.
- Role awal: **staf** (data entry + rekonsiliasi), **camat** (supervisi + override), **student** (read-only ke instance demo, untuk edukasi mahasiswa), **admin** (konfigurasi sistem). Read-only untuk pihak ketiga (auditor) ditambah belakangan kalau diminta.
- ACL per surat untuk kategori `rahasia` di samping role-based permission.

### Autentikasi Offline
- Login online; token cached untuk bertahan offline.
- TTL token: balance — terlalu pendek = staf terkunci saat offline, terlalu panjang = laptop hilang jadi masalah. Default 7 hari, configurable. Refresh saat kembali online.

### UI Obligations
- **Pending-sync indicator**: setiap staf harus melihat "N perubahan belum terupload" agar tidak menutup browser dengan data yang masih di IndexedDB.
- **Antrian rekonsiliasi duplikat**: view khusus untuk role yang memegang rekonsiliasi.
- **Indikator offline**: banner persistent "Anda offline — data terakhir disinkron pada [timestamp]".

## Mandat Edukasi

Student-mode dan concept catalog adalah first-class artifact, bukan dokumentasi sambil lalu.

### Matkul yang Dialign
Struktur Data, Algoritma, Basis Data. Tidak diperluas ke matkul lain.

### Concept Catalog
- Konten markdown di `docs/concepts/`, struktur folder per matkul (`struktur-data/`, `algoritma/`, `basis-data/`).
- Frontmatter wajib: `id`, `courses`, `prereq`, `related`, `fase`.
- Render ke static site via mdBook, build di GitHub Actions, deploy ke GitHub Pages dengan custom domain `concepts.<domain>`.
- Repo publik (untuk free-tier GitHub Pages dan keterbacaan permalink ke source code oleh mahasiswa).

### Code Anchoring
- Marker comment di source code: `// concept:<id>:start` dan `// concept:<id>:end` (atau ekuivalen `# concept:...:start|end` untuk SQL/Python).
- Tooling `tools/concept-links/` (Go) scan marker → resolve ke (file, line range, commit SHA) → generate GitHub permalink → inject ke markdown concept page + emit `concept-links.json` untuk Vue student drawer.
- CI gate: `@anchor:foo` orphan (tanpa marker pasangan di kode) atau marker tanpa concept page → build fail.

### Student Mode di App
- Role `student` (read-only, hanya akses instance demo).
- Toggle bisa diaktifkan untuk role admin/dev untuk debugging.
- Backend middleware: kalau student-mode aktif, append blok `_edu` di response (data structure, complexity O(...), SQL query, EXPLAIN ANALYZE output, concept IDs).
- Frontend: drawer student-mode render `_edu` payload + dereference concept ID ke konten dari concept catalog.
- **Production binary: student mode hard-disabled lewat compile-time flag atau env check yang gagal-aman.** Tidak ada kemungkinan kebocoran ke data kecamatan.

### Dataset Demo
- Goose migration di folder terpisah: `db/migrations/demo-seed/` (vs `db/migrations/schema/`).
- Production deployment hanya apply schema. Demo deployment apply schema + seed.
- Reproducibility praktikum: pakai app version tag (CalVer `YYYY.MM.NN`). Checkout tag = state schema + seed deterministik. Tidak ada tag dataset terpisah — versi app sudah cover state dataset.
- Reset script: `make reset-demo` — rollback semua seed migration, re-apply, schema tidak disentuh.
- Seed harus idempotent (`INSERT ... ON CONFLICT DO NOTHING` atau gate dengan check) supaya aman re-run.

## Tech Stack

### Backend
- Go (latest stable), plain `net/http` dengan ServeMux (Go 1.22+ routing). **Tidak menggunakan web framework** (Gin/Echo/Fiber/chi).
- Database driver: `jackc/pgx`
- Query generation: `sqlc` — typed Go dari SQL
- Migrations: `goose`
- Validation: `go-playground/validator`
- Auth: JWT + refresh token, roll manual di atas stdlib

### Frontend
- Vue 3 (Composition API) + Vite
- State: Pinia
- Router: Vue Router
- PWA: `vite-plugin-pwa`
- IndexedDB: Dexie.js
- Query cache / offline sync: TanStack Query (Vue adapter)
- UI/table library: **Naive UI**

### Database
- PostgreSQL

### Concept Catalog
- mdBook (build di GitHub Actions, tidak wajib install lokal)
- Hosting: GitHub Pages dengan custom domain `concepts.<domain>`

### Deployment
- VPS existing di **Biznet Gio**
- systemd service untuk Go binary, nginx reverse proxy, Let's Encrypt TLS
- Tidak menggunakan Docker/Kubernetes
- Concept catalog hosted di GitHub Pages (terpisah dari VPS app, mengurangi beban)

### Versioning
- App version: **CalVer `YYYY.MM.NN`** (mis. `2026.05.01` = release pertama Mei 2026, `.02` = release kedua di bulan yang sama).
- Tidak ada tag terpisah untuk dataset atau roadmap — git history + app version tag sudah cukup.
- E2E test framework: **Playwright**.

### Alasan Pilihan Stack

- **Plain Go, bukan Gin/Echo/Fiber**: Go 1.22 ServeMux menutup gap routing utama (method + path params native). Menghindari framework churn, dependency tree lebih kecil, mahasiswa kontributor belajar HTTP asli tanpa abstraksi framework.
- **sqlc, bukan ORM**: SQL tetap visible (sesuai mandat edukasi Basis Data), generated Go code type-safe.
- **Vue, bukan React/Svelte**: komunitas Indonesia lebih besar, UI/table lib matang.
- **Naive UI, bukan PrimeVue/Element Plus**: didesain ground-up untuk Vue 3 (bukan port dari Vue 2 atau React). Mahasiswa kontributor belum ada pengalaman Vue sama sekali — pattern idiomatic Vue 3 yang bersih membantu proses belajar dari awal yang benar. Bonus: bundle paling kecil (penting untuk PWA), TypeScript first-class, theming via JS object (no SCSS preprocessor di stack).
- **Tanpa Docker**: single binary Go + systemd sudah deployable secara langsung. Menambahkan Docker tidak menyelesaikan masalah apapun di scope ini.
- **mdBook, bukan Hugo/Docusaurus**: paling minimalis untuk content-heavy markdown dengan navigation tree + search bawaan. Single binary, fast build di CI.

## Open Items Sebelum Mulai Koding

- [ ] Nama kecamatan dan kontak resmi (Bu Camat, PIC staf)
- [ ] MoU/kesepakatan kerja: kepemilikan data, SLA, cakupan maintenance, batasan liability
- [ ] Jumlah staf yang akan jadi user (menentukan seeding role & ekspektasi beban)
- [ ] Spek VPS Biznet Gio existing (RAM, CPU, storage, region) untuk capacity planning & backup target
- [ ] Retention policy untuk PDF (berapa lama disimpan, arsip offline, dsb.)
- [ ] Data sensitivity review: surat pemerintah bisa mengandung data pribadi — tentukan enkripsi at-rest dan in-transit, akses log
- [ ] Domain utama + subdomain `concepts.<domain>` + DNS
- [ ] Konfirmasi praktik penomoran disposisi di kantor — kalau tidak ada, drop field `nomor_disposisi`

## Roadmap Implementasi

Detail per fase (scope, tasks, exit criteria) ada di `ROADMAP.md`. Ringkasan:

- **Fase 0** — Fondasi (schema, tooling, CI/CD, mdBook setup)
- **Fase 1** — MVP Online (CRUD surat, traceability, audit log)
- **Fase 2** — Supervisi & Kolaborasi (disposisi, komentar, ACL rahasia, watermark)
- **Fase 3** — PWA Read-Only Offline
- **Fase 4** — PWA Offline Write & Sync
- **Fase 5** — Dedup & Rekonsiliasi
- **Fase 6** — Reporting & Operasional (statistik, bulk import, handout export)
- **Fase 7** — Future (FTS, OCR, push notification, multi-tenancy)

## Referensi

- Catatan proyek di life repo: `~/workspace/life/tazkia/projects/aplikasi-surat-kecamatan.md`
