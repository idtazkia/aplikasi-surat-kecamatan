// Package student-mode `_edu` payload — append konten edukasi ke response API
// saat env STUDENT_MODE_ENABLED=true AND user punya akses (role student
// always-on; admin/dev opt-in via header).
//
// Production: STUDENT_MODE_ENABLED=false hard-disable seluruh logic — fail-safe
// untuk mencegah leak data internal (SQL queries) ke pengguna umum.

package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
)

// EduPayload struct bentuk untuk JSON serialization. Field optional supaya
// endpoint bisa kontribusi sebagian (mis. cuma concept_ids tanpa SQL).
type EduPayload struct {
	Operation      string                 `json:"operation"`
	DataStructures []string               `json:"data_structures,omitempty"`
	Complexity     map[string]interface{} `json:"complexity,omitempty"`
	SQL            string                 `json:"sql,omitempty"`
	Explain        string                 `json:"explain,omitempty"`
	ConceptIDs     []string               `json:"concept_ids,omitempty"`
}

// eduEnabled = check apakah _edu block harus di-inject untuk request ini.
//
// Aturan (sengaja minimal — hardcoded role check, no opt-in dari klien):
//   - STUDENT_MODE_ENABLED=false → never inject (production hard-disable)
//   - role 'student' → always inject
//   - role lain → never inject (admin yang ingin debug login as student)
func eduEnabled(d Deps, r *http.Request) bool {
	if !d.StudentMode {
		return false
	}
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return false
	}
	for _, role := range claims.Roles {
		if role == "student" {
			return true
		}
	}
	return false
}

// writeJSONWithEdu mirip writeJSON tapi append `_edu` field kalau eduEnabled.
// Trade-off vs alternative (envelope wrapping): inject ke shape body asli
// supaya tidak break API contract — frontend tetap dapat data di posisi lama.
//
// Cost: 2x marshal/unmarshal saat student mode active. Acceptable karena
// student mode = development/learning context, bukan hot path produksi.
func writeJSONWithEdu(w http.ResponseWriter, r *http.Request, d Deps, code int, body any, eduBuilder func() *EduPayload) {
	if !eduEnabled(d, r) {
		writeJSON(w, code, body)
		return
	}
	edu := eduBuilder()
	if edu == nil {
		writeJSON(w, code, body)
		return
	}

	// Marshal body, decode ke map, inject `_edu`, re-marshal.
	raw, err := json.Marshal(body)
	if err != nil {
		d.Logger.Error("edu: marshal body", "err", err)
		writeJSON(w, code, body)
		return
	}
	var asMap map[string]interface{}
	if err := json.Unmarshal(raw, &asMap); err != nil {
		// Body bukan object (mis. array). Wrap dalam envelope sebagai fallback —
		// klien yang request student mode harus expect potential envelope.
		d.Logger.Warn("edu: body bukan object, fallback envelope", "err", err)
		writeJSON(w, code, map[string]interface{}{"data": body, "_edu": edu})
		return
	}
	asMap["_edu"] = edu
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(asMap)
}

// eduCtxKey ada kalau handler perlu pass _edu data via context daripada
// closure (rare case — biasanya closure cukup).
type eduCtxKey struct{}

// EduFromContext ambil EduPayload yang sudah di-set di context — utility
// untuk middleware.
func EduFromContext(ctx context.Context) *EduPayload {
	if v, ok := ctx.Value(eduCtxKey{}).(*EduPayload); ok {
		return v
	}
	return nil
}

// WithEduContext attach EduPayload ke context (chain middleware bisa pakai).
func WithEduContext(ctx context.Context, edu *EduPayload) context.Context {
	return context.WithValue(ctx, eduCtxKey{}, edu)
}
