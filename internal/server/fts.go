// Full-text search support: extract text dari PDF lampiran + sync ke
// surat.search_doc tsvector.
//
// Library: github.com/ledongthuc/pdf — pure Go, BSD license, ringan.
// Tidak handle PDF complex (forms, tables, embedded fonts) — sufficient
// untuk surat resmi yang typical-nya plain text scan.

package server

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	pdfreader "github.com/ledongthuc/pdf"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
)

// FTSStore subset interface untuk update search_doc.
type FTSStore interface {
	UpdateSearchDoc(ctx context.Context, suratID string, attachmentText string) error
	GetSuratByID(ctx context.Context, id string) (*store.SuratDetail, error)
}

// extractPDFText baca PDF dari path, return concat plain text per page.
// Cap 1MB extracted text per file untuk avoid bloat tsvector (postgres
// limit 1MB per tsvector).
const maxExtractedTextBytes = 1 << 20 // 1MB

func extractPDFText(absPath string) (string, error) {
	f, r, err := pdfreader.Open(absPath)
	if err != nil {
		return "", fmt.Errorf("pdf open: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	totalPages := r.NumPage()
	for pageIdx := 1; pageIdx <= totalPages; pageIdx++ {
		p := r.Page(pageIdx)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			// Page-level error tidak fatal — skip + lanjut
			continue
		}
		buf.WriteString(text)
		buf.WriteByte('\n')
		if buf.Len() >= maxExtractedTextBytes {
			break
		}
	}

	out := buf.String()
	if len(out) > maxExtractedTextBytes {
		out = out[:maxExtractedTextBytes]
	}
	// Sanitize: collapse multiple whitespace
	return strings.Join(strings.Fields(out), " "), nil
}

// rebuildSearchDoc — re-extract text dari semua PDF aktif lampiran surat
// dan update tsvector. Dipanggil dari handler upload setelah AddAttachment.
//
// Trade-off vs incremental concat: rebuild from scratch lebih simpel +
// idempotent, cost akseptabel untuk surat dengan ≤10 lampiran. Kalau
// volume besar di masa depan, switch ke per-attachment extracted_text
// column + concat di SQL.
func rebuildSearchDoc(ctx context.Context, d Deps, suratID string) error {
	surat, err := d.FTSStore.GetSuratByID(ctx, suratID)
	if err != nil {
		return fmt.Errorf("rebuild fts: get surat: %w", err)
	}

	var combined bytes.Buffer
	for _, att := range surat.Attachments {
		if att.MimeType != "application/pdf" {
			continue
		}
		path := filepath.Join(d.AttachmentRoot, lookupAttachmentFilePath(ctx, d, att.ID))
		if path == d.AttachmentRoot {
			continue // file_path kosong — skip
		}
		text, err := extractPDFText(path)
		if err != nil {
			d.Logger.Warn("fts: extract gagal, skip", "att_id", att.ID, "err", err)
			continue
		}
		combined.WriteString(text)
		combined.WriteByte(' ')
		if combined.Len() >= maxExtractedTextBytes {
			break
		}
	}

	finalText := combined.String()
	if len(finalText) > maxExtractedTextBytes {
		finalText = finalText[:maxExtractedTextBytes]
	}
	return d.FTSStore.UpdateSearchDoc(ctx, suratID, finalText)
}

// lookupAttachmentFilePath ambil file_path dari AttachmentByID (subset path
// info). Helper supaya rebuildSearchDoc tidak perlu handle null/error sendiri.
func lookupAttachmentFilePath(ctx context.Context, d Deps, attID string) string {
	att, err := d.AttachmentStore.AttachmentByID(ctx, attID)
	if err != nil {
		return ""
	}
	return att.FilePath
}

