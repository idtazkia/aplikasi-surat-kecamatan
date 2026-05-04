import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { setActivePinia, createPinia } from "pinia";
import { useOfflineStore } from "../offline";

// Mock @/offline/sync supaya tests fokus ke store logic, bukan Dexie I/O.
const mockRunSync = vi.fn();
const mockGetSyncMeta = vi.fn();
vi.mock("@/offline/sync", () => ({
  runSync: () => mockRunSync(),
  getSyncMeta: () => mockGetSyncMeta(),
  resetCache: vi.fn(),
}));

describe("useOfflineStore", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    mockRunSync.mockReset();
    mockGetSyncMeta.mockReset();
    mockGetSyncMeta.mockResolvedValue({});
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("default online state mengambil dari navigator.onLine", () => {
    const store = useOfflineStore();
    expect(store.online).toBe(typeof navigator !== "undefined" ? navigator.onLine : true);
  });

  it("bindBrowserEvents respond ke window online/offline events", async () => {
    const store = useOfflineStore();
    store.online = true;
    store.bindBrowserEvents();

    window.dispatchEvent(new Event("offline"));
    expect(store.online).toBe(false);

    mockRunSync.mockResolvedValueOnce({ watermark: "x", deltaSize: 0 });
    window.dispatchEvent(new Event("online"));
    expect(store.online).toBe(true);
    // Auto-sync triggered (await microtask flush)
    await Promise.resolve();
    expect(mockRunSync).toHaveBeenCalled();
  });

  it("sync skip kalau offline", async () => {
    const store = useOfflineStore();
    store.online = false;
    await store.sync();
    expect(mockRunSync).not.toHaveBeenCalled();
  });

  it("sync skip kalau syncing.value already true (dedup concurrent)", async () => {
    const store = useOfflineStore();
    store.online = true;
    store.syncing = true;
    await store.sync();
    expect(mockRunSync).not.toHaveBeenCalled();
  });

  it("sync sukses set lastSyncAt + clear error", async () => {
    const store = useOfflineStore();
    store.online = true;
    mockRunSync.mockResolvedValueOnce({ watermark: "2026-05-04T00:00:00Z", deltaSize: 5 });

    await store.sync();
    expect(store.lastSyncAt).not.toBeNull();
    expect(store.lastSyncError).toBeNull();
    expect(store.syncing).toBe(false);
  });

  it("sync gagal set lastSyncError", async () => {
    const store = useOfflineStore();
    store.online = true;
    mockRunSync.mockRejectedValueOnce(new Error("network down"));

    // suppress console.error noise
    const errSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    await store.sync();
    expect(store.lastSyncError).toBe("network down");
    expect(store.syncing).toBe(false);
    errSpy.mockRestore();
  });

  it("refreshMetaFromCache populate lastSyncAt dari Dexie", async () => {
    const store = useOfflineStore();
    mockGetSyncMeta.mockResolvedValueOnce({ last_sync_at: "2026-05-04T10:00:00Z", watermark: "x" });
    await store.refreshMetaFromCache();
    expect(store.lastSyncAt).toBe("2026-05-04T10:00:00Z");
  });

  describe("lastSyncRelative formatter", () => {
    it("'belum pernah' kalau lastSyncAt null", () => {
      const store = useOfflineStore();
      store.lastSyncAt = null;
      expect(store.lastSyncRelative).toBe("belum pernah");
    });

    it("'baru saja' untuk < 1 menit", () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date("2026-05-04T10:00:00Z"));
      const store = useOfflineStore();
      store.lastSyncAt = "2026-05-04T09:59:30Z"; // 30 detik lalu
      expect(store.lastSyncRelative).toBe("baru saja");
    });

    it("menit lalu untuk < 1 jam", () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date("2026-05-04T10:00:00Z"));
      const store = useOfflineStore();
      store.lastSyncAt = "2026-05-04T09:45:00Z"; // 15 menit lalu
      expect(store.lastSyncRelative).toBe("15 menit lalu");
    });

    it("jam lalu untuk < 24 jam", () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date("2026-05-04T10:00:00Z"));
      const store = useOfflineStore();
      store.lastSyncAt = "2026-05-04T07:00:00Z"; // 3 jam lalu
      expect(store.lastSyncRelative).toBe("3 jam lalu");
    });

    it("hari lalu untuk >= 24 jam", () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date("2026-05-04T10:00:00Z"));
      const store = useOfflineStore();
      store.lastSyncAt = "2026-05-02T10:00:00Z"; // 2 hari lalu
      expect(store.lastSyncRelative).toBe("2 hari lalu");
    });
  });
});
