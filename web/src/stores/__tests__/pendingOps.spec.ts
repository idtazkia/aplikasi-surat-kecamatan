import { describe, it, expect, beforeEach, vi } from "vitest";
import { setActivePinia, createPinia } from "pinia";

// Mock Dexie liveQuery dan db.ops via vi.hoisted untuk menghindari
// "Cannot access before initialization" error (vi.mock hoisted ke top
// dan mocks butuh references yang juga hoisted).
const { subscribers, mockUnsubscribe, mockOpsTable } = vi.hoisted(() => ({
  subscribers: [] as Array<(val: unknown) => void>,
  mockUnsubscribe: vi.fn(),
  mockOpsTable: {
    filter: vi.fn().mockReturnThis(),
    sortBy: vi.fn().mockResolvedValue([]),
  },
}));

vi.mock("dexie", () => ({
  liveQuery: (queryFn: () => Promise<unknown>) => ({
    subscribe: ({ next, error }: { next: (v: unknown) => void; error?: (e: unknown) => void }) => {
      subscribers.push(next);
      void queryFn().then(next).catch(error);
      return { unsubscribe: mockUnsubscribe };
    },
  }),
}));

vi.mock("@/offline/db", () => ({
  db: { ops: mockOpsTable },
}));

// Import after mock setup
import { usePendingOpsStore } from "../pendingOps";

describe("usePendingOpsStore", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    subscribers.length = 0;
    mockUnsubscribe.mockReset();
    mockOpsTable.filter.mockClear().mockReturnThis();
    mockOpsTable.sortBy.mockReset().mockResolvedValue([]);
  });

  it("default count 0 + items kosong", () => {
    const store = usePendingOpsStore();
    expect(store.count).toBe(0);
    expect(store.items).toEqual([]);
  });

  it("start subscribe ke liveQuery + update items saat emit", async () => {
    const fakeOps = [
      { client_op_id: "op1", entity_type: "surat", action: "update", synced_at: undefined },
      { client_op_id: "op2", entity_type: "surat", action: "update", synced_at: undefined },
    ];
    mockOpsTable.sortBy.mockResolvedValueOnce(fakeOps);

    const store = usePendingOpsStore();
    store.start();

    // Wait initial async emission
    await new Promise((r) => setTimeout(r, 10));
    expect(store.count).toBe(2);
    expect(store.items).toEqual(fakeOps);
  });

  it("start dipanggil ulang tidak duplicate subscribe", () => {
    const store = usePendingOpsStore();
    store.start();
    const firstCount = subscribers.length;
    store.start();
    expect(subscribers.length).toBe(firstCount); // tidak nambah
  });

  it("stop unsubscribe + bisa di-restart", () => {
    const store = usePendingOpsStore();
    store.start();
    store.stop();
    expect(mockUnsubscribe).toHaveBeenCalled();

    // Restart
    store.start();
    expect(subscribers.length).toBeGreaterThan(0);
  });

  it("stop tanpa start tidak error", () => {
    const store = usePendingOpsStore();
    expect(() => store.stop()).not.toThrow();
  });
});
