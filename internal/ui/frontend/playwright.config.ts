import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: false,
  retries: 0,
  workers: 1,
  reporter: "line",
  use: {
    baseURL: "http://127.0.0.1:8879",
    browserName: "chromium",
    headless: true,
    trace: "retain-on-failure",
  },
  webServer: {
    command: "go run ../../../cmd/repolens ui --addr 127.0.0.1:8879",
    url: "http://127.0.0.1:8879/api/health",
    reuseExistingServer: false,
    timeout: 120_000,
  },
});
