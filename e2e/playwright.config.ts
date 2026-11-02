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
    // Firefox runs A4 and the static-resource regressions. A4 shows that a refresh lock release made as its task
    // settles can reach the next tab before the task's localStorage writes, and that one task later it follows them (M3
    // closeout, F-A4). The other stories expect Chromium's console messages and clipboard permissions.
    // Static resources also exercise Firefox's native NS_BINDING_ABORTED image restart.
    { name: "firefox", use: { ...devices["Desktop Firefox"] }, grep: /A4 \(page\)|static resource guard/ },
  ],
});
