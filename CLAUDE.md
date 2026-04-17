# CLAUDE.md

Panduan untuk sesi Claude Code di repo ini. Baca sebelum melakukan perubahan.

## Konteks Proyek

Aplikasi manajemen surat masuk/keluar untuk kantor kecamatan, dibangun sebagai PkM STMIK Tazkia. Permintaan datang dari Bu Camat (nama kecamatan — TBD) karena aplikasi surat keluar pemerintah sering offline pada saat dibutuhkan.

Bu Camat bukan user data-entry utama. Data entry dilakukan oleh staf (beberapa orang). Bu Camat melakukan supervisi dan review, dengan frekuensi input sendiri sekitar 1–2x per minggu.

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
- Client-generated UUID (v7, roughly time-ordered) untuk semua entitas. Menghilangkan create-conflict antar-klien.
- Operation log + last-write-wins per field untuk update. Tidak menggunakan CRDT — overkill untuk domain ini.
- Append-only untuk komentar/catatan — conflict-free by construction.
- Setiap perubahan tercatat di audit log (siapa, kapan, apa). Dibangun dari hari pertama, bukan retrofit.

### Deduplikasi Surat (dua staf input surat yang sama saat offline)
Kunci deduplikasi berbeda per jenis:
- **Surat masuk**: `(normalized_sender + sender_nomor + tanggal_terima)`. Normalisasi nama pengirim perlu perhatian ("Kemendagri" vs "Kementerian Dalam Negeri").
- **Surat keluar**: `nomor_surat` (dari aplikasi pemerintah, globally unique).

Strategi:
1. **Online pre-save check**: lookup kunci deduplikasi sebelum simpan — warn "surat ini sudah ada, edit saja?". Menangkap kasus umum (kantor dengan internet normal).
2. **Offline merge-on-sync**: jika server mendeteksi duplikat saat sync, jangan tolak. Simpan kedua record terhubung ke kunci yang sama, munculkan di antrian rekonsiliasi.
3. **Merge UI**: tampilkan side-by-side, pilih salah satu atau edit jadi versi kanonik. Audit log menyimpan kedua original.

Antrian rekonsiliasi saat ini dipegang role **staf**. Bisa dipindah ke supervisor/camat nanti lewat permission matrix.

### Role & Permission
- Role-permission scaffolding ada dari hari pertama, walaupun semua role awalnya resolve ke permission yang sama.
- Alasan: menghindari migrasi skema saat role matrix perlu berubah. Show/hide fitur per role jadi config change, bukan DB change.
- Role awal: staf (data entry + rekonsiliasi), camat (supervisi + override), optional read-only untuk pihak lain.

### Autentikasi Offline
- Login online; token cached untuk bertahan offline.
- TTL token: balance — terlalu pendek = staf terkunci saat offline, terlalu panjang = laptop hilang jadi masalah. Tetapkan eksplisit.
- Refresh saat kembali online.

### UI Obligations
- Pending-sync indicator: setiap staf harus melihat "N perubahan belum terupload" agar tidak menutup browser dengan data yang masih di IndexedDB.
- Antrian rekonsiliasi duplikat: view khusus untuk role yang memegang rekonsiliasi.

## Open Items Sebelum Mulai Koding

- [ ] Nama kecamatan dan kontak resmi (Bu Camat, PIC staf)
- [ ] MoU/kesepakatan kerja: kepemilikan data, SLA, cakupan maintenance, batasan liability
- [ ] Jumlah staf yang akan jadi user (menentukan seeding role & ekspektasi beban)
- [ ] Stack pilihan: backend, frontend, DB, deployment tooling
- [ ] Hosting VPS: provider, region, backup strategy, monitoring
- [ ] Retention policy untuk PDF (berapa lama disimpan, arsip offline, dsb.)
- [ ] Data sensitivity review: surat pemerintah bisa mengandung data pribadi — tentukan enkripsi at-rest dan in-transit, akses log

## Referensi

- Catatan proyek di life repo: `~/workspace/life/tazkia/projects/aplikasi-surat-kecamatan.md`
