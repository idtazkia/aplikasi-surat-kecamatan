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

async function getToken(page: Page): Promise<string> {
  const auth = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  return JSON.parse(auth!).accessToken;
}

// Inline-generate minimal valid PDF (1-page blank).
// Format ini parseable oleh pdfcpu — verified manual via go test.
function makeSamplePDF(): Buffer {
  return Buffer.from(
    "%PDF-1.4\n" +
    "1 0 obj\n<</Type /Catalog /Pages 2 0 R>>\nendobj\n" +
    "2 0 obj\n<</Type /Pages /Kids [3 0 R] /Count 1>>\nendobj\n" +
    "3 0 obj\n<</Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources <<>>>>\nendobj\n" +
    "4 0 obj\n<</Length 0>>\nstream\nendstream\nendobj\n" +
    "xref\n0 5\n0000000000 65535 f \n0000000009 00000 n \n0000000052 00000 n \n0000000098 00000 n \n0000000189 00000 n \n" +
    "trailer\n<</Size 5 /Root 1 0 R>>\nstartxref\n227\n%%EOF\n",
  );
}

test("FULL UI flow + API verify: PDF download di surat restricted → bytes berbeda dari source (watermarked)", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  // Setup: surat restricted dengan PDF
  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;
  const createResp = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: `WATERMARK-TEST/${Date.now()}/2026`,
      perihal: "Surat restricted untuk watermark test",
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "restricted",
    },
  });
  const suratID = (await createResp.json()).id as string;

  // Upload PDF via UI
  await page.goto(`/surat/${suratID}`);
  await page.getByTestId("add-attachment-btn").click();
  const roleSelect = page.locator('.n-modal').locator('.n-base-selection').first();
  await roleSelect.click();
  await page.locator(".n-base-select-option").filter({ hasText: "PDF Utama" }).click();

  const samplePDF = makeSamplePDF();
  await page.locator('[data-testid="add-attachment-upload"] input[type="file"]').setInputFiles({
    name: "restricted-doc.pdf",
    mimeType: "application/pdf",
    buffer: samplePDF,
  });
  await page.getByTestId("submit-add-attachment").click();
  await expect(page.getByText("1 file ter-upload")).toBeVisible({ timeout: 5000 });

  // Ambil att_id dari list (lihat di lampiran-card)
  // Lebih reliable via API listing
  const detailResp = await request.get(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const detail = await detailResp.json();
  expect(detail.attachments).toHaveLength(1);
  const attID = detail.attachments[0].id as string;

  // Download via API → bytes harus berbeda dari source (watermark applied)
  const downloadResp = await request.get(`/api/surat/${suratID}/attachments/${attID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(downloadResp.status()).toBe(200);
  const downloaded = await downloadResp.body();

  // Watermark applied → resulting bytes != source bytes
  expect(downloaded.length).not.toEqual(samplePDF.length);
  // Magic header tetap %PDF
  expect(downloaded.slice(0, 4).toString()).toBe("%PDF");

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

test("FULL UI flow + API verify: PDF download di surat public → bytes IDENTIK source (no watermark)", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;
  const createResp = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: `WATERMARK-PUBLIC-TEST/${Date.now()}/2026`,
      perihal: "Surat public — tidak butuh watermark",
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  const suratID = (await createResp.json()).id as string;

  await page.goto(`/surat/${suratID}`);
  await page.getByTestId("add-attachment-btn").click();
  const roleSelect = page.locator('.n-modal').locator('.n-base-selection').first();
  await roleSelect.click();
  await page.locator(".n-base-select-option").filter({ hasText: "PDF Utama" }).click();

  const samplePDF = makeSamplePDF();
  await page.locator('[data-testid="add-attachment-upload"] input[type="file"]').setInputFiles({
    name: "public-doc.pdf",
    mimeType: "application/pdf",
    buffer: samplePDF,
  });
  await page.getByTestId("submit-add-attachment").click();
  await expect(page.getByText("1 file ter-upload")).toBeVisible({ timeout: 5000 });

  const detailResp = await request.get(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const detail = await detailResp.json();
  const attID = detail.attachments[0].id as string;

  const downloadResp = await request.get(`/api/surat/${suratID}/attachments/${attID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(downloadResp.status()).toBe(200);
  const downloaded = await downloadResp.body();

  // Public → identik dengan source bytes (tidak ada watermark)
  expect(downloaded.length).toBe(samplePDF.length);
  expect(Buffer.compare(downloaded, samplePDF)).toBe(0);

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});
