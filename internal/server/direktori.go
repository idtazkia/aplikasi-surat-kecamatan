package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/auth"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/store"
	"github.com/idtazkia/aplikasi-surat-kecamatan/internal/uuid7"
)

// DirektoriStore extends untuk master data ops.
type DirektoriStore interface {
	SearchInstansi(ctx context.Context, keyword string, limit int) ([]store.InstansiItem, error)
	CreateInstansi(ctx context.Context, in store.CreateInstansiInput) error
	GetInstansi(ctx context.Context, id string) (*store.InstansiItem, error)
	ListKlasifikasi(ctx context.Context) ([]store.LookupItem, error)
	ListSifat(ctx context.Context) ([]store.LookupItem, error)
}

type instansiDTO struct {
	ID          string   `json:"id"`
	NamaKanonik string   `json:"nama_kanonik"`
	Aliases     []string `json:"aliases"`
	Alamat      *string  `json:"alamat,omitempty"`
	Kontak      *string  `json:"kontak,omitempty"`
}

func instansiSearchHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		q := r.URL.Query().Get("q")
		limit := 20
		if v := r.URL.Query().Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				writeError(w, http.StatusBadRequest, "limit harus integer positif")
				return
			}
			limit = n
		}

		items, err := d.DirektoriStore.SearchInstansi(r.Context(), q, limit)
		if err != nil {
			d.Logger.Error("instansi: search", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		out := make([]instansiDTO, 0, len(items))
		for _, it := range items {
			aliases := it.Aliases
			if aliases == nil {
				aliases = []string{}
			}
			out = append(out, instansiDTO{
				ID: it.ID, NamaKanonik: it.NamaKanonik,
				Aliases: aliases, Alamat: it.Alamat, Kontak: it.Kontak,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": out})
	}
}

type createInstansiRequest struct {
	NamaKanonik string   `json:"nama_kanonik"`
	Aliases     []string `json:"aliases"`
	Alamat      *string  `json:"alamat,omitempty"`
	Kontak      *string  `json:"kontak,omitempty"`
}

func instansiCreateHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}

		var req createInstansiRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.NamaKanonik == "" {
			writeError(w, http.StatusBadRequest, "nama_kanonik required")
			return
		}

		newID, err := uuid7.New()
		if err != nil {
			d.Logger.Error("uuid: generate", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		aliases := req.Aliases
		if aliases == nil {
			aliases = []string{}
		}

		input := store.CreateInstansiInput{
			ID: newID.String(), NamaKanonik: req.NamaKanonik,
			Aliases: aliases, Alamat: req.Alamat, Kontak: req.Kontak,
		}

		if err := d.DirektoriStore.CreateInstansi(r.Context(), input); err != nil {
			if errors.Is(err, store.ErrConflict) {
				writeError(w, http.StatusConflict, "nama_kanonik sudah dipakai")
				return
			}
			d.Logger.Error("instansi: create", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{"id": newID.String()})
	}
}

type lookupDTO struct {
	ID        string  `json:"id"`
	Kode      string  `json:"kode"`
	Nama      string  `json:"nama"`
	Deskripsi *string `json:"deskripsi,omitempty"`
}

func klasifikasiListHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}
		items, err := d.DirektoriStore.ListKlasifikasi(r.Context())
		if err != nil {
			d.Logger.Error("klasifikasi: list", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": toLookupDTOs(items)})
	}
}

func sifatListHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "claims missing")
			return
		}
		items, err := d.DirektoriStore.ListSifat(r.Context())
		if err != nil {
			d.Logger.Error("sifat: list", "err", err)
			writeError(w, http.StatusInternalServerError, "internal error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": toLookupDTOs(items)})
	}
}

func toLookupDTOs(items []store.LookupItem) []lookupDTO {
	out := make([]lookupDTO, 0, len(items))
	for _, it := range items {
		out = append(out, lookupDTO{
			ID: it.ID, Kode: it.Kode, Nama: it.Nama, Deskripsi: it.Deskripsi,
		})
	}
	return out
}
