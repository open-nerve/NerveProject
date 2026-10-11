import type { Locator, Page, Response } from "@playwright/test";

import { answerTo, transitionsEnded } from "./settings-pages";
import { memberRow } from "./workspace-pages";

// A project's pages (M3 design 7.6), as a person uses them.

/** A request the page sends: its method and its path, as answerTo and holdAnswer take them. */
type Sent = { method: string; path: string };

/** The request that removes the project membership of id. */
export function removalOf(id: string): Sent {
  return { method: "DELETE", path: `/api/v0/project-members/${id}` };
}

/** The request that ends the caller's membership of the project of id. */
export function leavingOf(id: string): Sent {
  return { method: "POST", path: `/api/v0/projects/${id}/leave` };
}

/**
 * Ends a membership from a project's members page, which page shows: opens the menu of the row of email, picks its
 * entry (Leave on the caller's own row, Remove on another's), and confirms it in the dialog that asks. Resolves with
 * nerve's answer to the request the page sends.
 */
export async function endProjectMembership(
  page: Page,
  email: string,
  entry: "Leave" | "Remove",
  request: Sent
): Promise<Response> {
  // the row's menu is its first button: the role's select, when there is one, comes after it
  await memberRow(page, email).locator("button").first().click();
  await page.getByRole("menuitem", { name: entry }).click();
  return answerTo(page, request.method, request.path, () =>
    page.getByRole("dialog").getByRole("button", { name: entry, exact: true }).click()
  );
}

/**
 * Presses Tab on page until the focus is back where it was, at most presses times, and gives the stops in order: each
 * the focused element's label, a field's id or a button's text, marked "outside: " when within does not hold it.
 */
export async function tabCycle(page: Page, within: Locator, presses = 40): Promise<string[]> {
  const box = await within.elementHandle();
  const start = await page.evaluateHandle(() => document.activeElement);
  // each press waits for the one before it: a stop after another
  const from = async (stops: string[]): Promise<string[]> => {
    await page.keyboard.press("Tab");
    const { stop, back } = await page.evaluate(
      ({ inside, first }) => {
        const focused = document.activeElement;
        const field = focused?.matches("input, textarea") ? focused.id : focused?.textContent?.trim();
        const name = focused?.getAttribute("aria-label") || field || focused?.tagName || "";
        return { stop: inside?.contains(focused) ? name : `outside: ${name}`, back: focused === first };
      },
      { inside: box, first: start }
    );
    return back || stops.length === presses - 1 ? [...stops, stop] : from([...stops, stop]);
  };
  return from([]);
}

/**
 * Drags item onto onto by its grip, which may show only while the pointer is over item, as a hand does: once page's
 * transitions have ended (one still under way, as when a toast comes in, can scroll the document, and a scroll between
 * the press and the move starts no drag), it presses on the grip, moves within item, which starts the drag, then over
 * onto, at its point at (its middle when left out), and lets go. Playwright's dragTo moves at once, which pragmatic-dnd
 * does not take for a drag.
 */
export async function dragOnto(
  page: Page,
  item: Locator,
  grip: Locator,
  onto: Locator,
  at?: { x: number; y: number }
): Promise<void> {
  await transitionsEnded(page.locator("body"));
  await item.hover();
  await grip.hover();
  await page.mouse.down();
  await item.hover({ position: { x: 60, y: 8 } });
  await onto.hover(at && { position: at });
  await page.mouse.up();
}
