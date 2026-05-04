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

// =============================================================================
// API kontrak: stats endpoints
// =============================================================================

test("camat: GET /api/stats/by-period → seed punya minimal 1 bucket", async ({ page, request }) => {
  await loginAs(page, "camat");
  const token = await getToken(page);

  const resp = await request.get("/api/stats/by-period", {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.status()).toBe(200);
  const body = await resp.json();
  expect(body.items.length).toBeGreaterThan(0);
  // Bucket format YYYY-MM
  expect(body.items[0].bucket).toMatch(/^\d{4}-\d{2}$/);
  expect(body.items[0].jenis_count).toBeDefined();
});

test("camat: GET /api/stats/by-classification → return per-klasifikasi count", async ({ page, request }) => {
  await loginAs(page, "camat");
  const token = await getToken(page);

  const resp = await request.get("/api/stats/by-classification", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const body = await resp.json();
  expect(body.items.length).toBeGreaterThan(0);
  for (const item of body.items) {
    expect(item.count).toBeGreaterThan(0);
  }
});

test("camat: GET /api/stats/by-sender?top=3 → max 3 entries, descending count", async ({ page, request }) => {
  await loginAs(page, "camat");
  const token = await getToken(page);

  const resp = await request.get("/api/stats/by-sender?top=3", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const body = await resp.json();
  expect(body.items.length).toBeLessThanOrEqual(3);
  // Descending order
  for (let i = 1; i < body.items.length; i++) {
    expect(body.items[i].count).toBeLessThanOrEqual(body.items[i - 1].count);
  }
});

test("camat: GET /api/stats/staff-load → status_count + overdue per staf", async ({ page, request }) => {
  await loginAs(page, "camat");
  const token = await getToken(page);

  const resp = await request.get("/api/stats/staff-load", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const body = await resp.json();
  expect(body.items.length).toBeGreaterThan(0);
  for (const item of body.items) {
    expect(typeof item.status_count).toBe("object");
    expect(typeof item.overdue_count).toBe("number");
    expect(typeof item.total_active).toBe("number");
  }
});

test("staf TIDAK punya akses stats endpoints → 403", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  for (const path of [
    "/api/stats/by-period",
    "/api/stats/by-classification",
    "/api/stats/by-sender",
    "/api/stats/staff-load",
  ]) {
    const resp = await request.get(path, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(resp.status()).toBe(403);
  }
});

// =============================================================================
// FULL UI: navigate ke /stats, lihat 4 cards
// =============================================================================

test("staf TIDAK lihat tombol Statistik di topbar + akses /stats redirect", async ({ page }) => {
  await loginAs(page, "staf1");
  await expect(page.getByTestId("nav-stats")).toHaveCount(0);

  await page.goto("/stats");
  await expect(page).toHaveURL(/\/surat$/);
});

test("FULL UI: camat → topbar Statistik → 4 cards visible dengan data seed", async ({ page }) => {
  await loginAs(page, "camat");

  await page.getByTestId("nav-stats").click();
  await expect(page).toHaveURL(/\/stats$/);
  // Header "Statistik" — pakai exact match supaya tidak match instansi
  // "Badan Pusat Statistik" yang muncul di Top Pengirim card.
  await expect(page.getByText("Statistik", { exact: true })).toBeVisible();

  // 4 cards
  await expect(page.getByTestId("stats-period-card")).toBeVisible({ timeout: 5000 });
  await expect(page.getByTestId("stats-sender-card")).toBeVisible();
  await expect(page.getByTestId("stats-classification-card")).toBeVisible();
  await expect(page.getByTestId("stats-staff-load-card")).toBeVisible();

  // Period card: minimal 1 row
  const periodRows = page.locator('[data-testid^="period-row-"]');
  await expect(periodRows.first()).toBeVisible({ timeout: 3000 });

  // Sender card: minimal 1 row
  const senderRows = page.locator('[data-testid^="sender-row-"]');
  await expect(senderRows.first()).toBeVisible({ timeout: 3000 });
});
