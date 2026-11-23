import { addProjectMembers, createLabel, createProject } from "../../fixtures/api";
import { intakeRowOf, triageActivities } from "../../fixtures/assert/intake";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { recordApi, seenBy } from "../../fixtures/cycle-pages";
import { submitIntakeIssue, triageIntakeIssue } from "../../fixtures/intake";
import {
  chooseOrder,
  demoteToGuest,
  detailReads,
  expectTabs,
  filterOption,
  intakeDetail,
  intakeEntry,
  intakeListRead,
  intakeListReads,
  intakePath,
  intakeRows,
  intakeWorld,
  openFilters,
  readsOf,
  shownRows,
  submitInOrder,
} from "../../fixtures/intake-pages";
import { APP, INTAKE_COUNTS, PROJECT, PROJECT_MEMBER, WORKSPACE, valued } from "../../fixtures/mounts";
import { zonedDay } from "../../fixtures/notification-pages";
import { moveWithinApp, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// H9's page version (M7 design 2, 3.5, 3.10; P5a spec 2.2, 2.5–2.7, 2.12, R6, R7, R9, R10, R21, R31; re-94 m9): a
// project's intake page. Its open tab lists the pending work items, the newest first, 50 at a time, the next 50 read by
// nerve's cursor only once the list is scrolled to its end; while chosen, it says how many its filters select; the
// closed tab lists the accepted, the declined and the duplicates, and never says how many. The page selects the first
// row of a list it shows with nothing selected, and reads its detail. Every role of the project is offered the intake
// in the sidebar, the pending work items of its intakes beside it, which a triage counts again; a guest who sees his own
// work items alone is listed and counted his own. Each filter and each order is one read of the list whose query says
// it exactly, its popover and its menu opening beside their buttons. A project whose intake is off shows the disabled
// page and reads no list. Opened anew, an empty intake reads the app's start, the workspace's with the pending counts,
// the project's and the list alone; a list with rows then reads the detail of the row it selects, which the stories
// wait for apart. The API version is h9-intake-list.spec.ts; the tests are four so that none takes long in Firefox.

/** The browser's time zone: east of UTC, so that a day's preset of the last days holds what was submitted just now. */
const ZONE = "Asia/Tokyo";

test.use({ timezoneId: ZONE });

/** The queries of the open and the closed tabs' lists as they open: their statuses, the newest first, 50 a page. */
const OPEN = "status=pending&order_by=-created_at&limit=50";
const CLOSED = "status=accepted,declined,duplicate&order_by=-created_at&limit=50";

/** What the open and the closed tabs say when they list nothing, and what a project whose intake is off says. */
const EMPTY = { open: "Log Intake requests", closed: "No request closed yet" };
const DISABLED = "Intake is not enabled for the project.";

/** The name of the row of the list that shows issue: its identifier in Web, and its name. */
function rowOf(issue: { sequence_id: number; name: string }): string {
  return `WEB-${issue.sequence_id} ${issue.name}`;
}

/** value, which the story made: undefined is the story's own mistake. */
function made<T>(value: T | undefined): T {
  if (value === undefined) throw new Error("the story did not make it");
  return value;
}

test("H9 (page): Web's empty intake, opened anew by alice, its admin, reads the app's start, the workspace's with the pending counts of its intakes, the project's and the open tab's list alone, and selects nothing; the closed tab reads its own list, the open tab no longer saying a total; bob, a member, and gus, a guest, are offered it in the sidebar too; Ops, its intake off, shows the disabled page, the way to turn it on alice's alone, and reads no list", async ({
  api,
  baseURL,
  browser,
  signedInPage,
}, testInfo) => {
  const { alice, bob, gus, slug, web } = await intakeWorld(api, testInfo);
  // Ops, alice's, of which bob is a member, its intake off as a new project's is.
  const ops = await createProject(api, alice.token, slug, { name: "Ops", identifier: "OPS" });
  await addProjectMembers(api, alice.token, ops.id, [{ member_id: bob.id, role: 15 }]);

  // Opened anew: the app's start, the workspace's, the pending counts (Web's intake is on), the project's, its own
  // resources and the open tab's list, nothing else; no row, so nothing selected and no detail read.
  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  const opened = recordApi(page);
  await page.goto(intakePath(slug, web.id));
  await expect(page.getByText(EMPTY.open, { exact: true })).toBeVisible();
  await expect.poll(opened).toEqual({
    pending: 0,
    requests: [
      ...[...APP, ...WORKSPACE, ...INTAKE_COUNTS, ...PROJECT, ...PROJECT_MEMBER].map((request) =>
        valued(request, { [slug]: "{slug}", [web.id]: "{project}" })
      ),
      intakeListRead(web.id, OPEN),
    ].toSorted(),
  });
  await expectTabs(page, "Open, 0 work items", "open");
  await expect(page.getByText("Select an Intake work item to view its details", { exact: true })).toBeVisible();
  await expect(intakeDetail(page)).toHaveCount(0);
  expect(new URL(page.url()).searchParams.get("inboxIssueId")).toBeNull();
  // Its entry in the sidebar, without a count: none pending.
  await expect(intakeEntry(page, slug, web.id)).toBeVisible();

  // The closed tab: its own list's read alone, its own empty state; the open tab says no total once not chosen.
  const closing = recordApi(page);
  await page.getByRole("tab", { name: "Closed", exact: true }).click();
  await expect.poll(closing).toEqual({ pending: 0, requests: [intakeListRead(web.id, CLOSED)] });
  await expect(page.getByText(EMPTY.closed, { exact: true })).toBeVisible();
  await expect(page.getByText(EMPTY.open, { exact: true })).toHaveCount(0);
  await expect(page).toHaveURL(/[?&]currentTab=closed(&|$)/);
  await expectTabs(page, "Open", "closed");

  // Ops' intake, its address followed within the app: the disabled page, its button alice's, and no list read.
  const moved = recordApi(page);
  await moveWithinApp(page, intakePath(slug, ops.id));
  await expect(page.getByText(DISABLED, { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Manage features", exact: true })).toBeEnabled();
  await expect.poll(() => moved().pending).toBe(0);
  expect(intakeListReads(moved().requests)).toEqual([]);

  // bob, a member: offered Web's intake and its submission; Ops' disabled page, its button not his, no entry for it in
  // the sidebar, no list read.
  await seenBy(browser, baseURL ?? "", api, bob.email, intakePath(slug, web.id), async (other, seen) => {
    await expect(other.getByText(EMPTY.open, { exact: true })).toBeVisible();
    await expect(intakeEntry(other, slug, web.id)).toBeVisible();
    await expect(other.getByRole("button", { name: "Add work item", exact: true })).toBeVisible();
    await moveWithinApp(other, intakePath(slug, ops.id));
    await expect(other.getByText(DISABLED, { exact: true })).toBeVisible();
    await expect(other.getByRole("button", { name: "Manage features", exact: true })).toBeDisabled();
    const sidebar = other.getByRole("complementary", { name: "Main sidebar", exact: true });
    await expect(
      sidebar
        .getByRole("link", { name: "Work items", exact: true })
        .and(other.locator(`a[href="/${slug}/projects/${ops.id}/issues"]`))
    ).toBeVisible();
    await expect(intakeEntry(other, slug, ops.id)).toHaveCount(0);
    expect(intakeListReads(seen.apiRequests).filter((request) => request.includes(ops.id))).toEqual([]);
  });
  // gus, a guest: offered Web's intake and its submission too.
  await seenBy(browser, baseURL ?? "", api, gus.email, intakePath(slug, web.id), async (other) => {
    await expect(other.getByText(EMPTY.open, { exact: true })).toBeVisible();
    await expect(intakeEntry(other, slug, web.id)).toBeVisible();
    await expect(other.getByRole("button", { name: "Add work item", exact: true })).toBeVisible();
  });

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H9 (page): Web's intake of 51 pending work items and one declined: the open tab says 51 and lists the newest 50, the first selected and its detail read; scrolled to its end, the list reads the next page by nerve's cursor, and only then; alice declines the one selected, and the sidebar's count, the open tab's total, the list and the selection follow; the closed tab lists the declined, without a total", async ({
  api,
  db,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  // bob's Old request, declined by alice; then his Request 01 to Request 51, one after another, pending.
  const old = await submitIntakeIssue(api, bob.token, web.id, { name: "Old request" });
  await triageIntakeIssue(api, alice.token, old.id, { status: "declined" });
  const requests = await submitInOrder(
    api,
    bob.token,
    web.id,
    Array.from({ length: 51 }, (_, index) => ({ name: `Request ${String(index + 1).padStart(2, "0")}` }))
  );
  const newest = requests.toReversed();
  const [selected, next] = [made(newest[0]), made(newest[1])];

  // Opened: the newest 50, the open tab's total and the sidebar's count 51; the first row selected, by the address,
  // and its detail read: the work item, its activity, comments, links, reactions, relations, sub-work items and
  // description versions, and the project's states again.
  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  const loaded = recordApi(page);
  const firstPage = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return url.pathname === `/api/v0/projects/${web.id}/intake-issues` && !url.searchParams.has("cursor");
  });
  await page.goto(intakePath(slug, web.id));
  const { next_cursor: cursor } = (await (await firstPage).json()) as { next_cursor: string | null };
  await expect(intakeRows(page)).toHaveCount(50);
  expect(await shownRows(page)).toEqual(newest.slice(0, 50).map(rowOf));
  await expectTabs(page, "Open, 51 work items", "open");
  await expect(intakeEntry(page, slug, web.id, 51)).toBeVisible();
  await expect(intakeRows(page).first()).toHaveAttribute("aria-current", "true");
  await expect(page).toHaveURL(new RegExp(`[?&]inboxIssueId=${selected.id}(&|$)`));
  await expect.poll(() => readsOf(loaded().requests, selected.id)).toEqual(detailReads(selected.id));
  const states = `GET /api/v0/projects/${web.id}/states`;
  await expect.poll(() => loaded().requests.filter((request) => request === states)).toEqual([states, states]);
  await expect.poll(() => loaded().pending).toBe(0);
  expect(intakeListReads(loaded().requests)).toEqual([intakeListRead(web.id, OPEN)]);

  // Scrolled to the end by the wheel, as a person does: the next page, by nerve's cursor, and the 51st row.
  expect(cursor, "the first page's cursor").not.toBeNull();
  // (Firefox scrolls a wheel's turn a few rows at most: the wheel turns on until the list has read on.)
  await intakeRows(page).first().hover();
  await expect(async () => {
    await page.mouse.wheel(0, 2_000);
    await expect(intakeRows(page)).toHaveCount(51, { timeout: 250 });
  }).toPass({ timeout: 15_000 });
  await expect(intakeRows(page).last()).toHaveAccessibleName(rowOf(made(requests[0])));
  await expect
    .poll(() => intakeListReads(loaded().requests))
    .toEqual([
      intakeListRead(web.id, OPEN),
      intakeListRead(web.id, `${OPEN}&cursor=${encodeURIComponent(cursor ?? "")}`),
    ]);

  // alice declines the one selected: one triage, declined; the count and the total 50; the list without it; the next
  // row selected, by the address, and focused; its intake row declined by her, and one activity of hers.
  await intakeDetail(page).getByRole("button", { name: "Decline", exact: true }).click();
  const dialog = page.getByRole("dialog", { name: "Decline work item", exact: true });
  await expect(dialog).toContainText(`Are you sure you want to decline work item WEB-${selected.sequence_id}?`);
  const declined = await sentTo(page, "POST", `/api/v0/intake-issues/${selected.id}/triage`, () =>
    dialog.getByRole("button", { name: "Decline", exact: true }).click()
  );
  expect([declined.body, declined.answer.status()]).toEqual([{ status: "declined" }, 200]);
  await expect(intakeEntry(page, slug, web.id, 50)).toBeVisible();
  await expectTabs(page, "Open, 50 work items", "open");
  await expect(page).toHaveURL(new RegExp(`[?&]inboxIssueId=${next.id}(&|$)`));
  await expect(intakeRows(page)).toHaveCount(50);
  await expect(intakeRows(page).first()).toHaveAccessibleName(rowOf(next));
  await expect(intakeRows(page).first()).toBeFocused();
  expect(await intakeRowOf(db, selected.id)).toEqual({
    status: "declined",
    snoozed_till: null,
    duplicate_to_id: null,
    written_by: alice.id,
  });
  expect(await triageActivities(db, selected.id)).toEqual([
    { verb: "declined", old_value: "pending", new_value: "declined", actor: alice.id },
  ]);

  // The closed tab: its list's read, the declined two, the newest first; no total on either tab.
  const closing = recordApi(page);
  await page.getByRole("tab", { name: "Closed", exact: true }).click();
  await expect.poll(() => shownRows(page)).toEqual([rowOf(selected), rowOf(old)]);
  await expect.poll(() => intakeListReads(closing().requests)).toEqual([intakeListRead(web.id, CLOSED)]);
  await expectTabs(page, "Open", "closed");

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H9 (page): alice filters Web's intake by status, priority, assignee, creator, label, the days of submission and of the last change, and orders it, each a read of its list whose query says exactly that, with the rows it reads; the filters' popover and the order's menu open beside their buttons, wide and on a narrow screen with the list shown", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { alice, bob, slug, web } = await intakeWorld(api, testInfo);
  const bug = await createLabel(api, alice.token, web.id, { name: "Bug" });
  const ux = await createLabel(api, alice.token, web.id, { name: "UX" });
  // Crash on save, bob's, high, assigned to alice, a bug; Slow search, alice's, low, of UX; Typo in footer, bob's,
  // snoozed till tomorrow by alice: the open tab's first filters list the first two, the newest first.
  const crash = await submitIntakeIssue(api, bob.token, web.id, {
    name: "Crash on save",
    priority: "high",
    assignee_ids: [alice.id],
    label_ids: [bug.id],
  });
  const slow = await submitIntakeIssue(api, alice.token, web.id, {
    name: "Slow search",
    priority: "low",
    label_ids: [ux.id],
  });
  const typo = await submitIntakeIssue(api, bob.token, web.id, { name: "Typo in footer" });
  const tomorrow = new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString();
  await triageIntakeIssue(api, alice.token, typo.id, { status: "snoozed", snoozed_till: tomorrow });
  const [crashRow, slowRow, typoRow] = [rowOf(crash), rowOf(slow), rowOf(typo)];

  // Opened, the list of the open tab's first filters; its first row selected, its detail read and shown.
  const page = await signedInPage(alice.tokens);
  const watch = await watchPage(page);
  const loaded = recordApi(page);
  await page.goto(intakePath(slug, web.id));
  await expect.poll(() => shownRows(page)).toEqual([slowRow, crashRow]);
  await expect(intakeDetail(page).getByText("created the work item.")).toBeVisible();
  await expect.poll(() => loaded().pending).toBe(0);
  expect(readsOf(loaded().requests, slow.id)).toEqual(detailReads(slow.id));

  /** Runs act, then checks that the page read the list once, by query, and shows rows. */
  const asked = async (act: () => Promise<void>, query: string, rows: string[]) => {
    const sent = recordApi(page);
    await act();
    await expect.poll(() => shownRows(page)).toEqual(rows);
    await expect.poll(sent).toEqual({ pending: 0, requests: [intakeListRead(web.id, query)] });
  };
  /** Presses option of the filter titled title in the filters' popover, opened beside its button, then Escape. */
  const pick = async (title: string, option: string | RegExp) => {
    const panel = await openFilters(page);
    await filterOption(panel, title, option).click();
    await page.keyboard.press("Escape");
    await expect(panel).toBeHidden();
  };
  /** Removes the value named name from the filters applied, by its button. */
  const remove = (name: string) => page.getByRole("button", { name: `Remove ${name}`, exact: true }).click();
  const first = OPEN;

  // Each of the priority, assignee, creator and label filters, one value, then removed: the first list again.
  await asked(() => pick("Priority", "High"), "status=pending&priority=high&order_by=-created_at&limit=50", [crashRow]);
  await asked(() => remove("High"), first, [slowRow, crashRow]);
  await asked(
    () => pick("Assignees", /Alice$/),
    `status=pending&assignee_id=${alice.id}&order_by=-created_at&limit=50`,
    [crashRow]
  );
  await asked(() => remove("Alice"), first, [slowRow, crashRow]);
  await asked(
    () => pick("Created by", /Alice$/),
    `status=pending&created_by_id=${alice.id}&order_by=-created_at&limit=50`,
    [slowRow]
  );
  await asked(() => remove("Alice"), first, [slowRow, crashRow]);
  await asked(() => pick("Labels", "UX"), `status=pending&label_id=${ux.id}&order_by=-created_at&limit=50`, [slowRow]);
  await asked(() => remove("UX"), first, [slowRow, crashRow]);

  // The snoozed too, both statuses in one parameter; then the last 7 days of submission, days of Tokyo's calendar, and
  // a last change before today, by the custom dates' dialog: one range each, kept together.
  const statuses = "status=pending,snoozed";
  await asked(() => pick("Status", "Snoozed"), `${statuses}&order_by=-created_at&limit=50`, [
    typoRow,
    slowRow,
    crashRow,
  ]);
  const week = `created_at=${zonedDay(-7, ZONE)}..${zonedDay(0, ZONE)}`;
  await asked(() => pick("Created date", "Last 7 days"), `${statuses}&${week}&order_by=-created_at&limit=50`, [
    typoRow,
    slowRow,
    crashRow,
  ]);
  await asked(
    async () => {
      const panel = await openFilters(page);
      await filterOption(panel, "Last updated date", "Custom").click();
      const dialog = page.getByRole("dialog", { name: "Last updated date", exact: true });
      await dialog.getByRole("button", { name: "Last updated date range" }).click();
      await page.getByRole("option", { name: "Last updated date before" }).click();
      await dialog.getByRole("button", { name: "Apply", exact: true }).click();
      await expect(dialog).toHaveCount(0);
      await page.keyboard.press("Escape");
      await expect(panel).toBeHidden();
    },
    `${statuses}&${week}&updated_at=..${zonedDay(0, ZONE)}&order_by=-created_at&limit=50`,
    [typoRow, slowRow, crashRow]
  );

  // Clear all: the tab's first filters; then by number, the highest first, then the lowest.
  await asked(() => page.getByRole("button", { name: "Clear all", exact: true }).click(), first, [slowRow, crashRow]);
  await asked(() => chooseOrder(page, "ID"), "status=pending&order_by=-sequence_id&limit=50", [slowRow, crashRow]);
  await asked(() => chooseOrder(page, "Ascending"), "status=pending&order_by=sequence_id&limit=50", [
    crashRow,
    slowRow,
  ]);

  // On a narrow screen, the list shown over the detail: the popover and the menu open beside their buttons still.
  await page.setViewportSize({ width: 800, height: 720 });
  await expect(intakeRows(page).first()).toBeInViewport();
  await asked(() => pick("Priority", "High"), "status=pending&priority=high&order_by=sequence_id&limit=50", [crashRow]);
  await asked(() => chooseOrder(page, "Descending"), "status=pending&priority=high&order_by=-sequence_id&limit=50", [
    crashRow,
  ]);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});

test("H9 (page): odo, made Web's guest once he had submitted, sees his own work items of its intake alone, as nerve lists them by the same queries as anyone's: the open tab's total, its list and the sidebar's count his one pending, the closed tab his one declined; his detail reads no description versions", async ({
  api,
  signedInPage,
}, testInfo) => {
  const { alice, bob, odo, slug, web } = await intakeWorld(api, testInfo);
  // odo, a member still, submits two, bob two after him; alice declines the first of each; then odo is made a guest.
  const [typo, report] = await submitInOrder(api, odo.token, web.id, [
    { name: "Odo's typo" },
    { name: "Odo's report" },
  ]);
  const [older] = await submitInOrder(api, bob.token, web.id, [{ name: "Bob's old one" }, { name: "Bob's crash" }]);
  await triageIntakeIssue(api, alice.token, made(typo).id, { status: "declined" });
  await triageIntakeIssue(api, alice.token, made(older).id, { status: "declined" });
  await demoteToGuest(api, alice.token, web, odo.id);

  // His open tab: his pending one alone, said by the tab and by the sidebar; selected, its detail read, without the
  // description's versions, which a guest does not read.
  const page = await signedInPage(odo.tokens);
  const watch = await watchPage(page);
  const loaded = recordApi(page);
  await page.goto(intakePath(slug, web.id));
  await expect.poll(() => shownRows(page)).toEqual([rowOf(made(report))]);
  await expectTabs(page, "Open, 1 work item", "open");
  await expect(intakeEntry(page, slug, web.id, 1)).toBeVisible();
  await expect(intakeDetail(page).getByText("created the work item.")).toBeVisible();
  await expect.poll(() => loaded().pending).toBe(0);
  expect(readsOf(loaded().requests, made(report).id)).toEqual(detailReads(made(report).id, false));
  expect(intakeListReads(loaded().requests)).toEqual([intakeListRead(web.id, OPEN)]);

  // His closed tab: his declined one alone.
  const closing = recordApi(page);
  await page.getByRole("tab", { name: "Closed", exact: true }).click();
  await expect.poll(() => shownRows(page)).toEqual([rowOf(made(typo))]);
  await expect.poll(() => intakeListReads(closing().requests)).toEqual([intakeListRead(web.id, CLOSED)]);

  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[], [], []]);
  await expectQuietConsole(page, watch);
});
