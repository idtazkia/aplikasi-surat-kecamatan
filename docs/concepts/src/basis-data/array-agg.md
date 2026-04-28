---
id: sql-aggregation-array-agg
courses: [basis-data]
prereq: [sql-group-by, sql-join]
related: [recursive-cte, btree-partial-index-soft-delete]
fase: [0]
---

# SQL Aggregation: array_agg + FILTER

## Teori

**Aggregation function** memproses banyak baris → satu nilai per group. Klasik: `COUNT`, `SUM`, `AVG`. Yang sering kurang dieksplorasi:

- `array_agg(col)` — kumpulkan nilai kolom jadi array
- `string_agg(col, sep)` — concat dengan separator
- `jsonb_agg(col)` — array JSON

**FILTER clause** (SQL standard, didukung PostgreSQL):

```sql
agg_function(expr) FILTER (WHERE predicate)
```

Hanya baris yang match predicate yang masuk agregasi. Beda dari `WHERE` biasa: `FILTER` per-aggregation, bukan global.

Contoh perbedaan:

```sql
-- WHERE: filter rows BEFORE grouping
SELECT user_id, COUNT(*)
FROM orders
WHERE status = 'paid'
GROUP BY user_id;

-- FILTER: per-aggregation, bisa beda criteria untuk agg yang beda
SELECT user_id,
       COUNT(*) FILTER (WHERE status = 'paid') AS paid_count,
       COUNT(*) FILTER (WHERE status = 'cancelled') AS cancelled_count
FROM orders
GROUP BY user_id;
```

## Implementasi di App

Login flow butuh **user data + roles** dalam satu query. Tanpa aggregation:

```sql
-- Query 1: get user
SELECT id, password_hash FROM users WHERE username = $1;

-- Query 2: get roles (separate)
SELECT r.code FROM user_roles ur JOIN roles r ON r.id = ur.role_id WHERE ur.user_id = $user_id;
```

Ini classic **N+1 problem** kalau scaled (mis. list 100 user dengan role mereka = 1 + 100 query).

Solusi: single query dengan `array_agg`:

```sql
SELECT u.id, u.password_hash,
       COALESCE(array_agg(r.code) FILTER (WHERE r.code IS NOT NULL), '{}') AS roles
FROM users u
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN roles r ON r.id = ur.role_id
WHERE u.username = $1
GROUP BY u.id;
```

Penjelasan:

- `LEFT JOIN` — user yang **tidak punya role** tetap muncul (tidak hilang seperti INNER JOIN)
- `FILTER (WHERE r.code IS NOT NULL)` — buang NULL yang muncul karena LEFT JOIN ke user tanpa role. Tanpa FILTER, hasil `array_agg` jadi `{NULL}` untuk user tanpa role
- `COALESCE(..., '{}')` — kalau hasil agregasi NULL (semua filtered out), return empty array
- `GROUP BY u.id` — wajib karena ada non-aggregated kolom

Result: satu row dengan field `roles TEXT[]` siap di-decode di Go.

## Source Code

@anchor:sql-aggregation-array-agg

## Big-O

| Pendekatan | Query count | DB round-trip | Time |
|---|---|---|---|
| N+1 (separate queries) | 1 + N | N+1 | O((1+N) × RTT) |
| Single query dengan array_agg | 1 | 1 | O(N × RTT_one + sort_cost) |

Untuk operasi single-user (login): perbedaan negligible. Tapi pattern ini scale ke list-N-user dengan role mereka — situ 1 query vs 1+N.

## Eksperimen

1. Test result struktur: jalankan query di `psql` dengan user yang punya 0, 1, atau multiple roles. Lihat bentuk `roles` TEXT[]:
   - 0 role: `{}` (empty array)
   - 1 role: `{staf}`
   - 2 role: `{staf,admin}`

2. Hapus `FILTER (WHERE r.code IS NOT NULL)`. Re-run untuk user tanpa role. Hasil: `roles = {NULL}`. Diskusi: kenapa empty array lebih clean dari `{NULL}` untuk konsumsi di Go?

3. Variasi advanced: tampilkan **per-role permission count** dengan FILTER:
   ```sql
   SELECT r.code,
          COUNT(*) FILTER (WHERE p.code LIKE 'surat:%') AS surat_perms,
          COUNT(*) FILTER (WHERE p.code LIKE 'audit:%') AS audit_perms
   FROM roles r
   LEFT JOIN role_permissions rp ON rp.role_id = r.id
   LEFT JOIN permissions p ON p.id = rp.permission_id
   GROUP BY r.code;
   ```

## Referensi

- [PostgreSQL Docs — Aggregate Functions](https://www.postgresql.org/docs/current/functions-aggregate.html)
- [PostgreSQL Docs — FILTER clause](https://www.postgresql.org/docs/current/sql-expressions.html#SYNTAX-AGGREGATES)
- N+1 Problem: [Bullet by Joe Ferris](https://github.com/flyerhzm/bullet) (Ruby world, but conceptual reference)
