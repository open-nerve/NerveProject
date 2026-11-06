import { defineConfig, devices } from "@playwright/test";

// The default deployment demo migrates and runs a production nerve of its own in each browser; it runs through
// playwright.m5-demo.config.ts, outside the default stories.
const defaultDeployment = /files\/f18-default-deployment\.spec\.ts$/;
const s3Fixture = /smoke\/s3-lifecycle\.spec\.ts$/;

export default defineConfig({
  testDir: "stories",
  testIgnore: defaultDeployment,
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
    // The S3 fixture's own lifecycle tests start and remove containers. On a Linux host that changes the network
    // interfaces, and Chromium then fails every request in flight with net::ERR_NETWORK_CHANGED, so they finish before
    // any browser starts (P8 T14). The stories' S3 service is the run's, started by global-setup.ts.
    { name: "s3-fixture", testMatch: s3Fixture },
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
      testIgnore: [defaultDeployment, s3Fixture],
      dependencies: ["s3-fixture"],
    },
    // Firefox runs A4 and the static-resource regressions. A4 shows that a refresh lock release made as its task
    // settles can reach the next tab before the task's localStorage writes, and that one task later it follows them (M3
    // closeout, F-A4). The other stories expect Chromium's console messages and clipboard permissions.
    // Static resources also exercise Firefox's native NS_BINDING_ABORTED image restart.
    {
      name: "firefox",
      use: { ...devices["Desktop Firefox"] },
      grep: /A4 \(page\)|static resource guard/,
      dependencies: ["s3-fixture"],
    },
  ],
});
