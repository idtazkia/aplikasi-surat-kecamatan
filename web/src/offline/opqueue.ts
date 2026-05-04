import { apiClient } from "@/api/client";
import { db, type PendingOp } from "./db";
import { uuidv7 } from "./uuid7";

// Operation queue untuk offline write. Setiap mutation user lewat enqueue() —
// kalau online, langsung POST /api/sync/operations; kalau gagal, tetap di
// Dexie ops store untuk retry kemudian.
//
// concept:event-sourcing-operation-log (master di backend operation-log table)

interface ApplyOperationsResponse {
  results: Array<{
    client_op_id: string;
    status: "applied" | "duplicate" | "rejected";
    reason?: string;
  }>;
}

export interface EnqueueInput {
  entity_type: PendingOp["entity_type"];
  entity_id: string;
  action: PendingOp["action"];
  field_changes: Record<string, unknown>;
}

// EnqueueResult mengkomunikasikan apakah op sudah applied immediately atau
// masih di-queue (caller bisa pilih: optimistic update vs wait).
export interface EnqueueResult {
  client_op_id: string;
  applied: boolean;          // true = server confirm, false = queued
  reason?: string;           // populated kalau rejected immediately
}

// enqueue: simpan op ke Dexie + try sync immediate (kalau online).
// Idempotent — boleh dipanggil ulang untuk op yang sama akan duplikat
// di server (dijaga via client_op_id PK constraint).
export async function enqueue(input: EnqueueInput): Promise<EnqueueResult> {
  const op: PendingOp = {
    client_op_id: uuidv7(),
    entity_type: input.entity_type,
    entity_id: input.entity_id,
    action: input.action,
    field_changes: input.field_changes,
    client_timestamp: new Date().toISOString(),
    retry_count: 0,
  };

  await db.ops.add(op);

  // Try sync immediate kalau online. Kalau gagal, op sudah ter-persist —
  // drainer akan retry kemudian.
  if (typeof navigator !== "undefined" && navigator.onLine) {
    const result = await tryApplyOne(op);
    return result;
  }
  return { client_op_id: op.client_op_id, applied: false };
}

// tryApplyOne kirim 1 op ke server. Update Dexie row sesuai hasil.
// Return result untuk konsumsi UI.
async function tryApplyOne(op: PendingOp): Promise<EnqueueResult> {
  try {
    const resp = await apiClient.post<ApplyOperationsResponse>(
      "/api/sync/operations",
      {
        operations: [
          {
            client_op_id: op.client_op_id,
            entity_type: op.entity_type,
            entity_id: op.entity_id,
            action: op.action,
            field_changes: op.field_changes,
            client_timestamp: op.client_timestamp,
          },
        ],
      },
    );

    const r = resp.results[0];
    if (r.status === "applied" || r.status === "duplicate") {
      await db.ops.update(op.client_op_id, {
        synced_at: new Date().toISOString(),
        error: undefined,
      });
      return { client_op_id: op.client_op_id, applied: true };
    }
    // Rejected — simpan reason, jangan retry (lostnya stale LWW atau
    // validation error → user harus aware).
    await db.ops.update(op.client_op_id, {
      error: r.reason ?? "rejected",
    });
    return { client_op_id: op.client_op_id, applied: false, reason: r.reason };
  } catch (e) {
    // Network error — schedule retry via next_retry_at.
    const errStr = e instanceof Error ? e.message : String(e);
    const nextRetry = backoffDelay(op.retry_count);
    await db.ops.update(op.client_op_id, {
      retry_count: op.retry_count + 1,
      next_retry_at: new Date(Date.now() + nextRetry).toISOString(),
      error: errStr,
    });
    return { client_op_id: op.client_op_id, applied: false, reason: errStr };
  }
}

// Exponential backoff: 1s, 2s, 4s, 8s, … cap 60s.
function backoffDelay(retryCount: number): number {
  const seconds = Math.min(2 ** retryCount, 60);
  return seconds * 1000;
}

// drainQueue: ambil semua pending ops (synced_at null) yang sudah due
// (next_retry_at <= now), kirim ke server dalam batch maksimal 50.
//
// FIFO order via client_timestamp ASC supaya update later tidak overwrite
// update earlier dari user yang sama.
export async function drainQueue(): Promise<{ applied: number; rejected: number; failed: number }> {
  const now = new Date().toISOString();
  const pending = await db.ops
    .filter((op) => !op.synced_at && (!op.next_retry_at || op.next_retry_at <= now))
    .sortBy("client_timestamp");

  if (pending.length === 0) return { applied: 0, rejected: 0, failed: 0 };

  // Batch limit 50 ops per request — server cap 100 per request, kita
  // lebih kecil untuk safety + faster feedback.
  const batch = pending.slice(0, 50);
  let applied = 0;
  let rejected = 0;
  let failed = 0;

  try {
    const resp = await apiClient.post<ApplyOperationsResponse>(
      "/api/sync/operations",
      {
        operations: batch.map((op) => ({
          client_op_id: op.client_op_id,
          entity_type: op.entity_type,
          entity_id: op.entity_id,
          action: op.action,
          field_changes: op.field_changes,
          client_timestamp: op.client_timestamp,
        })),
      },
    );

    // Map results back to ops by client_op_id
    const resultMap = new Map(resp.results.map((r) => [r.client_op_id, r]));
    for (const op of batch) {
      const r = resultMap.get(op.client_op_id);
      if (!r) {
        failed++;
        continue;
      }
      if (r.status === "applied" || r.status === "duplicate") {
        await db.ops.update(op.client_op_id, {
          synced_at: new Date().toISOString(),
          error: undefined,
        });
        applied++;
      } else {
        await db.ops.update(op.client_op_id, { error: r.reason ?? "rejected" });
        rejected++;
      }
    }
  } catch (e) {
    // Batch network error — schedule retry untuk semua ops di batch.
    failed = batch.length;
    const errStr = e instanceof Error ? e.message : String(e);
    for (const op of batch) {
      const nextRetry = backoffDelay(op.retry_count);
      await db.ops.update(op.client_op_id, {
        retry_count: op.retry_count + 1,
        next_retry_at: new Date(Date.now() + nextRetry).toISOString(),
        error: errStr,
      });
    }
  }

  return { applied, rejected, failed };
}

// countPending: jumlah ops yang belum applied — untuk badge di UI.
export async function countPending(): Promise<number> {
  return db.ops.filter((op) => !op.synced_at).count();
}

// listPending: untuk tooltip detail di indicator badge.
export async function listPending(limit = 10): Promise<PendingOp[]> {
  return db.ops
    .filter((op) => !op.synced_at)
    .limit(limit)
    .toArray();
}

// Drainer: schedule background drain saat online + periodic.
// Dipanggil dari main.ts setelah pinia init.
let drainTimer: ReturnType<typeof setInterval> | null = null;
const DRAIN_INTERVAL_MS = 30_000;

export function startDrainer() {
  stopDrainer();
  drainTimer = setInterval(() => {
    if (typeof navigator !== "undefined" && navigator.onLine) {
      void drainQueue().catch((e) => console.error("opqueue drain:", e));
    }
  }, DRAIN_INTERVAL_MS);
  // Trigger immediate drain saat startup
  if (typeof navigator !== "undefined" && navigator.onLine) {
    void drainQueue().catch((e) => console.error("opqueue drain initial:", e));
  }
}

export function stopDrainer() {
  if (drainTimer) clearInterval(drainTimer);
  drainTimer = null;
}
