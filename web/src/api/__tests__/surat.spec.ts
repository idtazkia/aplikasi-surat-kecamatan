import { describe, it, expect, beforeEach, vi } from "vitest";
import { setActivePinia, createPinia } from "pinia";
import {
  suratApi,
  disposisiApi,
  komentarApi,
  dashboardApi,
  notificationApi,
  direktoriApi,
  reconciliationApi,
  statsApi,
} from "../surat";

// Helper untuk mock fetch result.
function mockJsonOnce(body: unknown, ok = true, status = 200) {
  global.fetch = vi.fn().mockResolvedValueOnce({
    ok,
    status,
    json: async () => body,
    blob: async () => new Blob([JSON.stringify(body)]),
    text: async () => JSON.stringify(body),
    headers: new Headers(),
  } as never);
}

beforeEach(() => {
  setActivePinia(createPinia());
  localStorage.clear();
  vi.restoreAllMocks();
});

describe("suratApi", () => {
  describe("list", () => {
    it("default (no params) → GET /api/surat tanpa querystring", async () => {
      mockJsonOnce({ items: [], next_cursor: null });
      await suratApi.list();
      const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      expect(call[0]).toBe("/api/surat");
    });

    it("compose querystring dari params", async () => {
      mockJsonOnce({ items: [], next_cursor: null });
      await suratApi.list({ jenis: "masuk", search: "pandemi", limit: 10 });
      const url = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0];
      expect(url).toContain("jenis=masuk");
      expect(url).toContain("search=pandemi");
      expect(url).toContain("limit=10");
    });

    it("skip undefined dan empty-string param", async () => {
      mockJsonOnce({ items: [], next_cursor: null });
      await suratApi.list({ jenis: undefined, search: "" });
      const url = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0];
      expect(url).toBe("/api/surat");
    });
  });

  describe("get/create/update/remove", () => {
    it("get → GET /api/surat/:id", async () => {
      mockJsonOnce({ id: "abc" });
      await suratApi.get("abc");
      const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      expect(call[0]).toBe("/api/surat/abc");
      expect(call[1].method).toBe("GET");
    });

    it("create → POST /api/surat dengan body", async () => {
      mockJsonOnce({ id: "new-id" });
      await suratApi.create({
        jenis: "masuk",
        nomor_surat: "X/1/2026",
        perihal: "Test",
        tanggal_surat: "2026-04-01",
        instansi_id: "i1",
        access_level: "public",
      });
      const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      expect(call[0]).toBe("/api/surat");
      expect(call[1].method).toBe("POST");
      expect(JSON.parse(call[1].body)).toMatchObject({ nomor_surat: "X/1/2026" });
    });

    it("update → PATCH /api/surat/:id", async () => {
      mockJsonOnce({ status: "updated" });
      await suratApi.update("abc", { perihal: "edit" });
      const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      expect(call[0]).toBe("/api/surat/abc");
      expect(call[1].method).toBe("PATCH");
    });

    it("remove → DELETE /api/surat/:id", async () => {
      mockJsonOnce({ status: "deleted" });
      await suratApi.remove("abc");
      const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      expect(call[0]).toBe("/api/surat/abc");
      expect(call[1].method).toBe("DELETE");
    });
  });

  describe("uploadAttachments", () => {
    it("compose FormData dengan field name primary/lampiran", async () => {
      localStorage.setItem(
        "surat-kec-auth",
        JSON.stringify({ accessToken: "tk", refreshToken: "r", userID: "u", roles: [] }),
      );
      global.fetch = vi.fn().mockResolvedValueOnce({
        ok: true,
        json: async () => ({ uploaded: [] }),
      } as never);

      const f1 = new File(["x"], "p.pdf", { type: "application/pdf" });
      const f2 = new File(["y"], "l.pdf", { type: "application/pdf" });
      await suratApi.uploadAttachments("s1", [
        { file: f1, role: "primary" },
        { file: f2, role: "lampiran" },
      ]);

      const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      expect(call[0]).toBe("/api/surat/s1/attachments");
      expect(call[1].method).toBe("POST");
      expect(call[1].body).toBeInstanceOf(FormData);
      expect(call[1].headers["Authorization"]).toBe("Bearer tk");
      const fd = call[1].body as FormData;
      expect(fd.has("primary")).toBe(true);
      expect(fd.has("lampiran")).toBe(true);
    });

    it("throw error dengan status code kalau response not ok", async () => {
      global.fetch = vi.fn().mockResolvedValueOnce({
        ok: false,
        status: 413,
        text: async () => "too large",
      } as never);
      const f = new File(["x"], "big.pdf");
      await expect(
        suratApi.uploadAttachments("s1", [{ file: f, role: "primary" }]),
      ).rejects.toThrow(/413/);
    });
  });

  describe("replaceAttachment", () => {
    it("PATCH dengan FormData berisi field 'file'", async () => {
      localStorage.setItem(
        "surat-kec-auth",
        JSON.stringify({ accessToken: "tk", refreshToken: "r", userID: "u", roles: [] }),
      );
      global.fetch = vi.fn().mockResolvedValueOnce({
        ok: true,
        json: async () => ({ id: "newID", file_name: "v2.pdf", file_size: 100, mime_type: "application/pdf" }),
      } as never);

      const f = new File(["x"], "v2.pdf", { type: "application/pdf" });
      const result = await suratApi.replaceAttachment("s1", "att1", f);
      expect(result.id).toBe("newID");

      const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      expect(call[0]).toBe("/api/surat/s1/attachments/att1/replace");
      expect(call[1].method).toBe("PATCH");
      const fd = call[1].body as FormData;
      expect(fd.has("file")).toBe(true);
    });
  });

  describe("fetchAttachmentPreviewBlobURL", () => {
    it("fetch with bearer token dan return blob URL", async () => {
      localStorage.setItem(
        "surat-kec-auth",
        JSON.stringify({ accessToken: "tk", refreshToken: "r", userID: "u", roles: [] }),
      );
      const fakeBlob = new Blob(["pdf"], { type: "application/pdf" });
      global.fetch = vi.fn().mockResolvedValueOnce({
        ok: true,
        blob: async () => fakeBlob,
      } as never);
      const createObjURL = vi.fn().mockReturnValue("blob:fake");
      global.URL.createObjectURL = createObjURL;

      const url = await suratApi.fetchAttachmentPreviewBlobURL("s1", "att1");
      expect(url).toBe("blob:fake");
      expect(createObjURL).toHaveBeenCalledWith(fakeBlob);

      const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      expect(call[0]).toBe("/api/surat/s1/attachments/att1/preview");
      expect(call[1].headers.Authorization).toBe("Bearer tk");
    });

    it("throw error kalau status not ok", async () => {
      global.fetch = vi.fn().mockResolvedValueOnce({
        ok: false,
        status: 404,
      } as never);
      await expect(
        suratApi.fetchAttachmentPreviewBlobURL("s1", "att1"),
      ).rejects.toThrow(/404/);
    });
  });

  describe("download/preview URL helpers", () => {
    it("attachmentDownloadURL compose path", () => {
      expect(suratApi.attachmentDownloadURL("s1", "a1")).toBe("/api/surat/s1/attachments/a1");
    });
    it("attachmentPreviewURL compose path", () => {
      expect(suratApi.attachmentPreviewURL("s1", "a1")).toBe("/api/surat/s1/attachments/a1/preview");
    });
  });

  describe("references + tembusan + thread + versions thin wrappers", () => {
    it("addReference → POST /references", async () => {
      mockJsonOnce({ id: "r1" });
      await suratApi.addReference("s1", { relationship: "balasan", to_surat_id: "s2" });
      const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      expect(call[0]).toBe("/api/surat/s1/references");
      expect(call[1].method).toBe("POST");
    });

    it("removeReference → DELETE /references/:id", async () => {
      mockJsonOnce({ status: "deleted" });
      await suratApi.removeReference("s1", "r1");
      const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
      expect(call[0]).toBe("/api/surat/s1/references/r1");
      expect(call[1].method).toBe("DELETE");
    });

    it("addTembusan → POST /tembusan", async () => {
      mockJsonOnce({ id: "t1" });
      await suratApi.addTembusan("s1", { instansi_id: "i1" });
      expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/surat/s1/tembusan");
    });

    it("removeTembusan → DELETE /tembusan/:id", async () => {
      mockJsonOnce({ status: "deleted" });
      await suratApi.removeTembusan("s1", "t1");
      expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/surat/s1/tembusan/t1");
    });

    it("getThread → GET /thread", async () => {
      mockJsonOnce({ nodes: [] });
      await suratApi.getThread("s1");
      expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/surat/s1/thread");
    });

    it("listAttachmentVersions → GET /versions", async () => {
      mockJsonOnce({ versions: [] });
      await suratApi.listAttachmentVersions("s1", "a1");
      expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe(
        "/api/surat/s1/attachments/a1/versions",
      );
    });
  });
});

describe("disposisiApi", () => {
  it("create → POST /api/disposisi", async () => {
    mockJsonOnce({ id: "d1" });
    await disposisiApi.create({ surat_id: "s1", assigned_to: "u1", instruksi: "x" });
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/disposisi");
  });

  it("update → PATCH /api/disposisi/:id", async () => {
    mockJsonOnce({ status: "updated" });
    await disposisiApi.update("d1", { status: "in_progress" });
    const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(call[0]).toBe("/api/disposisi/d1");
    expect(call[1].method).toBe("PATCH");
  });

  it("list dengan mine=true → query param", async () => {
    mockJsonOnce({ items: [] });
    await disposisiApi.list({ mine: true });
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toContain("mine=true");
  });

  it("list dengan filter status", async () => {
    mockJsonOnce({ items: [] });
    await disposisiApi.list({ status: "done" });
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toContain("status=done");
  });

  it("listAssignableUsers → GET /api/users/assignable", async () => {
    mockJsonOnce({ items: [] });
    await disposisiApi.listAssignableUsers();
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/users/assignable");
  });
});

describe("komentarApi", () => {
  it("list → GET /api/surat/:id/komentar", async () => {
    mockJsonOnce({ items: [] });
    await komentarApi.list("s1");
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/surat/s1/komentar");
  });

  it("append → POST dengan body { body }", async () => {
    mockJsonOnce({ id: "k1" });
    await komentarApi.append("s1", "halo");
    const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(call[0]).toBe("/api/surat/s1/komentar");
    expect(JSON.parse(call[1].body)).toEqual({ body: "halo" });
  });
});

describe("dashboardApi", () => {
  it("camat → GET /api/dashboard/camat", async () => {
    mockJsonOnce({ surat_masuk_hari_ini: 0, disposisi_belum_assign: 0, disposisi_overdue: 0, disposisi_assigned_to_me: 0 });
    await dashboardApi.camat();
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/dashboard/camat");
  });
});

describe("notificationApi", () => {
  it("list dengan unreadOnly=true → query param", async () => {
    mockJsonOnce({ items: [], unread: 0 });
    await notificationApi.list(true);
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/notifications?unread=true");
  });

  it("list default → tanpa unread param", async () => {
    mockJsonOnce({ items: [], unread: 0 });
    await notificationApi.list();
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/notifications");
  });

  it("markRead → PATCH /:id/read", async () => {
    mockJsonOnce({ status: "ok" });
    await notificationApi.markRead("n1");
    const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(call[0]).toBe("/api/notifications/n1/read");
    expect(call[1].method).toBe("PATCH");
  });

  it("markAllRead → POST /read-all", async () => {
    mockJsonOnce({ status: "ok" });
    await notificationApi.markAllRead();
    const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(call[0]).toBe("/api/notifications/read-all");
    expect(call[1].method).toBe("POST");
  });
});

describe("reconciliationApi", () => {
  it("list default → GET /api/reconciliation tanpa query", async () => {
    mockJsonOnce({ items: [] });
    await reconciliationApi.list();
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/reconciliation");
  });

  it("list dengan includeResolved=true → query include_resolved=true", async () => {
    mockJsonOnce({ items: [] });
    await reconciliationApi.list(true);
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe(
      "/api/reconciliation?include_resolved=true",
    );
  });

  it("get → GET /api/reconciliation/:group_id", async () => {
    mockJsonOnce({ group_id: "g1", surats: [] });
    await reconciliationApi.get("g1");
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/reconciliation/g1");
  });

  it("merge → POST /merge dengan canonical_surat_id", async () => {
    mockJsonOnce({ status: "merged" });
    await reconciliationApi.merge("g1", "s1");
    const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(call[0]).toBe("/api/reconciliation/g1/merge");
    expect(call[1].method).toBe("POST");
    expect(JSON.parse(call[1].body)).toEqual({ canonical_surat_id: "s1" });
  });

  it("keepBoth → POST /keep-both dengan body kosong", async () => {
    mockJsonOnce({ status: "kept_both" });
    await reconciliationApi.keepBoth("g1");
    const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(call[0]).toBe("/api/reconciliation/g1/keep-both");
    expect(call[1].method).toBe("POST");
  });
});

describe("statsApi", () => {
  it("byPeriod tanpa filter → GET /api/stats/by-period", async () => {
    mockJsonOnce({ items: [] });
    await statsApi.byPeriod();
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/stats/by-period");
  });

  it("byPeriod dengan from + to → query string", async () => {
    mockJsonOnce({ items: [] });
    await statsApi.byPeriod("2026-01-01", "2026-12-31");
    const url = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0];
    expect(url).toContain("from=2026-01-01");
    expect(url).toContain("to=2026-12-31");
  });

  it("byClassification → GET", async () => {
    mockJsonOnce({ items: [] });
    await statsApi.byClassification();
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/stats/by-classification");
  });

  it("bySender(top) → query top=N", async () => {
    mockJsonOnce({ items: [] });
    await statsApi.bySender(5);
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/stats/by-sender?top=5");
  });

  it("staffLoad → GET", async () => {
    mockJsonOnce({ items: [] });
    await statsApi.staffLoad();
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/stats/staff-load");
  });
});

describe("direktoriApi", () => {
  it("searchInstansi → GET /api/instansi dengan q + limit", async () => {
    mockJsonOnce({ items: [] });
    await direktoriApi.searchInstansi("Kemen", 5);
    const url = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0];
    expect(url).toContain("q=Kemen");
    expect(url).toContain("limit=5");
  });

  it("createInstansi → POST dengan default aliases [] kalau tidak diset", async () => {
    mockJsonOnce({ id: "i1" });
    await direktoriApi.createInstansi({ nama_kanonik: "X" });
    const body = JSON.parse(
      (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][1].body as string,
    );
    expect(body.aliases).toEqual([]);
    expect(body.nama_kanonik).toBe("X");
  });

  it("listKlasifikasi → GET /api/klasifikasi", async () => {
    mockJsonOnce({ items: [] });
    await direktoriApi.listKlasifikasi();
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/klasifikasi");
  });

  it("listSifat → GET /api/sifat", async () => {
    mockJsonOnce({ items: [] });
    await direktoriApi.listSifat();
    expect((global.fetch as ReturnType<typeof vi.fn>).mock.calls[0][0]).toBe("/api/sifat");
  });
});
