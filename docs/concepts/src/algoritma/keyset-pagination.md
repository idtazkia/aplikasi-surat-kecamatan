---
id: keyset-pagination
courses: [algoritma, basis-data]
prereq: [btree-partial-index-soft-delete]
related: [btree-partial-index-soft-delete, sql-aggregation-array-agg]
fase: [1]
---

# Keyset Pagination

## Teori

**Pagination** = memecah hasil query yang besar jadi halaman-halaman kecil. Dua pendekatan utama:

### OFFSET-based (klasik, problematic)

```sql
SELECT * FROM surat
ORDER BY created_at DESC
LIMIT 20 OFFSET 200;
```

PostgreSQL harus **scan dan skip 200 baris pertama** sebelum return halaman ke-11. Implikasi:

| Issue | Penjelasan |
|---|---|
| **Slow di halaman jauh** | Page 100 (OFFSET 2000) scan 2000 baris sebelum filter LIMIT |
| **Drift hasil** | Kalau ada INSERT/DELETE antar fetch halaman, baris bisa skip atau duplikat |
| **No early-stop** | Database tetap proses skip walaupun klien hanya mau page tertentu |

Kompleksitas: O(offset + limit). Untuk dataset besar dengan halaman jauh, dominated by offset.

### Keyset-based (cursor)

```sql
SELECT * FROM surat
WHERE (created_at, id) < ($cursor_created_at, $cursor_id)  -- tuple compare
ORDER BY created_at DESC, id DESC
LIMIT 20;
```

Klien simpan **cursor** = (last_created_at, last_id) dari halaman terakhir. Query berikutnya filter `WHERE (created_at, id) < cursor`. Implikasi:

| Property | Penjelasan |
|---|---|
| **Stable** | Insert/delete tidak ganggu — cursor-based filter deterministik |
| **O(log n) per page** | B-Tree index seek langsung ke posisi cursor |
| **No deep-page penalty** | Page 1 = page 100 dalam waktu (kalau index sehat) |

Trade-off:
- ⚠ **Tidak bisa loncat ke halaman tertentu** — harus sequential next/prev. Untuk web app standard "next page" ini cukup; untuk "go to page 50" perlu OFFSET (atau hybrid).
- ⚠ **Cursor harus include tiebreaker** — kalau order by created_at saja dan ada duplikat timestamp, baris bisa skip. Tambah `id` sebagai tiebreaker.

## Big-O

| Pagination | Page 1 | Page N |
|---|---|---|
| OFFSET | O(limit) | O((N-1) × limit) — scan all skipped rows |
| Keyset | O(log n + limit) | O(log n + limit) — same regardless of N |

Untuk dataset 1M row, page 100 dengan limit 20:
- OFFSET: scan ~2000 row → ~milliseconds
- Keyset: index seek + scan 20 row → ~tens of microseconds

## Implementasi di App

`GET /api/surat` di Fase 1 pakai keyset pagination dengan cursor (`created_at`, `id`):

- **Response** include `next_cursor: { created_at, id }` kalau result count == limit (kemungkinan ada page berikutnya)
- **Request** kirim `?after_created_at=...&after_id=...&limit=20` untuk fetch page selanjutnya
- **B-Tree index** `(created_at, id)` ekspressed sebagai partial index `WHERE NOT is_deleted` di tabel `surat`

Tuple comparison di PostgreSQL: `(a, b) < (c, d)` ekuivalen dengan `a < c OR (a = c AND b < d)`. Compose natural untuk multi-column ordering.

## Source Code

@anchor:keyset-pagination

## Eksperimen

1. Run query dengan EXPLAIN untuk lihat plan keyset vs OFFSET:

   ```sql
   -- Keyset: index scan langsung ke posisi
   EXPLAIN ANALYZE
   SELECT * FROM surat
   WHERE NOT is_deleted
     AND (created_at, id) < ('2026-04-01 10:00:00'::timestamptz, '00000000-0000-0000-0007-000000000005')
   ORDER BY created_at DESC, id DESC
   LIMIT 20;

   -- OFFSET: tetap scan + skip
   EXPLAIN ANALYZE
   SELECT * FROM surat
   WHERE NOT is_deleted
   ORDER BY created_at DESC
   LIMIT 20 OFFSET 100;
   ```

   Bandingkan `Buffers shared hit` dan `Execution Time`.

2. Pertanyaan: kenapa cursor butuh **tiebreaker `id`** kalau sudah ada `created_at`? Hint: bayangkan 5 surat di-insert dalam transaksi yang sama → mungkin punya `created_at` identik. Tanpa tiebreaker, `(created_at) < cursor` skip atau duplikat baris di boundary.

3. Modifikasi keyset untuk support **bidirectional pagination** (next + prev). Hint: balikan operator dan ORDER untuk prev — jadi `(created_at, id) > cursor ORDER BY ASC LIMIT n` lalu reverse.

4. Hybrid scenarios: untuk admin yang butuh "go to page 50" sesekali, kombinasi keyset (default fast path) + OFFSET (slow path saat user explicit jump). Diskusi UX: kapan masing-masing cocok?

## Referensi

- [No Offset — Markus Winand](https://use-the-index-luke.com/no-offset)
- [Faster Pagination in MySQL — Surya](https://www.percona.com/blog/2024/01/12/why-not-offset/) (konsep applies ke PostgreSQL)
- [Cursor Pagination — Slack Engineering](https://slack.engineering/evolving-api-pagination-at-slack/)
