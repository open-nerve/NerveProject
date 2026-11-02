import { networkInterfaces } from "node:os";
import { expect, type Browser, type BrowserContext, type Locator, type Page, type TestInfo } from "@playwright/test";
import type { Database } from "./db";
import { startNerve } from "./server";
import type { PageWatch } from "./browser";

/** A real LAN HTTP origin, without localhost's secure-context exception or a clipboard stub. */
export async function collaborationLANPage(browser: Browser, db: Database, info: TestInfo) {
  const address = Object.values(networkInterfaces())
    .flat()
    .find((entry) => entry && !entry.internal && entry.family === "IPv4")?.address;
  if (!address) throw new Error("C8 requires a reachable non-loopback IPv4 interface");
  const nerve = await startNerve(db.url, info.outputPath("lan-http.log"), { NERVE_SERVER__ADDR: `${address}:0` });
  let context: BrowserContext | undefined;
  try {
    context = await browser.newContext({ baseURL: nerve.baseURL });
    const owned = context;
    const page = await owned.newPage();
    return {
      page,
      origin: nerve.baseURL,
      close: async () => {
        try {
          await owned.close();
        } finally {
          await nerve.stop();
        }
      },
    };
  } catch (error) {
    try {
      await context?.close();
    } finally {
      await nerve.stop();
    }
    throw error;
  }
}

/** Let the real transient gesture expire before activating the selected control; do not replace either clipboard API. */
export async function copyWithoutUserActivation(control: Locator) {
  return control.evaluate((element) => {
    if (!(element instanceof HTMLElement)) throw new Error("copy control is not an HTMLElement");
    return new Promise<{ secureContext: boolean; activeGesture: boolean; asyncClipboard: boolean }>((resolve) => {
      const activate = () => {
        if (navigator.userActivation.isActive) {
          requestAnimationFrame(activate);
          return;
        }
        const state = {
          secureContext: window.isSecureContext,
          activeGesture: navigator.userActivation.isActive,
          asyncClipboard: !!navigator.clipboard,
        };
        element.click();
        resolve(state);
      };
      requestAnimationFrame(activate);
    });
  });
}

/** Observe the native fallback's actual result without changing its implementation, arguments or return value. */
export async function observeLANClipboard(page: Page) {
  return page.evaluateHandle(() => {
    const native = document.execCommand;
    const calls: {
      command: string;
      returned: boolean;
      text: string | null;
      secureContext: boolean;
      activeGesture: boolean;
      asyncClipboard: boolean;
      diagnosticSource: string | null;
    }[] = [];
    document.execCommand = (...args: Parameters<typeof document.execCommand>) => {
      const context = {
        command: args[0],
        text: document.activeElement instanceof HTMLTextAreaElement ? document.activeElement.value : null,
        secureContext: window.isSecureContext,
        activeGesture: navigator.userActivation.isActive,
        asyncClipboard: !!navigator.clipboard,
        diagnosticSource: (new Error() as Error & { fileName?: string }).fileName ?? null,
      };
      const returned = native.apply(document, args);
      calls.push({ ...context, returned });
      return returned;
    };
    return { calls };
  });
}

/** Firefox's insecure-login warning is expected; a native denial additionally has its exact diagnostic. */
export function expectLANClipboardWarnings(
  watch: PageWatch,
  browserName: string,
  origin: string,
  loginPath: string,
  denied: boolean,
  diagnosticSource: string | null
) {
  if (browserName !== "firefox") return [];
  expect(watch.consoleWarnings).toHaveLength(denied ? 2 : 1);
  const [passwordWarning, clipboardWarning] = watch.consoleWarnings;
  expect(passwordWarning).toBe(
    `[JavaScript Warning: "Password fields present on an insecure (http://) page. This is a security risk that allows user login credentials to be stolen." {file: "${new URL(loginPath, origin).href}" line: 0}]`
  );
  if (!denied) return [passwordWarning!];
  const source = clipboardWarning?.match(
    /^\[JavaScript Warning: "document\.execCommand\(‘cut’\/‘copy’\) was denied because it was not called from inside a short running user-generated event handler\." \{file: "([^"]+)" line: [1-9]\d*\}\]$/
  );
  expect(source).not.toBeNull();
  // Firefox reports the eval source invoking the native API, not a sourceURL or the production caller.
  // Match the source observed at that exact call; keep the diagnostic text and warning count strict.
  expect(diagnosticSource).toMatch(/^debugger eval code line [1-9]\d* > eval$/);
  expect(source![1]).toBe(diagnosticSource);
  return [passwordWarning!, clipboardWarning!];
}
