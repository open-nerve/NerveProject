import { defineConfig, devices } from "@playwright/test";
import base from "./playwright.config";

/** The default deployment demo (F18): a production nerve on its own database, in each browser. */
export default defineConfig({
  ...base,
  testMatch: /files\/f18-default-deployment\.spec\.ts$/,
  testIgnore: [],
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] } },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
