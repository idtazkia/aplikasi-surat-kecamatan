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
    environment: "jsdom",
    include: ["src/**/*.{test,spec}.ts"],
    exclude: ["node_modules", "dist", "e2e"],
    coverage: {
      provider: "v8",
      reporter: ["text", "html", "json-summary"],
      include: ["src/**/*.{ts,vue}"],
      exclude: [
        "src/**/*.spec.ts",
        "src/**/*.test.ts",
        "src/main.ts",
        "src/env.d.ts",
        "src/router/**",
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
