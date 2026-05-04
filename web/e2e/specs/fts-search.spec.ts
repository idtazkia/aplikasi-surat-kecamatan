// Fase 7 Bracket 3 #1: Full-text search PDF.
// Verify search box di /surat menemukan keyword dari perihal + extracted PDF.

import { test, expect, Page } from "@playwright/test";

async function loginAs(page: Page, username: string) {
  await page.goto("/login");
  await page.evaluate(() => localStorage.clear());
  await page.goto("/login");
  await page.getByPlaceholder("staf1 / camat / admin / auditor").fill(username);
  await page.getByPlaceholder("demo123").fill("demo123");
  await Promise.all([
    page.waitForURL(/\/surat$/),
    page.getByRole("button", { name: "Masuk" }).click(),
  ]);
}

async function getToken(page: Page): Promise<string> {
  const auth = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  return JSON.parse(auth!).accessToken;
}

// Catatan: PDF text extraction E2E pakai handcrafted PDF tidak reliable
// karena ledongthuc/pdf strict pada xref byte-offsets. Path
// "extract from uploaded PDF" di-cover Go unit test (extractPDFText)
// atau manual smoke; E2E ini fokus ke FTS metadata path + integrasi UI.

// =============================================================================
// API kontrak: search via /api/surat?search=
// =============================================================================

test("FTS: keyword di perihal → ditemukan via /api/surat?search=", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;

  // Create surat dengan keyword unik di perihal
  const uniqueKeyword = `xenon${Date.now()}`;
  const create = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: `FTS-PERIHAL/${Date.now()}`,
      perihal: `Surat tentang ${uniqueKeyword} dan kontrak`,
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  const suratID = (await create.json()).id;

  // Search dengan keyword unik
  const searchResp = await request.get(`/api/surat?search=${uniqueKeyword}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(searchResp.status()).toBe(200);
  const items = (await searchResp.json()).items;
  expect(items.length).toBeGreaterThan(0);
  expect(items.some((s: { id: string }) => s.id === suratID)).toBe(true);

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

test("FTS: keyword tidak ada → result kosong", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  const resp = await request.get(
    "/api/surat?search=zzznonexistentkeywordxxx",
    {
      headers: { Authorization: `Bearer ${token}` },
    },
  );
  const items = (await resp.json()).items;
  expect(items).toEqual([]);
});

// =============================================================================
// FULL UI: search box di list
// =============================================================================

test("FULL UI: ketik keyword di search box → list filtered", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;

  const uniqueKeyword = `quantum${Date.now()}`;
  const create = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: `FTS-UI/${Date.now()}`,
      perihal: `Penelitian ${uniqueKeyword} oleh universitas`,
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  const suratID = (await create.json()).id;

  await page.goto("/surat");
  await page.getByPlaceholder("Kata kunci").fill(uniqueKeyword);
  await page.getByRole("button", { name: "Terapkan" }).click();
  await page.waitForTimeout(500);

  // Tabel berisi surat dengan keyword
  const bodyText = await page.locator(".n-layout-content").textContent();
  expect(bodyText).toContain(uniqueKeyword);

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});
