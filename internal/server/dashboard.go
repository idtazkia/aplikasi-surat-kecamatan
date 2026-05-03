package server

import (
	"context"
	"net/http"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
)

// DashboardStore subset interface untuk dashboard handlers.
type DashboardStore interface {
	GetDashboardCamat(ctx context.Context, camatUserID string) (*store.DashboardCamat, error)
}

type dashboardCamatDTO struct {
	SuratMasukHariIni     int `json:"surat_masuk_hari_ini"`
	DisposisiBelumAssign  int `json:"disposisi_belum_assign"`
	DisposisiOverdue      int `json:"disposisi_overdue"`
	DisposisiAssignedToMe int `json:"disposisi_assigned_to_me"`
}

func dashboardCamatHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		// Hanya camat dan admin yang punya akses dashboard supervisi.
		if !hasRole(claims.Roles, "camat") && !hasRole(claims.Roles, "admin") {
			writeError(w, http.StatusForbidden, "dashboard hanya untuk role camat/admin")
			return
		}

		dc, err := d.DashboardStore.GetDashboardCamat(r.Context(), claims.Sub)
		if err != nil {
			d.Logger.Error("dashboard: query", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		writeJSON(w, http.StatusOK, dashboardCamatDTO{
			SuratMasukHariIni:     dc.SuratMasukHariIni,
			DisposisiBelumAssign:  dc.DisposisiBelumAssign,
			DisposisiOverdue:      dc.DisposisiOverdue,
			DisposisiAssignedToMe: dc.DisposisiAssignedToMe,
		})
	}
}

