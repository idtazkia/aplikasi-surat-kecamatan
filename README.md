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

## Status

Perencanaan selesai. Implementasi belum dimulai.

## Lisensi

Apache License 2.0 — lihat `LICENSE`.
