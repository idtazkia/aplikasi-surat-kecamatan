# Mode PWA & Offline

Aplikasi ini adalah **Progressive Web App** (PWA) — bisa di-install di laptop/komputer/HP seperti aplikasi native, dan tetap berfungsi sebagian saat offline.

## Mengapa PWA

Konteks kantor kecamatan: internet kadang tidak stabil. Aplikasi pemerintah pusat sering down. Solusi:

- **Cache di klien**: data surat (metadata) tersimpan di browser via IndexedDB
- **Service worker**: intercept request → kalau API gagal, fallback ke cache
- **Indicator offline**: user transparent kalau sedang offline + kapan terakhir sync

## Instalasi sebagai App

Di Chrome/Edge:
1. Buka aplikasi di tab biasa
2. Address bar di kanan akan muncul ikon **install** ➕
3. Klik → konfirmasi
4. Aplikasi muncul di Start Menu / Application sebagai app standalone

Di Safari (iOS / macOS):
1. Tap tombol **Share** ⤴
2. Pilih **Add to Home Screen** / **Add to Dock**
3. Konfirmasi nama

Setelah install:
- Ikon di desktop/launcher
- Window terpisah (tanpa address bar browser)
- Workflow lebih mirip aplikasi native

## Apa yang Di-Cache Otomatis

Saat login pertama, aplikasi otomatis sync **snapshot** semua data master + surat metadata ke browser local:
- Semua surat (metadata only — perihal, nomor, instansi, tanggal, dll.)
- Master instansi (untuk autocomplete)
- Master klasifikasi & sifat
- Watermark `last_sync_at` untuk tracking staleness

**Yang TIDAK di-cache:**
- File PDF lampiran (terlalu besar — tetap online-only)
- Komentar (real-time multi-user)
- Disposisi (dynamic state)
- Notifikasi (poll-based)

> **Rationale storage**: PDF di-cache berarti puluhan-ratusan MB di laptop staf. Membatasi cache ke metadata = ringan + tetap berguna untuk lihat daftar surat saat offline.

## Auto Sync

Snapshot sync trigger di:
1. **Login pertama**: full snapshot
2. **Setelah reconnect online**: delta sync (hanya yang berubah sejak watermark)
3. **Periodik 30 detik** (drainer pending queue)
4. **Manual** via tombol **🔄 Sync**

Delta sync mendukung **tombstones** — kalau ada surat yang dihapus di server, klien terima list `surat_deleted_ids` dan menghapusnya dari cache lokal.

## Skenario Offline Berhari-hari

Misalnya kantor mati listrik 2 hari. Saat kembali online:

1. Browser fire event `online` → service worker aktif kembali
2. Snapshot sync auto-trigger dengan `since=last_watermark`
3. Server return delta surat yang berubah selama 2 hari + tombstones
4. Pending write queue auto-drain (mis. edit yang dilakukan saat offline)
5. **Service worker auto-update**: kalau ada deployment versi baru selama offline, SW akan grab + apply saat next online visit (semantic "update on next online visit")

> **No data loss**: edit Anda saat offline TIDAK hilang. Tetap di IndexedDB queue. Saat reconnect, sync ke server.

## Konflik Saat Sync

Skenario: Anda edit perihal surat saat offline. Sementara itu camat juga edit perihal surat yang sama dari laptop lain (online).

Saat Anda reconnect, sync queue akan kirim edit Anda. Server lakukan **Last-Write-Wins (LWW)** check:

- **Kalau edit Anda lebih baru** dari server: edit Anda apply (overwrite camat)
- **Kalau edit camat lebih baru**: edit Anda **rejected** dengan reason `stale: server has newer update (LWW lost)`

UI akan tampilkan ini di pending sync dropdown sebagai item dengan error merah.

> **Mitigasi**: Kalau edit penting Anda di-reject karena LWW, lihat detail surat untuk lihat versi server saat ini. Kalau perlu, edit ulang dengan modifikasi yang sesuai.
>
> Future: per-field LWW yang lebih granular (mis. Anda edit perihal, camat edit instansi → keduanya merge). Lihat ROADMAP "Further Improvement".

## Pengelolaan Storage

Browser akan otomatis kelola storage. Tapi kalau Anda butuh manual reset:
- **Logout**: cache cleared otomatis (mencegah leak antar user di shared device)
- **Clear browser data**: cache hilang. Saat login berikutnya, full snapshot akan dilakukan ulang

Storage size estimate: 1-2 MB per 1000 surat metadata. Acceptable untuk ribuan surat.

## Limitasi Service Worker di Dev Mode

Service worker default di-disable saat development (`npm run dev`). Untuk test SW behavior penuh, build production:

```bash
make web-build && cd web && npx vite preview
```

Atau jalankan via Docker (`docker compose up`) yang sudah pakai production build.
