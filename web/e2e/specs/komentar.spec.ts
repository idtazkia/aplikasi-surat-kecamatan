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
      nomor_surat: `KOMENTAR-TEST/${Date.now()}/2026`,
      perihal: "Surat untuk komentar UI test",
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  const created = await createResp.json();
  return created.id as string;
}

test("FULL UI: dua user post komentar bergantian → terlihat berurutan oleh keduanya", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const stafToken = await getToken(page);
  const suratID = await createTestSurat(request, stafToken);

  await page.goto(`/surat/${suratID}`);

  await expect(page.getByTestId("komentar-card")).toBeVisible();
  await expect(page.getByText("Belum ada komentar")).toBeVisible();

  // Staf1 post komentar pertama
  await page.locator('[data-testid="komentar-input"] textarea').fill("Komentar pertama dari staf1");
  await page.getByTestId("submit-komentar").click();
  await expect(page.getByText("Komentar ditambahkan")).toBeVisible({ timeout: 3000 });

  const card = page.getByTestId("komentar-card");
  await expect(card.getByText("Komentar pertama dari staf1")).toBeVisible();
  // Input cleared after submit
  await expect(page.locator('[data-testid="komentar-input"] textarea')).toHaveValue("");

  // Switch to camat untuk balas
  await loginAs(page, "camat");
  await page.goto(`/surat/${suratID}`);

  await expect(card.getByText("Komentar pertama dari staf1")).toBeVisible();

  await page.locator('[data-testid="komentar-input"] textarea').fill("Tanggapan dari camat");
  await page.getByTestId("submit-komentar").click();
  await expect(page.getByText("Komentar ditambahkan")).toBeVisible({ timeout: 3000 });

  // Both visible, ordered by created_at ASC
  const items = card.locator('[data-testid^="komentar-item-"]');
  await expect(items).toHaveCount(2);
  await expect(items.nth(0)).toContainText("Komentar pertama dari staf1");
  await expect(items.nth(1)).toContainText("Tanggapan dari camat");

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${stafToken}` },
  });
});

test("validasi: submit komentar kosong tidak men-trigger request (button disabled)", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);
  const suratID = await createTestSurat(request, token);

  await page.goto(`/surat/${suratID}`);

  const submitBtn = page.getByTestId("submit-komentar");
  await expect(submitBtn).toBeDisabled();

  // Type space — masih trimmed jadi disabled
  await page.locator('[data-testid="komentar-input"] textarea').fill("   ");
  await expect(submitBtn).toBeDisabled();

  // Real content enables
  await page.locator('[data-testid="komentar-input"] textarea').fill("Konten valid");
  await expect(submitBtn).not.toBeDisabled();

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});
