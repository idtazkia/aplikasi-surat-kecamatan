---
id: tree-traversal
courses: [struktur-data]
prereq: [linked-list-version-chain]
related: [btree-partial-index-soft-delete, dag-cycle-detection, graph-bfs-dfs]
fase: [0, 2]
---

# Tree

> **Map ke materi kuliah**: [Skenario 3 — Disposisi Surat Internal](../../../materi-kuliah-2025-struktur-data/case-study-aplikasi-surat-kecamatan.md). "Bagaimana kamu menyimpan hubungan 'siapa disposisi ke siapa' untuk satu surat?... Camat → Sekcam → Kasi Pemerintahan → Staf Ani → Magang-1." Hierarki single-parent → tree.
>
> Untuk korespondensi antar-instansi (Skenario 4) yang punya multiple parent + cycle, lihat [Graph](./graph.md).

## Teori

**Tree** = directed acyclic graph dengan struktur:

- Tepat satu **root** (node tanpa parent)
- Setiap node lain punya tepat satu **parent**
- Tidak ada cycle

Terminologi:

- **Root**: node tanpa parent
- **Leaf**: node tanpa children
- **Depth** node: jumlah edge dari root ke node tersebut
- **Height** tree: depth dari leaf terjauh
- **Subtree**: setiap node + semua keturunannya

Variasi penting:

| Tree | Properti |
|---|---|
| **Binary tree** | Maks 2 children per node |
| **Binary Search Tree (BST)** | Binary + invariant left < node < right |
| **AVL / Red-Black** | BST self-balancing → operasi O(log n) guaranteed |
| **B-Tree** | Multi-way branch (>2), optimasi disk I/O |
| **Heap** | Complete binary tree dengan heap-property (lihat [Heap](./heap.md)) |
| **Trie** | Tree tempat path = string; cocok untuk prefix lookup |

## Traversal

| Pre-order | Visit node, lalu left, lalu right |
| In-order | Left, node, right (BST → sorted output) |
| Post-order | Left, right, node |
| BFS / Level-order | Pakai queue: tiap level dari atas ke bawah |
| DFS | Pakai stack atau recursion: dalam dulu sebelum melebar |

Kompleksitas semua traversal: O(n).

## Implementasi di App

Beberapa struktur tree di app:

1. **`surat_references`** — bukan strict tree (potensial DAG/graph karena bisa multiple parent + cycle). Tree-like saat traversed dari satu surat — Fase 2 akan implement recursive CTE untuk traversal thread korespondensi.

2. **`surat_attachments` version chain** — degenerate tree (linked list = tree dengan branching factor 1). Lihat [Linked List](./linked-list.md).

3. **B-Tree index PostgreSQL** (lihat [B-Tree Partial Index](../basis-data/btree-partial-index.md)) — multi-way tree untuk index lookup, internal di engine DB.

4. **Klasifikasi hierarki** (potential, belum ada): kode klasifikasi 100, 110, 111 punya implicit hierarchy. Bisa di-model sebagai tree pakai `parent_id` self-reference. Saat ini flat (single level kode).

5. **DOM tree** di frontend Vue — implicit tree HTML elements.

Reference implementation generic n-ary tree dengan BFS dan DFS traversal ada di package `internal/datastruct/tree`. Tree pakai parent pointer + children slice — pre-order DFS rekursif dengan early-stop propagation, BFS pakai queue eksplisit.

## Source Code

@anchor:tree-traversal

## Big-O

| Operasi pada Tree berimbang (height = log n) | Kompleksitas |
|---|---|
| Search BST | O(log n) average, O(n) worst |
| Insert BST | O(log n) average, O(n) worst |
| Search BST balanced (AVL/RB) | O(log n) guaranteed |
| Traversal | O(n) |
| Compute height | O(n) |
| Lowest Common Ancestor | O(n) tanpa preprocessing, O(1) dengan Tarjan |

## Forward Reference (Fase 2)

Recursive CTE PostgreSQL untuk traversal `surat_references`:

```sql
WITH RECURSIVE thread AS (
    SELECT s.id, s.nomor_surat, s.perihal, 0 AS depth
    FROM surat s WHERE s.id = $1     -- root
    UNION ALL
    SELECT s.id, s.nomor_surat, s.perihal, t.depth + 1
    FROM surat s
    JOIN surat_references r ON r.from_surat_id = s.id
    JOIN thread t ON r.to_surat_id = t.id
    WHERE t.depth < 10               -- cycle prevention
)
SELECT * FROM thread ORDER BY depth;
```

Konsep yang akan di-anchor:

- DFS recursive via SQL
- Cycle detection (depth limit, atau tracking visited set)
- Lihat juga [DAG](./dag.md) untuk kasus saat tree assumption tidak holds.

## Eksperimen (General)

1. Implementasi BST dengan insert + in-order traversal, verify output sorted.

2. AVL self-balancing — gambar rotation kasus left-left, right-right, left-right, right-left. Implement insert dengan rotation.

3. Pertanyaan: kapan tree degenerate jadi linked list? (Hint: insert sorted sequence ke BST tanpa balancing). Bagaimana Red-Black tree mencegah ini?

## Referensi

- [CLRS Bab 12-13 — Binary Search Trees, Red-Black Trees](https://mitpress.mit.edu/9780262046305/introduction-to-algorithms/)
- [Visualizing Algorithms — VisuAlgo BST](https://visualgo.net/en/bst)
