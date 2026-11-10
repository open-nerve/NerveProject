import { changeProject } from "../../fixtures/api";
import {
  databaseNow,
  expectCountedInDatabase,
  freezeRanSince,
  frozenSnapshot,
  frozenWrite,
  snapshotOf,
} from "../../fixtures/assert/progress";
import { createCycle, listCycles, readCycle } from "../../fixtures/cycles";
import { createIssue, updateIssue } from "../../fixtures/issues";
import {
  burndown,
  byAssignee,
  byLabel,
  counts,
  countsOf,
  cycleProgress,
  endCycle,
  progressWorld,
  type CycleProgress,
  type Progress,
} from "../../fixtures/planning-progress";
import { expect, test } from "../../fixtures/test";

// G6, a cycle's progress frozen once it ends (M6 design 3.5, M6 P4 spec 2.4, 2.5, R5, R6): the periodic freeze (every
// 2s in the tests) writes it once, of every work item, and never again; it leaves the cycle's last write as it was.
// Web keeps the time of a zone where it is about noon as the story starts (noonZone).

/** An ended cycle's progress as one who sees only his own work items reads it: his live progress p, nothing frozen. */
function ownOnly(p: Progress): CycleProgress {
  return { status: "completed", live: p, progress_snapshot: null };
}

test("G6 (API): Sprint runs, and mem puts four work items in it, one of each group but the backlog, walt's among them; the admin moves its days into the past and it ends; the periodic freeze writes its progress, the numbers it has then, without changing its last write; mem completes Fix the login and takes Plan out: the frozen progress, in the database, read and in the list's row, stays as it was, after another run of the freeze too, while the live one changes; walt, acme's admin and Web's guest, counts his own alone and is shown no frozen progress until Web shows its guests every work item; mem reads alike with either token", async ({
  api,
  db,
}, testInfo) => {
  const { day, admin, mem, walt, web, states, bug, browser } = await progressWorld(api, testInfo);
  const sprint = await createCycle(api, admin.token, web.id, { name: "Sprint", start_date: day(-3), end_date: day(3) });

  // Fix the login, to do, mem's, Bug; Write the docs, done, mem's; walt's Plan, in progress; Drop IE, cancelled. mem
  // puts each in Sprint.
  const fix = await createIssue(api, admin.token, web.id, {
    name: "Fix the login",
    state_id: states.todo,
    assignee_ids: [mem.id],
    label_ids: [bug.id],
  });
  const docs = await createIssue(api, admin.token, web.id, {
    name: "Write the docs",
    state_id: states.done,
    assignee_ids: [mem.id],
  });
  const plan = await createIssue(api, walt.token, web.id, { name: "Plan", state_id: states.started });
  const drop = await createIssue(api, admin.token, web.id, { name: "Drop IE", state_id: states.cancelled });
  expect(
    await Promise.all(
      [fix, docs, plan, drop].map(
        async (item) => (await updateIssue(api, browser, item.id, { cycle_id: sprint.id })).cycle_id
      )
    ),
    "the work items in Sprint"
  ).toEqual(Array(4).fill(sprint.id));

  // The admin moves Sprint's days into the past: it ended four days ago, and Write the docs was done after it.
  const ended = await endCycle(api, admin.token, sprint.id, day(-10), day(-4));
  const frozenNumbers: Progress = {
    ...counts({ unstarted: 1, started: 1, completed: 1, cancelled: 1 }),
    assignees: byAssignee([mem.id, 1, 1], [null, 0, 1]),
    labels: byLabel([bug.id, 0, 1], [null, 1, 1]),
    burndown: burndown(day(-10), day(-4), 3, 3, 3, 3, 3, 3, 3, 3),
  };
  const read = (token: string) => cycleProgress(api, token, sprint.id);
  expect((await read(browser)).live, "Sprint's live progress as it ends").toEqual(frozenNumbers);

  // The periodic freeze writes it: in the database as read, the numbers it had, after Sprint ended; Sprint's last
  // write is still the admin's change of its days.
  const column = await frozenSnapshot(db, sprint.id);
  const seen = await read(browser);
  const frozen = seen.progress_snapshot;
  if (!frozen) {
    throw new Error("Sprint frozen in the database, and read without its frozen progress");
  }
  expect(seen, "Sprint's progress as mem reads it once frozen").toEqual({
    status: "completed",
    live: frozenNumbers,
    progress_snapshot: { ...frozenNumbers, frozen_at: frozen.frozen_at },
  });
  expect(column, "Sprint's frozen column").toEqual(frozen);
  const unwritten = { by: admin.email, written_then: true, frozen_within: true };
  const freeze = () => frozenWrite(db, sprint.id, ended.updated_at, ended.updated_at, new Date().toISOString());
  expect(await freeze(), "Sprint's last write and its freeze's moment").toEqual(unwritten);
  expect((await readCycle(api, browser, sprint.id)).updated_at, "Sprint's last write as read").toBe(ended.updated_at);
  await expectCountedInDatabase(db, { cycle: sprint.id }, frozenNumbers);
  const listed = async (token: string) => {
    const row = (await listCycles(api, token, web.id)).find((c) => c.id === sprint.id);
    return row && { ...countsOf(row), assignee_ids: row.assignee_ids, progress_snapshot: row.progress_snapshot };
  };
  const frozenCounts = { ...countsOf(frozenNumbers), frozen_at: frozen.frozen_at };
  expect(await listed(browser), "Sprint's row in Web's list").toEqual({
    ...countsOf(frozenNumbers),
    assignee_ids: [mem.id],
    progress_snapshot: frozenCounts,
  });

  // walt counts only Plan, his own, and is shown no frozen progress.
  const plans: Progress = {
    ...counts({ started: 1 }),
    assignees: byAssignee([null, 0, 1]),
    labels: byLabel([null, 0, 1]),
    burndown: burndown(day(-10), day(-4), 1, 1, 1, 1, 1, 1, 1, 1),
  };
  expect(await read(walt.token), "Sprint's progress as walt reads it").toEqual(ownOnly(plans));
  expect(await listed(walt.token), "Sprint's row in walt's list").toEqual({
    ...countsOf(plans),
    assignee_ids: [],
    progress_snapshot: null,
  });
  await expectCountedInDatabase(db, { cycle: sprint.id }, plans, walt.id);

  // mem completes Fix the login and takes Plan out of Sprint, which has ended: the live progress changes, the frozen
  // one does not, nor does another run of the freeze change it.
  expect(
    [
      (await updateIssue(api, browser, fix.id, { state_id: states.done })).state_id,
      (await updateIssue(api, browser, plan.id, { cycle_id: null })).cycle_id,
    ],
    "mem completes Fix the login and takes Plan out"
  ).toEqual([states.done, null]);
  const since = await databaseNow(db);
  await freezeRanSince(db, since);
  const now: Progress = {
    ...counts({ completed: 2, cancelled: 1 }),
    assignees: byAssignee([mem.id, 2, 0]),
    labels: byLabel([bug.id, 1, 0], [null, 1, 0]),
    burndown: burndown(day(-10), day(-4), 2, 2, 2, 2, 2, 2, 2, 2),
  };
  const after = await read(browser);
  expect(after, "Sprint's progress as mem reads it after the changes").toEqual({
    status: "completed",
    live: now,
    progress_snapshot: frozen,
  });
  expect([await read(mem.token), await read(admin.token)], "the same with mem's token and to the admin").toEqual([
    after,
    after,
  ]);
  expect(await snapshotOf(db, sprint.id), "Sprint's frozen column after the changes").toEqual(column);
  expect(await freeze(), "Sprint's last write and its freeze's moment after the changes").toEqual(unwritten);
  await expectCountedInDatabase(db, { cycle: sprint.id }, now);
  expect(await listed(browser), "Sprint's row in Web's list after the changes").toEqual({
    ...countsOf(now),
    assignee_ids: [mem.id],
    progress_snapshot: frozenCounts,
  });

  // walt has none left in Sprint; once Web shows its guests every work item, he reads it as mem.
  const none: Progress = {
    ...counts({}),
    assignees: [],
    labels: [],
    burndown: burndown(day(-10), day(-4), 0, 0, 0, 0, 0, 0, 0, 0),
  };
  expect(await read(walt.token), "Sprint's progress as walt reads it after the changes").toEqual(ownOnly(none));
  expect(
    (await changeProject(api, admin.token, web.id, { guest_view_all_features: true })).status,
    "the admin shows Web's guests every work item"
  ).toBe(200);
  expect(await read(walt.token), "Sprint's progress to walt now").toEqual(after);
  expect(await listed(walt.token), "Sprint's row in walt's list now").toEqual(await listed(browser));
});
