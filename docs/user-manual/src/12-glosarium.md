# Glosarium

Definisi istilah teknis yang sering muncul di aplikasi dan panduan ini.

## A

**Append-Only**
:   Pattern data: row hanya bisa ditambah, tidak bisa diubah/dihapus. Dipakai untuk komentar thread + audit log. Audit by construction — tidak perlu tabel audit terpisah karena data sendiri immutable.

**Audit Log**
:   Tabel `audit_log` di database yang catat setiap perubahan: siapa (user_id), kapan (timestamp), apa (action: create/update/delete), before-after JSONB. Tidak terlihat di UI normal, akses via DBA.

**Auditor**
:   Role read-only untuk pemeriksa eksternal (inspektorat, BPK). Akses lihat surat tapi tidak bisa edit/buat/hapus. Tidak akses surat `secret`.

## B

**B-Tree Index**
:   Struktur data index database PostgreSQL default. Disorting + balanced. Cocok untuk lookup, range scan, dan sort. Index `(created_at, id)` di table `surat` adalah B-tree untuk keyset pagination.

## C

**Camat**
:   Role supervisor di kantor kecamatan. Wewenang: assign disposisi, review semua surat (termasuk secret), resolve antrian rekonsiliasi, akses dashboard + statistik.

**Concept Catalog**
:   mdBook static site di `docs/concepts/` yang berisi penjelasan konsep matkul (Struktur Data, Algoritma, Basis Data) yang dipakai di app. Linkage ke source code lewat marker `// concept:<id>:start|end`.

**CTE (Common Table Expression)**
:   Named subquery di SQL. Recursive CTE: refer ke dirinya sendiri untuk traversal struktur hierarki/graph (mis. thread korespondensi surat).

## D

**Dedup Tuple**
:   Kombinasi field yang dijadikan kunci dedup deteksi. Untuk surat masuk: `(instansi_id, nomor_surat, tanggal_terima)`. Untuk surat keluar: `nomor_surat` (unique constraint global).

**Delta Sync**
:   Sync mode di mana server hanya kirim row yang berubah sejak watermark terakhir, bukan full snapshot. Hemat bandwidth + faster.

**Dexie**
:   Library JavaScript untuk IndexedDB API yang ergonomis. Dipakai untuk cache lokal di browser (surat metadata + write queue).

**Disposisi**
:   Penugasan formal surat ke staf untuk tindak lanjut. Status: pending → in_progress → done (atau cancelled).

## F

**FTS (Full-Text Search)**
:   Pencarian berbasis index `tsvector` PostgreSQL. Tokenize text + match per-keyword. Lebih cepat dan akurat dari `ILIKE %text%` untuk dataset besar.

## G

**GIN Index**
:   Generalized Inverted Index. Index PostgreSQL untuk struktur data komposit seperti `tsvector`, `jsonb`, array. Dipakai untuk full-text search di table `surat`.

## I

**IndexedDB**
:   Database NoSQL embedded di browser, key-value + index. API native browser. Dipakai aplikasi untuk cache offline (via Dexie wrapper).

**Idempotency**
:   Property operasi: bisa dijalankan berulang kali tanpa side effect berbeda dari yang pertama. Operation log pakai `client_op_id` PK constraint untuk enforce idempotency by-database.

## K

**Keyset Pagination**
:   Pagination berbasis cursor `(created_at, id)` daripada `OFFSET`. Stable saat ada insert/delete di antara fetch, dan response time konstan O(log n + page_size) tidak peduli halaman ke berapa.

## L

**Linked List**
:   Struktur data sequence node terhubung via pointer `next`. Dipakai untuk versioning attachment: setiap row punya `replaced_by` ke versi pengganti.

**LWW (Last-Write-Wins)**
:   Strategi resolusi konflik: bandingkan timestamp, yang lebih baru menang. Dipakai untuk sync queue — kalau client edit lebih lama dari server, reject dengan reason `stale`.

## M

**mdBook**
:   Static site generator untuk markdown. Dipakai untuk concept catalog dan user manual ini. Single binary, fast build, search bawaan.

## N

**Naive UI**
:   Vue 3 component library yang dipakai aplikasi. Pure Vue 3, TypeScript first-class, bundle size kecil.

## O

**Operation Log**
:   Tabel `operation_log` yang catat semua mutation klien (terutama saat offline). PK `client_op_id` untuk idempotency. Dipakai sync queue.

## P

**PWA (Progressive Web App)**
:   Web app yang bisa di-install seperti native app, dengan service worker untuk offline support + cache strategy.

**plainto_tsquery**
:   PostgreSQL function: convert plain text input ke tsquery (struktur full-text search). Match per-keyword AND.

## R

**Rekonsiliasi**
:   Proses menyelesaikan surat duplikat (dua staf input surat yang sama). Antrian `reconciliation_queue` menampung pending. Camat resolve via merge atau keep-both.

## S

**Service Worker**
:   Script JavaScript yang berjalan di background browser, intercept network request, manage cache. Berlaku PWA + offline mode.

**Soft Delete**
:   Hapus dengan `is_deleted = TRUE` flag, bukan DELETE row. Audit log preserved, restore mungkin.

**Student Mode**
:   Fitur edukasi yang inject `_edu` payload di response API untuk role student. Frontend drawer tampilkan struktur data, complexity, SQL, link ke concept catalog.

## T

**Tembusan**
:   Distribution list untuk surat — instansi/kontak yang juga harus dapat copy. Bisa internal (instansi terdaftar) atau external (text bebas).

**Tombstone**
:   Marker "row dihapus" di delta sync. Server kirim list `surat_deleted_ids` saat delta sync, klien pakai untuk purge cache lokal.

**Trie**
:   Prefix tree. Struktur data untuk autocomplete cepat. Belum dipakai di aplikasi (ada di "Further Improvement").

**tsvector**
:   PostgreSQL data type untuk lexemes (kata yang sudah di-tokenize). Dipakai oleh FTS search engine.

## U

**UUIDv7**
:   Versi UUID dengan timestamp 48-bit di prefix. Time-ordered → cocok untuk PK B-tree append-only insert. Dipakai aplikasi untuk semua entity (surat, disposisi, komentar, ops, dll.).

## V

**VAPID**
:   Voluntary Application Server Identification — protokol untuk push notification authentication. Belum dipakai aplikasi (Further Improvement).

## W

**Watermark**
:   Overlay teks "Diunduh oleh X — timestamp" diagonal di setiap halaman PDF saat download. Hanya untuk surat akses `restricted` atau `secret`. Implementasi via library `pdfcpu`.

**Watermark Sync**
:   Timestamp server saat snapshot delta sync. Klien simpan, kirim balik di sync berikutnya untuk request delta dari titik tersebut.
