# Pengantar

Concept catalog ini adalah jembatan antara **konsep matkul** dan **source code aplikasi-surat-kecamatan**. Setiap halaman menjelaskan satu konsep dari salah satu matkul:

- **Struktur Data**
- **Algoritma**
- **Basis Data**

dengan tiga bagian utama:

1. **Teori** — penjelasan singkat konsep
2. **Implementasi di App** — narasi kenapa & bagaimana konsep tersebut diaplikasikan di kode
3. **Source Code** — link langsung ke baris kode di GitHub yang mengimplementasikan konsep

## Cara Pakai

- Link source code di setiap halaman membuka kode di GitHub di **commit spesifik** (bukan branch). Jadi link tidak akan rot saat code di-refactor.
- Klik link "📋 Source" untuk lihat implementasi langsung.
- Bagian "Eksperimen" berisi langkah praktis untuk explore konsep di aplikasi yang berjalan.
- Untuk navigasi: pakai sidebar kiri atau search bar di atas.

## Konteks Proyek

Aplikasi-surat-kecamatan adalah PkM (Pengabdian kepada Masyarakat) STMIK Tazkia untuk kantor kecamatan, dibangun dengan dual-mandate: aplikasi yang dipakai operasional **plus** medium edukasi mahasiswa.

Setiap pilihan teknis di proyek dievaluasi juga dari perspektif "apakah ini bahan ajar yang baik?". Itu sebabnya stack-nya:

- Go `net/http` (bukan framework) — supaya HTTP dasar tetap visible
- sqlc + raw SQL (bukan ORM) — supaya SQL tetap visible
- JWT manual (bukan library) — supaya HMAC + base64url terlihat
- UUIDv7 hand-rolled — supaya bit-manipulation terlihat

## Repo

[github.com/idtazkia/aplikasi-surat-kecamatan](https://github.com/idtazkia/aplikasi-surat-kecamatan)
