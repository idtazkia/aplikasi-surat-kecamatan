// Fase 5 E2E: dedup detection + reconciliation merge UI.
//
// Skenario: 2 staf input surat masuk dengan dedup tuple sama → server detect,
// masuk reconciliation queue. Staf buka /reconciliation → pilih kanonik →
// merge → loser di-soft-delete.

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

async function createSuratMasuk(
  request: import("@playwright/test").APIRequestContext,
  token: string,
  data: { nomor_surat: string; perihal: string; instansi_id: string; tanggal_terima: string },
): Promise<{ id: string; reconciliation_group_id?: string }> {
  const resp = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: data.nomor_surat,
      perihal: data.perihal,
      tanggal_surat: "2026-04-15",
      tanggal_terima: data.tanggal_terima,
      instansi_id: data.instansi_id,
      access_level: "public",
    },
  });
  expect(resp.status()).toBe(201);
  return resp.json();
}

// =============================================================================
// API: dedup detection on create
// =============================================================================

test("create surat masuk dengan dedup tuple sama → response berisi reconciliation_group_id", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;

  const nomor = `DEDUP-API/${Date.now()}/2026`;
  const tanggal = "2026-04-16";

  // Create #1 — tidak ada duplikat
  const r1 = await createSuratMasuk(request, token, {
    nomor_surat: nomor, perihal: "First entry", instansi_id: instansiID, tanggal_terima: tanggal,
  });
  expect(r1.reconciliation_group_id).toBeUndefined();

  // Create #2 — duplikat tuple
  const r2 = await createSuratMasuk(request, token, {
    nomor_surat: nomor, perihal: "Duplicate entry", instansi_id: instansiID, tanggal_terima: tanggal,
  });
  expect(r2.reconciliation_group_id).toBeDefined();
  expect(r2.reconciliation_group_id).toMatch(/^[\w-]{36}$/);

  // GET /api/reconciliation → camat-only access (staf 403). Login ulang sbg camat
  // untuk verify list endpoint.
  await loginAs(page, "camat");
  const camatToken = await getToken(page);
  const listResp = await request.get("/api/reconciliation", {
    headers: { Authorization: `Bearer ${camatToken}` },
  });
  expect(listResp.status()).toBe(200);
  const list = await listResp.json();
  const found = list.items.find(
    (g: { group_id: string }) => g.group_id === r2.reconciliation_group_id,
  );
  expect(found).toBeDefined();
  expect(found.surat_count).toBe(2);
  expect(found.status).toBe("pending");

  // Cleanup
  await request.delete(`/api/surat/${r1.id}`, { headers: { Authorization: `Bearer ${token}` } });
  await request.delete(`/api/surat/${r2.id}`, { headers: { Authorization: `Bearer ${token}` } });
});

test("create surat dengan tuple BERBEDA → tidak masuk recon queue", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;

  const r1 = await createSuratMasuk(request, token, {
    nomor_surat: `UNIQUE-A/${Date.now()}`, perihal: "A", instansi_id: instansiID,
    tanggal_terima: "2026-04-16",
  });
  const r2 = await createSuratMasuk(request, token, {
    nomor_surat: `UNIQUE-B/${Date.now()}`, perihal: "B", instansi_id: instansiID,
    tanggal_terima: "2026-04-16",
  });
  expect(r1.reconciliation_group_id).toBeUndefined();
  expect(r2.reconciliation_group_id).toBeUndefined();

  await request.delete(`/api/surat/${r1.id}`, { headers: { Authorization: `Bearer ${token}` } });
  await request.delete(`/api/surat/${r2.id}`, { headers: { Authorization: `Bearer ${token}` } });
});

// =============================================================================
// FULL UI: navigate ke /reconciliation, pilih kanonik, merge
// =============================================================================

test("FULL UI: dedup pair → /reconciliation list → pilih kanonik → merge → loser disappear from list surat", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;

  const nomor = `DEDUP-UI/${Date.now()}/2026`;
  const tanggal = "2026-04-17";

  const r1 = await createSuratMasuk(request, token, {
    nomor_surat: nomor, perihal: "Versi A — input staf1 pagi", instansi_id: instansiID,
    tanggal_terima: tanggal,
  });
  const r2 = await createSuratMasuk(request, token, {
    nomor_surat: nomor, perihal: "Versi B — input staf2 sore", instansi_id: instansiID,
    tanggal_terima: tanggal,
  });
  expect(r2.reconciliation_group_id).toBeDefined();

  // Switch ke camat untuk operasi rekonsiliasi (staf tidak punya akses).
  await loginAs(page, "camat");
  await page.goto("/surat");
  await page.getByTestId("nav-reconciliation").click();
  await expect(page).toHaveURL(/\/reconciliation$/);

  // Klik group yang baru dibuat
  const groupCard = page.getByTestId(`recon-group-${r2.reconciliation_group_id}`);
  await expect(groupCard).toBeVisible({ timeout: 5000 });
  await groupCard.click();

  // Detail view — pilih r1 sebagai kanonik
  await expect(page.getByTestId("recon-detail-view")).toBeVisible({ timeout: 5000 });
  await page.getByTestId(`recon-surat-card-${r1.id}`).click();

  // Submit merge
  await page.getByTestId("recon-merge-btn").click();
  await expect(page.getByText("Surat di-merge")).toBeVisible({ timeout: 5000 });

  // Group hilang dari list pending
  await expect(page.getByTestId(`recon-group-${r2.reconciliation_group_id}`)).toHaveCount(0, {
    timeout: 5000,
  });

  // Verify via API: r1 tetap aktif, r2 soft-deleted
  const r1Resp = await request.get(`/api/surat/${r1.id}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(r1Resp.status()).toBe(200);

  const r2Resp = await request.get(`/api/surat/${r2.id}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(r2Resp.status()).toBe(404);

  // Cleanup r1
  await request.delete(`/api/surat/${r1.id}`, { headers: { Authorization: `Bearer ${token}` } });
});

test("FULL UI: keep-both flow → kedua surat tetap aktif, status group → kept_both", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;

  const nomor = `KEEPBOTH-UI/${Date.now()}`;
  const tanggal = "2026-04-18";
  const r1 = await createSuratMasuk(request, token, {
    nomor_surat: nomor, perihal: "Versi A", instansi_id: instansiID, tanggal_terima: tanggal,
  });
  const r2 = await createSuratMasuk(request, token, {
    nomor_surat: nomor, perihal: "Versi B (kebetulan tuple sama)", instansi_id: instansiID,
    tanggal_terima: tanggal,
  });

  // Switch ke camat untuk operasi rekonsiliasi
  await loginAs(page, "camat");
  await page.goto(`/reconciliation`);
  await page.getByTestId(`recon-group-${r2.reconciliation_group_id}`).click();

  // Klik keep-both
  await page.getByTestId("recon-keep-both-btn").click();
  await page.locator(".n-popconfirm__action button").last().click();
  await expect(page.getByText(/Kedua surat di-tandai bukan duplikat/)).toBeVisible({
    timeout: 5000,
  });

  // Verify keduanya masih ada via API
  const r1Resp = await request.get(`/api/surat/${r1.id}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const r2Resp = await request.get(`/api/surat/${r2.id}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(r1Resp.status()).toBe(200);
  expect(r2Resp.status()).toBe(200);

  // Cleanup keduanya
  await request.delete(`/api/surat/${r1.id}`, { headers: { Authorization: `Bearer ${token}` } });
  await request.delete(`/api/surat/${r2.id}`, { headers: { Authorization: `Bearer ${token}` } });
});
