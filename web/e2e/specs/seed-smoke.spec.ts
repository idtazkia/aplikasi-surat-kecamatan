// Smoke tests yang exercise demo seed dataset secara read-only.
// Tujuan: ensure seed data terbaca konsisten di semua feature surface
// (detail komentar, disposisi seed, references) tanpa create state baru.

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

// =============================================================================
// SEED KOMENTAR display
// =============================================================================

test("seed komentar tampil di detail surat 0007-...0008 (Permohonan Surket Tazkia) — 3 komentar berurutan", async ({ page }) => {
  await loginAs(page, "staf1");

  // Surat seed dengan 3 komentar (lihat 0007_seed_workflow.sql)
  await page.goto("/surat/00000000-0000-0000-0007-000000000008");

  const card = page.getByTestId("komentar-card");
  await expect(card).toBeVisible();

  const items = card.locator('[data-testid^="komentar-item-"]');
  await expect(items).toHaveCount(3);

  // ASC order: staf cek dokumen (1st) → camat lanjutkan (2nd) → staf done (3rd)
  await expect(items.nth(0)).toContainText(/cek kelengkapan dokumen/i);
  await expect(items.nth(1)).toContainText(/format SK domisili/i);
  await expect(items.nth(2)).toContainText(/diserahkan ke kurir/i);
});

// =============================================================================
// SEED DISPOSISI display + relasi ke surat asal
// =============================================================================

test("seed disposisi 'Permohonan Surket Tazkia' tampil dengan status done + assignee staf1", async ({ page }) => {
  await loginAs(page, "camat");

  // Surat 0007-...0008 punya disposisi seed: assigned_to staf1, status done
  await page.goto("/surat/00000000-0000-0000-0007-000000000008");

  const card = page.getByTestId("disposisi-card");
  await expect(card).toBeVisible();

  const items = card.locator('[data-testid^="disposisi-item-"]');
  await expect(items).toHaveCount(1);

  // Status seed: done
  await expect(items.nth(0).getByText("Selesai", { exact: true })).toBeVisible();
  // Instruksi mengandung "surat keterangan domisili"
  await expect(items.nth(0)).toContainText(/surat keterangan domisili/i);
  // Assignee staf1 = "Siti Aminah" (full_name dari seed users)
  await expect(items.nth(0)).toContainText(/Siti Aminah/);
  // Creator: Bu Camat
  await expect(items.nth(0)).toContainText(/Bu Camat/);
});

test("seed disposisi 'audiensi karang taruna' tampil dengan status in_progress + deadline", async ({ page }) => {
  await loginAs(page, "camat");

  // Surat 0007-...000c (Permohonan audiensi karang taruna)
  await page.goto("/surat/00000000-0000-0000-0007-00000000000c");

  const card = page.getByTestId("disposisi-card");
  const items = card.locator('[data-testid^="disposisi-item-"]');
  await expect(items).toHaveCount(1);

  // Status in_progress
  await expect(items.nth(0).getByText("Sedang dikerjakan", { exact: true })).toBeVisible();
  // Deadline 2026-04-30 — saat tes dijalankan tanggal sekarang (Mei 2026), seharusnya overdue
  await expect(items.nth(0).getByText("Overdue", { exact: true })).toBeVisible();
});

// =============================================================================
// SEED INBOX (staf2 punya 2 disposisi assigned)
// =============================================================================

test("inbox staf2 menampilkan 2 disposisi seed (audiensi + data BPS)", async ({ page }) => {
  await loginAs(page, "staf2");
  await page.goto("/inbox");

  // Tunggu list render
  await page.locator('[data-testid^="inbox-item-"]').first().waitFor({ state: "visible", timeout: 5000 });

  const items = page.locator('[data-testid^="inbox-item-"]');
  const count = await items.count();
  // Minimal 2 disposisi seed assigned ke staf2 (lihat 0007_seed_workflow.sql)
  expect(count).toBeGreaterThanOrEqual(2);
});

// =============================================================================
// SEED REFERENCES → thread (cross-feature integrity)
// =============================================================================

test("dari seed surat 'Tanggapan atas Edaran' → thread modal show predecessor 'Edaran Penanganan Pandemi'", async ({ page }) => {
  await loginAs(page, "staf1");

  await page.getByPlaceholder("Kata kunci").fill("Tanggapan atas Edaran");
  await page.getByRole("button", { name: "Terapkan" }).click();
  await page.waitForTimeout(500);

  await page.locator("table tbody tr").first().click();
  await expect(page).toHaveURL(/\/surat\/[\w-]+$/);

  await page.getByTestId("thread-view-btn").click();
  const modal = page.getByTestId("thread-modal");
  await expect(modal).toBeVisible();

  // Predecessor: Edaran Penanganan Pandemi
  await expect(modal.getByText(/Edaran Penanganan Pandemi/i)).toBeVisible();
});
