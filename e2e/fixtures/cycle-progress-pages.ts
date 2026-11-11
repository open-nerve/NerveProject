import { expect, type Locator, type Page } from "@playwright/test";

import type { Api } from "./api";
import { bearer, login, type AuthTokens } from "./auth";

// A cycle's progress as its pages show it (M6 P5a spec 2.4, 2.5, 2.9): the card of the cycle in progress, a cycle's
// progress sidebar, its burndown chart, and the entries of a transfer, as a person finds them by role and name. The
// stories of G5–G7 and G19 build their world through the API (planning-progress.ts) and assert the database as their
// API versions do (assert/progress.ts, assert/issue-planning.ts).

/** Marks the accounts of tokens onboarded, as registerOnboarded does: the web app opens its pages to them. */
export async function onboard(api: Api, ...tokens: string[]): Promise<void> {
  await tokens.reduce(async (previous, token) => {
    await previous;
    const { response } = await api.PATCH("/api/v0/me/profile", {
      body: { is_onboarded: true },
      headers: bearer(token),
    });
    expect(response.status, "onboarded").toBe(200);
  }, Promise.resolve());
}

/** Signs the account of email in anew, onboarded, and gives the sign-in's tokens, which a browser takes. */
export async function pageSignIn(api: Api, email: string): Promise<AuthTokens> {
  const tokens = await login(api, email);
  await onboard(api, tokens.access_token);
  return tokens;
}

/**
 * The rows of the list items within, in their order, each as it reads: its texts, apart where they are apart in the
 * page, without what images and hidden marks hold (an avatar's initials, a color's dot). A breakdown's row reads
 * "max 50% of 2", a target of a transfer "Next In progress".
 */
export async function rowsOf(within: Locator): Promise<string[]> {
  return within.getByRole("listitem").evaluateAll((items) =>
    items.map((item) => {
      const walker = item.ownerDocument.createTreeWalker(item, NodeFilter.SHOW_TEXT);
      let text = "";
      let last: Element | null = null;
      for (let node = walker.nextNode(); node; node = walker.nextNode()) {
        const parent = node.parentElement;
        if (!parent || parent.closest('[role="img"], [aria-hidden="true"]')) continue;
        text += `${parent === last ? "" : " "}${node.textContent ?? ""}`;
        last = parent;
      }
      return text.replace(/\s+/g, " ").trim();
    })
  );
}

/** The points a burndown chart draws: of its line of the work left, of its ideal line, and whether each is finite. */
export interface BurndownDots {
  remaining: number;
  ideal: number;
  finite: boolean;
}

/**
 * The points the one burndown chart within draws (BurndownChart: recharts' areas, the work left first, the ideal line
 * second), each a dot at its day: a day after today has none on the line of the work left. Its dots show once its
 * areas have been drawn in, so a story polls it.
 */
export async function burndownDots(within: Locator): Promise<BurndownDots> {
  const [remaining = [], ideal = []] = await within
    .locator("g.recharts-area")
    .evaluateAll((areas) =>
      areas.map((area) =>
        Array.from(area.querySelectorAll("circle.recharts-area-dot"), (dot) =>
          ["cx", "cy"].map((axis) => Number.parseFloat(dot.getAttribute(axis) ?? "NaN"))
        )
      )
    );
  return {
    remaining: remaining.length,
    ideal: ideal.length,
    finite: [...remaining, ...ideal].flat().every((coordinate) => Number.isFinite(coordinate)),
  };
}

/** The value of the detail of title (Lead, Members, Work items) in the progress sidebar page shows. */
export function sidebarDetail(page: Page, title: "Lead" | "Members" | "Work items"): Locator {
  return page
    .locator("dl > div")
    .filter({ has: page.locator("dt", { hasText: new RegExp(`^${title}$`) }) })
    .locator("dd");
}

/** The names of the avatars within, sorted: the people a cycle's sidebar or row shows. */
export async function avatarNames(within: Locator): Promise<string[]> {
  return (
    await within
      .getByRole("img")
      .evaluateAll((avatars) => avatars.map((avatar) => avatar.getAttribute("aria-label") ?? ""))
  ).toSorted();
}

/** The way to transfer the unfinished work items of a cycle's row: "Transfer N work items" (TransferIssuesButton). */
export function transferEntry(row: Locator): Locator {
  return row.getByRole("button", { name: /^Transfer \d+ work items?$/ });
}

/** The banner of an ended cycle's page whose unfinished work items may be transferred, by its button. */
export function transferBanner(page: Page): Locator {
  return page.getByRole("button", { name: "Transfer work items", exact: true });
}
