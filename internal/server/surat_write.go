package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/uuid7"
)

// SuratWriteStore extends SuratStore dengan write methods.
type SuratWriteStore interface {
	SuratStore
	CreateSurat(ctx context.Context, in store.CreateSuratInput) error
	UpdateSurat(ctx context.Context, id string, in store.UpdateSuratInput) error
	SoftDeleteSurat(ctx context.Context, id, actorID string) error
	RestoreSurat(ctx context.Context, id, actorID string) error
}

type createSuratRequest struct {
	Jenis         string  `json:"jenis"`
	NomorSurat    string  `json:"nomor_surat"`
	Perihal       string  `json:"perihal"`
	TanggalSurat  string  `json:"tanggal_surat"`
	TanggalTerima *string `json:"tanggal_terima,omitempty"`
	InstansiID    string  `json:"instansi_id"`
	KlasifikasiID *string `json:"klasifikasi_id,omitempty"`
	SifatID       *string `json:"sifat_id,omitempty"`
	AccessLevel   string  `json:"access_level"`
}

type createSuratResponse struct {
	ID string `json:"id"`
}

func suratCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		var req createSuratRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		if errMsg := validateCreateSurat(&req); errMsg != "" {
			writeError(w, http.StatusBadRequest, errMsg)
			return
		}
		if req.AccessLevel == "" {
			req.AccessLevel = "public"
		}

		// Hanya camat/admin yang boleh set access_level=secret
		if req.AccessLevel == "secret" && !hasReadSecret(claims.Roles) {
			writeError(w, http.StatusForbidden, "set access_level=secret butuh role camat/admin")
			return
		}

		tanggalSurat, _ := time.Parse("2006-01-02", req.TanggalSurat)
		var tanggalTerima *time.Time
		if req.TanggalTerima != nil {
			t, _ := time.Parse("2006-01-02", *req.TanggalTerima)
			tanggalTerima = &t
		}

		// Generate UUIDv7 server-side untuk online flow.
		// Di Fase 4 client akan generate dan kirim ID via header.
		newID, err := uuid7.New()
		if err != nil {
			d.Logger.Error("uuid: generate", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		input := store.CreateSuratInput{
			ID:            newID.String(),
			Jenis:         req.Jenis,
			NomorSurat:    req.NomorSurat,
			Perihal:       req.Perihal,
			TanggalSurat:  tanggalSurat,
			TanggalTerima: tanggalTerima,
			InstansiID:    req.InstansiID,
			KlasifikasiID: req.KlasifikasiID,
			SifatID:       req.SifatID,
			AccessLevel:   req.AccessLevel,
			CreatedBy:     claims.Sub,
		}

		if err := d.SuratStore.(SuratWriteStore).CreateSurat(r.Context(), input); err != nil {
			if errors.Is(err, store.ErrConflict) {
				writeError(w, http.StatusConflict, "nomor_surat sudah dipakai (untuk surat keluar harus unique)")
				return
			}
			d.Logger.Error("create surat: store", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		writeJSON(w, http.StatusCreated, createSuratResponse{ID: input.ID})
	}
}

type updateSuratRequest struct {
	NomorSurat    *string `json:"nomor_surat,omitempty"`
	Perihal       *string `json:"perihal,omitempty"`
	TanggalSurat  *string `json:"tanggal_surat,omitempty"`
	TanggalTerima *string `json:"tanggal_terima,omitempty"`
	InstansiID    *string `json:"instansi_id,omitempty"`
	KlasifikasiID *string `json:"klasifikasi_id,omitempty"`
	SifatID       *string `json:"sifat_id,omitempty"`
	AccessLevel   *string `json:"access_level,omitempty"`
}

func suratUpdateHandler(d Deps) http.HandlerFunc {
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

		var req updateSuratRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		// Set access_level=secret butuh permission
		if req.AccessLevel != nil && *req.AccessLevel == "secret" && !hasReadSecret(claims.Roles) {
			writeError(w, http.StatusForbidden, "set access_level=secret butuh role camat/admin")
			return
		}
		if req.AccessLevel != nil {
			al := *req.AccessLevel
			if al != "public" && al != "restricted" && al != "secret" {
				writeError(w, http.StatusBadRequest, "access_level harus public/restricted/secret")
				return
			}
		}

		input := store.UpdateSuratInput{
			NomorSurat:    req.NomorSurat,
			Perihal:       req.Perihal,
			InstansiID:    req.InstansiID,
			KlasifikasiID: req.KlasifikasiID,
			SifatID:       req.SifatID,
			AccessLevel:   req.AccessLevel,
			UpdatedBy:     claims.Sub,
		}
		if req.TanggalSurat != nil {
			t, err := time.Parse("2006-01-02", *req.TanggalSurat)
			if err != nil {
				writeError(w, http.StatusBadRequest, "tanggal_surat format invalid")
				return
			}
			input.TanggalSurat = &t
		}
		if req.TanggalTerima != nil {
			t, err := time.Parse("2006-01-02", *req.TanggalTerima)
			if err != nil {
				writeError(w, http.StatusBadRequest, "tanggal_terima format invalid")
				return
			}
			input.TanggalTerima = &t
		}

		if err := d.SuratStore.(SuratWriteStore).UpdateSurat(r.Context(), id, input); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "surat tidak ditemukan")
				return
			}
			if errors.Is(err, store.ErrConflict) {
				writeError(w, http.StatusConflict, "nomor_surat conflict")
				return
			}
			d.Logger.Error("update surat: store", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

func suratDeleteHandler(d Deps) http.HandlerFunc {
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

		if err := d.SuratStore.(SuratWriteStore).SoftDeleteSurat(r.Context(), id, claims.Sub); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "surat tidak ditemukan")
				return
			}
			d.Logger.Error("delete surat: store", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func suratRestoreHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		// Hanya admin yang bisa restore
		if !hasRole(claims.Roles, "admin") {
			writeError(w, http.StatusForbidden, "restore butuh role admin")
			return
		}

		id := r.PathValue("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "id required")
			return
		}

		if err := d.SuratStore.(SuratWriteStore).RestoreSurat(r.Context(), id, claims.Sub); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "surat tidak ditemukan atau bukan deleted")
				return
			}
			d.Logger.Error("restore surat: store", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "restored"})
	}
}

func validateCreateSurat(req *createSuratRequest) string {
	if req.Jenis != "masuk" && req.Jenis != "keluar" {
		return "jenis harus 'masuk' atau 'keluar'"
	}
	if req.NomorSurat == "" {
		return "nomor_surat required"
	}
	if req.Perihal == "" {
		return "perihal required"
	}
	if req.TanggalSurat == "" {
		return "tanggal_surat required"
	}
	if _, err := time.Parse("2006-01-02", req.TanggalSurat); err != nil {
		return "tanggal_surat format invalid (YYYY-MM-DD)"
	}
	if req.Jenis == "masuk" {
		if req.TanggalTerima == nil || *req.TanggalTerima == "" {
			return "tanggal_terima required untuk surat masuk"
		}
		if _, err := time.Parse("2006-01-02", *req.TanggalTerima); err != nil {
			return "tanggal_terima format invalid (YYYY-MM-DD)"
		}
	}
	if req.Jenis == "keluar" && req.TanggalTerima != nil && *req.TanggalTerima != "" {
		return "tanggal_terima tidak boleh ada untuk surat keluar"
	}
	if req.InstansiID == "" {
		return "instansi_id required"
	}
	if req.AccessLevel != "" && req.AccessLevel != "public" && req.AccessLevel != "restricted" && req.AccessLevel != "secret" {
		return "access_level harus public/restricted/secret"
	}
	return ""
}

func hasRole(roles []string, code string) bool {
	for _, r := range roles {
		if r == code {
			return true
		}
	}
	return false
}
