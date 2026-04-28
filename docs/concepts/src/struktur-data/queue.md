---
id: queue-fifo-natural-order
courses: [struktur-data]
prereq: [linked-list-version-chain, btree-partial-index-soft-delete]
related: [stack, heap, operation-log-idempotency]
fase: [0, 4]
---

# Queue (FIFO)

> **Map ke materi kuliah**:
> - [Skenario 1 — Antrian Rekonsiliasi Duplikat](../../../materi-kuliah-2025-struktur-data/case-study-aplikasi-surat-kecamatan.md): "Yang mana yang dikerjakan duluan? Kenapa urutan itu adil?" — FIFO sebagai alasan keadilan.
> - [Skenario 5 — Prioritas Surat Urgent](../../../materi-kuliah-2025-struktur-data/case-study-aplikasi-surat-kecamatan.md): "Bandingkan dengan antrian di skenario 1 — apa bedanya FIFO biasa dengan antrian berdasarkan prioritas?" — lihat juga [Heap & Priority Queue](./heap.md).
>
> Bridge ke Java: [queue/Antrian.java](https://github.com/idtazkia/materi-kuliah-2025-struktur-data/blob/main/queue/Antrian.java).

## Teori

**Queue** = sequence dengan operasi:

- `enqueue(x)` — tambah ke belakang (tail)
- `dequeue()` — ambil dari depan (head), removed
- `peek()` — lihat head tanpa remove

Properti **FIFO** (First-In-First-Out): elemen yang masuk lebih dulu, keluar lebih dulu.

Implementasi standar:

| Implementasi | enqueue | dequeue | peek |
|---|---|---|---|
| Array dengan head/tail index (circular) | O(1) | O(1) | O(1) |
| Linked list dengan head/tail pointer | O(1) | O(1) | O(1) |
| Two stacks (klasik interview) | O(1) amortized | O(1) amortized | — |

Variasi:

- **Deque** (double-ended queue): enqueue/dequeue di kedua ujung
- **Priority queue**: dequeue ambil elemen dengan prioritas tertinggi (lihat [Heap](./heap.md))
- **Bounded queue**: ada kapasitas max, enqueue penuh = block atau reject

## Implementasi di App

Tabel `notifications` = persistent queue per user:

- **Enqueue** = `INSERT INTO notifications` saat event (mis. disposisi baru)
- **Dequeue** = `SELECT ... WHERE read_at IS NULL ORDER BY id LIMIT 1` lalu `UPDATE ... SET read_at = NOW()`
- **Peek (count unread)** = `SELECT COUNT(*) ... WHERE read_at IS NULL`

UUIDv7 sebagai PK = time-ordered → `ORDER BY id` = FIFO order. Tidak butuh kolom `sequence_number` terpisah.

Catatan: ini bukan strict FIFO seperti message broker. User bisa mark-read out-of-order, atau abaikan beberapa notifikasi. Untuk semantik strict FIFO (mis. job queue di Fase 4), pakai `FOR UPDATE SKIP LOCKED` agar dua worker tidak ambil item sama:

```sql
SELECT id, payload FROM job_queue
WHERE status = 'pending'
ORDER BY id
LIMIT 1
FOR UPDATE SKIP LOCKED;
```

## Source Code

@anchor:queue-fifo-natural-order

## Big-O di Konteks App

| Operasi | Kompleksitas | Implementasi |
|---|---|---|
| Enqueue | O(log n) | INSERT B-Tree index |
| Dequeue (mark-read) | O(log n) | UPDATE dengan PK lookup |
| Peek count unread | O(k), k = unread count | Partial index `WHERE read_at IS NULL` |
| Peek next unread | O(log n) | Index scan dengan LIMIT 1 |

## Eksperimen

1. Compare 2 implementasi queue di-memory di Vue:
   ```ts
   // A: Array dengan shift()
   const queueA = [];
   queueA.push(item);    // O(1)
   queueA.shift();       // O(n) — array index re-shift

   // B: Linked list (manual)
   class Node { constructor(v) { this.v = v; this.next = null; } }
   let head = null, tail = null;
   function enqueue(v) {
     const n = new Node(v);
     if (!tail) head = n; else tail.next = n;
     tail = n;
   }
   function dequeue() {
     if (!head) return null;
     const v = head.v; head = head.next;
     if (!head) tail = null;
     return v;
   }
   ```
   Ukur dengan 10,000 enqueue + 10,000 dequeue. Bandingkan ms.

2. Pertanyaan: kenapa `notifications` pakai UUID PK, bukan SERIAL/BIGSERIAL? (Hint: offline-first, no central counter, plus UUIDv7 properti time-ordered).

3. **Sync queue** di Fase 4 (forward-looking): client-side IndexedDB queue dengan retry + exponential backoff. Pikirkan: apa happens kalau queue disimpan tapi browser crashed? (Hint: persistent storage = recover-able. RAM-only queue would lose data).

## Forward Reference (Fase 4)

Sync queue klien akan berbasis IndexedDB. Setiap mutation user offline = enqueue ke `operation_log` store. Online lagi = dequeue batch + send ke server. Idempotency by `client_op_id` (UUIDv7) = retry-safe.

## Referensi

- [CLRS Bab 10.1 — Stacks and Queues](https://mitpress.mit.edu/9780262046305/introduction-to-algorithms/)
- [PostgreSQL FOR UPDATE SKIP LOCKED — multi-worker queue](https://www.2ndquadrant.com/en/blog/what-is-select-skip-locked-for-in-postgresql-9-5/)
