// Tombol sync global: trigger paralel notifikasi + opQueue drain + Dexie snapshot.
//
// Polling 30s tetap jalan sebagai fallback — manual sync adalah explicit
// trigger untuk refresh segera.

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

test("FULL UI: tombol sync visible di topbar saat login", async ({ page }) => {
  await loginAs(page, "staf1");
  await expect(page.getByTestId("global-sync-btn")).toBeVisible();
});

test("FULL UI: klik tombol sync → toast 'Sinkronisasi selesai'", async ({ page }) => {
  await loginAs(page, "staf1");
  await page.getByTestId("global-sync-btn").click();
  await expect(page.getByText("Sinkronisasi selesai")).toBeVisible({ timeout: 5000 });
});

test("FULL UI: sync saat offline → warning 'Tidak bisa sync — sedang offline'", async ({ page, context }) => {
  await loginAs(page, "staf1");

  // Wait initial sync settle
  await page.waitForFunction(async () => {
    return new Promise((resolve) => {
      const req = indexedDB.open("surat-kec-cache");
      req.onsuccess = () => {
        const db = req.result;
        const tx = db.transaction("surat", "readonly");
        const countReq = tx.objectStore("surat").count();
        countReq.onsuccess = () => resolve(countReq.result > 0);
      };
      req.onerror = () => resolve(false);
    });
  }, undefined, { timeout: 10_000 });

  await context.setOffline(true);
  await page.evaluate(() => window.dispatchEvent(new Event("offline")));

  await page.getByTestId("global-sync-btn").click();
  await expect(page.getByText(/Tidak bisa sync.*sedang offline/i)).toBeVisible({ timeout: 3000 });

  await context.setOffline(false);
});

test("FULL UI: sync trigger notifikasi pull baru — assign disposisi via API → klik sync → notif badge bertambah tanpa nunggu polling", async ({ page, request }) => {
  await loginAs(page, "staf1");

  // Capture initial badge state (mungkin sudah ada notif dari sebelumnya)
  const initialBadgeText = await page.getByTestId("notif-badge").textContent().catch(() => "");
  const initialCount = parseInt(initialBadgeText?.match(/\d+/)?.[0] ?? "0", 10);

  // Camat (via API) assign disposisi baru ke staf1
  await page.evaluate(() => localStorage.clear());
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
      nomor_surat: `SYNC-NOTIF/${Date.now()}`,
      perihal: "Trigger notif untuk sync test",
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
      instruksi: "Test manual sync trigger",
    },
  });

  // Login ulang sebagai staf1, klik sync immediately (jangan nunggu polling)
  await loginAs(page, "staf1");
  await page.getByTestId("global-sync-btn").click();
  await expect(page.getByText("Sinkronisasi selesai")).toBeVisible({ timeout: 5000 });

  // Badge harus naik (initialCount + ≥1)
  const newBadge = page.getByTestId("notif-badge");
  await expect(newBadge).toBeVisible();
  const newText = await newBadge.textContent();
  const newCount = parseInt(newText?.match(/\d+/)?.[0] ?? "0", 10);
  expect(newCount).toBeGreaterThan(initialCount);

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${camatToken}` },
  });
});
