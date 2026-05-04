import Dexie, { type EntityTable } from "dexie";

// Cached projection — sengaja flat (no nested objects) supaya cepat
// untuk index lookups + JSON serialization saat upsert dari sync API.
export interface CachedSurat {
  id: string;
  jenis: "masuk" | "keluar";
  nomor_surat: string;
  perihal: string;
  tanggal_surat: string;       // YYYY-MM-DD
  tanggal_terima?: string;
  instansi_id: string;
  instansi_nama: string;       // denormalized untuk avoid join saat list
  klasifikasi_kode?: string;
  sifat_kode?: string;
  access_level: "public" | "restricted" | "secret";
}

export interface CachedInstansi {
  id: string;
  nama_kanonik: string;
  aliases: string[];
  alamat?: string;
  kontak?: string;
}

export interface CachedLookup {
  id: string;
  kode: string;
  nama: string;
  deskripsi?: string;
}

// Meta: simpan watermark sync terakhir + last_sync_at untuk indikator UI.
export interface CachedMeta {
  key: "sync";
  watermark?: string;          // RFC3339, dari server
  last_sync_at?: string;       // RFC3339, client-side wallclock
}

export type SuratKecDB = Dexie & {
  surat: EntityTable<CachedSurat, "id">;
  instansi: EntityTable<CachedInstansi, "id">;
  klasifikasi: EntityTable<CachedLookup, "id">;
  sifat: EntityTable<CachedLookup, "id">;
  meta: EntityTable<CachedMeta, "key">;
};

// Schema versioning: kalau ubah index, bump version. Dexie handle migration
// untuk add/drop index secara otomatis (data tetap), tapi rename property
// butuh upgrade hook eksplisit.
export const db = new Dexie("surat-kec-cache") as SuratKecDB;
db.version(1).stores({
  // Index: id PK + secondary indexes untuk query yang sering — tanggal_terima
  // (sort daftar surat masuk), instansi_id (filter per instansi),
  // nomor_surat (lookup deduplikasi).
  surat: "id, tanggal_terima, instansi_id, nomor_surat, jenis, access_level",
  instansi: "id, nama_kanonik",
  klasifikasi: "id, kode",
  sifat: "id, kode",
  meta: "key",
});
