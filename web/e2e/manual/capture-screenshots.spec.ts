// Capture screenshots untuk user manual.
//
// Strategi: 1 test per role/section flow yang setup state via API + UI,
// lalu navigate ke setiap halaman dan save screenshot. Output ke
// docs/user-manual/src/screenshots/<section>/<id>.png.
//
// Dijalankan via:
//   make user-manual-capture
//   atau: npx playwright test --config=playwright.manual.config.ts
//
// Ekspektasi: testcontainer postgres + Go backend di-spin-up oleh
// global-setup; frontend dev server via webServer config.

import { test, type Page, type APIRequestContext } from "@playwright/test";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { mkdir } from "node:fs/promises";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(__dirname, "../../..");
const SCREENSHOTS_DIR = path.join(REPO_ROOT, "docs/user-manual/src/screenshots");

async function ensureDir(section: string): Promise<string> {
  const dir = path.join(SCREENSHOTS_DIR, section);
  await mkdir(dir, { recursive: true });
  return dir;
}

async function snap(page: Page, section: string, id: string, fullPage = false) {
  const dir = await ensureDir(section);
  await page.screenshot({
    path: path.join(dir, `${id}.png`),
    fullPage,
  });
  console.log(`📸 ${section}/${id}.png`);
}

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
  // Tunggu list render + initial sync
  await page.waitForTimeout(500);
}

async function getToken(page: Page): Promise<string> {
  const auth = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  return JSON.parse(auth!).accessToken;
}

// =============================================================================
// SECTION: pengenalan — login + landing
// =============================================================================

test("capture: pengenalan", async ({ page }) => {
  // Login page (logout state)
  await page.goto("/login");
  await page.evaluate(() => localStorage.clear());
  await page.goto("/login");
  await page.waitForTimeout(300);
  await snap(page, "pengenalan", "01-login");

  // Staf landing
  await loginAs(page, "staf1");
  await snap(page, "pengenalan", "02-staf-landing");

  // Camat landing — show Dashboard + Statistik + Rekonsiliasi nav
  await loginAs(page, "camat");
  await snap(page, "pengenalan", "03-camat-landing");

  // Auditor landing — read-only, no "+ Surat Baru" button
  await loginAs(page, "auditor");
  await snap(page, "pengenalan", "04-auditor-landing");
});

// =============================================================================
// SECTION: input-surat — form, upload, hasil
// =============================================================================

test("capture: input surat", async ({ page, request }) => {
  await loginAs(page, "staf1");

  // Form kosong
  await page.goto("/surat/baru");
  await page.waitForTimeout(300);
  await snap(page, "input-surat", "01-form-kosong");

  // Form terisi (jenis masuk + perihal + tanggal)
  await page.getByPlaceholder("045/123/IV/2026").fill("044/PKM-DEMO/V/2026");
  await page.getByPlaceholder("Subject surat").fill("Permohonan Audiensi Pengabdian Masyarakat STMIK Tazkia");

  // Pick instansi via search
  const instansiInput = page.locator('[data-testid="instansi-field"] input').first();
  await page.locator('[data-testid="instansi-field"] .n-base-selection').click();
  await page.waitForTimeout(150);
  await instansiInput.fill("Kemen");
  await page.waitForTimeout(800);
  await page.locator(".n-base-select-option:visible").first().click();

  await page.waitForTimeout(300);
  await snap(page, "input-surat", "02-form-terisi");

  // Detail surat existing dengan attachment lengkap untuk demo "hasil" view
  // Pakai surat seed yang punya komentar + disposisi (Permohonan Surket Tazkia)
  await page.goto("/surat/00000000-0000-0000-0007-000000000008");
  await page.waitForTimeout(800);
  await snap(page, "input-surat", "03-detail-surat", true);

  // Edit form
  await page.goto("/surat/00000000-0000-0000-0007-000000000008/edit");
  await page.waitForTimeout(500);
  await snap(page, "input-surat", "04-form-edit");

  // Cleanup: tidak perlu — tidak ada surat baru di-create di test ini
  void request;
});

// =============================================================================
// SECTION: detail-surat — komponen yang ada di halaman detail
// =============================================================================

test("capture: detail-surat features", async ({ page }) => {
  await loginAs(page, "camat");

  // Surat dengan thread korespondensi (Tanggapan atas Edaran Pandemi)
  await page.getByPlaceholder("Kata kunci").fill("Tanggapan atas Edaran");
  await page.getByRole("button", { name: "Terapkan" }).click();
  await page.waitForTimeout(500);
  await page.locator("table tbody tr").first().click();
  await page.waitForURL(/\/surat\/[\w-]+$/);
  await page.waitForTimeout(500);

  // Klik thread modal
  await page.getByTestId("thread-view-btn").click();
  await page.waitForTimeout(500);
  await snap(page, "detail-surat", "01-thread-modal");

  // Close modal
  await page.keyboard.press("Escape");
  await page.waitForTimeout(300);

  // Kembali ke surat dengan komentar lengkap
  await page.goto("/surat/00000000-0000-0000-0007-000000000008");
  await page.waitForTimeout(500);

  // Disposisi card (sudah tampil)
  await snap(page, "detail-surat", "02-disposisi-list");

  // Komentar thread (scroll ke bawah)
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
  await page.waitForTimeout(300);
  await snap(page, "detail-surat", "03-komentar-thread");
});

// =============================================================================
// SECTION: disposisi — assign + status flow
// =============================================================================

test("capture: disposisi flow", async ({ page, request }) => {
  await loginAs(page, "camat");
  const token = await getToken(page);

  // Setup: surat baru tanpa disposisi
  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;
  const create = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: `MANUAL-DISP/${Date.now()}`,
      perihal: "Demo: Permohonan Disposisi untuk User Manual",
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  const suratID = (await create.json()).id;
  await page.goto(`/surat/${suratID}`);
  await page.waitForTimeout(500);

  // Snap disposisi card kosong
  await snap(page, "disposisi", "01-card-kosong");

  // Buka dialog
  await page.getByTestId("add-disposisi-btn").click();
  await page.waitForTimeout(400);

  // Isi assignee + instruksi
  const assigneeSelect = page.locator(
    '[data-testid="disposisi-assignee-select"] .n-base-selection',
  );
  await assigneeSelect.click();
  await page.locator(".n-base-select-option").filter({ hasText: /staf1|Siti/ }).first().click();
  await page.locator('[data-testid="disposisi-instruksi-input"] textarea').fill(
    "Tolong proses surat ini sesuai SOP. Koordinasi dengan tim sebelum 30 April.",
  );
  await page.waitForTimeout(300);
  await snap(page, "disposisi", "02-dialog-isi");

  // Submit
  await page.getByTestId("submit-add-disposisi").click();
  await page.waitForTimeout(800);
  await snap(page, "disposisi", "03-pending");

  // Login as staf1, lihat di inbox
  await loginAs(page, "staf1");
  await page.goto("/inbox");
  await page.waitForTimeout(500);
  await snap(page, "disposisi", "04-inbox-staf");

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

// =============================================================================
// SECTION: dashboard-stats — supervisi camat
// =============================================================================

test("capture: dashboard + stats", async ({ page }) => {
  await loginAs(page, "camat");

  // Dashboard 4 cards
  await page.goto("/dashboard");
  await page.waitForTimeout(800);
  await snap(page, "dashboard-stats", "01-dashboard-camat");

  // Statistik
  await page.goto("/stats");
  await page.waitForTimeout(800);
  await snap(page, "dashboard-stats", "02-stats", true);
});

// =============================================================================
// SECTION: rekonsiliasi — handle dedup queue
// =============================================================================

test("capture: rekonsiliasi", async ({ page }) => {
  await loginAs(page, "camat");

  // List page
  await page.goto("/reconciliation");
  await page.waitForTimeout(800);
  await snap(page, "rekonsiliasi", "01-list");

  // Klik group pertama untuk detail merge
  const firstGroup = page.locator('[data-testid^="recon-group-"]').first();
  if ((await firstGroup.count()) > 0) {
    await firstGroup.click();
    await page.waitForTimeout(800);
    await snap(page, "rekonsiliasi", "02-detail-merge", true);
  }
});

// =============================================================================
// SECTION: notifikasi-sync — bell, pending sync, offline banner, sync btn
// =============================================================================

test("capture: notifikasi & sync", async ({ page, request, context }) => {
  // Setup: assign disposisi via API supaya staf1 punya notif
  await loginAs(page, "camat");
  const camatToken = await getToken(page);

  const usersResp = await request.get("/api/users/assignable", {
    headers: { Authorization: `Bearer ${camatToken}` },
  });
  const users = (await usersResp.json()).items as { id: string; username: string }[];
  const staf1 = users.find((u) => u.username === "staf1")!;

  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${camatToken}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;

  const surat = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${camatToken}` },
    data: {
      jenis: "masuk",
      nomor_surat: `MANUAL-NOTIF/${Date.now()}`,
      perihal: "Demo: Surat untuk notifikasi screenshot",
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  const suratID = (await surat.json()).id;
  await request.post("/api/disposisi", {
    headers: { Authorization: `Bearer ${camatToken}` },
    data: {
      surat_id: suratID,
      assigned_to: staf1.id,
      instruksi: "Demo notifikasi untuk capture user manual screenshot",
    },
  });

  // Login staf1, klik bell
  await loginAs(page, "staf1");
  await page.getByTestId("notif-bell").click();
  await page.waitForTimeout(500);
  await snap(page, "notifikasi-sync", "01-notif-dropdown");

  // Close popover
  await page.keyboard.press("Escape");
  await page.waitForTimeout(200);

  // Pending sync indicator: trigger edit offline supaya muncul
  await page.goto(`/surat/${suratID}/edit`);
  await page.waitForTimeout(500);
  await context.setOffline(true);
  await page.evaluate(() => window.dispatchEvent(new Event("offline")));
  await page.waitForTimeout(300);

  // Snap offline banner
  await snap(page, "notifikasi-sync", "02-offline-banner");

  // Edit + submit (akan masuk pending queue)
  const perihalArea = page.locator("textarea").first();
  await perihalArea.fill("Edited offline untuk demo pending sync indicator");
  await page.getByRole("button", { name: "Simpan Perubahan" }).click();
  await page.waitForTimeout(500);

  // Reconnect supaya bisa navigate
  await context.setOffline(false);
  await page.evaluate(() => window.dispatchEvent(new Event("online")));
  await page.goto("/surat");
  await page.waitForTimeout(300);

  // Klik pending indicator badge
  await page.getByTestId("pending-sync-button").click();
  await page.waitForTimeout(500);
  await snap(page, "notifikasi-sync", "03-pending-sync");
  await page.keyboard.press("Escape");

  // Sync button highlighted
  await page.locator('[data-testid="global-sync-btn"]').hover();
  await page.waitForTimeout(200);
  await snap(page, "notifikasi-sync", "04-sync-button");

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${camatToken}` },
  });
});

// =============================================================================
// SECTION: student-mode — drawer dengan _edu payload
// =============================================================================

test("capture: student-mode drawer", async ({ page }) => {
  await loginAs(page, "student");
  // Drawer auto-open setelah list fetch karena role student
  await page.waitForTimeout(1500);
  await snap(page, "student-mode", "01-drawer", true);
});

// =============================================================================
// SECTION: auditor — read-only view
// =============================================================================

test("capture: auditor view", async ({ page }) => {
  await loginAs(page, "auditor");

  // Surat detail view tanpa Edit/Hapus + tanpa add buttons
  await page.locator("table tbody tr").first().click();
  await page.waitForURL(/\/surat\/[\w-]+$/);
  await page.waitForTimeout(500);
  await snap(page, "auditor", "01-detail-readonly", true);
});
