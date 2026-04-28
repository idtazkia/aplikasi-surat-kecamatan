package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	ListenAddr  string
	LogLevel    string
	StudentMode bool
}

func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		ListenAddr:  os.Getenv("LISTEN_ADDR"),
		LogLevel:    os.Getenv("LOG_LEVEL"),
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

	return cfg, nil
}
