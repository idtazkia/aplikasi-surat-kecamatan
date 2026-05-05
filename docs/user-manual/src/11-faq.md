# FAQ & Troubleshooting

## Login

### Q: Aplikasi tidak menerima login saya, padahal username/password benar.

Kemungkinan:
1. **CapsLock**: cek lampu CapsLock keyboard
2. **User inactive**: admin pernah disable akun Anda. Hubungi admin untuk verify status `is_active`
3. **Token expired di browser lain**: kalau Anda baru login di komputer lain, token lama otomatis tetap valid (multi-device). Tapi kalau Anda klik **Keluar** di komputer lain dengan opsi "logout dari semua device" (future feature), token Anda dicabut.

### Q: Kenapa setelah login langsung di-redirect ke /login lagi?

Token cached di browser tapi server reject. Kemungkinan: token expired (default 7 hari) atau JWT secret server berubah karena restart deployment. Solusi: clear browser data lalu login ulang.

### Q: Bisa simpan password di browser?

Bisa — browser native password manager akan tawarkan saat login. Disarankan **HANYA** di komputer pribadi, bukan di komputer kantor yang dipakai bergantian.

## Daftar Surat

### Q: Surat yang baru saya input tidak muncul di daftar.

Kemungkinan:
1. **Filter aktif**: cek apakah filter jenis/tanggal/keyword di sidebar Anda apply. Klik **Terapkan** dengan filter kosong untuk reset.
2. **Akses level secret**: kalau surat baru di-set akses `secret`, hanya camat/admin yang lihat. Kalau Anda staf, surat tersebut tidak akan tampil.
3. **Belum sync**: kalau Anda input offline dan masih ada pending sync (badge ⏳ > 0), surat ada di queue tapi belum reach server.

### Q: Search "tanggap" tidak menemukan surat dengan perihal "Tanggapan ...".

Full-text search match exact word. "tanggap" ≠ "tanggapan" karena tidak ada stemming Bahasa Indonesia. Coba search keyword yang persis sama dengan perihal, atau search dengan keyword lain yang lebih unik.

Future: stemming Bahasa Indonesia di "Further Improvement" roadmap.

### Q: Kenapa surat masuk tidak punya field "tanggal terima"?

Field tanggal terima muncul kalau jenis = `Surat Masuk`. Kalau Anda lihat form tanpa field tersebut, kemungkinan jenis-nya `Surat Keluar`. Surat keluar tidak punya konsep "tanggal terima" karena tidak diterima — diterbitkan.

## Disposisi

### Q: Saya staf — kenapa tidak bisa buat disposisi?

Anda bisa kalau punya role `staf`, `camat`, atau `admin`. Kalau tombol **+ Buat Disposisi** tidak muncul, cek role Anda di topbar (di samping nama). Hubungi admin kalau role Anda salah.

### Q: Camat tidak bisa override status disposisi saya.

Camat seharusnya bisa override semua disposisi. Kalau tidak bisa, kemungkinan bug — laporkan via GitHub issue dengan screenshot + steps reproduction.

### Q: Disposisi assignee saya kerja tapi statusnya masih "pending".

Assignee mungkin belum klik **Mulai**. Untuk eskalasi: post komentar di thread surat dengan ping ke assignee, atau hubungi langsung.

## Lampiran PDF

### Q: Upload PDF >25 MB gagal dengan error 413.

Limit per file 25 MB di server (untuk avoid disk fill + sustain upload time). Solusi: split PDF jadi beberapa file (mis. PDF utama 25 MB + lampiran-lampiran terpisah).

### Q: Preview PDF tidak muncul di iframe.

Kemungkinan:
1. Browser block embed iframe untuk security (Chrome strict mode). Klik **Unduh** instead.
2. PDF corrupted — buka langsung di browser via Unduh untuk verify.

### Q: Watermark di PDF download tidak muncul.

Watermark hanya di-apply untuk surat dengan akses `restricted` atau `secret`. Surat `public` di-download as-is. Cek akses level surat di header detail page.

### Q: Saya replace lampiran tapi versi lama hilang.

Versi lama TIDAK hilang — disimpan di chain history. Klik tombol **Versi** di lampiran untuk lihat semua versi historis. File "Aktif" hijau adalah versi terkini, yang lain adalah arsip.

## Notifikasi

### Q: Notifikasi muncul telat — sudah 5 menit lewat assignment, baru muncul.

Polling default 30 detik. Untuk refresh segera, klik tombol **🔄 Sync** di topbar. Future: push notification untuk real-time.

### Q: Notifikasi banyak, mau hapus semua sekaligus.

Klik bell 🔔 → tombol **Tandai semua dibaca** di pojok atas dropdown.

### Q: Saya tidak dapat notifikasi padahal seharusnya jadi participant.

Cek role Anda — kalau bukan staf/camat/admin/auditor, mungkin tidak masuk participant set. Notifikasi komentar dikirim ke participant disposisi (assignee + creator). Kalau Anda hanya komentator (bukan disposisi participant), Anda tidak akan dapat notif lanjutan.

## Offline & Sync

### Q: Saya edit surat saat offline, kemudian online lagi, tapi perubahan tidak masuk.

Cek badge ⏳ di topbar. Kalau ada angka, klik → list pending. Item dengan error merah berarti sync gagal — biasanya karena LWW (server lebih baru). Klik item untuk detail.

Kalau badge `0` tapi perubahan tetap tidak ada di server: kemungkinan IndexedDB Anda corrupt. Solusi: logout (auto clear cache) + login ulang.

### Q: Banner offline muncul padahal saya online.

Browser cache atau extension network monitoring kadang false-positive. Refresh page (Ctrl+R / Cmd+R) untuk reset connection state.

### Q: Aplikasi crash saat saya offline.

Kalau halaman blank putih, kemungkinan service worker belum cache asset yang Anda butuh (mis. route yang belum pernah Anda kunjungi). Solusi: kembali ke route yang sudah ter-cache (mis. `/surat`), atau wait sampai online untuk re-load.

## Performance

### Q: Daftar surat lambat di-load saat ada ribuan surat.

Pakai filter (jenis, tanggal, keyword) untuk mempersempit. Pagination keyset memastikan response time stabil regardless dari total surat — kalau lambat, kemungkinan filter terlalu broad.

### Q: Search keyword "panjang sekali" hasilnya kosong padahal saya yakin ada.

Full-text search match exact words. Phrase search tidak didukung saat ini. Kurangi keyword ke 1-2 kata unik. Future: `to_tsquery` dengan operators `&` dan `|`.

## Audit Log

### Q: Bagaimana cara akses audit log lengkap?

Audit log tabel di database. Untuk role pengguna umum (staf/camat), tampilan UI menampilkan komentar thread sebagai audit visible — append-only.

Untuk forensik mendalam (siapa edit field X kapan), butuh akses DBA. Hubungi admin sistem.

### Q: Ada surat yang "dihapus" lalu muncul lagi.

Surat di-soft-delete, kemudian admin restore. Audit log tabel `audit_log` mencatat sequence dengan timestamp + user.

## Browser Compatibility

### Q: Aplikasi support browser apa?

Yang ditest: **Chrome** + **Edge** + **Firefox** + **Safari** versi 2 tahun terakhir. PWA install bekerja optimal di Chromium-based (Chrome, Edge). Safari support PWA dengan limitasi (mis. notification API tidak ada, push notification tidak ada).

### Q: IE 11 atau browser lawas?

Tidak didukung. Service worker, IndexedDB API modern, dan fitur ES2022 dipakai aplikasi.
