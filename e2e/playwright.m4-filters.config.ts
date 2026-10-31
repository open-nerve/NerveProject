import { defineConfig, devices } from "@playwright/test";
import base from "./playwright.config";

/** P13 C2/C9 and the three actual page stories; each browser has the same scope. */
export default defineConfig({
  ...base,
  testMatch:
    /issue\/(?:i5-list-preferences|i15-archive-issue|i19-member-issues|filter-archive-profile-interactions)\.spec\.ts$/,
  grep: /\(page\)|C[29]:/,
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] } },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
