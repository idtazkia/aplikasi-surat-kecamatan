import { describe, it, expect, beforeEach, vi } from "vitest";
import { setActivePinia, createPinia } from "pinia";
import { apiClient, ApiError } from "../client";
import { useEduPanelStore } from "@/stores/eduPanel";

describe("apiClient", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it("GET sukses return parsed body", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      json: async () => ({ status: "ok" }),
    } as never);

    const r = await apiClient.get<{ status: string }>("/healthz");
    expect(r.status).toBe("ok");
    expect(global.fetch).toHaveBeenCalledWith(
      "/healthz",
      expect.objectContaining({ method: "GET" }),
    );
  });

  it("POST kirim JSON body + Content-Type header", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      json: async () => ({ ok: true }),
    } as never);

    await apiClient.post("/api/auth/login", { username: "x", password: "y" });

    const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(call[0]).toBe("/api/auth/login");
    expect(call[1].method).toBe("POST");
    expect(call[1].headers["Content-Type"]).toBe("application/json");
    expect(JSON.parse(call[1].body)).toEqual({ username: "x", password: "y" });
  });

  it("auto-inject Authorization header dari localStorage", async () => {
    localStorage.setItem(
      "surat-kec-auth",
      JSON.stringify({ accessToken: "test-token", refreshToken: "x", userID: "u", roles: [] }),
    );
    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      json: async () => ({}),
    } as never);

    await apiClient.get("/api/me");

    const call = (global.fetch as ReturnType<typeof vi.fn>).mock.calls[0];
    expect(call[1].headers.Authorization).toBe("Bearer test-token");
  });

  it("non-OK response throw ApiError dengan status", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: false,
      status: 401,
      text: async () => '{"error":"invalid credentials"}',
    } as never);

    await expect(apiClient.post("/api/auth/login", {})).rejects.toThrowError(ApiError);
    try {
      await apiClient.post("/api/auth/login", {});
    } catch (e) {
      expect(e).toBeInstanceOf(ApiError);
      expect((e as ApiError).status).toBe(401);
    }
  });

  it("_edu payload di response auto-record ke eduPanel store", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        data: [],
        _edu: { operation: "list_surat", complexity: { theoretical: "O(log n)" } },
      }),
    } as never);

    const edu = useEduPanelStore();
    edu.enabled = true; // store harus enabled supaya recordPayload simpan
    await apiClient.get("/api/surat");

    expect(edu.lastPayload?.operation).toBe("list_surat");
  });

  it("_edu payload tidak record kalau eduPanel disabled", async () => {
    global.fetch = vi.fn().mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        data: [],
        _edu: { operation: "list_surat" },
      }),
    } as never);

    const edu = useEduPanelStore();
    // enabled tetap default false
    await apiClient.get("/api/surat");

    expect(edu.lastPayload).toBeNull();
  });

  it("PATCH dan DELETE methods", async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({}),
    } as never);

    await apiClient.patch("/api/surat/1", { perihal: "edit" });
    await apiClient.delete("/api/surat/1");

    const calls = (global.fetch as ReturnType<typeof vi.fn>).mock.calls;
    expect(calls[0][1].method).toBe("PATCH");
    expect(calls[1][1].method).toBe("DELETE");
  });
});
