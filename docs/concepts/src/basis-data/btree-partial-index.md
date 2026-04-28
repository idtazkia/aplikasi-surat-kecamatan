---
id: btree-partial-index-soft-delete
courses: [basis-data, struktur-data]
prereq: [btree-data-structure, sql-where-clause]
related: [array-agg, recursive-cte]
fase: [0, 1]
---

# B-Tree Partial Index (Soft Delete)

## Teori

**B-Tree** adalah struktur data tree yang self-balancing dengan banyak children per node. Cocok untuk disk-based index karena meminimalkan I/O — dengan branching factor tinggi (mis. 100+), tree dengan jutaan entry hanya butuh 3-4 level.

PostgreSQL default index = B-Tree. Properti:

- Lookup: O(log n)
- Range scan: O(log n + k) di mana k = result count
- Insert: O(log n) amortized
- Sorted iteration: free (B-Tree leaf nodes linked)

**Partial index** = B-Tree yang hanya mengindeks subset baris yang match `WHERE` predicate. Sintaksis PostgreSQL:

```sql
CREATE INDEX idx ON table_name (column_name)
WHERE predicate;
```

Trade-off:

- **Lebih kecil**: hanya baris yang predicate-true di-index → memory + disk hemat
- **Lebih cepat**: less data to traverse, more pages fit in cache
- **Query harus include predicate** untuk planner pakai partial index

## Implementasi di App

Soft delete: tabel `surat` punya kolom `is_deleted BOOLEAN`. Hard delete tidak dilakukan untuk preservation audit trail.

Tanpa partial index:

```sql
CREATE INDEX idx_surat_tanggal ON surat (tanggal_terima);
```

Index ini akan termasuk surat yang sudah di-soft-delete. Untuk dataset skala puluh ribu surat dengan ~1% sudah deleted, ini buang-buang storage 1%.

Dengan partial index:

```sql
CREATE INDEX idx_surat_tanggal_terima ON surat (tanggal_terima)
    WHERE NOT is_deleted AND jenis = 'masuk';
```

- 99% lebih kecil dari index full
- Filter umum di list view (`WHERE NOT is_deleted AND jenis = 'masuk' AND tanggal_terima BETWEEN ...`) langsung bisa pakai partial index
- Trade-off: kalau query tidak include predicate, partial index tidak terpakai → fallback seq scan

## Source Code

@anchor:btree-partial-index-soft-delete

## Big-O

| Operasi | Tanpa index | Full index | Partial index (kalau predicate match) |
|---|---|---|---|
| Lookup by tanggal | O(n) seq scan | O(log n) | O(log m) di mana m = subset size |
| Range scan | O(n) | O(log n + k) | O(log m + k) |
| Insert | O(1) tabel saja | O(log n) untuk index | O(log m) — lebih cepat |
| Storage | tabel saja | tabel + n entry | tabel + m entry (m < n) |

## Eksperimen

1. Setup test: insert 10,000 surat ke staging DB, soft-delete 100 (random).

2. Run query:
   ```sql
   EXPLAIN ANALYZE
   SELECT * FROM surat
   WHERE NOT is_deleted AND jenis = 'masuk'
   ORDER BY tanggal_terima DESC LIMIT 20;
   ```
   Lihat planner: `Index Scan using idx_surat_tanggal_terima`.

3. Drop partial index, ganti dengan full index:
   ```sql
   DROP INDEX idx_surat_tanggal_terima;
   CREATE INDEX idx_surat_tanggal_full ON surat (tanggal_terima);
   ```
   Run query yang sama. Bandingkan:
   - Index size: `pg_size_pretty(pg_indexes_size('surat'))`
   - Buffer hits: `EXPLAIN (ANALYZE, BUFFERS) ...`

4. Pertanyaan: query `SELECT * FROM surat WHERE id = ?` (lookup by PK) — apakah partial index relevant? (Jawab: tidak, PK punya index sendiri yang meliputi semua baris).

## Referensi

- [PostgreSQL Docs — Partial Indexes](https://www.postgresql.org/docs/current/indexes-partial.html)
- [Use The Index, Luke! — Markus Winand](https://use-the-index-luke.com/)
- [The Art of PostgreSQL — Dimitri Fontaine](https://theartofpostgresql.com/)
