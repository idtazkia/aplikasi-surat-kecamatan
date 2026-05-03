package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// Disposisi = assignment surat ke user dengan instruksi + deadline.
type Disposisi struct {
	ID             string
	SuratID        string
	SuratNomor     string
	SuratPerihal   string
	AssignedTo     string
	AssigneeName   string
	NomorDisposisi *string
	Instruksi      string
	Deadline       *time.Time
	Status         string // pending | in_progress | done | cancelled
	CompletedAt    *time.Time
	CreatedBy      string
	CreatorName    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreateDisposisiInput parameter untuk insert disposisi.
type CreateDisposisiInput struct {
	ID             string // UUIDv7 caller-generated
	SuratID        string
	AssignedTo     string
	NomorDisposisi *string
	Instruksi      string
	Deadline       *time.Time
	CreatedBy      string
}

// CreateDisposisi insert + verify surat & assignee exist.
func (s *Store) CreateDisposisi(ctx context.Context, in CreateDisposisiInput) error {
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

	err = tx.QueryRow(ctx, `SELECT id::text FROM users WHERE id = $1 AND is_active`, in.AssignedTo).Scan(&dummy)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrAssigneeNotFound
	}
	if err != nil {
		return fmt.Errorf("store: check assignee: %w", err)
	}

	const q = `
		INSERT INTO disposisi (id, surat_id, assigned_to, nomor_disposisi, instruksi, deadline, status, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7)`
	_, err = tx.Exec(ctx, q,
		in.ID, in.SuratID, in.AssignedTo, in.NomorDisposisi, in.Instruksi, in.Deadline, in.CreatedBy)
	if err != nil {
		return fmt.Errorf("store: insert disposisi: %w", err)
	}

	return tx.Commit(ctx)
}

// UpdateDisposisiStatusInput partial update untuk status & instruksi.
type UpdateDisposisiStatusInput struct {
	Status    string // pending | in_progress | done | cancelled
	Instruksi *string
	UpdatedBy string
}

// UpdateDisposisiStatus update status + auto-set completed_at saat done.
// Caller harus authorize (assignee atau creator/camat) di handler layer.
func (s *Store) UpdateDisposisiStatus(ctx context.Context, id string, in UpdateDisposisiStatusInput) error {
	if !validDisposisiStatus[in.Status] {
		return fmt.Errorf("store: status invalid: %s", in.Status)
	}

	sets := []string{"status = $1", "updated_at = NOW()"}
	args := []interface{}{in.Status}
	if in.Status == "done" {
		sets = append(sets, "completed_at = NOW()")
	} else if in.Status != "done" {
		sets = append(sets, "completed_at = NULL")
	}
	if in.Instruksi != nil {
		sets = append(sets, fmt.Sprintf("instruksi = $%d", len(args)+1))
		args = append(args, *in.Instruksi)
	}
	args = append(args, id)
	q := fmt.Sprintf("UPDATE disposisi SET %s WHERE id = $%d",
		strings.Join(sets, ", "), len(args))

	tag, err := s.pool.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("store: update disposisi: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListDisposisiFilter compose filter untuk list query.
type ListDisposisiFilter struct {
	SuratID       string // semua disposisi untuk satu surat
	AssignedTo    string // disposisi yang ditugaskan ke user X
	CreatedBy     string // disposisi yang dibuat oleh user X
	Status        string // filter status (kosong = semua status)
	IncludeSecret bool   // kalau false, exclude disposisi untuk surat access_level=secret
	Limit         int
}

// ListDisposisi return disposisi by filter dengan join surat + user.
// Order: created_at DESC.
func (s *Store) ListDisposisi(ctx context.Context, f ListDisposisiFilter) ([]Disposisi, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > 200 {
		f.Limit = 200
	}

	conditions := []string{"NOT s.is_deleted"}
	var args []interface{}
	addArg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if f.SuratID != "" {
		conditions = append(conditions, "d.surat_id = "+addArg(f.SuratID))
	}
	if f.AssignedTo != "" {
		conditions = append(conditions, "d.assigned_to = "+addArg(f.AssignedTo))
	}
	if f.CreatedBy != "" {
		conditions = append(conditions, "d.created_by = "+addArg(f.CreatedBy))
	}
	if f.Status != "" {
		conditions = append(conditions, "d.status = "+addArg(f.Status))
	}
	if !f.IncludeSecret {
		conditions = append(conditions, "s.access_level <> 'secret'")
	}

	q := `
		SELECT d.id::text, d.surat_id::text, s.nomor_surat, s.perihal,
		       d.assigned_to::text, ua.full_name,
		       d.nomor_disposisi, d.instruksi, d.deadline, d.status, d.completed_at,
		       d.created_by::text, uc.full_name, d.created_at, d.updated_at
		FROM disposisi d
		JOIN surat s ON s.id = d.surat_id
		JOIN users ua ON ua.id = d.assigned_to
		JOIN users uc ON uc.id = d.created_by
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY d.created_at DESC
		LIMIT ` + addArg(f.Limit)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list disposisi: %w", err)
	}
	defer rows.Close()

	var out []Disposisi
	for rows.Next() {
		var d Disposisi
		if err := rows.Scan(
			&d.ID, &d.SuratID, &d.SuratNomor, &d.SuratPerihal,
			&d.AssignedTo, &d.AssigneeName,
			&d.NomorDisposisi, &d.Instruksi, &d.Deadline, &d.Status, &d.CompletedAt,
			&d.CreatedBy, &d.CreatorName, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("store: scan disposisi: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// GetDisposisiByID single fetch untuk authorization & detail.
func (s *Store) GetDisposisiByID(ctx context.Context, id string) (*Disposisi, error) {
	const q = `
		SELECT d.id::text, d.surat_id::text, s.nomor_surat, s.perihal,
		       d.assigned_to::text, ua.full_name,
		       d.nomor_disposisi, d.instruksi, d.deadline, d.status, d.completed_at,
		       d.created_by::text, uc.full_name, d.created_at, d.updated_at
		FROM disposisi d
		JOIN surat s ON s.id = d.surat_id
		JOIN users ua ON ua.id = d.assigned_to
		JOIN users uc ON uc.id = d.created_by
		WHERE d.id = $1`
	var d Disposisi
	err := s.pool.QueryRow(ctx, q, id).Scan(
		&d.ID, &d.SuratID, &d.SuratNomor, &d.SuratPerihal,
		&d.AssignedTo, &d.AssigneeName,
		&d.NomorDisposisi, &d.Instruksi, &d.Deadline, &d.Status, &d.CompletedAt,
		&d.CreatedBy, &d.CreatorName, &d.CreatedAt, &d.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: get disposisi: %w", err)
	}
	return &d, nil
}

// AssignableUser = projection user untuk dropdown picker assignee.
type AssignableUser struct {
	ID       string
	Username string
	FullName string
	Roles    []string
}

// GetUserName lookup full_name + username untuk satu user (dipakai watermark, audit).
func (s *Store) GetUserName(ctx context.Context, userID string) (fullName, username string, err error) {
	const q = `SELECT full_name, username FROM users WHERE id = $1`
	if err := s.pool.QueryRow(ctx, q, userID).Scan(&fullName, &username); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrNotFound
		}
		return "", "", fmt.Errorf("store: get user name: %w", err)
	}
	return fullName, username, nil
}

// ListAssignableUsers return active users untuk assignee picker.
// Filter: hanya role staf, camat, admin (bukan student).
func (s *Store) ListAssignableUsers(ctx context.Context) ([]AssignableUser, error) {
	const q = `
		SELECT u.id::text, u.username, u.full_name,
		       array_agg(r.code) FILTER (WHERE r.code IS NOT NULL)
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		JOIN roles r ON r.id = ur.role_id
		WHERE u.is_active AND r.code IN ('staf', 'camat', 'admin')
		GROUP BY u.id
		ORDER BY u.full_name`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("store: list assignable: %w", err)
	}
	defer rows.Close()

	var out []AssignableUser
	for rows.Next() {
		var u AssignableUser
		if err := rows.Scan(&u.ID, &u.Username, &u.FullName, &u.Roles); err != nil {
			return nil, fmt.Errorf("store: scan user: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

var validDisposisiStatus = map[string]bool{
	"pending":     true,
	"in_progress": true,
	"done":        true,
	"cancelled":   true,
}

// ErrAssigneeNotFound = user target disposisi tidak ada / inactive.
var ErrAssigneeNotFound = errors.New("store: assignee tidak ditemukan")
