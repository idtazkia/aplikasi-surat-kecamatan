import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { runSync, getSyncMeta } from "@/offline/sync";

// useOfflineStore = single source of truth untuk status koneksi + sync state.
// Logic ini dipisah dari views supaya banner indikator dan auto-sync bisa
// share state tanpa duplicating navigator.onLine listeners.
export const useOfflineStore = defineStore("offline", () => {
  const online = ref<boolean>(typeof navigator !== "undefined" ? navigator.onLine : true);
  const syncing = ref<boolean>(false);
  const lastSyncAt = ref<string | null>(null);
  const lastSyncError = ref<string | null>(null);

  function bindBrowserEvents() {
    if (typeof window === "undefined") return;
    window.addEventListener("online", () => {
      online.value = true;
      // Auto-sync saat reconnect — staf yang offline beberapa hari langsung
      // dapatkan delta tanpa harus refresh manual.
      void sync();
    });
    window.addEventListener("offline", () => {
      online.value = false;
    });
  }

  async function refreshMetaFromCache() {
    const m = await getSyncMeta();
    lastSyncAt.value = m.last_sync_at ?? null;
  }

  async function sync(): Promise<void> {
    if (!online.value) return; // tidak ada gunanya try saat offline
    if (syncing.value) return; // dedup concurrent triggers
    syncing.value = true;
    lastSyncError.value = null;
    try {
      const result = await runSync();
      lastSyncAt.value = new Date().toISOString();
      // Watermark dari server di-store di IndexedDB; di sini kita hanya track
      // wallclock untuk indikator UI.
      void result;
    } catch (e) {
      lastSyncError.value = e instanceof Error ? e.message : String(e);
      console.error("offline sync error:", e);
    } finally {
      syncing.value = false;
    }
  }

  const lastSyncRelative = computed<string>(() => {
    if (!lastSyncAt.value) return "belum pernah";
    const diffMs = Date.now() - new Date(lastSyncAt.value).getTime();
    const minutes = Math.floor(diffMs / 60000);
    if (minutes < 1) return "baru saja";
    if (minutes < 60) return `${minutes} menit lalu`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours} jam lalu`;
    const days = Math.floor(hours / 24);
    return `${days} hari lalu`;
  });

  return {
    online,
    syncing,
    lastSyncAt,
    lastSyncError,
    lastSyncRelative,
    bindBrowserEvents,
    refreshMetaFromCache,
    sync,
  };
});
