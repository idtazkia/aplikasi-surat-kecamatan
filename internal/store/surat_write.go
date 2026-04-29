package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// CreateSuratInput parameter untuk insert surat baru.
// ID harus di-generate caller (UUIDv7).
type CreateSuratInput struct {
	ID            string
	Jenis         string
	NomorSurat    string
	Perihal       string
	TanggalSurat  time.Time
	TanggalTerima *time.Time
	InstansiID    string
	KlasifikasiID *string
	SifatID       *string
	AccessLevel   string
	CreatedBy     string
}

// CreateSurat insert surat + record audit log dalam single transaction.
// Return ErrConflict kalau surat keluar dengan nomor_surat sama sudah ada.
func (s *Store) CreateSurat(ctx context.Context, in CreateSuratInput) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO surat (id, jenis, nomor_surat, perihal, tanggal_surat, tanggal_terima,
		                   instansi_id, klasifikasi_id, sifat_id, access_level, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err = tx.Exec(ctx, q,
		in.ID, in.Jenis, in.NomorSurat, in.Perihal, in.TanggalSurat, in.TanggalTerima,
		in.InstansiID, in.KlasifikasiID, in.SifatID, in.AccessLevel, in.CreatedBy)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return fmt.Errorf("store: insert surat: %w", err)
	}

	after, _ := json.Marshal(in)
	if err := s.recordAudit(ctx, tx, "surat", in.ID, "create", in.CreatedBy, nil, after); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UpdateSuratInput partial update — nil field = no change.
type UpdateSuratInput struct {
	NomorSurat    *string
	Perihal       *string
	TanggalSurat  *time.Time
	TanggalTerima *time.Time
	InstansiID    *string
	KlasifikasiID *string
	SifatID       *string
	AccessLevel   *string
	UpdatedBy     string
}

// UpdateSurat partial update + record before/after audit log.
// Return ErrNotFound kalau surat tidak ada atau soft-deleted.
func (s *Store) UpdateSurat(ctx context.Context, id string, in UpdateSuratInput) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Fetch before-state untuk audit
	before, err := fetchSuratSnapshot(ctx, tx, id)
	if err != nil {
		return err
	}

	// Build dynamic SET clause
	sets := []string{"updated_at = NOW()"}
	var args []interface{}
	addArg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if in.NomorSurat != nil {
		sets = append(sets, "nomor_surat = "+addArg(*in.NomorSurat))
	}
	if in.Perihal != nil {
		sets = append(sets, "perihal = "+addArg(*in.Perihal))
	}
	if in.TanggalSurat != nil {
		sets = append(sets, "tanggal_surat = "+addArg(*in.TanggalSurat))
	}
	if in.TanggalTerima != nil {
		sets = append(sets, "tanggal_terima = "+addArg(*in.TanggalTerima))
	}
	if in.InstansiID != nil {
		sets = append(sets, "instansi_id = "+addArg(*in.InstansiID))
	}
	if in.KlasifikasiID != nil {
		sets = append(sets, "klasifikasi_id = "+addArg(*in.KlasifikasiID))
	}
	if in.SifatID != nil {
		sets = append(sets, "sifat_id = "+addArg(*in.SifatID))
	}
	if in.AccessLevel != nil {
		sets = append(sets, "access_level = "+addArg(*in.AccessLevel))
	}

	if len(sets) == 1 {
		// No changes other than updated_at — caller passed empty patch
		return tx.Commit(ctx)
	}

	q := fmt.Sprintf("UPDATE surat SET %s WHERE id = %s AND NOT is_deleted",
		strings.Join(sets, ", "), addArg(id))
	tag, err := tx.Exec(ctx, q, args...)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return fmt.Errorf("store: update surat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	after, err := fetchSuratSnapshot(ctx, tx, id)
	if err != nil {
		return err
	}

	if err := s.recordAudit(ctx, tx, "surat", id, "update", in.UpdatedBy, before, after); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// SoftDeleteSurat set is_deleted=true. Idempotent — already deleted return nil.
func (s *Store) SoftDeleteSurat(ctx context.Context, id, actorID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	before, err := fetchSuratSnapshot(ctx, tx, id)
	if err != nil {
		return err
	}

	const q = `
		UPDATE surat
		SET is_deleted = TRUE, deleted_at = NOW(), deleted_by = $2, updated_at = NOW()
		WHERE id = $1 AND NOT is_deleted`
	tag, err := tx.Exec(ctx, q, id, actorID)
	if err != nil {
		return fmt.Errorf("store: delete surat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	after, err := fetchSuratSnapshot(ctx, tx, id)
	if err != nil {
		return err
	}

	if err := s.recordAudit(ctx, tx, "surat", id, "delete", actorID, before, after); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// RestoreSurat undo soft delete. Hanya admin yang harus pakai (di handler layer).
func (s *Store) RestoreSurat(ctx context.Context, id, actorID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Fetch including deleted
	before, err := fetchSuratSnapshotIncludeDeleted(ctx, tx, id)
	if err != nil {
		return err
	}

	const q = `
		UPDATE surat
		SET is_deleted = FALSE, deleted_at = NULL, deleted_by = NULL, updated_at = NOW()
		WHERE id = $1 AND is_deleted`
	tag, err := tx.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("store: restore surat: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	after, err := fetchSuratSnapshotIncludeDeleted(ctx, tx, id)
	if err != nil {
		return err
	}

	if err := s.recordAudit(ctx, tx, "surat", id, "restore", actorID, before, after); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// recordAudit insert audit_log entry. Caller harus provide tx — dijalankan dalam
// transaction yang sama dengan write supaya atomically commit/rollback bersama.
func (s *Store) recordAudit(ctx context.Context, tx pgx.Tx, entityType, entityID, action, actorID string, before, after []byte) error {
	const q = `
		INSERT INTO audit_log (id, entity_type, entity_id, action, actor_user_id, before_jsonb, after_jsonb)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)`
	_, err := tx.Exec(ctx, q, entityType, entityID, action, actorID, before, after)
	if err != nil {
		return fmt.Errorf("store: record audit: %w", err)
	}
	return nil
}

// fetchSuratSnapshot ambil row-as-JSONB untuk audit before/after.
// Return nil tanpa error kalau row tidak ada — caller harus check tag separately.
func fetchSuratSnapshot(ctx context.Context, tx pgx.Tx, id string) ([]byte, error) {
	const q = `SELECT to_jsonb(s) FROM surat s WHERE s.id = $1 AND NOT s.is_deleted`
	var data []byte
	err := tx.QueryRow(ctx, q, id).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: snapshot: %w", err)
	}
	return data, nil
}

func fetchSuratSnapshotIncludeDeleted(ctx context.Context, tx pgx.Tx, id string) ([]byte, error) {
	const q = `SELECT to_jsonb(s) FROM surat s WHERE s.id = $1`
	var data []byte
	err := tx.QueryRow(ctx, q, id).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: snapshot: %w", err)
	}
	return data, nil
}

// isUniqueViolation check pgx error untuk unique constraint code 23505.
func isUniqueViolation(err error) bool {
	type pgError interface {
		SQLState() string
	}
	var pe pgError
	if errors.As(err, &pe) {
		return pe.SQLState() == "23505"
	}
	return false
}

// ErrConflict dikembalikan saat unique constraint violated (mis. nomor_surat keluar duplikat).
var ErrConflict = errors.New("store: conflict")
