import { test, expect } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.goto("/login");
  await page.evaluate(() => localStorage.clear());
});

test("akses / tanpa login -> redirect ke /login dengan ?next", async ({ page }) => {
  await page.goto("/surat");
  await expect(page).toHaveURL(/\/login\?next=/);
});

test("login -> refresh page -> tetap authenticated", async ({ page }) => {
  await page.goto("/login");
  await page.getByPlaceholder("staf1 / camat / admin").fill("staf1");
  await page.getByPlaceholder("demo123").fill("demo123");
  await Promise.all([
    page.waitForURL(/\/surat$/),
    page.getByRole("button", { name: "Masuk" }).click(),
  ]);

  // Hard refresh
  await page.reload();

  await expect(page).toHaveURL(/\/surat$/);
  await expect(page.getByText("Daftar Surat")).toBeVisible();
});

test("logout -> token cleared dari localStorage + redirect login", async ({ page }) => {
  await page.goto("/login");
  await page.getByPlaceholder("staf1 / camat / admin").fill("staf1");
  await page.getByPlaceholder("demo123").fill("demo123");
  await Promise.all([
    page.waitForURL(/\/surat$/),
    page.getByRole("button", { name: "Masuk" }).click(),
  ]);

  // Click Keluar
  await Promise.all([
    page.waitForURL(/\/login/),
    page.getByRole("button", { name: "Keluar" }).click(),
  ]);

  const stored = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  expect(stored).toBeNull();
});

test("dark mode toggle persist setelah refresh", async ({ page }) => {
  await page.goto("/login");
  await page.getByPlaceholder("staf1 / camat / admin").fill("staf1");
  await page.getByPlaceholder("demo123").fill("demo123");
  await Promise.all([
    page.waitForURL(/\/surat$/),
    page.getByRole("button", { name: "Masuk" }).click(),
  ]);

  // Toggle dark mode lewat tombol di header (☾ atau ☀)
  await page.getByRole("button", { name: /☾|☀/ }).click();

  // Verify localStorage
  const theme = await page.evaluate(() => localStorage.getItem("surat-kec-theme"));
  expect(theme).toBe("dark");

  // Refresh, masih dark
  await page.reload();
  const themeAfter = await page.evaluate(() => localStorage.getItem("surat-kec-theme"));
  expect(themeAfter).toBe("dark");
});

test("/healthz dapat diakses tanpa auth", async ({ page }) => {
  const resp = await page.request.get("/healthz");
  expect(resp.status()).toBe(200);
  const body = await resp.json();
  expect(body.status).toBe("ok");
});
