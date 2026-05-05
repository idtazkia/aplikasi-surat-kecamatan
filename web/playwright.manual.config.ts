// Playwright config terpisah untuk capture screenshot user manual.
//
// Beda dengan playwright.config.ts (E2E):
// - testDir → ./e2e/manual (bukan ./e2e/specs)
// - Tidak retry (manual capture deterministic flow, bukan test assertion)
// - Tidak auto screenshot/video on failure (kita save screenshot eksplisit
//   ke docs/user-manual/src/screenshots/)
// - Reuse global-setup untuk testcontainer + backend
//
// Usage:
//   npx playwright test --config=playwright.manual.config.ts
import { defineConfig, devices } from "@playwright/test";

const PORT_FRONTEND = 5173;

export default defineConfig({
  testDir: "./e2e/manual",
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: "list",
  timeout: 60_000,

  globalSetup: "./e2e/global-setup.ts",

  use: {
    baseURL: `http://localhost:${PORT_FRONTEND}`,
    // Viewport realistic untuk screenshot manual (bukan test viewport tiny)
    viewport: { width: 1440, height: 900 },
  },

  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"], viewport: { width: 1440, height: 900 } },
    },
  ],

  webServer: [
    {
      command: "npm run dev",
      port: PORT_FRONTEND,
      reuseExistingServer: !process.env.CI,
      timeout: 30_000,
    },
  ],
});
