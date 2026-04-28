// Package store membungkus akses database via pgx.
//
// Catatan untuk Fase 0: query masih ditulis raw dengan pgx. sqlc generation
// direncanakan tapi belum di-wire (CGO build issue di sebagian environment macOS).
// Saat sqlc resolved (Fase 1), file ini akan jadi thin wrapper di atas
// generated code di sub-package store/queries.
package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound dikembalikan saat query single-row tidak match apapun.
var ErrNotFound = errors.New("store: not found")

// Store agregasi semua repository (Fase 0: minimal).
type Store struct {
	pool *pgxpool.Pool
}

// New membuat Store dengan pool yang sudah di-init oleh caller.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Ping mengembalikan error kalau koneksi DB tidak sehat. Dipakai /healthz.
func (s *Store) Ping(ctx context.Context) error {
	if err := s.pool.Ping(ctx); err != nil {
		return fmt.Errorf("store: ping: %w", err)
	}
	return nil
}

// UserForLogin adalah projection user yang dipakai login flow.
type UserForLogin struct {
	ID           string
	Username     string
	PasswordHash string
	IsActive     bool
	Roles        []string
}

// concept:sql-aggregation-array-agg:start
// GetUserForLogin lookup user by username + roles via array_agg.
// Tunggal query, tidak N+1: roles di-aggregate di DB lewat LEFT JOIN + array_agg
// dengan FILTER clause untuk handle user tanpa role (FILTER hapus NULL dari array).
func (s *Store) GetUserForLogin(ctx context.Context, username string) (*UserForLogin, error) {
	const q = `
		SELECT u.id::text, u.username, u.password_hash, u.is_active,
		       COALESCE(array_agg(r.code) FILTER (WHERE r.code IS NOT NULL), '{}') AS roles
		FROM users u
		LEFT JOIN user_roles ur ON ur.user_id = u.id
		LEFT JOIN roles r ON r.id = ur.role_id
		WHERE u.username = $1
		GROUP BY u.id`

	var u UserForLogin
	err := s.pool.QueryRow(ctx, q, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.IsActive, &u.Roles)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: get user: %w", err)
	}
	return &u, nil
}

// concept:sql-aggregation-array-agg:end
