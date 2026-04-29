---
id: multipart-streaming
courses: [algoritma]
prereq: [queue-fifo-natural-order]
related: [streaming-sliding-window]
fase: [1]
---

# Multipart Streaming Upload

## Teori

**Multipart/form-data** (RFC 7578) = encoding HTTP request body untuk upload file. Body dipisah jadi *parts*, masing-masing dengan header dan content terpisah, di-delimit boundary string:

```
POST /api/upload HTTP/1.1
Content-Type: multipart/form-data; boundary=----abc123

------abc123
Content-Disposition: form-data; name="file"; filename="surat.pdf"
Content-Type: application/pdf

[binary PDF bytes...]
------abc123--
```

Dua approach parsing:

### A. Buffered (`r.ParseMultipartForm`)

Server load **semua** parts ke memory + temp file. Sederhana di kode (`r.MultipartForm.File["files"]`), tapi:

- **Memory bound**: kalau client upload 1GB file, server alokasi buffer 1GB (atau pakai temp file di disk untuk part > maxMemory threshold)
- **Latency penalty**: handler tidak bisa proses part #1 sampai semua parts selesai diterima
- **DoS surface**: client bisa kirim banyak part atau part besar, eat memory/disk

### B. Streaming (`r.MultipartReader`)

Server iterasi part demi part, langsung stream ke disk:

```go
mr, _ := r.MultipartReader()
for {
    part, err := mr.NextPart()
    if err == io.EOF { break }
    // part is io.Reader — stream to disk via io.Copy
}
```

Properti:
- **Memory bounded O(1)** per file regardless ukuran (default `io.Copy` pakai 32KB buffer)
- **Latency**: part #1 bisa diproses sebelum part #2 diterima
- **Backpressure**: kalau disk lambat, TCP receive window mengecil — flow control natural
- **Per-part size enforcement**: pakai `io.LimitReader(part, maxBytes+1)` lalu cek apakah copy melebihi limit

## Big-O

| Approach | Memory | Throughput |
|---|---|---|
| `ParseMultipartForm(maxMemory=10MB)` | O(N) untuk part > 10MB (di-spill ke disk temp) | Block sampai full body received |
| `MultipartReader` + io.Copy | O(buffer_size) — konstan, biasanya 32KB | Pipeline saat receive |

Untuk app upload PDF surat (≤ 25MB per file): keduanya OK functionally. Tapi streaming lebih scalable saat trafik upload tinggi atau file size limit dinaikkan.

## Implementasi di App

`POST /api/surat/{id}/attachments` di Fase 1 pakai streaming approach:

1. `r.MultipartReader()` — get reader untuk multipart body
2. Loop `mr.NextPart()` — tiap part = satu file
3. `streamPartToDisk` — sniff first 512 byte untuk MIME detection (don't trust client's Content-Type), tulis ke disk dengan UUIDv7 filename, enforce per-file size limit via `io.LimitReader`
4. Insert attachment row ke DB (referensi file_path = UUIDv7 disk filename)
5. Hapus file dari disk kalau DB insert gagal (compensating action)

MIME sniffing pakai `http.DetectContentType` (256-rule signature matching, RFC). Validasi terhadap whitelist — kalau MIME tidak match, hapus file + return 415.

## Source Code

@anchor:multipart-streaming

## Eksperimen

1. Test upload 30MB file dengan max 25MB → harus return 413 Payload Too Large.

2. Test upload file dengan extension PDF tapi binary content random → MIME sniff detect bukan PDF, return 415 Unsupported Media Type. Hapus file dari disk.

3. Bandingkan memory consumption pakai `pprof`:
   ```sh
   # Buffered version (uncomment alternate handler)
   curl -F file=@bigfile.pdf localhost:8080/upload-buffered

   # Streaming
   curl -F file=@bigfile.pdf localhost:8080/upload-streaming
   ```
   Profile pakai `go tool pprof http://localhost:6060/debug/pprof/heap`. Streaming heap stable; buffered spike hingga file size.

4. Pertanyaan diskusi: kapan buffered approach masih make sense? (Hint: kalau handler butuh validasi *seluruh* body dulu sebelum proses — misal validasi cryptographic signature di footer multipart).

## Referensi

- [RFC 7578 — multipart/form-data](https://datatracker.ietf.org/doc/html/rfc7578)
- [Go `net/http` Multipart docs](https://pkg.go.dev/mime/multipart)
- [Buffer Pool Pattern di Go HTTP](https://blog.golang.org/profiling-go-programs)
