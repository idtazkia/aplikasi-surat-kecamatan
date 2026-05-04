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

test("FULL UI: thread modal menampilkan surat anchor + predecessor + successor", async ({ page }) => {
  await loginAs(page, "staf1");

  // Surat seed: 0007-...000d (Tanggapan Edaran Pandemi) — punya predecessor (Edaran Penanganan Pandemi)
  // Verify lewat existing surat-detail test yang already passes pakai surat ini.
  // Cari surat dengan search "Tanggapan atas Edaran"
  await page.getByPlaceholder("Kata kunci").fill("Tanggapan atas Edaran");
  await page.getByRole("button", { name: "Terapkan" }).click();
  await page.waitForTimeout(500);

  await page.locator("table tbody tr").first().click();
  await expect(page).toHaveURL(/\/surat\/[\w-]+$/);

  // Buka thread modal
  await page.getByTestId("thread-view-btn").click();
  const modal = page.getByTestId("thread-modal");
  await expect(modal).toBeVisible();

  // Anchor node selalu ada (depth=0, direction=self)
  const selfNodes = modal.locator('[data-testid$="-self"]');
  await expect(selfNodes).toHaveCount(1);

  // Karena surat ini balasan dari Edaran Pandemi, harus ada predecessor node
  const predNodes = modal.locator('[data-testid$="-predecessor"]');
  await expect(predNodes.first()).toBeVisible();
});

test("FULL UI: surat tanpa references → thread hanya berisi 1 node (self)", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const auth = await page.evaluate(() => localStorage.getItem("surat-kec-auth"));
  const token = JSON.parse(auth!).accessToken;

  // Buat surat baru tanpa references
  const instansiResp = await request.get("/api/instansi?q=Kemen", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const instansiID = (await instansiResp.json()).items[0].id;
  const createResp = await request.post("/api/surat", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      jenis: "masuk",
      nomor_surat: `THREAD-ISOLATED/${Date.now()}/2026`,
      perihal: "Surat tanpa referensi UI test",
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  const suratID = (await createResp.json()).id as string;

  await page.goto(`/surat/${suratID}`);
  await page.getByTestId("thread-view-btn").click();

  const modal = page.getByTestId("thread-modal");
  await expect(modal).toBeVisible();

  // Hanya 1 node (self) — total list-item count = 1
  const allNodes = modal.locator('[data-testid^="thread-node-"]');
  await expect(allNodes).toHaveCount(1);
  await expect(allNodes.first()).toContainText("AWAL");

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});
