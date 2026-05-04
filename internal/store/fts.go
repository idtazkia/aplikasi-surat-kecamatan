package store

import (
	"context"
	"fmt"
)

// UpdateSearchDoc rebuild tsvector untuk satu surat dari kombinasi:
//   - metadata: perihal + nomor_surat
//   - extracted text: text yang sudah di-extract dari PDF lampiran (caller pass)
//
// Caller (handler) bertanggungjawab extract text dari PDF + concat semua
// lampiran sebelum panggil function ini.
//
// to_tsvector('simple', ...) tokenize tanpa stemming — match exact words.
// Cocok untuk Bahasa Indonesia karena PostgreSQL tidak punya stemmer 'id'
// built-in (snowball ekstensi bisa di-add Fase 7+).
func (s *Store) UpdateSearchDoc(ctx context.Context, suratID string, attachmentText string) error {
	const q = `
		UPDATE surat
		SET search_doc = to_tsvector('simple',
			coalesce(perihal, '') || ' ' ||
			coalesce(nomor_surat, '') || ' ' ||
			$2)
		WHERE id = $1 AND NOT is_deleted`
	_, err := s.pool.Exec(ctx, q, suratID, attachmentText)
	if err != nil {
		return fmt.Errorf("store: update search_doc: %w", err)
	}
	return nil
}

// ConcatAttachmentText helper: ambil semua extracted_text dari attachment
// active untuk surat (kalau kita store di kolom). MVP tidak persist
// extracted text per attachment — caller pass langsung.
//
// Untuk versioning future: kalau replace attachment, search_doc perlu
// rebuild dari scratch. Caller bisa panggil UpdateSearchDoc lagi dengan
// text dari versi aktif terbaru.
