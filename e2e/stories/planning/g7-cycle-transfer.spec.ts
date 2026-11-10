import { randomUUID } from "node:crypto";

import { changeProject, createProject } from "../../fixtures/api";
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
import { expectCountedInDatabase, frozenSnapshot, frozenWrite, snapshotOf } from "../../fixtures/assert/progress";
import { createCycle, cycleCalls } from "../../fixtures/cycles";
import { validationFailed } from "../../fixtures/issue-planning";
import { createIssue, issueArchived, updateIssue, type Issue } from "../../fixtures/issues";
import {
  burndown,
  byAssignee,
  byLabel,
  counts,
  cycleProgress,
  endCycle,
  moved,
  progressWorld,
  transferred,
  type Progress,
} from "../../fixtures/planning-progress";
import { createState, updateState } from "../../fixtures/states";
import { expect, test } from "../../fixtures/test";

// G7, moving a completed cycle's unfinished work items to another cycle (M6 design 3.5, M6 P4 spec 2.3, 2.6, R7, R8):
// those in the backlog, unstarted or started move, each with a row of activity, last changed by the caller; the
// completed, the cancelled and the archived stay; the cycle's progress is frozen first, unless it is already, with
// the numbers it has before the move. mem calls with the token of his sign-in, the admin with a personal access
// token. Web keeps the time of a zone where it is about noon as the story starts (noonZone).

test("G7 (API): mem puts six work items in Sprint, one of each group and Shelf, archived, whose state then moves to the started group; Sprint ends; mem moves its unfinished three to Next, frozen first, by the periodic freeze or the transfer, with the numbers before the move, each row of theirs removed and added at that moment by him, with a row of activity naming both, while Ship, Drop and Shelf stay as they were; a second transfer moves none and writes nothing; Ship to do again, a transfer from Next, Later or Someday, which have not ended, to Ended, ended, to Shelved, archived, to Gone, deleted, to Ops' Infra, to none, to Sprint itself, from Shelved, by walt, from none, and with Web's cycles off are refused, each writing nothing; the admin then moves Ship to Someday, a draft, the frozen progress still the first", async ({
  api,
  db,
}, testInfo) => {
  const { day, admin, mem, walt, slug, web, states, browser } = await progressWorld(api, testInfo);
  const ops = await createProject(api, admin.token, slug, { name: "Ops", identifier: "OPS" });
  expect((await changeProject(api, admin.token, ops.id, { cycle_view: true })).status, "Ops' cycles on").toBe(200);
  const infra = await createCycle(api, admin.token, ops.id, { name: "Infra" });
  const sprint = await createCycle(api, admin.token, web.id, { name: "Sprint", start_date: day(-3), end_date: day(3) });

  // Plan, in the backlog; Fix, to do; Build, in progress; Ship, done; Drop, cancelled; Shelf, in Parked, a completed
  // state. mem puts each in Sprint. The admin archives Shelf, then moves Parked to the started group.
  const parked = await createState(api, admin.token, web.id, { name: "Parked", color: "#8B5CF6", group: "completed" });
  const made = (name: string, state_id: string) => createIssue(api, admin.token, web.id, { name, state_id });
  const items = [
    await made("Plan", states.backlog),
    await made("Fix", states.todo),
    await made("Build", states.started),
    await made("Ship", states.done),
    await made("Drop", states.cancelled),
    await made("Shelf", parked.id),
  ];
  const joined = new Map(
    await Promise.all(
      items.map(async (item): Promise<[string, LastWrite]> => {
        const { cycle_id } = await updateIssue(api, browser, item.id, { cycle_id: sprint.id });
        expect(cycle_id, `${item.name} in Sprint`).toBe(sprint.id);
        return [item.id, await lastWrite(db, item.id)];
      })
    )
  );
  const [plan, fix, build, ship, drop, shelf] = items as [Issue, Issue, Issue, Issue, Issue, Issue];
  expect((await issueArchived(api, admin.token, shelf.id)).status, "the admin archives Shelf").toBe(200);
  expect((await updateState(api, admin.token, parked.id, { group: "started" })).group, "Parked started").toBe(
    "started"
  );
  const at = (item: Issue) => joined.get(item.id)?.at ?? "";

  // The admin ends Sprint; Next runs, Later is ahead, Someday a draft; Ended ended, Shelved ended and is archived;
  // Gone is deleted.
  const sprintEnded = await endCycle(api, admin.token, sprint.id, day(-20), day(-14));
  const cycle = (name: string, start?: string, end?: string) =>
    createCycle(api, admin.token, web.id, start && end ? { name, start_date: start, end_date: end } : { name });
  const next = await cycle("Next", day(-1), day(4));
  const later = await cycle("Later", day(10), day(12));
  const someday = await cycle("Someday");
  const ended = await cycle("Ended", day(-40), day(-30));
  const shelved = await cycle("Shelved", day(-50), day(-45));
  const gone = await cycle("Gone");
  const calls = cycleCalls(api);
  expect(
    [await calls.archive(admin.token, shelved.id), await calls.remove(admin.token, gone.id)],
    "the admin archives Shelved and deletes Gone"
  ).toEqual([{ status: 200 }, { status: 204 }]);

  // Before the move: one of each group, Shelf archived out of the count; seven days, four to burn, Ship done after.
  const before: Progress = {
    ...counts({ backlog: 1, unstarted: 1, started: 1, completed: 1, cancelled: 1 }),
    assignees: byAssignee([null, 1, 3]),
    labels: byLabel([null, 1, 3]),
    burndown: burndown(day(-20), day(-14), 4, 4, 4, 4, 4, 4, 4, 4),
  };
  expect((await cycleProgress(api, browser, sprint.id)).live, "Sprint's progress as it ends").toEqual(before);
  await expectCountedInDatabase(db, { cycle: sprint.id }, before);
  const stay = [ship, drop, shelf];
  const stayed = await Promise.all(stay.map((item) => lastWrite(db, item.id)));

  // mem moves Plan, Fix and Build to Next: each one's row of Sprint removed and one of Next added at one moment, his,
  // with a row of activity; Ship, Drop and Shelf stay as they were.
  expect(await transferred(api, browser, sprint.id, next.id), "mem moves Sprint's unfinished to Next").toEqual(
    moved(3)
  );
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
    await Promise.all(stay.map((item) => lastWrite(db, item.id))),
    "Ship's, Drop's and Shelf's last writes"
  ).toEqual(stayed);
  await Promise.all(
    stay.map(async (item) => {
      await expectCycleRows(db, item.id, [
        { target: "Sprint", added: at(item), removed: null, adder: mem.email, by: mem.email },
      ]);
      await expectPlanningActivity(db, item.id, wrote(mem.email, at(item), cycleJoined("Sprint")));
    })
  );

  // Sprint was frozen, after it ended and by the move, with the numbers before it, its last write still the
  // admin's; it has Ship and Drop left, Next the three.
  const column = await frozenSnapshot(db, sprint.id);
  expect(
    await frozenWrite(db, sprint.id, sprintEnded.updated_at, sprintEnded.updated_at, first.at),
    "Sprint's freeze"
  ).toEqual({
    by: admin.email,
    written_then: true,
    frozen_within: true,
  });
  const left: Progress = {
    ...counts({ completed: 1, cancelled: 1 }),
    assignees: byAssignee([null, 1, 0]),
    labels: byLabel([null, 1, 0]),
    burndown: burndown(day(-20), day(-14), 1, 1, 1, 1, 1, 1, 1, 1),
  };
  const read = await cycleProgress(api, browser, sprint.id);
  expect(read, "Sprint's progress after the move").toEqual({
    status: "completed",
    live: left,
    progress_snapshot: { ...before, frozen_at: read.progress_snapshot?.frozen_at },
  });
  expect(column, "Sprint's frozen column").toEqual(read.progress_snapshot);
  expect(await cycleProgress(api, admin.token, sprint.id), "the same to the admin").toEqual(read);
  await expectCountedInDatabase(db, { cycle: sprint.id }, left);
  const three: Progress = {
    ...counts({ backlog: 1, unstarted: 1, started: 1 }),
    assignees: byAssignee([null, 0, 3]),
    labels: byLabel([null, 0, 3]),
    burndown: burndown(day(-1), day(4), 3, 3, 3),
  };
  expect(await cycleProgress(api, browser, next.id), "Next's progress").toEqual({
    status: "current",
    live: three,
    progress_snapshot: null,
  });
  await expectCountedInDatabase(db, { cycle: next.id }, three);

  // Again: none left to move, nothing written.
  const rows = await workItemRows(db, web.id);
  expect(await transferred(api, admin.token, sprint.id, next.id), "the admin moves Sprint's unfinished again").toEqual(
    moved(0)
  );
  expect(await workItemRows(db, web.id), "Web's work items' rows after a second transfer").toEqual(rows);
  expect(await snapshotOf(db, sprint.id), "Sprint's frozen column after a second transfer").toEqual(column);

  // Ship to do again. Refused, each writing nothing: from a cycle not ended; to one ended or archived; to one not
  // Web's undeleted others; from an archived one; by walt, who sees only his own; from none; with Web's cycles off.
  expect((await updateIssue(api, browser, ship.id, { state_id: states.todo })).state_id, "Ship to do").toBe(
    states.todo
  );
  const unchanged = await workItemRows(db, web.id);
  const notCompleted = { status: 409, code: "planning.cycle_not_completed" };
  const notAllowed = validationFailed(["target_cycle_id", "not_allowed"]);
  expect(
    [
      await transferred(api, browser, next.id, someday.id),
      await transferred(api, browser, later.id, next.id),
      await transferred(api, browser, someday.id, next.id),
      await transferred(api, browser, sprint.id, ended.id),
      await transferred(api, browser, sprint.id, shelved.id),
      await transferred(api, browser, sprint.id, gone.id),
      await transferred(api, admin.token, sprint.id, infra.id),
      await transferred(api, browser, sprint.id, randomUUID()),
      await transferred(api, browser, sprint.id, sprint.id),
      await transferred(api, browser, shelved.id, next.id),
      await transferred(api, walt.token, sprint.id, next.id),
      await transferred(api, browser, randomUUID(), next.id),
    ],
    "transfers from Next, Later and Someday; to Ended, Shelved, Gone, Infra, none and Sprint; from Shelved; walt's; from none"
  ).toEqual([
    notCompleted,
    notCompleted,
    notCompleted,
    { status: 409, code: "planning.cycle_completed" },
    { status: 409, code: "planning.archived" },
    notAllowed,
    notAllowed,
    notAllowed,
    notAllowed,
    { status: 409, code: "planning.archived" },
    { status: 403, code: "forbidden" },
    { status: 404, code: "planning.cycle_not_found" },
  ]);
  const toggle = (on: boolean) => changeProject(api, admin.token, web.id, { cycle_view: on });
  expect((await toggle(false)).status, "Web's cycles off").toBe(200);
  expect(await transferred(api, browser, sprint.id, next.id), "a transfer with Web's cycles off").toEqual({
    status: 409,
    code: "planning.cycles_disabled",
  });
  expect((await toggle(true)).status, "Web's cycles on").toBe(200);
  expect(await workItemRows(db, web.id), "Web's work items' rows after the refusals").toEqual(unchanged);

  // The admin moves Ship, unfinished again, to Someday, a draft: its row and its activity, his; the frozen progress
  // is still the first.
  expect(await transferred(api, admin.token, sprint.id, someday.id), "the admin moves Ship to Someday").toEqual(
    moved(1)
  );
  const third = await lastWrite(db, ship.id);
  expect(third.by, "who moved Ship").toBe(admin.email);
  await expectCycleRows(db, ship.id, [
    { target: "Sprint", added: at(ship), removed: third.at, adder: mem.email, by: admin.email },
    { target: "Someday", added: third.at, removed: null, adder: admin.email, by: admin.email },
  ]);
  await expectPlanningActivity(db, ship.id, [
    ...wrote(mem.email, at(ship), cycleJoined("Sprint")),
    ...wrote(admin.email, third.at, cycleMoved("Sprint", "Someday")),
  ]);
  expect(await snapshotOf(db, sprint.id), "Sprint's frozen column after the third transfer").toEqual(column);
  expect((await cycleProgress(api, browser, sprint.id)).progress_snapshot, "Sprint's frozen progress as read").toEqual(
    read.progress_snapshot
  );
});
