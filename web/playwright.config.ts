import { defineConfig, devices } from "@playwright/test";

const PORT_FRONTEND = 5173;

// Backend di-spawn di global-setup.ts (perlu DATABASE_URL dari testcontainers).
// Hanya frontend dev server yang via webServer config.
export default defineConfig({
  testDir: "./e2e/specs",
  fullyParallel: false, // share single backend; tests stateful (seed DB)
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? "github" : "list",

  globalSetup: "./e2e/global-setup.ts",

  use: {
    baseURL: `http://localhost:${PORT_FRONTEND}`,
    trace: "on-first-retry",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },

  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
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
