package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// SuratListItem adalah projection minimal untuk halaman list.
// Detail view lebih lengkap — tambah lampiran + references.
type SuratListItem struct {
	ID              string
	Jenis           string
	NomorSurat      string
	Perihal         string
	TanggalSurat    time.Time
	TanggalTerima   *time.Time
	InstansiID      string
	InstansiNama    string
	KlasifikasiKode *string
	SifatKode       *string
	AccessLevel     string
	CreatedAt       time.Time
}

// ListSuratFilter compose query parameter list. All optional kecuali Limit.
type ListSuratFilter struct {
	Jenis            string // "masuk" | "keluar" | "" (both)
	TanggalDari      *time.Time
	TanggalSampai    *time.Time
	InstansiID       string // UUID
	KlasifikasiID    string // UUID
	SifatID          string // UUID
	Search           string // ILIKE perihal
	IncludeSecret    bool   // true kalau user punya permission surat:read_secret
	Limit            int    // default 20, max 100
	AfterCreatedAt   *time.Time
	AfterID          string // tiebreaker dengan created_at sama
}

// concept:keyset-pagination:start
// ListSurat = list surat dengan filter + keyset pagination.
//
// Kenapa keyset bukan OFFSET:
//   - OFFSET tetap scan baris yang di-skip (kalau OFFSET 10000, scan 10000 baris)
//   - Keyset pakai (created_at, id) sebagai cursor; query langsung ke posisi
//     dengan B-Tree index seek — O(log n + page_size) konstan untuk semua halaman
//   - Keyset stable: kalau ada insert/delete antar fetch, tidak ada drift hasil
//
// Cursor encode: (after_created_at, after_id). Tiebreaker `id` untuk
// determinasi penuh (created_at bisa duplikat di batch insert).
//
// Order: created_at DESC, id DESC. Newest first.
func (s *Store) ListSurat(ctx context.Context, f ListSuratFilter) ([]SuratListItem, error) {
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}

	var (
		conditions = []string{"NOT s.is_deleted"}
		args       []interface{}
	)
	addArg := func(v interface{}) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	if f.Jenis != "" {
		conditions = append(conditions, "s.jenis = "+addArg(f.Jenis))
	}
	if f.TanggalDari != nil {
		conditions = append(conditions, "s.tanggal_surat >= "+addArg(*f.TanggalDari))
	}
	if f.TanggalSampai != nil {
		conditions = append(conditions, "s.tanggal_surat <= "+addArg(*f.TanggalSampai))
	}
	if f.InstansiID != "" {
		conditions = append(conditions, "s.instansi_id = "+addArg(f.InstansiID))
	}
	if f.KlasifikasiID != "" {
		conditions = append(conditions, "s.klasifikasi_id = "+addArg(f.KlasifikasiID))
	}
	if f.SifatID != "" {
		conditions = append(conditions, "s.sifat_id = "+addArg(f.SifatID))
	}
	if f.Search != "" {
		conditions = append(conditions, "s.perihal ILIKE "+addArg("%"+f.Search+"%"))
	}
	if !f.IncludeSecret {
		conditions = append(conditions, "s.access_level <> 'secret'")
	}
	// Keyset cursor — kalau ada, ambil row dengan (created_at, id) lebih kecil dari cursor.
	if f.AfterCreatedAt != nil && f.AfterID != "" {
		conditions = append(conditions,
			fmt.Sprintf("(s.created_at, s.id) < (%s, %s)",
				addArg(*f.AfterCreatedAt), addArg(f.AfterID)))
	}

	q := `
		SELECT s.id::text, s.jenis, s.nomor_surat, s.perihal,
		       s.tanggal_surat, s.tanggal_terima,
		       s.instansi_id::text, i.nama_kanonik,
		       k.kode, sf.kode,
		       s.access_level, s.created_at
		FROM surat s
		JOIN instansi i ON i.id = s.instansi_id
		LEFT JOIN klasifikasi k ON k.id = s.klasifikasi_id
		LEFT JOIN sifat sf ON sf.id = s.sifat_id
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY s.created_at DESC, s.id DESC
		LIMIT ` + addArg(f.Limit)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: list surat: %w", err)
	}
	defer rows.Close()

	var out []SuratListItem
	for rows.Next() {
		var it SuratListItem
		if err := rows.Scan(
			&it.ID, &it.Jenis, &it.NomorSurat, &it.Perihal,
			&it.TanggalSurat, &it.TanggalTerima,
			&it.InstansiID, &it.InstansiNama,
			&it.KlasifikasiKode, &it.SifatKode,
			&it.AccessLevel, &it.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("store: scan surat: %w", err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: iter surat: %w", err)
	}
	return out, nil
}

// concept:keyset-pagination:end

// SuratDetail = surat metadata + lampiran + references (predecessors + successors) + tembusan.
type SuratDetail struct {
	SuratListItem
	DeskripsiKlasifikasi *string
	NamaSifat            *string
	Attachments          []SuratAttachment
	Predecessors         []SuratReference
	Successors           []SuratReference
	Tembusan             []SuratTembusan
}

// SuratAttachment = file lampiran (active version saja).
type SuratAttachment struct {
	ID         string
	Role       string // "primary" | "lampiran"
	FileName   string
	FileSize   int64
	MimeType   string
	UploadedAt time.Time
}

// SuratReference = link ke surat lain (atau external_ref).
type SuratReference struct {
	ID            string
	ToSuratID     *string // nil kalau external
	ToNomorSurat  *string
	ToPerihal     *string
	Relationship  string
	ExternalRef   *string
	Note          *string
	CreatedAt     time.Time
}

// GetSuratByID — fetch detail surat + attachments + references.
// Return ErrNotFound kalau tidak ada atau soft-deleted.
// Caller harus filter access_level secret di handler layer.
func (s *Store) GetSuratByID(ctx context.Context, id string) (*SuratDetail, error) {
	const headerQuery = `
		SELECT s.id::text, s.jenis, s.nomor_surat, s.perihal,
		       s.tanggal_surat, s.tanggal_terima,
		       s.instansi_id::text, i.nama_kanonik,
		       k.kode, sf.kode,
		       s.access_level, s.created_at,
		       k.deskripsi, sf.nama
		FROM surat s
		JOIN instansi i ON i.id = s.instansi_id
		LEFT JOIN klasifikasi k ON k.id = s.klasifikasi_id
		LEFT JOIN sifat sf ON sf.id = s.sifat_id
		WHERE s.id = $1 AND NOT s.is_deleted`

	var d SuratDetail
	err := s.pool.QueryRow(ctx, headerQuery, id).Scan(
		&d.ID, &d.Jenis, &d.NomorSurat, &d.Perihal,
		&d.TanggalSurat, &d.TanggalTerima,
		&d.InstansiID, &d.InstansiNama,
		&d.KlasifikasiKode, &d.SifatKode,
		&d.AccessLevel, &d.CreatedAt,
		&d.DeskripsiKlasifikasi, &d.NamaSifat,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: get surat header: %w", err)
	}

	// Attachments — yang masih aktif (replaced_by chain dipotong di is_active).
	const attachQuery = `
		SELECT id::text, role, file_name, file_size, mime_type, uploaded_at
		FROM surat_attachments
		WHERE surat_id = $1 AND is_active
		ORDER BY role, uploaded_at`
	rows, err := s.pool.Query(ctx, attachQuery, id)
	if err != nil {
		return nil, fmt.Errorf("store: get attachments: %w", err)
	}
	for rows.Next() {
		var a SuratAttachment
		if err := rows.Scan(&a.ID, &a.Role, &a.FileName, &a.FileSize, &a.MimeType, &a.UploadedAt); err != nil {
			rows.Close()
			return nil, fmt.Errorf("store: scan attachment: %w", err)
		}
		d.Attachments = append(d.Attachments, a)
	}
	rows.Close()

	// Predecessors — surat ini merujuk surat lain (from_surat_id = ini).
	d.Predecessors, err = s.queryReferences(ctx, id, "from")
	if err != nil {
		return nil, err
	}

	// Successors — surat lain merujuk surat ini (to_surat_id = ini).
	d.Successors, err = s.queryReferences(ctx, id, "to")
	if err != nil {
		return nil, err
	}

	// Tembusan — list instansi/external yang ditembus.
	d.Tembusan, err = s.listTembusan(ctx, id)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

// queryReferences ambil references di salah satu arah ("from" atau "to").
func (s *Store) queryReferences(ctx context.Context, suratID, direction string) ([]SuratReference, error) {
	var q string
	switch direction {
	case "from":
		// Surat ini di sisi "from" — ambil references yang from_surat_id = ini.
		// Join ke surat lain di sisi to (kalau ada — bisa null untuk external).
		q = `
			SELECT r.id::text, r.to_surat_id::text, ts.nomor_surat, ts.perihal,
			       r.relationship, r.external_ref, r.note, r.created_at
			FROM surat_references r
			LEFT JOIN surat ts ON ts.id = r.to_surat_id AND NOT ts.is_deleted
			WHERE r.from_surat_id = $1
			ORDER BY r.created_at`
	case "to":
		// Surat ini di sisi "to" — ambil references yang to_surat_id = ini.
		q = `
			SELECT r.id::text, r.from_surat_id::text, fs.nomor_surat, fs.perihal,
			       r.relationship, NULL::text, r.note, r.created_at
			FROM surat_references r
			JOIN surat fs ON fs.id = r.from_surat_id AND NOT fs.is_deleted
			WHERE r.to_surat_id = $1
			ORDER BY r.created_at`
	default:
		return nil, fmt.Errorf("store: invalid direction %q", direction)
	}

	rows, err := s.pool.Query(ctx, q, suratID)
	if err != nil {
		return nil, fmt.Errorf("store: query refs: %w", err)
	}
	defer rows.Close()

	var out []SuratReference
	for rows.Next() {
		var r SuratReference
		var otherID *string
		if err := rows.Scan(&r.ID, &otherID, &r.ToNomorSurat, &r.ToPerihal,
			&r.Relationship, &r.ExternalRef, &r.Note, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan ref: %w", err)
		}
		r.ToSuratID = otherID
		out = append(out, r)
	}
	return out, rows.Err()
}
