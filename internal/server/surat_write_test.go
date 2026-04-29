package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
)

// fakeSuratWriteStore extends fakeSuratStore dengan write methods.
type fakeSuratWriteStore struct {
	*fakeSuratStore
	created  *store.CreateSuratInput
	updated  map[string]store.UpdateSuratInput
	deleted  map[string]string // id -> actor
	restored map[string]string

	// Untuk simulate not-found / conflict
	conflictOnCreate bool
	notFoundOnUpdate bool
}

func (f *fakeSuratWriteStore) CreateSurat(_ context.Context, in store.CreateSuratInput) error {
	if f.conflictOnCreate {
		return store.ErrConflict
	}
	f.created = &in
	return nil
}

func (f *fakeSuratWriteStore) UpdateSurat(_ context.Context, id string, in store.UpdateSuratInput) error {
	if f.notFoundOnUpdate {
		return store.ErrNotFound
	}
	if f.updated == nil {
		f.updated = map[string]store.UpdateSuratInput{}
	}
	f.updated[id] = in
	return nil
}

func (f *fakeSuratWriteStore) SoftDeleteSurat(_ context.Context, id, actor string) error {
	if f.deleted == nil {
		f.deleted = map[string]string{}
	}
	f.deleted[id] = actor
	return nil
}

func (f *fakeSuratWriteStore) RestoreSurat(_ context.Context, id, actor string) error {
	if f.restored == nil {
		f.restored = map[string]string{}
	}
	f.restored[id] = actor
	return nil
}

func newWriteDeps(t *testing.T, fs *fakeSuratWriteStore) Deps {
	t.Helper()
	d := newSuratDeps(t, fs.fakeSuratStore)
	d.SuratStore = fs
	return d
}

func TestSuratCreate_Masuk_Success(t *testing.T) {
	fs := &fakeSuratWriteStore{fakeSuratStore: &fakeSuratStore{}}
	d := newWriteDeps(t, fs)
	srv := New(d)

	body := `{"jenis":"masuk","nomor_surat":"X/1/I/2026","perihal":"Test","tanggal_surat":"2026-01-01","tanggal_terima":"2026-01-02","instansi_id":"00000000-0000-0000-0005-000000000001","access_level":"public"}`
	req := httptest.NewRequest(http.MethodPost, "/api/surat", strings.NewReader(body))
	req.Header.Set("Authorization", bearerHeader(t, d, "user-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if fs.created == nil {
		t.Fatal("create not called")
	}
	if fs.created.Jenis != "masuk" {
		t.Errorf("jenis = %q", fs.created.Jenis)
	}
	if fs.created.TanggalTerima == nil {
		t.Error("tanggal_terima nil")
	}
	if fs.created.CreatedBy != "user-1" {
		t.Errorf("created_by = %q", fs.created.CreatedBy)
	}
}

func TestSuratCreate_Keluar_NoTanggalTerima(t *testing.T) {
	fs := &fakeSuratWriteStore{fakeSuratStore: &fakeSuratStore{}}
	d := newWriteDeps(t, fs)
	srv := New(d)

	body := `{"jenis":"keluar","nomor_surat":"OUT/1/I/2026","perihal":"Test","tanggal_surat":"2026-01-01","instansi_id":"00000000-0000-0000-0005-000000000001"}`
	req := httptest.NewRequest(http.MethodPost, "/api/surat", strings.NewReader(body))
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body)
	}
	if fs.created.TanggalTerima != nil {
		t.Error("tanggal_terima should be nil for keluar")
	}
}

func TestSuratCreate_MasukRequiresTanggalTerima(t *testing.T) {
	fs := &fakeSuratWriteStore{fakeSuratStore: &fakeSuratStore{}}
	d := newWriteDeps(t, fs)
	srv := New(d)

	body := `{"jenis":"masuk","nomor_surat":"X","perihal":"P","tanggal_surat":"2026-01-01","instansi_id":"id"}`
	req := httptest.NewRequest(http.MethodPost, "/api/surat", strings.NewReader(body))
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (missing tanggal_terima)", rec.Code)
	}
}

func TestSuratCreate_StafTidakBolehSecret(t *testing.T) {
	fs := &fakeSuratWriteStore{fakeSuratStore: &fakeSuratStore{}}
	d := newWriteDeps(t, fs)
	srv := New(d)

	body := `{"jenis":"masuk","nomor_surat":"X","perihal":"P","tanggal_surat":"2026-01-01","tanggal_terima":"2026-01-02","instansi_id":"id","access_level":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/surat", strings.NewReader(body))
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestSuratCreate_CamatBolehSecret(t *testing.T) {
	fs := &fakeSuratWriteStore{fakeSuratStore: &fakeSuratStore{}}
	d := newWriteDeps(t, fs)
	srv := New(d)

	body := `{"jenis":"masuk","nomor_surat":"X","perihal":"P","tanggal_surat":"2026-01-01","tanggal_terima":"2026-01-02","instansi_id":"id","access_level":"secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/surat", strings.NewReader(body))
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"camat"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, body = %s", rec.Code, rec.Body)
	}
}

func TestSuratCreate_ConflictNomorKeluar(t *testing.T) {
	fs := &fakeSuratWriteStore{fakeSuratStore: &fakeSuratStore{}, conflictOnCreate: true}
	d := newWriteDeps(t, fs)
	srv := New(d)

	body := `{"jenis":"keluar","nomor_surat":"DUPLICATE","perihal":"P","tanggal_surat":"2026-01-01","instansi_id":"id"}`
	req := httptest.NewRequest(http.MethodPost, "/api/surat", strings.NewReader(body))
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", rec.Code)
	}
}

func TestSuratUpdate_Partial(t *testing.T) {
	fs := &fakeSuratWriteStore{fakeSuratStore: &fakeSuratStore{}}
	d := newWriteDeps(t, fs)
	srv := New(d)

	body := `{"perihal":"Edited"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/surat/some-id", strings.NewReader(body))
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
	upd, ok := fs.updated["some-id"]
	if !ok {
		t.Fatal("update not called")
	}
	if upd.Perihal == nil || *upd.Perihal != "Edited" {
		t.Error("perihal not updated")
	}
	if upd.NomorSurat != nil {
		t.Error("nomor_surat should be nil (not in PATCH)")
	}
}

func TestSuratUpdate_NotFound(t *testing.T) {
	fs := &fakeSuratWriteStore{fakeSuratStore: &fakeSuratStore{}, notFoundOnUpdate: true}
	d := newWriteDeps(t, fs)
	srv := New(d)

	req := httptest.NewRequest(http.MethodPatch, "/api/surat/missing", strings.NewReader(`{"perihal":"X"}`))
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestSuratUpdate_StafTidakBolehSetSecret(t *testing.T) {
	fs := &fakeSuratWriteStore{fakeSuratStore: &fakeSuratStore{}}
	d := newWriteDeps(t, fs)
	srv := New(d)

	req := httptest.NewRequest(http.MethodPatch, "/api/surat/x", strings.NewReader(`{"access_level":"secret"}`))
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestSuratDelete(t *testing.T) {
	fs := &fakeSuratWriteStore{fakeSuratStore: &fakeSuratStore{}}
	d := newWriteDeps(t, fs)
	srv := New(d)

	req := httptest.NewRequest(http.MethodDelete, "/api/surat/abc", nil)
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
	if fs.deleted["abc"] != "u-1" {
		t.Errorf("delete actor = %q", fs.deleted["abc"])
	}
}

func TestSuratRestore_AdminOnly(t *testing.T) {
	fs := &fakeSuratWriteStore{fakeSuratStore: &fakeSuratStore{}}
	d := newWriteDeps(t, fs)
	srv := New(d)

	// Staf cannot restore
	req := httptest.NewRequest(http.MethodPost, "/api/surat/x/restore", nil)
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("staf restore status = %d, want 403", rec.Code)
	}

	// Admin can
	req2 := httptest.NewRequest(http.MethodPost, "/api/surat/x/restore", nil)
	req2.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"admin"}))
	rec2 := httptest.NewRecorder()
	srv.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("admin restore status = %d, body = %s", rec2.Code, rec2.Body)
	}
	if fs.restored["x"] != "u-1" {
		t.Errorf("restore actor = %q", fs.restored["x"])
	}
}
