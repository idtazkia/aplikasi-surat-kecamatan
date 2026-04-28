package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testSecret = "this-is-a-test-secret-32-bytes-long!"

func newTestService(t *testing.T) *Service {
	t.Helper()
	s, err := NewService([]byte(testSecret), 1*time.Hour, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return s
}

func TestNewService_ShortSecret(t *testing.T) {
	_, err := NewService([]byte("short"), time.Hour, 24*time.Hour)
	if err == nil {
		t.Fatal("expected error for short secret")
	}
}

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := HashPassword("mypassword123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Errorf("expected bcrypt hash prefix, got %q", hash)
	}
	if err := VerifyPassword("mypassword123", hash); err != nil {
		t.Errorf("VerifyPassword (correct) failed: %v", err)
	}
	if err := VerifyPassword("wrongpass", hash); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("VerifyPassword (wrong) error = %v, want ErrInvalidCredentials", err)
	}
}

func TestHashPassword_Empty(t *testing.T) {
	_, err := HashPassword("")
	if err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestJWT_IssueAndVerify(t *testing.T) {
	s := newTestService(t)

	token, err := s.Issue(Claims{Sub: "user-123", Roles: []string{"staf"}})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	claims, err := s.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.Sub != "user-123" {
		t.Errorf("Sub = %q, want user-123", claims.Sub)
	}
	if claims.Type != "access" {
		t.Errorf("Type = %q, want access", claims.Type)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "staf" {
		t.Errorf("Roles = %v, want [staf]", claims.Roles)
	}
}

func TestJWT_TamperedSignature(t *testing.T) {
	s := newTestService(t)
	token, _ := s.Issue(Claims{Sub: "user-1"})

	// Flip last char of signature
	tampered := token[:len(token)-1] + "X"
	if tampered == token {
		tampered = token[:len(token)-1] + "Y"
	}

	_, err := s.Verify(tampered)
	if !errors.Is(err, ErrInvalidSignature) && !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected signature/token error, got %v", err)
	}
}

func TestJWT_Expired(t *testing.T) {
	s := newTestService(t)
	token, _ := s.Issue(Claims{
		Sub: "user-1",
		Iat: time.Now().Add(-2 * time.Hour).Unix(),
		Exp: time.Now().Add(-1 * time.Hour).Unix(),
	})
	_, err := s.Verify(token)
	if !errors.Is(err, ErrTokenExpired) {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestJWT_DifferentSecretFailsVerify(t *testing.T) {
	s1, _ := NewService([]byte("first-secret-32-bytes-long-okayy!"), time.Hour, 24*time.Hour)
	s2, _ := NewService([]byte("second-secret-32-bytes-long-pad!!"), time.Hour, 24*time.Hour)

	token, _ := s1.Issue(Claims{Sub: "user-1"})
	_, err := s2.Verify(token)
	if !errors.Is(err, ErrInvalidSignature) {
		t.Errorf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestIssueAccessAndRefresh(t *testing.T) {
	s := newTestService(t)
	access, refresh, err := s.IssueAccessAndRefresh("user-1", []string{"staf"})
	if err != nil {
		t.Fatalf("IssueAccessAndRefresh: %v", err)
	}

	ac, _ := s.Verify(access)
	rc, _ := s.Verify(refresh)
	if ac.Type != "access" || rc.Type != "refresh" {
		t.Errorf("token types: access=%q refresh=%q", ac.Type, rc.Type)
	}
}

func TestMiddleware_NoToken(t *testing.T) {
	s := newTestService(t)
	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestMiddleware_ValidToken_InjectsClaims(t *testing.T) {
	s := newTestService(t)
	token, _ := s.Issue(Claims{Sub: "user-1", Roles: []string{"camat"}})

	called := false
	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		c, ok := ClaimsFromContext(r.Context())
		if !ok {
			t.Error("ClaimsFromContext returned ok=false")
		}
		if c.Sub != "user-1" {
			t.Errorf("Sub = %q", c.Sub)
		}
		if !HasRole(r.Context(), "camat") {
			t.Error("HasRole(camat) should be true")
		}
		if HasRole(r.Context(), "admin") {
			t.Error("HasRole(admin) should be false")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler was not called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestMiddleware_RefreshTokenRejected(t *testing.T) {
	s := newTestService(t)
	refresh, _ := s.Issue(Claims{Sub: "user-1", Type: "refresh"})

	handler := s.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for refresh token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+refresh)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

// Sanity check: HasRole tanpa context tidak panic
func TestHasRole_NoContext(t *testing.T) {
	if HasRole(context.Background(), "anything") {
		t.Error("HasRole on bare context should be false")
	}
}
