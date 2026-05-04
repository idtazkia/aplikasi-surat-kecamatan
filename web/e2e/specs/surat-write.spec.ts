import { test, expect, Page, Locator } from "@playwright/test";

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

// Calendar panel click — sama dengan helper di surat-upload.spec.ts.
async function pickDate(page: Page, pickerInput: Locator, target: Date) {
  await pickerInput.click();
  await page.waitForSelector(".n-date-panel", { timeout: 3000 });

  const monthHeader = page.locator(".n-date-panel-month__month-year").first();
  for (let i = 0; i < 60; i++) {
    const headerText = (await monthHeader.textContent())?.trim() ?? "";
    const m = headerText.match(/(\d{1,2})[^\d]+(\d{4})/);
    if (m) {
      const currentMonth = parseInt(m[1], 10) - 1;
      const currentYear = parseInt(m[2], 10);
      if (currentMonth === target.getMonth() && currentYear === target.getFullYear()) {
        break;
      }
      const goNext =
        new Date(target.getFullYear(), target.getMonth()).getTime() >
        new Date(currentYear, currentMonth).getTime();
      await page.locator(
        goNext ? ".n-date-panel-month__next" : ".n-date-panel-month__prev",
      ).first().click();
    } else {
      break;
    }
    await page.waitForTimeout(80);
  }

  const dayStr = String(target.getDate());
  await page
    .locator(".n-date-panel-date:not(.n-date-panel-date--excluded)")
    .filter({ hasText: new RegExp(`^${dayStr}$`) })
    .first()
    .click();
  await page.waitForTimeout(200);
}

async function pickInstansi(page: Page, query: string) {
  const select = page.locator('[data-testid="instansi-field"] .n-base-selection');
  await select.click();
  await page.waitForTimeout(150);
  // Type via the actual filterable input — lebih reliable dari page.keyboard
  // karena tidak ada race dengan focus state
  const searchInput = page.locator('[data-testid="instansi-field"] input').first();
  await searchInput.fill(query);
  // Wait debounce 200ms + API + dropdown render
  await page.waitForTimeout(900);
  await page.locator(".n-base-select-option:visible").first().click();
}

// =============================================================================
// NAVIGATION + VALIDATION
// =============================================================================

test("navigasi ke /surat/baru menampilkan form create", async ({ page }) => {
  await loginAs(page, "staf1");
  await page.goto("/surat/baru");

  await expect(page.getByText("Surat Baru", { exact: true })).toBeVisible();
  await expect(page.getByPlaceholder("045/123/IV/2026")).toBeVisible();
  await expect(page.getByPlaceholder("Subject surat")).toBeVisible();
  await expect(page.getByRole("button", { name: "Buat Surat" })).toBeVisible();
});

test("validasi: submit kosong → warning + tetap di form", async ({ page }) => {
  await loginAs(page, "staf1");
  await page.goto("/surat/baru");

  await page.getByRole("button", { name: "Buat Surat" }).click();

  await expect(page).toHaveURL(/\/surat\/baru$/);
  await expect(page.getByText(/wajib/i).first()).toBeVisible({ timeout: 3000 });
});

// =============================================================================
// EDIT (UI flow, API setup OK karena state preparation, bukan SUT)
// =============================================================================

test("edit surat existing via UI → simpan → lihat perubahan", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const auth = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  const token = JSON.parse(auth!).accessToken;

  // Setup state via API (test prep, bukan tested behavior)
  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;
  const createResp = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: "EDIT-TEST/01/2026",
      perihal: "Original perihal untuk diedit",
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  const created = await createResp.json();

  // ===== UI test: edit perihal lewat form =====
  await page.goto(`/surat/${created.id}/edit`);
  await expect(page).toHaveURL(/\/edit$/);

  const perihalArea = page.locator("textarea").first();
  await expect(perihalArea).toHaveValue(/Original perihal/, { timeout: 10000 });

  await perihalArea.fill("Edited perihal via E2E");

  await Promise.all([
    page.waitForURL(new RegExp(`/surat/${created.id}$`), { timeout: 10000 }),
    page.getByRole("button", { name: "Simpan Perubahan" }).click(),
  ]);

  await expect(page.getByText("Edited perihal via E2E")).toBeVisible({ timeout: 10000 });

  // Cleanup state
  await request.delete(`/api/surat/${created.id}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

// =============================================================================
// DELETE (UI flow, API setup OK)
// =============================================================================

test("delete surat via UI → kembali ke list, tidak muncul lagi", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const auth = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  const token = JSON.parse(auth!).accessToken;

  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;
  const createResp = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: "DELETE-ME/01/2026",
      perihal: "Surat untuk dihapus E2E",
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  const created = await createResp.json();

  // ===== UI test: delete via popconfirm + verify hilang dari list =====
  await page.goto(`/surat/${created.id}`);
  await expect(page.getByText("DELETE-ME/01/2026")).toBeVisible();

  await page.getByRole("button", { name: "Hapus" }).click();
  await page.locator(".n-popconfirm__action button").last().click();

  await page.waitForURL(/\/surat$/, { timeout: 5000 });

  await page.getByPlaceholder("Kata kunci").fill("DELETE-ME");
  await page.getByRole("button", { name: "Terapkan" }).click();
  await page.waitForTimeout(800);

  const bodyText = await page.locator(".n-layout-content").textContent();
  expect(bodyText ?? "").not.toContain("DELETE-ME/01/2026");
});

// =============================================================================
// CONFLICT 409 (UI flow — fill form dengan nomor yang sudah ada di seed)
// =============================================================================

test("FULL UI: surat keluar dengan nomor sama → toast 'Nomor surat sudah dipakai'", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const auth = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  const token = JSON.parse(auth!).accessToken;

  // Setup: pre-create surat keluar dengan nomor X (state prep, bukan SUT)
  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;
  const duplicateNomor = "CONFLICT-UI/01/2026";

  const first = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "keluar",
      nomor_surat: duplicateNomor,
      perihal: "First surat keluar (pre-created)",
      tanggal_surat: "2026-04-15",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  expect(first.status()).toBe(201);
  const firstID = (await first.json()).id;

  // ===== UI test: coba create surat keluar duplikat via form, expect error toast =====
  await page.goto("/surat/baru");
  await expect(page.getByPlaceholder("045/123/IV/2026")).toBeVisible();
  await page.waitForLoadState("networkidle");

  // Ganti jenis ke "keluar"
  const jenisDropdown = page.locator('.n-form-item').filter({ hasText: /^Jenis Surat/ }).locator('.n-base-selection').first();
  await jenisDropdown.click();
  await page.locator(".n-base-select-option").filter({ hasText: "Surat Keluar" }).click();

  // Fill required fields dengan duplicateNomor
  await page.getByPlaceholder("045/123/IV/2026").fill(duplicateNomor);
  await page.getByPlaceholder("Subject surat").fill("Second attempt with same nomor");

  const dateInputs = page.locator(".n-date-picker input");
  await pickDate(page, dateInputs.nth(0), new Date(2026, 3, 20));

  await pickInstansi(page, "Kemendagri");

  // Submit — expect 409 → toast "Nomor surat sudah dipakai"
  await page.getByRole("button", { name: "Buat Surat" }).click();

  await expect(page.getByText("Nomor surat sudah dipakai")).toBeVisible({ timeout: 5000 });
  // Tetap di /surat/baru (tidak navigate)
  await expect(page).toHaveURL(/\/surat\/baru$/);

  // Cleanup
  await request.delete(`/api/surat/${firstID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});
