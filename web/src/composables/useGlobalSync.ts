import { ref } from "vue";
import { useMessage } from "naive-ui";
import { useOfflineStore } from "@/stores/offline";
import { drainQueue } from "@/offline/opqueue";
import { notificationApi } from "@/api/surat";

// useGlobalSync = manual trigger semua sync channel sekaligus.
// Dipakai oleh tombol "🔄 Sync" di topbar.
//
// 3 channel paralel:
//   1. notificationApi.list() — pull notifikasi terbaru (replace polling 30s)
//   2. drainQueue() — push pending write ops (read-write app — Fase 4)
//   3. offline.sync() — refresh Dexie snapshot metadata (Fase 3)
//
// Polling 30s di NotificationBell + drainer 30s di opqueue tetap jalan
// sebagai fallback. Manual sync = explicit user action saat butuh refresh
// segera (mis. setelah camat assign disposisi, staf langsung klik refresh
// daripada nunggu 30 detik).
export function useGlobalSync() {
  const offline = useOfflineStore();
  const message = useMessage();
  const syncing = ref(false);

  async function runAll() {
    if (syncing.value) return;
    if (!offline.online) {
      message.warning("Tidak bisa sync — sedang offline");
      return;
    }
    syncing.value = true;
    try {
      const results = await Promise.allSettled([
        notificationApi.list(),
        drainQueue(),
        offline.sync(),
      ]);

      const errors = results
        .filter((r): r is PromiseRejectedResult => r.status === "rejected")
        .map((r) => String(r.reason));

      if (errors.length === 0) {
        message.success("Sinkronisasi selesai");
      } else {
        message.warning(`Sinkronisasi sebagian gagal: ${errors.join("; ")}`);
      }
    } finally {
      syncing.value = false;
    }
  }

  return { syncing, runAll };
}
