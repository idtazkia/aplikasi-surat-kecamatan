package config

import (
	"strings"
	"testing"
)

func TestLoad_MissingRequired(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("LISTEN_ADDR", "")
	t.Setenv("LOG_LEVEL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing env vars, got nil")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("expected DATABASE_URL in missing list, got %v", err)
	}
}

func TestLoad_AllPresent(t *testing.T) {
	setBaseEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DatabaseURL != "postgres://x" {
		t.Errorf("DatabaseURL mismatch")
	}
	if cfg.StudentMode {
		t.Errorf("StudentMode should be false")
	}
	if cfg.Tenant.AppName != "Aplikasi Test" {
		t.Errorf("Tenant.AppName mismatch")
	}
}

func TestLoad_TenantMissing(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("LISTEN_ADDR", ":8080")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("STUDENT_MODE_ENABLED", "false")
	t.Setenv("ATTACHMENT_STORAGE_PATH", "/tmp/test-attach")
	// Sengaja tidak set TENANT_*

	_, err := Load()
	if err == nil {
		t.Fatal("expected error untuk missing TENANT_* env, got nil")
	}
	if !strings.Contains(err.Error(), "TENANT_APP_NAME") {
		t.Errorf("expected TENANT_APP_NAME di error, got %v", err)
	}
}

func TestLoad_TenantInvalidColor(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("TENANT_BRANDING_PRIMARY", "blue") // bukan hex

	_, err := Load()
	if err == nil {
		t.Fatal("expected error untuk invalid hex color")
	}
	if !strings.Contains(err.Error(), "hex color") {
		t.Errorf("expected hex error message, got %v", err)
	}
}

// setBaseEnv set semua env required untuk load berhasil — base case yang
// dipakai test lain untuk override sebagian saja.
func setBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("LISTEN_ADDR", ":8080")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("STUDENT_MODE_ENABLED", "false")
	t.Setenv("ATTACHMENT_STORAGE_PATH", "/tmp/test-attach")
	t.Setenv("TENANT_APP_NAME", "Aplikasi Test")
	t.Setenv("TENANT_INSTITUTION_NAME", "Kantor Test")
	t.Setenv("TENANT_BRANDING_PRIMARY", "#204397")
	t.Setenv("TENANT_BRANDING_PRIMARY_HOVER", "#1a3680")
	t.Setenv("TENANT_BRANDING_ACCENT", "#c8475b")
}

func TestLoad_InvalidStudentMode(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("LISTEN_ADDR", ":8080")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("STUDENT_MODE_ENABLED", "yes")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid STUDENT_MODE_ENABLED")
	}
}
