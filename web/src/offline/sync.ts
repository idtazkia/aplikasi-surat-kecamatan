import { apiClient } from "@/api/client";
import { db, type CachedSurat, type CachedInstansi, type CachedLookup } from "./db";

interface SyncSnapshotResponse {
  watermark: string;
  surat: CachedSurat[];
  surat_deleted_ids: string[];
  instansi: CachedInstansi[];
  klasifikasi: CachedLookup[];
  sifat: CachedLookup[];
}

// runSync = fetch /api/sync/snapshot dari watermark terakhir, upsert IndexedDB,
// hapus tombstone, simpan watermark baru. Idempotent — boleh dipanggil
// berulang kali (mis. saat reconnect online).
//
// Strategi:
//   - First sync: since=null → server return semua active rows
//   - Delta sync: since=last_watermark → server return hanya delta + tombstones
//
// Klasifikasi/sifat tidak punya updated_at di schema → server kirim full list
// setiap kali. Klien pakai bulkPut yang upsert by primary key (id).
export async function runSync(): Promise<{ watermark: string; deltaSize: number }> {
  const meta = await db.meta.get("sync");
  const since = meta?.watermark;

  const path = since
    ? `/api/sync/snapshot?since=${encodeURIComponent(since)}`
    : "/api/sync/snapshot";

  const resp = await apiClient.get<SyncSnapshotResponse>(path);

  const deltaSize =
    resp.surat.length +
    resp.surat_deleted_ids.length +
    resp.instansi.length +
    resp.klasifikasi.length +
    resp.sifat.length;

  // Atomic transaction across all stores supaya snapshot tetap konsisten.
  // Kalau salah satu gagal, rollback semua.
  await db.transaction("rw", [db.surat, db.instansi, db.klasifikasi, db.sifat, db.meta], async () => {
    if (resp.surat.length > 0) await db.surat.bulkPut(resp.surat);
    if (resp.surat_deleted_ids.length > 0) await db.surat.bulkDelete(resp.surat_deleted_ids);
    if (resp.instansi.length > 0) await db.instansi.bulkPut(resp.instansi);
    // Klasifikasi/sifat: server selalu kirim full list — clear & repopulate
    // untuk drop entries yang sudah tidak active.
    if (since == null) {
      // First sync: clear-then-bulk-put (untuk hilangkan stale row dari iterasi sebelumnya)
      await db.klasifikasi.clear();
      await db.sifat.clear();
    }
    if (resp.klasifikasi.length > 0) await db.klasifikasi.bulkPut(resp.klasifikasi);
    if (resp.sifat.length > 0) await db.sifat.bulkPut(resp.sifat);

    await db.meta.put({
      key: "sync",
      watermark: resp.watermark,
      last_sync_at: new Date().toISOString(),
    });
  });

  return { watermark: resp.watermark, deltaSize };
}

export async function getSyncMeta(): Promise<{ watermark?: string; last_sync_at?: string }> {
  const m = await db.meta.get("sync");
  return { watermark: m?.watermark, last_sync_at: m?.last_sync_at };
}

// resetCache — hapus semua data lokal. Dipanggil saat logout supaya tidak
// data leak antar user di shared device.
export async function resetCache(): Promise<void> {
  await db.transaction("rw", [db.surat, db.instansi, db.klasifikasi, db.sifat, db.meta], async () => {
    await db.surat.clear();
    await db.instansi.clear();
    await db.klasifikasi.clear();
    await db.sifat.clear();
    await db.meta.clear();
  });
}
