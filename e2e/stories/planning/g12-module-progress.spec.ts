import { changeProject } from "../../fixtures/api";
import { databaseNow, expectCountedInDatabase, freezeRanSince } from "../../fixtures/assert/progress";
import { createIssue, issueArchived, updateIssue, type IssueCreate } from "../../fixtures/issues";
import { createModule, listModules, moduleCalls } from "../../fixtures/modules";
import {
  burndown,
  byAssignee,
  byLabel,
  counts,
  countsOf,
  moduleProgress,
  progressWorld,
  type Progress,
} from "../../fixtures/planning-progress";
import { expect, test } from "../../fixtures/test";

// G12, a module's progress (M6 design 3.5, M6 P4 spec 2.1, 2.3, R1, R3, R4, R9, R10): counted as a cycle's, its
// burndown from its start to its target day, and never frozen; a work item in several modules counts in each. Web
// keeps the time of a zone where it is about noon as the story starts (noonZone).

test("G12 (API): Core, from two days ago to two days ahead, has four work items, Fix also in Loose and Ship, done today, in Past: its progress counts them, by assignee and by label, the cancelled left out, and burns down from three to two today, nothing after; archived and removed ones are not counted; Loose, without a target day, and Bare, without days, have no burndown; Past, ended, counts live after a run of the periodic freeze as before it, archived too; the list's rows, and the archived catalog's, count as the progress; the database counts the same; walt, acme's admin and Web's guest, counts only his own until Web shows its guests every work item; mem reads alike with either token", async ({
  api,
  db,
}, testInfo) => {
  const { day, admin, mem, walt, web, states, bug, ui, browser } = await progressWorld(api, testInfo);
  const module = (name: string, body: { start_date?: string; target_date?: string; status?: "completed" } = {}) =>
    createModule(api, admin.token, web.id, { name, ...body });
  const core = await module("Core", { start_date: day(-2), target_date: day(2) });
  const loose = await module("Loose", { start_date: day(-2) });
  const bare = await module("Bare");
  const past = await module("Past", { start_date: day(-10), target_date: day(-5), status: "completed" });

  // Fix, to do, mem's, Bug, in Core and Loose; Ship, done today, mem's and the admin's, Bug and UI, in Core and Past;
  // Drop, cancelled, mem's; walt's Plan, in the backlog; Build, in progress, in Past alone. Old news, archived, and
  // Left, removed from Core, are not counted.
  const item = (token: string, body: IssueCreate) => createIssue(api, token, web.id, body);
  await item(admin.token, {
    name: "Fix",
    state_id: states.todo,
    assignee_ids: [mem.id],
    label_ids: [bug.id],
    module_ids: [core.id, loose.id],
  });
  const ship = await item(browser, {
    name: "Ship",
    state_id: states.done,
    assignee_ids: [mem.id, admin.id],
    label_ids: [bug.id, ui.id],
    module_ids: [core.id, past.id],
  });
  await item(admin.token, { name: "Drop", state_id: states.cancelled, assignee_ids: [mem.id], module_ids: [core.id] });
  await item(walt.token, { name: "Plan", state_id: states.backlog, module_ids: [core.id] });
  await item(admin.token, { name: "Build", state_id: states.started, module_ids: [past.id] });
  const old = await item(admin.token, { name: "Old news", state_id: states.done, module_ids: [core.id] });
  const left = await item(admin.token, { name: "Left", state_id: states.todo, module_ids: [core.id] });
  expect((await issueArchived(api, admin.token, old.id)).status, "the admin archives Old news").toBe(200);
  expect((await updateIssue(api, admin.token, left.id, { module_ids: [] })).module_ids, "Left removed").toEqual([]);

  // As mem and the admin read them, and as the database counts them.
  const coreNumbers: Progress = {
    ...counts({ backlog: 1, unstarted: 1, completed: 1, cancelled: 1 }),
    assignees: byAssignee([mem.id, 1, 1], [admin.id, 1, 0], [null, 0, 1]),
    labels: byLabel([bug.id, 1, 1], [ui.id, 1, 0], [null, 0, 1]),
    burndown: burndown(day(-2), day(2), 3, 3, 3, 2),
  };
  const looseNumbers: Progress = {
    ...counts({ unstarted: 1 }),
    assignees: byAssignee([mem.id, 0, 1]),
    labels: byLabel([bug.id, 0, 1]),
    burndown: null,
  };
  const bareNumbers: Progress = { ...counts({}), assignees: [], labels: [], burndown: null };
  const pastNumbers = (done: number): Progress => ({
    ...counts({ unstarted: 1 - done, started: 1, completed: done }),
    assignees: byAssignee([mem.id, done, 1 - done], [admin.id, done, 1 - done], [null, 0, 1]),
    labels: byLabel([bug.id, done, 1 - done], [ui.id, done, 1 - done], [null, 0, 1]),
    burndown: burndown(day(-10), day(-5), 2, 2, 2, 2, 2, 2, 2),
  });
  const modules = { Core: core, Loose: loose, Bare: bare, Past: past };
  const read = async (token: string) =>
    Object.fromEntries(
      await Promise.all(
        Object.entries(modules).map(async ([name, m]) => [name, await moduleProgress(api, token, m.id)])
      )
    );
  const want = { Core: coreNumbers, Loose: looseNumbers, Bare: bareNumbers, Past: pastNumbers(1) };
  const seen = await read(browser);
  expect(seen, "Web's modules' progress as mem reads it").toEqual(want);
  expect([await read(mem.token), await read(admin.token)], "the same with mem's token and to the admin").toEqual([
    seen,
    seen,
  ]);
  await Promise.all(
    Object.entries(modules).map(([name, m]) =>
      expectCountedInDatabase(db, { module: m.id }, want[name as keyof typeof want])
    )
  );

  // Web's list counts as the progress.
  const rows = async (token: string, archived = false) =>
    Object.fromEntries((await listModules(api, token, web.id, archived)).map((m) => [m.name, countsOf(m)]));
  const listed = await rows(browser);
  expect(listed, "Web's modules as mem lists them").toEqual({
    Core: countsOf(coreNumbers),
    Loose: countsOf(looseNumbers),
    Bare: countsOf(bareNumbers),
    Past: countsOf(pastNumbers(1)),
  });
  expect([await rows(mem.token), await rows(admin.token)], "the same with mem's token and to the admin").toEqual([
    listed,
    listed,
  ]);

  // walt counts only Plan, in Core, his own.
  const plans: Progress = {
    ...counts({ backlog: 1 }),
    assignees: byAssignee([null, 0, 1]),
    labels: byLabel([null, 0, 1]),
    burndown: burndown(day(-2), day(2), 1, 1, 1, 1),
  };
  const none: Progress = { ...bareNumbers, burndown: burndown(day(-10), day(-5), 0, 0, 0, 0, 0, 0, 0) };
  const walts = { Core: plans, Loose: { ...bareNumbers }, Bare: bareNumbers, Past: none };
  expect(await read(walt.token), "Web's modules' progress as walt reads it").toEqual(walts);
  await expectCountedInDatabase(db, { module: core.id }, plans, walt.id);
  expect(await rows(walt.token), "Web's modules as walt lists them").toEqual({
    Core: countsOf(plans),
    Loose: counts({}),
    Bare: counts({}),
    Past: counts({}),
  });

  // A module is never frozen: after a run of the freeze, mem puts Ship back to do, and Past, ended, counts it so;
  // archived, Past is listed in the archived catalog as counted, and read so. A module keeps no frozen column.
  await freezeRanSince(db, await databaseNow(db));
  expect((await updateIssue(api, browser, ship.id, { state_id: states.todo })).state_id, "Ship to do").toBe(
    states.todo
  );
  expect(await moduleProgress(api, browser, past.id), "Past's progress after Ship to do").toEqual(pastNumbers(0));
  await expectCountedInDatabase(db, { module: past.id }, pastNumbers(0));
  expect(await moduleCalls(api).archive(admin.token, past.id), "the admin archives Past").toEqual({ status: 200 });
  expect(await rows(browser, true), "Web's archived modules as mem lists them").toEqual({
    Past: countsOf(pastNumbers(0)),
  });
  expect(await moduleProgress(api, mem.token, past.id), "Past's progress once archived").toEqual(pastNumbers(0));
  expect(
    await db.query(
      `SELECT table_name FROM information_schema.columns
        WHERE table_schema = current_schema() AND column_name = 'progress_snapshot'`
    ),
    "the tables with a frozen progress"
  ).toEqual([{ table_name: "cycles" }]);

  // Web shows its guests every work item: walt reads and lists Core as mem.
  expect(
    (await changeProject(api, admin.token, web.id, { guest_view_all_features: true })).status,
    "the admin shows Web's guests every work item"
  ).toBe(200);
  expect(
    [await moduleProgress(api, walt.token, core.id), (await rows(walt.token)).Core],
    "Core's progress and row to walt now"
  ).toEqual([await moduleProgress(api, browser, core.id), (await rows(browser)).Core]);
});
