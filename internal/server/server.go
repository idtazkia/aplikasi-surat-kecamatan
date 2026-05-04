// Package server mengonstruksi HTTP router dan menjahit dependencies bersama.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
)

// UserStore adalah interface yang dibutuhkan handler. Pakai interface supaya
// handler bisa di-test dengan in-memory implementation tanpa real DB.
type UserStore interface {
	GetUserForLogin(ctx context.Context, username string) (*store.UserForLogin, error)
	Ping(ctx context.Context) error
}

// Deps semua dependency yang dibutuhkan router.
type Deps struct {
	Logger          *slog.Logger
	Auth            *auth.Service
	Store           UserStore
	SuratStore      SuratStore
	AttachmentStore AttachmentStore
	ReferenceStore  ReferenceStore
	TembusanStore   TembusanStore
	DisposisiStore  DisposisiStore
	KomentarStore   KomentarStore
	DashboardStore     DashboardStore
	NotificationStore  NotificationStore
	SyncStore          SyncStore
	DirektoriStore     DirektoriStore
	AttachmentRoot  string
}

// New membangun *http.ServeMux dengan semua route ter-register.
func New(d Deps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthHandler(d))
	mux.HandleFunc("POST /api/auth/login", loginHandler(d))
	mux.HandleFunc("POST /api/auth/refresh", refreshHandler(d))

	// Protected: butuh access token.
	mux.Handle("GET /api/me", d.Auth.Middleware(http.HandlerFunc(meHandler(d))))
	mux.Handle("GET /api/surat", d.Auth.Middleware(http.HandlerFunc(suratListHandler(d))))
	mux.Handle("GET /api/surat/{id}", d.Auth.Middleware(http.HandlerFunc(suratDetailHandler(d))))
	mux.Handle("POST /api/surat", d.Auth.Middleware(http.HandlerFunc(suratCreateHandler(d))))
	mux.Handle("PATCH /api/surat/{id}", d.Auth.Middleware(http.HandlerFunc(suratUpdateHandler(d))))
	mux.Handle("DELETE /api/surat/{id}", d.Auth.Middleware(http.HandlerFunc(suratDeleteHandler(d))))
	mux.Handle("POST /api/surat/{id}/restore", d.Auth.Middleware(http.HandlerFunc(suratRestoreHandler(d))))
	mux.Handle("POST /api/surat/{id}/attachments", d.Auth.Middleware(http.HandlerFunc(suratAttachmentsUploadHandler(d))))
	mux.Handle("GET /api/surat/{id}/attachments/{att_id}", d.Auth.Middleware(http.HandlerFunc(suratAttachmentDownloadHandler(d))))
	mux.Handle("GET /api/surat/{id}/attachments/{att_id}/preview", d.Auth.Middleware(http.HandlerFunc(suratAttachmentPreviewHandler(d))))
	mux.Handle("PATCH /api/surat/{id}/attachments/{att_id}/replace", d.Auth.Middleware(http.HandlerFunc(suratAttachmentReplaceHandler(d))))
	mux.Handle("GET /api/surat/{id}/attachments/{att_id}/versions", d.Auth.Middleware(http.HandlerFunc(suratAttachmentVersionsHandler(d))))
	mux.Handle("POST /api/surat/{id}/references", d.Auth.Middleware(http.HandlerFunc(referenceAddHandler(d))))
	mux.Handle("DELETE /api/surat/{id}/references/{ref_id}", d.Auth.Middleware(http.HandlerFunc(referenceDeleteHandler(d))))
	mux.Handle("GET /api/surat/{id}/thread", d.Auth.Middleware(http.HandlerFunc(suratThreadHandler(d))))
	mux.Handle("POST /api/surat/{id}/tembusan", d.Auth.Middleware(http.HandlerFunc(tembusanAddHandler(d))))
	mux.Handle("DELETE /api/surat/{id}/tembusan/{tembusan_id}", d.Auth.Middleware(http.HandlerFunc(tembusanDeleteHandler(d))))
	mux.Handle("POST /api/disposisi", d.Auth.Middleware(http.HandlerFunc(disposisiCreateHandler(d))))
	mux.Handle("PATCH /api/disposisi/{id}", d.Auth.Middleware(http.HandlerFunc(disposisiUpdateHandler(d))))
	mux.Handle("GET /api/disposisi", d.Auth.Middleware(http.HandlerFunc(disposisiListHandler(d))))
	mux.Handle("GET /api/users/assignable", d.Auth.Middleware(http.HandlerFunc(assignableUsersHandler(d))))
	mux.Handle("POST /api/surat/{id}/komentar", d.Auth.Middleware(http.HandlerFunc(komentarAppendHandler(d))))
	mux.Handle("GET /api/surat/{id}/komentar", d.Auth.Middleware(http.HandlerFunc(komentarListHandler(d))))
	mux.Handle("GET /api/dashboard/camat", d.Auth.Middleware(http.HandlerFunc(dashboardCamatHandler(d))))
	mux.Handle("GET /api/notifications", d.Auth.Middleware(http.HandlerFunc(notificationListHandler(d))))
	mux.Handle("PATCH /api/notifications/{id}/read", d.Auth.Middleware(http.HandlerFunc(notificationMarkReadHandler(d))))
	mux.Handle("POST /api/notifications/read-all", d.Auth.Middleware(http.HandlerFunc(notificationMarkAllReadHandler(d))))
	mux.Handle("GET /api/sync/snapshot", d.Auth.Middleware(http.HandlerFunc(syncSnapshotHandler(d))))
	mux.Handle("GET /api/instansi", d.Auth.Middleware(http.HandlerFunc(instansiSearchHandler(d))))
	mux.Handle("POST /api/instansi", d.Auth.Middleware(http.HandlerFunc(instansiCreateHandler(d))))
	mux.Handle("GET /api/klasifikasi", d.Auth.Middleware(http.HandlerFunc(klasifikasiListHandler(d))))
	mux.Handle("GET /api/sifat", d.Auth.Middleware(http.HandlerFunc(sifatListHandler(d))))

	return mux
}

func healthHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := "ok"
		code := http.StatusOK
		if err := d.Store.Ping(r.Context()); err != nil {
			d.Logger.Error("healthz: db ping failed", "err", err)
			status = "db_unreachable"
			code = http.StatusServiceUnavailable
		}
		writeJSON(w, code, map[string]string{"status": status})
	}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func loginHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Username == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "username dan password required")
			return
		}

		u, err := d.Store.GetUserForLogin(r.Context(), req.Username)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		if err != nil {
			d.Logger.Error("login: store lookup", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if !u.IsActive {
			writeError(w, http.StatusUnauthorized, "user inactive")
			return
		}
		if err := auth.VerifyPassword(req.Password, u.PasswordHash); err != nil {
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}

		access, refresh, err := d.Auth.IssueAccessAndRefresh(u.ID, u.Roles)
		if err != nil {
			d.Logger.Error("login: issue token", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		writeJSON(w, http.StatusOK, loginResponse{AccessToken: access, RefreshToken: refresh})
	}
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func refreshHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req refreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		claims, err := d.Auth.Verify(req.RefreshToken)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid refresh token")
			return
		}
		if claims.Type != "refresh" {
			writeError(w, http.StatusUnauthorized, "not a refresh token")
			return
		}
		// Issue access token baru (refresh tetap, akan expire sesuai TTL).
		access, err := d.Auth.Issue(auth.Claims{Sub: claims.Sub, Type: "access", Roles: claims.Roles})
		if err != nil {
			d.Logger.Error("refresh: issue token", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"access_token": access})
	}
}

func meHandler(_ Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusInternalServerError, "claims missing")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"user_id": c.Sub,
			"roles":   c.Roles,
		})
	}
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
