import { defineConfig, devices } from "@playwright/test";
import base from "./playwright.config";

/** P14's actual detail, form, editor and identity stories have the same scope in every browser. */
export default defineConfig({
  ...base,
  testMatch:
    /(?:issue\/(?:i1-create-issue|i2-update-issue|i2-description|i8-sub-issues|i9-relations|i10-links|i17-delete-issue|i20-guest-boundary|i22-session-switch|detail-interactions|editor-loading|filter-archive-profile-interactions)|smoke\/s2-web-app)\.spec\.ts$/,
  grep: /\((?:page|detail)\)|C[2345]:|C1[012]:/,
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] } },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
