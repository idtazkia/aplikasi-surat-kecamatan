package server

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
)

// StatsStore subset interface untuk reporting handlers.
type StatsStore interface {
	GetStatsByPeriod(ctx context.Context, from, to *time.Time) ([]store.StatsByPeriod, error)
	GetStatsByClassification(ctx context.Context) ([]store.StatsByClassification, error)
	GetStatsBySender(ctx context.Context, limit int) ([]store.StatsBySender, error)
	GetStatsStaffLoad(ctx context.Context) ([]store.StatsStaffLoad, error)
}

type statsByPeriodDTO struct {
	Bucket      string         `json:"bucket"`
	JenisCount  map[string]int `json:"jenis_count"`
}

type statsByClassDTO struct {
	KlasifikasiKode *string `json:"klasifikasi_kode,omitempty"`
	KlasifikasiNama *string `json:"klasifikasi_nama,omitempty"`
	Count           int     `json:"count"`
}

type statsBySenderDTO struct {
	InstansiID   string `json:"instansi_id"`
	InstansiNama string `json:"instansi_nama"`
	Count        int    `json:"count"`
}

type statsStaffLoadDTO struct {
	UserID       string         `json:"user_id"`
	FullName     string         `json:"full_name"`
	StatusCount  map[string]int `json:"status_count"`
	OverdueCount int            `json:"overdue_count"`
	TotalActive  int            `json:"total_active"`
}

// requireSupervisor — stats endpoints camat/admin only. Staf bisa lihat
// disposisi mine sendiri lewat /api/disposisi?mine=true, tapi reporting
// agregat untuk supervisor.
func requireSupervisor(roles []string) bool {
	for _, r := range roles {
		if r == "camat" || r == "admin" {
			return true
		}
	}
	return false
}

func statsByPeriodHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}
		if !requireSupervisor(claims.Roles) {
			writeError(w, http.StatusForbidden, "stats hanya untuk role camat/admin")
			return
		}

		var from, to *time.Time
		q := r.URL.Query()
		if v := q.Get("from"); v != "" {
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				writeError(w, http.StatusBadRequest, "from format invalid (YYYY-MM-DD)")
				return
			}
			from = &t
		}
		if v := q.Get("to"); v != "" {
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				writeError(w, http.StatusBadRequest, "to format invalid (YYYY-MM-DD)")
				return
			}
			to = &t
		}

		data, err := d.StatsStore.GetStatsByPeriod(r.Context(), from, to)
		if err != nil {
			d.Logger.Error("stats period", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		out := make([]statsByPeriodDTO, 0, len(data))
		for _, s := range data {
			out = append(out, statsByPeriodDTO{Bucket: s.Bucket, JenisCount: s.JenisCount})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out})
	}
}

func statsByClassificationHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}
		if !requireSupervisor(claims.Roles) {
			writeError(w, http.StatusForbidden, "stats hanya untuk role camat/admin")
			return
		}
		data, err := d.StatsStore.GetStatsByClassification(r.Context())
		if err != nil {
			d.Logger.Error("stats classification", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		out := make([]statsByClassDTO, 0, len(data))
		for _, s := range data {
			out = append(out, statsByClassDTO{
				KlasifikasiKode: s.KlasifikasiKode,
				KlasifikasiNama: s.KlasifikasiNama,
				Count:           s.Count,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out})
	}
}

func statsBySenderHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}
		if !requireSupervisor(claims.Roles) {
			writeError(w, http.StatusForbidden, "stats hanya untuk role camat/admin")
			return
		}
		limit := 10
		if v := r.URL.Query().Get("top"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				writeError(w, http.StatusBadRequest, "top harus integer positif")
				return
			}
			limit = n
		}
		data, err := d.StatsStore.GetStatsBySender(r.Context(), limit)
		if err != nil {
			d.Logger.Error("stats sender", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		out := make([]statsBySenderDTO, 0, len(data))
		for _, s := range data {
			out = append(out, statsBySenderDTO{
				InstansiID: s.InstansiID, InstansiNama: s.InstansiNama, Count: s.Count,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out})
	}
}

func statsStaffLoadHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}
		if !requireSupervisor(claims.Roles) {
			writeError(w, http.StatusForbidden, "stats hanya untuk role camat/admin")
			return
		}
		data, err := d.StatsStore.GetStatsStaffLoad(r.Context())
		if err != nil {
			d.Logger.Error("stats staff load", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		out := make([]statsStaffLoadDTO, 0, len(data))
		for _, s := range data {
			out = append(out, statsStaffLoadDTO{
				UserID: s.UserID, FullName: s.FullName,
				StatusCount: s.StatusCount, OverdueCount: s.OverdueCount,
				TotalActive: s.TotalActive,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out})
	}
}
