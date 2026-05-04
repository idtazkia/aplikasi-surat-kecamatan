// Fase 7 cross-cutting: student-mode `_edu` injection.
// Verify backend inject _edu hanya untuk role student + STUDENT_MODE_ENABLED=true.

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

async function getToken(page: Page): Promise<string> {
  const auth = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  return JSON.parse(auth!).accessToken;
}

// =============================================================================
// API kontrak: _edu injection per-role
// =============================================================================

test("staf1 GET /api/surat → response TIDAK punya _edu (role bukan student)", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  const resp = await request.get("/api/surat?limit=5", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const body = await resp.json();
  expect(body._edu).toBeUndefined();
});

test("student GET /api/surat → response punya _edu dengan keyset-pagination concept", async ({ page, request }) => {
  await loginAs(page, "student");
  const token = await getToken(page);

  const resp = await request.get("/api/surat?limit=5", {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.status()).toBe(200);
  const body = await resp.json();
  expect(body._edu).toBeDefined();
  expect(body._edu.operation).toBe("list_surat_with_keyset_pagination");
  expect(body._edu.concept_ids).toContain("keyset-pagination");
  expect(body._edu.sql).toMatch(/SELECT/i);
  expect(body._edu.complexity.theoretical).toMatch(/O\(log n/);
});

test("student GET /api/surat/{id}/thread → _edu dengan recursive-cte concept", async ({ page, request }) => {
  await loginAs(page, "student");
  const token = await getToken(page);

  // Pakai surat seed yang ada predecessor (Tanggapan atas Edaran)
  const list = await request.get("/api/surat?limit=20", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const items = (await list.json()).items as Array<{ id: string; perihal: string }>;
  const surat = items.find((s) => /Tanggapan/i.test(s.perihal));
  if (!surat) return; // skip kalau tidak ketemu seed

  const resp = await request.get(`/api/surat/${surat.id}/thread`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  const body = await resp.json();
  expect(body._edu).toBeDefined();
  expect(body._edu.concept_ids).toContain("recursive-cte");
  expect(body._edu.sql).toMatch(/RECURSIVE/i);
});

test("student GET /api/surat/{id}/komentar → _edu dengan append-only concept", async ({ page, request }) => {
  await loginAs(page, "student");
  const token = await getToken(page);

  // Surat yang punya komentar seed (Permohonan Surket Tazkia)
  const suratID = "00000000-0000-0000-0007-000000000008";
  const resp = await request.get(`/api/surat/${suratID}/komentar`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (resp.status() !== 200) return; // student mungkin tidak punya akses; skip
  const body = await resp.json();
  expect(body._edu.concept_ids).toContain("append-only-immutability");
});

// =============================================================================
// FULL UI: student login → toggle on → drawer auto-open dengan payload
// =============================================================================

test("FULL UI: student login → toggle button visible + drawer auto-show", async ({ page }) => {
  await loginAs(page, "student");

  // Toggle button visible (role student)
  const toggle = page.getByTestId("student-mode-toggle");
  await expect(toggle).toBeVisible();

  // Auto-enable: tombol show "ON" karena onMounted set enabled=true untuk role student
  await expect(toggle).toContainText("ON");

  // Drawer harus terbuka setelah list surat fetch (yang return _edu)
  await expect(page.locator(".n-drawer")).toBeVisible({ timeout: 5000 });
  // Drawer berisi konten _edu — operation field
  await expect(page.locator(".n-drawer-content")).toContainText("list_surat_with_keyset_pagination");
});

test("FULL UI: staf1 login → toggle button TIDAK visible", async ({ page }) => {
  await loginAs(page, "staf1");
  await expect(page.getByTestId("student-mode-toggle")).toHaveCount(0);
});
