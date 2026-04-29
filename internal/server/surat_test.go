package server

import (
	"context"
	"encoding/json"
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

// fakeSuratStore in-memory untuk handler test.
type fakeSuratStore struct {
	items   []store.SuratListItem
	details map[string]*store.SuratDetail

	lastFilter store.ListSuratFilter
}

func (f *fakeSuratStore) ListSurat(_ context.Context, filter store.ListSuratFilter) ([]store.SuratListItem, error) {
	f.lastFilter = filter
	out := make([]store.SuratListItem, 0, len(f.items))
	for _, it := range f.items {
		if filter.Jenis != "" && it.Jenis != filter.Jenis {
			continue
		}
		if !filter.IncludeSecret && it.AccessLevel == "secret" {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(it.Perihal), strings.ToLower(filter.Search)) {
			continue
		}
		out = append(out, it)
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeSuratStore) GetSuratByID(_ context.Context, id string) (*store.SuratDetail, error) {
	d, ok := f.details[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	return d, nil
}

func newSuratDeps(t *testing.T, fs *fakeSuratStore) Deps {
	t.Helper()
	a, err := auth.NewService([]byte("test-secret-32-bytes-long-padding"), time.Hour, 24*time.Hour)
	if err != nil {
		t.Fatalf("auth: %v", err)
	}
	return Deps{
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Auth:       a,
		Store:      &fakeStore{users: map[string]*store.UserForLogin{}},
		SuratStore: fs,
	}
}

func bearerHeader(t *testing.T, d Deps, sub string, roles []string) string {
	t.Helper()
	tok, err := d.Auth.Issue(auth.Claims{Sub: sub, Type: "access", Roles: roles})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	return "Bearer " + tok
}

func sampleListItem(id, jenis, perihal, accessLevel string) store.SuratListItem {
	return store.SuratListItem{
		ID:           id,
		Jenis:        jenis,
		NomorSurat:   "X/" + id + "/IV/2026",
		Perihal:      perihal,
		TanggalSurat: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		InstansiID:   "inst-1",
		InstansiNama: "Kemendagri",
		AccessLevel:  accessLevel,
		CreatedAt:    time.Now(),
	}
}

func TestSuratList_RequiresAuth(t *testing.T) {
	d := newSuratDeps(t, &fakeSuratStore{})
	srv := New(d)
	req := httptest.NewRequest(http.MethodGet, "/api/surat", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (no auth header)", rec.Code)
	}
}

func TestSuratList_StafTidakLihatSecret(t *testing.T) {
	fs := &fakeSuratStore{
		items: []store.SuratListItem{
			sampleListItem("1", "masuk", "Edaran biasa", "public"),
			sampleListItem("2", "masuk", "Audit internal", "secret"),
			sampleListItem("3", "keluar", "Tanggapan", "public"),
		},
	}
	d := newSuratDeps(t, fs)
	srv := New(d)

	req := httptest.NewRequest(http.MethodGet, "/api/surat", nil)
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var resp suratListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Items) != 2 {
		t.Errorf("items = %d, want 2 (secret excluded)", len(resp.Items))
	}
	for _, it := range resp.Items {
		if it.AccessLevel == "secret" {
			t.Errorf("secret leaked to staf: %v", it)
		}
	}
	if fs.lastFilter.IncludeSecret {
		t.Error("filter.IncludeSecret should be false untuk staf")
	}
}

func TestSuratList_CamatLihatSecret(t *testing.T) {
	fs := &fakeSuratStore{
		items: []store.SuratListItem{
			sampleListItem("1", "masuk", "Edaran biasa", "public"),
			sampleListItem("2", "masuk", "Audit internal", "secret"),
		},
	}
	d := newSuratDeps(t, fs)
	srv := New(d)

	req := httptest.NewRequest(http.MethodGet, "/api/surat", nil)
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"camat"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
	}
	if !fs.lastFilter.IncludeSecret {
		t.Error("camat should include_secret=true")
	}
	var resp suratListResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Items) != 2 {
		t.Errorf("items = %d, want 2 (camat lihat semua)", len(resp.Items))
	}
}

func TestSuratList_FilterJenis(t *testing.T) {
	fs := &fakeSuratStore{
		items: []store.SuratListItem{
			sampleListItem("1", "masuk", "Edaran", "public"),
			sampleListItem("2", "keluar", "Tanggapan", "public"),
			sampleListItem("3", "masuk", "Undangan", "public"),
		},
	}
	d := newSuratDeps(t, fs)
	srv := New(d)

	req := httptest.NewRequest(http.MethodGet, "/api/surat?jenis=masuk", nil)
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if fs.lastFilter.Jenis != "masuk" {
		t.Errorf("filter.Jenis = %s, want masuk", fs.lastFilter.Jenis)
	}
	var resp suratListResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Items) != 2 {
		t.Errorf("items = %d, want 2 (filter masuk)", len(resp.Items))
	}
}

func TestSuratList_SearchPerihal(t *testing.T) {
	fs := &fakeSuratStore{
		items: []store.SuratListItem{
			sampleListItem("1", "masuk", "Edaran Penanganan Pandemi", "public"),
			sampleListItem("2", "masuk", "Undangan Rapat", "public"),
		},
	}
	d := newSuratDeps(t, fs)
	srv := New(d)

	req := httptest.NewRequest(http.MethodGet, "/api/surat?search=pandemi", nil)
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	var resp suratListResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Items) != 1 {
		t.Errorf("items = %d, want 1 (search pandemi)", len(resp.Items))
	}
}

func TestSuratList_InvalidDateFormat(t *testing.T) {
	d := newSuratDeps(t, &fakeSuratStore{})
	srv := New(d)

	req := httptest.NewRequest(http.MethodGet, "/api/surat?tanggal_dari=not-a-date", nil)
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestSuratList_NextCursorReturnedWhenLimitReached(t *testing.T) {
	items := make([]store.SuratListItem, 5)
	for i := 0; i < 5; i++ {
		items[i] = sampleListItem(string(rune('a'+i)), "masuk", "Surat", "public")
	}
	fs := &fakeSuratStore{items: items}
	d := newSuratDeps(t, fs)
	srv := New(d)

	req := httptest.NewRequest(http.MethodGet, "/api/surat?limit=3", nil)
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	var resp suratListResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)

	if len(resp.Items) != 3 {
		t.Errorf("items = %d, want 3", len(resp.Items))
	}
	if resp.NextCursor == nil {
		t.Error("expected next_cursor (limit reached)")
	}
}

func TestSuratList_NoNextCursorWhenFewerThanLimit(t *testing.T) {
	fs := &fakeSuratStore{items: []store.SuratListItem{
		sampleListItem("1", "masuk", "X", "public"),
	}}
	d := newSuratDeps(t, fs)
	srv := New(d)

	req := httptest.NewRequest(http.MethodGet, "/api/surat?limit=20", nil)
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	var resp suratListResponse
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.NextCursor != nil {
		t.Errorf("expected no cursor, got %+v", resp.NextCursor)
	}
}

func TestSuratDetail_NotFound(t *testing.T) {
	fs := &fakeSuratStore{details: map[string]*store.SuratDetail{}}
	d := newSuratDeps(t, fs)
	srv := New(d)

	req := httptest.NewRequest(http.MethodGet, "/api/surat/abc-123", nil)
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestSuratDetail_StafCannotAccessSecret(t *testing.T) {
	id := "secret-id"
	fs := &fakeSuratStore{details: map[string]*store.SuratDetail{
		id: {
			SuratListItem: sampleListItem(id, "masuk", "Audit", "secret"),
		},
	}}
	d := newSuratDeps(t, fs)
	srv := New(d)

	req := httptest.NewRequest(http.MethodGet, "/api/surat/"+id, nil)
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestSuratDetail_CamatAccessSecret(t *testing.T) {
	id := "secret-id"
	fs := &fakeSuratStore{details: map[string]*store.SuratDetail{
		id: {
			SuratListItem: sampleListItem(id, "masuk", "Audit", "secret"),
		},
	}}
	d := newSuratDeps(t, fs)
	srv := New(d)

	req := httptest.NewRequest(http.MethodGet, "/api/surat/"+id, nil)
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"camat"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestSuratDetail_ResponseShape(t *testing.T) {
	id := "uuid-1"
	external := "Surat lama tahun 2020"
	note := "ditembuskan ke Dinkes"
	target := "uuid-target"
	targetNomor := "X/Y/Z"
	targetPerihal := "Tujuan"
	fs := &fakeSuratStore{details: map[string]*store.SuratDetail{
		id: {
			SuratListItem: sampleListItem(id, "masuk", "Edaran", "public"),
			Attachments: []store.SuratAttachment{
				{ID: "att-1", Role: "primary", FileName: "edaran.pdf", FileSize: 1024, MimeType: "application/pdf", UploadedAt: time.Now()},
			},
			Predecessors: []store.SuratReference{
				{ID: "ref-1", ExternalRef: &external, Note: &note, Relationship: "lanjutan", CreatedAt: time.Now()},
			},
			Successors: []store.SuratReference{
				{ID: "ref-2", ToSuratID: &target, ToNomorSurat: &targetNomor, ToPerihal: &targetPerihal, Relationship: "balasan", CreatedAt: time.Now()},
			},
		},
	}}
	d := newSuratDeps(t, fs)
	srv := New(d)

	req := httptest.NewRequest(http.MethodGet, "/api/surat/"+id, nil)
	req.Header.Set("Authorization", bearerHeader(t, d, "u-1", []string{"staf"}))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp suratDetailResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ID != id {
		t.Errorf("ID = %s, want %s", resp.ID, id)
	}
	if len(resp.Attachments) != 1 || resp.Attachments[0].FileName != "edaran.pdf" {
		t.Errorf("attachments mismatch: %+v", resp.Attachments)
	}
	if len(resp.Predecessors) != 1 || *resp.Predecessors[0].ExternalRef != external {
		t.Errorf("predecessors mismatch: %+v", resp.Predecessors)
	}
	if len(resp.Successors) != 1 || resp.Successors[0].Relationship != "balasan" {
		t.Errorf("successors mismatch: %+v", resp.Successors)
	}
}
