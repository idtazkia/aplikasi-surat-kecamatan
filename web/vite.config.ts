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
      registerType: "prompt", // user prompt saat versi baru tersedia
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
        // Strategy detail untuk metadata vs PDF di-implement Fase 3.
        // Sekarang minimal: precache static assets, jangan cache /api/*.
        navigateFallback: "/index.html",
        runtimeCaching: [
          {
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
