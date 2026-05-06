package config

import (
	"fmt"
	"os"
	"regexp"
)

var hexColorRE = regexp.MustCompile(`^#[0-9a-fA-F]{3,8}$`)

type Config struct {
	DatabaseURL           string
	JWTSecret             string
	ListenAddr            string
	LogLevel              string
	StudentMode           bool
	AttachmentStoragePath string
	Tenant                TenantConfig
}

// TenantConfig — branding + identitas instansi yang berbeda per kecamatan
// di multi-instance deployment. Di-serve via GET /api/config untuk frontend
// + accessible langsung dari backend untuk future watermark/email subject/etc.
//
// Validasi regex untuk hex color karena frontend pakai langsung di Naive UI
// theme override — invalid color akan break UI tanpa error jelas.
type TenantConfig struct {
	AppName             string
	InstitutionName     string
	BrandingLogoURL     string // optional, boleh kosong
	BrandingPrimary     string // hex
	BrandingPrimaryHover string // hex
	BrandingAccent      string // hex
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:           os.Getenv("DATABASE_URL"),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		ListenAddr:            os.Getenv("LISTEN_ADDR"),
		LogLevel:              os.Getenv("LOG_LEVEL"),
		AttachmentStoragePath: os.Getenv("ATTACHMENT_STORAGE_PATH"),
	}

	var missing []string
	if cfg.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if cfg.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if cfg.ListenAddr == "" {
		missing = append(missing, "LISTEN_ADDR")
	}
	if cfg.LogLevel == "" {
		missing = append(missing, "LOG_LEVEL")
	}
	if cfg.AttachmentStoragePath == "" {
		missing = append(missing, "ATTACHMENT_STORAGE_PATH")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %v", missing)
	}

	switch os.Getenv("STUDENT_MODE_ENABLED") {
	case "true":
		cfg.StudentMode = true
	case "false", "":
		cfg.StudentMode = false
	default:
		return nil, fmt.Errorf("STUDENT_MODE_ENABLED must be 'true' or 'false', got %q", os.Getenv("STUDENT_MODE_ENABLED"))
	}

	cfg.Tenant = TenantConfig{
		AppName:              os.Getenv("TENANT_APP_NAME"),
		InstitutionName:      os.Getenv("TENANT_INSTITUTION_NAME"),
		BrandingLogoURL:      os.Getenv("TENANT_BRANDING_LOGO_URL"), // optional
		BrandingPrimary:      os.Getenv("TENANT_BRANDING_PRIMARY"),
		BrandingPrimaryHover: os.Getenv("TENANT_BRANDING_PRIMARY_HOVER"),
		BrandingAccent:       os.Getenv("TENANT_BRANDING_ACCENT"),
	}
	if cfg.Tenant.AppName == "" {
		missing = append(missing, "TENANT_APP_NAME")
	}
	if cfg.Tenant.InstitutionName == "" {
		missing = append(missing, "TENANT_INSTITUTION_NAME")
	}
	for name, val := range map[string]string{
		"TENANT_BRANDING_PRIMARY":       cfg.Tenant.BrandingPrimary,
		"TENANT_BRANDING_PRIMARY_HOVER": cfg.Tenant.BrandingPrimaryHover,
		"TENANT_BRANDING_ACCENT":        cfg.Tenant.BrandingAccent,
	} {
		if val == "" {
			missing = append(missing, name)
		} else if !hexColorRE.MatchString(val) {
			return nil, fmt.Errorf("%s harus hex color (mis. '#204397'), got %q", name, val)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required env vars: %v", missing)
	}

	return cfg, nil
}
