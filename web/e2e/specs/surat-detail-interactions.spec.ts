import { test, expect, Page } from "@playwright/test";

async function loginAs(page: Page, username: string) {
  await page.goto("/login");
  await page.evaluate(() => localStorage.clear());
  await page.goto("/login");
  await page.getByPlaceholder("staf1 / camat / admin").fill(username);
  await page.getByPlaceholder("demo123").fill("demo123");
  await Promise.all([
    page.waitForURL(/\/surat$/),
    page.getByRole("button", { name: "Masuk" }).click(),
  ]);
}

// Setup helper: bikin satu surat lewat API supaya UI test fokus ke interaksi detail
async function createTestSurat(request: import("@playwright/test").APIRequestContext, token: string, perihal: string) {
  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;
  const createResp = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: `TEST-INT/${Date.now()}/2026`,
      perihal,
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  const created = await createResp.json();
  return { suratID: created.id as string, instansiID: instansiID as string };
}

async function getToken(page: Page): Promise<string> {
  const auth = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  return JSON.parse(auth!).accessToken;
}

// =============================================================================
// TEMBUSAN
// =============================================================================

test("FULL UI: tambah tembusan internal (instansi search) → muncul di list, hapus → hilang", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);
  const { suratID } = await createTestSurat(request, token, "Tembusan UI test surat");

  await page.goto(`/surat/${suratID}`);
  await expect(page.getByTestId("tembusan-card")).toBeVisible();
  await expect(page.getByText("Belum ada tembusan")).toBeVisible();

  // Buka dialog
  await page.getByTestId("add-tembusan-btn").click();
  await expect(page.locator('.n-modal').getByText("Tambah Tembusan", { exact: true })).toBeVisible();

  // Search instansi — data-testid ada di wrapper NInput, target input child
  await page.locator('[data-testid="tembusan-instansi-search"] input').fill("Pemkab");
  await page.waitForTimeout(400);

  // Pilih hasil pertama
  await page.locator('[data-testid^="tembusan-option-"]').first().click();

  // Submit
  await page.getByTestId("submit-add-tembusan").click();
  await expect(page.getByText("Tembusan ditambahkan")).toBeVisible({ timeout: 3000 });

  // Verify muncul di list
  await expect(page.getByTestId("tembusan-card").locator(".n-list-item").first()).toBeVisible();

  // Hapus
  await page.getByTestId("delete-tembusan-btn").click();
  await page.locator(".n-popconfirm__action button").last().click();
  await expect(page.getByText("Tembusan dihapus")).toBeVisible({ timeout: 3000 });

  await expect(page.getByText("Belum ada tembusan")).toBeVisible();

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

test("FULL UI: tambah tembusan external (text bebas) → muncul dengan tag External", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);
  const { suratID } = await createTestSurat(request, token, "Tembusan external test surat");

  await page.goto(`/surat/${suratID}`);

  await page.getByTestId("add-tembusan-btn").click();

  // Switch ke external mode
  const modeSelect = page.locator('.n-modal').locator('.n-base-selection').first();
  await modeSelect.click();
  await page.locator(".n-base-select-option").filter({ hasText: "External" }).first().click();

  await page.locator('.n-modal textarea').fill("Kepala Bidang Lingkungan Hidup");

  await page.getByTestId("submit-add-tembusan").click();
  await expect(page.getByText("Tembusan ditambahkan")).toBeVisible({ timeout: 3000 });

  // Verify display
  const card = page.getByTestId("tembusan-card");
  await expect(card.getByText("External")).toBeVisible();
  await expect(card.getByText("Kepala Bidang Lingkungan Hidup")).toBeVisible();

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

// =============================================================================
// REFERENCE add (lock in dialog yang baru ditambahkan)
// =============================================================================

test("FULL UI: tambah reference internal → muncul di Predecessor list, hapus → hilang", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);
  const { suratID } = await createTestSurat(request, token, "Reference UI test surat");

  await page.goto(`/surat/${suratID}`);

  await page.getByTestId("add-reference-btn").click();
  await expect(page.locator('.n-modal').getByText("Tambah Referensi", { exact: true })).toBeVisible();

  // Search untuk surat target
  await page.locator('.n-modal input[placeholder*="Cari"]').fill("pandemi");
  await page.waitForTimeout(400);

  // Pilih hasil pertama
  await page.locator('[data-testid^="ref-option-"]').first().click();

  await page.getByTestId("submit-add-reference").click();
  await expect(page.getByText("Referensi ditambahkan")).toBeVisible({ timeout: 3000 });

  // Verify muncul di Predecessor section
  await expect(page.getByText("Predecessor (surat ini merujuk):")).toBeVisible();

  // Hapus
  await page.getByTestId("delete-reference-btn").first().click();
  await page.locator(".n-popconfirm__action button").last().click();
  await expect(page.getByText("Referensi dihapus")).toBeVisible({ timeout: 3000 });

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

// =============================================================================
// ATTACHMENT add via detail page
// =============================================================================

test("FULL UI: detail page tambah lampiran via dialog → muncul di Lampiran list", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);
  const { suratID } = await createTestSurat(request, token, "Detail att add UI test");

  await page.goto(`/surat/${suratID}`);

  // Awalnya kosong
  await expect(page.getByText("Belum ada lampiran")).toBeVisible();

  await page.getByTestId("add-attachment-btn").click();
  await expect(page.locator('.n-modal').getByText("Tambah Lampiran", { exact: true })).toBeVisible();

  // Upload PDF — pakai file yang sudah ada di fixtures
  const pdfBuffer = Buffer.from(
    "%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Count 0>>endobj\nxref\n0 3\n0000000000 65535 f\n0000000009 00000 n\n0000000051 00000 n\ntrailer<</Size 3/Root 1 0 R>>\nstartxref\n89\n%%EOF",
  );
  const fileInput = page.locator('[data-testid="add-attachment-upload"] input[type="file"]');
  await fileInput.setInputFiles({
    name: "lampiran-detail.pdf",
    mimeType: "application/pdf",
    buffer: pdfBuffer,
  });

  await page.getByTestId("submit-add-attachment").click();
  await expect(page.getByText("1 file ter-upload")).toBeVisible({ timeout: 5000 });

  // Verify muncul di Lampiran card (bukan di upload widget yang masih open)
  const lampiranCard = page.locator('.n-card').filter({ hasText: /^Lampiran/ });
  await expect(lampiranCard.getByText("lampiran-detail.pdf")).toBeVisible({ timeout: 3000 });

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});
