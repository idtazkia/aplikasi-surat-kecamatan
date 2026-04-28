// Thin fetch wrapper. Pinia auth store inject token; eduPanel store extract _edu.
//
// Tidak pakai axios — fetch native + Promise sudah cukup untuk skala app ini.
// Mahasiswa bisa baca implementasi tanpa kenalan library tambahan.

import { useEduPanelStore } from "@/stores/eduPanel";

interface RequestOptions {
  method?: string;
  body?: unknown;
  headers?: Record<string, string>;
  signal?: AbortSignal;
}

async function request<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, headers = {}, signal } = opts;

  // Inject auth token kalau ada
  const authRaw = localStorage.getItem("surat-kec-auth");
  if (authRaw) {
    try {
      const auth = JSON.parse(authRaw);
      if (auth.accessToken) {
        headers["Authorization"] = `Bearer ${auth.accessToken}`;
      }
    } catch {
      // ignore parse error
    }
  }

  const init: RequestInit = {
    method,
    headers: { "Content-Type": "application/json", ...headers },
    signal,
  };
  if (body !== undefined) {
    init.body = JSON.stringify(body);
  }

  const resp = await fetch(path, init);
  if (!resp.ok) {
    const errBody = await resp.text();
    throw new ApiError(resp.status, errBody);
  }

  const json = (await resp.json()) as T & { _edu?: unknown };

  // Kalau ada _edu block, push ke student panel store
  if (json && typeof json === "object" && "_edu" in json && json._edu) {
    const eduStore = useEduPanelStore();
    eduStore.recordPayload(json._edu as never);
  }

  return json;
}

export class ApiError extends Error {
  status: number;
  body: string;
  constructor(status: number, body: string) {
    super(`API error ${status}: ${body}`);
    this.status = status;
    this.body = body;
  }
}

export const apiClient = {
  get: <T>(path: string, opts?: Omit<RequestOptions, "method" | "body">) =>
    request<T>(path, { ...opts, method: "GET" }),
  post: <T>(path: string, body?: unknown, opts?: Omit<RequestOptions, "method" | "body">) =>
    request<T>(path, { ...opts, method: "POST", body }),
  patch: <T>(path: string, body?: unknown, opts?: Omit<RequestOptions, "method" | "body">) =>
    request<T>(path, { ...opts, method: "PATCH", body }),
  delete: <T>(path: string, opts?: Omit<RequestOptions, "method" | "body">) =>
    request<T>(path, { ...opts, method: "DELETE" }),
};
