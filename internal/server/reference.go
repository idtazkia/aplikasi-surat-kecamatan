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

// ReferenceStore extends untuk reference ops.
type ReferenceStore interface {
	GetSuratByID(ctx context.Context, id string) (*store.SuratDetail, error)
	AddReference(ctx context.Context, in store.ReferenceInput) error
	DeleteReference(ctx context.Context, refID, suratID string) error
	GetSuratThread(ctx context.Context, suratID string, includeSecret bool) ([]store.ThreadNode, error)
}

var validRelationships = map[string]bool{
	"balasan":          true,
	"lanjutan":         true,
	"disposisi_hasil":  true,
	"revisi":           true,
	"terkait":          true,
}

type addReferenceRequest struct {
	ToSuratID    *string `json:"to_surat_id,omitempty"`
	ExternalRef  *string `json:"external_ref,omitempty"`
	Relationship string  `json:"relationship"`
	Note         *string `json:"note,omitempty"`
}

func referenceAddHandler(d Deps) http.HandlerFunc {
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

		var req addReferenceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		if !validRelationships[req.Relationship] {
			writeError(w, http.StatusBadRequest,
				"relationship harus salah satu: balasan, lanjutan, disposisi_hasil, revisi, terkait")
			return
		}

		hasInternal := req.ToSuratID != nil && *req.ToSuratID != ""
		hasExternal := req.ExternalRef != nil && *req.ExternalRef != ""
		if !hasInternal && !hasExternal {
			writeError(w, http.StatusBadRequest, "salah satu to_surat_id atau external_ref harus diisi")
			return
		}
		if hasInternal && hasExternal {
			writeError(w, http.StatusBadRequest, "to_surat_id dan external_ref tidak bisa keduanya diisi")
			return
		}

		// Verify ACL on from_surat
		surat, err := d.ReferenceStore.GetSuratByID(r.Context(), suratID)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "surat tidak ditemukan")
			return
		}
		if err != nil {
			d.Logger.Error("ref: surat lookup", "err", err)
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

		input := store.ReferenceInput{
			ID:           newID.String(),
			FromSuratID:  suratID,
			ToSuratID:    req.ToSuratID,
			ExternalRef:  req.ExternalRef,
			Relationship: req.Relationship,
			Note:         req.Note,
			CreatedBy:    claims.Sub,
		}

		if err := d.ReferenceStore.AddReference(r.Context(), input); err != nil {
			if errors.Is(err, store.ErrToSuratNotFound) {
				writeError(w, http.StatusBadRequest, "to_surat_id tidak ditemukan")
				return
			}
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "surat tidak ditemukan")
				return
			}
			d.Logger.Error("ref: insert", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{"id": newID.String()})
	}
}

type threadNodeDTO struct {
	ID           string  `json:"id"`
	NomorSurat   string  `json:"nomor_surat"`
	Perihal      string  `json:"perihal"`
	Jenis        string  `json:"jenis"`
	TanggalSurat string  `json:"tanggal_surat"`
	AccessLevel  string  `json:"access_level"`
	FromSuratID  *string `json:"from_surat_id,omitempty"`
	Relationship *string `json:"relationship,omitempty"`
	ExternalRef  *string `json:"external_ref,omitempty"`
	Depth        int     `json:"depth"`
	Direction    string  `json:"direction"`
}

func suratThreadHandler(d Deps) http.HandlerFunc {
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

		// Verify ACL anchor surat
		surat, err := d.ReferenceStore.GetSuratByID(r.Context(), suratID)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "surat tidak ditemukan")
			return
		}
		if err != nil {
			d.Logger.Error("thread: surat", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if surat.AccessLevel == "secret" && !hasReadSecret(claims.Roles) {
			writeError(w, http.StatusForbidden, "akses surat rahasia ditolak")
			return
		}

		nodes, err := d.ReferenceStore.GetSuratThread(r.Context(), suratID, hasReadSecret(claims.Roles))
		if err != nil {
			d.Logger.Error("thread: query", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		out := make([]threadNodeDTO, 0, len(nodes))
		for _, n := range nodes {
			out = append(out, threadNodeDTO{
				ID: n.ID, NomorSurat: n.NomorSurat, Perihal: n.Perihal,
				Jenis: n.Jenis, TanggalSurat: n.TanggalSurat, AccessLevel: n.AccessLevel,
				FromSuratID: n.FromSuratID, Relationship: n.Relationship,
				ExternalRef: n.ExternalRef, Depth: n.Depth, Direction: n.Direction,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"nodes": out})
	}
}

func referenceDeleteHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		suratID := r.PathValue("id")
		refID := r.PathValue("ref_id")
		if suratID == "" || refID == "" {
			writeError(w, http.StatusBadRequest, "id and ref_id required")
			return
		}

		if err := d.ReferenceStore.DeleteReference(r.Context(), refID, suratID); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "reference tidak ditemukan")
				return
			}
			d.Logger.Error("ref: delete", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}
