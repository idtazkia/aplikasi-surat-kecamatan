package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// InstansiItem projection untuk autocomplete.
type InstansiItem struct {
	ID           string
	NamaKanonik  string
	Aliases      []string
	Alamat       *string
	Kontak       *string
}

// SearchInstansi return instansi yang cocok dengan keyword di nama_kanonik
// atau salah satu alias. Limit 20 (cukup untuk autocomplete dropdown).
func (s *Store) SearchInstansi(ctx context.Context, keyword string, limit int) ([]InstansiItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	q := `
		SELECT id::text, nama_kanonik, aliases, alamat, kontak
		FROM instansi
		WHERE is_active`
	args := []interface{}{}
	if keyword != "" {
		q += ` AND (nama_kanonik ILIKE $1 OR EXISTS (
			SELECT 1 FROM unnest(aliases) a WHERE a ILIKE $1
		))`
		args = append(args, "%"+keyword+"%")
	}
	q += fmt.Sprintf(` ORDER BY nama_kanonik LIMIT %d`, limit)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store: search instansi: %w", err)
	}
	defer rows.Close()

	var out []InstansiItem
	for rows.Next() {
		var it InstansiItem
		if err := rows.Scan(&it.ID, &it.NamaKanonik, &it.Aliases, &it.Alamat, &it.Kontak); err != nil {
			return nil, fmt.Errorf("store: scan instansi: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// CreateInstansiInput
type CreateInstansiInput struct {
	ID          string // UUIDv7 caller-generated
	NamaKanonik string
	Aliases     []string
	Alamat      *string
	Kontak      *string
}

// CreateInstansi insert instansi baru. Return ErrConflict kalau nama_kanonik dup.
func (s *Store) CreateInstansi(ctx context.Context, in CreateInstansiInput) error {
	const q = `
		INSERT INTO instansi (id, nama_kanonik, aliases, alamat, kontak)
		VALUES ($1, $2, $3, $4, $5)`
	_, err := s.pool.Exec(ctx, q, in.ID, in.NamaKanonik, in.Aliases, in.Alamat, in.Kontak)
	if err != nil {
		if isUniqueViolation(err) {
			return ErrConflict
		}
		return fmt.Errorf("store: insert instansi: %w", err)
	}
	return nil
}

// GetInstansi by ID. Return ErrNotFound kalau tidak ada.
func (s *Store) GetInstansi(ctx context.Context, id string) (*InstansiItem, error) {
	const q = `
		SELECT id::text, nama_kanonik, aliases, alamat, kontak
		FROM instansi
		WHERE id = $1 AND is_active`
	var it InstansiItem
	err := s.pool.QueryRow(ctx, q, id).Scan(&it.ID, &it.NamaKanonik, &it.Aliases, &it.Alamat, &it.Kontak)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: get instansi: %w", err)
	}
	return &it, nil
}

// LookupItem (klasifikasi atau sifat) — generic untuk kedua master.
type LookupItem struct {
	ID        string
	Kode      string
	Nama      string
	Deskripsi *string
}

// ListKlasifikasi return semua klasifikasi aktif.
func (s *Store) ListKlasifikasi(ctx context.Context) ([]LookupItem, error) {
	const q = `
		SELECT id::text, kode, nama, deskripsi
		FROM klasifikasi
		WHERE is_active
		ORDER BY kode`
	return s.queryLookup(ctx, q)
}

// ListSifat return semua sifat aktif (urut prioritas asc).
func (s *Store) ListSifat(ctx context.Context) ([]LookupItem, error) {
	const q = `
		SELECT id::text, kode, nama, NULL::text
		FROM sifat
		WHERE is_active
		ORDER BY prioritas, kode`
	return s.queryLookup(ctx, q)
}

func (s *Store) queryLookup(ctx context.Context, query string) ([]LookupItem, error) {
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("store: lookup query: %w", err)
	}
	defer rows.Close()

	var out []LookupItem
	for rows.Next() {
		var it LookupItem
		if err := rows.Scan(&it.ID, &it.Kode, &it.Nama, &it.Deskripsi); err != nil {
			return nil, fmt.Errorf("store: scan lookup: %w", err)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
