// PWA offline read-only mode: verify sync endpoint + Dexie fallback path.
// Service worker behavior (SW install, cache strategies) sulit di-test via
// Playwright dev server karena SW di-disable di dev mode by default —
// scope test ini: API kontrak + IndexedDB integrasi.

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
// SYNC SNAPSHOT API kontrak
// =============================================================================

test("GET /api/sync/snapshot (full) → return semua active rows + watermark", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  const resp = await request.get("/api/sync/snapshot", {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.status()).toBe(200);
  const body = await resp.json();

  // Watermark RFC3339
  expect(body.watermark).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/);

  // Surat seed: minimal beberapa rows non-secret
  expect(body.surat.length).toBeGreaterThan(0);

  // Lookups full list
  expect(body.klasifikasi.length).toBeGreaterThan(0);
  expect(body.sifat.length).toBeGreaterThan(0);
  expect(body.instansi.length).toBeGreaterThan(0);

  // Tombstones empty saat full sync
  expect(body.surat_deleted_ids).toEqual([]);

  // Staf TIDAK lihat surat secret
  for (const s of body.surat) {
    expect(s.access_level).not.toBe("secret");
  }
});

test("GET /api/sync/snapshot?since=now → empty surat (tidak ada updated)", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  const future = new Date(Date.now() + 60_000).toISOString();
  const resp = await request.get(`/api/sync/snapshot?since=${encodeURIComponent(future)}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.status()).toBe(200);
  const body = await resp.json();
  expect(body.surat).toEqual([]);
  expect(body.surat_deleted_ids).toEqual([]);
});

test("camat sync snapshot lihat surat secret juga", async ({ page, request }) => {
  await loginAs(page, "camat");
  const token = await getToken(page);

  const resp = await request.get("/api/sync/snapshot", {
    headers: { Authorization: `Bearer ${token}` },
  });
  const body = await resp.json();

  const hasSecret = body.surat.some((s: { access_level: string }) => s.access_level === "secret");
  expect(hasSecret).toBe(true);
});

test("snapshot since invalid format → 400", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  const resp = await request.get("/api/sync/snapshot?since=not-a-date", {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect(resp.status()).toBe(400);
});

// =============================================================================
// FULL UI: sync triggered on login → IndexedDB populated → offline fallback
// =============================================================================

test("FULL UI: login trigger sync → Dexie surat store ter-populate", async ({ page }) => {
  await loginAs(page, "staf1");

  // Tunggu sync (auto-trigger di main.ts setelah login). Sync runs async,
  // wait sampai data muncul di IndexedDB.
  await page.waitForFunction(async () => {
    const dbs = await indexedDB.databases?.();
    return dbs?.some((d) => d.name === "surat-kec-cache");
  }, undefined, { timeout: 5000 });

  // Verify surat tersimpan
  const cachedCount = await page.evaluate(async () => {
    const req = indexedDB.open("surat-kec-cache");
    return new Promise<number>((resolve, reject) => {
      req.onsuccess = () => {
        const db = req.result;
        const tx = db.transaction("surat", "readonly");
        const store = tx.objectStore("surat");
        const countReq = store.count();
        countReq.onsuccess = () => resolve(countReq.result);
        countReq.onerror = () => reject(countReq.error);
      };
      req.onerror = () => reject(req.error);
    });
  });
  expect(cachedCount).toBeGreaterThan(0);
});

test("FULL UI: simulasi offline → banner muncul + apply filter fallback ke Dexie cache", async ({ page, context }) => {
  await loginAs(page, "staf1");

  // Wait sync complete (cek IndexedDB punya data)
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

  // Toggle offline mode (network only — page sudah loaded)
  await context.setOffline(true);
  await page.evaluate(() => window.dispatchEvent(new Event("offline")));

  // Banner muncul
  await expect(page.getByTestId("offline-banner")).toBeVisible({ timeout: 3000 });
  await expect(page.getByTestId("offline-banner")).toContainText("Anda offline");

  // Trigger fetch ulang via apply filter button → fetch /api/surat akan
  // gagal (offline), fallback ke Dexie cache via tryOfflineFallback.
  await page.getByRole("button", { name: "Terapkan" }).click();

  // Toast warning "Memuat dari cache lokal" muncul
  await expect(page.getByText(/Memuat dari cache lokal/)).toBeVisible({ timeout: 5000 });

  // Tabel render dari cache
  const rows = page.locator("table tbody tr");
  await expect(rows.first()).toBeVisible({ timeout: 3000 });

  await context.setOffline(false);
});
