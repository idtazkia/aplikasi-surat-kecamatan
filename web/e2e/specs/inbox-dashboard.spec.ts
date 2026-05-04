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

// =============================================================================
// INBOX
// =============================================================================

test("staf navigate ke /inbox via topbar → list disposisi seed", async ({ page }) => {
  await loginAs(page, "staf1");

  // Topbar punya nav-inbox link
  await page.getByTestId("nav-inbox").click();
  await expect(page).toHaveURL(/\/inbox$/);
  await expect(page.getByText("Inbox: Disposisi Saya")).toBeVisible();

  // Dari seed: staf1 punya disposisi 0007-...0008 (Permohonan Surket Tazkia)
  // dan 0007-...000a (audiensi karang taruna - tapi itu untuk staf2)
  // Verify ada minimal 1 item dari seed
  await expect(page.locator('[data-testid^="inbox-item-"]').first()).toBeVisible({ timeout: 5000 });
});

test("filter status di inbox → hanya disposisi yang match status terlihat", async ({ page }) => {
  await loginAs(page, "staf2");
  await page.goto("/inbox");

  // Pilih filter "Selesai"
  const statusSelect = page.locator('.n-base-selection').first();
  await statusSelect.click();
  await page.locator('.n-base-select-option').filter({ hasText: "Selesai" }).click();

  // Wait for re-fetch
  await page.waitForTimeout(500);

  // Semua item yang muncul harus punya status "Selesai"
  // Kalau staf2 tidak punya disposisi done, bisa kosong — yang penting tidak ada in_progress/pending
  const tags = page.locator('[data-testid^="inbox-item-"] .n-tag');
  const count = await tags.count();
  for (let i = 0; i < count; i++) {
    const text = await tags.nth(i).textContent();
    // Setiap item bisa punya banyak tag; setidaknya satu tag harus mengandung "Selesai" atau "Overdue"
    // Kita cek tidak ada tag "Pending" atau "Sedang dikerjakan"
    expect(text).not.toMatch(/^Pending$|^Sedang dikerjakan$/);
  }
});

test("klik inbox item → navigate ke surat detail", async ({ page }) => {
  await loginAs(page, "staf1");
  await page.goto("/inbox");

  const firstItem = page.locator('[data-testid^="inbox-item-"]').first();
  await firstItem.waitFor({ state: "visible", timeout: 5000 });
  await firstItem.click();

  await expect(page).toHaveURL(/\/surat\/[\w-]+$/);
});

// =============================================================================
// DASHBOARD CAMAT
// =============================================================================

test("staf TIDAK punya akses /dashboard → redirect ke /surat", async ({ page }) => {
  await loginAs(page, "staf1");
  await page.goto("/dashboard");

  // Router guard redirect karena role bukan camat
  await expect(page).toHaveURL(/\/surat$/);
});

test("staf TIDAK lihat tombol Dashboard di topbar", async ({ page }) => {
  await loginAs(page, "staf1");
  await expect(page.getByTestId("nav-inbox")).toBeVisible();
  await expect(page.getByTestId("nav-dashboard")).toHaveCount(0);
});

test("camat: navigate ke /dashboard via topbar → 4 cards muncul dengan stats", async ({ page }) => {
  await loginAs(page, "camat");

  await page.getByTestId("nav-dashboard").click();
  await expect(page).toHaveURL(/\/dashboard$/);
  await expect(page.getByText("Dashboard Camat")).toBeVisible();

  // 4 cards
  await expect(page.getByTestId("card-surat-masuk-hari-ini")).toBeVisible();
  await expect(page.getByTestId("card-disposisi-belum-assign")).toBeVisible();
  await expect(page.getByTestId("card-disposisi-overdue")).toBeVisible();
  await expect(page.getByTestId("card-disposisi-mine")).toBeVisible();

  // Click "Disposisi Saya" → navigate ke inbox
  await page.getByTestId("card-disposisi-mine").click();
  await expect(page).toHaveURL(/\/inbox$/);
});
