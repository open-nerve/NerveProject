import { expect, type Browser, type Locator, type Page, type Request, type TestInfo } from "@playwright/test";

import {
  addProjectMembers,
  changeProject,
  changeRole,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
  type Api,
  type Project,
} from "./api";
import { accountId, emailFor, login, type AuthTokens } from "./auth";
import { expectQuietConsole, watchPage } from "./browser";
import { registerOnboarded } from "./settings-pages";
import { anotherBrowser } from "./workspace-pages";

// A project's cycle pages (M6 P5a spec 2.5, 2.6, 2.9), as a person uses them: their addresses, what a story finds on
// them by role and name, a cycle's calendar, and the world of their stories. A page story and the API story of its
// number assert the same rows of the database (assert/planning.ts).

/** One of the world's people: his address, his id, and the tokens of his sign-in, which his first browser takes. */
interface Person {
  email: string;
  id: string;
  tokens: AuthTokens;
}

/** Signs label up for this run of the test, onboarded, and gives the person. */
async function person(api: Api, testInfo: TestInfo, label: string): Promise<Person> {
  const email = emailFor(testInfo, label);
  const tokens = await registerOnboarded(api, email);
  return { email, id: await accountId(api, tokens.access_token), tokens };
}

/**
 * Acme's world of the cycle pages' stories: the admin's workspace Acme and its project Web; mem and nia, members of
 * both; gus, a guest of both; and wanda, Acme's admin and Web's guest. Web's guests do not see every work item (M4
 * design 3.4): nerve shows wanda only her own there, so she writes none of its cycles, as gus does not (P5a fix F1,
 * spec R8). Web's cycles are on unless cyclesOn is false: a new project's are off.
 */
export async function cyclePagesWorld(api: Api, testInfo: TestInfo, cyclesOn = true) {
  const [admin, mem, nia, gus, wanda] = await Promise.all(
    ["admin", "mem", "nia", "gus", "wanda"].map((label) => person(api, testInfo, label))
  );
  if (!admin || !mem || !nia || !gus || !wanda) throw new Error("the world's people were not registered");
  const slug = slugFor(testInfo);
  const token = admin.tokens.access_token;
  await createWorkspace(api, token, { name: "Acme", slug });
  await (
    [
      [mem, 15],
      [nia, 15],
      [gus, 5],
      [wanda, 15],
    ] as const
  ).reduce(async (previous, [one, role]) => {
    await previous;
    await inviteAndAccept(api, token, slug, { email: one.email, token: one.tokens.access_token }, role);
  }, Promise.resolve());
  const web = await createProject(api, token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, token, web.id, [
    { member_id: mem.id, role: 15 },
    { member_id: nia.id, role: 15 },
    { member_id: gus.id, role: 5 },
    { member_id: wanda.id, role: 5 },
  ]);
  await changeRole(api, token, slug, wanda.id, 20);
  if (cyclesOn) await turnCycles(api, token, web, true);
  return { admin, mem, nia, gus, wanda, slug, web, token };
}

/** Turns the cycles of project on or off, as its admin of token. */
export async function turnCycles(api: Api, token: string, project: Project, on: boolean): Promise<void> {
  const turned = await changeProject(api, token, project.id, { cycle_view: on });
  expect(turned.status, `${project.name}'s cycles turned ${on ? "on" : "off"}: ${JSON.stringify(turned)}`).toBe(200);
}

/** The address of the cycles list of the project of projectId in the workspace of slug. */
export function cyclesPath(slug: string, projectId: string): string {
  return `/${slug}/projects/${projectId}/cycles`;
}

/** The address of the page of the cycle of id of that project. */
export function cyclePath(slug: string, projectId: string, id: string): string {
  return `${cyclesPath(slug, projectId)}/${id}`;
}

/** The address of that project's archive of cycles. */
export function archivedCyclesPath(slug: string, projectId: string): string {
  return `/${slug}/projects/${projectId}/archives/cycles`;
}

/**
 * Goes to the cycles list of the project of projectId, which page shows, by its link in the sidebar's project: within
 * the app, as a person does, not by a load of another document a moment after this one's (Firefox then serves the
 * fonts the document before loaded, and reports the font its root preloads as unused).
 */
export async function openCyclesList(page: Page, slug: string, projectId: string): Promise<void> {
  const list = cyclesPath(slug, projectId);
  await page
    .getByRole("link", { name: "Cycles", exact: true })
    .and(page.locator(`a[href="${list}"]`))
    .first()
    .click();
  await expect(page).toHaveURL(list);
}

/** The part of a cycles list named name: the card of the cycle in progress, or a section of the others. */
export function cycleSection(page: Page, name: "Active cycle" | "Upcoming cycle" | "Completed cycle"): Locator {
  return page.getByRole("region", { name, exact: true });
}

/** The heading of the section of name of a cycles list, which opens and closes it: named by its title and count. */
export function sectionToggle(page: Page, name: "Upcoming cycle" | "Completed cycle"): Locator {
  return cycleSection(page, name).getByRole("button", { name: new RegExp(`^${name}`) });
}

/**
 * The ids of the cycles whose pages the links within lead to, each once, in the order of the links: a list's rows, an
 * archive's, the card's cycle.
 */
export async function linkedCycles(within: Locator): Promise<string[]> {
  const hrefs = await within
    .locator('a[href*="/cycles/"]')
    .evaluateAll((links) => links.map((link) => link.getAttribute("href") ?? ""));
  return [...new Set(hrefs.map((href) => href.slice(href.lastIndexOf("/") + 1)))];
}

/** The row of the cycle of id, named name, on page's list or archive: its link, its facts and its menu. */
export function cycleRow(page: Page, id: string, name: string): Locator {
  return page
    .locator("div.relative")
    .filter({ has: page.locator(`a[href$="/cycles/${id}"]`) })
    .filter({ has: cycleMenu(page, name) })
    .last();
}

/** The button of the menu of the cycle of name: named by the cycle (common.item_options). */
export function cycleMenu(page: Page, name: string): Locator {
  return page.getByRole("button", { name: `${name} options`, exact: true });
}

/**
 * The entries of the menu of the cycle of name, which page shows, as the menu offers them: each by its title, marked
 * "(disabled)" when it may not be chosen. The menu is closed again by Escape.
 */
export async function menuEntries(page: Page, name: string): Promise<string[]> {
  await cycleMenu(page, name).click();
  const menu = page.getByRole("menu");
  await expect(menu.getByRole("menuitem").first()).toBeVisible();
  const entries = await menu
    .getByRole("menuitem")
    .evaluateAll((items) =>
      items.map(
        (item) =>
          `${item.querySelector("h5")?.textContent ?? ""}${item.matches(":disabled,[aria-disabled=true]") ? " (disabled)" : ""}`
      )
    );
  await page.keyboard.press("Escape");
  await expect(menu).toHaveCount(0);
  return entries;
}

/** Chooses entry from the menu of the cycle of name, which page shows. */
export async function chooseFromMenu(page: Page, name: string, entry: string): Promise<void> {
  await cycleMenu(page, name).click();
  await page.getByRole("menu").getByRole("menuitem", { name: entry, exact: true }).click();
}

/** The calendar a cycle's days open in (DateRangeDropdown's), named as its button is. */
export function calendarOf(page: Page): Locator {
  return page.getByRole("dialog", { name: "Date range", exact: true });
}

/**
 * Opens the calendar of button and picks days, YYYY-MM-DD each, one after another, as a range's days are picked: each
 * in its month, chosen by the calendar's year and month. The calendar stays open.
 */
export async function pickDays(page: Page, button: Locator, days: readonly string[]): Promise<void> {
  await button.click();
  const calendar = calendarOf(page);
  await expect(calendar).toBeVisible();
  await days.reduce(async (previous, day) => {
    await previous;
    const [year = "", month = ""] = day.split("-");
    await calendar.getByRole("combobox", { name: "Choose the Year" }).selectOption(year);
    await calendar.getByRole("combobox", { name: "Choose the Month" }).selectOption(String(Number(month) - 1));
    await calendar.locator(`[data-day="${day}"] button`).click();
  }, Promise.resolve());
}

/** A month's short name, by its number. */
const MONTHS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

/**
 * The days start to end, YYYY-MM-DD each, as a cycle's row and its sidebar show them (MergedDateDisplay): the days of
 * the project's calendar, whatever the browser's time zone (R6).
 */
export function shownDays(start: string, end: string): string {
  const [sy = "", sm = "", sd = ""] = start.split("-");
  const [ey = "", em = "", ed = ""] = end.split("-");
  const [smon, emon] = [MONTHS[Number(sm) - 1], MONTHS[Number(em) - 1]];
  if (sy !== ey) return `${smon} ${sd}, ${sy} - ${emon} ${ed}, ${ey}`;
  return sm === em ? `${smon} ${sd} - ${ed}, ${sy}` : `${smon} ${sd} - ${emon} ${ed}, ${sy}`;
}

/** A day, YYYY-MM-DD, as a cycle's form shows it (renderFormattedDate): "Mar 01, 2031". */
export function shownDay(day: string): string {
  const [year = "", month = "", date = ""] = day.split("-");
  return `${MONTHS[Number(month) - 1]} ${date}, ${year}`;
}

/** The notification that says text, among the page's toasts. */
export function toast(page: Page, text: string): Locator {
  return page.getByLabel("Notifications", { exact: true }).getByText(text, { exact: true });
}

/**
 * Closes every toast page shows by its close button, one after another, each gone before the next: a story that then
 * checks that a toast does not come cannot see one of before. A toast's close button is hidden from assistive
 * technology (its toast is read without it), so it is found by its label.
 */
export async function dismissToasts(page: Page): Promise<void> {
  const close = page.getByLabel("Notifications", { exact: true }).locator('button[aria-label="Close"]');
  const shown = await close.count();
  if (shown === 0) return;
  await close.first().click();
  await expect(close).toHaveCount(shown - 1);
  return dismissToasts(page);
}

/** What each status of an answer is called in the console's report of it. */
const STATUS_TEXTS: Record<number, string> = {
  403: "Forbidden",
  404: "Not Found",
  409: "Conflict",
  422: "Unprocessable Entity",
};

/**
 * What browserName logs of answers of statuses, in order: Chromium and WebKit report each answer of 400 or more,
 * Firefox none.
 */
export function failedLoads(browserName: string, ...statuses: number[]): string[] {
  return browserName === "firefox"
    ? []
    : statuses.map(
        (status) => `Failed to load resource: the server responded with a status of ${status} (${STATUS_TEXTS[status]})`
      );
}

/** The key that moves the focus to the next control, buttons too: Option+Tab in WebKit on macOS, as in Safari. */
export function tabKey(browserName: string): string {
  return browserName === "webkit" && process.platform === "darwin" ? "Alt+Tab" : "Tab";
}

/**
 * Presses key (tabKey) on page until the focus is back where it was, at most presses times, and gives the stops in
 * order: each the focused element's aria-label, else its placeholder, else its text, marked "outside: " when within
 * does not hold it.
 */
export async function tabStops(page: Page, within: Locator, key: string, presses = 12): Promise<string[]> {
  const box = await within.elementHandle();
  const start = await page.evaluateHandle(() => document.activeElement);
  const from = async (stops: string[]): Promise<string[]> => {
    await page.keyboard.press(key);
    const { stop, back } = await page.evaluate(
      ({ inside, first }) => {
        const focused = document.activeElement;
        const name =
          focused?.getAttribute("aria-label") ||
          focused?.getAttribute("placeholder") ||
          focused?.textContent?.trim() ||
          focused?.tagName ||
          "";
        return { stop: inside?.contains(focused) ? name : `outside: ${name}`, back: focused === first };
      },
      { inside: box, first: start }
    );
    return back || stops.length === presses - 1 ? [...stops, stop] : from([...stops, stop]);
  };
  return from([]);
}

/**
 * Every API request page sends from now on, as "<method> <path>" with its query, as S2 records them: the function
 * returned gives them sorted, and how many have no answer yet.
 */
export function recordApi(page: Page): () => { pending: number; requests: string[] } {
  const sent = new Map<Request, { request: string; ended: boolean }>();
  page.on("request", (request) => {
    const { pathname, search } = new URL(request.url());
    if (pathname.startsWith("/api/"))
      sent.set(request, { request: `${request.method()} ${pathname}${search}`, ended: false });
  });
  const end = (request: Request) => {
    const record = sent.get(request);
    if (record) record.ended = true;
  };
  page.on("requestfinished", end);
  page.on("requestfailed", end);
  return () => ({
    pending: [...sent.values()].filter((record) => !record.ended).length,
    requests: [...sent.values()].map((record) => record.request).toSorted(),
  });
}

/**
 * Opens path in a browser of its own, signed in anew as the account of email (a session of its own: nerve rotates a
 * session's refresh token), runs check on its page, and checks that nothing failed there and its console stayed quiet.
 */
export async function seenBy(
  browser: Browser,
  baseURL: string,
  api: Api,
  email: string,
  path: string,
  check: (page: Page) => Promise<void>
): Promise<void> {
  const other = await anotherBrowser(browser, baseURL, await login(api, email));
  try {
    const watch = await watchPage(other.page);
    await other.page.goto(path);
    await check(other.page);
    expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors], `what failed for ${email}`).toEqual([
      [],
      [],
      [],
    ]);
    await expectQuietConsole(other.page, watch);
  } finally {
    await other.close();
  }
}
