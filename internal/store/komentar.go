package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// concept:append-only-immutability:start
// Komentar = catatan append-only di surat. Tidak ada update/delete.
// Kalau salah ketik, append entry koreksi baru. Audit by construction —
// tidak perlu audit_log terpisah untuk komentar.
type Komentar struct {
	ID         string
	SuratID    string
	UserID     string
	UserName   string
	Body       string
	CreatedAt  time.Time
}

// AppendKomentarInput parameter untuk insert komentar baru.
type AppendKomentarInput struct {
	ID      string // UUIDv7 caller-generated
	SuratID string
	UserID  string
	Body    string
}

// AppendKomentar insert komentar. Verify surat exist + tidak deleted.
// Tidak ada update/delete — kalau salah, append koreksi baru.
func (s *Store) AppendKomentar(ctx context.Context, in AppendKomentarInput) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var dummy string
	err = tx.QueryRow(ctx, `SELECT id::text FROM surat WHERE id = $1 AND NOT is_deleted`, in.SuratID).Scan(&dummy)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: check surat: %w", err)
	}

	const q = `
		INSERT INTO komentar (id, surat_id, user_id, body)
		VALUES ($1, $2, $3, $4)`
	_, err = tx.Exec(ctx, q, in.ID, in.SuratID, in.UserID, in.Body)
	if err != nil {
		return fmt.Errorf("store: insert komentar: %w", err)
	}

	return tx.Commit(ctx)
}

// ListKomentarBySurat return semua komentar untuk satu surat, sorted by created_at ASC.
// Order ASC karena reading thread mengikuti urutan diskusi.
func (s *Store) ListKomentarBySurat(ctx context.Context, suratID string) ([]Komentar, error) {
	const q = `
		SELECT k.id::text, k.surat_id::text, k.user_id::text, u.full_name,
		       k.body, k.created_at
		FROM komentar k
		JOIN users u ON u.id = k.user_id
		WHERE k.surat_id = $1
		ORDER BY k.created_at ASC, k.id ASC`

	rows, err := s.pool.Query(ctx, q, suratID)
	if err != nil {
		return nil, fmt.Errorf("store: list komentar: %w", err)
	}
	defer rows.Close()

	var out []Komentar
	for rows.Next() {
		var k Komentar
		if err := rows.Scan(&k.ID, &k.SuratID, &k.UserID, &k.UserName, &k.Body, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan komentar: %w", err)
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// concept:append-only-immutability:end
