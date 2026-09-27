import { expect, type Page, type Request } from "@playwright/test";

/** What a page did that a story checks: its API calls, and what went wrong in it. */
export interface PageWatch {
  /** Every API request, as "<method> <path>". */
  readonly apiRequests: string[];
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
    apiFailures: [],
    oldApiRequests: [],
    pageErrors: [],
    consoleErrors: [],
    consoleWarnings: [],
    cspViolations: [],
  };
  page.on("request", (request) => {
    const path = apiPath(request);
    if (path === undefined) {
      return;
    }
    watch.apiRequests.push(`${request.method()} ${path}`);
    if (!path.startsWith("/api/v0/")) {
      watch.oldApiRequests.push(`${request.method()} ${path}`);
    }
  });
  page.on("response", (response) => {
    const path = apiPath(response.request());
    if (path !== undefined && response.status() >= 400) {
      watch.apiFailures.push(`${response.status()} ${response.request().method()} ${path}`);
    }
  });
  page.on("requestfailed", (request) => {
    const path = apiPath(request);
    if (path !== undefined) {
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
 * Checks that page logged no error and no warning since watch began (M2 design 9.6), such as React Router's
 * "navigate() should be called in useEffect". It logs a probe of each kind first and expects to find it,
 * so that a watch that does not hear the console cannot pass. thirdPartyWarnings are the warnings, not the
 * app's own, that page logs, in order, each named by the caller with where it comes from: another warning
 * fails, and so does a named one that stops coming.
 */
export async function expectQuietConsole(
  page: Page,
  watch: PageWatch,
  thirdPartyWarnings: readonly string[] = []
): Promise<void> {
  const probe = "nerve-e2e: console probe";
  await page.evaluate((text) => {
    console.error(text);
    console.warn(text);
  }, probe);
  await expect
    .poll(() => ({ errors: watch.consoleErrors, warnings: watch.consoleWarnings }))
    .toEqual({ errors: [probe], warnings: [...thirdPartyWarnings, probe] });
}
