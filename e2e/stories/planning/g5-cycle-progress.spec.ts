import { addProjectMembers, changeProject, inviteAndAccept } from "../../fixtures/api";
import { expectCountedInDatabase } from "../../fixtures/assert/progress";
import { newAccount } from "../../fixtures/auth";
import { createCycle, cycleCalls, listCycles } from "../../fixtures/cycles";
import {
  createIssue,
  issueArchived,
  issueDeleted,
  listIssues,
  updateIssue,
  type IssueCreate,
} from "../../fixtures/issues";
import {
  ascending,
  burndown,
  byAssignee,
  byLabel,
  counts,
  countsOf,
  cycleProgress,
  cycleProgressAnswer,
  progressWorld,
  type Counts,
  type Progress,
} from "../../fixtures/planning-progress";
import { expect, test } from "../../fixtures/test";

// G5, a cycle's progress (M6 design 3.5, M6 P4 spec 2.1, 2.3, R1, R3, R4, R9, R10): the counts of its work items by
// their states' groups, its distributions by assignee and label without the cancelled, its burndown on the project's
// calendar, and the counts of its list's row, all of the work items the caller sees. Web keeps the time of a zone
// where it is about noon as the story starts (noonZone), so today's completions fall on Web's today whenever it runs.

/** A row of a project's list of cycles as G5 compares it: its name, its counts, its assignees and its frozen counts. */
function row(name: string, c: Counts, assignees: string[], frozen: unknown = null) {
  return { name, counts: c, assignee_ids: assignees, progress_snapshot: frozen };
}

test("G5 (API): Sprint, from two days ago to two days ahead, has seven work items, a sub-item among them, of every group: its progress counts them, by assignee and by label each work item under each of its own, or none, the cancelled left out, and burns down from six, the cancelled out of the scope, by the two done today, with no number after today; archived, deleted and moved out ones are not counted, nor do Someday, a draft, and Other count Sprint's; the list's rows count as the progress, with the assignees of all the work items, the cancelled one's too; the database counts the same; walt, acme's admin and Web's guest, counts only his own two and sees no frozen progress until Web shows its guests every work item; mem reads alike with either token", async ({
  api,
  db,
}, testInfo) => {
  const { day, admin, mem, walt, slug, web, states, bug, ui, browser } = await progressWorld(api, testInfo);
  // gil, Web's member, is assigned only the cancelled work item.
  const gil = await newAccount(api, testInfo, "gil");
  await inviteAndAccept(api, admin.token, slug, gil, 15);
  await addProjectMembers(api, admin.token, web.id, [{ member_id: gil.id, role: 15 }]);
  const sprint = await createCycle(api, admin.token, web.id, { name: "Sprint", start_date: day(-2), end_date: day(2) });
  const other = await createCycle(api, admin.token, web.id, { name: "Other", start_date: day(5), end_date: day(7) });
  const someday = await createCycle(api, admin.token, web.id, { name: "Someday" });

  // In Sprint: Fix the login, mem's, Bug, to do, and its sub-item Fix the logout; Ship v1, done today, mem's and the
  // admin's, Bug and UI; Build the page, in progress, UI; Drop IE, cancelled, gil's; walt's Plan, in the backlog, and
  // Release, done today, the admin's. Old news, archived, Typo, deleted, and Later, moved to Other, are no longer
  // counted; Idea is in Someday.
  const item = (token: string, body: IssueCreate) => createIssue(api, token, web.id, { cycle_id: sprint.id, ...body });
  const fix = await item(admin.token, {
    name: "Fix the login",
    state_id: states.todo,
    assignee_ids: [mem.id],
    label_ids: [bug.id],
  });
  await item(browser, { name: "Fix the logout", state_id: states.todo, parent_id: fix.id });
  await item(admin.token, {
    name: "Ship v1",
    state_id: states.done,
    assignee_ids: [mem.id, admin.id],
    label_ids: [bug.id, ui.id],
  });
  await item(browser, { name: "Build the page", state_id: states.started, label_ids: [ui.id] });
  await item(admin.token, { name: "Drop IE", state_id: states.cancelled, assignee_ids: [gil.id] });
  await item(walt.token, { name: "Plan", state_id: states.backlog });
  await item(walt.token, { name: "Release", state_id: states.done, assignee_ids: [admin.id] });
  const old = await item(admin.token, { name: "Old news", state_id: states.done });
  const typo = await item(admin.token, { name: "Typo", state_id: states.todo });
  const later = await item(admin.token, { name: "Later", state_id: states.todo });
  await item(admin.token, { name: "Idea", cycle_id: someday.id });
  expect(
    [(await issueArchived(api, admin.token, old.id)).status, (await issueDeleted(api, admin.token, typo.id)).status],
    "the admin archives Old news and deletes Typo"
  ).toEqual([200, 204]);
  expect((await updateIssue(api, admin.token, later.id, { cycle_id: other.id })).cycle_id, "Later moved").toBe(
    other.id
  );

  // Sprint's progress as its members and the admin read it: seven work items; by assignee, Ship v1 under mem and the
  // admin, three without one, Drop IE nowhere; by label, Ship v1 under Bug and UI; six to burn, four left today.
  const everyone: Progress = {
    ...counts({ backlog: 1, unstarted: 2, started: 1, completed: 2, cancelled: 1 }),
    assignees: byAssignee([admin.id, 2, 0], [mem.id, 1, 1], [null, 0, 3]),
    labels: byLabel([bug.id, 1, 1], [ui.id, 1, 1], [null, 1, 2]),
    burndown: burndown(day(-2), day(2), 6, 6, 6, 4),
  };
  const read = (token: string) => cycleProgress(api, token, sprint.id);
  const seen = await read(browser);
  expect(seen, "Sprint's progress as mem reads it").toEqual({
    status: "current",
    live: everyone,
    progress_snapshot: null,
  });
  expect([await read(mem.token), await read(admin.token)], "the same with mem's token and to the admin").toEqual([
    seen,
    seen,
  ]);
  await expectCountedInDatabase(db, { cycle: sprint.id }, everyone);

  // Someday, a draft, has Idea, in the backlog, and no burndown; Other has Later.
  const idea: Progress = {
    ...counts({ backlog: 1 }),
    assignees: byAssignee([null, 0, 1]),
    labels: byLabel([null, 0, 1]),
    burndown: null,
  };
  expect(await cycleProgress(api, browser, someday.id), "Someday's progress").toEqual({
    status: "draft",
    live: idea,
    progress_snapshot: null,
  });
  await expectCountedInDatabase(db, { cycle: someday.id }, idea);

  // Web's list: each row counts as its progress, with the assignees of all its work items, cancelled Drop IE's gil
  // too, and no frozen counts; the list engine lists as many work items of Sprint.
  const rows = async (token: string) =>
    (await listCycles(api, token, web.id)).map((c) => row(c.name, countsOf(c), c.assignee_ids, c.progress_snapshot));
  const listed = await rows(browser);
  expect(listed, "Web's cycles as mem lists them").toEqual([
    row("Someday", countsOf(idea), []),
    row("Other", counts({ unstarted: 1 }), []),
    row("Sprint", countsOf(everyone), ascending(admin.id, gil.id, mem.id)),
  ]);
  expect([await rows(mem.token), await rows(admin.token)], "the same with mem's token and to the admin").toEqual([
    listed,
    listed,
  ]);
  const listTotal = async (token: string) =>
    (await listIssues(api, token, web.id, { cycle_id: [sprint.id], sub_issue: "true", archived: "false" })).total;
  expect(await listTotal(browser), "Sprint's work items as mem lists them").toBe(everyone.total_issues);

  // walt sees only his own: Plan and Release; the same counts in his list, Release's assignee his row's only, and the
  // list engine's. No frozen progress is shown him.
  const walts: Progress = {
    ...counts({ backlog: 1, completed: 1 }),
    assignees: byAssignee([admin.id, 1, 0], [null, 0, 1]),
    labels: byLabel([null, 1, 1]),
    burndown: burndown(day(-2), day(2), 2, 2, 2, 1),
  };
  expect(await read(walt.token), "Sprint's progress as walt reads it").toEqual({
    status: "current",
    live: walts,
    progress_snapshot: null,
  });
  await expectCountedInDatabase(db, { cycle: sprint.id }, walts, walt.id);
  expect(await rows(walt.token), "Web's cycles as walt lists them").toEqual([
    row("Someday", counts({}), []),
    row("Other", counts({}), []),
    row("Sprint", countsOf(walts), [admin.id]),
  ]);
  expect(await listTotal(walt.token), "Sprint's work items as walt lists them").toBe(walts.total_issues);

  // Web shows its guests every work item: walt reads and lists as mem.
  expect(
    (await changeProject(api, admin.token, web.id, { guest_view_all_features: true })).status,
    "the admin shows Web's guests every work item"
  ).toBe(200);
  expect([await read(walt.token), await rows(walt.token)], "Sprint and Web's cycles to walt now").toEqual([
    seen,
    listed,
  ]);

  // A deleted cycle's progress is not found.
  expect(await cycleCalls(api).remove(admin.token, other.id), "the admin deletes Other").toEqual({ status: 204 });
  expect(await cycleProgressAnswer(api, browser, other.id), "Other's progress once deleted").toEqual({
    status: 404,
    code: "planning.cycle_not_found",
  });
});
