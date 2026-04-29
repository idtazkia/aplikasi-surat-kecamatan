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

  // Tetap di /surat/baru, message warning tampil
  await expect(page).toHaveURL(/\/surat\/baru$/);
  await expect(page.getByText(/wajib/i).first()).toBeVisible({ timeout: 3000 });
});

test("create surat masuk via API helper, verify di list", async ({ page, request }) => {
  await loginAs(page, "staf1");

  // Get token dari localStorage untuk API call
  const auth = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  const token = JSON.parse(auth!).accessToken;

  // Search instansi pertama
  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiBody = await instansiResp.json();
  expect(instansiBody.items.length).toBeGreaterThan(0);
  const instansiID = instansiBody.items[0].id;

  // Create surat via API
  const createResp = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: "API-TEST/E2E/2026",
      perihal: "Test E2E via API",
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  expect(createResp.status()).toBe(201);
  const created = await createResp.json();

  // Verify di list page (search by perihal)
  await page.goto("/surat");
  await page.getByPlaceholder("Kata kunci").fill("Test E2E via API");
  await page.getByRole("button", { name: "Terapkan" }).click();
  await page.waitForTimeout(500);

  const allText = await page.locator("table tbody").textContent();
  expect(allText).toContain("API-TEST/E2E/2026");

  // Cleanup: delete via API
  await request.delete(`/api/surat/${created.id}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

test("edit surat existing via UI → simpan → lihat perubahan", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const auth = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  const token = JSON.parse(auth!).accessToken;

  // Create surat untuk di-edit (avoid mengganggu data lain)
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

  // Navigate ke edit page
  await page.goto(`/surat/${created.id}/edit`);
  await expect(page).toHaveURL(/\/edit$/);

  // Tunggu form ter-populate dengan data existing (loadForEdit selesai)
  const perihalArea = page.locator("textarea").first();
  await expect(perihalArea).toHaveValue(/Original perihal/, { timeout: 10000 });

  // Edit perihal
  await perihalArea.fill("Edited perihal via E2E");

  // Save
  await Promise.all([
    page.waitForURL(new RegExp(`/surat/${created.id}$`), { timeout: 10000 }),
    page.getByRole("button", { name: "Simpan Perubahan" }).click(),
  ]);

  // Verify perihal updated di detail page
  await expect(page.getByText("Edited perihal via E2E")).toBeVisible({ timeout: 10000 });

  // Cleanup
  await request.delete(`/api/surat/${created.id}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

test("delete surat via UI → kembali ke list, tidak muncul lagi", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const auth = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  const token = JSON.parse(auth!).accessToken;

  // Create surat untuk di-delete
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

  // Navigate ke detail
  await page.goto(`/surat/${created.id}`);
  await expect(page.getByText("DELETE-ME/01/2026")).toBeVisible();

  // Click Hapus → konfirmasi popconfirm
  await page.getByRole("button", { name: "Hapus" }).click();
  // NPopconfirm action button — ambil yang positive (OK/Confirm)
  await page.locator(".n-popconfirm__action button").last().click();

  // Redirect ke list
  await page.waitForURL(/\/surat$/, { timeout: 5000 });

  // Search nomor surat → tidak muncul. Empty result tampilkan NEmpty komponen.
  await page.getByPlaceholder("Kata kunci").fill("DELETE-ME");
  await page.getByRole("button", { name: "Terapkan" }).click();
  await page.waitForTimeout(800);

  // Either NEmpty visible OR table tidak mengandung DELETE-ME
  const bodyText = await page.locator(".n-layout-content").textContent();
  expect(bodyText ?? "").not.toContain("DELETE-ME/01/2026");
});

test("conflict: nomor surat keluar duplikat → 409", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const auth = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  const token = JSON.parse(auth!).accessToken;

  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;

  const payload = {
    jenis: "keluar" as const,
    nomor_surat: "CONFLICT/01/2026",
    perihal: "First surat keluar",
    tanggal_surat: "2026-04-15",
    instansi_id: instansiID,
    access_level: "public" as const,
  };

  // First create
  const first = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: payload,
  });
  expect(first.status()).toBe(201);
  const firstID = (await first.json()).id;

  // Second with same nomor_surat → conflict
  const second = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: payload,
  });
  expect(second.status()).toBe(409);

  // Cleanup
  await request.delete(`/api/surat/${firstID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

test("staf tidak bisa restore (admin only)", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const auth = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  const token = JSON.parse(auth!).accessToken;

  // Pakai dummy ID — kalau forbidden duluan sebelum check exist, OK
  const resp = await request.post("/api/surat/00000000-0000-0000-0099-000000000099/restore", {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.status()).toBe(403);
});
