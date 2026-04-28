---
id: append-only-immutability
courses: [struktur-data, basis-data]
prereq: [linked-list, log-data-structure]
related: [operation-log-idempotency]
fase: [0, 2]
---

# Append-Only Pattern (Immutability)

## Teori

**Append-only** adalah disiplin di mana data structure hanya boleh ditambah, tidak boleh diubah atau dihapus. Sifat penting:

- **Conflict-free by construction**: dua aktor yang append concurrent tidak pernah konflik (urutannya saja yang perlu di-resolve, bukan kontennya)
- **Audit trail otomatis**: history = semua append. Tidak perlu mekanisme "log changes" terpisah
- **Time-travel cheap**: state pada waktu T = semua append dengan timestamp ≤ T

Trade-off:

- **Storage tumbuh monoton** — perlu archival policy untuk data lama
- **Query "current state"** butuh agregasi atau materialized view

Contoh struktur data yang inherently append-only:
- Linked list dengan tail pointer
- Append-only log file (mis. Kafka topic)
- Git commit graph

## Implementasi di App

Tabel `komentar` di aplikasi-surat-kecamatan dirancang **append-only**:

- Tidak ada kolom `updated_at`
- Tidak ada kolom `is_deleted`
- Tidak ada endpoint PATCH atau DELETE untuk komentar
- Salah ketik? Append komentar koreksi baru.

Konsekuensi positif:

1. **Kolaborasi offline tanpa konflik**. Dua staf yang sedang offline dan masing-masing append komentar di surat yang sama, saat sync online tidak ada konflik — kedua komentar tinggal masuk dengan timestamp masing-masing.

2. **Audit gratis**. Tidak butuh `audit_log` entry khusus untuk komentar. Semua history sudah tersimpan secara natural di tabel itu sendiri.

3. **Tidak butuh mekanisme LWW (last-write-wins)**. Pattern ini di-pakai untuk update; append-only menghindari kebutuhannya.

## Source Code

@anchor:append-only-immutability

Lihat juga test scenarios di Fase 4 (akan ditambah) yang demonstrasi 2 client append concurrent → keduanya masuk tanpa konflik.

## Big-O

| Operasi | Kompleksitas |
|---|---|
| Append | O(1) amortized (B-Tree insert dengan UUIDv7 — locality bagus) |
| Query semua komentar surat | O(log n + k) — index `(surat_id, created_at)` |
| Query "current state" | Sama dengan query semua (tidak ada current vs historical) |
| Storage growth | O(n) monoton — archival diperlukan jangka panjang |

## Eksperimen

1. Buka `db/migrations/schema/0001_init.sql`, cari tabel `komentar`. Bandingkan dengan tabel `surat` yang punya `updated_at` dan `is_deleted`.

2. Pertanyaan diskusi: kenapa `disposisi` punya `status` field yang berubah (pending → done) sementara `komentar` immutable? Jawaban: disposisi adalah **state machine** dengan transisi yang valid; komentar adalah **fact** yang sudah terjadi.

3. Eksperimen dengan tabel `audit_log`: meskipun namanya log, dia juga append-only by construction. Coba apakah ada handler yang melakukan UPDATE/DELETE ke audit_log (jawabannya: tidak).

## Referensi

- [Designing Data-Intensive Applications](https://dataintensive.net/) — Bab 11 (Stream Processing) bahas log-based architecture
- [Event Sourcing — Martin Fowler](https://martinfowler.com/eaaDev/EventSourcing.html)
