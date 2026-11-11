import { createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { emailFor } from "../../fixtures/auth";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import {
  cycleRow,
  cycleSection,
  cyclesPath,
  linkedCycles,
  sectionToggle,
  shownDays,
  turnCycles,
} from "../../fixtures/cycle-pages";
import { createCycle } from "../../fixtures/cycles";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";
import { anotherBrowser } from "../../fixtures/workspace-pages";

// G2's page version (M6 P5a spec 2.5, 2.9, R5, R6): a project's cycles list in its parts by the status nerve gives
// each cycle, read from its days on the project's calendar day (M6 design 3.2), never from the browser's. The API
// version is g2-cycle-status.spec.ts; a status is not stored, so neither asserts rows of the database.

const shanghai = "Asia/Shanghai";

/** The calendar day, YYYY-MM-DD, that it is now in the time zone zone. */
function today(zone: string): string {
  return new Intl.DateTimeFormat("en-CA", { timeZone: zone, year: "numeric", month: "2-digit", day: "2-digit" }).format(
    new Date()
  );
}

/** The calendar day, YYYY-MM-DD, days after day (before it when negative). */
function addDays(day: string, days: number): string {
  const at = new Date(`${day}T00:00:00Z`);
  at.setUTCDate(at.getUTCDate() + days);
  return at.toISOString().slice(0, 10);
}

/**
 * A time zone whose calendar day is not Shanghai's (UTC+8) now, nor for an hour at least: Pago Pago's (UTC-11), the
 * day before until 19:00 in Shanghai, chosen before 18:00 there; Kiritimati's (UTC+14), the day after from 18:00 there
 * to Shanghai's midnight, chosen from 18:00.
 */
function anotherDayZone(): string {
  const hour = Number(
    new Intl.DateTimeFormat("en-GB", { timeZone: shanghai, hour: "2-digit", hourCycle: "h23" }).format(new Date())
  );
  return hour < 18 ? "Pacific/Pago_Pago" : "Pacific/Kiritimati";
}

test("G2 (page): Web keeps Shanghai's time and the admin's browser another day's; of Web's cycles, those of Shanghai's yesterday and of last year are completed, today's current, tomorrow's and one ten days on upcoming, Icebox and Backlog drafts; his list shows today's in the card of the cycle in progress, the upcoming ones by their first days then the drafts by name, and the completed ones, closed until opened, in nerve's order, each with its days of Web's calendar", async ({
  api,
  baseURL,
  browser,
}, testInfo) => {
  const tokens = await registerOnboarded(api, emailFor(testInfo, "admin"));
  const token = tokens.access_token;
  const slug = slugFor(testInfo);
  await createWorkspace(api, token, { name: "Acme", slug });
  const web = await createProject(api, token, slug, { name: "Web", identifier: "WEB", timezone: shanghai });
  await turnCycles(api, token, web, true);
  const day = today(shanghai);
  const zone = anotherDayZone();
  expect(today(zone), `the browser's day in ${zone}, Shanghai's ${day}`).not.toBe(day);

  // Created in this order, each before the one before it.
  const make = (name: string, start?: string, end?: string) =>
    createCycle(api, token, web.id, start && end ? { name, start_date: start, end_date: end } : { name });
  const lastYear = await make("Last year", "2025-01-01", "2025-01-14");
  const yesterday = await make("Yesterday", addDays(day, -1), addDays(day, -1));
  const current = await make("Today", day, day);
  const tomorrow = await make("Tomorrow", addDays(day, 1), addDays(day, 1));
  const later = await make("Later", addDays(day, 10), addDays(day, 20));
  const icebox = await make("Icebox");
  const backlog = await make("Backlog");
  expect(
    [lastYear, yesterday, current, tomorrow, later, icebox, backlog].map((c) => [c.name, c.status]),
    `the cycles' statuses on Shanghai's ${day}`
  ).toEqual([
    ["Last year", "completed"],
    ["Yesterday", "completed"],
    ["Today", "current"],
    ["Tomorrow", "upcoming"],
    ["Later", "upcoming"],
    ["Icebox", "draft"],
    ["Backlog", "draft"],
  ]);

  // The admin's browser keeps another day than Web's: by its own day, today's cycle would be upcoming or completed.
  const admin = await anotherBrowser(browser, baseURL ?? "", tokens, zone);
  const { page } = admin;
  const watch = await watchPage(page);
  await page.goto(cyclesPath(slug, web.id));
  expect(
    await page.evaluate(() =>
      new Intl.DateTimeFormat("en-CA", { year: "numeric", month: "2-digit", day: "2-digit" }).format(new Date())
    ),
    `the browser's own day in ${zone}`
  ).toBe(today(zone));

  // The card of the cycle in progress holds Today's; "Upcoming cycle", open, Tomorrow and Later by their first days,
  // then the drafts by name; "Completed cycle", closed, opens on Yesterday and Last year, in nerve's order.
  await expect.poll(() => linkedCycles(cycleSection(page, "Active cycle"))).toEqual([current.id]);
  await expect
    .poll(() => linkedCycles(cycleSection(page, "Upcoming cycle")))
    .toEqual([tomorrow.id, later.id, backlog.id, icebox.id]);
  const completed = sectionToggle(page, "Completed cycle");
  await expect(completed).toHaveAttribute("aria-expanded", "false");
  expect(await linkedCycles(cycleSection(page, "Completed cycle"))).toEqual([]);
  await completed.click();
  await expect(completed).toHaveAttribute("aria-expanded", "true");
  await expect.poll(() => linkedCycles(cycleSection(page, "Completed cycle"))).toEqual([yesterday.id, lastYear.id]);

  // Each row shows its days as Web's calendar has them, whatever the browser's day and zone.
  await Promise.all(
    [yesterday, current, tomorrow, later, lastYear].map(async (cycle) =>
      expect(cycleRow(page, cycle.id, cycle.name), cycle.name).toContainText(
        shownDays(cycle.start_date ?? "", cycle.end_date ?? "")
      )
    )
  );
  expect(today(shanghai), "Shanghai's day, unchanged while the list was read").toBe(day);
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
  await admin.close();
});
