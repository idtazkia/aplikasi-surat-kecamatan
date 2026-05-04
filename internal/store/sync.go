package store

import (
	"context"
	"fmt"
	"time"
)

// SyncSnapshot = delta payload untuk PWA offline read-only sync.
// Kalau `since` nil → full snapshot (active rows saja). Kalau ada → hanya
// row dengan updated_at > since, plus tombstones surat yang di-soft-delete
// sejak watermark itu.
type SyncSnapshot struct {
	Watermark       time.Time
	Surat           []SuratListItem
	SuratDeletedIDs []string // tombstones — klien hapus dari cache lokal
	Instansi        []InstansiItem
	Klasifikasi     []LookupItem
	Sifat           []LookupItem
}

// GetSyncSnapshot ambil delta sejak `since` (nil = full).
// Server pakai NOW() sebagai watermark kembali — klien simpan, kirim ulang
// di sync berikutnya. Single transaction supaya snapshot konsisten.
func (s *Store) GetSyncSnapshot(ctx context.Context, since *time.Time, includeSecret bool) (*SyncSnapshot, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("store: sync tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var watermark time.Time
	if err := tx.QueryRow(ctx, `SELECT NOW()`).Scan(&watermark); err != nil {
		return nil, fmt.Errorf("store: sync watermark: %w", err)
	}

	// Initialize slice fields (bukan nil) supaya JSON encode jadi `[]` bukan `null`.
	// Klien yang konsumsi hasil expect array — bukan null.
	out := SyncSnapshot{
		Watermark:       watermark,
		Surat:           []SuratListItem{},
		SuratDeletedIDs: []string{},
		Instansi:        []InstansiItem{},
		Klasifikasi:     []LookupItem{},
		Sifat:           []LookupItem{},
	}

	// Surat: active rows yang updated_at > since (atau full kalau since nil)
	// PLUS deleted rows yang deleted_at > since (tombstones).
	suratQ := `
		SELECT s.id::text, s.jenis, s.nomor_surat, s.perihal,
		       s.tanggal_surat, s.tanggal_terima,
		       s.instansi_id::text, i.nama_kanonik,
		       k.kode, sf.kode,
		       s.access_level, s.created_at
		FROM surat s
		JOIN instansi i ON i.id = s.instansi_id
		LEFT JOIN klasifikasi k ON k.id = s.klasifikasi_id
		LEFT JOIN sifat sf ON sf.id = s.sifat_id
		WHERE NOT s.is_deleted`
	args := []interface{}{}
	if since != nil {
		suratQ += ` AND s.updated_at > $1`
		args = append(args, *since)
	}
	if !includeSecret {
		suratQ += ` AND s.access_level <> 'secret'`
	}

	rows, err := tx.Query(ctx, suratQ, args...)
	if err != nil {
		return nil, fmt.Errorf("store: sync surat: %w", err)
	}
	for rows.Next() {
		var it SuratListItem
		if err := rows.Scan(
			&it.ID, &it.Jenis, &it.NomorSurat, &it.Perihal,
			&it.TanggalSurat, &it.TanggalTerima,
			&it.InstansiID, &it.InstansiNama,
			&it.KlasifikasiKode, &it.SifatKode,
			&it.AccessLevel, &it.CreatedAt,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("store: sync scan surat: %w", err)
		}
		out.Surat = append(out.Surat, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: sync iter surat: %w", err)
	}

	// Tombstones (only relevant kalau ada `since`, full sync tidak butuh)
	if since != nil {
		tRows, err := tx.Query(ctx, `
			SELECT id::text FROM surat
			WHERE is_deleted AND deleted_at > $1`, *since)
		if err != nil {
			return nil, fmt.Errorf("store: sync tombstones: %w", err)
		}
		for tRows.Next() {
			var id string
			if err := tRows.Scan(&id); err != nil {
				tRows.Close()
				return nil, fmt.Errorf("store: scan tombstone: %w", err)
			}
			out.SuratDeletedIDs = append(out.SuratDeletedIDs, id)
		}
		tRows.Close()
	}

	// Instansi: filter pakai updated_at
	instansiQ := `
		SELECT id::text, nama_kanonik, aliases, alamat, kontak
		FROM instansi
		WHERE is_active`
	instansiArgs := []interface{}{}
	if since != nil {
		instansiQ += ` AND updated_at > $1`
		instansiArgs = append(instansiArgs, *since)
	}
	iRows, err := tx.Query(ctx, instansiQ, instansiArgs...)
	if err != nil {
		return nil, fmt.Errorf("store: sync instansi: %w", err)
	}
	for iRows.Next() {
		var it InstansiItem
		if err := iRows.Scan(&it.ID, &it.NamaKanonik, &it.Aliases, &it.Alamat, &it.Kontak); err != nil {
			iRows.Close()
			return nil, fmt.Errorf("store: scan instansi: %w", err)
		}
		out.Instansi = append(out.Instansi, it)
	}
	iRows.Close()

	// Klasifikasi + Sifat: tidak ada updated_at, dan dataset kecil + jarang
	// berubah → kirim full active list setiap kali. Klien overwrite cache.
	// Trade-off vs schema migration tambah updated_at: payload tetap < 1KB.
	kRows, err := tx.Query(ctx, `
		SELECT id::text, kode, nama, deskripsi
		FROM klasifikasi WHERE is_active ORDER BY kode`)
	if err != nil {
		return nil, fmt.Errorf("store: sync klasifikasi: %w", err)
	}
	for kRows.Next() {
		var k LookupItem
		if err := kRows.Scan(&k.ID, &k.Kode, &k.Nama, &k.Deskripsi); err != nil {
			kRows.Close()
			return nil, fmt.Errorf("store: scan klasifikasi: %w", err)
		}
		out.Klasifikasi = append(out.Klasifikasi, k)
	}
	kRows.Close()

	sfRows, err := tx.Query(ctx, `
		SELECT id::text, kode, nama, NULL::text
		FROM sifat WHERE is_active ORDER BY prioritas`)
	if err != nil {
		return nil, fmt.Errorf("store: sync sifat: %w", err)
	}
	for sfRows.Next() {
		var s LookupItem
		if err := sfRows.Scan(&s.ID, &s.Kode, &s.Nama, &s.Deskripsi); err != nil {
			sfRows.Close()
			return nil, fmt.Errorf("store: scan sifat: %w", err)
		}
		out.Sifat = append(out.Sifat, s)
	}
	sfRows.Close()

	return &out, tx.Commit(ctx)
}
