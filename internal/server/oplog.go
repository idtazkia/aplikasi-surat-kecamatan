package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
)

// OpLogStore subset interface untuk handler.
type OpLogStore interface {
	ApplyOperations(ctx context.Context, ops []store.ApplyOperationInput) ([]store.ApplyOperationResult, error)
}

type applyOperationsRequest struct {
	Operations []clientOperationDTO `json:"operations"`
}

type clientOperationDTO struct {
	ClientOpID      string          `json:"client_op_id"`
	EntityType      string          `json:"entity_type"`
	EntityID        string          `json:"entity_id"`
	Action          string          `json:"action"`
	FieldChanges    json.RawMessage `json:"field_changes"`
	ClientTimestamp time.Time       `json:"client_timestamp"`
}

type applyOperationsResponse struct {
	Results []applyOperationResultDTO `json:"results"`
}

type applyOperationResultDTO struct {
	ClientOpID string `json:"client_op_id"`
	Status     string `json:"status"` // applied | duplicate | rejected
	Reason     string `json:"reason,omitempty"`
}

// applyOperationsHandler POST /api/sync/operations
//
// Body: { operations: [{ client_op_id, entity_type, entity_id, action,
// field_changes, client_timestamp }, ...] }
//
// Per-op result di-return dengan status "applied" / "duplicate" / "rejected".
// Klien bertanggungjawab interpret hasil — duplicate boleh di-treat sukses
// (idempotency), rejected butuh inspect reason untuk decide retry vs surrender.
func applyOperationsHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		var req applyOperationsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if len(req.Operations) == 0 {
			writeError(w, http.StatusBadRequest, "operations kosong")
			return
		}
		const maxBatch = 100
		if len(req.Operations) > maxBatch {
			writeError(w, http.StatusBadRequest, "max 100 operations per request")
			return
		}

		inputs := make([]store.ApplyOperationInput, 0, len(req.Operations))
		for _, op := range req.Operations {
			if op.ClientOpID == "" || op.EntityType == "" || op.EntityID == "" || op.Action == "" {
				writeError(w, http.StatusBadRequest,
					"client_op_id, entity_type, entity_id, action wajib")
				return
			}
			inputs = append(inputs, store.ApplyOperationInput{
				ClientOpID:      op.ClientOpID,
				EntityType:      op.EntityType,
				EntityID:        op.EntityID,
				Action:          op.Action,
				FieldChanges:    op.FieldChanges,
				ClientTimestamp: op.ClientTimestamp,
				UserID:          claims.Sub,
			})
		}

		results, err := d.OpLogStore.ApplyOperations(r.Context(), inputs)
		if err != nil {
			d.Logger.Error("oplog: apply", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		out := applyOperationsResponse{
			Results: make([]applyOperationResultDTO, 0, len(results)),
		}
		for _, r := range results {
			out.Results = append(out.Results, applyOperationResultDTO{
				ClientOpID: r.ClientOpID,
				Status:     r.Status,
				Reason:     r.Reason,
			})
		}
		writeJSON(w, http.StatusOK, out)
	}
}
