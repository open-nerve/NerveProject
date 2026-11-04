import { defineConfig, devices } from "@playwright/test";
import base from "./playwright.config";

/** P6's real image controls and lifetime boundaries, with identical scope in each browser. */
export default defineConfig({
  ...base,
  testMatch: /files\/(?:f[567]-.*-page|f16-profile-upload-lifetime|f17-profile-upload-accessibility)\.spec\.ts$/,
  grepInvert: /\[s3\]/,
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] } },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
