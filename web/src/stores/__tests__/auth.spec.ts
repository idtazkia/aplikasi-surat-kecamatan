import { describe, it, expect, beforeEach, vi } from "vitest";
import { setActivePinia, createPinia } from "pinia";
import { useAuthStore } from "../auth";
import { apiClient } from "@/api/client";

vi.mock("@/api/client", () => ({
  apiClient: {
    post: vi.fn(),
  },
  ApiError: class extends Error {
    status: number;
    body: string;
    constructor(status: number, body: string) {
      super(`API error ${status}: ${body}`);
      this.status = status;
      this.body = body;
    }
  },
}));

const mockPost = apiClient.post as unknown as ReturnType<typeof vi.fn>;

// Helper untuk encode JWT-like payload (base64url) untuk test
function fakeAccessToken(payload: object): string {
  const header = btoa(JSON.stringify({ alg: "HS256", typ: "JWT" }))
    .replace(/=/g, "")
    .replace(/\+/g, "-")
    .replace(/\//g, "_");
  const body = btoa(JSON.stringify(payload))
    .replace(/=/g, "")
    .replace(/\+/g, "-")
    .replace(/\//g, "_");
  return `${header}.${body}.signature-placeholder`;
}

describe("auth store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    localStorage.clear();
    mockPost.mockReset();
  });

  it("login persists token + roles ke localStorage", async () => {
    const access = fakeAccessToken({ sub: "user-1", roles: ["staf"] });
    mockPost.mockResolvedValueOnce({
      access_token: access,
      refresh_token: fakeAccessToken({ sub: "user-1", typ: "refresh" }),
    });

    const auth = useAuthStore();
    await auth.login("staf1", "demo123");

    expect(auth.accessToken).toBe(access);
    expect(auth.userID).toBe("user-1");
    expect(auth.roles).toEqual(["staf"]);
    expect(auth.isAuthenticated).toBe(true);

    const stored = JSON.parse(localStorage.getItem("surat-kec-auth") as string);
    expect(stored.accessToken).toBe(access);
    expect(stored.userID).toBe("user-1");
  });

  it("logout clear token dari state + localStorage", async () => {
    mockPost.mockResolvedValueOnce({
      access_token: fakeAccessToken({ sub: "user-1", roles: ["staf"] }),
      refresh_token: "refresh",
    });
    const auth = useAuthStore();
    await auth.login("staf1", "demo123");
    expect(auth.isAuthenticated).toBe(true);

    auth.logout();

    expect(auth.accessToken).toBe("");
    expect(auth.userID).toBe("");
    expect(auth.roles).toEqual([]);
    expect(auth.isAuthenticated).toBe(false);
    expect(localStorage.getItem("surat-kec-auth")).toBeNull();
  });

  it("hasRole check membership di roles array", async () => {
    mockPost.mockResolvedValueOnce({
      access_token: fakeAccessToken({ sub: "user-1", roles: ["camat", "staf"] }),
      refresh_token: "refresh",
    });
    const auth = useAuthStore();
    await auth.login("user", "x");

    expect(auth.hasRole("camat")).toBe(true);
    expect(auth.hasRole("staf")).toBe(true);
    expect(auth.hasRole("admin")).toBe(false);
    expect(auth.hasRole("student")).toBe(false);
  });

  it("login error tidak persist token", async () => {
    mockPost.mockRejectedValueOnce(new Error("network"));
    const auth = useAuthStore();
    await expect(auth.login("staf1", "wrong")).rejects.toThrow();

    expect(auth.accessToken).toBe("");
    expect(localStorage.getItem("surat-kec-auth")).toBeNull();
  });

  it("refresh meng-update access token tanpa ganti refresh", async () => {
    const initialAccess = fakeAccessToken({ sub: "user-1", roles: ["staf"] });
    const newAccess = fakeAccessToken({ sub: "user-1", roles: ["staf"], iat: 99999 });
    mockPost
      .mockResolvedValueOnce({
        access_token: initialAccess,
        refresh_token: "refresh-token-original",
      })
      .mockResolvedValueOnce({
        access_token: newAccess,
      });

    const auth = useAuthStore();
    await auth.login("user", "x");
    await auth.refresh();

    expect(auth.accessToken).toBe(newAccess);
    expect(auth.refreshToken).toBe("refresh-token-original");
  });

  it("init store dari localStorage existing", () => {
    localStorage.setItem(
      "surat-kec-auth",
      JSON.stringify({
        accessToken: "stored-access",
        refreshToken: "stored-refresh",
        userID: "user-99",
        roles: ["admin"],
      }),
    );
    setActivePinia(createPinia()); // recreate pinia untuk fresh store init
    const auth = useAuthStore();
    expect(auth.accessToken).toBe("stored-access");
    expect(auth.userID).toBe("user-99");
    expect(auth.roles).toEqual(["admin"]);
  });

  it("init store dari localStorage corrupt -> reset ke kosong", () => {
    localStorage.setItem("surat-kec-auth", "not-valid-json{");
    setActivePinia(createPinia());
    const auth = useAuthStore();
    expect(auth.accessToken).toBe("");
    expect(auth.isAuthenticated).toBe(false);
  });
});
