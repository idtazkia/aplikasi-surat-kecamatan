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

async function createTestSurat(
  request: import("@playwright/test").APIRequestContext,
  token: string,
  perihal: string,
) {
  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;
  const createResp = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: `DISPOSISI-TEST/${Date.now()}/2026`,
      perihal,
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  const created = await createResp.json();
  return created.id as string;
}

// =============================================================================
// FULL UI: camat assigns disposisi → staf melihat & marks done
// =============================================================================

test("FULL UI: camat buat disposisi via UI → muncul di list dengan status pending", async ({ page, request }) => {
  await loginAs(page, "camat");
  const token = await getToken(page);
  const suratID = await createTestSurat(request, token, "Surat untuk disposisi UI test");

  await page.goto(`/surat/${suratID}`);
  await expect(page.getByTestId("disposisi-card")).toBeVisible();
  await expect(page.getByText("Belum ada disposisi")).toBeVisible();

  // Buka dialog
  await page.getByTestId("add-disposisi-btn").click();
  await expect(page.locator(".n-modal").getByText("Buat Disposisi", { exact: true })).toBeVisible();

  // Pilih assignee — buka dropdown lalu pilih staf1
  const assigneeSelect = page.locator('[data-testid="disposisi-assignee-select"] .n-base-selection');
  await assigneeSelect.click();
  await page.locator('.n-base-select-option').filter({ hasText: /staf1|Staf Kecamatan 1/ }).first().click();

  // Isi instruksi
  await page.locator('[data-testid="disposisi-instruksi-input"] textarea').fill("Tolong proses surat ini sesuai SOP");

  // Submit
  await page.getByTestId("submit-add-disposisi").click();
  await expect(page.getByText("Disposisi dibuat")).toBeVisible({ timeout: 3000 });

  // Verify muncul di list
  const card = page.getByTestId("disposisi-card");
  await expect(card.getByText("Tolong proses surat ini sesuai SOP")).toBeVisible();
  await expect(card.getByText("Pending", { exact: true })).toBeVisible();

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

test("FULL UI: camat buat disposisi WITH deadline picker → display deadline di list", async ({ page, request }) => {
  await loginAs(page, "camat");
  const token = await getToken(page);
  const suratID = await createTestSurat(request, token, "Disposisi deadline UI test");

  await page.goto(`/surat/${suratID}`);
  await page.getByTestId("add-disposisi-btn").click();

  // Assignee
  const assigneeSelect = page.locator('[data-testid="disposisi-assignee-select"] .n-base-selection');
  await assigneeSelect.click();
  await page.locator('.n-base-select-option').filter({ hasText: /staf1|Staf Kecamatan 1/ }).first().click();

  // Instruksi
  await page.locator('[data-testid="disposisi-instruksi-input"] textarea').fill("Test deadline picker");

  // Pick deadline lewat NDatePicker datetime — klik input, navigate calendar, pilih hari, confirm
  const deadlineInput = page.locator('.n-modal .n-date-picker input').first();
  await deadlineInput.click();
  await page.waitForSelector(".n-date-panel", { timeout: 3000 });

  // Navigate ke June 2026 dari current header
  const targetDate = new Date(2026, 5, 30); // 30 Juni 2026
  const monthHeader = page.locator(".n-date-panel-month__month-year").first();
  for (let i = 0; i < 60; i++) {
    const headerText = (await monthHeader.textContent())?.trim() ?? "";
    const m = headerText.match(/(\d{1,2})[^\d]+(\d{4})/);
    if (!m) break;
    const currentMonth = parseInt(m[1], 10) - 1;
    const currentYear = parseInt(m[2], 10);
    if (currentMonth === targetDate.getMonth() && currentYear === targetDate.getFullYear()) break;
    const goNext =
      new Date(targetDate.getFullYear(), targetDate.getMonth()).getTime() >
      new Date(currentYear, currentMonth).getTime();
    await page.locator(goNext ? ".n-date-panel-month__next" : ".n-date-panel-month__prev").first().click();
    await page.waitForTimeout(80);
  }

  await page
    .locator(".n-date-panel-date:not(.n-date-panel-date--excluded)")
    .filter({ hasText: /^30$/ })
    .first()
    .click();
  await page.waitForTimeout(150);

  // Datetime picker butuh confirm (panel actions di kanan bawah)
  const confirmBtn = page.locator(".n-date-panel-actions__suffix .n-button").last();
  if (await confirmBtn.isVisible()) {
    await confirmBtn.click();
  }
  await page.waitForTimeout(200);

  await page.getByTestId("submit-add-disposisi").click();
  await expect(page.getByText("Disposisi dibuat")).toBeVisible({ timeout: 3000 });

  const card = page.getByTestId("disposisi-card");
  await expect(card.getByText("Test deadline picker")).toBeVisible();
  // Deadline label muncul di description
  await expect(card.getByText(/Deadline:.*30/i)).toBeVisible();

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

test("FULL UI: assignee mulai → selesai → status berubah", async ({ page, request }) => {
  // Camat buat disposisi via API (state prep), kemudian staf update via UI
  await loginAs(page, "camat");
  const camatToken = await getToken(page);
  const suratID = await createTestSurat(request, camatToken, "Surat status update UI test");

  // Lookup staf1 ID
  const usersResp = await request.get("/api/users/assignable", {
    headers: { Authorization: `Bearer ${camatToken}` },
  });
  const users = (await usersResp.json()).items as { id: string; username: string }[];
  const staf1 = users.find((u) => u.username === "staf1")!;

  // Create disposisi via API
  await request.post("/api/disposisi", {
    headers: { Authorization: `Bearer ${camatToken}` },
    data: {
      surat_id: suratID,
      assigned_to: staf1.id,
      instruksi: "Test status update flow",
    },
  });

  // Login sebagai staf1, buka detail surat
  await loginAs(page, "staf1");
  await page.goto(`/surat/${suratID}`);

  const card = page.getByTestId("disposisi-card");
  await expect(card.getByText("Test status update flow")).toBeVisible({ timeout: 5000 });
  await expect(card.getByText("Pending", { exact: true })).toBeVisible();

  // Klik Mulai
  await page.locator('[data-testid^="disposisi-start-"]').click();
  await expect(page.getByText("Status diubah ke Sedang dikerjakan")).toBeVisible({ timeout: 3000 });
  await expect(card.getByText("Sedang dikerjakan", { exact: true })).toBeVisible();

  // Klik Selesai
  await page.locator('[data-testid^="disposisi-done-"]').click();
  await expect(page.getByText("Status diubah ke Selesai")).toBeVisible({ timeout: 3000 });
  await expect(card.getByText("Selesai", { exact: true })).toBeVisible();

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${camatToken}` },
  });
});

// =============================================================================
// Display existing disposisi dari seed
// =============================================================================

test("disposisi card menampilkan disposisi seed yang sudah ada", async ({ page }) => {
  await loginAs(page, "camat");

  // ID surat dari seed yang punya disposisi: 0007-...0008 (Permohonan Surket Tazkia)
  await page.goto("/surat/00000000-0000-0000-0007-000000000008");

  const card = page.getByTestId("disposisi-card");
  await expect(card).toBeVisible();
  // Disposisi seed: instruksi mengandung "surat keterangan domisili" (scope ke card)
  await expect(card.getByText(/surat keterangan domisili/i)).toBeVisible();
  // Status seed: done
  await expect(card.getByText("Selesai", { exact: true })).toBeVisible();
});
