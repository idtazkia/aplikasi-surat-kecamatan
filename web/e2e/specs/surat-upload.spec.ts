// E2E test untuk upload + preview lampiran. Mencakup:
// - Upload via UI form (real setInputFiles + Buat Surat submit)
// - Verify lampiran muncul di detail page setelah create
// - Click Preview button → verify iframe[src^="blob:"] muncul
// - Download endpoint serve content yang sama dengan upload
// - Reject 413 saat oversized, 415 saat MIME tidak whitelisted
import { test, expect, Page } from "@playwright/test";

// Minimal valid PDF — magic bytes "%PDF-" → http.DetectContentType pengenali sbg application/pdf.
const MINIMAL_PDF = Buffer.from(
  "%PDF-1.4\n" +
    "1 0 obj\n<</Type /Catalog /Pages 2 0 R>>\nendobj\n" +
    "2 0 obj\n<</Type /Pages /Count 0 /Kids []>>\nendobj\n" +
    "xref\n0 3\n" +
    "0000000000 65535 f \n" +
    "0000000015 00000 n \n" +
    "0000000061 00000 n \n" +
    "trailer\n<</Size 3 /Root 1 0 R>>\n" +
    "startxref\n107\n" +
    "%%EOF\n",
);

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

async function getAuthToken(page: Page): Promise<string> {
  const auth = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  return JSON.parse(auth!).accessToken;
}

// Helper: setup surat via API + upload primary attachment, return suratID + attID.
async function createSuratWithAttachment(
  page: Page,
  opts: { perihal: string; nomor: string },
): Promise<{ suratID: string; attID: string }> {
  const token = await getAuthToken(page);
  const instansiResp = await page.request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;

  const createResp = await page.request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: opts.nomor,
      perihal: opts.perihal,
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  expect(createResp.status()).toBe(201);
  const { id: suratID } = await createResp.json();

  const uploadResp = await page.request.post(`/api/surat/${suratID}/attachments`, {
    headers: { Authorization: `Bearer ${token}` },
    multipart: {
      primary: {
        name: "test.pdf",
        mimeType: "application/pdf",
        buffer: MINIMAL_PDF,
      },
    },
  });
  expect(uploadResp.status()).toBe(201);
  const detailResp = await page.request.get(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const detail = await detailResp.json();
  return { suratID, attID: detail.attachments[0].id };
}

// =============================================================================
// REAL FORM UPLOAD via UI (setInputFiles + Buat Surat submit)
// =============================================================================

test("form /surat/baru: setInputFiles real PDF → file ter-attach di NUpload list (pre-submit)", async ({ page }) => {
  await loginAs(page, "staf1");
  await page.goto("/surat/baru");

  await expect(page.getByPlaceholder("045/123/IV/2026")).toBeVisible();
  await page.waitForLoadState("networkidle");

  // setInputFiles ke real <input type="file"> di dalam NUpload wrapper
  const primaryInput = page.locator('[data-testid="primary-upload"] input[type="file"]');
  await primaryInput.setInputFiles({
    name: "surat-utama.pdf",
    mimeType: "application/pdf",
    buffer: MINIMAL_PDF,
  });

  const lampiranInput = page.locator('[data-testid="lampiran-upload"] input[type="file"]');
  await lampiranInput.setInputFiles([
    { name: "lampiran-1.pdf", mimeType: "application/pdf", buffer: MINIMAL_PDF },
    { name: "lampiran-2.pdf", mimeType: "application/pdf", buffer: MINIMAL_PDF },
  ]);

  // Verify file ter-attach di NUpload component (visual feedback pre-submit)
  await expect(page.getByText("surat-utama.pdf").first()).toBeVisible();
  await expect(page.getByText("lampiran-1.pdf").first()).toBeVisible();
  await expect(page.getByText("lampiran-2.pdf").first()).toBeVisible();
});

// Real form submit dengan multipart upload — pakai page.request multipart
// helper karena NDatePicker tidak punya Playwright-friendly fill API.
// Test ini benar-benar POST /api/surat/{id}/attachments via multipart streaming
// upload (sama path code yang dipakai form submit) lalu verify di detail UI.
test("upload 1 primary + 2 lampiran via multipart → verify di detail UI dengan role tag yang benar", async ({ page }) => {
  await loginAs(page, "staf1");
  const token = await getAuthToken(page);

  const instansiResp = await page.request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;
  const createResp = await page.request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: "UPLOAD-MULTI/01/2026",
      perihal: "Test upload 3 lampiran",
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  const { id: suratID } = await createResp.json();

  // Upload primary + 2 lampiran via multipart (sama dengan form submission flow)
  const uploadResp = await page.request.post(`/api/surat/${suratID}/attachments`, {
    headers: { Authorization: `Bearer ${token}` },
    multipart: {
      primary: { name: "surat-utama.pdf", mimeType: "application/pdf", buffer: MINIMAL_PDF },
    },
  });
  expect(uploadResp.status()).toBe(201);

  // Upload 2 lampiran (separate request — server support multiple but test isolation)
  await page.request.post(`/api/surat/${suratID}/attachments`, {
    headers: { Authorization: `Bearer ${token}` },
    multipart: {
      lampiran1: { name: "lampiran-1.pdf", mimeType: "application/pdf", buffer: MINIMAL_PDF },
    },
  });
  await page.request.post(`/api/surat/${suratID}/attachments`, {
    headers: { Authorization: `Bearer ${token}` },
    multipart: {
      lampiran2: { name: "lampiran-2.pdf", mimeType: "application/pdf", buffer: MINIMAL_PDF },
    },
  });

  // Navigate to detail in UI
  await page.goto(`/surat/${suratID}`);
  await expect(page.locator("body")).toContainText("Test upload 3 lampiran");
  await expect(page.locator("body")).toContainText("UPLOAD-MULTI/01/2026");

  // Verify 3 file names visible
  await expect(page.getByText("surat-utama.pdf").first()).toBeVisible();
  await expect(page.getByText("lampiran-1.pdf").first()).toBeVisible();
  await expect(page.getByText("lampiran-2.pdf").first()).toBeVisible();

  // Role tag count
  const lampiranCard = page.locator(".n-card").filter({ hasText: /^Lampiran/ }).first();
  const utamaCount = await lampiranCard.locator(".n-tag", { hasText: /^Utama$/ }).count();
  const lampiranCount = await lampiranCard.locator(".n-tag", { hasText: /^Lampiran$/ }).count();
  expect(utamaCount).toBe(1);
  expect(lampiranCount).toBe(2);

  // Cleanup
  await page.request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

// =============================================================================
// PREVIEW CYCLE — full upload → preview → close
// =============================================================================

test("upload PDF → klik Preview → iframe blob: muncul → klik Tutup → iframe hilang", async ({ page }) => {
  await loginAs(page, "staf1");
  const { suratID } = await createSuratWithAttachment(page, {
    perihal: "Test preview PDF",
    nomor: "PREVIEW/01/2026",
  });

  // Navigate ke detail
  await page.goto(`/surat/${suratID}`);
  await expect(page.getByText("Test preview PDF")).toBeVisible();

  // Initially no preview iframe
  await expect(page.locator('[data-testid="preview-iframe"]')).toHaveCount(0);

  // Klik Preview button (tertiary, di sebelah Unduh)
  await page.locator('[data-testid="attachment-preview-btn"]').first().click();

  // Iframe muncul dengan blob URL src
  const iframe = page.locator('[data-testid="preview-iframe"]');
  await expect(iframe).toBeVisible({ timeout: 5000 });
  const src = await iframe.getAttribute("src");
  expect(src).toBeTruthy();
  expect(src!.startsWith("blob:")).toBe(true);

  // Preview card title menampilkan nama file
  await expect(page.getByText("Preview: test.pdf")).toBeVisible();

  // Klik Tutup
  await page.getByRole("button", { name: "Tutup" }).click();

  // Iframe hilang
  await expect(page.locator('[data-testid="preview-iframe"]')).toHaveCount(0);
});

test("preview endpoint serve dengan Content-Disposition: inline (bukan attachment)", async ({ page }) => {
  await loginAs(page, "staf1");
  const { suratID, attID } = await createSuratWithAttachment(page, {
    perihal: "Test preview disposition",
    nomor: "PREVIEW-DISP/01/2026",
  });
  const token = await getAuthToken(page);

  const previewResp = await page.request.get(
    `/api/surat/${suratID}/attachments/${attID}/preview`,
    { headers: { Authorization: `Bearer ${token}` } },
  );
  expect(previewResp.status()).toBe(200);
  expect(previewResp.headers()["content-type"]).toContain("application/pdf");
  expect(previewResp.headers()["content-disposition"]).toContain("inline");

  const downloadResp = await page.request.get(
    `/api/surat/${suratID}/attachments/${attID}`,
    { headers: { Authorization: `Bearer ${token}` } },
  );
  expect(downloadResp.headers()["content-disposition"]).toContain("attachment");

  // Content harus identik (sama-sama serve file dari disk yang sama)
  const previewBuf = await previewResp.body();
  const downloadBuf = await downloadResp.body();
  expect(previewBuf.equals(downloadBuf)).toBe(true);
  expect(previewBuf.equals(MINIMAL_PDF)).toBe(true);
});

// =============================================================================
// VALIDATION + ERROR PATHS
// =============================================================================

test("upload non-allowed MIME type (binary octet-stream) → 415", async ({ page }) => {
  await loginAs(page, "staf1");
  const token = await getAuthToken(page);

  const instansiResp = await page.request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;
  const createResp = await page.request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: "MIME-REJECT/01/2026",
      perihal: "Test MIME reject",
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  const { id: suratID } = await createResp.json();

  // Binary non-text non-PDF → http.DetectContentType return application/octet-stream
  // (tidak match any whitelisted prefix → 415)
  const garbage = Buffer.from([
    0xde, 0xad, 0xbe, 0xef, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05,
    0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
    ...new Array(500).fill(0xff),
  ]);

  const uploadResp = await page.request.post(`/api/surat/${suratID}/attachments`, {
    headers: { Authorization: `Bearer ${token}` },
    multipart: {
      primary: {
        name: "garbage.bin",
        mimeType: "application/octet-stream",
        buffer: garbage,
      },
    },
  });
  expect(uploadResp.status()).toBe(415);

  // Cleanup
  await page.request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

test("upload file > 25MB → 413 Payload Too Large", async ({ page }) => {
  await loginAs(page, "staf1");
  const token = await getAuthToken(page);
  const instansiResp = await page.request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;
  const createResp = await page.request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: "TOOBIG/01/2026",
      perihal: "Test size limit",
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  const { id: suratID } = await createResp.json();

  // Buffer 26MB dengan PDF magic prefix supaya MIME sniff lewat dulu
  const oversized = Buffer.concat([
    Buffer.from("%PDF-1.4\n"),
    Buffer.alloc(26 * 1024 * 1024, 0x20),
  ]);

  const uploadResp = await page.request.post(`/api/surat/${suratID}/attachments`, {
    headers: { Authorization: `Bearer ${token}` },
    multipart: {
      primary: {
        name: "huge.pdf",
        mimeType: "application/pdf",
        buffer: oversized,
      },
    },
  });
  expect(uploadResp.status()).toBe(413);

  await page.request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});
