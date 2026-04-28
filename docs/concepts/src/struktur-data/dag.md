---
id: dag-cycle-detection
courses: [struktur-data]
prereq: [tree-traversal, graph-adjacency-list]
related: [graph-adjacency-list, set-composite-key]
fase: [0, 2]
---

# DAG (Directed Acyclic Graph)

> **Map ke materi kuliah**: [Skenario 4 — Ketertelusuran Korespondensi Antar-Institusi](../../../materi-kuliah-2025-struktur-data/case-study-aplikasi-surat-kecamatan.md). "Apakah mungkin ada siklus (surat A balas surat B, lalu B balas A lagi, lalu A balas lagi)? Kalau iya, bagaimana algoritma penelusuranmu menghindari loop tak terhingga?"
>
> Lihat juga [Graph](./graph.md) untuk konsep parent yang lebih general (DAG = special case graph + acyclic).

## Teori

**Graph** = (V, E) dengan vertices V dan edges E. Variasi:

- **Undirected**: edge tanpa arah
- **Directed (digraph)**: edge punya arah, ditulis (u, v) artinya u → v
- **Cyclic**: ada cycle (mulai dari node u, ikuti edges, balik ke u)
- **Acyclic**: tidak ada cycle

**DAG** = directed + acyclic. Tree adalah special case DAG dengan tepat satu in-edge per node (kecuali root).

Aplikasi DAG di komputasi:

- **Build dependency** (Make, Bazel): file A depends on B, B depends on C → topological order menentukan urutan compile
- **Task scheduling**: task dengan dependency → topological sort
- **Version control commit graph** (Git): commit DAG, parent-child relation, no cycle (kecuali rebase yang membuat alternative history)
- **Compiler IR**: SSA (Static Single Assignment) graph
- **Spreadsheet formula evaluation**: cell A=B+C requires evaluating B and C first
- **Surat references** (di app ini): surat A balas B, B lanjutan C — tipikal DAG, bisa cycle bug kalau user salah input

## Cycle Detection

Algoritma:

1. **DFS dengan color marking**:
   - White = belum visit
   - Gray = sedang di-visit (in DFS stack)
   - Black = sudah selesai
   - Saat traversal, kalau ketemu Gray node → cycle ditemukan
   - Kompleksitas: O(V + E)

2. **Tarjan's strongly connected components**: lebih kuat (mendeteksi semua SCC), juga O(V + E).

3. **Recursive CTE dengan depth limit** (PostgreSQL): bukan cycle detection sebenarnya, tapi cap depth untuk prevent infinite recursion. Praktis untuk app domain dengan known max depth.

## Topological Sort

Linear ordering vertex sehingga untuk setiap edge (u, v), u muncul sebelum v.

Algoritma:

1. **Kahn's algorithm** (BFS-based):
   - Mulai dari semua node dengan in-degree 0 → queue
   - Dequeue, "remove" node + edges ke neighbor
   - Neighbor yang in-degree turun ke 0 → enqueue
   - Sampai queue kosong
   - Kalau ada node tersisa → cycle ada (tidak bisa topo-sort)
   - Kompleksitas: O(V + E)

2. **DFS-based**: post-order DFS, push ke stack saat selesai. Reverse stack = topological order.

## Implementasi di App

`surat_references` adalah graph M:N:

- Vertex = surat
- Edge = relationship (balasan, lanjutan, disposisi_hasil, revisi, terkait)
- Direction: `from_surat_id` → `to_surat_id`

**Idealnya DAG**, tapi tidak guaranteed:

- Surat A merujuk B sebagai balasan
- Surat B (entah typo atau workflow weird) merujuk A sebagai lanjutan
- → Cycle

Cycle harus di-detect karena recursive CTE tanpa cycle detection = infinite loop.

## Implementasi di App

Reference implementation `HasCycle()` (3-color DFS) dan `TopologicalSort()` (Kahn's BFS-based) ada di package `internal/datastruct/graph`.

## Source Code

@anchor:dag-cycle-detection

## Forward Reference (Fase 2)

Implementasi cycle detection di recursive CTE:

```sql
WITH RECURSIVE thread AS (
    SELECT s.id, s.perihal,
           ARRAY[s.id] AS visited,    -- track visited node
           0 AS depth
    FROM surat s WHERE s.id = $1
    UNION ALL
    SELECT s.id, s.perihal,
           t.visited || s.id,
           t.depth + 1
    FROM surat s
    JOIN surat_references r ON r.from_surat_id = s.id
    JOIN thread t ON r.to_surat_id = t.id
    WHERE NOT s.id = ANY(t.visited)   -- cycle check
      AND t.depth < 20                -- safety net
)
SELECT * FROM thread;
```

`ARRAY` di PostgreSQL berperan sebagai set visited (lihat [Set](./set.md) untuk konsep underlying).

## Big-O

| Operasi | Kompleksitas |
|---|---|
| Cycle detection (DFS) | O(V + E) |
| Topological sort (Kahn's) | O(V + E) |
| Shortest path DAG | O(V + E) — pakai topo sort + relax |
| All pairs shortest path | O(V × E) untuk DAG, O(V³) untuk arbitrary graph (Floyd-Warshall) |

## Eksperimen (General)

1. Implementasi cycle detection di JavaScript dengan adjacency list:
   ```ts
   function hasCycle(graph: Map<string, string[]>): boolean {
     const visited = new Set<string>();
     const inStack = new Set<string>();

     function dfs(node: string): boolean {
       visited.add(node);
       inStack.add(node);
       for (const next of graph.get(node) ?? []) {
         if (!visited.has(next)) {
           if (dfs(next)) return true;
         } else if (inStack.has(next)) {
           return true; // back edge → cycle
         }
       }
       inStack.delete(node);
       return false;
     }

     for (const node of graph.keys()) {
       if (!visited.has(node) && dfs(node)) return true;
     }
     return false;
   }
   ```

2. Pertanyaan: kalau cycle terdeteksi di `surat_references`, apa response yang masuk akal? Reject save? Mark sebagai informational warning? Diskusi UX implications.

3. Aplikasi praktis: bayangkan klasifikasi hierarki dengan parent_id. Apa yang harus dicegah saat user assign parent? (Hint: cycle = klasifikasi A parent B, B parent A → infinite hierarchy).

## Referensi

- [CLRS Bab 22 — Elementary Graph Algorithms](https://mitpress.mit.edu/9780262046305/introduction-to-algorithms/)
- [Topological Sorting — Wikipedia](https://en.wikipedia.org/wiki/Topological_sorting)
- [PostgreSQL Recursive Queries](https://www.postgresql.org/docs/current/queries-with.html)
