package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
)

// fakeStore in-memory untuk test handler tanpa real DB.
type fakeStore struct {
	users   map[string]*store.UserForLogin
	pingErr error
}

func (f *fakeStore) GetUserForLogin(_ context.Context, username string) (*store.UserForLogin, error) {
	u, ok := f.users[username]
	if !ok {
		return nil, store.ErrNotFound
	}
	return u, nil
}

func (f *fakeStore) Ping(_ context.Context) error {
	return f.pingErr
}

func newDeps(t *testing.T, fs *fakeStore) Deps {
	t.Helper()
	a, err := auth.NewService([]byte("test-secret-32-bytes-long-padding"), time.Hour, 24*time.Hour)
	if err != nil {
		t.Fatalf("auth: %v", err)
	}
	return Deps{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Auth:   a,
		Store:  fs,
	}
}

func TestHealth_OK(t *testing.T) {
	d := newDeps(t, &fakeStore{users: map[string]*store.UserForLogin{}})
	srv := New(d)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status=%d body=%s", rec.Code, rec.Body)
	}
}

func TestHealth_DBDown(t *testing.T) {
	d := newDeps(t, &fakeStore{users: map[string]*store.UserForLogin{}, pingErr: errors.New("conn refused")})
	srv := New(d)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status=%d, want 503", rec.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	hash, err := auth.HashPassword("demo123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	fs := &fakeStore{users: map[string]*store.UserForLogin{
		"staf1": {ID: "user-1", Username: "staf1", PasswordHash: hash, IsActive: true, Roles: []string{"staf"}},
	}}
	d := newDeps(t, fs)
	srv := New(d)

	body := `{"username":"staf1","password":"demo123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
	}

	var resp loginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" {
		t.Errorf("missing tokens: %+v", resp)
	}

	// Verify access token works for /api/me
	req2 := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req2.Header.Set("Authorization", "Bearer "+resp.AccessToken)
	rec2 := httptest.NewRecorder()
	srv.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("/api/me status=%d", rec2.Code)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	hash, _ := auth.HashPassword("demo123")
	fs := &fakeStore{users: map[string]*store.UserForLogin{
		"staf1": {ID: "user-1", Username: "staf1", PasswordHash: hash, IsActive: true, Roles: []string{"staf"}},
	}}
	d := newDeps(t, fs)
	srv := New(d)

	body := `{"username":"staf1","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status=%d, want 401", rec.Code)
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	fs := &fakeStore{users: map[string]*store.UserForLogin{}}
	d := newDeps(t, fs)
	srv := New(d)

	body := `{"username":"ghost","password":"x"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status=%d, want 401", rec.Code)
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	hash, _ := auth.HashPassword("demo123")
	fs := &fakeStore{users: map[string]*store.UserForLogin{
		"sleeper": {ID: "user-2", Username: "sleeper", PasswordHash: hash, IsActive: false, Roles: []string{"staf"}},
	}}
	d := newDeps(t, fs)
	srv := New(d)

	body := `{"username":"sleeper","password":"demo123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status=%d, want 401", rec.Code)
	}
}

func TestRefresh(t *testing.T) {
	d := newDeps(t, &fakeStore{users: map[string]*store.UserForLogin{}})
	srv := New(d)

	// Issue refresh manually
	refresh, err := d.Auth.Issue(auth.Claims{Sub: "user-1", Type: "refresh", Roles: []string{"staf"}})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	body := `{"refresh_token":"` + refresh + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
	}
}

func TestRefresh_AccessTokenRejected(t *testing.T) {
	d := newDeps(t, &fakeStore{users: map[string]*store.UserForLogin{}})
	srv := New(d)

	access, _ := d.Auth.Issue(auth.Claims{Sub: "user-1", Type: "access"})
	body := `{"refresh_token":"` + access + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", strings.NewReader(body))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status=%d, want 401", rec.Code)
	}
}
