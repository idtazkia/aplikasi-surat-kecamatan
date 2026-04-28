# ROADMAP

Rencana implementasi per fase. Tiap fase deployable mandiri — kalau budget habis di fase tertentu, user tetap dapat aplikasi yang dipakai.

## Prinsip

- Tiap fase punya **exit criteria** eksplisit (acceptance), bukan hanya task list.
- Tiap fitur baru ship dengan minimum 1 marker `// concept:<id>:start|end` di source code dan concept page yang merefer marker tersebut. Concept catalog tumbuh paralel dengan fitur, bukan retrofit.
- Coverage test ≥70% per fase sebelum dianggap selesai.
- Cross-cutting concerns (concept catalog, student-mode `_edu` block) dibangun incremental per fitur, bukan fase terpisah.

## Fase 0 — Fondasi

**Tidak dirilis ke user**. Infrastruktur untuk semua fase berikutnya berjalan tanpa retrofit.

### Backend
- Repo Go module init, struktur folder (`internal/`, `cmd/`, `db/`, `tools/`, `docs/`)
- Schema migration di `db/migrations/schema/`:
  - `users`, `roles`, `permissions`, `role_permissions`, `user_roles`
  - `surat` (id uuid, jenis enum [masuk|keluar], nomor_surat, perihal, tanggal_surat, tanggal_terima, sifat, klasifikasi_id, instansi_id, access_level, is_deleted, deleted_at, deleted_by, created_at, updated_at)
  - `surat_attachments` (id, surat_id, file_path, file_size, mime_type, version, uploaded_by, uploaded_at)
  - `surat_pdf_versions` (untuk replace history)
  - `surat_tembusan` (surat_id, instansi_id atau external_text)
  - `surat_references` (id, from_surat_id, to_surat_id nullable, relationship enum, external_ref text, note)
  - `surat_acl` (surat_id, user_id) untuk ACL surat rahasia
  - `instansi` (id, nama_kanonik, aliases array, alamat, kontak)
  - `klasifikasi`, `sifat` (configurable)
  - `disposisi` (id, surat_id, assigned_to, deadline, status, nomor_disposisi nullable, created_by, created_at)
  - `komentar` (id, surat_id, user_id, body, created_at) — append-only
  - `audit_log` (entity_type, entity_id, action, actor_user_id, before_jsonb, after_jsonb, created_at)
  - `read_audit_log` (entity_type, entity_id, actor_user_id, action enum [view|download], created_at) — terpisah untuk volume management
  - `operation_log` (client_op_id, entity_type, entity_id, field_changes_jsonb, created_at, applied_at) — untuk sync
  - `reconciliation_queue` (placeholder, populated di Fase 5)
- Auth: login endpoint, JWT issuance, refresh token, middleware permission check
- UUIDv7 generator (server) — wrapper dari library, atau implementasi sendiri (~50 baris)
- Audit log middleware (otomatis catat write transaction)
- Read audit log helper (manual call dari handler yang butuh)
- Health check endpoint

### Tooling
- `tools/concept-links/` (Go binary):
  - Sub-command `inject`: scan marker `// concept:<id>:start|end` di repo → resolve (file, lines, commit SHA) → replace `@anchor:<id>` di markdown concept page dengan GitHub permalink absolut
  - Sub-command `emit-json`: generate `concept-links.json` (machine-readable mapping concept_id → permalink) untuk Vue app
  - Sub-command `lint`: orphan detection (`@anchor:foo` tanpa marker, atau marker tanpa concept page) — exit non-zero
- `Makefile` targets: `migrate-up`, `migrate-down`, `seed-demo`, `reset-demo`, `concepts-build`, `test`, `coverage`, `dev`

### CI/CD
- `.github/workflows/test.yml`: lint (golangci-lint, eslint), test, coverage gate ≥70%
- `.github/workflows/concepts.yml`: jalan saat push ke main yang menyentuh `docs/concepts/**` atau source dengan marker → `concept-links inject` → `mdbook build` → deploy ke `gh-pages` branch
- Concept-link orphan check sebagai gate di kedua workflow

### Frontend skeleton
- Vue 3 + Vite project init
- Naive UI setup: theme provider, dark mode toggle, locale Indonesia
- Routing struktur (login, dashboard placeholder, list placeholder)
- Pinia store skeleton: `auth`, `syncQueue` (placeholder, dipakai mulai Fase 4)
- PWA setup awal (`vite-plugin-pwa`) — service worker registration ada, strategy detail di Fase 3
- Student drawer komponen skeleton (Pinia store `eduPanel` untuk `_edu` payload terakhir)

### Demo seed
- Migration di `db/migrations/demo-seed/`:
  - Users dummy (1 staf, 1 camat, 1 student, 1 admin)
  - 20 instansi dummy dengan alias variation
  - Klasifikasi & sifat default (~10 entri masing-masing)
  - 50 surat baseline (variasi jenis, sifat, klasifikasi)
  - 5 thread referensi multi-level
  - 3 kasus dedup conflict
  - 5 disposisi chain
- Idempotent (`INSERT ... ON CONFLICT DO NOTHING`)

### Deployment infra
- systemd unit template untuk Go binary
- nginx config template (reverse proxy + TLS)
- Let's Encrypt automation (acme.sh atau certbot)
- pg_dump backup script + retention (harian 7 hari, mingguan 4 minggu, bulanan 12 bulan)
- Verification job mingguan: restore ke staging, smoke test, alert kalau gagal
- Health check endpoint registered ke uptime monitor eksternal (UptimeRobot atau ekuivalen)

### Concept catalog seed
- mdBook setup (`book.toml`, `src/SUMMARY.md`)
- 5–8 concept page sebagai validasi pipeline:
  - `struktur-data/btree-index.md`
  - `struktur-data/uuid-v7.md`
  - `algoritma/jwt-hmac.md`
  - `algoritma/big-o-pengantar.md`
  - `basis-data/migration-versioning.md`
  - `basis-data/audit-log-pattern.md`
  - `basis-data/explain-analyze.md`

### Exit Criteria
- `make migrate-up && make seed-demo` works
- Login endpoint functional, JWT verifiable
- CI green: test ≥70% coverage, no orphan concept-link
- mdBook build + deploy ke `gh-pages` sukses; situs publik tampil
- systemd service deploy ke VPS staging, health check OK
- Backup + restore verification job sukses minimal sekali

---

## Fase 1 — MVP Online: Arsip Digital Surat

**Versi pertama yang dipakai user**. Online-only. Menggantikan filing fisik.

### Backend
- `POST /api/surat` — create (masuk/keluar)
- `GET /api/surat` — list dengan filter (tanggal range, instansi, perihal, klasifikasi, sifat) + pagination keyset
- `GET /api/surat/{id}` — detail dengan lampiran, references, audit log expandable
- `PATCH /api/surat/{id}` — update metadata
- `DELETE /api/surat/{id}` — soft delete
- `POST /api/surat/{id}/restore` — admin only
- `POST /api/surat/{id}/attachments` — multipart upload, multiple files
- `GET /api/surat/{id}/attachments/{att_id}` — download streaming
- `GET /api/surat/{id}/attachments/{att_id}/preview` — preview inline
- `POST /api/surat/{id}/references` — add reference (internal atau external)
- `DELETE /api/surat/{id}/references/{ref_id}`
- `GET /api/instansi?q=` — autocomplete
- `POST /api/instansi` — create on-the-fly saat input surat (auto-collect strategy)
- Admin endpoints: CRUD users, klasifikasi, sifat
- sqlc query untuk semua di atas
- Index: B-Tree pada `surat(tanggal_terima)`, `surat(nomor_surat)`, partial index `WHERE NOT is_deleted`

### Frontend
- Halaman login + auth flow (token storage, redirect)
- Halaman daftar surat: filter sidebar + tabel + pagination
- Halaman input surat (form berbeda untuk masuk vs keluar):
  - Field metadata
  - Multi-file upload dengan drag-drop
  - Picker referensi: search surat existing atau input external_ref bebas
  - Tipe relasi dropdown
- Halaman detail surat:
  - Header: nomor, perihal, tanggal, sifat (dengan badge), klasifikasi
  - Section lampiran: list dengan preview inline (PDF viewer)
  - Section "Riwayat korespondensi": dua list (predecessors + successors) dengan tipe relasi
  - Section "Catatan": placeholder Fase 2
  - Footer: audit log expandable
- Halaman admin: user management, konfigurasi klasifikasi/sifat
- Direktori instansi: input field surat punya autocomplete dari `instansi` existing; kalau ketik nama baru, prompt "buat baru?"

### Concept anchoring (sample, bukan exhaustive)
- Marker di schema migration → concept `btree-index`, `partial-index-soft-delete`
- Marker di sqlc query (`list_surat_filtered`) → concept `query-plan-explain`, `keyset-pagination`
- Marker di handler attachment upload → concept `multipart-streaming`, `memory-bounded-io`
- Marker di JWT middleware → concept `jwt-hmac`, `auth-token-flow`
- Marker di audit log middleware → concept `audit-log-pattern`, `transaction-isolation`

### Exit Criteria
- Staf dapat input, edit, search, view, soft-delete surat masuk & keluar lewat UI
- Multi-file upload + preview inline berfungsi
- Referensi surat tercatat & ditampilkan dengan tipe relasi
- Audit log lengkap untuk setiap mutation
- Coverage ≥70%
- Deploy ke VPS production
- User acceptance: trial 1–2 minggu dengan PIC kantor, tidak ada bug major

---

## Fase 2 — Supervisi & Kolaborasi

**Setelah Phase 1 stabil**. Tambahkan kolaborasi staf dan supervisi camat.

### Backend
- Disposisi:
  - `POST /api/disposisi` — assign
  - `PATCH /api/disposisi/{id}` — update status
  - `GET /api/disposisi?assigned_to=&status=` — list
- Komentar:
  - `POST /api/surat/{id}/komentar` — append
  - `GET /api/surat/{id}/komentar` — list (read-only)
- Permission matrix camat: full access, override status disposisi
- ACL surat rahasia:
  - Schema: `surat.access_level` enum, `surat_acl` tabel
  - Middleware: cek access_level + ACL membership
- Direktori instansi full:
  - `POST/PATCH/DELETE /api/instansi`
  - `POST /api/instansi/{id}/merge` — merge alias instansi yang inkonsisten
- Watermark PDF saat download (library: `pdfcpu` atau `unipdf`):
  - Overlay nama user + timestamp di setiap halaman
  - Hanya untuk `access_level >= restricted`
- Read audit log aktif untuk akses surat dengan `access_level != public`
- PDF versioning: replace lampiran simpan versi lama di `surat_pdf_versions`
- Notifikasi in-app:
  - Schema: `notifications` (user_id, type, payload_jsonb, read_at, created_at)
  - Trigger: disposisi baru, komentar baru, mention di komentar
  - Endpoint poll: `GET /api/notifications?since=` (Fase 7 ganti push)

### Frontend
- Dashboard camat:
  - Surat masuk hari ini
  - Disposisi belum diassign
  - Disposisi overdue
  - Surat butuh review
- Halaman disposisi:
  - List dengan filter
  - Form create + edit
- Komentar inline di detail surat (form append + list)
- Inbox view (route terpisah): surat masuk hari ini, belum diopen, butuh disposisi
- Notifikasi bell di topbar + dropdown
- Direktori instansi UI admin: CRUD + merge alias dengan konfirmasi
- ACL UI di detail surat (admin): set access_level, manage user list

### Concept anchoring
- Recursive CTE untuk traversal `surat_references` → `recursive-cte`, `cycle-detection`
- RBAC + ACL resolution → `rbac-pattern`, `acl-row-level`
- Append-only komentar → `immutable-data-pattern`, `event-sourcing-intro`
- PDF watermark layer → `pdf-content-stream`

### Exit Criteria
- Bu Camat punya dashboard dengan informasi yang useful tanpa hand-holding
- Disposisi mengalir camat → staf → done
- Komentar tercatat append-only, tidak bisa diedit (UI + backend enforce)
- Surat sensitif terbatas akses sesuai ACL (verified dengan test scenario)
- Watermark download PDF terbaca di hasil
- Notifikasi muncul saat ada disposisi baru
- Coverage ≥70%

---

## Fase 3 — PWA Read-Only Offline

**Investasi offline pertama**. Read-only karena lebih sederhana dan sudah memberi value besar.

### Backend
- `GET /api/sync/snapshot?since=` — incremental sync metadata (delta sejak timestamp)
- ETag / If-None-Match support untuk efisiensi
- Endpoint return: list surat + instansi + klasifikasi + sifat dalam payload terstruktur
- Pagination snapshot kalau dataset > N entries (chunking)

### Frontend
- Service worker via `vite-plugin-pwa`:
  - **stale-while-revalidate** untuk static assets (HTML, CSS, JS bundles)
  - **network-first dengan cache fallback** untuk metadata API (`/api/surat`, `/api/instansi`)
  - **never-cache** untuk PDF endpoints (online-only, sesuai keputusan arsitektur)
- IndexedDB schema (Dexie):
  - `surat` object store dengan index pada `tanggal_terima`, `instansi_id`, `nomor_surat`
  - `instansi`, `klasifikasi`, `sifat` lookup stores
  - `sync_meta` store: `last_sync_at`, `schema_version`
- Sync logic: pada online, fetch incremental, write ke Dexie (transactional)
- Offline detection: `navigator.onLine` + heartbeat ping endpoint
- Login token cache di IndexedDB dengan TTL eksplisit (default 7 hari, configurable)
- Update strategy "on next online visit":
  - SW check update on `visibilitychange` + `online` event
  - Kalau ada update, prompt "Versi baru tersedia, refresh sekarang?"
  - Kalau user offline saat ada update server, tunggu sampai online lagi
- Banner persistent "Anda offline — data terakhir disinkron pada [timestamp]"

### Concept anchoring
- Service worker lifecycle (install/activate/fetch) → `sw-state-machine`
- IndexedDB internal (browser pakai LevelDB-like / B-Tree variant) → `indexeddb-storage`
- Cache strategy taxonomy → `cache-strategies`
- Stale-while-revalidate semantic → `cache-invalidation-tradeoff`

### Exit Criteria
- Offline: list, search, detail metadata berfungsi (verified dengan DevTools network throttling)
- PDF tetap online-only — kalau offline, jelas messaging "PDF tidak tersedia offline"
- Login bertahan offline sampai TTL habis
- SW update flow handle benar (verified dengan deploy versi baru saat client offline)
- Lighthouse PWA score ≥90
- Coverage ≥70%

---

## Fase 4 — PWA Offline Write & Sync

**Inti selling point**. Paling kompleks; ditaruh setelah read-only matang.

### Backend
- `POST /api/sync/operations` — terima batch operation log dari client
  - Body: array of operations dengan `client_op_id`, `entity_type`, `entity_id`, `action`, `field_changes`, `client_timestamp`
  - Idempotency check: kalau `client_op_id` sudah pernah applied, return existing result
  - Apply per-field LWW: bandingkan `client_timestamp` per field dengan `updated_at` per field di server
  - Return: per-operation status (applied / conflict-resolved / rejected dengan reason)
- Schema: tambahkan `surat_field_timestamps` table atau JSONB column `field_updated_at` untuk LWW per-field tracking
- Conflict log (informational): kalau LWW terapply, log siapa yang lost
- `GET /api/sync/status` — return server time + last applied op per client

### Frontend
- UUIDv7 client-generated:
  - Library: `uuidv7` package atau implementasi sendiri (~50 baris, jadi marker concept)
- Operation log di Dexie: setiap mutation (create/update/delete/append komentar) emit operation
  - Schema: `operation_log` store dengan index pada `created_at`, `synced_at`
- Sync queue:
  - Process queue saat online dengan batch (mis. 50 ops per request)
  - FIFO order
  - Retry dengan exponential backoff (1s → 2s → 4s → … cap 60s)
  - Idempotency: setiap op punya `client_op_id` (UUIDv7)
- Pending-sync indicator (mandatory per UI obligation):
  - Hitung pending operations dari Dexie
  - Visualisasi di topbar: "N perubahan menunggu sync" + spinner saat sync running
- Pre-save dedup check: kalau online, hit `GET /api/surat/check-duplicate` sebelum simpan
  - Untuk surat masuk: kunci `(normalized_sender, sender_nomor, tanggal_terima)`
  - Untuk surat keluar: kunci `nomor_surat`
- Conflict resolution UI: untuk konflik LWW yang ambigu (misal: 2 staf edit field sama dengan timestamp dekat) — show toast "Perubahan kamu di-overwrite oleh [user]; lihat detail"

### Concept anchoring
- Event sourcing pattern → `event-sourcing-operation-log`
- LWW merge per field → `lww-merge-semantic`
- UUIDv7 ordering kenapa solve create-conflict → `uuid-v7-create-conflict-free`
- Vector clock vs Lamport vs LWW → `clock-comparison-tradeoff`
- Idempotency via key → `idempotency-pattern`
- At-least-once delivery + exponential backoff → `retry-strategy`

### Exit Criteria
- Input surat offline → sync saat online berhasil
- 2 client offline edit field berbeda → keduanya merged correctly
- 2 client offline edit field sama → LWW (pemenang sesuai timestamp), dengan log konflik
- Komentar append-only — 2 client append → kedua komentar masuk, tidak ada konflik
- Pending-sync indicator selalu akurat (verified dengan stress test)
- Sync queue retry handle network error tanpa data loss (verified dengan kill network mid-sync)
- Coverage ≥70%

---

## Fase 5 — Dedup & Rekonsiliasi

**Edge case offline kolaborasi**. Saat 2 staf input surat sama saat sama-sama offline.

### Backend
- Server-side dedup detection saat sync apply:
  - Surat masuk: query `(normalized_sender, sender_nomor, tanggal_terima)` — `normalized_sender` di-resolve via `instansi.nama_kanonik` atau alias match
  - Surat keluar: query `nomor_surat`
- Kalau duplikat: tidak reject; mark kedua record `pending_reconciliation = true`, link via `reconciliation_group_id`, buat entry di `reconciliation_queue`
- Endpoint reconciliation:
  - `GET /api/reconciliation` — list pending (filter by group)
  - `GET /api/reconciliation/{group_id}` — detail kedua versi side-by-side
  - `POST /api/reconciliation/{group_id}/merge` — pilih kanonik atau merge custom
- Audit log: simpan kedua original + keputusan merge
- Normalisasi nama pengirim:
  - Helper function: lowercase + hapus punctuation + trim + map alias dictionary dari `instansi.aliases`
  - Untuk fuzzy match (saat tidak exact): Levenshtein distance dengan threshold (configurable, default ≤ 3 untuk string ≥ 8 karakter)
  - Trie struktur untuk prefix lookup di autocomplete
- External-reference resolver:
  - Background job (cron 1x sehari atau pasca-sync) atau event-driven hook
  - Scan `surat_references.external_ref` (text) yang berpotensi match dengan surat baru ter-input
  - Match heuristic: nomor surat di teks ekstrak vs `nomor_surat` baru
  - Kalau match: notifikasi ke user yang membuat reference asli — "Surat baru ditambahkan, mungkin terkait?"

### Frontend
- Halaman antrian rekonsiliasi:
  - List grup pending dengan summary (key dedup, jumlah versi, instansi)
  - Default visible untuk role staf
- Merge UI side-by-side:
  - Field per field comparison dengan highlight diff
  - Pilih kanonik (radio button per field) atau edit gabungan
  - Confirm dengan preview hasil merge
- External-reference suggestion:
  - Notification toast "Mungkin terkait dengan referensi di surat X"
  - Tombol confirm/reject auto-link

### Concept anchoring
- Levenshtein / Jaro-Winkler distance → `string-distance-algorithms`
- Trie untuk prefix lookup → `trie-data-structure`
- Why we don't use CRDT (kontras vs LWW + manual reconcile) → `crdt-vs-lww-tradeoff`
- Manual reconciliation pattern → `human-as-conflict-resolver`
- Composite unique key untuk dedup → `composite-key-dedup`

### Exit Criteria
- Skenario test: 2 staf input surat masuk identik offline → setelah sync, muncul di antrian rekonsiliasi (bukan di-reject)
- Merge UI bisa pilih kanonik atau edit gabungan
- Audit log simpan kedua original
- External-reference auto-suggest works untuk match nomor_surat di teks
- Normalisasi nama pengirim correct untuk kasus alias dictionary + Levenshtein fallback
- Coverage ≥70%

---

## Fase 6 — Reporting & Operasional

**Polish & utility**. Setelah core lengkap.

### Backend
- Aggregation endpoints:
  - `GET /api/stats/by-period?bucket=month` — surat per bulan, dengan BRIN index pada `tanggal_terima`
  - `GET /api/stats/by-classification`
  - `GET /api/stats/by-sender?top=10`
  - `GET /api/stats/staff-load` — disposisi handled per staf
  - `GET /api/stats/overdue` — surat yang belum dibalas / belum didisposisi (configurable threshold per sifat)
- Bulk import:
  - `POST /api/import/dry-run` — validate CSV structure, return per-row errors tanpa write
  - `POST /api/import/execute` — actual import dengan link PDF dari upload folder
  - Progress endpoint untuk monitor (long-running)
- Reminder & deadline:
  - Background job: scan surat overdue setiap N jam, emit notifikasi
  - Configurable threshold per sifat: misal "balas dalam 7 hari untuk biasa, 1 hari untuk segera"
- Export:
  - `GET /api/export/surat?format=csv|xlsx&filter=...` — stream hasil filter
- Handout export tooling: `tools/handout-export/`:
  - Collect concept catalog → PDF per matkul (Pandoc → LaTeX → PDF)
  - QR code di setiap concept page yang berisi GitHub permalink

### Frontend
- Halaman statistik dengan chart (library: Chart.js — pilih saat scaffold; alternatif echarts kalau butuh visualisasi lebih kompleks):
  - Time series: surat masuk/keluar per bulan
  - Bar: per klasifikasi
  - Top-N: pengirim teratas
- Halaman beban kerja per staf
- Halaman bulk import:
  - Upload CSV + folder PDF
  - Preview dry-run dengan error per row
  - Confirm + progress bar saat actual run
- Halaman pengaturan reminder threshold (admin)
- Print-friendly view per surat & list (dedicated print stylesheet)

### Concept anchoring
- BRIN index untuk timeseries → `brin-index-timeseries`
- Aggregation query GROUP BY + indexed expression → `aggregation-optimization`
- Keyset pagination kontras vs OFFSET → `pagination-keyset-vs-offset`
- Background job pattern → `cron-vs-event-driven`

### Exit Criteria
- Statistik akurat untuk dataset sample (validasi manual lewat hand-count)
- Bulk import 100 row sukses dengan dry-run + actual run, error report jelas
- Reminder dikirim sesuai threshold (verified dengan test scenario)
- Export hasil filter sesuai dengan UI list
- Handout PDF generated dengan QR code permalink berfungsi
- Coverage ≥70%

---

## Fase 7 — Future Candidates

**Tidak dijanjikan**. Diputuskan per item kalau ada permintaan konkret dari user.

- Full-text search PDF (ekstraksi teks via `pdftotext`, index `tsvector` PostgreSQL)
- OCR untuk PDF scan tanpa text layer (Tesseract atau cloud OCR)
- Read-only role pihak ketiga (auditor/inspektorat)
- Push notification (Web Push API)
- Multi-tenancy (Tazkia scale ke kecamatan lain)
- Mobile-specific layout
- Migrasi antrian rekonsiliasi ke camat (sudah didesain — config change, bukan DB change)
- Calendar export (undangan → iCal)

---

## Cross-Cutting Concerns

### Concept Catalog (Setiap Fase)
- PR fitur baru wajib bawa minimum 1 marker `// concept:<id>:start|end` di source code
- PR review checklist: "apakah konsep representatif sudah ter-anchor?"
- Concept catalog deploy via GitHub Actions setiap merge ke main
- Coverage anchor tidak harus 100% — anchor hanya pada bagian representatif untuk konsep matkul, bukan setiap helper

### Student Mode (Setiap Fase)
- Setiap endpoint baru di-wrap middleware student-mode (kalau aktif untuk role/user, append `_edu` block)
- Frontend student drawer render `_edu` payload + dereference concept ID ke konten dari concept catalog
- Production binary hard-disabled student mode lewat compile-time flag

### Testing
- **Unit test**: `testing` package + `testify` (Go), Vitest (Vue)
- **Integration test**: testcontainers PostgreSQL untuk Go; mock service worker untuk Vue
- **E2E test**: Playwright
- **Coverage gate** ≥70% via CI

### Backup
- pg_dump harian ke object storage via rclone
- Verification: restore mingguan ke staging + smoke test, alert via email kalau gagal
- Retention: harian 7, mingguan 4, bulanan 12

### Deployment Pipeline
- Branch `main` → CI build → deploy ke staging otomatis
- Tag CalVer `YYYY.MM.NN` (mis. `2026.05.01`) → deploy ke production manual approval
- Migration auto-apply saat deploy (`goose up`) — schema saja, demo-seed terpisah
- VPS production: existing di Biznet Gio

### Monitoring
- Health check endpoint `/healthz` untuk uptime monitor eksternal
- Application log: structured JSON ke stdout, captured oleh systemd journal
- Error tracking: optional di Fase 6+ (Sentry self-hosted atau ekuivalen)

---

## Versi Roadmap

Roadmap ini hidup — di-update setiap akhir fase atau saat ada perubahan keputusan arsitektur signifikan. Perubahan tercatat di git history. Tidak ada tag snapshot terpisah — checkout app version tag (CalVer `YYYY.MM.NN`) sudah memberi state roadmap, schema, dan dataset yang konsisten di titik tersebut.
