---
id: uuid-v7-generation
courses: [struktur-data, algoritma]
prereq: [byte-order-endianness, hash-vs-time-ordered]
related: [btree-partial-index-soft-delete]
fase: [0]
---

# UUID v7 — Time-Ordered Identifier

## Teori

UUID (Universally Unique Identifier) adalah 128-bit identifier yang collision probability-nya sangat rendah tanpa koordinasi pusat. Versi yang umum:

| Versi | Source | Properti |
|---|---|---|
| **v4** | Random 122-bit | Tidak terurut. Insert ke B-Tree index → random walk, hot pages tersebar |
| **v7** | 48-bit timestamp + 74-bit random | Time-ordered. Insert berurutan → cache locality lebih baik |

UUIDv7 layout (RFC 9562, 2024):

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                           unix_ts_ms                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|          unix_ts_ms           |  ver  |       rand_a          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|var|                        rand_b                             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                            rand_b                             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- 48 bit pertama: Unix milliseconds, big-endian
- 4 bit version (= 7)
- 12 bit `rand_a`
- 2 bit variant (= 10₂)
- 62 bit `rand_b`

Total entropy random: 74 bit → collision practically impossible untuk skala kantor.

## Implementasi di App

Aplikasi-surat-kecamatan pakai UUIDv7 sebagai **primary key semua entitas** (surat, attachment, komentar, dst). Kenapa v7 bukan v4:

1. **Primary key = clustered index di banyak DB**. Insert UUIDv4 random membuat B-Tree page splits acak, bad locality. UUIDv7 insert selalu di "akhir" tree → page lebih dense.

2. **Sortable secara natural**. `ORDER BY id` ≈ `ORDER BY created_at`. Tidak perlu kolom timestamp tambahan untuk pagination berdasarkan urutan insert.

3. **Solve create-conflict di offline-first**. Dua client offline yang generate UUID untuk surat baru tidak akan collide karena 74-bit random + timestamp millisecond. Server tinggal accept tanpa validasi unik (untuk PK).

## Source Code

@anchor:uuid-v7-generation

Implementasi pakai `crypto/rand` untuk random bytes, `time.Now().UnixMilli()` untuk timestamp. Bit-manipulation eksplisit untuk set version (4 bit) dan variant (2 bit) sesuai spec.

## Big-O

| Operasi | Kompleksitas |
|---|---|
| Generate | O(1) — fixed 16 byte |
| Compare | O(1) |
| String roundtrip | O(1) — fixed 36 char |
| Insert ke B-Tree (v7) | O(log n) dengan **good locality** |
| Insert ke B-Tree (v4) | O(log n) dengan **poor locality** (page splits acak) |

## Eksperimen

1. Buka file `internal/uuid7/uuid7_test.go` — perhatikan test `TestNew_TimeOrdering`. UUID yang di-generate berurutan akan compare sebagai sorted-by-time **lewat string compare**, bukan butuh decode.

2. Modifikasi test: ganti `uuid7.New()` dengan random UUIDv4 (mis. `crypto/rand` 16 byte langsung tanpa set version 7), lihat apakah string compare masih preserve order.

3. Tambah benchmark insert ke PostgreSQL: 10,000 row dengan UUIDv7 vs UUIDv4. Bandingkan ukuran index pakai `SELECT pg_size_pretty(pg_indexes_size('table_name'))`.

## Referensi

- [RFC 9562 — UUID Version 7](https://datatracker.ietf.org/doc/rfc9562/)
- [Buzzword: Why UUIDv7 — Brandur Leach](https://brandur.org/nanoglyphs/026-ids)
