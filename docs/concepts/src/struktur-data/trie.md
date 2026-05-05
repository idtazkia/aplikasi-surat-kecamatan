---
id: trie-prefix-tree
courses: [struktur-data]
prereq: [tree-traversal, hash-table-map]
related: [tree-traversal, hash-table-map]
fase: [0, 5]
---

# Trie (Prefix Tree)

> **Map ke materi kuliah**: [Skenario 6 Bagian A — Autocomplete instansi pengirim](https://github.com/idtazkia/materi-kuliah-2025-struktur-data/blob/main/case-study-aplikasi-surat-kecamatan.md). "Saat staf mulai mengetik 'Kem...', sistem harus segera menampilkan daftar instansi yang cocok... harus terasa instan (di bawah 50 milidetik) meski ada 500 pilihan."

## Teori

**Trie** (dari "re**trie**val") = tree di mana path dari root ke node merepresentasikan **string prefix**. Setiap node punya:

- Mapping `char → child node`
- Flag `is_end_of_word` (apakah path sampai sini = kata lengkap)

Properti unik:

- **Shared prefix** — kata `"Kemendagri"`, `"Kemensos"`, `"Kemenkes"` share prefix `"Kemen"`, hanya divergen setelahnya
- **Lookup by prefix O(k)** di mana k = panjang prefix, **bukan** O(n) di mana n = jumlah kata

```
              root
              /
             K
             /
            e
            /
           m
           /
          e
          /
         n
        / | \
       d  s  k
      /   |   \
     a   o    e
    /    |    \
   g   s(•)   s(•)
   /
  r
  /
 i(•)            (•) = is_end_of_word
```

## Big-O

| Operasi | Trie | Hash table | Sorted array + binary search |
|---|---|---|---|
| insert(word) | O(k) | O(k) — hash word | O(n × k) — shift |
| search exact | O(k) | O(k) average | O(k log n) |
| **search prefix → daftar** | **O(k + p)** p = result size | O(n × k) — scan semua | O(k log n + p) — binary search dapat first match, scan |
| Memory | O(total chars) | O(n × avg_len) | O(n × avg_len) |

**Trie menang untuk prefix search**. Hash table tidak punya prefix concept — harus iterate semua key. Sorted array bisa, tapi binary search harus comparator-aware untuk "starts with".

## Implementasi di App

### Saat ini (Fase 1, sederhana)

Direktori instansi auto-collect (Fase 1) pakai PostgreSQL untuk autocomplete:

```sql
SELECT nama_kanonik FROM instansi
WHERE nama_kanonik ILIKE 'Kem%'
   OR EXISTS (SELECT 1 FROM unnest(aliases) a WHERE a ILIKE 'Kem%')
LIMIT 10;
```

Untuk dataset 500 instansi, ILIKE prefix dengan B-Tree text index sudah ~20ms. Cukup untuk Skenario 6 requirement (50ms). Tidak butuh in-memory trie.

### Reference Implementation

Generic trie di Go ada di `internal/datastruct/trie`. Sederhana, runes (Unicode), Insert/Contains/SearchPrefix/Delete dengan empty-branch cleanup. Cocok untuk dipakai langsung di backend Go atau di-port ke TypeScript untuk client-side cache.

## Source Code

@anchor:trie-prefix-tree

### Trade-off In-Memory Trie vs SQL `ILIKE`

| In-memory trie (`internal/datastruct/trie`) | PostgreSQL `ILIKE 'prefix%'` (Fase 1) |
|---|---|
| O(k + p) lookup | O(log n + p) dengan B-Tree text index |
| Memory: load semua kata ke trie | Memory: O(1) di app, DB kelola index |
| Update: butuh sync trie + DB | Update: cukup INSERT/UPDATE row |
| Fuzzy match dengan Levenshtein: O(k × m × \|alphabet\|) dengan pruning | SQL: ekstensi `pg_trgm` + GIN index untuk fuzzy |

Untuk **scale kecil (≤ 1000 entries)**: SQL cukup, no trie needed.
Untuk **scale besar (> 100k entries) atau frequent fuzzy lookup**: in-memory trie atau dedicated search engine (Elasticsearch, Meilisearch) menang.

## Eksperimen

1. Buka `internal/datastruct/trie/trie.go` dan baca implementasinya. Tulis benchmark Go: insert 1000 nama instansi, lookup prefix vs `slices.IndexFunc` linear scan. Bandingkan ns/op.

2. Pertanyaan: kalau alphabet hanya `[a-z]` (26 char), kenapa lebih efisien pakai array of 26 child slots daripada `Map`? Berapa speedup factor expected?

3. **Compressed trie / Radix tree**: kalau banyak node di trie cuma punya 1 child (long shared prefix tanpa branching), bisa di-collapse jadi single edge. Implementasi dan perbandingan space-efficiency.

4. Aplikasi: bayangkan autocomplete bukan hanya `nama_kanonik` tapi juga `aliases`. Apakah satu trie cukup atau perlu separate? Apa implikasi memory? (Hint: bisa pakai single trie dengan flag node "alias of X").

## Referensi

- [CLRS Bab 32.4 — Trie](https://mitpress.mit.edu/9780262046305/introduction-to-algorithms/)
- [PostgreSQL pg_trgm — fuzzy text search](https://www.postgresql.org/docs/current/pgtrgm.html)
- [Suffix Tree, Suffix Array — variasi trie advanced](https://en.wikipedia.org/wiki/Suffix_tree)
