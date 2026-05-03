package store

import (
	"context"
	"errors"
	"fmt"
	"time"

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

// concept:linked-list-version-chain:start
// ReplaceAttachment = atomic: insert new attachment + mark old is_active=FALSE +
// set old.replaced_by=new.id. Old surat_id harus sama dengan new (defensif),
// dan old harus belum direplace.
// Linked list update: old.replaced_by → new (next pointer), new.replaced_by=NULL (tail).
func (s *Store) ReplaceAttachment(ctx context.Context, oldID string, in AttachmentInput) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock old row + verify state
	var oldSuratID, oldRole string
	var oldActive bool
	err = tx.QueryRow(ctx, `
		SELECT surat_id::text, role, is_active
		FROM surat_attachments
		WHERE id = $1
		FOR UPDATE`, oldID).Scan(&oldSuratID, &oldRole, &oldActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("store: lock old attachment: %w", err)
	}
	if !oldActive {
		return ErrAlreadyReplaced
	}
	if oldSuratID != in.SuratID {
		return fmt.Errorf("store: surat_id mismatch (old=%s new=%s)", oldSuratID, in.SuratID)
	}
	// Role harus konsisten — replace primary→primary, lampiran→lampiran
	if oldRole != in.Role {
		return fmt.Errorf("store: role mismatch (old=%s new=%s)", oldRole, in.Role)
	}

	// Insert new attachment as active tail
	const insertQ = `
		INSERT INTO surat_attachments (id, surat_id, role, file_name, file_path,
		                                file_size, mime_type, is_active, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, $8)`
	_, err = tx.Exec(ctx, insertQ,
		in.ID, in.SuratID, in.Role, in.FileName, in.FilePath,
		in.FileSize, in.MimeType, in.UploadedBy)
	if err != nil {
		return fmt.Errorf("store: insert new version: %w", err)
	}

	// Patch old: replaced_by=new.id, is_active=FALSE
	const updateQ = `
		UPDATE surat_attachments
		SET is_active = FALSE, replaced_by = $1
		WHERE id = $2`
	_, err = tx.Exec(ctx, updateQ, in.ID, oldID)
	if err != nil {
		return fmt.Errorf("store: deactivate old: %w", err)
	}

	return tx.Commit(ctx)
}

// AttachmentVersion = entry di chain version (dari head ke tail).
type AttachmentVersion struct {
	ID           string
	FileName     string
	FileSize     int64
	MimeType     string
	IsActive     bool
	ReplacedBy   *string // null kalau tail
	UploadedBy   string
	UploaderName string
	UploadedAt   time.Time
}

// ListAttachmentVersions traverse linked list mundur dari ID yang diberikan
// kembali ke head (oldest), lalu reverse sehingga return order = oldest → newest.
// Implementasi: ambil tail (id given OR follow replaced_by chain to active),
// kemudian recursive CTE traversal mundur via `replaced_by` reverse.
func (s *Store) ListAttachmentVersions(ctx context.Context, anyVersionID string) ([]AttachmentVersion, error) {
	// Step 1: cari root (head) chain — node yang tidak ada predecessor.
	// Pakai recursive CTE: walk dari given ID, jump ke node yang punya
	// replaced_by = current.id (predecessor); berhenti saat tidak ada predecessor.
	const q = `
		WITH RECURSIVE chain AS (
			-- Anchor: row given
			SELECT id, surat_id, file_name, file_size, mime_type, is_active,
			       replaced_by, uploaded_by, uploaded_at, 0 as depth
			FROM surat_attachments
			WHERE id = $1
			UNION ALL
			-- Jump ke predecessor (row yang replaced_by = current.id)
			SELECT prev.id, prev.surat_id, prev.file_name, prev.file_size, prev.mime_type,
			       prev.is_active, prev.replaced_by, prev.uploaded_by, prev.uploaded_at,
			       c.depth + 1
			FROM surat_attachments prev
			JOIN chain c ON prev.replaced_by = c.id
			WHERE c.depth < 100  -- safety: max 100 versions
		),
		full_chain AS (
			-- Setelah dapat root, walk forward (replaced_by chain) sampai tail
			SELECT id, file_name, file_size, mime_type, is_active,
			       replaced_by::text, uploaded_by, uploaded_at, 0 as fdepth
			FROM surat_attachments
			WHERE id = (SELECT id FROM chain ORDER BY depth DESC LIMIT 1)
			UNION ALL
			SELECT next.id, next.file_name, next.file_size, next.mime_type, next.is_active,
			       next.replaced_by::text, next.uploaded_by, next.uploaded_at,
			       fc.fdepth + 1
			FROM surat_attachments next
			JOIN full_chain fc ON next.id = fc.replaced_by::uuid
			WHERE fc.fdepth < 100
		)
		SELECT fc.id::text, fc.file_name, fc.file_size, fc.mime_type, fc.is_active,
		       fc.replaced_by, fc.uploaded_by::text, u.full_name, fc.uploaded_at
		FROM full_chain fc
		JOIN users u ON u.id = fc.uploaded_by
		ORDER BY fc.fdepth ASC`

	rows, err := s.pool.Query(ctx, q, anyVersionID)
	if err != nil {
		return nil, fmt.Errorf("store: list versions: %w", err)
	}
	defer rows.Close()

	var out []AttachmentVersion
	for rows.Next() {
		var v AttachmentVersion
		if err := rows.Scan(&v.ID, &v.FileName, &v.FileSize, &v.MimeType, &v.IsActive,
			&v.ReplacedBy, &v.UploadedBy, &v.UploaderName, &v.UploadedAt); err != nil {
			return nil, fmt.Errorf("store: scan version: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// concept:linked-list-version-chain:end

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

// ErrAlreadyReplaced = old attachment sudah pernah direplace (tidak bisa
// direplace ulang — untuk replace versi terbaru, caller harus pakai ID
// versi terkini).
var ErrAlreadyReplaced = errors.New("store: attachment sudah direplace")
