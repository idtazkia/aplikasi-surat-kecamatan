package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
)

// ReconciliationStore subset interface untuk dedup queue handlers.
type ReconciliationStore interface {
	CreateReconciliationGroupIfDuplicate(ctx context.Context, suratID string) (string, error)
	ListReconciliationGroups(ctx context.Context, includeResolved bool) ([]store.ReconciliationGroup, error)
	GetReconciliationDetail(ctx context.Context, groupID string) (*store.ReconciliationDetail, error)
	MergeReconciliationGroup(ctx context.Context, groupID, canonicalSuratID, resolvedBy string) error
	KeepBothReconciliationGroup(ctx context.Context, groupID, resolvedBy string) error
}

type reconGroupDTO struct {
	GroupID       string     `json:"group_id"`
	DedupKey      string     `json:"dedup_key"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	ResolvedAt    *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy    *string    `json:"resolved_by,omitempty"`
	SuratCount    int        `json:"surat_count"`
	InstansiNama  string     `json:"instansi_nama"`
	NomorSurat    string     `json:"nomor_surat"`
	TanggalTerima *string    `json:"tanggal_terima,omitempty"`
}

type reconDetailDTO struct {
	GroupID   string                `json:"group_id"`
	DedupKey  string                `json:"dedup_key"`
	Status    string                `json:"status"`
	CreatedAt time.Time             `json:"created_at"`
	Surats    []suratDetailResponse `json:"surats"`
}

func reconciliationListHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		includeResolved := r.URL.Query().Get("include_resolved") == "true"
		groups, err := d.ReconStore.ListReconciliationGroups(r.Context(), includeResolved)
		if err != nil {
			d.Logger.Error("recon list", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		out := make([]reconGroupDTO, 0, len(groups))
		for _, g := range groups {
			dto := reconGroupDTO{
				GroupID: g.GroupID, DedupKey: g.DedupKey, Status: g.Status,
				CreatedAt: g.CreatedAt, ResolvedAt: g.ResolvedAt, ResolvedBy: g.ResolvedBy,
				SuratCount: g.SuratCount, InstansiNama: g.InstansiNama, NomorSurat: g.NomorSurat,
			}
			if g.TanggalTerima != nil {
				str := g.TanggalTerima.Format("2006-01-02")
				dto.TanggalTerima = &str
			}
			out = append(out, dto)
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out})
	}
}

func reconciliationDetailHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		groupID := r.PathValue("group_id")
		if groupID == "" {
			writeError(w, http.StatusBadRequest, "group_id required")
			return
		}

		detail, err := d.ReconStore.GetReconciliationDetail(r.Context(), groupID)
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "group tidak ditemukan")
			return
		}
		if err != nil {
			d.Logger.Error("recon detail", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		// Reuse suratDetailResponse format dari surat handler — klien sudah
		// punya rendering untuk DTO ini.
		surats := make([]suratDetailResponse, 0, len(detail.Surats))
		for _, s := range detail.Surats {
			surats = append(surats, buildSuratDetailDTO(s))
		}

		writeJSON(w, http.StatusOK, reconDetailDTO{
			GroupID:   detail.GroupID,
			DedupKey:  detail.DedupKey,
			Status:    detail.Status,
			CreatedAt: detail.CreatedAt,
			Surats:    surats,
		})
	}
}

type mergeRequest struct {
	CanonicalSuratID string `json:"canonical_surat_id"`
}

func reconciliationMergeHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		groupID := r.PathValue("group_id")
		if groupID == "" {
			writeError(w, http.StatusBadRequest, "group_id required")
			return
		}

		var req mergeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.CanonicalSuratID == "" {
			writeError(w, http.StatusBadRequest, "canonical_surat_id wajib")
			return
		}

		if err := d.ReconStore.MergeReconciliationGroup(r.Context(), groupID, req.CanonicalSuratID, claims.Sub); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "group tidak ditemukan / sudah resolved")
				return
			}
			d.Logger.Error("recon merge", "err", err)
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "merged"})
	}
}

func reconciliationKeepBothHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		groupID := r.PathValue("group_id")
		if groupID == "" {
			writeError(w, http.StatusBadRequest, "group_id required")
			return
		}

		if err := d.ReconStore.KeepBothReconciliationGroup(r.Context(), groupID, claims.Sub); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusNotFound, "group tidak ditemukan / sudah resolved")
				return
			}
			d.Logger.Error("recon keep both", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "kept_both"})
	}
}

// buildSuratDetailDTO refactor extract dari suratDetailHandler — dipakai juga
// di reconciliation detail untuk render side-by-side.
func buildSuratDetailDTO(detail store.SuratDetail) suratDetailResponse {
	resp := suratDetailResponse{
		suratListItemDTO:     toListDTO(detail.SuratListItem),
		DeskripsiKlasifikasi: detail.DeskripsiKlasifikasi,
		NamaSifat:            detail.NamaSifat,
		Attachments:          make([]suratAttachmentDTO, 0, len(detail.Attachments)),
		Predecessors:         make([]suratReferenceDTO, 0, len(detail.Predecessors)),
		Successors:           make([]suratReferenceDTO, 0, len(detail.Successors)),
		Tembusan:             make([]suratTembusanDTO, 0, len(detail.Tembusan)),
	}
	for _, a := range detail.Attachments {
		resp.Attachments = append(resp.Attachments, suratAttachmentDTO{
			ID: a.ID, Role: a.Role, FileName: a.FileName,
			FileSize: a.FileSize, MimeType: a.MimeType, UploadedAt: a.UploadedAt,
		})
	}
	for _, r := range detail.Predecessors {
		resp.Predecessors = append(resp.Predecessors, toRefDTO(r))
	}
	for _, r := range detail.Successors {
		resp.Successors = append(resp.Successors, toRefDTO(r))
	}
	for _, t := range detail.Tembusan {
		resp.Tembusan = append(resp.Tembusan, suratTembusanDTO{
			ID: t.ID, InstansiID: t.InstansiID, InstansiNama: t.InstansiNama,
			ExternalText: t.ExternalText, Urutan: t.Urutan,
		})
	}
	return resp
}
