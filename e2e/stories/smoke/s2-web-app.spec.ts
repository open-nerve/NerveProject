import type { Page, Response } from "@playwright/test";

import { expect, test } from "../../fixtures/test";

/** A page of the frontend's router, not a file: nerve answers it with index.html. */
const deepLink = "/acme/projects/0199f1c2-7a1b-7c3d-8e4f-5a6b7c8d9e0f/issues";

/**
 * Static resources are everything outside /api/ (M0 design 3.3). API calls are
 * left out: in M0 the Plane frontend still calls Plane's API, such as
 * GET /api/instances/, and nerve answers those with 404.
 */
function isStatic(url: string): boolean {
  return !new URL(url).pathname.startsWith("/api/");
}

interface Visit {
  document: Response;
  /** Static resources that loaded, as "<status> <url>". */
  loaded: string[];
  /** Static resources that failed: an HTTP error, or refused or aborted by the browser. */
  failed: string[];
  /** Requests to any origin other than nerve's; the frontend is served same-origin. */
  elsewhere: string[];
}

/** Opens path and waits until the network is idle. */
async function open(page: Page, path: string): Promise<Visit> {
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
  const document = await page.goto(path, { waitUntil: "networkidle" });
  if (!document) {
    throw new Error(`no document response for ${path}`);
  }
  const origin = new URL(document.url()).origin;
  const elsewhere = requested.filter((url) => new URL(url).origin !== origin);
  return { document, loaded, failed, elsewhere };
}

test("S2: a user opens the home page in a browser", async ({ page }) => {
  const { document, loaded, failed, elsewhere } = await open(page, "/");

  expect(document.status()).toBe(200);
  expect(document.headers()["content-type"]).toBe("text/html; charset=utf-8");
  expect(loaded).toContainEqual(expect.stringMatching(/^200 .*\/assets\/[^/]+\.js$/));
  expect(failed).toEqual([]);
  expect(elsewhere).toEqual([]);
});

test("S2: a user opens a deep link directly", async ({ page, request }) => {
  const { document, failed, elsewhere } = await open(page, deepLink);

  expect(document.status()).toBe(200);
  expect(document.headers()["content-type"]).toBe("text/html; charset=utf-8");
  expect(await document.body()).toEqual(await (await request.get("/")).body());
  expect(failed).toEqual([]);
  expect(elsewhere).toEqual([]);
});
