---
id: recursive-cte
courses: [basis-data, algoritma]
prereq: [graph-adjacency-list, dag-cycle-detection]
related: [tree-traversal, dag-cycle-detection]
fase: [2]
---

# Recursive CTE — Graph Traversal di SQL

> **Map ke materi kuliah**: Basis Data — Common Table Expressions; Algoritma — graph traversal (BFS-equivalent dengan UNION ALL).

## Teori

**Common Table Expression (CTE)** = named subquery yang scope-nya terbatas pada satu statement. **Recursive CTE** = CTE yang merefer ke dirinya sendiri, dipakai untuk traversal struktur hierarki/graph di SQL tanpa loop di application code.

Struktur CTE rekursif PostgreSQL:

```sql
WITH RECURSIVE cte_name AS (
    -- 1. Anchor (non-recursive term): row "seed" awal
    SELECT ... FROM ... WHERE seed_condition

    UNION ALL

    -- 2. Recursive term: refer ke cte_name (output iterasi sebelumnya)
    SELECT ... FROM cte_name JOIN ...
    WHERE termination_condition
)
SELECT * FROM cte_name;
```

Engine evaluasi:

1. Eksekusi anchor → working set initial
2. Eksekusi recursive term dengan working set sebagai input → temp set
3. Append temp set ke result + jadikan working set baru
4. Repeat sampai temp set empty

Kompleksitas: setiap iterasi adalah JOIN — total O(V + E) untuk graph dengan V vertex dan E edge, tergantung depth.

**Constraint PostgreSQL** (penting untuk dipahami):

- Tepat **1 anchor + 1 recursive term**, dipisah `UNION ALL` (atau `UNION` untuk dedup)
- Multiple branch traversal (mis. predecessor + successor sekaligus) harus digabung dalam satu recursive term — typically pakai `LATERAL JOIN` dengan `UNION ALL` di dalam.
- Recursive term tidak boleh refer ke CTE lain yang juga recursive

## Cycle Detection

Graph dengan cycle bisa bikin recursive CTE infinite loop. Solusi:

1. **Path tracking via array**: tiap row carry array `visited` berisi id yang sudah dilewati. Sebelum expand, check `NOT (next.id = ANY(visited))`.
2. **Depth cap**: set hard limit `WHERE depth < N` defensive di samping path tracking.

```sql
WITH RECURSIVE walk AS (
    SELECT id, ARRAY[id] AS path, 0 AS depth
    FROM nodes WHERE id = $1

    UNION ALL

    SELECT n.id, w.path || n.id, w.depth + 1
    FROM walk w
    JOIN edges e ON e.from_id = w.id
    JOIN nodes n ON n.id = e.to_id
    WHERE NOT (n.id = ANY(w.path)) AND w.depth < 50
)
SELECT * FROM walk;
```

Trade-off vs application-level recursion:

- **Pro CTE**: 1 round-trip ke DB; planner bisa optimize join order; data tetap di engine (bukan ditarik ke aplikasi tiap iterasi)
- **Pro app code**: lebih mudah debug, lebih flexible (mis. branch berdasarkan business logic), tidak terbatas SQL idiom

## Implementasi di App

Aplikasi pakai recursive CTE di 3 tempat:

1. **Thread korespondensi surat** (`internal/store/reference.go`):
   - Anchor: surat yang user buka
   - Recursive: traverse `surat_references` kedua arah (predecessor: from→to, successor: to→from) via `LATERAL UNION ALL`
   - Cycle detection via `visited` array; depth cap 50

2. **Linked list versi attachment** (`internal/store/attachment.go`):
   - Anchor: row attachment (any version)
   - Recursive #1: walk backward ke head (predecessor lookup via `replaced_by`)
   - Recursive #2: walk forward ke tail (successor lookup)
   - Output ordered head→tail

3. **(Future)** DAG disposisi turunan, tree organisasi instansi.

## Source Code

@anchor:recursive-cte

## Latihan

1. Tulis recursive CTE untuk hitung jumlah successor di graph surat — berapa banyak surat yang merujuk ke surat X transitively?
2. Modifikasi cycle detection untuk return jalur cycle (sequence node yang membentuk loop) bukan sekedar skip.
3. Bandingkan EXPLAIN ANALYZE recursive CTE vs application-loop yang panggil query 1-hop berulang. Konfirmasi single-CTE lebih cepat.
