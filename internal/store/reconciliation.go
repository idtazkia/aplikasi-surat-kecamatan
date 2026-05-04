package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ReconciliationGroup = grup pending dedup yang berisi 2+ surat dengan
// dedup_key sama. Group_id stable — tetap meskipun salah satu surat
// di-merge keluar (record di queue tetap untuk audit).
type ReconciliationGroup struct {
	GroupID    string
	DedupKey   string
	Status     string // "pending" | "merged" | "kept_both"
	CreatedAt  time.Time
	ResolvedAt *time.Time
	ResolvedBy *string
	SuratCount int
	// Summary fields untuk list view (dari salah satu surat di group)
	InstansiNama string
	NomorSurat   string
	TanggalTerima *time.Time
}

// ReconciliationDetail = grup dengan list lengkap surat di dalamnya.
type ReconciliationDetail struct {
	GroupID   string
	DedupKey  string
	Status    string
	CreatedAt time.Time
	Surats    []SuratDetail // Full surat dengan attachment + reference
}

// CreateReconciliationGroupIfDuplicate — setelah CreateSurat, panggil ini
// untuk detect duplikat. Kalau ada surat lain dengan dedup_key sama:
//
//  1. Buat group_id baru (atau reuse kalau yang lama sudah ada)
//  2. Insert kedua surat ke reconciliation_queue
//
// Return groupID yang dibuat, atau empty string kalau tidak duplicate.
func (s *Store) CreateReconciliationGroupIfDuplicate(
	ctx context.Context,
	suratID string,
) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("store: begin recon tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Ambil dedup key surat yang baru di-create
	var jenis, nomorSurat, instansiID string
	var tanggalTerima *time.Time
	err = tx.QueryRow(ctx, `
		SELECT jenis, nomor_surat, instansi_id::text, tanggal_terima
		FROM surat WHERE id = $1 AND NOT is_deleted`, suratID).Scan(
		&jenis, &nomorSurat, &instansiID, &tanggalTerima)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("store: get surat for dedup: %w", err)
	}

	// MVP scope: dedup hanya untuk surat masuk. Surat keluar sudah ada
	// UNIQUE constraint di nomor_surat — duplikat di-reject di insert level.
	if jenis != "masuk" || tanggalTerima == nil {
		return "", tx.Commit(ctx)
	}

	dedupKey := fmt.Sprintf("masuk:%s:%s:%s",
		instansiID, nomorSurat, tanggalTerima.Format("2006-01-02"))

	// Cari surat lain dengan dedup tuple sama (excluding sendiri)
	rows, err := tx.Query(ctx, `
		SELECT id::text FROM surat
		WHERE jenis = 'masuk' AND NOT is_deleted
		  AND instansi_id = $1::uuid AND nomor_surat = $2 AND tanggal_terima = $3
		  AND id <> $4`,
		instansiID, nomorSurat, *tanggalTerima, suratID)
	if err != nil {
		return "", fmt.Errorf("store: dedup lookup: %w", err)
	}
	var siblingIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return "", fmt.Errorf("store: scan sibling: %w", err)
		}
		siblingIDs = append(siblingIDs, id)
	}
	rows.Close()

	if len(siblingIDs) == 0 {
		return "", tx.Commit(ctx)
	}

	// Cek apakah salah satu sibling sudah ada di reconciliation_queue
	// dengan group_id existing — kalau ada, reuse group_id itu.
	var groupID string
	err = tx.QueryRow(ctx, `
		SELECT group_id::text FROM reconciliation_queue
		WHERE surat_id = ANY($1::uuid[])
		  AND status = 'pending'
		LIMIT 1`, siblingIDs).Scan(&groupID)
	if errors.Is(err, pgx.ErrNoRows) {
		// Generate new group_id
		err = tx.QueryRow(ctx, `SELECT gen_random_uuid()::text`).Scan(&groupID)
		if err != nil {
			return "", fmt.Errorf("store: gen group_id: %w", err)
		}
	} else if err != nil {
		return "", fmt.Errorf("store: lookup existing group: %w", err)
	}

	// Insert entry untuk surat baru + sibling yang belum di-queue
	const insertQ = `
		INSERT INTO reconciliation_queue (id, group_id, surat_id, dedup_key, status)
		VALUES (gen_random_uuid(), $1::uuid, $2::uuid, $3, 'pending')
		ON CONFLICT DO NOTHING`
	_, err = tx.Exec(ctx, insertQ, groupID, suratID, dedupKey)
	if err != nil {
		return "", fmt.Errorf("store: insert recon (new): %w", err)
	}
	for _, sib := range siblingIDs {
		// Skip kalau sibling sudah di-queue
		var existed bool
		err = tx.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM reconciliation_queue
			  WHERE surat_id = $1::uuid AND status = 'pending')`, sib).Scan(&existed)
		if err != nil {
			return "", fmt.Errorf("store: check sibling queued: %w", err)
		}
		if existed {
			continue
		}
		_, err = tx.Exec(ctx, insertQ, groupID, sib, dedupKey)
		if err != nil {
			return "", fmt.Errorf("store: insert recon (sibling): %w", err)
		}
	}

	return groupID, tx.Commit(ctx)
}

// ListReconciliationGroups untuk halaman antrian. Aggregate per group_id
// dengan count + sample summary fields.
func (s *Store) ListReconciliationGroups(ctx context.Context, includeResolved bool) ([]ReconciliationGroup, error) {
	statusFilter := "AND rq.status = 'pending'"
	if includeResolved {
		statusFilter = ""
	}
	q := fmt.Sprintf(`
		SELECT rq.group_id::text, rq.dedup_key, rq.status,
		       MIN(rq.created_at) as created_at,
		       MAX(rq.resolved_at) as resolved_at,
		       MAX(rq.resolved_by::text) as resolved_by,
		       COUNT(*) as surat_count,
		       MAX(i.nama_kanonik) as instansi_nama,
		       MAX(s.nomor_surat) as nomor_surat,
		       MAX(s.tanggal_terima) as tanggal_terima
		FROM reconciliation_queue rq
		JOIN surat s ON s.id = rq.surat_id
		JOIN instansi i ON i.id = s.instansi_id
		WHERE TRUE %s
		GROUP BY rq.group_id, rq.dedup_key, rq.status
		ORDER BY MIN(rq.created_at) DESC`, statusFilter)

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("store: list recon groups: %w", err)
	}
	defer rows.Close()

	var out []ReconciliationGroup
	for rows.Next() {
		var g ReconciliationGroup
		if err := rows.Scan(
			&g.GroupID, &g.DedupKey, &g.Status,
			&g.CreatedAt, &g.ResolvedAt, &g.ResolvedBy,
			&g.SuratCount, &g.InstansiNama, &g.NomorSurat, &g.TanggalTerima,
		); err != nil {
			return nil, fmt.Errorf("store: scan recon group: %w", err)
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// GetReconciliationDetail untuk merge UI side-by-side. Return semua surat
// di group dengan full detail (attachments + references).
func (s *Store) GetReconciliationDetail(ctx context.Context, groupID string) (*ReconciliationDetail, error) {
	const headerQ = `
		SELECT MIN(dedup_key), MAX(status), MIN(created_at)
		FROM reconciliation_queue WHERE group_id = $1::uuid`
	var d ReconciliationDetail
	d.GroupID = groupID
	err := s.pool.QueryRow(ctx, headerQ, groupID).Scan(&d.DedupKey, &d.Status, &d.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) || d.DedupKey == "" {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: recon header: %w", err)
	}

	// Ambil daftar surat_id di group
	rows, err := s.pool.Query(ctx, `
		SELECT surat_id::text FROM reconciliation_queue
		WHERE group_id = $1::uuid ORDER BY created_at`, groupID)
	if err != nil {
		return nil, fmt.Errorf("store: recon surat list: %w", err)
	}
	var suratIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("store: scan surat id: %w", err)
		}
		suratIDs = append(suratIDs, id)
	}
	rows.Close()

	// Fetch full SuratDetail untuk masing-masing
	for _, id := range suratIDs {
		s, err := s.GetSuratByID(ctx, id)
		if err != nil {
			// Surat sudah di-soft-delete (mis. dari merge sebelumnya) — skip
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, fmt.Errorf("store: recon get surat %s: %w", id, err)
		}
		d.Surats = append(d.Surats, *s)
	}
	return &d, nil
}

// MergeReconciliationGroup pilih kanonik surat. Surat yang lain di-soft-delete.
// Status group → "merged", resolved_at + resolved_by ter-set.
//
// canonicalSuratID harus salah satu yang ada di group. Kalau bukan: error.
func (s *Store) MergeReconciliationGroup(
	ctx context.Context,
	groupID, canonicalSuratID, resolvedBy string,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store: merge tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Ambil semua surat di group
	rows, err := tx.Query(ctx, `
		SELECT surat_id::text FROM reconciliation_queue
		WHERE group_id = $1::uuid AND status = 'pending'`, groupID)
	if err != nil {
		return fmt.Errorf("store: list group surat: %w", err)
	}
	var suratIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("store: scan group surat: %w", err)
		}
		suratIDs = append(suratIDs, id)
	}
	rows.Close()

	if len(suratIDs) == 0 {
		return ErrNotFound
	}

	// Verify canonical ada di list
	canonicalFound := false
	for _, id := range suratIDs {
		if id == canonicalSuratID {
			canonicalFound = true
			break
		}
	}
	if !canonicalFound {
		return fmt.Errorf("canonical surat %s tidak ada di group %s", canonicalSuratID, groupID)
	}

	// Soft delete sisanya
	for _, id := range suratIDs {
		if id == canonicalSuratID {
			continue
		}
		_, err = tx.Exec(ctx, `
			UPDATE surat SET is_deleted = TRUE, deleted_at = NOW(), deleted_by = $1::uuid,
			                 updated_at = NOW()
			WHERE id = $2::uuid AND NOT is_deleted`, resolvedBy, id)
		if err != nil {
			return fmt.Errorf("store: soft delete loser %s: %w", id, err)
		}
	}

	// Mark group merged
	_, err = tx.Exec(ctx, `
		UPDATE reconciliation_queue
		SET status = 'merged', resolved_by = $1::uuid, resolved_at = NOW()
		WHERE group_id = $2::uuid AND status = 'pending'`, resolvedBy, groupID)
	if err != nil {
		return fmt.Errorf("store: mark merged: %w", err)
	}

	return tx.Commit(ctx)
}

// KeepBothReconciliationGroup tandai group sebagai "kept_both" — kedua surat
// tetap aktif (mis. ternyata bukan duplikat sebenarnya, hanya kebetulan
// kunci dedup sama).
func (s *Store) KeepBothReconciliationGroup(
	ctx context.Context,
	groupID, resolvedBy string,
) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE reconciliation_queue
		SET status = 'kept_both', resolved_by = $1::uuid, resolved_at = NOW()
		WHERE group_id = $2::uuid AND status = 'pending'`, resolvedBy, groupID)
	if err != nil {
		return fmt.Errorf("store: keep both: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
