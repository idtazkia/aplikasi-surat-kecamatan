// Package uuid7 implements UUID version 7 (RFC 9562) generator dan parser.
//
// UUIDv7 = 48-bit Unix milliseconds (big-endian) + 4-bit version + 12-bit
// random + 2-bit variant + 62-bit random. Properti penting: time-ordered.
// Dua UUID yang di-generate berurutan akan compare sebagai sorted-by-time,
// yang membuat insert ke B-Tree index relatif locality-friendly (tidak
// random-walk seperti UUIDv4).
//
// Untuk konteks aplikasi-surat-kecamatan, properti time-ordered ini yang
// menyelesaikan create-conflict di skenario offline-first: dua client yang
// generate UUID untuk surat baru tidak akan collide karena 48-bit timestamp
// + 74-bit random membuat probability collision negligible.
package uuid7

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// UUID adalah 16-byte representation.
type UUID [16]byte

// Nil adalah zero-value UUID (00000000-0000-0000-0000-000000000000).
var Nil UUID

// concept:uuid-v7-generation:start
// New generate UUIDv7 baru: 48-bit ms timestamp + version + random.
//
// Layout (RFC 9562):
//
//	 0                   1                   2                   3
//	 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                           unix_ts_ms                          |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|          unix_ts_ms           |  ver  |       rand_a          |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|var|                        rand_b                             |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                            rand_b                             |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
func New() (UUID, error) {
	var u UUID

	// 48-bit unix ms timestamp, big-endian (byte 0..5)
	ms := time.Now().UnixMilli()
	u[0] = byte(ms >> 40)
	u[1] = byte(ms >> 32)
	u[2] = byte(ms >> 24)
	u[3] = byte(ms >> 16)
	u[4] = byte(ms >> 8)
	u[5] = byte(ms)

	// Sisa 10 byte (byte 6..15) random
	if _, err := rand.Read(u[6:]); err != nil {
		return Nil, fmt.Errorf("uuid7: read random: %w", err)
	}

	// Set version (4 high bits di byte 6) ke 0111 = 7
	u[6] = (u[6] & 0x0f) | 0x70

	// Set variant (2 high bits di byte 8) ke 10 (RFC 4122 variant)
	u[8] = (u[8] & 0x3f) | 0x80

	return u, nil
}

// concept:uuid-v7-generation:end

// String returns canonical hyphenated representation.
func (u UUID) String() string {
	var buf [36]byte
	hex.Encode(buf[0:8], u[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], u[10:16])
	return string(buf[:])
}

// MarshalText implements encoding.TextMarshaler untuk JSON encoding.
func (u UUID) MarshalText() ([]byte, error) {
	b := []byte(u.String())
	return b, nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (u *UUID) UnmarshalText(data []byte) error {
	parsed, err := Parse(string(data))
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}

// IsZero reports whether u == Nil.
func (u UUID) IsZero() bool {
	return u == Nil
}

// Time mengembalikan timestamp yang di-encode di prefix 48-bit.
// Hanya valid untuk UUIDv7 (tidak validate version field).
func (u UUID) Time() time.Time {
	ms := int64(u[0])<<40 |
		int64(u[1])<<32 |
		int64(u[2])<<24 |
		int64(u[3])<<16 |
		int64(u[4])<<8 |
		int64(u[5])
	return time.UnixMilli(ms)
}

// ErrInvalidUUID dikembalikan saat parse string yang bukan format UUID valid.
var ErrInvalidUUID = errors.New("uuid7: invalid format")

// Parse string canonical hyphenated jadi UUID.
func Parse(s string) (UUID, error) {
	if len(s) != 36 {
		return Nil, fmt.Errorf("%w: length %d", ErrInvalidUUID, len(s))
	}
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return Nil, fmt.Errorf("%w: missing hyphens", ErrInvalidUUID)
	}

	var u UUID
	cleaned := s[0:8] + s[9:13] + s[14:18] + s[19:23] + s[24:36]
	if _, err := hex.Decode(u[:], []byte(cleaned)); err != nil {
		return Nil, fmt.Errorf("%w: %v", ErrInvalidUUID, err)
	}
	return u, nil
}

// MustParse panik kalau parse gagal. Hanya untuk constant testing/seed.
func MustParse(s string) UUID {
	u, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}
