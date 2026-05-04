package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
)

// SuratStore adalah subset interface untuk surat queries.
// Mock-able untuk handler test.
type SuratStore interface {
	ListSurat(ctx context.Context, f store.ListSuratFilter) ([]store.SuratListItem, error)
	GetSuratByID(ctx context.Context, id string) (*store.SuratDetail, error)
}

// suratListResponse adalah envelope dengan cursor untuk next page.
type suratListResponse struct {
	Items      []suratListItemDTO `json:"items"`
	NextCursor *suratCursor       `json:"next_cursor,omitempty"`
}

type suratCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

type suratListItemDTO struct {
	ID              string  `json:"id"`
	Jenis           string  `json:"jenis"`
	NomorSurat      string  `json:"nomor_surat"`
	Perihal         string  `json:"perihal"`
	TanggalSurat    string  `json:"tanggal_surat"`              // YYYY-MM-DD
	TanggalTerima   *string `json:"tanggal_terima,omitempty"`
	InstansiID      string  `json:"instansi_id"`
	InstansiNama    string  `json:"instansi_nama"`
	KlasifikasiKode *string `json:"klasifikasi_kode,omitempty"`
	SifatKode       *string `json:"sifat_kode,omitempty"`
	AccessLevel     string  `json:"access_level"`
	CreatedAt       time.Time `json:"created_at"`
}

func suratListHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		q := r.URL.Query()
		filter := store.ListSuratFilter{
			Jenis:         q.Get("jenis"),
			InstansiID:    q.Get("instansi_id"),
			KlasifikasiID: q.Get("klasifikasi_id"),
			SifatID:       q.Get("sifat_id"),
			Search:        q.Get("search"),
			IncludeSecret: hasReadSecret(claims.Roles),
		}

		if v := q.Get("tanggal_dari"); v != "" {
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				writeError(w, http.StatusBadRequest, "tanggal_dari format invalid (YYYY-MM-DD)")
				return
			}
			filter.TanggalDari = &t
		}
		if v := q.Get("tanggal_sampai"); v != "" {
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				writeError(w, http.StatusBadRequest, "tanggal_sampai format invalid (YYYY-MM-DD)")
				return
			}
			filter.TanggalSampai = &t
		}
		if v := q.Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				writeError(w, http.StatusBadRequest, "limit harus integer positif")
				return
			}
			filter.Limit = n
		}
		if v := q.Get("after_id"); v != "" {
			t := q.Get("after_created_at")
			if t == "" {
				writeError(w, http.StatusBadRequest, "after_id butuh after_created_at sebagai pasangan cursor")
				return
			}
			parsed, err := time.Parse(time.RFC3339Nano, t)
			if err != nil {
				writeError(w, http.StatusBadRequest, "after_created_at format invalid (RFC3339)")
				return
			}
			filter.AfterID = v
			filter.AfterCreatedAt = &parsed
		}

		items, err := d.SuratStore.ListSurat(r.Context(), filter)
		if err != nil {
			d.Logger.Error("list surat: store", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		resp := suratListResponse{Items: make([]suratListItemDTO, 0, len(items))}
		for _, it := range items {
			resp.Items = append(resp.Items, toListDTO(it))
		}
		// Set next cursor kalau hasil sama dengan limit (kemungkinan ada page lanjutan).
		if filter.Limit > 0 && len(items) == filter.Limit {
			last := items[len(items)-1]
			resp.NextCursor = &suratCursor{CreatedAt: last.CreatedAt, ID: last.ID}
		}
		writeJSONWithEdu(w, r, d, http.StatusOK, resp, func() *EduPayload {
			return &EduPayload{
				Operation:      "list_surat_with_keyset_pagination",
				DataStructures: []string{"B-tree index pada (created_at, id)"},
				Complexity: map[string]interface{}{
					"theoretical":   "O(log n + page_size) — index seek + sequential scan",
					"without_index": "O(n) full scan",
					"actual": map[string]interface{}{
						"page_size": filter.Limit,
						"returned":  len(items),
					},
				},
				SQL: "SELECT ... FROM surat s\n" +
					"WHERE NOT s.is_deleted [+ filters]\n" +
					"  AND (s.created_at, s.id) < (cursor_created_at, cursor_id)  -- keyset\n" +
					"ORDER BY s.created_at DESC, s.id DESC\n" +
					"LIMIT $N",
				ConceptIDs: []string{"keyset-pagination", "btree-partial-index-soft-delete"},
			}
		})
	}
}

type suratDetailResponse struct {
	suratListItemDTO
	DeskripsiKlasifikasi *string                 `json:"deskripsi_klasifikasi,omitempty"`
	NamaSifat            *string                 `json:"nama_sifat,omitempty"`
	Attachments          []suratAttachmentDTO    `json:"attachments"`
	Predecessors         []suratReferenceDTO     `json:"predecessors"`
	Successors           []suratReferenceDTO     `json:"successors"`
	Tembusan             []suratTembusanDTO      `json:"tembusan"`
}

type suratTembusanDTO struct {
	ID           string  `json:"id"`
	InstansiID   *string `json:"instansi_id,omitempty"`
	InstansiNama *string `json:"instansi_nama,omitempty"`
	ExternalText *string `json:"external_text,omitempty"`
	Urutan       int     `json:"urutan"`
}

type suratAttachmentDTO struct {
	ID         string    `json:"id"`
	Role       string    `json:"role"`
	FileName   string    `json:"file_name"`
	FileSize   int64     `json:"file_size"`
	MimeType   string    `json:"mime_type"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type suratReferenceDTO struct {
	ID           string  `json:"id"`
	ToSuratID    *string `json:"to_surat_id,omitempty"`
	ToNomorSurat *string `json:"to_nomor_surat,omitempty"`
	ToPerihal    *string `json:"to_perihal,omitempty"`
	Relationship string  `json:"relationship"`
	ExternalRef  *string `json:"external_ref,omitempty"`
	Note         *string `json:"note,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

func suratDetailHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		id := r.PathValue("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "id required")
			return
		}

		detail, err := d.SuratStore.GetSuratByID(r.Context(), id)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "surat tidak ditemukan")
			return
		}
		if err != nil {
			d.Logger.Error("get surat: store", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		// Permission check: secret hanya untuk role yang punya surat:read_secret.
		if detail.AccessLevel == "secret" && !hasReadSecret(claims.Roles) {
			writeError(w, http.StatusForbidden, "akses surat rahasia ditolak")
			return
		}

		writeJSON(w, http.StatusOK, buildSuratDetailDTO(*detail))
	}
}

func toListDTO(it store.SuratListItem) suratListItemDTO {
	dto := suratListItemDTO{
		ID:              it.ID,
		Jenis:           it.Jenis,
		NomorSurat:      it.NomorSurat,
		Perihal:         it.Perihal,
		TanggalSurat:    it.TanggalSurat.Format("2006-01-02"),
		InstansiID:      it.InstansiID,
		InstansiNama:    it.InstansiNama,
		KlasifikasiKode: it.KlasifikasiKode,
		SifatKode:       it.SifatKode,
		AccessLevel:     it.AccessLevel,
		CreatedAt:       it.CreatedAt,
	}
	if it.TanggalTerima != nil {
		s := it.TanggalTerima.Format("2006-01-02")
		dto.TanggalTerima = &s
	}
	return dto
}

func toRefDTO(r store.SuratReference) suratReferenceDTO {
	return suratReferenceDTO{
		ID:           r.ID,
		ToSuratID:    r.ToSuratID,
		ToNomorSurat: r.ToNomorSurat,
		ToPerihal:    r.ToPerihal,
		Relationship: r.Relationship,
		ExternalRef:  r.ExternalRef,
		Note:         r.Note,
		CreatedAt:    r.CreatedAt,
	}
}

func hasReadSecret(roles []string) bool {
	// camat dan admin punya permission surat:read_secret di seed.
	// Untuk Fase 1 minimal hardcode role check; Fase 2 ganti ke permission check
	// dari claims (claims punya field roles, bukan permissions).
	for _, r := range roles {
		if r == "camat" || r == "admin" {
			return true
		}
	}
	return false
}
