package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	pdftypes "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/uuid7"
)

// AttachmentStore extends untuk attachment ops.
type AttachmentStore interface {
	GetSuratByID(ctx context.Context, id string) (*store.SuratDetail, error)
	AddAttachment(ctx context.Context, in store.AttachmentInput) error
	AttachmentByID(ctx context.Context, id string) (*store.AttachmentInput, error)
	ReplaceAttachment(ctx context.Context, oldID string, in store.AttachmentInput) error
	ListAttachmentVersions(ctx context.Context, anyVersionID string) ([]store.AttachmentVersion, error)
	GetUserName(ctx context.Context, userID string) (fullName, username string, err error)
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
		if !requireWriter(claims.Roles) {
			writeError(w, http.StatusForbidden, "role tidak punya hak tulis")
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

// suratAttachmentDownloadHandler stream file dengan Content-Disposition: attachment
// (browser save as).
func suratAttachmentDownloadHandler(d Deps) http.HandlerFunc {
	return suratAttachmentServeHandler(d, "attachment")
}

// suratAttachmentPreviewHandler stream file dengan Content-Disposition: inline
// supaya browser render langsung di iframe / new tab (PDF preview, image preview).
func suratAttachmentPreviewHandler(d Deps) http.HandlerFunc {
	return suratAttachmentServeHandler(d, "inline")
}

func suratAttachmentServeHandler(d Deps, disposition string) http.HandlerFunc {
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
			d.Logger.Error("attachment serve: surat", "err", err)
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

		// Watermark logic: hanya untuk PDF dengan access_level >= restricted.
		// Public surat tidak butuh watermark — anyway tidak sensitif.
		needsWatermark := att.MimeType == "application/pdf" &&
			(surat.AccessLevel == "restricted" || surat.AccessLevel == "secret")

		w.Header().Set("Content-Type", att.MimeType)
		w.Header().Set("Content-Disposition",
			fmt.Sprintf(`%s; filename="%s"`, disposition, sanitizeFilename(att.FileName)))

		if needsWatermark {
			fullName, _, ferr := d.AttachmentStore.GetUserName(r.Context(), claims.Sub)
			if ferr != nil {
				d.Logger.Error("watermark: lookup user name", "err", ferr)
				// Fall through tanpa watermark akan kebobolan — better fail.
				writeError(w, http.StatusInternalServerError, "user lookup error")
				return
			}
			text := fmt.Sprintf("Diunduh oleh %s — %s",
				fullName, time.Now().Format("2006-01-02 15:04 MST"))
			watermarked, werr := applyTextWatermark(f, text)
			if werr != nil {
				d.Logger.Error("watermark: apply", "err", werr, "att_id", att.ID)
				writeError(w, http.StatusInternalServerError, "watermark error")
				return
			}
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(watermarked)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(watermarked)
			return
		}

		w.Header().Set("Content-Length", fmt.Sprintf("%d", att.FileSize))
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, f)
	}
}

// concept:pdf-watermark:start
// applyTextWatermark = stream PDF dari ReadSeeker, apply text watermark
// diagonal di setiap halaman, return resulting bytes.
//
// pdfcpu API: TextWatermark builds *Watermark dari text + descriptor string.
// Descriptor "scale:1 abs, rot:45, opacity:0.3, col: 0.6 0.6 0.6" — abu-abu
// transparan diagonal supaya tidak menutup konten asli.
//
// Trade-off vs alternative library (unipdf): pdfcpu pure Go, MIT, lebih ringan;
// unipdf butuh license komersil untuk produksi.
func applyTextWatermark(rs io.ReadSeeker, text string) ([]byte, error) {
	wm, err := pdfapi.TextWatermark(
		text,
		"scale:0.9 abs, rot:45, opacity:0.25, col: 0.5 0.5 0.5, pos:c, points:24",
		true,  // onTop
		false, // update existing
		pdftypes.POINTS,
	)
	if err != nil {
		return nil, fmt.Errorf("watermark: parse: %w", err)
	}

	var out bytes.Buffer
	if err := pdfapi.AddWatermarks(rs, &out, nil, wm, nil); err != nil {
		return nil, fmt.Errorf("watermark: apply: %w", err)
	}
	return out.Bytes(), nil
}

// concept:pdf-watermark:end

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

// suratAttachmentReplaceHandler handle PATCH /api/surat/{id}/attachments/{att_id}/replace.
// Multipart body dengan single file part (form name = "file"). Atomic replace:
// new file stored, new row inserted, old row marked is_active=false + replaced_by.
func suratAttachmentReplaceHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		suratID := r.PathValue("id")
		oldAttID := r.PathValue("att_id")
		if suratID == "" || oldAttID == "" {
			writeError(w, http.StatusBadRequest, "id and att_id required")
			return
		}

		surat, err := d.AttachmentStore.GetSuratByID(r.Context(), suratID)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "surat tidak ditemukan")
			return
		}
		if err != nil {
			d.Logger.Error("replace: surat lookup", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if surat.AccessLevel == "secret" && !hasReadSecret(claims.Roles) {
			writeError(w, http.StatusForbidden, "akses surat rahasia ditolak")
			return
		}

		// Verify old attachment ada dan masih aktif
		oldAtt, err := d.AttachmentStore.AttachmentByID(r.Context(), oldAttID)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "attachment tidak ditemukan / sudah direplace")
			return
		}
		if err != nil {
			d.Logger.Error("replace: get old", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if oldAtt.SuratID != suratID {
			writeError(w, http.StatusNotFound, "attachment bukan milik surat ini")
			return
		}

		mr, err := r.MultipartReader()
		if err != nil {
			writeError(w, http.StatusBadRequest, "expected multipart/form-data")
			return
		}

		// Stream first file part
		var savedPath, mimeType, fileName string
		var fileSize int64
		for {
			part, err := mr.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				writeError(w, http.StatusBadRequest, "malformed multipart body")
				return
			}
			if part.FileName() == "" {
				_, _ = io.Copy(io.Discard, part)
				_ = part.Close()
				continue
			}
			fileName = part.FileName()
			savedPath, fileSize, mimeType, err = streamPartToDisk(part, d.AttachmentRoot)
			_ = part.Close()
			if err != nil {
				if errors.Is(err, errFileTooLarge) {
					writeError(w, http.StatusRequestEntityTooLarge,
						fmt.Sprintf("file lebih besar dari %d byte", maxFileSize))
					return
				}
				d.Logger.Error("replace: stream", "err", err)
				writeError(w, http.StatusInternalServerError, "internal error")
				return
			}
			break // hanya ambil first file
		}

		if savedPath == "" {
			writeError(w, http.StatusBadRequest, "file part tidak ditemukan")
			return
		}

		if !mimeAllowed(mimeType) {
			_ = os.Remove(filepath.Join(d.AttachmentRoot, savedPath))
			writeError(w, http.StatusUnsupportedMediaType,
				fmt.Sprintf("MIME type %q tidak diizinkan", mimeType))
			return
		}

		newAttID, err := uuid7.New()
		if err != nil {
			_ = os.Remove(filepath.Join(d.AttachmentRoot, savedPath))
			d.Logger.Error("uuid: generate", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		input := store.AttachmentInput{
			ID:         newAttID.String(),
			SuratID:    suratID,
			Role:       oldAtt.Role,
			FileName:   fileName,
			FilePath:   savedPath,
			FileSize:   fileSize,
			MimeType:   mimeType,
			UploadedBy: claims.Sub,
		}
		if err := d.AttachmentStore.ReplaceAttachment(r.Context(), oldAttID, input); err != nil {
			_ = os.Remove(filepath.Join(d.AttachmentRoot, savedPath))
			if errors.Is(err, store.ErrAlreadyReplaced) {
				writeError(w, http.StatusConflict, "attachment sudah pernah direplace — gunakan versi terkini")
				return
			}
			d.Logger.Error("replace: store", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		writeJSON(w, http.StatusOK, attachmentDTO{
			ID: newAttID.String(), Role: oldAtt.Role, FileName: fileName,
			FileSize: fileSize, MimeType: mimeType,
		})
	}
}

type attachmentVersionDTO struct {
	ID           string `json:"id"`
	FileName     string `json:"file_name"`
	FileSize     int64  `json:"file_size"`
	MimeType     string `json:"mime_type"`
	IsActive     bool   `json:"is_active"`
	ReplacedBy   *string `json:"replaced_by,omitempty"`
	UploadedBy   string `json:"uploaded_by"`
	UploaderName string `json:"uploader_name"`
	UploadedAt   string `json:"uploaded_at"` // RFC3339
}

func suratAttachmentVersionsHandler(d Deps) http.HandlerFunc {
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
			d.Logger.Error("versions: surat", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		if surat.AccessLevel == "secret" && !hasReadSecret(claims.Roles) {
			writeError(w, http.StatusForbidden, "akses surat rahasia ditolak")
			return
		}

		versions, err := d.AttachmentStore.ListAttachmentVersions(r.Context(), attID)
		if err != nil {
			d.Logger.Error("versions: list", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		out := make([]attachmentVersionDTO, 0, len(versions))
		for _, v := range versions {
			out = append(out, attachmentVersionDTO{
				ID: v.ID, FileName: v.FileName, FileSize: v.FileSize,
				MimeType: v.MimeType, IsActive: v.IsActive,
				ReplacedBy: v.ReplacedBy,
				UploadedBy: v.UploadedBy, UploaderName: v.UploaderName,
				UploadedAt: v.UploadedAt.Format("2006-01-02T15:04:05Z07:00"),
			})
		}
		writeJSONWithEdu(w, r, d, http.StatusOK, map[string]any{"versions": out}, func() *EduPayload {
			return &EduPayload{
				Operation: "traverse_linked_list_attachment_versions",
				DataStructures: []string{
					"Singly linked list (replaced_by pointer)",
					"B-tree index idx_attachments_replaced",
				},
				Complexity: map[string]interface{}{
					"theoretical": "O(k) where k = chain depth",
					"actual": map[string]interface{}{
						"version_count": len(versions),
					},
				},
				SQL: "WITH RECURSIVE chain AS (\n" +
					"  -- Walk backward dari given ID ke head (predecessor lookup)\n" +
					"  SELECT * FROM surat_attachments WHERE id = $1\n" +
					"  UNION ALL\n" +
					"  SELECT prev.* FROM surat_attachments prev JOIN chain c\n" +
					"    ON prev.replaced_by = c.id\n" +
					"),\n" +
					"full_chain AS (\n" +
					"  -- Walk forward dari head ke tail (forward lookup)\n" +
					"  SELECT * FROM (SELECT id FROM chain ORDER BY depth DESC LIMIT 1) head\n" +
					"  UNION ALL\n" +
					"  SELECT next.* FROM surat_attachments next JOIN full_chain fc\n" +
					"    ON next.id = fc.replaced_by\n" +
					")\n" +
					"SELECT * FROM full_chain ORDER BY fdepth ASC;",
				ConceptIDs: []string{"linked-list-version-chain", "recursive-cte"},
			}
		})
	}
}

