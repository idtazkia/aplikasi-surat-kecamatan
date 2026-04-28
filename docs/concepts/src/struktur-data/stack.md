---
id: stack
courses: [struktur-data]
pending: true
prereq: [linked-list-version-chain]
related: [queue-fifo-natural-order]
fase: [N/A]
---

# Stack (LIFO)

> **Status**: konsep fundamental — implementasi domain-specific belum ada di Fase 0. Halaman ini intro pengantar.
>
> **Map ke materi kuliah**: [Skenario 2 — Undo Edit Metadata](../../../materi-kuliah-2025-struktur-data/case-study-aplikasi-surat-kecamatan.md). "Bu Camat menekan tombol Undo. Sistem harus mengembalikan ke state sebelumnya... Bandingkan dengan skenario 1 — kenapa urutan pengambilannya berbeda?"
>
> Bridge ke Java: [stack/Tumpukan.java](https://github.com/idtazkia/materi-kuliah-2025-struktur-data/blob/main/stack/Tumpukan.java).

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

## Implementasi di App (Pending)

Aplikasi-surat-kecamatan **belum** punya struktur stack domain-specific di Fase 0. Stack kemungkinan muncul di:

- **Fase 5 — Antrian rekonsiliasi**: kandidat. Saat user resolve grup duplikat, history "decision yang sudah dibuat" untuk undo bisa pakai stack. Tapi praktiknya akan di-persist sebagai append-only log, bukan stack.
- **Recursive CTE traversal `surat_references`** (Fase 2): query plan PostgreSQL pakai stack internal untuk DFS — tapi ini implicit, tidak exposed ke kode aplikasi.
- **Vue component lifecycle**: framework internal pakai stack untuk render order — di luar kontrol aplikasi.

Untuk Fase 0, stack tetap penting sebagai konsep ajar — call stack dan recursion deeply terkait.

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
