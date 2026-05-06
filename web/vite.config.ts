import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import { VitePWA } from "vite-plugin-pwa";
import { fileURLToPath, URL } from "node:url";

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    // PWA setup minimal untuk Fase 0. Cache strategy detail di-tune di Fase 3.
    VitePWA({
      // autoUpdate = SW langsung pakai versi baru saat next online visit
      // (tidak prompt user). Penting untuk skenario "staf offline beberapa
      // hari" — saat reconnect, dapatkan UI terbaru tanpa friction.
      registerType: "autoUpdate",
      manifest: {
        name: "Aplikasi Surat Kecamatan",
        short_name: "Surat Kec",
        description: "Manajemen surat masuk/keluar kantor kecamatan",
        theme_color: "#2080f0",
        lang: "id",
        display: "standalone",
        icons: [],
      },
      workbox: {
        navigateFallback: "/index.html",
        // Cleanup expired caches on activation supaya tidak akumulasi.
        cleanupOutdatedCaches: true,
        runtimeCaching: [
          {
            // Tenant config: NEVER cache. Branding/identity bisa berubah saat
            // tenant re-deploy, dan SW yang shared antar tenant (kalau dist/
            // di-host CDN) tidak boleh leak config tenant lain.
            urlPattern: /\/api\/config/,
            handler: "NetworkOnly",
          },
          {
            // PDF endpoints: NEVER cache. Sesuai mandate arsitektur — PDF
            // tetap online-only, hindari fill IndexedDB / Cache Storage.
            urlPattern: ({ url }) =>
              /^\/api\/surat\/[^/]+\/attachments\/[^/]+(\/preview)?$/.test(url.pathname),
            handler: "NetworkOnly",
          },
          {
            // Sync snapshot: NetworkFirst (klien pakai watermark untuk dedup),
            // tidak butuh cache — biarkan IndexedDB jadi source of truth.
            urlPattern: /\/api\/sync\/snapshot/,
            handler: "NetworkOnly",
          },
          {
            // Metadata GET (/api/surat list, /api/instansi, /api/klasifikasi,
            // /api/sifat, /api/me) — NetworkFirst dengan cache fallback,
            // 30s timeout untuk fail-over ke cache saat slow connection.
            urlPattern: ({ url, request }) =>
              request.method === "GET" &&
              /^\/api\/(surat($|\?)|instansi|klasifikasi|sifat|me|dashboard|notifications|users\/assignable|disposisi)/.test(
                url.pathname,
              ),
            handler: "NetworkFirst",
            options: {
              cacheName: "api-metadata",
              networkTimeoutSeconds: 5,
              expiration: { maxEntries: 100, maxAgeSeconds: 60 * 60 * 24 * 7 }, // 1 week
              cacheableResponse: { statuses: [0, 200] },
            },
          },
          {
            // Mutations + auth: NetworkOnly (jangan cache POST/PATCH/DELETE).
            urlPattern: ({ url }) => url.pathname.startsWith("/api/"),
            handler: "NetworkOnly",
          },
        ],
      },
    }),
  ],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": "http://localhost:8080",
      "/healthz": "http://localhost:8080",
    },
  },
  build: {
    sourcemap: true,
  },
  test: {
    globals: true,
    environment: "jsdom",
  },
});
