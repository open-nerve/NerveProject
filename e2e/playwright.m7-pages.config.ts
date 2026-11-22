import { defineConfig, devices } from "@playwright/test";
import base from "./playwright.config";

/**
 * M7's collaboration pages (M7 P4 spec 2.11): each browser runs the real pages of the page stories, H1–H20's "(page)"
 * versions, never their API versions. The default config runs the same stories once in Chromium (CI).
 */
export default defineConfig({
  ...base,
  testMatch: /collaboration\/h\d+(?:-h\d+)?-[\w-]+-pages?\.spec\.ts$/,
  grep: /H\d+(?:–H\d+)? \(page\)/,
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] } },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
