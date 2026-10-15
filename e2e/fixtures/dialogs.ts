import { setTimeout as sleep } from "node:timers/promises";

import { errors, expect, type Download, type Locator, type Page, type Request } from "@playwright/test";

import { deferred } from "./deferred";

// A page's dialogs, and what a page does or shows within a second: what must not happen while the test holds what the
// page waits for stays undone the whole time, where a check made at once could pass before the page has done it.

/**
 * What a wait of a second that failed says: false when the second ran out (what was waited for did not come); any
 * other failure, such as a locator that matches several elements, rejects, so that it cannot read as "it did not".
 */
function timedOut(error: unknown): false {
  if (error instanceof errors.TimeoutError) return false;
  throw error;
}

/** An event of a page that a check waits for, the test of the one it waits for, and what makes the page act. */
type Watched<T> =
  | [event: "request", matches: (request: Request) => boolean, act: () => Promise<T>]
  | [event: "download", matches: (download: Download) => boolean, act: () => Promise<T>];

/**
 * Runs act, and resolves with what act resolved with and whether page emitted event, one that matches, while act ran
 * or within a second of its end: what page must not do, such as send a change its caller may not make or download a
 * secret answered after its page was left, stays undone the whole time, where a look made at once could come before
 * page has done it. It listens from before act runs, ends the second's timer as soon as the event comes, and stops
 * listening either way.
 */
async function happenedWithin<T>(
  page: Page,
  ...[event, matches, act]: Watched<T>
): Promise<{ result: T; happened: boolean }> {
  const happened = deferred<true>();
  let stopListening: () => void;
  if (event === "request") {
    const onRequest = (request: Request) => {
      if (matches(request)) happened.resolve(true);
    };
    page.on("request", onRequest);
    stopListening = () => page.off("request", onRequest);
  } else {
    const onDownload = (download: Download) => {
      if (matches(download)) happened.resolve(true);
    };
    page.on("download", onDownload);
    stopListening = () => page.off("download", onDownload);
  }
  const second = new AbortController();
  try {
    const result = await act();
    return { result, happened: await Promise.race([happened.promise, sleep(1_000, false, { signal: second.signal })]) };
  } finally {
    second.abort();
    stopListening();
  }
}

/**
 * Resolves with whether page sends a request of method to path (no query) while act runs or within a second of its
 * end (happenedWithin), such as a change its caller may not make.
 */
export async function sentWithin(page: Page, method: string, path: string, act: () => Promise<void>): Promise<boolean> {
  const sent = (request: Request) => request.method() === method && new URL(request.url()).pathname === path;
  return (await happenedWithin(page, "request", sent, act)).happened;
}

/**
 * Runs act, and resolves with what act resolved with and whether page began a download while act ran or within a
 * second of its end (happenedWithin), such as of the secret of a token answered after its page was left.
 */
export async function downloadedWithin<T>(
  page: Page,
  act: () => Promise<T>
): Promise<{ result: T; downloaded: boolean }> {
  const { result, happened } = await happenedWithin(page, "download", () => true, act);
  return { result, downloaded: happened };
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
 * The one modal dialog page shows: a toast is a dialog too, not a modal one, and does not count.
 */
export function modalDialog(page: Page): Locator {
  return page.locator('[role="dialog"][aria-modal="true"]');
}

/**
 * Resolves with whether the one modal dialog page shows closes within a second, such as one that should stay once
 * nerve has refused what it sent. A toast is a dialog too, not a modal one: it does not count. A dialog that closes
 * still shows its content while it fades out (ModalCore's leave transition, 200 ms), so a check made at once cannot
 * tell it from one that stays: the second outlasts the transition. Only the second running out means it stayed
 * (timedOut); any other failure, such as two modal dialogs, rejects.
 */
export async function closedWithin(page: Page): Promise<boolean> {
  return modalDialog(page)
    .waitFor({ state: "detached", timeout: 1_000 })
    .then(() => true, timedOut);
}

/**
 * Resolves with whether the page behind page's dialogs, its first main element, is inert and hidden from assistive
 * technology (aria-hidden), as ModalCore makes it while a dialog is open: nothing behind the dialog is reached by Tab
 * or the pointer, or read; and neither once the dialog has closed. The page is read by its element, not by a role:
 * Playwright's role queries do not see what is aria-hidden.
 */
export async function pageBehindHidden(page: Page): Promise<boolean> {
  return page
    .locator("main")
    .first()
    .evaluate((main) => main.closest("[inert]") !== null && main.closest('[aria-hidden="true"]') !== null);
}

/**
 * Resolves with whether the focus is within the one modal dialog page shows, on the dialog itself or on something in
 * it, and so not on the page's body: ModalCore keeps it there while a button pressed in the dialog is disabled under
 * it, its request out (M3 closeout, C11-3).
 */
export async function focusInDialog(page: Page): Promise<boolean> {
  return modalDialog(page).evaluate((dialog) => dialog.contains(dialog.ownerDocument.activeElement));
}

/**
 * Resolves with whether element, shown while a dialog is open, is read out: within a live region (a toast's), and not
 * in the page behind the dialog, which is inert and hidden then (pageBehindHidden). The toast that says why nerve
 * refused what a dialog sent is read while the dialog stays open (A-M2).
 */
export async function announced(element: Locator): Promise<boolean> {
  return element.evaluate(
    (shown) =>
      shown.closest("[aria-live]") !== null &&
      shown.closest("[inert]") === null &&
      shown.closest('[aria-hidden="true"]') === null
  );
}

/** Presses Escape on page, which shows one modal dialog, and resolves with whether that dialog closed (closedWithin). */
export async function closedByEscape(page: Page): Promise<boolean> {
  const closed = closedWithin(page);
  await page.keyboard.press("Escape");
  return closed;
}

/**
 * Clicks the backdrop of the one modal dialog page shows, at the viewport's top left corner (a dialog keeps at least
 * 16 px from each edge: ModalCore's positions), and resolves with whether that dialog closed (closedWithin).
 */
export async function closedByBackdropClick(page: Page): Promise<boolean> {
  const closed = closedWithin(page);
  await page.mouse.click(1, 1);
  return closed;
}

/**
 * Dispatches on field, at once, what the browsers give the page when a key (an Escape, a Backspace) ends an input
 * method's composition (Chinese, say), in both their orders: Chromium's and Firefox's, the key's keydown while
 * composing (isComposing), between the composition's start and end; then Safari's, the keydown after the
 * composition's end, isComposing false and keyCode 229. A dropdown that follows the composition by its events (Base
 * UI's popover) sees each, as a page does with a real input method.
 */
export async function composingKey(field: Locator, key: string): Promise<void> {
  await field.evaluate((element, pressed) => {
    const init = { bubbles: true, cancelable: true, composed: true };
    element.dispatchEvent(new CompositionEvent("compositionstart", init));
    element.dispatchEvent(new KeyboardEvent("keydown", { ...init, key: pressed, isComposing: true }));
    element.dispatchEvent(new CompositionEvent("compositionend", init));
    element.dispatchEvent(new CompositionEvent("compositionstart", init));
    element.dispatchEvent(new CompositionEvent("compositionend", init));
    element.dispatchEvent(new KeyboardEvent("keydown", { ...init, key: pressed, keyCode: 229 }));
  }, key);
}

/** An Escape that ends a composition on field (composingKey). */
export async function composingEscape(field: Locator): Promise<void> {
  await composingKey(field, "Escape");
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
