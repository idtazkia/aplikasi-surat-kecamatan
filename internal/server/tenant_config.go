package server

import (
	"net/http"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/config"
)

// tenantConfigDTO — shape JSON yang frontend konsumsi. Field nama camelCase
// supaya konsisten dengan TypeScript convention (frontend pakai langsung
// tanpa key remap).
type tenantConfigDTO struct {
	APIBaseURL      string                 `json:"apiBaseUrl"`
	AppName         string                 `json:"appName"`
	InstitutionName string                 `json:"institutionName"`
	Branding        tenantConfigBrandingDTO `json:"branding"`
}

type tenantConfigBrandingDTO struct {
	LogoURL      string `json:"logoUrl"`
	Primary      string `json:"primary"`
	PrimaryHover string `json:"primaryHover"`
	Accent       string `json:"accent"`
}

// tenantConfigHandler GET /api/config — public, no auth.
//
// Public karena: (1) loaded sebelum login, (2) tidak ada secret, (3) sama
// untuk semua user dalam satu tenant. Frontend fetch ini sebagai bagian
// dari boot sequence sebelum Vue mount.
func tenantConfigHandler(t config.TenantConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// apiBaseUrl hardcoded "/api" karena frontend selalu same-origin
		// di multi-instance deployment (nginx serve dist/ + reverse proxy
		// /api/* ke Go backend). Future: kalau split CDN/API, bisa expose
		// sebagai env tambahan.
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, tenantConfigDTO{
			APIBaseURL:      "/api",
			AppName:         t.AppName,
			InstitutionName: t.InstitutionName,
			Branding: tenantConfigBrandingDTO{
				LogoURL:      t.BrandingLogoURL,
				Primary:      t.BrandingPrimary,
				PrimaryHover: t.BrandingPrimaryHover,
				Accent:       t.BrandingAccent,
			},
		})
	}
}
