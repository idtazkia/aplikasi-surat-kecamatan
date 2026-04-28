// Package auth menyediakan password hashing, JWT issuance/verification,
// dan HTTP middleware permission check.
//
// Filosofi: roll manual di atas stdlib + golang.org/x/crypto. Tidak pakai
// JWT framework pihak ketiga supaya konsep dasar (HMAC, base64url, signed
// claims) tetap visible untuk mahasiswa kontributor.
package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// concept:bcrypt-password-hashing:start
// HashPassword menghasilkan bcrypt hash untuk password plaintext.
// Cost factor menentukan jumlah iterasi (2^cost). Cost=12 ≈ 250ms di CPU
// modern, balance antara user-experience dan resistance to brute force.
// Cost dinaikkan secara berkala (2-3 tahun sekali) saat hardware makin cepat.
func HashPassword(plain string) (string, error) {
	const cost = 12
	if plain == "" {
		return "", errors.New("auth: empty password")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", fmt.Errorf("auth: bcrypt: %w", err)
	}
	return string(h), nil
}

// VerifyPassword membandingkan plain password dengan stored hash.
// Constant-time internally untuk mitigate timing attack.
// Return ErrInvalidCredentials kalau tidak match (bukan error generik) supaya
// caller bisa distinguish dari error sistem.
func VerifyPassword(plain, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("auth: bcrypt verify: %w", err)
	}
	return nil
}

// concept:bcrypt-password-hashing:end

// ErrInvalidCredentials dikembalikan saat password tidak match.
var ErrInvalidCredentials = errors.New("auth: invalid credentials")
