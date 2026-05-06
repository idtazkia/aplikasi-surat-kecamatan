import { createApp } from "vue";
import { createPinia } from "pinia";
import { VueQueryPlugin } from "@tanstack/vue-query";
import App from "./App.vue";
import router from "./router";
import { useOfflineStore } from "@/stores/offline";
import { useAuthStore } from "@/stores/auth";
import { startDrainer } from "@/offline/opqueue";
import { loadConfig } from "@/config";

import "vfonts/Lato.css";
import "vfonts/FiraCode.css";

// Boot order:
//   1. Fetch /config.json (tenant config) — fatal kalau gagal
//   2. createApp + Pinia + router
//   3. Init offline store + sync
//   4. Mount
//
// loadConfig() di-await sebelum createApp supaya useConfigStore dapat
// langsung initialize dari config yang sudah loaded — no race.
async function boot() {
  await loadConfig();

  const app = createApp(App);
  const pinia = createPinia();
  app.use(pinia);
  app.use(router);
  app.use(VueQueryPlugin);

  const offline = useOfflineStore(pinia);
  offline.bindBrowserEvents();
  void offline.refreshMetaFromCache();

  const auth = useAuthStore(pinia);
  if (auth.isAuthenticated && offline.online) {
    void offline.sync();
  }

  startDrainer();
  app.mount("#app");
}

boot().catch((err: unknown) => {
  // Fatal config load error → render minimal error page langsung di DOM.
  // Tidak pakai Vue/router karena belum ter-mount. Tujuan: pesan jelas
  // ke admin/operator soal apa yang misconfig.
  const root = document.querySelector("#app");
  if (root) {
    const message = err instanceof Error ? err.message : String(err);
    root.innerHTML = `
      <div style="font-family: system-ui, sans-serif; max-width: 600px; margin: 4rem auto; padding: 2rem; border: 1px solid #d33; border-radius: 8px; color: #222;">
        <h1 style="color: #d33; margin-top: 0;">Konfigurasi tenant gagal di-load</h1>
        <p>Aplikasi tidak bisa boot karena <code>/config.json</code> tidak tersedia atau invalid.</p>
        <p><strong>Error:</strong> <code>${escapeHtml(message)}</code></p>
        <p style="font-size: 0.875rem; color: #666;">Hubungi administrator sistem. Nginx config harus serve <code>/config.json</code> dengan field <code>apiBaseUrl</code>, <code>appName</code>, <code>institutionName</code>, dan <code>branding</code>.</p>
      </div>
    `;
  }
  console.error("[boot] config load failed:", err);
});

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}
