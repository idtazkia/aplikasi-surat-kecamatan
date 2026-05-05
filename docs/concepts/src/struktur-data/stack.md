---
id: stack-lifo
courses: [struktur-data]
prereq: [linked-list-version-chain]
related: [queue-fifo-natural-order, graph-bfs-dfs]
fase: [0]
---

# Stack (LIFO)

> **Map ke materi kuliah**: [Skenario 2 — Undo Edit Metadata](https://github.com/idtazkia/materi-kuliah-2025-struktur-data/blob/main/case-study-aplikasi-surat-kecamatan.md). "Bu Camat menekan tombol Undo. Sistem harus mengembalikan ke state sebelumnya... Bandingkan dengan skenario 1 — kenapa urutan pengambilannya berbeda?"

## Teori

**Stack** = sequence dengan operasi:

- `push(x)` — tambah ke top
- `pop()` — ambil dari top, removed
- `peek()` / `top()` — lihat top tanpa remove

Properti **LIFO** (Last-In-First-Out): elemen yang masuk paling baru, keluar paling dulu.

Implementasi:

| Implementasi | push | pop | peek |
|---|---|---|---|
| Array dengan tail index | O(1) amortized | O(1) | O(1) |
| Singly linked list (head sebagai top) | O(1) | O(1) | O(1) |

Stack mendasari banyak konsep:

- **Call stack** — frame fungsi push saat call, pop saat return. Stack overflow = call depth lebih dalam dari kapasitas
- **Expression evaluation** — postfix/prefix evaluator
- **Backtracking algorithms** — DFS rekursif = call stack stack
- **Undo/redo** — push action ke undo stack, pop saat undo
- **Browser history** — back button = pop

## Implementasi di App

Reference implementation generic stack ada di package `internal/datastruct/stack`. Dipakai oleh `internal/datastruct/graph` untuk DFS traversal eksplisit (iterative, hindari risiko stack overflow di graph dalam dengan rekursi).

## Source Code

@anchor:stack-lifo

## Big-O

| Operasi | Kompleksitas |
|---|---|
| push | O(1) |
| pop | O(1) |
| peek | O(1) |
| search | O(n) — bukan operasi standar stack |

## Eksperimen (General)

1. Implementasi stack pakai linked list di JavaScript:
   ```ts
   class Stack<T> {
     private head: Node<T> | null = null;
     push(v: T) {
       this.head = { value: v, next: this.head };
     }
     pop(): T | undefined {
       if (!this.head) return undefined;
       const v = this.head.value;
       this.head = this.head.next;
       return v;
     }
     peek(): T | undefined {
       return this.head?.value;
     }
   }
   interface Node<T> { value: T; next: Node<T> | null; }
   ```

2. Klasik: implementasi queue dengan dua stack. Apa kompleksitas amortized vs worst case per dequeue?

3. Recursion vs iteration: tulis fungsi reverse linked list secara rekursif (implicit stack via call stack) dan secara iteratif (explicit stack pakai array). Bandingkan readability dan stack overflow risk untuk list panjang.

## Referensi

- [CLRS Bab 10.1 — Stacks and Queues](https://mitpress.mit.edu/9780262046305/introduction-to-algorithms/)
- [Stack Overflow — Wikipedia](https://en.wikipedia.org/wiki/Stack_overflow)
