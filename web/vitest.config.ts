import { defineConfig } from "vitest/config";
import vue from "@vitejs/plugin-vue";
import { fileURLToPath, URL } from "node:url";

// Vitest config terpisah dari vite.config.ts supaya test config benar-benar
// di-pick-up dengan environment jsdom. vite.config.ts punya plugin PWA yang
// tidak relevant dan bisa berat untuk test.
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  test: {
    globals: true,
    environment: "happy-dom",
    setupFiles: ["./vitest.setup.ts"],
    include: ["src/**/*.{test,spec}.ts"],
    exclude: ["node_modules", "dist", "e2e"],
    coverage: {
      provider: "v8",
      reporter: ["text", "html", "json-summary"],
      // Vitest unit test cover stores + API client (logic).
      // Komponen Vue (views/components/App.vue) di-cover oleh Playwright E2E
      // — tidak relevan double-test di sini.
      include: ["src/stores/**/*.ts", "src/api/**/*.ts"],
      exclude: [
        "src/**/*.spec.ts",
        "src/**/*.test.ts",
        "src/**/__tests__/**",
      ],
      thresholds: {
        lines: 80,
        functions: 80,
        branches: 80,
        statements: 80,
      },
    },
  },
});
