import type { Page } from "@playwright/test";

import { addProjectMembers, changeProject, inviteAndAccept } from "../../fixtures/api";
import {
  cycleJoined,
  cycleMoved,
  expectCycleRows,
  expectPlanningActivity,
  lastWrite,
  workItemRows,
  wrote,
  type LastWrite,
} from "../../fixtures/assert/issue-planning";
import { expectCountedInDatabase, frozenSnapshot, snapshotOf } from "../../fixtures/assert/progress";
import { newAccount } from "../../fixtures/auth";
import { expectQuietConsole, watchPage } from "../../fixtures/browser";
import { cyclePath, cycleRow, cyclesPath, failedLoads, sectionToggle, seenBy, toast } from "../../fixtures/cycle-pages";
import {
  onboard,
  pageSignIn,
  rowsOf,
  sidebarDetail,
  transferBanner,
  transferEntry,
} from "../../fixtures/cycle-progress-pages";
import { createCycle, cycleCalls } from "../../fixtures/cycles";
import {
  announced,
  closedByEscape,
  closedWithin,
  enabledWithin,
  modalDialog,
  shownWithin,
} from "../../fixtures/dialogs";
import { createIssue, updateIssue, type Issue } from "../../fixtures/issues";
import {
  burndown,
  byAssignee,
  byLabel,
  counts,
  endCycle,
  progressWorld,
  type Progress,
} from "../../fixtures/planning-progress";
import { answerTo, sentHeld, sentTo } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

// G7's page version (M6 P5a spec 2.2, 2.5, 2.9, R8, R12): an ended cycle's unfinished work items transferred by its
// member in his browser, from its list's row and from its page's banner, to a cycle that has not ended. The API
// version is g7-cycle-transfer.spec.ts; both assert the same rows, activity and frozen column. Web keeps the time of
// a zone where it is about noon as the story starts (noonZone).

test("G7 (page): Sprint, ended, has Plan, walt's, in the backlog, Fix to do, Build in progress, Ship done and Drop cancelled; walt, acme's admin whom Web shows his own alone, and gus, Web's guest, shown every work item, have no transfer on its row or its page; mem's row offers to transfer its three unfinished, the targets Someday, Later and Next, none ended, archived or Sprint itself; Later, ended meanwhile, refuses them, the dialog staying with nerve's reason; Next takes them once nerve has, said with their count, each with its rows and activity, Ship and Drop staying; Ship to do again, Sprint's banner transfers it to Someday, the frozen progress the first", async ({
  api,
  baseURL,
  browser,
  browserName,
  db,
  signedInPage,
}, testInfo) => {
  // two other browsers, then mem's page
  test.setTimeout(60_000);
  const { day, admin, mem, walt, slug, web, states } = await progressWorld(api, testInfo);
  const gus = await newAccount(api, testInfo, "gus");
  await inviteAndAccept(api, admin.token, slug, gus, 5);
  await addProjectMembers(api, admin.token, web.id, [{ member_id: gus.id, role: 5 }]);
  await onboard(api, walt.token, gus.token);
  const sprint = await createCycle(api, admin.token, web.id, { name: "Sprint", start_date: day(-3), end_date: day(3) });

  // Plan, walt's, in the backlog; Fix, to do; Build, in progress; Ship, done; Drop, cancelled. mem puts each in Sprint.
  const items = [
    await createIssue(api, walt.token, web.id, { name: "Plan", state_id: states.backlog }),
    await createIssue(api, admin.token, web.id, { name: "Fix", state_id: states.todo }),
    await createIssue(api, admin.token, web.id, { name: "Build", state_id: states.started }),
    await createIssue(api, admin.token, web.id, { name: "Ship", state_id: states.done }),
    await createIssue(api, admin.token, web.id, { name: "Drop", state_id: states.cancelled }),
  ];
  const joined = new Map(
    await Promise.all(
      items.map(async (item): Promise<[string, LastWrite]> => {
        const { cycle_id } = await updateIssue(api, mem.token, item.id, { cycle_id: sprint.id });
        expect(cycle_id, `${item.name} in Sprint`).toBe(sprint.id);
        return [item.id, await lastWrite(db, item.id)];
      })
    )
  );
  const [plan, fix, build, ship, drop] = items as [Issue, Issue, Issue, Issue, Issue];
  const at = (item: Issue) => joined.get(item.id)?.at ?? "";

  // The admin ends Sprint; Next runs, Later is ahead, Someday a draft; Ended ended, Shelved ended and is archived.
  await endCycle(api, admin.token, sprint.id, day(-20), day(-14));
  const cycle = (name: string, start?: string, end?: string) =>
    createCycle(api, admin.token, web.id, start && end ? { name, start_date: start, end_date: end } : { name });
  const next = await cycle("Next", day(-1), day(4));
  const later = await cycle("Later", day(10), day(12));
  const someday = await cycle("Someday");
  const ended = await cycle("Ended", day(-40), day(-30));
  const shelved = await cycle("Shelved", day(-50), day(-45));
  expect(await cycleCalls(api).archive(admin.token, shelved.id), "the admin archives Shelved").toEqual({ status: 200 });
  const before: Progress = {
    ...counts({ backlog: 1, unstarted: 1, started: 1, completed: 1, cancelled: 1 }),
    assignees: byAssignee([null, 1, 3]),
    labels: byLabel([null, 1, 3]),
    burndown: burndown(day(-20), day(-14), 4, 4, 4, 4, 4, 4, 4, 4),
  };
  await expectCountedInDatabase(db, { cycle: sprint.id }, before);

  // Neither walt, whom Web shows his own Plan alone, unfinished, nor gus, Web's guest, once Web shows its guests every
  // work item, may transfer: no way on Sprint's row, nor a banner on its page. Their sidebars have no link to the
  // cycles: the list is loaded by its address.
  const noTransfer = (counted: string) => async (other: Page) => {
    await sectionToggle(other, "Completed cycle").click();
    const row = cycleRow(other, sprint.id, "Sprint");
    await expect(row.getByRole("progressbar", { name: "Progress" })).toBeVisible();
    await expect(transferEntry(row)).toHaveCount(0);
    await row.locator(`a[href="${cyclePath(slug, web.id, sprint.id)}"]`).click();
    await expect(sidebarDetail(other, "Work items")).toHaveText(counted);
    await expect(transferBanner(other)).toHaveCount(0);
    await expect(other.getByText("Completed cycles can no longer take work items.", { exact: true })).toHaveCount(0);
  };
  const path = cyclesPath(slug, web.id);
  await seenBy(browser, baseURL ?? "", api, walt.email, path, noTransfer("0/1"));
  const guestsSee = (on: boolean) => changeProject(api, admin.token, web.id, { guest_view_all_features: on });
  expect((await guestsSee(true)).status, "Web shows its guests every work item").toBe(200);
  await seenBy(browser, baseURL ?? "", api, gus.email, path, noTransfer("1/4"));
  expect((await guestsSee(false)).status, "Web shows its guests their own alone").toBe(200);

  // mem's row of Sprint offers to transfer its three unfinished; Ended, without work items, none. Its dialog, named by
  // its title, has a search and the cycles that may take them: Someday, a draft, Later, ahead, and Next, running; not
  // Ended, completed, Shelved, archived, or Sprint.
  const page = await signedInPage(await pageSignIn(api, mem.email));
  const watch = await watchPage(page);
  await page.goto(path);
  await sectionToggle(page, "Completed cycle").click();
  const row = cycleRow(page, sprint.id, "Sprint");
  const entry = transferEntry(row);
  await expect(entry).toHaveText("Transfer 3 work items");
  await expect(cycleRow(page, ended.id, "Ended").getByRole("progressbar", { name: "Progress" })).toBeVisible();
  await expect(transferEntry(cycleRow(page, ended.id, "Ended"))).toHaveCount(0);
  await entry.click();
  const dialog = modalDialog(page);
  await expect(dialog).toHaveAccessibleName("Transfer work items");
  await expect(dialog.getByRole("textbox", { name: "Search for a cycle", exact: true })).toBeVisible();
  await expect.poll(() => rowsOf(dialog)).toEqual(["Someday Draft", "Later Yet to start", "Next In progress"]);
  const target = (name: string) => dialog.getByRole("button", { name, exact: true });

  // Later ends meanwhile: nerve refuses it, said beside the dialog, which stays, never success; nothing is written.
  await endCycle(api, admin.token, later.id, day(-60), day(-55));
  const unchanged = await workItemRows(db, web.id);
  const stayed = await Promise.all([ship, drop].map((item) => lastWrite(db, item.id)));
  const transfer = `/api/v0/cycles/${sprint.id}/transfer-issues`;
  const refused = await sentTo(page, "POST", transfer, () => target("Later Yet to start").click());
  expect([refused.answer.status(), refused.body]).toEqual([409, { target_cycle_id: later.id }]);
  const reason = toast(page, "The cycle is completed: work items cannot join it, and only its order can change.");
  await expect(reason).toBeVisible();
  expect(await announced(reason)).toBe(true);
  expect(await closedWithin(page)).toBe(false);
  expect(await shownWithin(toast(page, "Work items transferred"))).toBe(false);
  expect(await workItemRows(db, web.id), "Web's work items' rows after the refusal").toEqual(unchanged);

  // To Next: the dialog waits for nerve, Escape not closing it nor another cycle to be chosen; once nerve has moved
  // the three, it closes and says how many, and the row's way goes with them.
  const moving = await sentHeld(page, "POST", transfer, () => target("Next In progress").click());
  expect(moving.body).toEqual({ target_cycle_id: next.id });
  expect(await closedByEscape(page)).toBe(false);
  expect(await enabledWithin(target("Someday Draft"))).toBe(false);
  expect((await moving.release()).status()).toBe(200);
  expect(await closedWithin(page)).toBe(true);
  await expect(toast(page, "3 work items were transferred to Next.")).toBeVisible();
  await expect(entry).toHaveCount(0);

  // Plan, Fix and Build moved by mem, each one's row of Sprint removed and one of Next added at one moment, with a
  // row of activity; Ship and Drop stay as they were. Sprint was frozen with the numbers before the move.
  const first = await lastWrite(db, plan.id);
  expect(first.by, "who moved Plan").toBe(mem.email);
  await Promise.all(
    [plan, fix, build].map(async (item) => {
      expect(await lastWrite(db, item.id), `${item.name}'s last write`).toEqual(first);
      await expectCycleRows(db, item.id, [
        { target: "Sprint", added: at(item), removed: first.at, adder: mem.email, by: mem.email },
        { target: "Next", added: first.at, removed: null, adder: mem.email, by: mem.email },
      ]);
      await expectPlanningActivity(db, item.id, [
        ...wrote(mem.email, at(item), cycleJoined("Sprint")),
        ...wrote(mem.email, first.at, cycleMoved("Sprint", "Next")),
      ]);
    })
  );
  expect(
    await Promise.all([ship, drop].map((item) => lastWrite(db, item.id))),
    "Ship's and Drop's last writes"
  ).toEqual(stayed);
  await Promise.all(
    [ship, drop].map(async (item) => {
      await expectCycleRows(db, item.id, [
        { target: "Sprint", added: at(item), removed: null, adder: mem.email, by: mem.email },
      ]);
      await expectPlanningActivity(db, item.id, wrote(mem.email, at(item), cycleJoined("Sprint")));
    })
  );
  const column = await frozenSnapshot(db, sprint.id);
  expect(column, "Sprint's frozen column").toEqual({ ...before, frozen_at: expect.any(String) });

  // Ship to do again: Sprint's page, from its row, has the banner, whose dialog no longer offers Later, ended; to
  // Someday, the one moved is said, and the banner goes. Ship's rows and activity are mem's; the frozen progress is
  // still the first.
  expect((await updateIssue(api, admin.token, ship.id, { state_id: states.todo })).state_id, "Ship to do").toBe(
    states.todo
  );
  await row.locator(`a[href="${cyclePath(slug, web.id, sprint.id)}"]`).click();
  await expect(page.getByText("Completed cycles can no longer take work items.", { exact: true })).toBeVisible();
  await transferBanner(page).click();
  await expect(dialog).toHaveAccessibleName("Transfer work items");
  await expect.poll(() => rowsOf(dialog)).toEqual(["Someday Draft", "Next In progress"]);
  expect((await answerTo(page, "POST", transfer, () => target("Someday Draft").click())).status()).toBe(200);
  expect(await closedWithin(page)).toBe(true);
  await expect(toast(page, "1 work item was transferred to Someday.")).toBeVisible();
  await expect(transferBanner(page)).toHaveCount(0);
  const second = await lastWrite(db, ship.id);
  expect(second.by, "who moved Ship").toBe(mem.email);
  await expectCycleRows(db, ship.id, [
    { target: "Sprint", added: at(ship), removed: second.at, adder: mem.email, by: mem.email },
    { target: "Someday", added: second.at, removed: null, adder: mem.email, by: mem.email },
  ]);
  await expectPlanningActivity(db, ship.id, [
    ...wrote(mem.email, at(ship), cycleJoined("Sprint")),
    ...wrote(mem.email, second.at, cycleMoved("Sprint", "Someday")),
  ]);
  expect(await snapshotOf(db, sprint.id), "Sprint's frozen column after the second transfer").toEqual(column);
  await expectCountedInDatabase(
    db,
    { cycle: someday.id },
    {
      ...counts({ unstarted: 1 }),
      assignees: byAssignee([null, 0, 1]),
      labels: byLabel([null, 0, 1]),
      burndown: null,
    }
  );
  expect([watch.apiFailures, watch.oldApiRequests, watch.pageErrors]).toEqual([[`409 POST ${transfer}`], [], []]);
  await expectQuietConsole(page, watch, { errors: failedLoads(browserName, 409) });
});
