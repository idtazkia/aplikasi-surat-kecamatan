---
id: jwt-hmac-sign-verify
courses: [algoritma]
prereq: [hmac-mac, base64-encoding, json]
related: [bcrypt-password-hashing]
fase: [0]
---

# JWT dengan HMAC-SHA256

## Teori

**JWT (JSON Web Token, RFC 7519)** adalah self-contained token bertanda tangan untuk authentication & authorization. Format:

```
base64url(header) . base64url(payload) . base64url(signature)
```

Tiga bagian dipisah titik. Contoh:

```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ1c2VyLTEiLCJleHAiOjE3MzAwMDAwMDB9.abc...
```

### HMAC-SHA256 (HS256)

HMAC = Hash-based Message Authentication Code. Dengan SHA-256:

```
HMAC(key, msg) = SHA256( (key XOR opad) || SHA256( (key XOR ipad) || msg ) )
```

di mana `opad` dan `ipad` adalah konstanta padding 64-byte. Properti:

- **Authenticated**: tanpa key, tidak bisa generate signature valid
- **Deterministic**: same key + same msg → same signature
- **Constant-time verify** wajib (pakai `hmac.Equal`, bukan `bytes.Equal`) untuk mitigate timing attack

### Kenapa HS256 (symmetric) vs RS256 (asymmetric)

| | HS256 | RS256 |
|---|---|---|
| Algorithm | HMAC-SHA256 | RSA + SHA256 |
| Key | Symmetric (1 secret) | Asymmetric (private/public) |
| Use case | Single service | Multi-service yang share token verifier |
| Performance | Cepat | Lebih lambat (orders of magnitude) |
| Complexity | Rendah | Lebih tinggi |

Aplikasi-surat-kecamatan single service → HS256 cukup.

## Implementasi di App

JWT di-roll **manual** di atas stdlib (`crypto/hmac`, `crypto/sha256`, `encoding/base64`, `encoding/json`) — tidak pakai library JWT pihak ketiga.

Alasan:

1. **Mandat edukasi**: spec JWT tidak besar. Mahasiswa lihat HMAC + base64url + signed claims tanpa abstraction layer.
2. **Dependency footprint kecil**: stdlib saja.
3. **Audit lebih mudah**: ~80 baris kode, vs library yang ribuan baris.

Claims yang dipakai:

- `sub` — user ID (UUID string)
- `iat` — issued at (unix seconds)
- `exp` — expires at (15 menit untuk access, 7 hari untuk refresh)
- `typ` — "access" atau "refresh"
- `roles` — list role codes user

Refresh token tidak boleh dipakai langsung untuk request handler — middleware reject. Refresh harus di-exchange via `POST /api/auth/refresh` untuk dapat access token baru.

## Source Code

@anchor:jwt-hmac-sign-verify

@anchor:bcrypt-password-hashing

## Big-O

| Operasi | Kompleksitas |
|---|---|
| Issue (sign) | O(\|payload\|) |
| Verify (HMAC + JSON parse) | O(\|payload\|) |
| Brute-force secret 32 byte | O(2^256) — infeasible |

## Eksperimen

1. Decode JWT dengan tangan: ambil access token dari response `/api/auth/login`, base64url-decode masing-masing bagian, lihat header dan payload.

2. Tamper dengan payload (mis. ganti `roles` ke `["admin"]`), encode ulang, kirim ke `/api/me` dengan signature lama. Server reject — kenapa? (Signature tidak match karena msg = `header.payload` berubah.)

3. Coba issue token dengan secret salah, verify dengan secret asli. Lihat error `ErrInvalidSignature`. Sekarang coba `bytes.Equal` vs `hmac.Equal` di `Verify` — diskusi: kapan timing attack relevan?

## Referensi

- [RFC 7519 — JWT](https://datatracker.ietf.org/doc/html/rfc7519)
- [RFC 7518 — JSON Web Algorithms (HS256, RS256)](https://datatracker.ietf.org/doc/html/rfc7518)
- [HMAC: Keyed-Hashing for Message Authentication — RFC 2104](https://datatracker.ietf.org/doc/html/rfc2104)
