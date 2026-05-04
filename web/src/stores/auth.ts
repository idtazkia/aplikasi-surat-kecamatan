import { defineStore } from "pinia";
import { ref, computed } from "vue";
import { apiClient } from "@/api/client";
import { resetCache } from "@/offline/sync";

const STORAGE_KEY = "surat-kec-auth";

interface AuthSnapshot {
  accessToken: string;
  refreshToken: string;
  userID: string;
  roles: string[];
}

function loadFromStorage(): AuthSnapshot | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as AuthSnapshot) : null;
  } catch {
    return null;
  }
}

export const useAuthStore = defineStore("auth", () => {
  const initial = loadFromStorage();
  const accessToken = ref<string>(initial?.accessToken ?? "");
  const refreshToken = ref<string>(initial?.refreshToken ?? "");
  const userID = ref<string>(initial?.userID ?? "");
  const roles = ref<string[]>(initial?.roles ?? []);

  const isAuthenticated = computed(() => Boolean(accessToken.value));
  const hasRole = (code: string) => roles.value.includes(code);

  function persist() {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        accessToken: accessToken.value,
        refreshToken: refreshToken.value,
        userID: userID.value,
        roles: roles.value,
      } satisfies AuthSnapshot),
    );
  }

  async function login(username: string, password: string) {
    const resp = await apiClient.post<{ access_token: string; refresh_token: string }>(
      "/api/auth/login",
      { username, password },
    );
    accessToken.value = resp.access_token;
    refreshToken.value = resp.refresh_token;
    // Decode JWT payload (no verify — server sudah verify saat issue)
    // untuk extract sub & roles. Frontend trust token + verify expiry only.
    const payload = JSON.parse(atob(resp.access_token.split(".")[1] ?? ""));
    userID.value = payload.sub ?? "";
    roles.value = payload.roles ?? [];
    persist();
  }

  function logout() {
    accessToken.value = "";
    refreshToken.value = "";
    userID.value = "";
    roles.value = [];
    localStorage.removeItem(STORAGE_KEY);
    // Reset offline cache supaya tidak data leak antar user di shared device.
    void resetCache().catch((e) => console.error("resetCache on logout:", e));
  }

  async function refresh() {
    if (!refreshToken.value) throw new Error("no refresh token");
    const resp = await apiClient.post<{ access_token: string }>("/api/auth/refresh", {
      refresh_token: refreshToken.value,
    });
    accessToken.value = resp.access_token;
    persist();
  }

  return {
    accessToken,
    refreshToken,
    userID,
    roles,
    isAuthenticated,
    hasRole,
    login,
    logout,
    refresh,
  };
});
