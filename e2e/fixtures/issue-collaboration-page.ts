import { networkInterfaces } from "node:os";
import { expect, type Browser, type BrowserContext, type Locator, type TestInfo } from "@playwright/test";
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

/** Firefox must report both deliberately exercised security conditions; no other diagnostic is accepted. */
export function expectLANClipboardWarnings(watch: PageWatch, browserName: string, origin: string, loginPath: string) {
  if (browserName !== "firefox") return [];
  expect(watch.consoleWarnings).toHaveLength(2);
  const [passwordWarning, clipboardWarning] = watch.consoleWarnings;
  expect(passwordWarning).toBe(
    `[JavaScript Warning: "Password fields present on an insecure (http://) page. This is a security risk that allows user login credentials to be stolen." {file: "${new URL(loginPath, origin).href}" line: 0}]`
  );
  const source = clipboardWarning?.match(
    /^\[JavaScript Warning: "document\.execCommand\(‘cut’\/‘copy’\) was denied because it was not called from inside a short running user-generated event handler\." \{file: "([^"]+)" line: [1-9]\d*\}\]$/
  );
  expect(source).not.toBeNull();
  const script = new URL(source![1]!);
  expect(script.origin).toBe(new URL(origin).origin);
  expect(script.pathname).toMatch(/^\/assets\/[^/]+\.js$/);
  expect(watch.requestUrls).toContain(script.href);
  return [passwordWarning!, clipboardWarning!];
}
