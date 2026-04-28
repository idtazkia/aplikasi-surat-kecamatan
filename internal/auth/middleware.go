package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type contextKey int

const claimsKey contextKey = 1

// Middleware mengambil bearer token dari Authorization header, validate,
// dan inject Claims ke request context. Handler downstream pakai
// ClaimsFromContext untuk akses.
//
// Tidak ada pengecekan permission di sini — middleware ini hanya verify
// authentication. Authorization (cek permission per endpoint) di handler
// atau via wrapper Require.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr := r.Header.Get("Authorization")
		if !strings.HasPrefix(hdr, "Bearer ") {
			writeAuthError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		token := strings.TrimPrefix(hdr, "Bearer ")

		claims, err := s.Verify(token)
		if err != nil {
			status := http.StatusUnauthorized
			msg := "invalid token"
			switch {
			case errors.Is(err, ErrTokenExpired):
				msg = "token expired"
			case errors.Is(err, ErrInvalidSignature):
				msg = "invalid signature"
			}
			writeAuthError(w, status, msg)
			return
		}

		// Hanya access token yang boleh dipakai untuk request handler.
		// Refresh token harus exchange dulu via /auth/refresh endpoint.
		if claims.Type != "access" {
			writeAuthError(w, http.StatusUnauthorized, "access token required")
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ClaimsFromContext mengambil Claims yang di-inject Middleware.
// Return zero Claims dan false kalau tidak ada (handler tanpa middleware).
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	c, ok := ctx.Value(claimsKey).(Claims)
	return c, ok
}

// HasRole mengecek apakah user di context punya role tertentu.
func HasRole(ctx context.Context, roleCode string) bool {
	c, ok := ClaimsFromContext(ctx)
	if !ok {
		return false
	}
	for _, r := range c.Roles {
		if r == roleCode {
			return true
		}
	}
	return false
}

func writeAuthError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
