package server

import (
	"context"
	"net/http"
	"time"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
)

// SyncStore subset interface untuk handler sync snapshot.
type SyncStore interface {
	GetSyncSnapshot(ctx context.Context, since *time.Time, includeSecret bool) (*store.SyncSnapshot, error)
}

type syncSuratDTO struct {
	ID              string  `json:"id"`
	Jenis           string  `json:"jenis"`
	NomorSurat      string  `json:"nomor_surat"`
	Perihal         string  `json:"perihal"`
	TanggalSurat    string  `json:"tanggal_surat"`
	TanggalTerima   *string `json:"tanggal_terima,omitempty"`
	InstansiID      string  `json:"instansi_id"`
	InstansiNama    string  `json:"instansi_nama"`
	KlasifikasiKode *string `json:"klasifikasi_kode,omitempty"`
	SifatKode       *string `json:"sifat_kode,omitempty"`
	AccessLevel     string  `json:"access_level"`
}

type syncInstansiDTO struct {
	ID          string   `json:"id"`
	NamaKanonik string   `json:"nama_kanonik"`
	Aliases     []string `json:"aliases"`
	Alamat      *string  `json:"alamat,omitempty"`
	Kontak      *string  `json:"kontak,omitempty"`
}

type syncLookupDTO struct {
	ID        string  `json:"id"`
	Kode      string  `json:"kode"`
	Nama      string  `json:"nama"`
	Deskripsi *string `json:"deskripsi,omitempty"`
}

type syncSnapshotDTO struct {
	Watermark        time.Time         `json:"watermark"`     // RFC3339 — klien kirim balik di sync berikutnya
	Surat            []syncSuratDTO    `json:"surat"`
	SuratDeletedIDs  []string          `json:"surat_deleted_ids"` // tombstones
	Instansi         []syncInstansiDTO `json:"instansi"`
	Klasifikasi      []syncLookupDTO   `json:"klasifikasi"`
	Sifat            []syncLookupDTO   `json:"sifat"`
}

// syncSnapshotHandler GET /api/sync/snapshot?since=<RFC3339>
//
// `since` optional — kalau kosong, return full snapshot. Kalau ada,
// return delta sejak watermark itu (rows updated, plus tombstone IDs).
//
// Klien expected:
//  1. First sync: GET /api/sync/snapshot (no since) → simpan watermark
//  2. Subsequent: GET /api/sync/snapshot?since={lastWatermark} → upsert delta,
//     hapus tombstone IDs dari cache, simpan new watermark
func syncSnapshotHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		var since *time.Time
		if v := r.URL.Query().Get("since"); v != "" {
			t, err := time.Parse(time.RFC3339Nano, v)
			if err != nil {
				writeError(w, http.StatusBadRequest, "since format invalid (RFC3339)")
				return
			}
			since = &t
		}

		snap, err := d.SyncStore.GetSyncSnapshot(r.Context(), since, hasReadSecret(claims.Roles))
		if err != nil {
			d.Logger.Error("sync: snapshot", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		out := syncSnapshotDTO{
			Watermark:       snap.Watermark,
			Surat:           make([]syncSuratDTO, 0, len(snap.Surat)),
			SuratDeletedIDs: snap.SuratDeletedIDs,
			Instansi:        make([]syncInstansiDTO, 0, len(snap.Instansi)),
			Klasifikasi:     make([]syncLookupDTO, 0, len(snap.Klasifikasi)),
			Sifat:           make([]syncLookupDTO, 0, len(snap.Sifat)),
		}
		for _, s := range snap.Surat {
			dto := syncSuratDTO{
				ID: s.ID, Jenis: s.Jenis, NomorSurat: s.NomorSurat, Perihal: s.Perihal,
				TanggalSurat: s.TanggalSurat.Format("2006-01-02"),
				InstansiID:   s.InstansiID, InstansiNama: s.InstansiNama,
				KlasifikasiKode: s.KlasifikasiKode, SifatKode: s.SifatKode,
				AccessLevel: s.AccessLevel,
			}
			if s.TanggalTerima != nil {
				str := s.TanggalTerima.Format("2006-01-02")
				dto.TanggalTerima = &str
			}
			out.Surat = append(out.Surat, dto)
		}
		for _, i := range snap.Instansi {
			out.Instansi = append(out.Instansi, syncInstansiDTO{
				ID: i.ID, NamaKanonik: i.NamaKanonik, Aliases: i.Aliases,
				Alamat: i.Alamat, Kontak: i.Kontak,
			})
		}
		for _, k := range snap.Klasifikasi {
			out.Klasifikasi = append(out.Klasifikasi, syncLookupDTO{
				ID: k.ID, Kode: k.Kode, Nama: k.Nama, Deskripsi: k.Deskripsi,
			})
		}
		for _, s := range snap.Sifat {
			out.Sifat = append(out.Sifat, syncLookupDTO{
				ID: s.ID, Kode: s.Kode, Nama: s.Nama, Deskripsi: s.Deskripsi,
			})
		}

		// Cache hint untuk service worker — sync snapshot adalah snapshot read,
		// klien yang manage staleness via watermark.
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, http.StatusOK, out)
	}
}
