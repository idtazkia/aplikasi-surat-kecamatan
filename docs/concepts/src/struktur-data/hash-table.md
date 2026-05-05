---
id: hash-table-map
courses: [struktur-data]
prereq: [hash-function, collision-resolution]
related: [set, btree-partial-index-soft-delete]
fase: [0]
---

# Hash Table

> **Map ke materi kuliah**: [Skenario 6 Bagian B — Deteksi Duplikat Instan](https://github.com/idtazkia/materi-kuliah-2025-struktur-data/blob/main/case-study-aplikasi-surat-kecamatan.md). "Pengecekan ini terjadi setiap kali staf klik Simpan — tidak boleh lambat meski database sudah berisi 50.000 surat... Apa kunci yang kamu pakai untuk lookup?"
>
> Untuk autocomplete prefix di Skenario 6 Bagian A, lihat [Trie](./trie.md). Hash table untuk exact match, trie untuk prefix match.

## Teori

**Hash table** = struktur key-value yang lookup, insert, dan delete-nya O(1) average dengan trade-off space O(n).

Komponen:

1. **Hash function** `h(key) → bucket_index`. Properti baik: uniform distribution, deterministic, fast.
2. **Bucket array** ukuran m. Item dengan `h(key) mod m` sama → sama bucket.
3. **Collision resolution**:
   - **Chaining**: tiap bucket = linked list. Insert append ke list.
   - **Open addressing**: cari bucket kosong (linear probe, quadratic probe, double hashing).

Load factor α = n / m:

- α tinggi → banyak collision → operasi degrade ke O(n) worst case
- α rendah → memory waste
- Praktis: resize (grow/shrink) ketika α melewati threshold (mis. 0.75 untuk Java HashMap, 6.5 average chain length untuk Go map)

## Big-O

| Operasi | Average | Worst |
|---|---|---|
| Lookup | O(1) | O(n) — semua key collide |
| Insert | O(1) amortized | O(n) saat resize |
| Delete | O(1) | O(n) |

Worst case O(n) jarang terjadi di praktik karena hash function yang baik. Tapi ada serangan **hash flooding**: attacker pilih key yang collide → degrade ke O(n) per request → DoS. Mitigasi: random seed di hash (Go map, Python dict 3.3+ default).

## Implementasi di App

JavaScript `Map` (dipakai di Vue store) = hash table dengan engine-specific implementation. V8 pakai variation dari open addressing dengan dynamic resize.

Di `eduPanel` store, lookup `concept_id → ConceptLink` pakai Map untuk O(1) per lookup. Tanpa map, render student drawer dengan multiple concept_id butuh `n × m` linear search:

- n = jumlah concept_id di payload
- m = total concept di catalog (~10-50)

Dengan map: build sekali O(m), tiap lookup O(1) → render total O(n + m). Untuk catalog yang tumbuh, ini bedanya jadi signifikan.

## Source Code

@anchor:hash-table-map

## Eksperimen

1. Bandingkan dengan implementasi linear search (tanpa map):
   ```ts
   // Linear: O(m) per lookup, O(n*m) total
   function findByID(links, id) {
     return links.find(l => l.id === id);
   }
   ```
   Tulis benchmark untuk catalog 1000 concept dengan 100 lookup. Ukur ms.

2. Pertanyaan: kenapa Object `{}` di JS bukan pure hash table? (Hint: Map punya guarantee insertion order, Object key terbatas string/Symbol, prototype pollution risk).

3. Eksperimen collision: tulis hash function `h(s) = s.length` (sengaja buruk). Insert 1000 string ke 100 bucket. Hitung max chain length. Bandingkan dengan FNV-1a atau SipHash.

## Aplikasi Lain di App

- **PostgreSQL hash index** (jarang dipakai vs B-Tree, hanya untuk equality lookup)
- **JWT claims**: object dengan key-value di-serialize jadi JSON
- **Pinia state**: reactive object pada dasarnya hash table dengan getter/setter
- **localStorage**: web storage = key-value store browser

## Referensi

- [CLRS Bab 11 — Hash Tables](https://mitpress.mit.edu/9780262046305/introduction-to-algorithms/)
- [Go map internals — Dave Cheney](https://dave.cheney.net/high-performance-go-workshop/sydney.html)
- [Why hash flooding matters](https://www.youtube.com/watch?v=R2Cq3CLI6H8)
