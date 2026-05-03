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

const minimalPDFV1 = Buffer.from(
  "%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Count 0>>endobj\nxref\n0 3\n0000000000 65535 f\n0000000009 00000 n\n0000000051 00000 n\ntrailer<</Size 3/Root 1 0 R>>\nstartxref\n89\n%%EOF",
);
const minimalPDFV2 = Buffer.from(
  "%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Count 1>>endobj\nxref\n0 3\n0000000000 65535 f\n0000000009 00000 n\n0000000051 00000 n\ntrailer<</Size 3/Root 1 0 R>>\nstartxref\n90\n%%EOF",
);

test("FULL UI: upload primary → replace dengan v2 → list shows v2 active, modal versi shows 2 entries", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  // Setup: surat dengan primary v1 lewat API
  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;
  const createResp = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: `REPLACE-TEST/${Date.now()}/2026`,
      perihal: "Surat untuk replace UI test",
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  const suratID = (await createResp.json()).id as string;

  // Upload v1 via UI (lebih simple test buka pakai UI dialog karena kita test full flow)
  await page.goto(`/surat/${suratID}`);
  await page.getByTestId("add-attachment-btn").click();

  // Pilih role "PDF Utama"
  const roleSelect = page.locator('.n-modal').locator('.n-base-selection').first();
  await roleSelect.click();
  await page.locator(".n-base-select-option").filter({ hasText: "PDF Utama" }).click();

  await page.locator('[data-testid="add-attachment-upload"] input[type="file"]').setInputFiles({
    name: "primary-v1.pdf",
    mimeType: "application/pdf",
    buffer: minimalPDFV1,
  });
  await page.getByTestId("submit-add-attachment").click();
  await expect(page.getByText("1 file ter-upload")).toBeVisible({ timeout: 5000 });

  // Verify primary-v1.pdf di list
  const lampiranCard = page.locator('.n-card').filter({ hasText: /^Lampiran/ });
  await expect(lampiranCard.getByText("primary-v1.pdf")).toBeVisible();

  // Click Replace button
  const replaceBtn = page.locator('[data-testid^="replace-attachment-btn-"]').first();
  await replaceBtn.click();
  await expect(page.locator('.n-modal').getByText("Replace Lampiran", { exact: true })).toBeVisible();

  // Upload v2
  await page.locator('[data-testid="replace-attachment-upload"] input[type="file"]').setInputFiles({
    name: "primary-v2.pdf",
    mimeType: "application/pdf",
    buffer: minimalPDFV2,
  });
  await page.getByTestId("submit-replace-attachment").click();
  await expect(page.getByText("Versi baru ter-upload")).toBeVisible({ timeout: 5000 });

  // Verify lampiran list now shows v2 (active), v1 hilang
  await expect(lampiranCard.getByText("primary-v2.pdf")).toBeVisible({ timeout: 3000 });
  await expect(lampiranCard.getByText("primary-v1.pdf")).toHaveCount(0);

  // Click Versi button → modal shows 2 entries
  const versionsBtn = page.locator('[data-testid^="versions-btn-"]').first();
  await versionsBtn.click();

  const modal = page.getByTestId("versions-modal");
  await expect(modal).toBeVisible();
  await expect(modal.getByText("Riwayat Versi")).toBeVisible();

  const items = modal.locator('[data-testid^="version-item-"]');
  await expect(items).toHaveCount(2);
  await expect(items.nth(0)).toContainText("primary-v1.pdf");
  await expect(items.nth(1)).toContainText("primary-v2.pdf");
  // V2 marked as Aktif
  await expect(items.nth(1).getByText("Aktif")).toBeVisible();

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});
