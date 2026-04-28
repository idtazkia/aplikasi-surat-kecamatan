---
id: heap-priority-queue
courses: [struktur-data]
prereq: [tree-traversal, queue-fifo-natural-order]
related: [queue-fifo-natural-order, tree-traversal]
fase: [0, 4]
---

# Heap & Priority Queue

> **Map ke materi kuliah**: [Skenario 5 — Prioritas Surat Urgent](../../../materi-kuliah-2025-struktur-data/case-study-aplikasi-surat-kecamatan.md). "Bagaimana kamu menyimpan 30 surat tadi supaya Camat selalu bisa ambil yang paling urgent dengan cepat?... Kalau pakai array biasa lalu sort tiap kali ambil, apa kelemahannya kalau surat baru datang terus?"

## Teori

**Heap** = complete binary tree (semua level penuh kecuali mungkin yang terbawah, fill kiri-ke-kanan) dengan heap-property:

- **Min-heap**: parent ≤ children (root = nilai terkecil)
- **Max-heap**: parent ≥ children (root = nilai terbesar)

Bukan BST! Heap tidak guaranteed sorted left-to-right, hanya antara parent-child.

**Priority queue** = abstract data type dengan operasi:

- `insert(x, priority)` — tambah dengan prioritas
- `extract_min()` / `extract_max()` — ambil dengan priority terendah / tertinggi
- `peek()` — lihat top tanpa extract

Implementasi paling efisien untuk priority queue = binary heap.

## Big-O

| Operasi | Binary heap | Sorted array | Unsorted array |
|---|---|---|---|
| insert | O(log n) | O(n) | O(1) |
| extract_min | O(log n) | O(1) | O(n) |
| peek | O(1) | O(1) | O(n) |
| build_heap dari n elements | O(n) | O(n log n) | O(1) |

Binary heap menang untuk workload **mixed insert + extract** seperti task scheduler.

## Implementasi Compact (Array-Based)

Karena complete binary tree, heap di-store di array tanpa pointer:

- Index 0 = root
- Untuk node di index i:
  - Left child: 2i + 1
  - Right child: 2i + 2
  - Parent: (i - 1) / 2

Insert:
1. Append ke end array (O(1))
2. **Sift up** — swap dengan parent kalau melanggar heap-property (O(log n))

Extract:
1. Swap root dengan last element
2. Pop last (return as result)
3. **Sift down** — swap root dengan child terkecil, recursive ke bawah (O(log n))

## Aplikasi Heap

- **Priority queue** untuk task scheduler (OS), event simulator
- **Dijkstra's shortest path** — extract min-distance node (dengan Fibonacci heap → O(E + V log V))
- **A* search** — open list = priority queue
- **Heap sort** — O(n log n) sort in-place
- **Top-k problem** — pakai heap ukuran k

## Implementasi di App

Reference implementation binary min-heap generik di package `internal/datastruct/priorityq`. Compare function injectable — pakai `compare(b, a)` untuk max-heap, atau struct comparison untuk priority by field.

## Source Code

@anchor:heap-priority-queue

## Forward Use Case

### Fase 4 — Sync Queue dengan Priority

Saat ini desain sync queue: FIFO dengan retry exponential backoff. Tapi ada kasus user yang punya **prioritas mixed**:

- Operasi yang lebih lama menunggu (older client_op) — prioritas naik (FIFO)
- Operasi pada surat dengan sifat "segera" — prioritas tinggi
- Operasi retry (failed sebelumnya) — prioritas dipertahankan, tidak naik atau turun

Priority bisa di-encode lewat composite ordering: `(retry_count, sifat_priority, client_op_id)`. Tetap pakai B-Tree index PostgreSQL untuk persistent queue (lihat [Queue](./queue.md)).

In-memory di client (IndexedDB), priority queue bisa pakai library atau implementasi sendiri. Kalau jumlah pending op kecil (< 100), array sorted juga cukup — overhead heap belum worth it.

### Forward Use Case Lain

- **Reminder/deadline** (Fase 6): sortir surat overdue by deadline ascending = min-heap on deadline
- **Beban kerja staf** (Fase 6): top-N staf dengan beban tertinggi = bounded max-heap

## Eksperimen (General)

1. Implementasi binary heap dari scratch:
   ```ts
   class MinHeap<T> {
     private heap: T[] = [];
     constructor(private compare: (a: T, b: T) => number) {}

     insert(v: T) {
       this.heap.push(v);
       this.siftUp(this.heap.length - 1);
     }

     extractMin(): T | undefined {
       if (this.heap.length === 0) return undefined;
       const min = this.heap[0];
       const last = this.heap.pop()!;
       if (this.heap.length > 0) {
         this.heap[0] = last;
         this.siftDown(0);
       }
       return min;
     }

     private siftUp(i: number) {
       while (i > 0) {
         const parent = Math.floor((i - 1) / 2);
         if (this.compare(this.heap[i], this.heap[parent]) < 0) {
           [this.heap[i], this.heap[parent]] = [this.heap[parent], this.heap[i]];
           i = parent;
         } else break;
       }
     }

     private siftDown(i: number) {
       const n = this.heap.length;
       while (true) {
         const l = 2 * i + 1, r = 2 * i + 2;
         let smallest = i;
         if (l < n && this.compare(this.heap[l], this.heap[smallest]) < 0) smallest = l;
         if (r < n && this.compare(this.heap[r], this.heap[smallest]) < 0) smallest = r;
         if (smallest === i) break;
         [this.heap[i], this.heap[smallest]] = [this.heap[smallest], this.heap[i]];
         i = smallest;
       }
     }
   }
   ```

2. Heap sort: insert n element ke heap, extract n kali → O(n log n) sorted output. Bandingkan dengan QuickSort dan MergeSort untuk dataset acak vs sorted vs reverse-sorted.

3. Pertanyaan: kenapa Java `PriorityQueue` butuh `Comparator`? (Hint: heap tidak butuh assumption tentang ordering selain "ada cara compare").

## Referensi

- [CLRS Bab 6 — Heapsort](https://mitpress.mit.edu/9780262046305/introduction-to-algorithms/)
- [Pairing Heap, Fibonacci Heap — variasi lebih advanced](https://en.wikipedia.org/wiki/Pairing_heap)
- [Heap visualizer — VisuAlgo](https://visualgo.net/en/heap)
