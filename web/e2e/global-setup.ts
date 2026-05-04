// Playwright global setup:
// 1. Spin up PostgreSQL via testcontainers
// 2. Apply schema + demo seed
// 3. Start Go backend dengan env DATABASE_URL dari container
// 4. Wait for /healthz OK sebelum tests jalan
//
// Returns cleanup function — Playwright call setelah semua test.
// Bahkan kalau cleanup gagal di-call, testcontainers reaper (Ryuk) hapus
// container otomatis saat parent process exit.
//
// Catatan: Playwright start webServer SEBELUM globalSetup, jadi backend
// tidak bisa pakai webServer config (DATABASE_URL belum ada saat itu).
// Solusi: backend di-spawn di sini, hanya frontend dev server yang pakai
// webServer config.
import { PostgreSqlContainer } from "@testcontainers/postgresql";
import type { StartedPostgreSqlContainer } from "@testcontainers/postgresql";
import { spawn, execSync } from "node:child_process";
import type { ChildProcess } from "node:child_process";
import { setTimeout as wait } from "node:timers/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(__dirname, "../..");
const BACKEND_PORT = 8080;
const BACKEND_HEALTH_URL = `http://127.0.0.1:${BACKEND_PORT}/healthz`;

async function waitForHealth(url: string, timeoutMs = 30_000): Promise<void> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    try {
      const r = await fetch(url);
      if (r.ok) return;
    } catch {
      /* not ready */
    }
    await wait(500);
  }
  throw new Error(`backend not ready setelah ${timeoutMs}ms: ${url}`);
}

async function globalSetup(): Promise<() => Promise<void>> {
  console.log("[e2e] Starting PostgreSQL testcontainer (postgres:16-alpine)...");
  const pg: StartedPostgreSqlContainer = await new PostgreSqlContainer(
    "postgres:16-alpine",
  )
    .withUsername("surat")
    .withPassword("surat")
    .withDatabase("surat_test")
    .start();
  const dbUrl = pg.getConnectionUri();
  console.log(`[e2e] DB ready`);

  const goose = `${process.env.GOPATH || `${process.env.HOME}/go`}/bin/goose`;

  console.log("[e2e] Applying schema migration...");
  execSync(`"${goose}" -dir db/migrations/schema postgres "${dbUrl}" up`, {
    cwd: REPO_ROOT,
    stdio: "inherit",
  });

  console.log("[e2e] Applying demo seed migration...");
  execSync(
    `"${goose}" -dir db/migrations/demo-seed -table goose_demo_seed_version postgres "${dbUrl}" up`,
    { cwd: REPO_ROOT, stdio: "inherit" },
  );

  // Pre-build binary kalau belum ada — lebih cepat dari go run karena
  // tidak compile per startup.
  const backendBinary = path.join(REPO_ROOT, "bin/server");
  console.log("[e2e] Building Go backend binary...");
  execSync(`go build -o ${backendBinary} ./cmd/server`, {
    cwd: REPO_ROOT,
    stdio: "inherit",
  });

  console.log("[e2e] Starting Go backend...");
  const backend: ChildProcess = spawn(backendBinary, [], {
    cwd: REPO_ROOT,
    env: {
      ...process.env,
      DATABASE_URL: dbUrl,
      JWT_SECRET: "test-secret-32-bytes-long-padding-x",
      LISTEN_ADDR: `127.0.0.1:${BACKEND_PORT}`,
      LOG_LEVEL: "warn",
      STUDENT_MODE_ENABLED: "true",
      ATTACHMENT_STORAGE_PATH: path.join(REPO_ROOT, "tmp", "e2e-attachments"),
    },
    stdio: ["ignore", "inherit", "inherit"],
  });

  await waitForHealth(BACKEND_HEALTH_URL);
  console.log("[e2e] Backend ready.");

  return async () => {
    console.log("[e2e] Cleanup: stopping backend...");
    backend.kill("SIGTERM");
    await new Promise<void>((resolve) => {
      const t = setTimeout(() => {
        backend.kill("SIGKILL");
        resolve();
      }, 5000);
      backend.on("exit", () => {
        clearTimeout(t);
        resolve();
      });
    });
    console.log("[e2e] Cleanup: stopping testcontainer...");
    await pg.stop();
    console.log("[e2e] Cleanup done.");
  };
}

export default globalSetup;
