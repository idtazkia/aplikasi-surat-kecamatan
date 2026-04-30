// E2E test untuk upload + preview lampiran via UI form.
// Semua assertion dilakukan via UI; setup state (pre-create surat) via API
// helper hanya kalau diperlukan untuk isolation, bukan sebagai SUT.
import { test, expect, Page, Locator } from "@playwright/test";

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

// Helper: pilih tanggal di NDatePicker via calendar panel click.
// Class names dari Naive UI source (lib/date-picker/src/panel/date.js + panelHeader.js):
//   .n-date-panel-month__month-year   — header text
//   .n-date-panel-month__prev / __next — navigation
//   .n-date-panel-date                 — day cell
//   .n-date-panel-date--excluded       — adjacent month (skip)
async function pickDate(page: Page, pickerInput: Locator, target: Date) {
  await pickerInput.click();
  await page.waitForSelector(".n-date-panel", { timeout: 3000 });

  const monthHeader = page.locator(".n-date-panel-month__month-year").first();

  // Header format Naive UI default: "MM<sep>YYYY" mis. "05 / 2026".
  // Parse manual via regex (more reliable dari new Date() yang
  // implementation-dependent).
  for (let i = 0; i < 60; i++) {
    const headerText = (await monthHeader.textContent())?.trim() ?? "";
    const m = headerText.match(/(\d{1,2})[^\d]+(\d{4})/);
    if (m) {
      const currentMonth = parseInt(m[1], 10) - 1;
      const currentYear = parseInt(m[2], 10);
      if (currentMonth === target.getMonth() && currentYear === target.getFullYear()) {
        break;
      }
      const goNext =
        new Date(target.getFullYear(), target.getMonth()).getTime() >
        new Date(currentYear, currentMonth).getTime();
      await page.locator(
        goNext ? ".n-date-panel-month__next" : ".n-date-panel-month__prev",
      ).first().click();
    } else {
      // Header format unknown — break to avoid infinite loop
      break;
    }
    await page.waitForTimeout(80);
  }

  const dayStr = String(target.getDate());
  await page
    .locator(".n-date-panel-date:not(.n-date-panel-date--excluded)")
    .filter({ hasText: new RegExp(`^${dayStr}$`) })
    .first()
    .click();

  await page.waitForTimeout(200);
}

// Helper fill semua field required form surat masuk + pilih instansi.
// Caller bertanggungjawab nambah file inputs + submit setelah ini.
async function fillSuratMasukForm(page: Page, opts: {
  nomor: string;
  perihal: string;
  tanggalSurat: Date;
  tanggalTerima: Date;
  instansiSearch: string;
}) {
  await page.getByPlaceholder("045/123/IV/2026").fill(opts.nomor);
  await page.getByPlaceholder("Subject surat").fill(opts.perihal);

  const dateInputs = page.locator(".n-date-picker input");
  await pickDate(page, dateInputs.nth(0), opts.tanggalSurat);
  await pickDate(page, dateInputs.nth(1), opts.tanggalTerima);

  const instansiSelect = page.locator('[data-testid="instansi-field"] .n-base-selection');
  await instansiSelect.click();
  await page.keyboard.type(opts.instansiSearch, { delay: 30 });
  await page.waitForTimeout(900);
  await page.locator(".n-base-select-option").first().click();
}

// =============================================================================
// PRE-SUBMIT FILE ATTACH
// =============================================================================

test("form /surat/baru: setInputFiles real PDF → file ter-attach di NUpload list (pre-submit)", async ({ page }) => {
  await loginAs(page, "staf1");
  await page.goto("/surat/baru");

  await expect(page.getByPlaceholder("045/123/IV/2026")).toBeVisible();
  await page.waitForLoadState("networkidle");

  await page.locator('[data-testid="primary-upload"] input[type="file"]').setInputFiles({
    name: "surat-utama.pdf",
    mimeType: "application/pdf",
    buffer: MINIMAL_PDF,
  });

  await page.locator('[data-testid="lampiran-upload"] input[type="file"]').setInputFiles([
    { name: "lampiran-1.pdf", mimeType: "application/pdf", buffer: MINIMAL_PDF },
    { name: "lampiran-2.pdf", mimeType: "application/pdf", buffer: MINIMAL_PDF },
  ]);

  await expect(page.getByText("surat-utama.pdf").first()).toBeVisible();
  await expect(page.getByText("lampiran-1.pdf").first()).toBeVisible();
  await expect(page.getByText("lampiran-2.pdf").first()).toBeVisible();
});

// =============================================================================
// FULL UI HAPPY PATH — fill form + upload + submit + verify detail
// =============================================================================

test("FULL UI: fill form + upload 1 primary + 2 lampiran → detail menampilkan 3 lampiran dengan role tag yang benar", async ({ page }) => {
  await loginAs(page, "staf1");
  await page.goto("/surat/baru");

  await expect(page.getByPlaceholder("045/123/IV/2026")).toBeVisible();
  await page.waitForLoadState("networkidle");

  await fillSuratMasukForm(page, {
    nomor: "UPLOAD-FULL/01/2026",
    perihal: "Full UI form test",
    tanggalSurat: new Date(2026, 3, 15),
    tanggalTerima: new Date(2026, 3, 16),
    instansiSearch: "Kemendagri",
  });

  await page.locator('[data-testid="primary-upload"] input[type="file"]').setInputFiles({
    name: "surat-utama.pdf",
    mimeType: "application/pdf",
    buffer: MINIMAL_PDF,
  });
  await page.locator('[data-testid="lampiran-upload"] input[type="file"]').setInputFiles([
    { name: "lampiran-1.pdf", mimeType: "application/pdf", buffer: MINIMAL_PDF },
    { name: "lampiran-2.pdf", mimeType: "application/pdf", buffer: MINIMAL_PDF },
  ]);

  await Promise.all([
    page.waitForURL(/\/surat\/[0-9a-f]{8}-/i, { timeout: 20000 }),
    page.getByRole("button", { name: "Buat Surat" }).click(),
  ]);

  await expect(page.locator("body")).toContainText("Full UI form test");
  await expect(page.locator("body")).toContainText("UPLOAD-FULL/01/2026");

  await expect(page.getByText("surat-utama.pdf").first()).toBeVisible();
  await expect(page.getByText("lampiran-1.pdf").first()).toBeVisible();
  await expect(page.getByText("lampiran-2.pdf").first()).toBeVisible();

  const lampiranCard = page.locator(".n-card").filter({ hasText: /^Lampiran/ }).first();
  expect(await lampiranCard.locator(".n-tag", { hasText: /^Utama$/ }).count()).toBe(1);
  expect(await lampiranCard.locator(".n-tag", { hasText: /^Lampiran$/ }).count()).toBe(2);
});

// =============================================================================
// PREVIEW CYCLE — upload + click preview + close, all via UI
// =============================================================================

test("FULL UI: upload PDF → klik Preview → iframe blob: muncul → klik Tutup → iframe hilang", async ({ page }) => {
  await loginAs(page, "staf1");
  await page.goto("/surat/baru");

  await expect(page.getByPlaceholder("045/123/IV/2026")).toBeVisible();
  await page.waitForLoadState("networkidle");

  await fillSuratMasukForm(page, {
    nomor: "PREVIEW-UI/01/2026",
    perihal: "Test preview via UI",
    tanggalSurat: new Date(2026, 3, 15),
    tanggalTerima: new Date(2026, 3, 16),
    instansiSearch: "Kemendagri",
  });

  await page.locator('[data-testid="primary-upload"] input[type="file"]').setInputFiles({
    name: "preview-target.pdf",
    mimeType: "application/pdf",
    buffer: MINIMAL_PDF,
  });

  await Promise.all([
    page.waitForURL(/\/surat\/[0-9a-f]{8}-/i, { timeout: 20000 }),
    page.getByRole("button", { name: "Buat Surat" }).click(),
  ]);

  // Detail page — initially no preview iframe
  await expect(page.locator('[data-testid="preview-iframe"]')).toHaveCount(0);

  // Klik Preview button
  await page.locator('[data-testid="attachment-preview-btn"]').first().click();

  // Iframe muncul dengan blob URL src
  const iframe = page.locator('[data-testid="preview-iframe"]');
  await expect(iframe).toBeVisible({ timeout: 5000 });
  const src = await iframe.getAttribute("src");
  expect(src).toBeTruthy();
  expect(src!.startsWith("blob:")).toBe(true);

  await expect(page.getByText("Preview: preview-target.pdf")).toBeVisible();

  // Klik Tutup
  await page.getByRole("button", { name: "Tutup" }).click();
  await expect(page.locator('[data-testid="preview-iframe"]')).toHaveCount(0);
});

// =============================================================================
// VALIDATION via UI — submit form dengan invalid file types/sizes,
// expect error toast yang terjadi karena backend reject upload
// =============================================================================

test("FULL UI: upload file non-allowed MIME (binary) → toast error 'Lampiran gagal ter-upload'", async ({ page }) => {
  await loginAs(page, "staf1");
  await page.goto("/surat/baru");

  await expect(page.getByPlaceholder("045/123/IV/2026")).toBeVisible();
  await page.waitForLoadState("networkidle");

  await fillSuratMasukForm(page, {
    nomor: "MIME-REJECT-UI/01/2026",
    perihal: "Test MIME reject via UI",
    tanggalSurat: new Date(2026, 3, 15),
    tanggalTerima: new Date(2026, 3, 16),
    instansiSearch: "Kemendagri",
  });

  // Binary garbage — http.DetectContentType return application/octet-stream → 415
  const garbage = Buffer.from([
    0xde, 0xad, 0xbe, 0xef, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05,
    0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
    ...new Array(500).fill(0xff),
  ]);

  await page.locator('[data-testid="primary-upload"] input[type="file"]').setInputFiles({
    name: "garbage.bin",
    mimeType: "application/octet-stream",
    buffer: garbage,
  });

  // Submit — surat akan ke-create tapi attachment upload gagal
  await Promise.all([
    page.waitForURL(/\/surat\/[0-9a-f]{8}-/i, { timeout: 20000 }),
    page.getByRole("button", { name: "Buat Surat" }).click(),
  ]);

  // Toast error untuk upload failure muncul
  await expect(page.getByText("Lampiran gagal ter-upload — silakan tambah dari halaman detail")).toBeVisible({ timeout: 5000 });

  // Verify lampiran section di detail page kosong (tidak ada file yang ter-upload)
  await expect(page.getByText("Belum ada lampiran")).toBeVisible({ timeout: 5000 });
});

test("FULL UI: upload file > 25MB → toast error karena backend reject 413", async ({ page }) => {
  await loginAs(page, "staf1");
  await page.goto("/surat/baru");

  await expect(page.getByPlaceholder("045/123/IV/2026")).toBeVisible();
  await page.waitForLoadState("networkidle");

  await fillSuratMasukForm(page, {
    nomor: "TOOBIG-UI/01/2026",
    perihal: "Test size limit via UI",
    tanggalSurat: new Date(2026, 3, 15),
    tanggalTerima: new Date(2026, 3, 16),
    instansiSearch: "Kemendagri",
  });

  // Buffer 26MB dengan PDF magic prefix supaya MIME sniff lewat dulu
  const oversized = Buffer.concat([
    Buffer.from("%PDF-1.4\n"),
    Buffer.alloc(26 * 1024 * 1024, 0x20),
  ]);

  await page.locator('[data-testid="primary-upload"] input[type="file"]').setInputFiles({
    name: "huge.pdf",
    mimeType: "application/pdf",
    buffer: oversized,
  });

  await Promise.all([
    page.waitForURL(/\/surat\/[0-9a-f]{8}-/i, { timeout: 30000 }),
    page.getByRole("button", { name: "Buat Surat" }).click(),
  ]);

  await expect(page.getByText("Lampiran gagal ter-upload — silakan tambah dari halaman detail")).toBeVisible({ timeout: 10000 });
  await expect(page.getByText("Belum ada lampiran")).toBeVisible({ timeout: 5000 });
});
