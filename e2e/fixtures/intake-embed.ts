import { expect, type Locator, type Page } from "@playwright/test";

import { recordApi } from "./cycle-pages";
import { identifierOf } from "./intake-pages";
import type { Issue } from "./issues";
import { cardItems, cardSaying } from "./notification-pages";

// The notifications page's embed of a work item of a project's intake (M7 P5b spec 2.5, R16–R18, R26, R27, R30), as
// its H10 (page) stories use it: the preview and the embed's close in it, a card's main button, the list's region the
// focus lands on once the card is gone, and the opening of a card, which waits for every read it makes. The intake
// page's own helpers are intake-pages.ts's and intake-focus.ts's; the notifications page's, notification-pages.ts's.

/** The notifications page's preview: the region named so, beside the list. */
export function previewOf(page: Page): Locator {
  return page.getByRole("region", { name: "Preview", exact: true });
}

/** The embed's close: the button named Close inside the preview (a toast's close is named so too, re-95 m12). */
export function embedClose(page: Page): Locator {
  return previewOf(page).getByRole("button", { name: "Close", exact: true });
}

/** The main button (its first) of the card of the notifications page's list that says text of the work item issue. */
export function cardButton(page: Page, text: string, issue: Issue): Locator {
  return cardSaying(page, text)
    .filter({ hasText: `${identifierOf(issue)} ${issue.name}` })
    .getByRole("button")
    .first();
}

/**
 * The region of the notifications page's list (its element marked data-notification-list, unnamed): where the focus
 * lands as the preview leaves a notification whose card the list does not show (R26).
 */
export function listRegion(page: Page): Locator {
  return page.locator("[data-notification-list]");
}

/** The notifications the page's list shows, by their ids, in its order. */
export function listedIds(page: Page): Promise<(string | null)[]> {
  return cardItems(page).evaluateAll((cards) => cards.map((card) => card.getAttribute("data-notification-id")));
}

/**
 * Presses open (a card's main button), and resolves once the page has the answers to every read that follows (the
 * work item issue read gets times, its description's versions too when versions: its admin and its members read them,
 * a guest not), nothing pending, and has built the work item's description's editor: a lazy chunk, whose arrival moves
 * down what lies below it (a popover opened before it stays where it opened, over its button, t9-probe-5). Gives how
 * many times the page read the work item (GET /api/v0/issues/{id}).
 */
export async function opened(
  page: Page,
  open: Locator,
  issue: Issue,
  reads: { gets: number; versions: boolean }
): Promise<number> {
  const path = `/api/v0/issues/${issue.id}`;
  const requests = recordApi(page);
  let gets = 0;
  const answered = [
    page.waitForResponse(
      (r) => r.request().method() === "GET" && new URL(r.url()).pathname === path && ++gets === reads.gets
    ),
    ...(reads.versions
      ? [
          page.waitForResponse(
            (r) => gets >= reads.gets - 1 && new URL(r.url()).pathname === `${path}/description-versions`
          ),
        ]
      : []),
  ];
  await open.click();
  await Promise.all(answered);
  await expect(page.locator(`[id="editor-container-${issue.id}"] .ProseMirror`)).toBeVisible();
  await expect.poll(() => requests().pending, { message: `the reads of ${issue.name}` }).toBe(0);
  return requests().requests.filter((request) => request === `GET ${path}`).length;
}
