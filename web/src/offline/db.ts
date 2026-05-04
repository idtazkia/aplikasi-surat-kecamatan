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

// PendingOp = mutation yang menunggu di-sync ke server. Schema sengaja
// flat — entity_type + action menentukan siapa yang interpret field_changes.
export interface PendingOp {
  client_op_id: string;        // UUIDv7, idempotency key
  entity_type: "surat" | "komentar";
  entity_id: string;
  action: "create" | "update" | "delete" | "append";
  field_changes: Record<string, unknown>;
  client_timestamp: string;    // RFC3339, saat op dibuat di klien
  synced_at?: string;          // RFC3339, saat server konfirmasi applied
  error?: string;              // populated kalau retry gagal
  retry_count: number;
  next_retry_at?: string;      // RFC3339, untuk exponential backoff scheduling
}

export type SuratKecDB = Dexie & {
  surat: EntityTable<CachedSurat, "id">;
  instansi: EntityTable<CachedInstansi, "id">;
  klasifikasi: EntityTable<CachedLookup, "id">;
  sifat: EntityTable<CachedLookup, "id">;
  meta: EntityTable<CachedMeta, "key">;
  ops: EntityTable<PendingOp, "client_op_id">;
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

// v2 — tambah ops store untuk Fase 4 offline write queue.
// Index: client_op_id PK + secondary index synced_at (untuk filter pending),
// next_retry_at (untuk drain scheduler).
db.version(2).stores({
  ops: "client_op_id, synced_at, next_retry_at, entity_type",
});
