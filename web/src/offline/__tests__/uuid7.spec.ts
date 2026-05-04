import { describe, it, expect, vi } from "vitest";
import { uuidv7 } from "../uuid7";

describe("uuidv7", () => {
  it("output format match RFC 4122 hex+hyphen pattern", () => {
    const id = uuidv7();
    expect(id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
  });

  it("version nibble = 7", () => {
    const id = uuidv7();
    // Char di index 14 (after 2 hyphens + 12 hex chars) harus '7'
    expect(id.charAt(14)).toBe("7");
  });

  it("variant bits = 10xx (RFC 4122 → first hex char di group 4 ∈ {8,9,a,b})", () => {
    const id = uuidv7();
    expect(id.charAt(19)).toMatch(/[89ab]/);
  });

  it("time-ordered: ID baru > ID lama (string comparison)", async () => {
    const a = uuidv7();
    await new Promise((r) => setTimeout(r, 5));
    const b = uuidv7();
    // 48-bit timestamp prefix → ID berikutnya selalu > sebelumnya saat string compare
    expect(b > a).toBe(true);
  });

  it("timestamp prefix encode ms saat ini", () => {
    const before = Date.now();
    const id = uuidv7();
    const after = Date.now();

    // Decode 48-bit prefix dari hex (8 chars + 4 chars)
    const prefix = id.slice(0, 8) + id.slice(9, 13);
    const ts = parseInt(prefix, 16);
    expect(ts).toBeGreaterThanOrEqual(before);
    expect(ts).toBeLessThanOrEqual(after);
  });

  it("dua call sequential berbeda (random component cegah collision)", () => {
    const a = uuidv7();
    const b = uuidv7();
    expect(a).not.toBe(b);
  });

  it("fallback ke Math.random kalau crypto.getRandomValues tidak ada", () => {
    const originalCrypto = globalThis.crypto;
    // @ts-expect-error force remove crypto
    delete globalThis.crypto;
    try {
      const id = uuidv7();
      expect(id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
    } finally {
      globalThis.crypto = originalCrypto;
    }
  });

  it("ID dari ms berbeda time-ordered di 48-bit prefix", async () => {
    // UUIDs dari ms yang sama random sort (rand_a + rand_b dominates),
    // tapi UUIDs dari ms berbeda pasti time-ordered di 48-bit prefix.
    const a = uuidv7();
    await new Promise((r) => setTimeout(r, 10));
    const b = uuidv7();
    await new Promise((r) => setTimeout(r, 10));
    const c = uuidv7();
    expect(a < b).toBe(true);
    expect(b < c).toBe(true);
  });

  // Suppress mock warning
  void vi;
});
