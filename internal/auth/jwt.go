package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Claims adalah payload JWT yang di-issue aplikasi ini.
// Field name pakai short form sesuai konvensi RFC 7519.
type Claims struct {
	Sub   string   `json:"sub"`             // user_id (UUID string)
	Iat   int64    `json:"iat"`             // issued at (unix seconds)
	Exp   int64    `json:"exp"`             // expires at (unix seconds)
	Nbf   int64    `json:"nbf,omitempty"`   // not before (optional)
	Type  string   `json:"typ"`             // "access" atau "refresh"
	Roles []string `json:"roles,omitempty"` // role codes user
}

// Service menangani JWT issuance dan verification dengan shared HMAC secret.
type Service struct {
	secret           []byte
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
}

// NewService membuat auth service. accessTTL umumnya pendek (15-60 menit).
// refreshTTL panjang (7-30 hari). Untuk PWA offline, refreshTTL dipakai
// sebagai cache token saat staf bekerja tanpa koneksi.
func NewService(secret []byte, accessTTL, refreshTTL time.Duration) (*Service, error) {
	if len(secret) < 32 {
		return nil, errors.New("auth: secret minimal 32 byte")
	}
	if accessTTL <= 0 || refreshTTL <= 0 {
		return nil, errors.New("auth: TTL must be positive")
	}
	return &Service{
		secret:          secret,
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}, nil
}

// concept:jwt-hmac-sign-verify:start
// Issue membuat JWT signed dengan HS256.
//
// Format final: base64url(header) + "." + base64url(payload) + "." + base64url(signature)
//
// Signature = HMAC-SHA256(secret, header_enc + "." + payload_enc).
// Verify pakai constant-time comparison (hmac.Equal) supaya tidak bocor info
// via timing attack.
func (s *Service) Issue(claims Claims) (string, error) {
	if claims.Sub == "" {
		return "", errors.New("auth: missing sub claim")
	}
	if claims.Iat == 0 {
		claims.Iat = time.Now().Unix()
	}
	if claims.Exp == 0 {
		ttl := s.accessTokenTTL
		if claims.Type == "refresh" {
			ttl = s.refreshTokenTTL
		}
		claims.Exp = time.Now().Add(ttl).Unix()
	}
	if claims.Type == "" {
		claims.Type = "access"
	}

	header := []byte(`{"alg":"HS256","typ":"JWT"}`)
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("auth: marshal claims: %w", err)
	}

	headerEnc := base64.RawURLEncoding.EncodeToString(header)
	payloadEnc := base64.RawURLEncoding.EncodeToString(payload)
	signingInput := headerEnc + "." + payloadEnc

	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(signingInput))
	sigEnc := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + sigEnc, nil
}

// Verify validate signature, expiry, dan return parsed claims.
func (s *Service) Verify(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(signingInput))
	expectedSig := mac.Sum(nil)

	actualSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	if !hmac.Equal(expectedSig, actualSig) {
		return Claims{}, ErrInvalidSignature
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}

	now := time.Now().Unix()
	if claims.Exp > 0 && now > claims.Exp {
		return Claims{}, ErrTokenExpired
	}
	if claims.Nbf > 0 && now < claims.Nbf {
		return Claims{}, ErrTokenNotYetValid
	}

	return claims, nil
}

// concept:jwt-hmac-sign-verify:end

// IssueAccessAndRefresh adalah helper untuk login flow: issue pasangan token.
func (s *Service) IssueAccessAndRefresh(userID string, roles []string) (access, refresh string, err error) {
	access, err = s.Issue(Claims{Sub: userID, Type: "access", Roles: roles})
	if err != nil {
		return "", "", err
	}
	refresh, err = s.Issue(Claims{Sub: userID, Type: "refresh"})
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

// JWT-related errors. Caller bisa check dengan errors.Is().
var (
	ErrInvalidToken     = errors.New("auth: invalid token")
	ErrInvalidSignature = errors.New("auth: invalid signature")
	ErrTokenExpired     = errors.New("auth: token expired")
	ErrTokenNotYetValid = errors.New("auth: token not yet valid")
)
