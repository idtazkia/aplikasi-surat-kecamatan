package store

import (
	"context"
	"fmt"
)

// DashboardCamat = ringkasan operasional yang berguna saat camat login pertama kali.
type DashboardCamat struct {
	SuratMasukHariIni     int
	DisposisiBelumAssign  int // jumlah surat masuk tanpa disposisi
	DisposisiOverdue      int // disposisi pending/in_progress dengan deadline < now
	DisposisiAssignedToMe int // disposisi yang ditugaskan ke camat
}

// GetDashboardCamat hitung 4 metric ringkas untuk dashboard camat.
// Single round-trip — semua query di-batch via UNION ALL.
func (s *Store) GetDashboardCamat(ctx context.Context, camatUserID string) (*DashboardCamat, error) {
	const q = `
		SELECT
			(SELECT COUNT(*) FROM surat
				WHERE jenis = 'masuk' AND NOT is_deleted
				  AND tanggal_terima = CURRENT_DATE),
			(SELECT COUNT(*) FROM surat s
				WHERE s.jenis = 'masuk' AND NOT s.is_deleted
				  AND NOT EXISTS (SELECT 1 FROM disposisi d WHERE d.surat_id = s.id)),
			(SELECT COUNT(*) FROM disposisi d
				JOIN surat s ON s.id = d.surat_id
				WHERE NOT s.is_deleted
				  AND d.status IN ('pending', 'in_progress')
				  AND d.deadline IS NOT NULL AND d.deadline < NOW()),
			(SELECT COUNT(*) FROM disposisi d
				JOIN surat s ON s.id = d.surat_id
				WHERE NOT s.is_deleted
				  AND d.assigned_to = $1
				  AND d.status IN ('pending', 'in_progress'))`

	var dc DashboardCamat
	err := s.pool.QueryRow(ctx, q, camatUserID).Scan(
		&dc.SuratMasukHariIni,
		&dc.DisposisiBelumAssign,
		&dc.DisposisiOverdue,
		&dc.DisposisiAssignedToMe,
	)
	if err != nil {
		return nil, fmt.Errorf("store: dashboard camat: %w", err)
	}
	return &dc, nil
}
