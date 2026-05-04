package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// OperationLog = catatan operasi dari klien yang sudah di-sync.
// PK = client_op_id supaya idempotency by-construction: insert ulang dengan
// client_op_id yang sama gagal pada PK constraint, klien aman retry.
// Concept anchor utama ada di schema (db/migrations/schema/0001_init.sql).
type OperationLog struct {
	ClientOpID      string
	UserID          string
	EntityType      string // "surat" | "komentar" | ...
	EntityID        string
	Action          string // "create" | "update" | "delete" | "append"
	FieldChanges   json.RawMessage
	ClientTimestamp time.Time
	AppliedAt       time.Time
}

// ApplyOperationInput parameter dari klien.
type ApplyOperationInput struct {
	ClientOpID      string
	EntityType      string
	EntityID        string
	Action          string
	FieldChanges    json.RawMessage
	ClientTimestamp time.Time
	UserID          string
}

// ApplyOperationResult per-op result yang di-return ke klien.
type ApplyOperationResult struct {
	ClientOpID string
	Status     string // "applied" | "duplicate" | "rejected"
	Reason     string // populated kalau rejected
}

// ApplyOperations terima batch ops dari klien. Untuk setiap op:
//
//  1. Cek apakah client_op_id sudah pernah applied — kalau ya, skip (status=duplicate)
//  2. Apply ke entity berdasarkan action + entity_type
//  3. Insert ke operation_log
//
// Semua ops di-apply dalam satu transaction — kalau salah satu fatal,
// rollback semua. Per-op error (mis. update non-existent surat) tidak fatal,
// dilaporkan sebagai status=rejected dan ops lain tetap apply.
func (s *Store) ApplyOperations(ctx context.Context, ops []ApplyOperationInput) ([]ApplyOperationResult, error) {
	results := make([]ApplyOperationResult, 0, len(ops))

	for _, op := range ops {
		res := s.applyOneOp(ctx, op)
		results = append(results, res)
	}
	return results, nil
}

// applyOneOp setiap op dalam transaction sendiri-sendiri supaya satu op
// gagal tidak invalidate yang lain. Trade-off vs single-tx-batch:
// single-tx atomic tapi all-or-nothing; per-op = lebih lenient.
func (s *Store) applyOneOp(ctx context.Context, op ApplyOperationInput) ApplyOperationResult {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ApplyOperationResult{
			ClientOpID: op.ClientOpID,
			Status:     "rejected",
			Reason:     fmt.Sprintf("begin tx: %v", err),
		}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Idempotency: cek apakah client_op_id sudah di-log
	var existed bool
	err = tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM operation_log WHERE client_op_id = $1)`,
		op.ClientOpID).Scan(&existed)
	if err != nil {
		return ApplyOperationResult{
			ClientOpID: op.ClientOpID,
			Status:     "rejected",
			Reason:     fmt.Sprintf("idempotency check: %v", err),
		}
	}
	if existed {
		return ApplyOperationResult{ClientOpID: op.ClientOpID, Status: "duplicate"}
	}

	// Apply by entity_type + action
	if op.EntityType == "surat" && op.Action == "update" {
		if reason := applySuratUpdate(ctx, tx, op); reason != "" {
			return ApplyOperationResult{
				ClientOpID: op.ClientOpID,
				Status:     "rejected",
				Reason:     reason,
			}
		}
	} else {
		return ApplyOperationResult{
			ClientOpID: op.ClientOpID,
			Status:     "rejected",
			Reason:     fmt.Sprintf("entity_type/action %q/%q tidak didukung", op.EntityType, op.Action),
		}
	}

	// Insert ke operation_log (idempotency record)
	const insertOpQ = `
		INSERT INTO operation_log
			(client_op_id, user_id, entity_type, entity_id, action, field_changes, client_timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = tx.Exec(ctx, insertOpQ,
		op.ClientOpID, op.UserID, op.EntityType, op.EntityID, op.Action,
		op.FieldChanges, op.ClientTimestamp)
	if err != nil {
		return ApplyOperationResult{
			ClientOpID: op.ClientOpID,
			Status:     "rejected",
			Reason:     fmt.Sprintf("insert oplog: %v", err),
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ApplyOperationResult{
			ClientOpID: op.ClientOpID,
			Status:     "rejected",
			Reason:     fmt.Sprintf("commit: %v", err),
		}
	}
	return ApplyOperationResult{ClientOpID: op.ClientOpID, Status: "applied"}
}

// applySuratUpdate apply field changes ke surat row. field_changes adalah
// JSON object dengan key=field name. Last-write-wins semantik: client_timestamp
// dibandingkan dengan updated_at server — kalau client lebih lama, reject
// dengan reason="stale".
//
// Untuk MVP: full row LWW (compare client_timestamp vs server.updated_at).
// Per-field LWW (target Fase 4 final) butuh schema tambahan field_updated_at;
// di-defer.
func applySuratUpdate(ctx context.Context, tx pgx.Tx, op ApplyOperationInput) string {
	var changes map[string]any
	if err := json.Unmarshal(op.FieldChanges, &changes); err != nil {
		return fmt.Sprintf("invalid field_changes JSON: %v", err)
	}

	// Lock + read current updated_at
	var serverUpdated time.Time
	err := tx.QueryRow(ctx,
		`SELECT updated_at FROM surat WHERE id = $1 AND NOT is_deleted FOR UPDATE`,
		op.EntityID).Scan(&serverUpdated)
	if errors.Is(err, pgx.ErrNoRows) {
		return "surat tidak ditemukan"
	}
	if err != nil {
		return fmt.Sprintf("lock surat: %v", err)
	}

	// LWW row-level: kalau server lebih baru, klien stale → reject
	if serverUpdated.After(op.ClientTimestamp) {
		return "stale: server has newer update (LWW lost)"
	}

	// Build SET clause dari known fields
	allowedFields := map[string]bool{
		"perihal": true, "tanggal_surat": true, "tanggal_terima": true,
		"nomor_surat": true, "instansi_id": true, "klasifikasi_id": true,
		"sifat_id": true, "access_level": true,
	}
	sets := []string{"updated_at = $1"}
	args := []any{op.ClientTimestamp}
	idx := 2
	for k, v := range changes {
		if !allowedFields[k] {
			continue
		}
		sets = append(sets, fmt.Sprintf("%s = $%d", k, idx))
		args = append(args, v)
		idx++
	}
	if len(sets) == 1 {
		return "tidak ada field valid untuk di-update"
	}
	args = append(args, op.EntityID)

	q := fmt.Sprintf("UPDATE surat SET %s WHERE id = $%d AND NOT is_deleted",
		joinComma(sets), idx)
	if _, err := tx.Exec(ctx, q, args...); err != nil {
		return fmt.Sprintf("update surat: %v", err)
	}
	return ""
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}

