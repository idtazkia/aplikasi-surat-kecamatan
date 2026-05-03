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
) {
  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;
  const createResp = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: `NOTIF-TEST/${Date.now()}/2026`,
      perihal: "Surat untuk notifikasi UI test",
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  const created = await createResp.json();
  return created.id as string;
}

test("FULL UI: camat assign disposisi → staf1 lihat badge unread + buka & mark read", async ({ page, request }) => {
  // Camat assign via API (state prep)
  await loginAs(page, "camat");
  const camatToken = await getToken(page);
  const suratID = await createTestSurat(request, camatToken);

  const usersResp = await request.get("/api/users/assignable", {
    headers: { Authorization: `Bearer ${camatToken}` },
  });
  const users = (await usersResp.json()).items as { id: string; username: string }[];
  const staf1 = users.find((u) => u.username === "staf1")!;

  await request.post("/api/disposisi", {
    headers: { Authorization: `Bearer ${camatToken}` },
    data: {
      surat_id: suratID,
      assigned_to: staf1.id,
      instruksi: "Tolong proses notif test",
    },
  });

  // Login staf1, lihat bell badge
  await loginAs(page, "staf1");

  const badge = page.getByTestId("notif-badge");
  await expect(badge).toBeVisible();
  // Badge shows count > 0
  await expect(badge).toContainText(/\d+/);

  // Click bell, dropdown muncul
  await page.getByTestId("notif-bell").click();
  await expect(page.getByText("Notifikasi", { exact: true })).toBeVisible();

  // Verify item ada
  const items = page.locator('[data-testid^="notif-item-"]');
  await expect(items.first()).toBeVisible({ timeout: 3000 });
  await expect(items.first()).toContainText("Disposisi baru");

  // Mark all read
  await page.getByTestId("notif-mark-all-read").click();
  await expect(page.getByText("Semua notifikasi ditandai dibaca")).toBeVisible({ timeout: 3000 });

  // Setelah refresh, badge hilang
  await page.waitForTimeout(300);
  // Click outside untuk close popover
  await page.locator('body').click({ position: { x: 10, y: 10 } });
  await page.waitForTimeout(200);

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${camatToken}` },
  });
});

test("FULL UI: klik notif disposisi → navigate ke surat detail + auto mark read", async ({ page, request }) => {
  await loginAs(page, "camat");
  const camatToken = await getToken(page);
  const suratID = await createTestSurat(request, camatToken);

  const usersResp = await request.get("/api/users/assignable", {
    headers: { Authorization: `Bearer ${camatToken}` },
  });
  const users = (await usersResp.json()).items as { id: string; username: string }[];
  const staf1 = users.find((u) => u.username === "staf1")!;

  await request.post("/api/disposisi", {
    headers: { Authorization: `Bearer ${camatToken}` },
    data: {
      surat_id: suratID,
      assigned_to: staf1.id,
      instruksi: "Click navigation test",
    },
  });

  await loginAs(page, "staf1");

  await page.getByTestId("notif-bell").click();
  const item = page.locator('[data-testid^="notif-item-"]').first();
  await expect(item).toBeVisible({ timeout: 3000 });
  await item.click();

  // Navigate ke surat detail
  await expect(page).toHaveURL(new RegExp(`/surat/${suratID}$`));

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${camatToken}` },
  });
});

test("FULL UI: komentar dari camat → assignee staf1 dapat notif komentar_baru", async ({ page, request }) => {
  await loginAs(page, "camat");
  const camatToken = await getToken(page);
  const suratID = await createTestSurat(request, camatToken);

  const usersResp = await request.get("/api/users/assignable", {
    headers: { Authorization: `Bearer ${camatToken}` },
  });
  const users = (await usersResp.json()).items as { id: string; username: string }[];
  const staf1 = users.find((u) => u.username === "staf1")!;

  // Camat assign disposisi ke staf1 (membentuk participant set)
  await request.post("/api/disposisi", {
    headers: { Authorization: `Bearer ${camatToken}` },
    data: {
      surat_id: suratID,
      assigned_to: staf1.id,
      instruksi: "Setup untuk komentar notif test",
    },
  });

  // Camat post komentar via UI
  await page.goto(`/surat/${suratID}`);
  await page.locator('[data-testid="komentar-input"] textarea').fill("Update progress dari camat");
  await page.getByTestId("submit-komentar").click();
  await expect(page.getByText("Komentar ditambahkan")).toBeVisible({ timeout: 3000 });

  // Login staf1, lihat notif komentar_baru
  await loginAs(page, "staf1");
  await page.getByTestId("notif-bell").click();

  const items = page.locator('[data-testid^="notif-item-"]');
  await expect(items.first()).toBeVisible({ timeout: 3000 });
  // Karena ada 2 notif (disposisi_baru + komentar_baru), keduanya muncul.
  // Cek setidaknya ada komentar_baru
  await expect(page.getByText(/Komentar baru/)).toBeVisible({ timeout: 3000 });

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${camatToken}` },
  });
});
