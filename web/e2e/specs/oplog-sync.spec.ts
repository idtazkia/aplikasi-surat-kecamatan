// Fase 4 E2E: operation log sync API + offline write flow.
//
// Scope: backend POST /api/sync/operations idempotency + LWW + UI offline
// edit flow yang enqueue ke Dexie, badge counter, drain on reconnect.

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
      nomor_surat: `OPLOG-TEST/${Date.now()}/2026`,
      perihal: "Surat oplog test awal",
      tanggal_surat: "2026-04-15",
      tanggal_terima: "2026-04-16",
      instansi_id: instansiID,
      access_level: "public",
    },
  });
  return (await createResp.json()).id as string;
}

// =============================================================================
// API kontrak: idempotency + LWW
// =============================================================================

test("POST /api/sync/operations applied → status applied", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);
  const suratID = await createTestSurat(request, token);

  const resp = await request.post("/api/sync/operations", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      operations: [
        {
          client_op_id: "00000000-0000-7000-8000-000000000001",
          entity_type: "surat",
          entity_id: suratID,
          action: "update",
          field_changes: { perihal: "Edited via oplog" },
          client_timestamp: new Date().toISOString(),
        },
      ],
    },
  });
  expect(resp.status()).toBe(200);
  const body = await resp.json();
  expect(body.results[0].status).toBe("applied");

  // Verify surat has new perihal
  const detailResp = await request.get(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect((await detailResp.json()).perihal).toBe("Edited via oplog");

  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

test("re-submit op dengan same client_op_id → status duplicate (idempotent)", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);
  const suratID = await createTestSurat(request, token);

  const opID = "00000000-0000-7000-8000-000000000002";
  const payload = {
    operations: [
      {
        client_op_id: opID,
        entity_type: "surat",
        entity_id: suratID,
        action: "update",
        field_changes: { perihal: "Idempotent test" },
        client_timestamp: new Date().toISOString(),
      },
    ],
  };

  const r1 = await request.post("/api/sync/operations", {
    headers: { Authorization: `Bearer ${token}` },
    data: payload,
  });
  expect((await r1.json()).results[0].status).toBe("applied");

  // Re-submit
  const r2 = await request.post("/api/sync/operations", {
    headers: { Authorization: `Bearer ${token}` },
    data: payload,
  });
  expect((await r2.json()).results[0].status).toBe("duplicate");

  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

test("op dengan client_timestamp lebih lama dari server.updated_at → rejected stale (LWW)", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);
  const suratID = await createTestSurat(request, token);

  // Edit surat sekarang via API (set updated_at = now)
  await request.patch(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { perihal: "Edited online dulu" },
  });

  // Submit op dengan client_timestamp 1 jam lalu — harus stale
  const oneHourAgo = new Date(Date.now() - 3600_000).toISOString();
  const resp = await request.post("/api/sync/operations", {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      operations: [
        {
          client_op_id: "00000000-0000-7000-8000-000000000003",
          entity_type: "surat",
          entity_id: suratID,
          action: "update",
          field_changes: { perihal: "Stale offline edit" },
          client_timestamp: oneHourAgo,
        },
      ],
    },
  });
  const body = await resp.json();
  expect(body.results[0].status).toBe("rejected");
  expect(body.results[0].reason).toMatch(/stale/i);

  // Verify perihal tetap "Edited online dulu"
  const detailResp = await request.get(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  expect((await detailResp.json()).perihal).toBe("Edited online dulu");

  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});

test("validation: operations kosong → 400", async ({ page, request }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);

  const resp = await request.post("/api/sync/operations", {
    headers: { Authorization: `Bearer ${token}` },
    data: { operations: [] },
  });
  expect(resp.status()).toBe(400);
});

// =============================================================================
// FULL UI: edit surat saat offline → enqueue → badge counter → drain on reconnect
// =============================================================================

test("FULL UI: edit surat saat offline → badge pending bertambah → online → tersinkron", async ({ page, request, context }) => {
  await loginAs(page, "staf1");
  const token = await getToken(page);
  const suratID = await createTestSurat(request, token);

  // Wait initial sync (Dexie populated)
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

  // Navigate to edit page DULU sebelum offline — page sudah ter-load
  await page.goto(`/surat/${suratID}/edit`);
  const perihalArea = page.locator("textarea").first();
  await expect(perihalArea).toHaveValue(/oplog test awal/, { timeout: 5000 });

  // Sekarang go offline
  await context.setOffline(true);
  await page.evaluate(() => window.dispatchEvent(new Event("offline")));

  await perihalArea.fill("Edited offline UI");

  // Submit — akan enqueue ke opQueue (offline → tidak panggil API)
  await page.getByRole("button", { name: "Simpan Perubahan" }).click();

  // Toast warning offline
  await expect(page.getByText(/Tersimpan offline/i)).toBeVisible({ timeout: 5000 });

  // Reconnect dulu sebelum navigate (offline navigate gagal di dev tanpa SW)
  await context.setOffline(false);
  await page.evaluate(() => window.dispatchEvent(new Event("online")));
  await page.goto(`/surat`);

  // Setelah back to /surat, main.ts re-init startDrainer() yang trigger
  // immediate drain (since online). Wait drain complete by polling API.
  // Max 5s wait — drain harus < 1s untuk batch 1 op.
  await expect(async () => {
    const detailResp = await request.get(`/api/surat/${suratID}`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect((await detailResp.json()).perihal).toBe("Edited offline UI");
  }).toPass({ timeout: 5000 });

  // Setelah drain, badge harus tidak visible (count = 0, badge hidden via :show)
  await expect(page.getByTestId("pending-sync-badge")).not.toContainText(/\d/, { timeout: 5000 });

  // Cleanup
  await request.delete(`/api/surat/${suratID}`, {
    headers: { Authorization: `Bearer ${token}` },
  });
});
