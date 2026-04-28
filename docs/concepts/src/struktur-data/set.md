---
id: set-composite-key
courses: [struktur-data, basis-data]
prereq: [hash-table-map, btree-partial-index-soft-delete]
related: [hash-table-map, append-only-immutability]
fase: [0]
---

# Set

## Teori

**Set** = koleksi unordered tanpa duplikat. Operasi dasar:

| Operasi | Kompleksitas (hash-set) | Kompleksitas (tree-set) |
|---|---|---|
| insert(x) | O(1) average | O(log n) |
| contains(x) | O(1) | O(log n) |
| delete(x) | O(1) | O(log n) |
| union(A, B) | O(|A| + |B|) | O(|A| + |B|) |
| intersect(A, B) | O(min(|A|, |B|)) | O(|A| + |B|) |
| iterate sorted | O(n log n) | O(n) — tree native sorted |

Kapan tree-set vs hash-set:

- **Hash-set**: kalau operasi dominan adalah membership check, no need for ordering
- **Tree-set**: kalau butuh range query, "smallest >= k", iterate sorted

## Implementasi di App

PostgreSQL primary key komposit `(role_id, permission_id)` = set semantik:

- Mathematik: `RolePermissions = {(r, p) | role r berhak permission p}`
- Tidak ada duplikat (unique constraint dari PK)
- `ON CONFLICT DO NOTHING` membuat seed idempotent (insert ulang = no-op)

PK komposit di PostgreSQL = B-Tree composite index → tree-set semantik. Kalau pure unique-only diperlukan, GIN/hash index alternatif. Kita pilih B-Tree karena:

1. Range query mungkin diperlukan (mis. "semua permission untuk role X" = scan range `role_id = X, permission_id any`)
2. Sorted iteration natural
3. Default index type, paling familiar untuk mahasiswa

## Source Code

@anchor:set-composite-key

## Big-O di Konteks App

| Query | Kompleksitas |
|---|---|
| `INSERT (r, p) ON CONFLICT DO NOTHING` | O(log n) |
| `SELECT EXISTS (... WHERE role_id=r AND permission_id=p)` | O(log n) |
| `SELECT permission_id FROM role_permissions WHERE role_id = r` | O(log n + k), k = perms per role |
| Set difference (perm yang belum di-assign) | O(n + m) dengan `EXCEPT` clause |

## Eksperimen

1. Set operations di SQL:
   ```sql
   -- Semua permission yang dimiliki BAIK role 'staf' DAN 'camat' (intersect)
   SELECT permission_id FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE code='staf')
   INTERSECT
   SELECT permission_id FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE code='camat');

   -- Permission yang dimiliki camat tapi tidak staf (set difference)
   SELECT permission_id FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE code='camat')
   EXCEPT
   SELECT permission_id FROM role_permissions WHERE role_id = (SELECT id FROM roles WHERE code='staf');
   ```

2. JavaScript `Set` di Vue auth store: `roles: string[]` sebenarnya bisa pakai `Set<string>`. Trade-off:
   - Array: serialize ke localStorage natural (JSON.stringify), order preserved
   - Set: contains-check O(1), tapi serialize butuh `[...set]`
   
   Kapan masing-masing menang? (Hint: berapa kali check membership per render).

3. PostgreSQL `aliases TEXT[]` di tabel `instansi` — adalah array, bukan set. Tidak ada uniqueness constraint built-in. Implikasi untuk dedup nama pengirim (Fase 5).

## Aplikasi Lain di App

- **`user_roles`** komposit PK — set (user, role) tuples
- **`surat_acl`** komposit PK — set (surat, user) untuk akses rahasia
- **Dedup key surat masuk** = composite tuple (instansi_id, nomor_surat, tanggal_terima) — semantik set untuk deteksi duplikat

## Referensi

- [PostgreSQL Set Operations](https://www.postgresql.org/docs/current/queries-union.html)
- [JavaScript Set vs Array](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Set)
