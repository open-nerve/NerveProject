import { defineConfig, devices } from "@playwright/test";
import base from "./playwright.config";

/** P15's collaboration, drafts, search and fresh-clock pages actually execute in every named engine. */
export default defineConfig({
  ...base,
  testMatch:
    /issue\/(?:i11-comments-reactions|i12-activity|i13-description-history|i14-drafts|i16-auto-archive|i18-search|i20-guest-boundary|collaboration-interactions)\.spec\.ts$/,
  grep: /I(?:11|12|13|14|16|18) \(page\)|I20 \(page\).*comment|C[6789]:/,
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] } },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
