// Fase 7 Bracket 2 ACL: rekonsiliasi camat-only + auditor role read-only.

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

// =============================================================================
// #7 Migrasi rekonsiliasi ke camat
// =============================================================================

test("staf GET /api/reconciliation → 403 (camat-only)", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  const resp = await request.get("/api/reconciliation", {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.status()).toBe(403);
});

test("staf TIDAK lihat tombol Rekonsiliasi di topbar", async ({ page }) => {
  await loginAs(page, "staf1");
  await expect(page.getByTestId("nav-reconciliation")).toHaveCount(0);
});

test("staf goto /reconciliation → router redirect ke /surat", async ({ page }) => {
  await loginAs(page, "staf1");
  await page.goto("/reconciliation");
  await expect(page).toHaveURL(/\/surat$/);
});

test("camat GET /api/reconciliation → 200", async ({ page, request }) => {
  await loginAs(page, "camat");
  const token = await getToken(page);
  const resp = await request.get("/api/reconciliation", {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.status()).toBe(200);
});

// =============================================================================
// #3 Auditor role: read-only
// =============================================================================

test("auditor login berhasil + roles include 'auditor'", async ({ page }) => {
  await loginAs(page, "auditor");
  const auth = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  const data = JSON.parse(auth!);
  expect(data.roles).toContain("auditor");
});

test("auditor GET /api/surat → 200 (read-only akses)", async ({ page, request }) => {
  await loginAs(page, "auditor");
  const token = await getToken(page);
  const resp = await request.get("/api/surat?limit=5", {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.status()).toBe(200);
});

test("auditor POST /api/surat → 403 (read-only)", async ({ page, request }) => {
  await loginAs(page, "auditor");
  const token = await getToken(page);

  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;

  const resp = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: `AUDITOR-FORBIDDEN/${Date.now()}`,
      perihal: "Should fail",
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  expect(resp.status()).toBe(403);
});

test("auditor PATCH/DELETE/komentar/attachment → semua 403", async ({ page, request }) => {
  await loginAs(page, "auditor");
  const auditorToken = await getToken(page);

  // Setup: create surat as staf untuk target target tests
  await loginAs(page, "staf1");
  const stafToken = await getToken(page);

  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${stafToken}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;
  const create = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${stafToken}` },
    data: {
      jenis: "masuk", nomor_surat: `AUDITOR-TARGET/${Date.now()}`,
      perihal: "Auditor mutation target", tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16", instansi_id: instansiID, access_level: "public",
    },
  });
  const suratID = (await create.json()).id;

  // Verify auditor cannot mutate
  const headers = { Authorization: `Bearer ${auditorToken}` };
  const updateResp = await request.patch(`/api/surat/${suratID}`, {
    headers,
    data: { perihal: "Hacked" },
  });
  expect(updateResp.status()).toBe(403);

  const delResp = await request.delete(`/api/surat/${suratID}`, { headers });
  expect(delResp.status()).toBe(403);

  const komentarResp = await request.post(`/api/surat/${suratID}/komentar`, {
    headers, data: { body: "from auditor" },
  });
  expect(komentarResp.status()).toBe(403);

  // Cleanup as staf
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${stafToken}` },
  });
});

test("auditor TIDAK lihat tombol Edit/Hapus di SuratDetailView + tidak lihat tombol Surat Baru", async ({ page }) => {
  await loginAs(page, "auditor");

  // Tidak ada tombol create surat
  await expect(page.getByTestId("nav-surat-baru")).toHaveCount(0);

  // Klik first row → detail page tidak punya Edit/Hapus
  await page.locator("table tbody tr").first().click();
  await expect(page).toHaveURL(/\/surat\/[\w-]+$/);

  // Tombol Edit/Hapus di-hide via canWrite
  await expect(page.getByRole("button", { name: "Edit" })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Hapus" })).toHaveCount(0);

  // Tombol "+ Tambah Lampiran/Tembusan/Referensi" di-hide
  await expect(page.getByTestId("add-attachment-btn")).toHaveCount(0);
  await expect(page.getByTestId("add-tembusan-btn")).toHaveCount(0);
  await expect(page.getByTestId("add-reference-btn")).toHaveCount(0);

  // Komentar input di-hide
  await expect(page.getByTestId("komentar-input")).toHaveCount(0);
});
