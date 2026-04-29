// E2E test login flow — really submit form, no shortcut via API.
// Setiap test dimulai fresh (clear localStorage), DB state shared dari seed.
import { test, expect } from "@playwright/test";

test.beforeEach(async ({ page }) => {
  await page.goto("/login");
  await page.evaluate(() => localStorage.clear());
  await page.goto("/login");
});

test("login dengan kredensial valid -> redirect ke home", async ({ page }) => {
  await page.getByPlaceholder("staf1 / camat / admin").fill("staf1");
  await page.getByPlaceholder("demo123").fill("demo123");

  // Click submit, tunggu navigation
  await Promise.all([
    page.waitForURL(/\/surat$/),
    page.getByRole("button", { name: "Masuk" }).click(),
  ]);

  // Surat list view tampil
  await expect(page.getByText("Daftar Surat")).toBeVisible();

  // Token tersimpan di localStorage
  const stored = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  expect(stored).toBeTruthy();
  const parsed = JSON.parse(stored as string);
  expect(parsed.accessToken).toMatch(/^eyJ/); // JWT base64url prefix
  expect(parsed.refreshToken).toMatch(/^eyJ/);
  expect(parsed.roles).toContain("staf");
});

test("login dengan password salah -> error message tampil", async ({ page }) => {
  await page.getByPlaceholder("staf1 / camat / admin").fill("staf1");
  await page.getByPlaceholder("demo123").fill("password-salah");
  await page.getByRole("button", { name: "Masuk" }).click();

  await expect(page.getByText(/Username atau password salah/)).toBeVisible();
  await expect(page).toHaveURL(/\/login/);

  // localStorage tidak ter-set
  const stored = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  expect(stored).toBeNull();
});

test("login dengan user tidak dikenal -> error 401", async ({ page }) => {
  await page.getByPlaceholder("staf1 / camat / admin").fill("ghost-user");
  await page.getByPlaceholder("demo123").fill("anything");
  await page.getByRole("button", { name: "Masuk" }).click();

  await expect(page.getByText(/Username atau password salah/)).toBeVisible();
  await expect(page).toHaveURL(/\/login/);
});

test("login dengan field kosong -> warning", async ({ page }) => {
  await page.getByRole("button", { name: "Masuk" }).click();

  await expect(page.getByText(/Username dan password wajib diisi/)).toBeVisible();
  // Tetap di /login (no API call dilakukan)
  await expect(page).toHaveURL(/\/login/);
});

test("login sebagai camat -> role camat tampil di header", async ({ page }) => {
  await page.getByPlaceholder("staf1 / camat / admin").fill("camat");
  await page.getByPlaceholder("demo123").fill("demo123");
  await Promise.all([
    page.waitForURL(/\/surat$/),
    page.getByRole("button", { name: "Masuk" }).click(),
  ]);

  await expect(page.getByText(/\(camat\)/)).toBeVisible();
});

test("login sebagai admin -> role admin", async ({ page }) => {
  await page.getByPlaceholder("staf1 / camat / admin").fill("admin");
  await page.getByPlaceholder("demo123").fill("demo123");
  await Promise.all([
    page.waitForURL(/\/surat$/),
    page.getByRole("button", { name: "Masuk" }).click(),
  ]);

  await expect(page.getByText(/\(admin\)/)).toBeVisible();
});
