import { test, expect } from "@playwright/test";

async function loginAs(page: import("@playwright/test").Page, username: string, password = "demo123") {
  await page.goto("/login");
  await page.evaluate(() => localStorage.clear());
  await page.goto("/login");
  await page.getByPlaceholder("staf1 / camat / admin / auditor").fill(username);
  await page.getByPlaceholder("demo123").fill(password);
  await Promise.all([
    page.waitForURL(/\/surat$/),
    page.getByRole("button", { name: "Masuk" }).click(),
  ]);
}

test("staf login -> redirect ke /surat dengan list dari demo seed", async ({ page }) => {
  await loginAs(page, "staf1");

  await expect(page).toHaveURL(/\/surat$/);
  await expect(page.getByText("Daftar Surat")).toBeVisible();

  // Demo seed punya 20 surat baseline (12 masuk + 8 keluar) + 6 dedup conflicts
  // = 26 visible (semua public). Limit 20 → muat lebih banyak available.
  const rows = page.locator("table tbody tr");
  await expect(rows.first()).toBeVisible({ timeout: 10000 });
  const count = await rows.count();
  expect(count).toBeGreaterThanOrEqual(15);
});

test("filter jenis 'masuk' menyaring hasil", async ({ page }) => {
  await loginAs(page, "staf1");

  // Klik select dropdown jenis
  await page.locator(".n-base-selection").first().click();
  // Pilih opsi "Surat Masuk" di dropdown menu
  await page.locator(".n-base-select-option").filter({ hasText: "Surat Masuk" }).click();
  await page.getByRole("button", { name: "Terapkan" }).click();

  // Tunggu data di-refetch
  await page.waitForTimeout(800);

  // Verify minimal ada 1 row dan tidak ada row dengan tag "Keluar"
  const rows = page.locator("table tbody tr");
  await expect(rows.first()).toBeVisible();
  const keluarTagCount = await page.locator("table tbody tr").locator(".n-tag", { hasText: "Keluar" }).count();
  expect(keluarTagCount).toBe(0);
});

test("search perihal 'pandemi' ketemu match", async ({ page }) => {
  await loginAs(page, "staf1");

  await page.getByPlaceholder("Kata kunci").fill("pandemi");
  await page.getByRole("button", { name: "Terapkan" }).click();
  await page.waitForTimeout(500);

  // Demo seed punya "Edaran Penanganan Pandemi Lanjutan"
  const rows = page.locator("table tbody tr");
  await expect(rows.first()).toBeVisible();
  const text = await rows.first().textContent();
  expect(text?.toLowerCase()).toContain("pandemi");
});

test("staf TIDAK lihat surat secret", async ({ page }) => {
  await loginAs(page, "staf1");

  // Demo seed: surat 0007-000000000006 (Audit Internal) dan
  // 0008-000000000005 (Laporan Tindak Lanjut Audit) punya access_level=secret.
  // Verify bahwa "Pemberitahuan Audit Internal" (secret) tidak muncul untuk staf.
  const allText = await page.locator("table tbody").textContent();
  expect(allText).not.toContain("Pemberitahuan Audit Internal Pemerintah Daerah");
});

test("camat lihat surat secret juga", async ({ page }) => {
  await loginAs(page, "camat");

  // Camat punya access ke secret. Cari surat audit secret dengan search
  await page.getByPlaceholder("Kata kunci").fill("Audit Internal");
  await page.getByRole("button", { name: "Terapkan" }).click();
  await page.waitForTimeout(500);

  const allText = await page.locator("table tbody").textContent();
  expect(allText).toContain("Audit Internal");
});
