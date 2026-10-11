import { expect, type Page, type Request } from "@playwright/test";

/** What a page did that a story checks: its API calls, and what went wrong in it. */
export interface PageWatch {
  /** Every API request, as "<method> <path>". */
  readonly apiRequests: string[];
  /** Every request, by its URL, of any origin: nerve's, or another (requestsElsewhere). */
  readonly requestUrls: string[];
  /** API requests that failed: "<status> <method> <path>", or the browser's error for one without an answer. */
  readonly apiFailures: string[];
  /** Requests to the API of an older frontend, outside /api/v0 (M2 design 3.1): M3's domains, for instance. */
  readonly oldApiRequests: string[];
  /** Uncaught exceptions and unhandled rejections. */
  readonly pageErrors: string[];
  /** Console messages of type error. */
  readonly consoleErrors: string[];
  /** Console messages of type warning. */
  readonly consoleWarnings: string[];
  /** Content-Security-Policy violations, as "<directive> <blocked URI>" (M2 design 8.3). */
  readonly cspViolations: string[];
}

function apiPath(request: Request): string | undefined {
  const { pathname } = new URL(request.url());
  return pathname.startsWith("/api/") ? pathname : undefined;
}

/**
 * Starts watching page, and every document it loads from now on: call it before the page's first
 * navigation, so that nothing the app does as it starts goes unseen.
 */
export async function watchPage(page: Page): Promise<PageWatch> {
  const watch: PageWatch = {
    apiRequests: [],
    requestUrls: [],
    apiFailures: [],
    oldApiRequests: [],
    pageErrors: [],
    consoleErrors: [],
    consoleWarnings: [],
    cspViolations: [],
  };
  page.on("request", (request) => {
    watch.requestUrls.push(request.url());
    const path = apiPath(request);
    if (path === undefined) {
      return;
    }
    watch.apiRequests.push(`${request.method()} ${path}`);
    if (!path.startsWith("/api/v0/")) {
      watch.oldApiRequests.push(`${request.method()} ${path}`);
    }
  });
  // A request nerve answered is judged by its status: Chromium also reports one failed (net::ERR_ABORTED)
  // when the page leaves the body of its answer unread, as openapi-fetch does with a 204's.
  const answered = new WeakSet<Request>();
  page.on("response", (response) => {
    answered.add(response.request());
    const path = apiPath(response.request());
    if (path !== undefined && response.status() >= 400) {
      watch.apiFailures.push(`${response.status()} ${response.request().method()} ${path}`);
    }
  });
  page.on("requestfailed", (request) => {
    const path = apiPath(request);
    if (path !== undefined && !answered.has(request)) {
      watch.apiFailures.push(`${request.failure()?.errorText} ${request.method()} ${path}`);
    }
  });
  page.on("pageerror", (error) => {
    watch.pageErrors.push(error.message);
  });
  page.on("console", (message) => {
    if (message.type() === "error") {
      watch.consoleErrors.push(message.text());
    } else if (message.type() === "warning") {
      watch.consoleWarnings.push(message.text());
    }
  });
  await page.exposeBinding("__nerveE2eCspViolation", (_source, violation: string) => {
    watch.cspViolations.push(violation);
  });
  await page.addInitScript(() => {
    document.addEventListener("securitypolicyviolation", (event) => {
      const report = (window as unknown as { __nerveE2eCspViolation: (violation: string) => void })
        .__nerveE2eCspViolation;
      report(`${event.effectiveDirective} ${event.blockedURI}`);
    });
  });
  return watch;
}

/**
 * The requests watch saw page send to an origin other than the one it shows now: nerve serves the app and all it
 * loads, the emoji picker's data among them (M3 design 7.7), and the page's CSP allows no other (M2 design 8.3).
 */
export function requestsElsewhere(page: Page, watch: PageWatch): string[] {
  const { origin } = new URL(page.url());
  return watch.requestUrls.filter((url) => new URL(url).origin !== origin);
}

/**
 * Follows the access token page sends from now on, in the Authorization header of its requests: the function
 * returned gives the last one sent, or "" before the first.
 */
export function followAccessToken(page: Page): () => string {
  let last = "";
  page.on("request", (request) => {
    last = request.headers().authorization?.replace(/^Bearer /, "") ?? last;
  });
  return () => last;
}

/** Run in a page: its navigator.clipboard.writeText rejects with a NotAllowedError, as a browser's refusal does. */
function refuseWrites(): void {
  navigator.clipboard.writeText = () => Promise.reject(new DOMException("Write permission denied.", "NotAllowedError"));
}

/**
 * Makes the browser refuse page the clipboard, as one that has not let it write there (refuseWrites): from now on, in
 * the document page holds and in each it loads.
 */
export async function refuseClipboardWrites(page: Page): Promise<void> {
  await page.addInitScript(refuseWrites);
  await page.evaluate(refuseWrites);
}

/**
 * Chromium's hint when a script reads a canvas back often, which a page logs once for each load that brings the
 * editor: tiptap builds the editor's Emoji node, which asks is-emoji-supported about each emoji version (a canvas
 * and getImageData each time). A deep link signed out loads it before the page goes to the sign-in (S2); a
 * settings page's command palette (ProjectsAppPowerKProvider, kept since M1) creates an editor on every load. A
 * hint about a third party's code, not an error of the app: stories name it in expectQuietConsole, once a load.
 */
export const EMOJI_CHECK_WARNING =
  "Canvas2D: Multiple readback operations using getImageData are faster with the willReadFrequently attribute set to true. See: https://html.spec.whatwg.org/multipage/canvas.html#concept-canvas-will-read-frequently";

/**
 * Checks that page logged no error and no warning since watch began (M2 design 9.6), such as React Router's
 * "navigate() should be called in useEffect". It logs a probe of each kind first and expects to find it,
 * so that a watch that does not hear the console cannot pass. expected names what page logs that is not
 * the app's own, in order, each named by the caller with where it comes from: the warnings of third
 * parties, and the errors the browser logs about what the story makes happen, such as its report of an
 * answer of 400 or more. Any other error or warning fails, and so does a named one that stops coming.
 */
export async function expectQuietConsole(
  page: Page,
  watch: PageWatch,
  expected: { warnings?: readonly string[]; errors?: readonly string[] } = {}
): Promise<void> {
  const probe = "nerve-e2e: console probe";
  await page.evaluate((text) => {
    console.error(text);
    console.warn(text);
  }, probe);
  await expect
    .poll(() => ({ errors: watch.consoleErrors, warnings: watch.consoleWarnings }))
    .toEqual({ errors: [...(expected.errors ?? []), probe], warnings: [...(expected.warnings ?? []), probe] });
}
