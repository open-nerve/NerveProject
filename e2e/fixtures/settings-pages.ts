import { expect, type Locator, type Page, type Response, type Route } from "@playwright/test";

import type { Api } from "./api";
import { bearer, register, type AuthTokens } from "./auth";

// The personal settings (M2 design 7.7), as a person uses them. Each load of a settings page logs
// EMOJI_CHECK_WARNING (browser.ts), which the stories name.

/** Signs email up through the API, onboarded so that the settings open, and returns the session's tokens. */
export async function registerOnboarded(api: Api, email: string): Promise<AuthTokens> {
  const tokens = await register(api, email);
  const { response } = await api.PATCH("/api/v0/me/profile", {
    body: { is_onboarded: true },
    headers: bearer(tokens.access_token),
  });
  expect(response.status, "onboarded").toBe(200);
  return tokens;
}

/** Resolves with nerve's answer to the request of method to path that act makes page send. */
export async function answerTo(page: Page, method: string, path: string, act: () => Promise<void>): Promise<Response> {
  const [response] = await Promise.all([
    page.waitForResponse((res) => res.request().method() === method && new URL(res.url()).pathname === path, {
      timeout: 10_000,
    }),
    act(),
  ]);
  return response;
}

/**
 * Holds from page nerve's answer to the next request of method to path that page sends: the request reaches nerve
 * at once, and page gets the answer only when the function returned is called. Later requests pass.
 */
export async function holdAnswer(page: Page, method: string, path: string): Promise<() => Promise<void>> {
  let release!: () => void;
  const released = new Promise<void>((resolve) => {
    release = resolve;
  });
  let held = false;
  await page.route(
    (url) => url.pathname === path,
    async (route: Route) => {
      if (held || route.request().method() !== method) {
        await route.fallback();
        return;
      }
      held = true;
      const response = await route.fetch();
      await released;
      await route.fulfill({ response });
    }
  );
  return async () => release();
}

/**
 * The records of a CSV text (RFC 4180): fields apart by commas, each record ending with CRLF (the last one may
 * not), a field in double quotes holding commas, CR, LF and double quotes, each of those doubled.
 */
export function parseCsv(text: string): string[][] {
  const records: string[][] = [];
  let record: string[] = [];
  let field = "";
  let quoted = false;
  for (let i = 0; i < text.length; i++) {
    const c = text.charAt(i);
    if (quoted) {
      if (c !== '"') field += c;
      else if (text.charAt(i + 1) === '"') {
        field += '"';
        i++;
      } else quoted = false;
    } else if (c === '"' && field === "") quoted = true;
    else if (c === ",") {
      record.push(field);
      field = "";
    } else if (c === "\r" && text.charAt(i + 1) === "\n") {
      records.push([...record, field]);
      record = [];
      field = "";
      i++;
    } else field += c;
  }
  expect(quoted, "every quoted field of the CSV closed").toBe(false);
  if (field !== "" || record.length > 0) records.push([...record, field]);
  return records;
}

/**
 * The forms a token could take where it must not be: as it is, the secret after its prefix, and each of those and
 * the secret's bytes as hex, base64, base64 without its padding and base64url (which has none). Some coincide, such
 * as the bytes' base64url, which is the secret: each form is listed once, ten at most.
 */
function tokenForms(token: string): string[] {
  const secret = token.slice("nrv_pat_".length);
  const encoded = [Buffer.from(token), Buffer.from(secret), Buffer.from(secret, "base64url")].flatMap((data) => {
    const base64 = data.toString("base64");
    return [data.toString("hex"), base64, base64.replace(/=+$/, ""), data.toString("base64url")];
  });
  return [...new Set([token, secret, ...encoded])];
}

/**
 * Checks that page holds token in none of its forms (tokenForms): not in the document, the values of its form
 * controls (a value typed or set by script need not show in the document's markup), its address, its
 * localStorage, sessionStorage or cookies, nor in what the page logged, which log collects.
 */
export async function expectTokenGone(page: Page, token: string, log: readonly string[]): Promise<void> {
  const held = await page.evaluate(() => {
    const [local, session] = [localStorage, sessionStorage].map((area) =>
      Array.from({ length: area.length }, (_, i) => `${area.key(i)}=${area.getItem(area.key(i) ?? "")}`).join("\n")
    );
    const controls = document.querySelectorAll<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>(
      "input, textarea, select"
    );
    return {
      document: document.documentElement.outerHTML,
      formValues: Array.from(controls, (control) => control.value).join("\n"),
      address: window.location.href,
      localStorage: local ?? "",
      sessionStorage: session ?? "",
      cookie: document.cookie,
    };
  });
  const where = { ...held, console: log.join("\n") };
  for (const form of tokenForms(token)) {
    for (const [place, text] of Object.entries(where)) {
      expect(text.includes(form), `${place} holds ${form}`).toBe(false);
    }
  }
}

/**
 * Records each personal access token that page's current document holds from now on, even for a moment, as a check
 * that looks later would miss one that went meanwhile: the function returned gives them, each once, in order.
 */
export async function recordTokensShown(page: Page): Promise<() => Promise<string[]>> {
  await page.evaluate(() => {
    const shown: string[] = [];
    (window as unknown as { __nerveE2eTokensShown: string[] }).__nerveE2eTokensShown = shown;
    new MutationObserver(() => {
      for (const [token] of document.documentElement.outerHTML.matchAll(/nrv_pat_[\w-]+/g)) {
        if (!shown.includes(token)) shown.push(token);
      }
    }).observe(document, { subtree: true, childList: true, characterData: true, attributes: true });
  });
  return () => page.evaluate(() => (window as unknown as { __nerveE2eTokensShown: string[] }).__nerveE2eTokensShown);
}

/**
 * Checks that the one list open on page shows beside button, which opened it (M2 design 7.7): right under it or
 * over it, and lined up with it, not where the page begins.
 */
export async function expectListBesideButton(page: Page, button: Locator): Promise<void> {
  const list = page.getByRole("listbox");
  await expect(list).toBeVisible();
  const [at, beside] = await Promise.all([list.boundingBox(), button.boundingBox()]);
  expect(at, "the list's box").not.toBeNull();
  expect(beside, "the button's box").not.toBeNull();
  if (at === null || beside === null) return;
  // Under the button, or over it when the page has no room below (Popper's flip); the list's margin is 4 px.
  const under = Math.abs(at.y - (beside.y + beside.height));
  const over = Math.abs(beside.y - (at.y + at.height));
  expect(Math.min(under, over), "from the button to the list, under it or over it").toBeLessThan(8);
  // The left edges lined up (bottom-start, as the date's calendar opens) or the right edges (bottom-end, as the
  // settings' selects open).
  const left = Math.abs(at.x - beside.x);
  const right = Math.abs(at.x + at.width - (beside.x + beside.width));
  expect(Math.min(left, right), "between the left edges or the right edges").toBeLessThan(2);
}

/**
 * Fills the security page's form, which page shows, with the current password and a new one typed twice, and
 * submits it: resolves with the status of nerve's answer to the one change it sends.
 */
export async function submitPasswordChange(page: Page, current: string, next: string): Promise<number> {
  await page.locator("#old_password").fill(current);
  await page.locator("#new_password").fill(next);
  await page.locator("#confirm_password").fill(next);
  const answer = await answerTo(page, "POST", "/api/v0/me/change-password", () =>
    page.getByRole("button", { name: "Change password" }).click()
  );
  return answer.status();
}

/**
 * The block of the security page's field of id: its heading, the field and the messages under it. That is the
 * innermost div around the field that holds a heading too, the last such div in document order. A message in a
 * toast or under another field is not in it.
 */
export function fieldBlock(page: Page, id: string): Locator {
  return page
    .locator("div", { has: page.locator(`#${id}`) })
    .filter({ has: page.getByRole("heading") })
    .last();
}
