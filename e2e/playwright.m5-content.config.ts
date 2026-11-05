import { defineConfig, devices } from "@playwright/test";
import base from "./playwright.config";

/** P7b's real content file pages and host switches, with identical scope in each browser. */
export default defineConfig({
  ...base,
  testMatch:
    /files\/(?:f8-issue-attachments-page|f9-comment-files-page|f10-draft-files-page|f11-editor-images-page|f12-image-history-page|f16-content-lifetime-page)\.spec\.ts$/,
  grepInvert: /\[s3\]/,
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] } },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
