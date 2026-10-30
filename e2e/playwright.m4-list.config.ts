import { defineConfig, devices } from "@playwright/test";
import base from "./playwright.config";

/** P12 C1/C3 only: each browser runs the actual list pages, without the default Firefox A4 restriction. */
export default defineConfig({
  ...base,
  testMatch:
    /issue\/(?:i3-list-groups|i4-list-order-calendar|i6-board-order|i7-quick-add|list-interactions)\.spec\.ts$/,
  grep: /\(page\)|C[13]:/,
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] } },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
