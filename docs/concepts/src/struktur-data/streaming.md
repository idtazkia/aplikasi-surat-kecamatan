---
id: streaming-sliding-window
courses: [struktur-data]
pending: true
prereq: [queue-fifo-natural-order, hash-table-map, heap]
related: [queue-fifo-natural-order, hash-table-map, heap]
fase: [6, 7]
---

# Sliding Window & Probabilistic Data Structures

> **Status**: implementasi dashboard real-time di Fase 6 (statistik per periode) atau Fase 7 (jika butuh probabilistic structure). Halaman ini intro pengantar.
>
> **Map ke materi kuliah**: [Skenario 7 — Dashboard Monitoring Surat Real-Time](https://github.com/idtazkia/materi-kuliah-2025-struktur-data/blob/main/case-study-aplikasi-surat-kecamatan.md). "Jumlah surat masuk dalam 1 jam terakhir... 24 jam terakhir... Top 5 pengirim dalam 7 hari... Update cepat, memori sedikit, estimasi cukup."

## Konteks: Streaming Computation

Workload **streaming** = data datang terus-menerus (event stream), perlu jawaban yang up-to-date tanpa scan ulang seluruh history. Contoh:

- Jumlah surat dalam 1 jam terakhir (windowed count)
- Top 5 pengirim dalam 7 hari (heavy hitters)
- Apakah pengirim "Kemendagri" sudah pernah kirim hari ini (membership query)
- Approximate distinct senders bulan ini (cardinality estimation)

Naive approach: SQL aggregate `WHERE created_at > NOW() - INTERVAL '1 hour'`. Issue di scale:

- **Repeated scan** — query tiap N detik = O(n) per query, scan ulang seluruh window
- **DB load** — dashboard query frekuen ganggu transaksional
- **Latency** — index scan tetap punya overhead

Untuk dashboard yang **toleran approximate result** dengan **memory bounded**, ada family struktur data probabilistic.

## Sliding Window

**Window** = sub-sequence dari stream dengan range waktu (1 jam, 24 jam, dst.). Implementasi:

### 1. Linked list / Deque dari (timestamp, count)

Tiap event tambah ke tail. Saat query, evict head yang lebih lama dari window:

```
events = [(t1, 1), (t2, 1), ...]
while events.front.timestamp < now - window_size:
    events.pop_front()
return sum of counts
```

Memory: O(events_in_window). Untuk window 1 jam dengan 200 surat/hari = ~10 events. Manageable.

### 2. Bucketed window

Bagi window jadi N bucket (mis. 60 bucket × 1 menit = 1 jam). Tiap bucket simpan count saja, bukan event individual:

```
buckets = [count_minute_-59, count_minute_-58, ..., count_minute_0]
on event: increment buckets[current_minute]
on query: sum(buckets) within window
periodic: shift buckets, drop oldest, add new empty
```

Memory: O(N) konstan. Trade-off: granularity = ukuran bucket (precision loss kecil).

## Heavy Hitters (Top-K)

Skenario 7: "Top 5 pengirim paling sering dalam 7 hari".

### Naive

`SELECT instansi_id, COUNT(*) FROM surat WHERE created_at > NOW() - INTERVAL '7 days' GROUP BY instansi_id ORDER BY 2 DESC LIMIT 5;`

Issue: dengan 50,000 surat per minggu dan ~500 instansi, query masih cepat (< 100ms dengan index). Tapi kalau data lebih besar atau frekuensi query tinggi, butuh inkremental.

### Min-heap of size K (exact, bounded memory)

Maintain min-heap of size K (= 5). Saat process event:
- Hitung total count per pengirim (di hash table)
- Insert ke heap kalau count > min(heap), evict min

Issue: butuh **complete count per pengirim** dulu. Kalau 500 pengirim, masih OK (500 entries di hash table).

### Count-Min Sketch (approximate, sublinear memory)

Untuk dataset sangat besar (jutaan unique pengirim), exact count tidak feasible. Count-Min Sketch = probabilistic structure:

- 2D array `width × depth`, di-init 0
- `d` hash functions `h_1, ..., h_d`
- **Increment(x)**: untuk tiap row i: `array[i][h_i(x) mod width] += 1`
- **Estimate(x)**: return `min(array[i][h_i(x) mod width])` dari semua row

Properti:
- **No false negative** untuk count (estimate ≥ actual)
- **Bounded over-estimation** dengan probability — tunable via width × depth
- Memory: O(width × depth), independen dari unique items

Untuk Skenario 7 dengan 500 pengirim: overkill. Berguna kalau pengirim bisa "anyone" (mis. tracking unique URL hits di web crawler).

## Membership Test

"Apakah pengirim X sudah pernah kirim hari ini?"

### Bloom Filter (probabilistic membership)

- Bit array ukuran m, di-init 0
- k hash functions `h_1, ..., h_k`
- **Add(x)**: set `bits[h_i(x) mod m] = 1` untuk semua i
- **Contains(x)**: cek apakah semua `bits[h_i(x) mod m]` = 1

Properti:
- **No false negative** — kalau Bloom bilang "tidak ada", pasti tidak ada
- **False positive** — bisa bilang "ada" padahal belum pernah
- Memory: O(m) bit, biasanya jauh lebih kecil dari hash set
- Tidak bisa delete (kecuali Counting Bloom Filter variant)

Use case: cache layer ("kalau Bloom bilang tidak ada, skip query DB; kalau bilang ada, query DB untuk konfirmasi"). Hemat trip DB untuk negative case.

## Cardinality Estimation

"Berapa unique sender bulan ini?"

### HyperLogLog

Algoritma sublinear untuk estimate count distinct. Memory ~12KB untuk error 1%, untuk dataset arbitrary size (jutaan, miliaran).

PostgreSQL: extension `postgresql-hll` atau `tdigest`.

## Implementasi di App

**Fase 6** dashboard akan implement:

- **Statistik per periode** — query langsung ke `surat` table dengan partial index pada `tanggal_terima` + BRIN index untuk timeseries (`tanggal_terima`). Untuk skala kantor kecamatan (~50–200 surat/hari, history beberapa tahun), exact count tetap cepat. **Probabilistic structure tidak diperlukan** di scope ini.

- **Real-time update** kalau di-perlukan: pakai materialized view dengan refresh periodik (10 menit), atau in-app cache dengan invalidation pada surat baru.

**Fase 7** kalau scale jauh lebih besar atau dashboard butuh sub-second update tanpa beban DB:
- Sliding window in-app (bucketed counter, tracked in memory dengan persist ke Redis/PostgreSQL setiap N detik)
- Bloom filter cek "sudah pernah dilihat" untuk dedup notifikasi

## Bridge ke Materi Kuliah

Skenario 7 tidak ada Java code di matkul (advanced topic). Untuk eksplor lebih lanjut, mahasiswa bisa baca:

- [Apache DataSketches](https://datasketches.apache.org/) — implementasi production-grade dari Bloom filter, HLL, Count-Min Sketch
- [Probabilistic Data Structures and Algorithms (Andrii Gakhov)](https://pdsa.gakhov.com/) — buku free berbahasa Inggris
- Eksperimen dengan PostgreSQL `pg_stat_statements` + `tdigest` extension

## Eksperimen

1. **Bucketed sliding window** in TypeScript:
   ```ts
   class SlidingWindowCounter {
     private buckets: number[];
     private current = 0;
     constructor(private windowSize: number, private bucketCount: number) {
       this.buckets = new Array(bucketCount).fill(0);
       setInterval(() => this.tick(), windowSize / bucketCount);
     }
     increment() { this.buckets[this.current]++; }
     count(): number { return this.buckets.reduce((a, b) => a + b, 0); }
     private tick() {
       this.current = (this.current + 1) % this.bucketCount;
       this.buckets[this.current] = 0;
     }
   }
   ```
   Test: simulate 200 events/hour, query tiap detik. Memory + accuracy benchmark.

2. **Bloom filter** sederhana dengan 3 hash function. Insert 1000 string, test 1000 lookup yang separuh exist + separuh fictional. Hitung false positive rate, compare dengan teoretis `(1 - e^{-kn/m})^k`.

3. Pertanyaan diskusi: kapan trade-off "approximate but cheap" worth it untuk dashboard pemerintahan? (Hint: surat masuk = 50–200/hari, berbeda dengan e-commerce 1M event/menit. Apakah probabilistic structure relevan untuk skala ini?)

## Referensi

- [Probabilistic Data Structures and Algorithms — Andrii Gakhov](https://pdsa.gakhov.com/) (free PDF)
- [Designing Data-Intensive Applications — Bab 11 (Stream Processing)](https://dataintensive.net/)
- [Apache DataSketches](https://datasketches.apache.org/) — production library
- [Count-Min Sketch original paper — Cormode & Muthukrishnan 2005](http://dimacs.rutgers.edu/~graham/pubs/papers/cmsoft.pdf)
