# aplikasi-surat-kecamatan

Aplikasi manajemen surat masuk dan surat keluar untuk kantor kecamatan.

Proyek PkM STMIK Tazkia. Dibangun untuk melengkapi aplikasi surat keluar yang diterbitkan pemerintah — aplikasi ini menjadi arsip lokal yang dapat dioperasikan saat aplikasi pemerintah tidak tersedia, sekaligus menjadi sistem utama untuk surat masuk.

## Cakupan

- **Surat masuk**: sistem utama (belum ada aplikasi pemerintah yang menggantikan)
- **Surat keluar**: mirror lokal — nomor surat dan dokumen asli tetap dihasilkan di aplikasi pemerintah, aplikasi ini menyimpan PDF hasil unduh plus metadata untuk pencarian offline

## Arsitektur Ringkas

- Deployment: VPS tunggal
- Klien: PWA (Progressive Web App) dengan service worker untuk mode offline
- Penyimpanan offline: metadata di IndexedDB, PDF tetap di server (tidak dicache di browser)
- Multi-user: beberapa staf kecamatan melakukan input data; supervisor (camat) melakukan review dan rekonsiliasi

## Status

Perencanaan. Belum ada kode.

## Lisensi

Apache License 2.0 — lihat `LICENSE`.
