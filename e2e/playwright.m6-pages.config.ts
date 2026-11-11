import { defineConfig, devices } from "@playwright/test";
import base from "./playwright.config";

/** M6's cycle and module pages (P5a spec 2.9, M4 handoff M6-04(a)): each browser runs the real pages and forms. */
export default defineConfig({
  ...base,
  testMatch: /planning\/g\d+-[\w-]+-page\.spec\.ts$/,
  grep: /G\d+ \(page\)/,
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] } },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
