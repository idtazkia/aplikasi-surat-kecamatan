import { describe, it, expect, beforeEach, vi } from "vitest";
import { setActivePinia, createPinia } from "pinia";
import { useEduPanelStore } from "../eduPanel";

describe("eduPanel store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.restoreAllMocks();
  });

  it("default state: disabled, drawer closed, no payload", () => {
    const edu = useEduPanelStore();
    expect(edu.enabled).toBe(false);
    expect(edu.drawerOpen).toBe(false);
    expect(edu.lastPayload).toBeNull();
    expect(edu.links).toBeNull();
  });

  it("recordPayload tidak menyimpan kalau disabled (no leak ke production)", () => {
    const edu = useEduPanelStore();
    edu.recordPayload({ operation: "test" });
    expect(edu.lastPayload).toBeNull();
  });

  it("recordPayload menyimpan saat enabled", () => {
    const edu = useEduPanelStore();
    edu.enabled = true;
    edu.recordPayload({
      operation: "list_surat",
      complexity: { theoretical: "O(log n + k)" },
      concept_ids: ["btree-partial-index-soft-delete"],
    });
    expect(edu.lastPayload?.operation).toBe("list_surat");
    expect(edu.lastPayload?.concept_ids).toContain("btree-partial-index-soft-delete");
  });

  it("loadLinks fetch dari /concept-links.json", async () => {
    const fakeData = {
      generated_from_commit: "abc123",
      repo_slug: "idtazkia/aplikasi-surat-kecamatan",
      links: [
        {
          id: "btree-partial-index-soft-delete",
          permalink: "https://github.com/...",
          file: "db/migrations/schema/0001_init.sql",
          start_line: 100,
          end_line: 110,
        },
      ],
    };
    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      json: async () => fakeData,
    } as never);

    const edu = useEduPanelStore();
    await edu.loadLinks();

    expect(edu.links).toEqual(fakeData);
    expect(global.fetch).toHaveBeenCalledWith("/concept-links.json");
  });

  it("loadLinks silent fail kalau fetch error (offline-friendly)", async () => {
    global.fetch = vi.fn().mockRejectedValueOnce(new Error("network"));

    const edu = useEduPanelStore();
    await edu.loadLinks(); // no throw

    expect(edu.links).toBeNull();
  });

  it("linkByID lookup map", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        generated_from_commit: "abc",
        repo_slug: "x/y",
        links: [
          { id: "uuid-v7-generation", permalink: "url-1", file: "f1", start_line: 1, end_line: 5 },
          { id: "jwt-hmac-sign-verify", permalink: "url-2", file: "f2", start_line: 10, end_line: 20 },
        ],
      }),
    } as never);

    const edu = useEduPanelStore();
    await edu.loadLinks();

    expect(edu.linkByID.get("uuid-v7-generation")?.permalink).toBe("url-1");
    expect(edu.linkByID.get("jwt-hmac-sign-verify")?.permalink).toBe("url-2");
    expect(edu.linkByID.get("not-exist")).toBeUndefined();
  });

  it("loadLinks dengan resp not ok -> links tetap null", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      json: async () => ({}),
    } as never);

    const edu = useEduPanelStore();
    await edu.loadLinks();

    expect(edu.links).toBeNull();
  });
});
