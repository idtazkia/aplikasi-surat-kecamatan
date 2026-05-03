package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
)

type notificationDTO struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	ReadAt    *time.Time      `json:"read_at,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

func notificationListHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		filter := store.ListNotificationsFilter{
			UserID:     claims.Sub,
			UnreadOnly: r.URL.Query().Get("unread") == "true",
		}
		items, err := d.NotificationStore.ListNotifications(r.Context(), filter)
		if err != nil {
			d.Logger.Error("notif: list", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		// Hitung unread count sekalian (cheap pakai partial index).
		unread, err := d.NotificationStore.CountUnreadNotifications(r.Context(), claims.Sub)
		if err != nil {
			d.Logger.Error("notif: count", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		out := make([]notificationDTO, 0, len(items))
		for _, n := range items {
			out = append(out, notificationDTO{
				ID: n.ID, Type: n.Type, Payload: n.Payload,
				ReadAt: n.ReadAt, CreatedAt: n.CreatedAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items":  out,
			"unread": unread,
		})
	}
}

func notificationMarkReadHandler(d Deps) http.HandlerFunc {
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
		if err := d.NotificationStore.MarkNotificationRead(r.Context(), id, claims.Sub); err != nil {
			d.Logger.Error("notif: mark read", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func notificationMarkAllReadHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}
		if err := d.NotificationStore.MarkAllNotificationsRead(r.Context(), claims.Sub); err != nil {
			d.Logger.Error("notif: mark all read", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
