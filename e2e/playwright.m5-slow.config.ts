import { defineConfig, devices } from "@playwright/test";
import base from "./playwright.config";

/** Drills that hold a real page to a slow network: Chromium only, outside the default stories. */
export default defineConfig({
  ...base,
  testDir: "drills",
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
