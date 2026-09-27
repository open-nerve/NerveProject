import type { Page, Response } from "@playwright/test";

import { signInPath } from "../../fixtures/auth-pages";
import { watchPage, type PageWatch } from "../../fixtures/browser";
import { expect, test } from "../../fixtures/test";

/** A page of the frontend's router, not a file: nerve answers it with index.html. */
const deepLink = "/acme/projects/0199f1c2-7a1b-7c3d-8e4f-5a6b7c8d9e0f/issues";

interface Visit {
  document: Response;
  watch: PageWatch;
  /** Static resources that loaded, as "<status> <url>". */
  loaded: string[];
  /** Static resources that failed: an HTTP error, or refused or aborted by the browser. */
  failed: string[];
  /** Requests to any origin other than nerve's; the frontend is served same-origin. */
  elsewhere: string[];
}

function isStatic(url: string): boolean {
  return !new URL(url).pathname.startsWith("/api/");
}

/**
 * Opens path, signed out, and waits until the app has started: nerve has answered the one request the app
 * makes as it starts, for the instance's settings, and the sign-in page shows. Not networkidle: a request
 * whose answer the page never reads keeps the network busy, and the wait would last until the test's
 * timeout instead of failing on what the page sent.
 */
async function open(page: Page, path: string): Promise<Visit> {
  const watch = await watchPage(page);
  const requested: string[] = [];
  const loaded: string[] = [];
  const failed: string[] = [];
  page.on("request", (req) => {
    requested.push(req.url());
  });
  page.on("response", (res) => {
    if (isStatic(res.url())) {
      (res.status() < 400 ? loaded : failed).push(`${res.status()} ${res.url()}`);
    }
  });
  page.on("requestfailed", (req) => {
    if (isStatic(req.url())) {
      failed.push(`${req.failure()?.errorText} ${req.url()}`);
    }
  });
  const [document] = await Promise.all([
    page.goto(path),
    page.waitForResponse((res) => new URL(res.url()).pathname === "/api/v0/instance", { timeout: 10_000 }),
  ]);
  if (!document) {
    throw new Error(`no document response for ${path}`);
  }
  await expect(page.getByRole("button", { name: "Go to workspace" })).toBeVisible();
  const origin = new URL(document.url()).origin;
  const elsewhere = requested.filter((url) => new URL(url).origin !== origin);
  return { document, watch, loaded, failed, elsewhere };
}

/**
 * Signed out, the app asks nerve for the instance's settings only: without a refresh token it neither
 * refreshes nor asks for /me (M2 design 7.1), so no API call fails; and the page's Content-Security-Policy
 * blocks nothing of it (8.3).
 */
function expectQuietSignedOut(watch: PageWatch): void {
  expect(watch.apiRequests).toEqual(["GET /api/v0/instance"]);
  expect(watch.apiFailures).toEqual([]);
  expect(watch.cspViolations).toEqual([]);
  expect(watch.pageErrors).toEqual([]);
  expect(watch.consoleErrors).toEqual([]);
}

test("S2: a user opens the home page in a browser", async ({ page }) => {
  const { document, watch, loaded, failed, elsewhere } = await open(page, "/");

  expect(document.status()).toBe(200);
  expect(document.headers()["content-type"]).toBe("text/html; charset=utf-8");
  expect(document.headers()["content-security-policy"]).toMatch(/^default-src 'self'; script-src 'self' 'sha256-/);
  expect(loaded).toContainEqual(expect.stringMatching(/^200 .*\/assets\/[^/]+\.js$/));
  expect(failed).toEqual([]);
  expect(elsewhere).toEqual([]);
  expectQuietSignedOut(watch);
});

test("S2: a user opens a deep link directly", async ({ page, request }) => {
  const { document, watch, failed, elsewhere } = await open(page, deepLink);

  expect(document.status()).toBe(200);
  expect(document.headers()["content-type"]).toBe("text/html; charset=utf-8");
  expect(await document.body()).toEqual(await (await request.get("/")).body());
  // Signed out, the page behind the sign-in goes to the sign-in page, which comes back to it.
  await expect(page).toHaveURL(signInPath(deepLink));
  expect(failed).toEqual([]);
  expect(elsewhere).toEqual([]);
  expectQuietSignedOut(watch);
});
