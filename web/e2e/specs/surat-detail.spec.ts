import { test, expect } from "@playwright/test";

async function loginAs(page: import("@playwright/test").Page, username: string) {
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

test("klik row di list -> navigate ke detail surat", async ({ page }) => {
  await loginAs(page, "staf1");

  // Click first row
  const firstRow = page.locator("table tbody tr").first();
  await firstRow.click();

  await expect(page).toHaveURL(/\/surat\/[\w-]+$/);
  await expect(page.getByText("Detail Surat")).toBeVisible();
});

test("detail surat masuk menampilkan metadata + lampiran section + riwayat korespondensi", async ({ page }) => {
  await loginAs(page, "staf1");

  // Filter ke surat yang punya references — search "tanggapan" menemukan
  // surat keluar yang membalas edaran pandemi (Skenario seed reference)
  await page.getByPlaceholder("Kata kunci").fill("Tanggapan atas Edaran");
  await page.getByRole("button", { name: "Terapkan" }).click();
  await page.waitForTimeout(500);

  await page.locator("table tbody tr").first().click();
  await expect(page).toHaveURL(/\/surat\/[\w-]+$/);

  // Header
  await expect(page.getByText("Tanggapan atas Edaran")).toBeVisible();

  // Sections
  await expect(page.getByText("Lampiran", { exact: true })).toBeVisible();
  await expect(page.getByText("Riwayat Korespondensi", { exact: true })).toBeVisible();

  // Surat ini predecessor: balasan dari "Edaran Penanganan Pandemi"
  await expect(page.getByText("Membalas")).toBeVisible();
});

test("staf akses surat secret langsung -> redirect dengan error", async ({ page }) => {
  await loginAs(page, "staf1");

  // ID seed surat secret: Pemberitahuan Audit Internal
  const secretID = "00000000-0000-0000-0007-000000000006";
  await page.goto(`/surat/${secretID}`);

  // Akan di-redirect ke /surat dengan message error
  await page.waitForURL(/\/surat$/, { timeout: 5000 });
});

test("akses surat tidak ada -> redirect ke list dengan error", async ({ page }) => {
  await loginAs(page, "staf1");

  await page.goto("/surat/00000000-0000-0000-9999-999999999999");
  await page.waitForURL(/\/surat$/, { timeout: 5000 });
});

test("dari detail kembali ke list via tombol Daftar", async ({ page }) => {
  await loginAs(page, "staf1");

  await page.locator("table tbody tr").first().click();
  await expect(page).toHaveURL(/\/surat\/[\w-]+$/);

  await page.getByText("← Daftar").click();
  await expect(page).toHaveURL(/\/surat$/);
});
