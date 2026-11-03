import { defineConfig, devices } from "@playwright/test";
import base from "./playwright.config";

/** The same browser transfer contract on each engine, with isolated worlds. */
export default defineConfig({
  ...base,
  testMatch: /files\/f4-s3-storage\.spec\.ts$/,
  workers: 1,
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] } },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
