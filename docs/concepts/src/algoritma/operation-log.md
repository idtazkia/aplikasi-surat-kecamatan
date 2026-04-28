---
id: operation-log-idempotency
courses: [algoritma, basis-data]
prereq: [event-sourcing-intro, idempotency]
related: [append-only-immutability, lww-merge]
fase: [4]
---

# Operation Log & Idempotency

## Teori

**Operation log** = catatan eksplisit setiap mutation (create/update/delete) sebagai event terstruktur. State akhir = replay semua operation dari awal. Pattern ini disebut **event sourcing**.

Konsep kunci:

1. **Event sebagai source of truth**, bukan state. State adalah turunan.
2. **Replay-able**: dengan log lengkap, state bisa di-rebuild kapan saja.
3. **Time-travel**: state pada waktu T = replay event sampai timestamp T.

**Idempotency** = property bahwa operasi yang dijalankan ulang menghasilkan efek sama dengan sekali jalan. Penting untuk:

- Network retry (at-least-once delivery)
- Sync mobile app yang reconnect setelah offline
- Distributed systems yang tidak bisa assume exactly-once

Cara mencapai idempotency:
- Operation ID yang unik (client-generated)
- Server: `INSERT ... ON CONFLICT (op_id) DO NOTHING`. Apply sekali, retry safe.

## Implementasi di App

Aplikasi-surat-kecamatan offline-first: staf bisa input/edit surat saat tidak ada koneksi. Saat online kembali, sync queue dikirim ke server.

Skema:

1. Setiap mutation di client membuat entry `operation_log` dengan `client_op_id` (UUIDv7 client-generated).
2. Saat sync, batch operations dikirim ke `POST /api/sync/operations`.
3. Server `INSERT ... ON CONFLICT (client_op_id) DO NOTHING`. Operation yang sudah pernah applied (mis. retry karena network error) tidak dobel-apply.

```sql
INSERT INTO operation_log (client_op_id, ...)
VALUES ($1, ...)
ON CONFLICT (client_op_id) DO NOTHING
RETURNING applied_at;
```

Kalau `RETURNING` kosong → ini adalah retry, server return existing result untuk operasi tersebut.

UUIDv7 sebagai operation ID juga memberi properti tambahan:
- **Time-ordered**: server bisa apply dalam urutan natural client (dengan toleransi clock skew kecil)
- **No coordination**: dua client offline tidak akan generate ID sama

## Source Code

@anchor:operation-log-idempotency

Schema definition. Application logic (sync handler) akan ditambah di Fase 4 dengan marker concept terpisah.

## Big-O

| Operasi | Kompleksitas |
|---|---|
| Insert (single op) | O(log n) — B-Tree index pada PK |
| Sync batch k ops | O(k × log n) |
| Replay seluruh log | O(n) |
| Idempotency check | O(log n) per op (lookup PK) |

Storage: O(n) monoton. Untuk skala kantor kecamatan dengan beberapa puluh ops/hari, log puluhan tahun masih sub-GB. Tidak perlu archival sampai sangat lama.

## Eksperimen

1. Buka `db/migrations/schema/0001_init.sql`, cari tabel `operation_log`. Perhatikan:
   - PK = `client_op_id` (bukan kolom `id` baru) — itu yang membuat idempotency by construction
   - `client_timestamp` ≠ `applied_at` (server time saat apply)
   - `field_changes JSONB` — sparse update, hanya field yang berubah

2. Pertanyaan: kalau `client_op_id` bukan UUIDv7 tapi sequential int per client, apa yang bisa salah? (Hint: dua client beda akan generate `1, 2, 3, ...` yang sama → collision saat sync).

3. Eksperimen di Fase 4 (akan ditambah): kirim batch yang sama ke `/api/sync/operations` dua kali. Verify count `operation_log` setelah operasi tetap N, bukan 2N.

## Referensi

- [Event Sourcing — Martin Fowler](https://martinfowler.com/eaaDev/EventSourcing.html)
- [Idempotency — Stripe Engineering](https://stripe.com/blog/idempotency)
- [Designing Data-Intensive Applications](https://dataintensive.net/) — Bab 11
