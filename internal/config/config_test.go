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
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("LISTEN_ADDR", ":8080")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("STUDENT_MODE_ENABLED", "false")

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
