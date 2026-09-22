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
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
