---
id: set-operations
courses: [struktur-data]
prereq: [hash-table-map]
related: [hash-table-map, append-only-immutability]
fase: [0]
---

# Set

## Teori

**Set** = abstract data type yang menyimpan koleksi *unordered*, *tanpa duplikat*. Operasi dasar:

| Operasi | Hash-set (Go map) | Tree-set (BST balanced) |
|---|---|---|
| Add(x) | O(1) average | O(log n) |
| Contains(x) | O(1) | O(log n) |
| Remove(x) | O(1) | O(log n) |
| Union(A, B) | O(\|A\| + \|B\|) | O(\|A\| + \|B\|) |
| Intersect(A, B) | O(min(\|A\|, \|B\|)) | O(\|A\| + \|B\|) |
| Difference(A, B) | O(\|A\|) | O(\|A\| + \|B\|) |
| Iterate sorted | O(n log n) — perlu sort dulu | O(n) — tree native sorted |
| IsSubset(A, B) | O(\|A\|) | O(\|A\| + \|B\|) |

Kapan pilih masing-masing:

- **Hash-set**: kalau operasi dominan = membership check, no need for ordering. **Default pilihan untuk in-memory set di Go** (`map[K]struct{}`).
- **Tree-set**: kalau butuh range query ("elemen ≥ k"), iterate sorted, atau bounded universe yang kecil sehingga BST balanced lebih cache-friendly.

### Set Operations Mathematik

| Notasi | Nama | Definisi |
|---|---|---|
| A ∪ B | Union | semua elemen di A atau B atau keduanya |
| A ∩ B | Intersection | elemen yang ada di A **dan** B |
| A \\ B | Difference | elemen di A tapi **tidak** di B |
| A ⊆ B | Subset | semua elemen A juga ada di B |
| A × B | Cartesian product | semua pasangan (a, b), a ∈ A, b ∈ B |

Catatan: difference asymmetric. A\\B ≠ B\\A. Union dan intersect commutative dan associative.

## Implementasi di App

Reference implementation `Set[T]` generik di Go: `internal/datastruct/hashset`. Backed by `map[T]struct{}` (hash table dengan zero-byte value untuk hemat memory). Mendukung Add/Contains/Remove + set operations Union/Intersect/Difference + IsSubset.

Use case dalam app:

| Use case | Set yang dipakai |
|---|---|
| Permission user (semua permission dari semua role yang dimiliki) | Union dari permission set per role |
| Validasi ACL ("apakah user punya semua permission yang dibutuhkan?") | IsSubset(required, user_perms) |
| Audit "permission yang belum di-assign ke role X" | Difference(all_perms, role_perms) |
| Visited set di graph traversal (di `internal/datastruct/graph`) | Add saat visit, Contains saat encounter neighbor |
| Dedup elemen di slice | `FromSlice([1,2,2,3]).ToSlice()` |

## Source Code

@anchor:set-operations

@anchor:hash-table-map

(Set operations + hash-table backing — dua konsep yang saling melengkapi di file yang sama.)

## SQL Set Operations

PostgreSQL juga mendukung set operations native via `UNION`, `INTERSECT`, `EXCEPT`:

```sql
-- Semua permission yang dimiliki BAIK 'staf' DAN 'camat' (intersect)
SELECT permission_id FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE code='staf')
INTERSECT
SELECT permission_id FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE code='camat');

-- Permission yang dimiliki camat tapi tidak staf (difference)
SELECT permission_id FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE code='camat')
EXCEPT
SELECT permission_id FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE code='staf');

-- Gabungan permission yang dimiliki staf ATAU camat (union)
SELECT permission_id FROM role_permissions
WHERE role_id IN (SELECT id FROM roles WHERE code IN ('staf', 'camat'))
UNION
SELECT permission_id FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE code='admin');
```

Trade-off Go in-memory vs SQL:

| Konteks | In-memory Set di Go | SQL `UNION/INTERSECT/EXCEPT` |
|---|---|---|
| Latency 1 query | O(\|A\| + \|B\|) di RAM, ns range | RTT + parse + plan + execute, ms range |
| Batch query banyak | Sekali load semua, in-memory ops | Tiap query roundtrip |
| Cross-table set | Kompleks (load 2 collections) | Native — SQL JOIN + set operator |
| Memory | Linier set size | Tergantung query plan |

Untuk **batch processing dengan banyak set ops kecil** (mis. ACL check per request) → in-memory Go. Untuk **set ops sekali yang melibatkan banyak data** (mis. report distinct senders bulan ini) → SQL native.

## Eksperimen

1. Buka `internal/datastruct/hashset/set_test.go`. Jalankan test, perhatikan asimetri `Difference` di `TestSet_DifferenceAsymmetric`.

2. Implementasi cek "user X bisa akses surat dengan access_level secret?" pakai `IsSubset`:
   ```go
   userPerms := hashset.FromSlice(claims.Roles).Union(... // resolve permissions per role
   required := hashset.FromSlice([]string{"surat:read_secret"})
   allowed := required.IsSubset(userPerms)
   ```
   Bandingkan dengan implementasi linear scan — kapan O(1) lookup mulai matter?

3. Pertanyaan diskusi: di Vue auth store, `roles: string[]` saat ini di-implement sebagai array biasa. Kalau diganti ke `Set<string>` (JS native Set):
   - Bagaimana persist ke `localStorage`? (`JSON.stringify(set)` tidak work, butuh `[...set]` lalu `new Set(parsed)`)
   - Berapa kali per render `hasRole()` dipanggil? Kalau hanya 1-2x, set vs array negligible. Kalau >10x dengan checking banyak roles, set menang.

4. **Multiset variant**: counter-based set yang allow duplikat dengan count. Use case: histogram. Implementasi pakai `map[T]int`, conceptual extension dari Set[T]. Coba implementasikan `Multiset[T]` dengan Increment, Count, Top(k).

## Referensi

- [Go map[K]struct{} idiom](https://go.dev/wiki/SliceTricks#filtering-without-allocating)
- [PostgreSQL Set Operations](https://www.postgresql.org/docs/current/queries-union.html)
- [Set theory dasar — visual](https://en.wikipedia.org/wiki/Algebra_of_sets)
