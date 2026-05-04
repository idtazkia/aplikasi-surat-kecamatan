package store

import (
	"context"
	"fmt"
	"time"
)

// Stats* = aggregation projections untuk reporting endpoint.
//
// Trade-off untuk volume rendah (<100k surat di kecamatan): SUM/COUNT GROUP BY
// langsung lebih simpel daripada materialized view. Cost negligible tanpa
// perlu cache layer; query sub-100ms di dataset realistis.

// StatsByPeriod = surat count per (year-month) bucket.
type StatsByPeriod struct {
	Bucket    string // "YYYY-MM"
	JenisCount map[string]int // "masuk" → count, "keluar" → count
}

// GetStatsByPeriod return time series count per bulan, sorted ASC.
// Filter: hanya surat masuk yang punya tanggal_terima, surat keluar pakai
// tanggal_surat. Soft-deleted di-exclude.
//
// Range: dari `from` (inclusive) sampai `to` (inclusive). Nil = no bound.
func (s *Store) GetStatsByPeriod(ctx context.Context, from, to *time.Time) ([]StatsByPeriod, error) {
	conditions := []string{"NOT s.is_deleted"}
	args := []interface{}{}
	if from != nil {
		args = append(args, *from)
		conditions = append(conditions, fmt.Sprintf(
			"COALESCE(s.tanggal_terima, s.tanggal_surat) >= $%d", len(args)))
	}
	if to != nil {
		args = append(args, *to)
		conditions = append(conditions, fmt.Sprintf(
			"COALESCE(s.tanggal_terima, s.tanggal_surat) <= $%d", len(args)))
	}

	q := `
		SELECT to_char(COALESCE(s.tanggal_terima, s.tanggal_surat), 'YYYY-MM') AS bucket,
		       s.jenis, COUNT(*) AS cnt
		FROM surat s
		WHERE ` + joinAnd(conditions) + `
		GROUP BY bucket, s.jenis
		ORDER BY bucket ASC`

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: stats period: %w", err)
	}
	defer rows.Close()

	// Aggregate (bucket, jenis) → bucket grouping
	bucketMap := map[string]map[string]int{}
	var order []string
	for rows.Next() {
		var bucket, jenis string
		var cnt int
		if err := rows.Scan(&bucket, &jenis, &cnt); err != nil {
			return nil, fmt.Errorf("store: scan stats period: %w", err)
		}
		if _, ok := bucketMap[bucket]; !ok {
			bucketMap[bucket] = map[string]int{}
			order = append(order, bucket)
		}
		bucketMap[bucket][jenis] = cnt
	}
	out := make([]StatsByPeriod, 0, len(order))
	for _, b := range order {
		out = append(out, StatsByPeriod{Bucket: b, JenisCount: bucketMap[b]})
	}
	return out, rows.Err()
}

// StatsByClassification = count per klasifikasi.
type StatsByClassification struct {
	KlasifikasiKode *string
	KlasifikasiNama *string
	Count           int
}

// GetStatsByClassification group surat by klasifikasi (NULL = "Tanpa klasifikasi").
func (s *Store) GetStatsByClassification(ctx context.Context) ([]StatsByClassification, error) {
	const q = `
		SELECT k.kode, k.nama, COUNT(*) AS cnt
		FROM surat s
		LEFT JOIN klasifikasi k ON k.id = s.klasifikasi_id
		WHERE NOT s.is_deleted
		GROUP BY k.kode, k.nama
		ORDER BY cnt DESC`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("store: stats klasifikasi: %w", err)
	}
	defer rows.Close()

	var out []StatsByClassification
	for rows.Next() {
		var c StatsByClassification
		if err := rows.Scan(&c.KlasifikasiKode, &c.KlasifikasiNama, &c.Count); err != nil {
			return nil, fmt.Errorf("store: scan klasifikasi: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// StatsBySender = top-N instansi pengirim.
type StatsBySender struct {
	InstansiID   string
	InstansiNama string
	Count        int
}

// GetStatsBySender top N instansi by surat masuk count.
func (s *Store) GetStatsBySender(ctx context.Context, limit int) ([]StatsBySender, error) {
	if limit <= 0 || limit > 100 {
		limit = 10
	}
	const q = `
		SELECT i.id::text, i.nama_kanonik, COUNT(*) AS cnt
		FROM surat s
		JOIN instansi i ON i.id = s.instansi_id
		WHERE s.jenis = 'masuk' AND NOT s.is_deleted
		GROUP BY i.id, i.nama_kanonik
		ORDER BY cnt DESC
		LIMIT $1`
	rows, err := s.pool.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("store: stats sender: %w", err)
	}
	defer rows.Close()

	var out []StatsBySender
	for rows.Next() {
		var x StatsBySender
		if err := rows.Scan(&x.InstansiID, &x.InstansiNama, &x.Count); err != nil {
			return nil, fmt.Errorf("store: scan sender: %w", err)
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// StatsStaffLoad = disposisi count per staf, breakdown by status.
type StatsStaffLoad struct {
	UserID         string
	FullName       string
	StatusCount    map[string]int // pending | in_progress | done | cancelled
	OverdueCount   int            // pending|in_progress dengan deadline < now
	TotalActive    int            // pending + in_progress
}

// GetStatsStaffLoad return load per assignee staf/camat. Used untuk dashboard
// "siapa kerja apa" — visibility supervisor.
func (s *Store) GetStatsStaffLoad(ctx context.Context) ([]StatsStaffLoad, error) {
	const q = `
		SELECT u.id::text, u.full_name, d.status,
		       COUNT(*) AS cnt,
		       COUNT(*) FILTER (
		         WHERE d.status IN ('pending', 'in_progress')
		           AND d.deadline IS NOT NULL AND d.deadline < NOW()
		       ) AS overdue_cnt
		FROM users u
		JOIN disposisi d ON d.assigned_to = u.id
		JOIN surat s ON s.id = d.surat_id AND NOT s.is_deleted
		JOIN user_roles ur ON ur.user_id = u.id
		JOIN roles r ON r.id = ur.role_id
		WHERE u.is_active AND r.code IN ('staf', 'camat')
		GROUP BY u.id, u.full_name, d.status
		ORDER BY u.full_name, d.status`
	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("store: stats staff load: %w", err)
	}
	defer rows.Close()

	loadMap := map[string]*StatsStaffLoad{}
	var order []string
	for rows.Next() {
		var uid, name, status string
		var cnt, overdue int
		if err := rows.Scan(&uid, &name, &status, &cnt, &overdue); err != nil {
			return nil, fmt.Errorf("store: scan staff load: %w", err)
		}
		entry, ok := loadMap[uid]
		if !ok {
			entry = &StatsStaffLoad{
				UserID: uid, FullName: name,
				StatusCount: map[string]int{},
			}
			loadMap[uid] = entry
			order = append(order, uid)
		}
		entry.StatusCount[status] = cnt
		entry.OverdueCount += overdue
		if status == "pending" || status == "in_progress" {
			entry.TotalActive += cnt
		}
	}
	out := make([]StatsStaffLoad, 0, len(order))
	for _, uid := range order {
		out = append(out, *loadMap[uid])
	}
	return out, rows.Err()
}

// joinAnd helper untuk SQL WHERE clause.
func joinAnd(conds []string) string {
	if len(conds) == 0 {
		return "TRUE"
	}
	out := conds[0]
	for _, c := range conds[1:] {
		out += " AND " + c
	}
	return out
}

