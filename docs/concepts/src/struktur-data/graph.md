---
id: graph-adjacency-list
courses: [struktur-data]
prereq: [tree-traversal, set-composite-key]
related: [tree-traversal, dag-cycle-detection, hash-table-map]
fase: [0, 2]
---

# Graph

> **Map ke materi kuliah**: [Skenario 4 — Ketertelusuran Korespondensi Antar-Institusi](../../../materi-kuliah-2025-struktur-data/case-study-aplikasi-surat-kecamatan.md). Surat-surat yang saling membalas dan men-tembuskan = graph dengan multiple parent dan potensial cycle. **Tidak cukup tree** seperti di disposisi internal (Skenario 3).

## Teori

**Graph** = (V, E) di mana V adalah set vertices dan E adalah set edges (pasangan vertex). Variasi:

| Properti | Variasi |
|---|---|
| Arah edge | Directed (digraph) vs Undirected |
| Edge weight | Weighted vs Unweighted |
| Cycle | Cyclic vs Acyclic |
| Connectivity | Connected vs Disconnected |
| Density | Sparse (\|E\| ≈ \|V\|) vs Dense (\|E\| ≈ \|V\|²) |

Kombinasi yang sering dibahas:
- **DAG** = directed + acyclic (lihat [DAG](./dag.md))
- **Tree** = connected + acyclic + undirected (atau directed dengan single parent)
- **Weighted DAG** = task scheduling dengan durasi
- **Bipartite graph** = vertex bisa di-partisi 2 set, edge selalu antar-set

## Representasi

Dua cara dominan:

| Representasi | Space | `is_neighbor(u, v)` | `neighbors(u)` |
|---|---|---|---|
| **Adjacency matrix** | O(V²) | O(1) | O(V) |
| **Adjacency list** | O(V + E) | O(degree(u)) | O(degree(u)) |

Adjacency list cocok untuk **sparse** graph (kebanyakan kasus real-world). Adjacency matrix cocok untuk dense graph atau kalau frekuensi `is_neighbor` query tinggi.

Di aplikasi-surat-kecamatan: `surat_references` adalah relational table yang ekuivalen dengan **edge list** (subset adjacency list). Tiap row = (from_surat_id, to_surat_id, relationship) = satu directed edge bertype.

## Algoritma Penting

### Traversal

| Algoritma | Struktur Bantuan | Use Case |
|---|---|---|
| **BFS** (Breadth-First Search) | Queue | Shortest path unweighted, level-order |
| **DFS** (Depth-First Search) | Stack (atau recursion) | Cycle detection, topological sort, SCC |

### Shortest Path

| Algoritma | Edge weight | Kompleksitas |
|---|---|---|
| **BFS** | unweighted | O(V + E) |
| **Dijkstra** | non-negative | O((V + E) log V) dengan priority queue |
| **Bellman-Ford** | negatif boleh, no negative cycle | O(V × E) |
| **Floyd-Warshall** | all-pairs | O(V³) |

### Cycle Detection

- DFS dengan color marking (white/gray/black)
- Topological sort yang gagal selesai → ada cycle

### Connectivity

- DFS / BFS untuk check connected
- Tarjan's algorithm untuk Strongly Connected Components

## Implementasi di App

`surat_references` di Fase 1 menyimpan edge:

```sql
CREATE TABLE surat_references (
    from_surat_id UUID,
    to_surat_id UUID,           -- nullable (external_ref untuk surat lama)
    relationship TEXT CHECK (...),
    ...
);
```

Vertex = `surat`, edge = `surat_references` row. Directed (from → to). Multi-edge (banyak relationship antar-pair). Potensial cyclic (misal balasan circular).

**Skenario 4** menyebut beberapa kebutuhan:

- **"Tunjukkan seluruh korespondensi terkait kejadian banjir"**: traversal dari surat root, ikuti edge — recursive CTE dengan visited set untuk avoid infinite loop di cycle.
- **"Jalur terpendek dari surat masuk pertama ke balasan di ujung"**: BFS untuk unweighted, atau Dijkstra kalau edge punya bobot (mis. lama waktu balasan).
- **"Lama waktu balasan sebagai bobot"**: weighted graph — bobot di kolom edge atau computed dari `tanggal_terima` minus `tanggal_surat`.

## Forward Reference (Fase 2)

Recursive CTE untuk traversal thread:

```sql
WITH RECURSIVE thread AS (
    SELECT id, perihal, ARRAY[id] AS path, 0 AS depth
    FROM surat WHERE id = $1
    UNION ALL
    SELECT s.id, s.perihal, t.path || s.id, t.depth + 1
    FROM surat s
    JOIN surat_references r ON r.to_surat_id = s.id
    JOIN thread t ON r.from_surat_id = t.id
    WHERE NOT s.id = ANY(t.path)        -- cycle prevention via visited set
      AND t.depth < 20                  -- safety net
)
SELECT * FROM thread ORDER BY depth;
```

`ARRAY` di-pakai sebagai visited set (lihat [Set](./set.md)).

## Bridge ke Materi Kuliah Java

Kelas Java di [`graph/Graph.java`](https://github.com/idtazkia/materi-kuliah-2025-struktur-data/blob/main/graph/Graph.java) mendemonstrasikan adjacency list di memory. Implementasi tipikal:

```java
class Graph {
    private Map<Node, List<Node>> adjacencyList = new HashMap<>();

    void addEdge(Node from, Node to) {
        adjacencyList.computeIfAbsent(from, k -> new ArrayList<>()).add(to);
    }

    List<Node> neighbors(Node n) {
        return adjacencyList.getOrDefault(n, List.of());
    }
}
```

Translasi pattern ke aplikasi:

| Materi Kuliah (Java in-memory) | aplikasi-surat-kecamatan (PostgreSQL) |
|---|---|
| `Map<Node, List<Node>>` adjacency list | tabel `surat_references` (edge list) |
| `addEdge(u, v)` | `INSERT INTO surat_references (from_surat_id, to_surat_id, ...)` |
| `neighbors(u)` | `SELECT to_surat_id FROM surat_references WHERE from_surat_id = ?` |
| BFS/DFS dengan loop | recursive CTE PostgreSQL |
| Visited set `Set<Node>` | `ARRAY` di kolom recursive CTE |

In-memory Java cocok untuk **transient computation** (sekali-jalan algoritma). Database table cocok untuk **persistent graph** yang banyak surat dan banyak query — query plan PostgreSQL handle index + join optimization tanpa kita harus tulis ulang.

## Big-O

| Operasi | Adjacency list | Edge table di SQL |
|---|---|---|
| Add edge | O(1) | O(log n) — B-Tree insert |
| Iterate neighbors | O(degree) | O(log n + k) — index scan |
| BFS / DFS | O(V + E) | O(V + E), tapi RTT per query |
| Shortest path BFS | O(V + E) | O(V + E) sekali query recursive CTE |

**Catatan untuk RTT**: kalau traversal dilakukan via banyak query terpisah, dominated by network round-trip. Pakai recursive CTE untuk single-roundtrip traversal.

## Eksperimen

1. Buka `db/migrations/demo-seed/0006_seed_references.sql`. Gambar manual graph yang dibentuk dari 9 reference seeded — vertex = surat (pakai perihal singkat), edge = relationship type. Identifikasi:
   - Apakah graph ini DAG atau ada cycle?
   - Berapa connected components?
   - Surat mana yang punya in-degree paling tinggi (di-rujuk paling banyak)?

2. Tulis recursive CTE untuk find semua surat yang reachable dari surat tertentu lewat edge `relationship = 'balasan' OR 'lanjutan'`. Bandingkan output untuk root surat di-seed yang berbeda.

3. Bayangkan tambah edge weight (mis. `respon_hari INT`). Modifikasi recursive CTE untuk compute total `respon_hari` di sepanjang path. Observasi: pure recursive CTE tidak optimal untuk Dijkstra — mengapa? (Hint: PostgreSQL recursive CTE eksplor BFS-style; Dijkstra butuh priority queue yang tidak ada di SQL).

4. Pertanyaan diskusi: kapan masalah cukup di-modelkan sebagai **tree** (Skenario 3), kapan harus **graph** (Skenario 4)? Apa yang membuat shape pohon tidak cukup?

## Referensi

- [CLRS Bab 22 — Elementary Graph Algorithms](https://mitpress.mit.edu/9780262046305/introduction-to-algorithms/)
- [CLRS Bab 24 — Single-Source Shortest Paths](https://mitpress.mit.edu/9780262046305/introduction-to-algorithms/)
- [PostgreSQL Recursive Queries](https://www.postgresql.org/docs/current/queries-with.html)
- Materi kuliah: [graph/Graph.java](https://github.com/idtazkia/materi-kuliah-2025-struktur-data/blob/main/graph/Graph.java)
