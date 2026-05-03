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

// disposisiBaruPayload = payload notifikasi yang dikirim ke assignee.
type disposisiBaruPayload struct {
	DisposisiID  string `json:"disposisi_id"`
	SuratID      string `json:"surat_id"`
	SuratNomor   string `json:"surat_nomor"`
	SuratPerihal string `json:"surat_perihal"`
	CreatorName  string `json:"creator_name"`
	Instruksi    string `json:"instruksi"`
}

// DisposisiStore subset interface untuk handler disposisi.
type DisposisiStore interface {
	GetSuratByID(ctx context.Context, id string) (*store.SuratDetail, error)
	CreateDisposisi(ctx context.Context, in store.CreateDisposisiInput) error
	UpdateDisposisiStatus(ctx context.Context, id string, in store.UpdateDisposisiStatusInput) error
	ListDisposisi(ctx context.Context, f store.ListDisposisiFilter) ([]store.Disposisi, error)
	GetDisposisiByID(ctx context.Context, id string) (*store.Disposisi, error)
	ListAssignableUsers(ctx context.Context) ([]store.AssignableUser, error)
}

// NotificationStore subset interface untuk write trigger.
type NotificationStore interface {
	CreateNotification(ctx context.Context, in store.NotificationInput) error
	ListNotifications(ctx context.Context, f store.ListNotificationsFilter) ([]store.Notification, error)
	CountUnreadNotifications(ctx context.Context, userID string) (int, error)
	MarkNotificationRead(ctx context.Context, id, userID string) error
	MarkAllNotificationsRead(ctx context.Context, userID string) error
}

type disposisiDTO struct {
	ID             string     `json:"id"`
	SuratID        string     `json:"surat_id"`
	SuratNomor     string     `json:"surat_nomor"`
	SuratPerihal   string     `json:"surat_perihal"`
	AssignedTo     string     `json:"assigned_to"`
	AssigneeName   string     `json:"assignee_name"`
	NomorDisposisi *string    `json:"nomor_disposisi,omitempty"`
	Instruksi      string     `json:"instruksi"`
	Deadline       *time.Time `json:"deadline,omitempty"`
	Status         string     `json:"status"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	CreatedBy      string     `json:"created_by"`
	CreatorName    string     `json:"creator_name"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func toDisposisiDTO(d store.Disposisi) disposisiDTO {
	return disposisiDTO{
		ID: d.ID, SuratID: d.SuratID, SuratNomor: d.SuratNomor, SuratPerihal: d.SuratPerihal,
		AssignedTo: d.AssignedTo, AssigneeName: d.AssigneeName,
		NomorDisposisi: d.NomorDisposisi, Instruksi: d.Instruksi, Deadline: d.Deadline,
		Status: d.Status, CompletedAt: d.CompletedAt,
		CreatedBy: d.CreatedBy, CreatorName: d.CreatorName,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

type createDisposisiRequest struct {
	SuratID        string  `json:"surat_id"`
	AssignedTo     string  `json:"assigned_to"`
	NomorDisposisi *string `json:"nomor_disposisi,omitempty"`
	Instruksi      string  `json:"instruksi"`
	Deadline       *string `json:"deadline,omitempty"` // RFC3339
}

func disposisiCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		// Authorize: hanya camat, admin, atau staf yang boleh assign disposisi.
		// Student dilarang. Aturan ini bisa di-tighten ke camat-only di Fase 3+.
		if !canAssignDisposisi(claims.Roles) {
			writeError(w, http.StatusForbidden, "tidak punya hak assign disposisi")
			return
		}

		var req createDisposisiRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.SuratID == "" || req.AssignedTo == "" || req.Instruksi == "" {
			writeError(w, http.StatusBadRequest, "surat_id, assigned_to, instruksi wajib diisi")
			return
		}

		var deadline *time.Time
		if req.Deadline != nil && *req.Deadline != "" {
			t, err := time.Parse(time.RFC3339, *req.Deadline)
			if err != nil {
				writeError(w, http.StatusBadRequest, "deadline format invalid (RFC3339)")
				return
			}
			deadline = &t
		}

		// Verify surat ACL — secret butuh role read_secret
		surat, err := d.DisposisiStore.GetSuratByID(r.Context(), req.SuratID)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "surat tidak ditemukan")
			return
		}
		if err != nil {
			d.Logger.Error("disposisi: surat lookup", "err", err)
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

		input := store.CreateDisposisiInput{
			ID:             newID.String(),
			SuratID:        req.SuratID,
			AssignedTo:     req.AssignedTo,
			NomorDisposisi: req.NomorDisposisi,
			Instruksi:      req.Instruksi,
			Deadline:       deadline,
			CreatedBy:      claims.Sub,
		}

		if err := d.DisposisiStore.CreateDisposisi(r.Context(), input); err != nil {
			if errors.Is(err, store.ErrAssigneeNotFound) {
				writeError(w, http.StatusBadRequest, "assigned_to user tidak ditemukan / inactive")
				return
			}
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "surat tidak ditemukan")
				return
			}
			d.Logger.Error("disposisi: insert", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		// Trigger notifikasi: assignee dapat notif disposisi_baru kalau bukan self-assignment.
		if req.AssignedTo != claims.Sub {
			payload, err := json.Marshal(disposisiBaruPayload{
				DisposisiID:  newID.String(),
				SuratID:      req.SuratID,
				SuratNomor:   surat.NomorSurat,
				SuratPerihal: surat.Perihal,
				CreatorName:  claims.Sub, // sub = userID; full_name di-resolve di frontend kalau perlu
				Instruksi:    req.Instruksi,
			})
			if err == nil {
				notifID, gerr := uuid7.New()
				if gerr == nil {
					if nerr := d.NotificationStore.CreateNotification(r.Context(), store.NotificationInput{
						ID: notifID.String(), UserID: req.AssignedTo,
						Type: "disposisi_baru", Payload: payload,
					}); nerr != nil {
						// Notifikasi gagal tidak block disposisi creation — log + continue.
						d.Logger.Error("notif: disposisi_baru insert failed", "err", nerr)
					}
				}
			}
		}

		writeJSON(w, http.StatusCreated, map[string]string{"id": newID.String()})
	}
}

type updateDisposisiRequest struct {
	Status    string  `json:"status"`
	Instruksi *string `json:"instruksi,omitempty"`
}

func disposisiUpdateHandler(d Deps) http.HandlerFunc {
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

		var req updateDisposisiRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Status == "" {
			writeError(w, http.StatusBadRequest, "status wajib diisi")
			return
		}

		// Authorize: hanya assignee atau creator/camat/admin yang boleh update.
		existing, err := d.DisposisiStore.GetDisposisiByID(r.Context(), id)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "disposisi tidak ditemukan")
			return
		}
		if err != nil {
			d.Logger.Error("disposisi: get", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		isAssignee := existing.AssignedTo == claims.Sub
		isCreator := existing.CreatedBy == claims.Sub
		isCamat := hasRole(claims.Roles, "camat") || hasRole(claims.Roles, "admin")
		if !isAssignee && !isCreator && !isCamat {
			writeError(w, http.StatusForbidden, "hanya assignee/creator/camat yang boleh update")
			return
		}

		input := store.UpdateDisposisiStatusInput{
			Status:    req.Status,
			Instruksi: req.Instruksi,
			UpdatedBy: claims.Sub,
		}
		if err := d.DisposisiStore.UpdateDisposisiStatus(r.Context(), id, input); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "disposisi tidak ditemukan")
				return
			}
			d.Logger.Error("disposisi: update", "err", err)
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

func disposisiListHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		q := r.URL.Query()
		filter := store.ListDisposisiFilter{
			SuratID:       q.Get("surat_id"),
			AssignedTo:    q.Get("assigned_to"),
			CreatedBy:     q.Get("created_by"),
			Status:        q.Get("status"),
			IncludeSecret: hasReadSecret(claims.Roles),
		}
		// Shortcut: ?mine=true → assigned_to = current user
		if q.Get("mine") == "true" {
			filter.AssignedTo = claims.Sub
		}

		items, err := d.DisposisiStore.ListDisposisi(r.Context(), filter)
		if err != nil {
			d.Logger.Error("disposisi: list", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		out := make([]disposisiDTO, 0, len(items))
		for _, d := range items {
			out = append(out, toDisposisiDTO(d))
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out})
	}
}

func assignableUsersHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		users, err := d.DisposisiStore.ListAssignableUsers(r.Context())
		if err != nil {
			d.Logger.Error("disposisi: list users", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		type userDTO struct {
			ID       string   `json:"id"`
			Username string   `json:"username"`
			FullName string   `json:"full_name"`
			Roles    []string `json:"roles"`
		}
		out := make([]userDTO, 0, len(users))
		for _, u := range users {
			out = append(out, userDTO{
				ID: u.ID, Username: u.Username, FullName: u.FullName, Roles: u.Roles,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out})
	}
}

func canAssignDisposisi(roles []string) bool {
	for _, r := range roles {
		if r == "staf" || r == "camat" || r == "admin" {
			return true
		}
	}
	return false
}

