---
id: bcrypt-password-hashing
courses: [algoritma]
prereq: [hashing-cryptographic, salt-rainbow-table]
related: [jwt-hmac-sign-verify]
fase: [0]
---

# Bcrypt — Password Hashing

## Teori

**Hashing kriptografi** untuk password punya tiga properti yang harus dipenuhi:

1. **One-way**: dari hash tidak bisa recover plaintext (asumsi modulo brute-force)
2. **Salted**: dua user dengan password sama → hash berbeda. Mencegah pre-computed attack (rainbow table)
3. **Slow**: cukup cepat untuk login user (≈ 100-300ms), cukup lambat untuk membuat brute-force tidak ekonomis

Bcrypt (1999, Niels Provos & David Mazières) adalah **adaptive hash function** berbasis Blowfish dengan parameter **cost factor**:

- Iterasi = 2^cost
- cost=10 → 1024 iterasi (≈ 80ms di CPU 2010)
- cost=12 → 4096 iterasi (≈ 250ms di CPU modern)
- cost=14 → 16,384 iterasi (≈ 1s)

Karena hardware makin cepat, cost dinaikkan periodik (rule of thumb: 2-3 tahun sekali). Bcrypt mendukung "upgrade hash on next login" tanpa migrasi paksa.

Format hash:

```
$2a$12$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
│ │ │  │                            │
│ │ │  │                            └── 31-char hash + 22-char salt encoded
│ │ │  └─────── salt + hash payload
│ │ └──────── cost factor (= 12)
│ └──── version (2a, 2b, 2y)
└──── prefix `$`
```

## Implementasi di App

Aplikasi pakai bcrypt cost=12 untuk password user. Library: `golang.org/x/crypto/bcrypt` (stdlib-adjacent).

Kenapa bcrypt vs alternatif:

- **vs PBKDF2**: bcrypt memory-hard (sedikit, tapi enough). PBKDF2 hanya CPU-hard, lebih rentan GPU attack.
- **vs Argon2id**: Argon2id lebih modern (winner Password Hashing Competition 2015), tapi adoption lebih kecil di Go ecosystem. Bcrypt mature, well-supported, cukup untuk skala kantor kecamatan.
- **vs scrypt**: similar trade-off dengan Argon2id; bcrypt lebih familiar untuk mahasiswa kontributor.

Compare hash pakai `bcrypt.CompareHashAndPassword` yang **constant-time internal**, mitigate timing attack.

## Source Code

@anchor:bcrypt-password-hashing

## Big-O

| Operasi | Kompleksitas |
|---|---|
| Hash | O(2^cost) — exponential dalam cost |
| Verify | O(2^cost) |
| Brute force k password dengan dictionary n word | O(n × 2^cost) per password |

## Eksperimen

1. Edit `internal/auth/password.go`, ganti cost dari 12 ke 4. Run test, lihat berapa cepat. Ganti ke 14, lihat berapa lambat. Trade-off security vs UX.

2. Ukur waktu hash sendiri:
   ```go
   start := time.Now()
   _, _ = bcrypt.GenerateFromPassword([]byte("test"), 12)
   fmt.Println(time.Since(start))
   ```

3. Pertanyaan: kenapa kita pakai cost=12 untuk production tapi cost=10 untuk seed user demo? (Hint: lihat `db/migrations/demo-seed/0004_seed_users.sql` — salt + hash di-precompute dengan cost=10 untuk speed reset-demo).

## Referensi

- [A Future-Adaptable Password Scheme — Provos & Mazières (1999)](https://www.usenix.org/legacy/events/usenix99/provos/provos.pdf)
- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
