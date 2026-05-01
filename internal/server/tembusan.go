package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/uuid7"
)

// TembusanStore extends untuk tembusan ops.
type TembusanStore interface {
	GetSuratByID(ctx context.Context, id string) (*store.SuratDetail, error)
	AddTembusan(ctx context.Context, in store.TembusanInput) error
	DeleteTembusan(ctx context.Context, tembusanID, suratID string) error
}

type addTembusanRequest struct {
	InstansiID   *string `json:"instansi_id,omitempty"`
	ExternalText *string `json:"external_text,omitempty"`
	Urutan       int     `json:"urutan,omitempty"`
}

func tembusanAddHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		suratID := r.PathValue("id")
		if suratID == "" {
			writeError(w, http.StatusBadRequest, "id required")
			return
		}

		var req addTembusanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		hasInternal := req.InstansiID != nil && *req.InstansiID != ""
		hasExternal := req.ExternalText != nil && *req.ExternalText != ""
		if !hasInternal && !hasExternal {
			writeError(w, http.StatusBadRequest, "salah satu instansi_id atau external_text harus diisi")
			return
		}
		if hasInternal && hasExternal {
			writeError(w, http.StatusBadRequest, "instansi_id dan external_text tidak bisa keduanya diisi")
			return
		}

		// Verify ACL on surat
		surat, err := d.TembusanStore.GetSuratByID(r.Context(), suratID)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "surat tidak ditemukan")
			return
		}
		if err != nil {
			d.Logger.Error("tembusan: surat lookup", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if surat.AccessLevel == "secret" && !hasReadSecret(claims.Roles) {
			writeError(w, http.StatusForbidden, "akses surat rahasia ditolak")
			return
		}

		newID, err := uuid7.New()
		if err != nil {
			d.Logger.Error("uuid: generate", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		input := store.TembusanInput{
			ID:           newID.String(),
			SuratID:      suratID,
			InstansiID:   req.InstansiID,
			ExternalText: req.ExternalText,
			Urutan:       req.Urutan,
		}

		if err := d.TembusanStore.AddTembusan(r.Context(), input); err != nil {
			if errors.Is(err, store.ErrInstansiNotFound) {
				writeError(w, http.StatusBadRequest, "instansi_id tidak ditemukan")
				return
			}
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "surat tidak ditemukan")
				return
			}
			d.Logger.Error("tembusan: insert", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{"id": newID.String()})
	}
}

func tembusanDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		suratID := r.PathValue("id")
		tembusanID := r.PathValue("tembusan_id")
		if suratID == "" || tembusanID == "" {
			writeError(w, http.StatusBadRequest, "id and tembusan_id required")
			return
		}

		if err := d.TembusanStore.DeleteTembusan(r.Context(), tembusanID, suratID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "tembusan tidak ditemukan")
				return
			}
			d.Logger.Error("tembusan: delete", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}
