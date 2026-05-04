import { defineStore } from "pinia";
import { ref } from "vue";
import { liveQuery } from "dexie";
import { db } from "@/offline/db";
import type { PendingOp } from "@/offline/db";

// usePendingOpsStore = reactive count + list pending ops, di-power oleh
// Dexie liveQuery (auto-rerun saat ops table berubah). Komponen NotificationBell
// pattern — single source of truth untuk badge UI.
export const usePendingOpsStore = defineStore("pendingOps", () => {
  const count = ref(0);
  const items = ref<PendingOp[]>([]);
  let unsubscribe: (() => void) | null = null;

  function start() {
    if (unsubscribe) return; // already started
    // liveQuery emit value setiap kali Dexie ops table change.
    const sub = liveQuery(async () => {
      const pending = await db.ops
        .filter((op) => !op.synced_at)
        .sortBy("client_timestamp");
      return pending;
    }).subscribe({
      next: (val) => {
        items.value = val;
        count.value = val.length;
      },
      error: (e) => {
        console.error("pendingOps liveQuery:", e);
      },
    });
    unsubscribe = () => sub.unsubscribe();
  }

  function stop() {
    if (unsubscribe) {
      unsubscribe();
      unsubscribe = null;
    }
  }

  return { count, items, start, stop };
});
