// Issue #1: tenant config dari /api/config (backend-served).
// Verify endpoint shape + frontend boot integration.

import { test, expect } from "@playwright/test";

test("GET /api/config public (no auth) → return tenant DTO", async ({ request }) => {
  const resp = await request.get("/api/config");
  expect(resp.status()).toBe(200);
  const body = await resp.json();

  // Required fields per env TENANT_* di global-setup.ts
  expect(body.apiBaseUrl).toBe("/api");
  expect(body.appName).toBe("Aplikasi Surat Kecamatan");
  expect(body.institutionName).toBe("Kantor Kecamatan Demo");
  expect(body.branding.primary).toMatch(/^#[0-9a-fA-F]+$/);
  expect(body.branding.primaryHover).toMatch(/^#[0-9a-fA-F]+$/);
  expect(body.branding.accent).toMatch(/^#[0-9a-fA-F]+$/);

  // Cache-Control: no-store supaya tenant config tidak ter-cache stale
  expect(resp.headers()["cache-control"]).toBe("no-store");
});

test("FULL UI: appName dari config muncul di login card", async ({ page }) => {
  await page.goto("/login");
  // App boot fetch /api/config terlebih dulu, lalu render login dengan appName.
  await expect(
    page.locator(".login-card").getByText("Aplikasi Surat Kecamatan", { exact: true }),
  ).toBeVisible({ timeout: 5000 });
});
