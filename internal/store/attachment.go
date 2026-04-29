package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// AttachmentInput parameter untuk insert attachment baru.
type AttachmentInput struct {
	ID         string // UUIDv7
	SuratID    string
	Role       string // "primary" | "lampiran"
	FileName   string
	FilePath   string // relatif ke storage root
	FileSize   int64
	MimeType   string
	UploadedBy string
}

// AddAttachment insert single attachment row dengan is_active=TRUE.
// Return ErrNotFound kalau surat tidak ada atau soft-deleted.
func (s *Store) AddAttachment(ctx context.Context, in AttachmentInput) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Verifikasi surat exist + tidak deleted
	var dummy string
	err = tx.QueryRow(ctx, `SELECT id::text FROM surat WHERE id = $1 AND NOT is_deleted`, in.SuratID).Scan(&dummy)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: check surat: %w", err)
	}

	const q = `
		INSERT INTO surat_attachments (id, surat_id, role, file_name, file_path,
		                                file_size, mime_type, is_active, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, $8)`
	_, err = tx.Exec(ctx, q,
		in.ID, in.SuratID, in.Role, in.FileName, in.FilePath,
		in.FileSize, in.MimeType, in.UploadedBy)
	if err != nil {
		return fmt.Errorf("store: insert attachment: %w", err)
	}

	return tx.Commit(ctx)
}

// AttachmentByID return single attachment dengan flag is_active dan path.
// Caller bisa pakai untuk download.
func (s *Store) AttachmentByID(ctx context.Context, id string) (*AttachmentInput, error) {
	const q = `
		SELECT id::text, surat_id::text, role, file_name, file_path,
		       file_size, mime_type
		FROM surat_attachments
		WHERE id = $1 AND is_active`
	var a AttachmentInput
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&a.ID, &a.SuratID, &a.Role, &a.FileName, &a.FilePath,
		&a.FileSize, &a.MimeType)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: get attachment: %w", err)
	}
	return &a, nil
}
