---
id: pdf-watermark
courses: [algoritma]
prereq: []
related: [multipart-streaming]
fase: [2]
---

# PDF Watermark — Content Stream Manipulation

> **Map ke materi kuliah**: Algoritma — manipulasi struktur file biner; sekilas kompresi, koordinat geometri, dan transformasi affine.

## Teori

PDF adalah format hybrid: header + cross-reference table + page tree berisi **content streams**. Setiap halaman punya stream berisi instruksi grafis (`q`, `cm`, `BT`, `Tf`, `Tj`, `ET`, `Q`, dll. — PostScript-like operators). Watermark dilakukan dengan menambah operator drawing ke content stream tiap halaman.

Pipeline watermark text:

1. **Parse PDF**: baca xref, page tree, content streams. Library bertanggungjawab dekompresi (`Filter /FlateDecode`).
2. **Compose watermark stream**: bangun fragment yang push state graphics, set font + warna + opacity, transform (rotate + translate), draw text, restore state.
3. **Inject ke setiap halaman**: append watermark stream ke content array. Atau alternatif: build form XObject sekali, reference dari setiap page (lebih efisien untuk multi-page).
4. **Serialize**: tulis ulang xref dengan offset baru, output PDF baru.

Operator dasar yang dipakai watermark:

| Operator | Fungsi |
|---|---|
| `q` ... `Q` | push/pop graphics state (jangan polusi state PDF asli) |
| `cm` | concat matrix — translate + rotate + scale |
| `BT` ... `ET` | begin/end text block |
| `Tf` | set font + size |
| `rg` / `RG` | set fill / stroke color |
| `gs` | extended graphics state (untuk opacity) |
| `Tj` | show text |

Matrix transform untuk rotate 45° + translate ke center:

```
cos(θ)  sin(θ)  0
-sin(θ) cos(θ)  0
tx      ty      1
```

CTM dalam PDF format: `a b c d e f cm` di mana `[a b 0; c d 0; e f 1]`.

## Trade-off Library Pilihan

| Library | License | Pure Go | Catatan |
|---|---|---|---|
| **pdfcpu** | Apache 2.0 | Ya | API ergonomis (`TextWatermark` + descriptor), CLI bawaan |
| unipdf | Komersial | Ya | API powerful tapi butuh license untuk produksi |
| ledongthuc/pdf | BSD | Ya | Read-only, tidak punya watermark API |
| Ghostscript binding | GPL | Tidak (cgo) | Dependency C |

Aplikasi pakai **pdfcpu** karena license permissive + sudah bawa parser yang robust + descriptor-based config (bisa expose ke admin nanti tanpa code change).

## Algoritmik Highlight

Watermark "diagonal at center" mengkombinasikan:

1. **Font metrics calculation**: ukur lebar text run untuk centering — `width = sum(advance_width(char_i))`. pdfcpu handle ini, tapi konsep ini sama dengan masalah classic compositor (text justification).

2. **Affine transformation chain**: T (translate ke center) ∘ R (rotate 45°) ∘ T' (translate ke origin). Order matters — non-commutative. Library biasanya kompose internal jadi single matrix.

3. **Alpha blending**: opacity 0.25 = src_alpha 0.25 + dest_alpha 0.75 → output. Implementasi via PDF `ExtGState` dengan `ca` / `CA` (current alpha).

## Implementasi di App

Watermark hanya di-apply saat:

- `mime_type = "application/pdf"`, AND
- `surat.access_level IN ('restricted', 'secret')`

Public surat di-stream langsung tanpa modifikasi (zero overhead). Format text:

```
Diunduh oleh {full_name} — {timestamp UTC}
```

Descriptor yang dipakai:

```
scale:0.9 abs, rot:45, opacity:0.25, col: 0.5 0.5 0.5, pos:c, points:24
```

Trade-off **decisional**:

- **Per-download watermark vs pre-watermark + cache**: per-download dipilih karena timestamp + user identity unik per request. Cost: O(n) parse PDF di setiap download. Mitigasi: limit file size 25MB, plus pdfcpu cukup cepat (~50ms untuk 10-page doc).
- **Stream watermark vs full re-emit**: pdfcpu re-emit full PDF (dengan optimasi obj reuse). Ini lebih simpel daripada incremental update yang butuh menjaga signature/encryption metadata.

## Source Code

@anchor:pdf-watermark

## Latihan

1. Modifikasi watermark agar muncul DUA baris (nama + timestamp di baris kedua). Eksperimen dengan multi-line text di PDF content stream.
2. Apa yang terjadi kalau PDF input sudah punya watermark? Test dengan download surat restricted dua kali (download → re-upload → download). Apakah teks jadi tumpang tindih?
3. Hitung overhead: ukur durasi watermark dengan PDF 1, 10, dan 100 halaman. Plot complexity vs page count.
