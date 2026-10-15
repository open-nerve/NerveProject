import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "stories",
  // Starts PostgreSQL and migrates the template database once per run.
  globalSetup: "./global-setup.ts",
  forbidOnly: !!process.env.CI,
  // The html report keeps the traces and screenshots of failed tests; CI uploads it.
  reporter: [["list"], ["html", { open: "never" }]],
  use: {
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
  },
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    // Firefox runs A4's page tests alone: Firefox hands a task's localStorage writes to the other tabs at the end of
    // the task, the timing the refresh lock's release has to follow (M3 closeout, F-A4). The other stories expect
    // Chromium's console messages and clipboard permissions.
    { name: "firefox", use: { ...devices["Desktop Firefox"] }, grep: /A4 \(page\)/ },
  ],
});
