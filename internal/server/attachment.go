package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/uuid7"
)

// AttachmentStore extends untuk attachment ops.
type AttachmentStore interface {
	GetSuratByID(ctx context.Context, id string) (*store.SuratDetail, error)
	AddAttachment(ctx context.Context, in store.AttachmentInput) error
	AttachmentByID(ctx context.Context, id string) (*store.AttachmentInput, error)
}

const (
	maxFileSize    = 25 * 1024 * 1024 // 25 MB per file
	maxFilesPerReq = 10
)

var allowedMimePrefixes = []string{
	"application/pdf",
	"image/",
	"application/msword",
	"application/vnd.openxmlformats-officedocument",
	"application/vnd.ms-excel",
	"text/",
}

// concept:multipart-streaming:start
// suratAttachmentsUploadHandler handle POST /api/surat/{id}/attachments multipart
// pakai r.MultipartReader() (streaming) — hindari load semua file ke memory.
//
// Setiap part di-stream ke disk pakai io.Copy + io.LimitReader untuk enforce
// per-file size limit. Kalau file melebihi limit, return 413 Payload Too Large.
//
// Kontras dengan r.ParseMultipartForm yang load all parts ke memory atau
// temp file — streaming approach memory-bounded O(1) per file regardless
// of file size.
func suratAttachmentsUploadHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		suratID := r.PathValue("id")
		if suratID == "" {
			writeError(w, http.StatusBadRequest, "id required")
			return
		}

		// Verifikasi surat exist (also enforce ACL secret di sini)
		surat, err := d.AttachmentStore.GetSuratByID(r.Context(), suratID)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "surat tidak ditemukan")
			return
		}
		if err != nil {
			d.Logger.Error("attachment: surat lookup", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if surat.AccessLevel == "secret" && !hasReadSecret(claims.Roles) {
			writeError(w, http.StatusForbidden, "akses surat rahasia ditolak")
			return
		}

		mr, err := r.MultipartReader()
		if err != nil {
			writeError(w, http.StatusBadRequest, "expected multipart/form-data")
			return
		}

		var uploaded []attachmentDTO
		fileCount := 0

		for {
			part, err := mr.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				d.Logger.Error("multipart: next part", "err", err)
				writeError(w, http.StatusBadRequest, "malformed multipart body")
				return
			}

			// Skip non-file parts
			if part.FileName() == "" {
				_, _ = io.Copy(io.Discard, part)
				_ = part.Close()
				continue
			}

			fileCount++
			if fileCount > maxFilesPerReq {
				_ = part.Close()
				writeError(w, http.StatusBadRequest, fmt.Sprintf("max %d files per request", maxFilesPerReq))
				return
			}

			// Determine role: form name "primary" → primary, otherwise lampiran
			role := "lampiran"
			if part.FormName() == "primary" {
				role = "primary"
			}

			savedPath, fileSize, mimeType, err := streamPartToDisk(part, d.AttachmentRoot)
			_ = part.Close()
			if err != nil {
				if errors.Is(err, errFileTooLarge) {
					writeError(w, http.StatusRequestEntityTooLarge,
						fmt.Sprintf("file %q lebih besar dari %d byte", part.FileName(), maxFileSize))
					return
				}
				d.Logger.Error("multipart: stream", "err", err)
				writeError(w, http.StatusInternalServerError, "internal error")
				return
			}

			if !mimeAllowed(mimeType) {
				_ = os.Remove(filepath.Join(d.AttachmentRoot, savedPath))
				writeError(w, http.StatusUnsupportedMediaType,
					fmt.Sprintf("MIME type %q tidak diizinkan", mimeType))
				return
			}

			attID, err := uuid7.New()
			if err != nil {
				d.Logger.Error("uuid: generate", "err", err)
				writeError(w, http.StatusInternalServerError, "internal error")
				return
			}

			in := storeAttachmentInput(attID.String(), suratID, role, part.FileName(), savedPath, fileSize, mimeType, claims.Sub)
			if err := d.AttachmentStore.AddAttachment(r.Context(), in); err != nil {
				_ = os.Remove(filepath.Join(d.AttachmentRoot, savedPath))
				d.Logger.Error("attachment: insert", "err", err)
				writeError(w, http.StatusInternalServerError, "internal error")
				return
			}

			uploaded = append(uploaded, attachmentDTO{
				ID: attID.String(), Role: role, FileName: part.FileName(),
				FileSize: fileSize, MimeType: mimeType,
			})
		}

		if fileCount == 0 {
			writeError(w, http.StatusBadRequest, "no files in multipart body")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"uploaded": uploaded,
		})
	}
}

// concept:multipart-streaming:end

type attachmentDTO struct {
	ID       string `json:"id"`
	Role     string `json:"role"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
	MimeType string `json:"mime_type"`
}

// streamPartToDisk simpan multipart part ke disk dengan UUIDv7 filename.
// Detect MIME dari first 512 byte (sniff) — jangan trust user's Content-Type.
// Return relative path (untuk DB), fileSize, mimeType.
func streamPartToDisk(part io.Reader, rootDir string) (relPath string, size int64, mime string, err error) {
	id, err := uuid7.New()
	if err != nil {
		return "", 0, "", err
	}

	// Pastikan storage root exists
	if err := os.MkdirAll(rootDir, 0o750); err != nil {
		return "", 0, "", fmt.Errorf("mkdir storage: %w", err)
	}

	relPath = id.String()
	absPath := filepath.Join(rootDir, relPath)

	out, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return "", 0, "", fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	// Sniff MIME dari first 512 byte
	sniff := make([]byte, 512)
	n, sniffErr := io.ReadFull(part, sniff)
	if sniffErr != nil && !errors.Is(sniffErr, io.EOF) && !errors.Is(sniffErr, io.ErrUnexpectedEOF) {
		_ = os.Remove(absPath)
		return "", 0, "", fmt.Errorf("read sniff: %w", sniffErr)
	}
	mime = http.DetectContentType(sniff[:n])
	if _, err := out.Write(sniff[:n]); err != nil {
		_ = os.Remove(absPath)
		return "", 0, "", fmt.Errorf("write sniff: %w", err)
	}
	size = int64(n)

	// Stream sisa dengan size limit
	limited := io.LimitReader(part, int64(maxFileSize-int(size)+1))
	written, err := io.Copy(out, limited)
	if err != nil {
		_ = os.Remove(absPath)
		return "", 0, "", fmt.Errorf("copy body: %w", err)
	}
	size += written

	if size > maxFileSize {
		_ = os.Remove(absPath)
		return "", 0, "", errFileTooLarge
	}
	return relPath, size, mime, nil
}

func mimeAllowed(mime string) bool {
	mime = strings.ToLower(mime)
	for _, prefix := range allowedMimePrefixes {
		if strings.HasPrefix(mime, prefix) {
			return true
		}
	}
	return false
}

var errFileTooLarge = errors.New("file too large")

// suratAttachmentDownloadHandler stream file dari disk untuk download.
func suratAttachmentDownloadHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		suratID := r.PathValue("id")
		attID := r.PathValue("att_id")
		if suratID == "" || attID == "" {
			writeError(w, http.StatusBadRequest, "id and att_id required")
			return
		}

		surat, err := d.AttachmentStore.GetSuratByID(r.Context(), suratID)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "surat tidak ditemukan")
			return
		}
		if err != nil {
			d.Logger.Error("attachment download: surat", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if surat.AccessLevel == "secret" && !hasReadSecret(claims.Roles) {
			writeError(w, http.StatusForbidden, "akses surat rahasia ditolak")
			return
		}

		att, err := d.AttachmentStore.AttachmentByID(r.Context(), attID)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "lampiran tidak ditemukan")
			return
		}
		if err != nil {
			d.Logger.Error("attachment: fetch", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if att.SuratID != suratID {
			writeError(w, http.StatusNotFound, "lampiran tidak ditemukan di surat ini")
			return
		}

		f, err := os.Open(filepath.Join(d.AttachmentRoot, att.FilePath))
		if err != nil {
			d.Logger.Error("attachment: open file", "err", err, "path", att.FilePath)
			writeError(w, http.StatusInternalServerError, "file storage error")
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", att.MimeType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", att.FileSize))
		w.Header().Set("Content-Disposition",
			fmt.Sprintf(`attachment; filename="%s"`, sanitizeFilename(att.FileName)))
		w.WriteHeader(http.StatusOK)

		// Streaming copy — bounded buffer (32KB default), no full-file load
		_, _ = io.Copy(w, f)
	}
}

// sanitizeFilename remove path separators dari filename untuk Content-Disposition header.
func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, `"`, `'`)
	return name
}

// storeAttachmentInput helper construct AttachmentInput dari handler.
func storeAttachmentInput(id, suratID, role, fileName, filePath string, fileSize int64, mimeType, uploadedBy string) store.AttachmentInput {
	return store.AttachmentInput{
		ID: id, SuratID: suratID, Role: role,
		FileName: fileName, FilePath: filePath,
		FileSize: fileSize, MimeType: mimeType,
		UploadedBy: uploadedBy,
	}
}

