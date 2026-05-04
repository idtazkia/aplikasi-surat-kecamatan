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
	ListDisposisi(ctx context.Context, f store.ListDisposisiFilter) ([]store.Disposisi, error)
}

// komentarBaruPayload = payload notifikasi komentar baru ke participant.
type komentarBaruPayload struct {
	KomentarID   string `json:"komentar_id"`
	SuratID      string `json:"surat_id"`
	SuratNomor   string `json:"surat_nomor"`
	SuratPerihal string `json:"surat_perihal"`
	AuthorID     string `json:"author_id"`
	BodyExcerpt  string `json:"body_excerpt"`
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

		// Trigger notifikasi: kirim ke semua participant disposisi pada surat ini,
		// kecuali author komentar sendiri.
		notifyKomentarParticipants(r.Context(), d, suratID, claims.Sub, surat, body, newID.String())

		writeJSON(w, http.StatusCreated, map[string]string{"id": newID.String()})
	}
}

// notifyKomentarParticipants build set participant lalu insert notifikasi.
// Gagal insert tidak block komentar — log + continue (komentar sudah ter-persist).
func notifyKomentarParticipants(
	ctx context.Context,
	d Deps,
	suratID, actorID string,
	surat *store.SuratDetail,
	body string,
	komentarID string,
) {
	disps, err := d.KomentarStore.ListDisposisi(ctx, store.ListDisposisiFilter{
		SuratID:       suratID,
		IncludeSecret: true,
	})
	if err != nil {
		d.Logger.Error("notif: list disposisi for komentar", "err", err)
		return
	}

	// Distinct set: assignee + creator dari semua disposisi, exclude actor.
	recipients := map[string]struct{}{}
	for _, dp := range disps {
		if dp.AssignedTo != actorID {
			recipients[dp.AssignedTo] = struct{}{}
		}
		if dp.CreatedBy != actorID {
			recipients[dp.CreatedBy] = struct{}{}
		}
	}
	if len(recipients) == 0 {
		return
	}

	excerpt := body
	if len(excerpt) > 120 {
		excerpt = excerpt[:120] + "..."
	}
	payload, err := json.Marshal(komentarBaruPayload{
		KomentarID:   komentarID,
		SuratID:      suratID,
		SuratNomor:   surat.NomorSurat,
		SuratPerihal: surat.Perihal,
		AuthorID:     actorID,
		BodyExcerpt:  excerpt,
	})
	if err != nil {
		d.Logger.Error("notif: marshal komentar payload", "err", err)
		return
	}

	for userID := range recipients {
		notifID, err := uuid7.New()
		if err != nil {
			d.Logger.Error("notif: gen uuid", "err", err)
			continue
		}
		if err := d.NotificationStore.CreateNotification(ctx, store.NotificationInput{
			ID: notifID.String(), UserID: userID,
			Type: "komentar_baru", Payload: payload,
		}); err != nil {
			d.Logger.Error("notif: komentar_baru insert", "err", err, "user_id", userID)
		}
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
		writeJSONWithEdu(w, r, d, http.StatusOK, map[string]any{"items": out}, func() *EduPayload {
			return &EduPayload{
				Operation: "list_komentar_append_only",
				DataStructures: []string{
					"Append-only log (no update/delete)",
					"B-tree index (surat_id, created_at) — natural FIFO order",
				},
				Complexity: map[string]interface{}{
					"theoretical": "O(log n + k) — index seek + sequential read k entries",
					"actual": map[string]interface{}{
						"komentar_count": len(items),
					},
				},
				SQL: "SELECT k.id, k.surat_id, k.user_id, u.full_name, k.body, k.created_at\n" +
					"FROM komentar k\n" +
					"JOIN users u ON u.id = k.user_id\n" +
					"WHERE k.surat_id = $1\n" +
					"ORDER BY k.created_at ASC, k.id ASC;\n" +
					"-- Tidak ada UPDATE/DELETE — typo dikoreksi via append entry baru.\n" +
					"-- Audit by construction.",
				ConceptIDs: []string{"append-only-immutability"},
			}
		})
	}
}
