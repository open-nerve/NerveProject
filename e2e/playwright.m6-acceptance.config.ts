/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { defineConfig, devices } from "@playwright/test";
import base from "./playwright.config";

/**
 * M6's acceptance checks of its pages (P8 spec 2.11): each browser runs them on the real pages, outside the default
 * stories and CI, before a release.
 */
export default defineConfig({
  ...base,
  testDir: "acceptance",
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] } },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
