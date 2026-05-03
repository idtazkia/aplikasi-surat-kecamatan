package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/uuid7"
)

// KomentarStore subset interface untuk handler komentar.
type KomentarStore interface {
	GetSuratByID(ctx context.Context, id string) (*store.SuratDetail, error)
	AppendKomentar(ctx context.Context, in store.AppendKomentarInput) error
	ListKomentarBySurat(ctx context.Context, suratID string) ([]store.Komentar, error)
}

type komentarDTO struct {
	ID        string    `json:"id"`
	SuratID   string    `json:"surat_id"`
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type appendKomentarRequest struct {
	Body string `json:"body"`
}

func komentarAppendHandler(d Deps) http.HandlerFunc {
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

		var req appendKomentarRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		body := strings.TrimSpace(req.Body)
		if body == "" {
			writeError(w, http.StatusBadRequest, "body wajib diisi")
			return
		}

		// Verify ACL surat
		surat, err := d.KomentarStore.GetSuratByID(r.Context(), suratID)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "surat tidak ditemukan")
			return
		}
		if err != nil {
			d.Logger.Error("komentar: surat lookup", "err", err)
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

		input := store.AppendKomentarInput{
			ID:      newID.String(),
			SuratID: suratID,
			UserID:  claims.Sub,
			Body:    body,
		}
		if err := d.KomentarStore.AppendKomentar(r.Context(), input); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "surat tidak ditemukan")
				return
			}
			d.Logger.Error("komentar: insert", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{"id": newID.String()})
	}
}

func komentarListHandler(d Deps) http.HandlerFunc {
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

		// Verify ACL surat
		surat, err := d.KomentarStore.GetSuratByID(r.Context(), suratID)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "surat tidak ditemukan")
			return
		}
		if err != nil {
			d.Logger.Error("komentar: surat lookup", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if surat.AccessLevel == "secret" && !hasReadSecret(claims.Roles) {
			writeError(w, http.StatusForbidden, "akses surat rahasia ditolak")
			return
		}

		items, err := d.KomentarStore.ListKomentarBySurat(r.Context(), suratID)
		if err != nil {
			d.Logger.Error("komentar: list", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		out := make([]komentarDTO, 0, len(items))
		for _, k := range items {
			out = append(out, komentarDTO{
				ID: k.ID, SuratID: k.SuratID,
				UserID: k.UserID, UserName: k.UserName,
				Body: k.Body, CreatedAt: k.CreatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out})
	}
}
