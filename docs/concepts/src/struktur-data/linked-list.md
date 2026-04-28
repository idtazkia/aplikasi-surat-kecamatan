---
id: linked-list-version-chain
courses: [struktur-data]
prereq: [pointer-reference]
related: [append-only-immutability, tree]
fase: [0, 2]
---

# Linked List

> **Map ke materi kuliah**: [Skenario 1 — Antrian Rekonsiliasi Duplikat](../../../materi-kuliah-2025-struktur-data/case-study-aplikasi-surat-kecamatan.md). Skenario eksplisit menanyakan: "Untuk menyimpan koleksi kasus antrian ini, kamu bisa pakai array atau linked list. Mana yang lebih cocok untuk skenario ini?"

## Teori

**Linked list** adalah sequence node yang dihubungkan via pointer. Beda dengan array: tidak butuh contiguous memory, insert/delete di tengah O(1) kalau pointer ke node sudah ada.

Variasi:

| Tipe | Pointer per node | Use case |
|---|---|---|
| **Singly linked** | `next` | Stack, simple sequence |
| **Doubly linked** | `prev` + `next` | Bisa traverse 2 arah, LRU cache |
| **Circular** | tail.next → head | Round-robin, queue ring |

Operasi dasar (singly linked):

| Operasi | Kompleksitas |
|---|---|
| Insert head | O(1) |
| Insert tail (dengan tail pointer) | O(1) |
| Insert tail (tanpa tail pointer) | O(n) |
| Delete by value | O(n) — cari dulu |
| Search by value | O(n) |
| Random access (index k) | O(k) |

Trade-off vs array:

- **Pro**: insert/delete tanpa shift, no pre-allocated capacity
- **Con**: cache locality buruk (node tersebar di memory), pointer overhead per node, tidak bisa random access

## Implementasi di App

Aplikasi pakai linked list pattern di **versioning lampiran surat** (`surat_attachments`):

- Setiap row punya kolom `replaced_by` — pointer (foreign key) ke row pengganti
- Versi terkini = node dengan `replaced_by IS NULL` (tail)
- Walk balik dari tail = traversal linked list versi-versi sebelumnya

Kenapa singly-linked, bukan separate `surat_attachment_versions` table:

1. **Schema kompak** — satu tabel, satu kolom tambahan
2. **Query "current version"** — `WHERE is_active = TRUE` (denormalisasi cache untuk hindari recursive walk setiap query)
3. **Query "history of attachment X"** — recursive CTE walk `replaced_by` chain

Trade-off: kalau perlu query "all attachments yang pernah replace siapa", denormalisasi `is_active` membutuhkan transaksional update saat replace.

## Source Code

@anchor:linked-list-version-chain

## Big-O di Konteks App

| Operasi | Kompleksitas | Catatan |
|---|---|---|
| Insert versi baru | O(log n) | B-Tree insert + UPDATE prev row |
| Get current version | O(log n) | Index pada `is_active = TRUE` |
| Walk full history | O(k × log n) | k = depth chain; recursive CTE |

## Eksperimen

1. Lihat schema `surat_attachments`. Tarik analogi: kolom `replaced_by` adalah pointer node berikutnya. Apa equivalent di in-memory linked list?

2. Tulis recursive CTE yang return semua versi dari `attachment_id` tertentu, urut dari versi terbaru ke terlama:
   ```sql
   WITH RECURSIVE version_chain AS (
       SELECT id, file_name, replaced_by, 1 AS depth FROM surat_attachments WHERE id = $1
       UNION ALL
       SELECT a.id, a.file_name, a.replaced_by, vc.depth + 1
       FROM surat_attachments a
       JOIN version_chain vc ON a.replaced_by = vc.id
   )
   SELECT * FROM version_chain ORDER BY depth;
   ```

3. Pertanyaan diskusi: kalau replace sering terjadi, kapan denormalisasi `is_active` berbahaya? (Hint: race condition saat 2 client replace concurrent — Fase 4 sync harus handle).

## Referensi

- [CLRS Bab 10.2 — Linked Lists](https://mitpress.mit.edu/9780262046305/introduction-to-algorithms/)
- [Linked List vs Array — Cache Locality](https://en.wikipedia.org/wiki/Locality_of_reference)
