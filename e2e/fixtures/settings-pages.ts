import { errors, expect, type Locator, type Page, type Response, type Route } from "@playwright/test";

import type { Api } from "./api";
import { bearer, register, type AuthTokens } from "./auth";
import { deferred } from "./deferred";

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
 * The bodies, as JSON, of the requests of method to path (no query) that page sends from now on, in the order they
 * leave it: the array returned grows as they do. Each body is read as its request leaves page, on its way (a route):
 * the request of a response does not have the bodies the web app's client sends. The route matches path by a
 * pattern, so it stops no other request, and stays for the page's life: a route removed while the page sends its next
 * request can leave that request waiting for good.
 */
export async function bodiesSentTo(page: Page, method: string, path: string): Promise<unknown[]> {
  const bodies: unknown[] = [];
  await page.route(`**${path}`, async (route: Route) => {
    if (route.request().method() === method) bodies.push(route.request().postDataJSON());
    await route.fallback();
  });
  return bodies;
}

/**
 * Resolves with the body, as JSON, of the request of method to path (no query) that act makes page send, read as it
 * leaves page (bodiesSentTo), and nerve's answer to it.
 */
export async function sentTo(
  page: Page,
  method: string,
  path: string,
  act: () => Promise<void>
): Promise<{ body: unknown; answer: Response }> {
  const bodies = await bodiesSentTo(page, method, path);
  const answer = await answerTo(page, method, path, act);
  return { body: bodies[0], answer };
}

/**
 * Holds from page nerve's answer to the next request of method to path that page sends: the request reaches nerve
 * at once, and page gets the answer only when the function returned is called. Later requests pass.
 */
export async function holdAnswer(page: Page, method: string, path: string): Promise<() => Promise<void>> {
  const released = deferred();
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
      await released.promise;
      await route.fulfill({ response });
    }
  );
  return async () => released.resolve();
}

/**
 * Holds from page nerve's answer to the request of method to path that act makes page send (holdAnswer): resolves once
 * the request has left page, with its body (bodiesSentTo), and the function that lets the answer through, which
 * resolves with it once page has it.
 */
export async function sentHeld(
  page: Page,
  method: string,
  path: string,
  act: () => Promise<void>
): Promise<{ body: unknown; release: () => Promise<Response> }> {
  const release = await holdAnswer(page, method, path);
  // registered after the hold, so that it reads the request first, on its way to the hold
  const bodies = await bodiesSentTo(page, method, path);
  await act();
  await expect.poll(() => bodies.length, { message: `${method} ${path} sent` }).toBeGreaterThan(0);
  return { body: bodies[0], release: () => answerTo(page, method, path, release) };
}

/**
 * Holds every script page asks for from now on, until release is called: a navigation to a page whose code the app has
 * not loaded yet waits for it. requested resolves once page has asked for one, the navigation under way.
 */
export async function holdScripts(page: Page): Promise<{ requested: Promise<void>; release: () => void }> {
  const released = deferred();
  const requested = deferred();
  await page.route(
    (url) => url.pathname.endsWith(".js"),
    async (route: Route) => {
      requested.resolve();
      await released.promise;
      await route.fallback();
    }
  );
  return { requested: requested.promise, release: released.resolve };
}

/**
 * What a wait of a second that failed says: false when the second ran out (what was waited for did not come); any
 * other failure, such as a locator that matches several elements, rejects, so that it cannot read as "it did not".
 */
function timedOut(error: unknown): false {
  if (error instanceof errors.TimeoutError) return false;
  throw error;
}

/**
 * Resolves with whether button is enabled within a second: one that stays disabled while the test holds what its page
 * waits for keeps it disabled the whole time, where a check made at once could pass before the page has re-rendered.
 * button is one element with the button role: a locator that matches none, several or something else fails, not as
 * "stayed disabled". Enabled is as Playwright's role queries take it (getByRole's disabled).
 */
export async function enabledWithin(button: Locator): Promise<boolean> {
  await expect(button, "the one button").toHaveRole("button");
  return button
    .and(button.page().getByRole("button", { disabled: false }))
    .waitFor({ state: "attached", timeout: 1_000 })
    .then(() => true, timedOut);
}

/**
 * Resolves with whether locator, one element, shows within a second: what must not show while the test holds what its
 * page waits for stays hidden the whole time, where a check made at once could pass before the page has rendered it.
 * Only the second running out means it did not show (timedOut); any other failure, such as a locator that matches
 * several elements, rejects.
 */
export async function shownWithin(locator: Locator): Promise<boolean> {
  return locator.waitFor({ state: "visible", timeout: 1_000 }).then(() => true, timedOut);
}

/** Resolves with whether locator, one element, hides within a second, as a panel that closes does: shownWithin's twin. */
export async function hiddenWithin(locator: Locator): Promise<boolean> {
  return locator.waitFor({ state: "hidden", timeout: 1_000 }).then(() => true, timedOut);
}

/**
 * Resolves with whether the one modal dialog page shows closes within a second, such as one that should stay once
 * nerve has refused what it sent. A toast is a dialog too, not a modal one: it does not count. A dialog that closes
 * still shows its content while it fades out (ModalCore's leave transition, 200 ms), so a check made at once cannot
 * tell it from one that stays: the second outlasts the transition. Only the second running out means it stayed
 * (timedOut); any other failure, such as two modal dialogs, rejects.
 */
export async function closedWithin(page: Page): Promise<boolean> {
  return page
    .locator('[role="dialog"][aria-modal="true"]')
    .waitFor({ state: "detached", timeout: 1_000 })
    .then(() => true, timedOut);
}

/** Presses Escape on page, which shows one modal dialog, and resolves with whether that dialog closed (closedWithin). */
export async function closedByEscape(page: Page): Promise<boolean> {
  const closed = closedWithin(page);
  await page.keyboard.press("Escape");
  return closed;
}

/**
 * Dispatches on field what the browser gives the page when an Escape ends an input method's composition (Chinese,
 * say): the composition's start, the Escape's keydown while composing, and the composition's end. A dropdown that
 * tracks the composition by its events (Base UI's popover) sees one, as a page does with a real input method.
 */
export async function composingEscape(field: Locator): Promise<void> {
  await field.dispatchEvent("compositionstart");
  await field.dispatchEvent("keydown", { key: "Escape", isComposing: true });
  await field.dispatchEvent("compositionend");
}

/**
 * Gives field, in the one modal dialog its page shows, an Escape that ends a composition (composingEscape), and
 * resolves with whether that dialog closed (closedWithin).
 */
export async function closedByComposingEscape(field: Locator): Promise<boolean> {
  const closed = closedWithin(field.page());
  await composingEscape(field);
  return closed;
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
 * Waits for the transitions of what locator finds, and of what it holds, to end (a modal's enter transition): those
 * that end, not an endless animation (a spinner's), which would never.
 */
export async function transitionsEnded(locator: Locator): Promise<void> {
  await locator.evaluate((element) =>
    Promise.all(
      element
        .getAnimations({ subtree: true })
        .filter((animation) => animation.effect?.getComputedTiming().endTime !== Infinity)
        .map((animation) => animation.finished)
    )
  );
}

/**
 * Counts the keydowns of key that reach page's document from now on, where a page-wide shortcut listens (a modal's
 * Escape, for one): the function returned gives the count so far.
 */
export async function keydownsReachingDocument(page: Page, key: string): Promise<() => Promise<number>> {
  const reached = await page.evaluateHandle((pressed) => {
    const count = { value: 0 };
    document.addEventListener("keydown", (event) => {
      if (event.key === pressed) count.value += 1;
    });
    return count;
  }, key);
  return async () => (await reached.jsonValue()).value;
}

/**
 * Moves page to path as a link of the app would, without a load: React Router follows the history's popstate. A
 * story's stand-in for a link the app does not have yet.
 */
export async function moveWithinApp(page: Page, path: string): Promise<void> {
  await page.evaluate((to) => {
    window.history.pushState(null, "", to);
    window.dispatchEvent(new PopStateEvent("popstate"));
  }, path);
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
 * The block of the field of id on the security or the general page: its heading, the field and the messages under
 * it. That is the innermost div around the field that holds a heading too, the last such div in document order. A
 * message in a toast or under another field is not in it.
 */
export function fieldBlock(page: Page, id: string): Locator {
  return page
    .locator("div", { has: page.locator(`#${id}`) })
    .filter({ has: page.getByRole("heading") })
    .last();
}
