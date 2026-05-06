// Runtime tenant config — di-fetch sekali saat boot dari GET /api/config
// (backend-served, public no-auth). Backend baca env TENANT_* dari Ansible
// per-tenant deployment, jadi single source of truth: backend juga punya
// akses values yang sama untuk future watermark/email subject/audit context.
//
// Mekanisme ini menggantikan Vite build-time env untuk tenant-specific values,
// supaya satu `web/dist/` artifact bisa dipakai semua tenant.
//
// No fallback policy: kalau /api/config gagal di-load atau invalid, app
// throw fatal error → render error page. Bukan default values yang
// menyembunyikan misconfig.

export interface TenantConfig {
  apiBaseUrl: string;          // ex. "/api"
  appName: string;             // ex. "Aplikasi Surat Kecamatan Bogor Tengah"
  institutionName: string;     // ex. "Kantor Kecamatan Bogor Tengah"
  branding: {
    logoUrl?: string;          // path ke logo, ex. "/branding/logo.svg"
    primary: string;           // hex color
    primaryHover: string;
    accent: string;
  };
}

let loaded: TenantConfig | null = null;

export async function loadConfig(): Promise<TenantConfig> {
  // Cache-busting via timestamp untuk menghindari stale config saat tenant
  // di-redeploy dengan branding berbeda. Backend juga set Cache-Control:
  // no-store sebagai redundancy.
  const resp = await fetch(`/api/config?t=${Date.now()}`, {
    cache: "no-store",
  });
  if (!resp.ok) {
    throw new Error(
      `Gagal load /api/config (status ${resp.status}). ` +
        `Pastikan backend Go running + env TENANT_* ter-set.`,
    );
  }
  const raw = await resp.json();
  validateConfig(raw);
  loaded = raw as TenantConfig;
  return loaded;
}

// validateConfig — strict shape check. Backend sudah validate env saat boot,
// tapi frontend re-validate sebagai defense-in-depth (mis. response corrupt
// dari proxy intermediary). Field hilang = error fatal.
function validateConfig(raw: unknown): asserts raw is TenantConfig {
  if (!raw || typeof raw !== "object") {
    throw new Error("/api/config bukan object JSON valid");
  }
  const c = raw as Record<string, unknown>;
  for (const k of ["apiBaseUrl", "appName", "institutionName"]) {
    if (typeof c[k] !== "string" || !(c[k] as string).trim()) {
      throw new Error(`/api/config field '${k}' wajib non-empty string`);
    }
  }
  if (!c.branding || typeof c.branding !== "object") {
    throw new Error("/api/config field 'branding' wajib object");
  }
  const b = c.branding as Record<string, unknown>;
  for (const k of ["primary", "primaryHover", "accent"]) {
    if (typeof b[k] !== "string" || !(b[k] as string).match(/^#[0-9a-fA-F]{3,8}$/)) {
      throw new Error(`/api/config branding.${k} harus hex color (mis. '#204397')`);
    }
  }
}

// loadedConfig — return config setelah loadConfig() resolve. Throw kalau
// dipanggil sebelum boot complete (programming error).
export function loadedConfig(): TenantConfig {
  if (!loaded) {
    throw new Error("loadedConfig() dipanggil sebelum loadConfig() — bug boot order");
  }
  return loaded;
}
