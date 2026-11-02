import { defineConfig, devices } from "@playwright/test";
import base from "./playwright.config";
import list from "./playwright.m4-list.config";
import filters from "./playwright.m4-filters.config";
import detail from "./playwright.m4-detail.config";
import collaboration from "./playwright.m4-collaboration.config";

const previous = [list, filters, detail, collaboration];

/** P16 prepares C12: the union of prior M4 pages, all S2 entries and the changed editor controls. */
export default defineConfig({
  ...base,
  testMatch: [
    ...previous.flatMap((config) => config.testMatch ?? []),
    /issue\/cleanup-interactions\.spec\.ts$/,
    /issue\/editor-review-interactions\.spec\.ts$/,
    /issue\/editor-review-links\.spec\.ts$/,
    /issue\/review-interactions\.spec\.ts$/,
    /issue\/review-boundaries\.spec\.ts$/,
    /issue\/review-lifecycle\.spec\.ts$/,
  ],
  grep: [...previous.flatMap((config) => config.grep ?? []), /S2:|C12 \((cleanup|review)\):/],
  projects: [
    { name: "chromium", use: { ...devices["Desktop Chrome"] } },
    { name: "firefox", use: { ...devices["Desktop Firefox"] } },
    { name: "webkit", use: { ...devices["Desktop Safari"] } },
  ],
});
