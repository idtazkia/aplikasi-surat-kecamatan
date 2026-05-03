package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/config"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/server"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load failed", "err", err)
		os.Exit(1)
	}

	bootCtx, bootCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer bootCancel()

	pool, err := pgxpool.New(bootCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("db pool init failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(bootCtx); err != nil {
		logger.Error("db ping failed", "err", err)
		os.Exit(1)
	}

	authSvc, err := auth.NewService([]byte(cfg.JWTSecret), 1*time.Hour, 7*24*time.Hour)
	if err != nil {
		logger.Error("auth init failed", "err", err)
		os.Exit(1)
	}

	st := store.New(pool)

	handler := server.New(server.Deps{
		Logger:          logger,
		Auth:            authSvc,
		Store:           st,
		SuratStore:      st,
		AttachmentStore: st,
		ReferenceStore:  st,
		TembusanStore:   st,
		DisposisiStore:  st,
		KomentarStore:   st,
		DirektoriStore:  st,
		AttachmentRoot:  cfg.AttachmentStoragePath,
	})

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("server starting", "addr", cfg.ListenAddr, "student_mode", cfg.StudentMode)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown failed", "err", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}
