import { randomUUID } from "node:crypto";

import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import {
  cycleJoined,
  cycleLeft,
  cycleMoved,
  expectCycleRows,
  expectPlanningActivity,
  lastWrite,
  workItemRows,
  wrote,
} from "../../fixtures/assert/issue-planning";
import { login, newAccount } from "../../fixtures/auth";
import { createCycle, cycleCalls } from "../../fixtures/cycles";
import { issuePlanningCalls, validationFailed } from "../../fixtures/issue-planning";
import { createIssue, listIssues } from "../../fixtures/issues";
import { expect, test } from "../../fixtures/test";

// G4, a work item's cycle (M6 design 3.3, 3.8, M6 P3a spec 2.3, 2.4, R2–R4): it joins a cycle as it is created or
// changed, moves to another and leaves it, through its own cycle_id; one cycle at a time, a row of its activity for
// each change. The admin calls with a personal access token, the member with the token of his sign-in.

test("G4 (API): Web's member creates Fix the login in Sprint 1 and the admin puts Write the docs in Sprint 2; the member moves Fix the login to Sprint 2, its row of Sprint 1 removed at that moment, and takes it out; its own cycle again writes nothing; Done, ended, and Old, archived, are refused with their own codes, the archive before a wrong label and the end after it; a deleted cycle, Ops' and one not there are refused at cycle_id with the other problems, created or changed, each writing nothing; with Web's cycles off a change of the cycle is refused but its own cycle is not, and turned on again Write the docs is still in Sprint 2; the work items read alike to either of the member's tokens", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  const mem = await newAccount(api, testInfo, "mem");
  await inviteAndAccept(api, admin.token, slug, mem, 15);
  const web = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  const ops = await createProject(api, admin.token, slug, { name: "Ops", identifier: "OPS" });
  await addProjectMembers(api, admin.token, web.id, [{ member_id: mem.id, role: 15 }]);
  const turn = async (projectId: string, on: boolean) => {
    const changed = await changeProject(api, admin.token, projectId, { cycle_view: on });
    expect(changed.status, `the admin turns ${projectId}'s cycles ${on ? "on" : "off"}`).toBe(200);
  };
  await turn(web.id, true);
  await turn(ops.id, true);
  const browser = (await login(api, mem.email)).access_token;
  const { create, update, read } = issuePlanningCalls(api);

  // Sprint 1 is a draft and Sprint 2 ahead, in 2099; Done ended in 2025, and Old too, which the admin archives; Gone
  // he deletes. Infra is Ops'.
  const sprint1 = await createCycle(api, admin.token, web.id, { name: "Sprint 1" });
  const sprint2 = await createCycle(api, admin.token, web.id, {
    name: "Sprint 2",
    start_date: "2099-01-01",
    end_date: "2099-01-14",
  });
  const done = await createCycle(api, admin.token, web.id, {
    name: "Done",
    start_date: "2025-01-01",
    end_date: "2025-01-14",
  });
  const old = await createCycle(api, admin.token, web.id, {
    name: "Old",
    start_date: "2025-02-01",
    end_date: "2025-02-14",
  });
  const gone = await createCycle(api, admin.token, web.id, { name: "Gone" });
  const infra = await createCycle(api, admin.token, ops.id, { name: "Infra" });
  const cycles = cycleCalls(api);
  expect(
    [await cycles.archive(admin.token, old.id), await cycles.remove(admin.token, gone.id)],
    "the admin archives Old and deletes Gone"
  ).toEqual([{ status: 200 }, { status: 204 }]);

  // mem creates Fix the login in Sprint 1: its row and its activity's, his, at its creation's moment.
  const fix = await createIssue(api, browser, web.id, { name: "Fix the login", cycle_id: sprint1.id });
  expect([fix.cycle_id, fix.module_ids], "Fix the login as created").toEqual([sprint1.id, []]);
  const created = await lastWrite(db, fix.id);
  expect(created.by, "who created Fix the login").toBe(mem.email);
  await expectCycleRows(db, fix.id, [
    { target: "Sprint 1", added: created.at, removed: null, adder: mem.email, by: mem.email },
  ]);
  let fixActivity = wrote(mem.email, created.at, cycleJoined("Sprint 1"));
  await expectPlanningActivity(db, fix.id, fixActivity);

  // The admin creates Write the docs in no cycle, and puts it in Sprint 2.
  const docs = await createIssue(api, admin.token, web.id, { name: "Write the docs" });
  expect(docs.cycle_id, "Write the docs as created").toBeNull();
  expect(await update(admin.token, docs.id, { cycle_id: sprint2.id }), "the admin puts it in Sprint 2").toEqual({
    status: 200,
    cycle_id: sprint2.id,
    module_ids: [],
  });
  const joined = await lastWrite(db, docs.id);
  expect(joined.by, "who put Write the docs in Sprint 2").toBe(admin.email);
  await expectCycleRows(db, docs.id, [
    { target: "Sprint 2", added: joined.at, removed: null, adder: admin.email, by: admin.email },
  ]);
  await expectPlanningActivity(db, docs.id, wrote(admin.email, joined.at, cycleJoined("Sprint 2")));

  // mem moves Fix the login to Sprint 2: its row of Sprint 1 removed and one of Sprint 2 added at one moment, his
  // (R4); one row of activity names both.
  expect(await update(browser, fix.id, { cycle_id: sprint2.id }), "mem moves Fix the login to Sprint 2").toEqual({
    status: 200,
    cycle_id: sprint2.id,
    module_ids: [],
  });
  const moved = await lastWrite(db, fix.id);
  expect(moved.by, "who moved Fix the login").toBe(mem.email);
  await expectCycleRows(db, fix.id, [
    { target: "Sprint 1", added: created.at, removed: moved.at, adder: mem.email, by: mem.email },
    { target: "Sprint 2", added: moved.at, removed: null, adder: mem.email, by: mem.email },
  ]);
  fixActivity = [...fixActivity, ...wrote(mem.email, moved.at, cycleMoved("Sprint 1", "Sprint 2"))];
  await expectPlanningActivity(db, fix.id, fixActivity);

  // Its own cycle again changes nothing: answered as it is, nothing written.
  let rows = await workItemRows(db, web.id);
  expect(await update(admin.token, fix.id, { cycle_id: sprint2.id }), "the admin gives it Sprint 2 again").toEqual({
    status: 200,
    cycle_id: sprint2.id,
    module_ids: [],
  });
  expect(await workItemRows(db, web.id), "Web's work items' rows after Sprint 2 again").toEqual(rows);

  // Refused, each writing nothing, as a change and as a creation: Done ended (409, last, after a wrong label's 422);
  // Old archived (409, before a wrong label's 422); deleted Gone, Ops' Infra and a cycle not there, at cycle_id with
  // the other problems (R2).
  const ghost = randomUUID();
  expect(
    [
      await update(browser, fix.id, { cycle_id: done.id }),
      await update(browser, fix.id, { cycle_id: done.id, label_ids: [ghost] }),
      await update(browser, fix.id, { cycle_id: old.id }),
      await update(browser, fix.id, { cycle_id: old.id, label_ids: [ghost] }),
      await update(browser, fix.id, { cycle_id: gone.id }),
      await update(admin.token, fix.id, { cycle_id: infra.id }),
      await update(browser, fix.id, { cycle_id: ghost, label_ids: [ghost] }),
    ],
    "Fix the login moved to Done, Done with a wrong label, Old, Old with a wrong label, Gone, Infra, and nowhere"
  ).toEqual([
    { status: 409, code: "planning.cycle_completed" },
    validationFailed(["label_ids[0]", "not_allowed"]),
    { status: 409, code: "planning.archived" },
    { status: 409, code: "planning.archived" },
    validationFailed(["cycle_id", "not_allowed"]),
    validationFailed(["cycle_id", "not_allowed"]),
    validationFailed(["label_ids[0]", "not_allowed"], ["cycle_id", "not_allowed"]),
  ]);
  expect(
    [
      await create(browser, web.id, { name: "Late", cycle_id: done.id }),
      await create(admin.token, web.id, { name: "Late", cycle_id: old.id }),
      await create(browser, web.id, { name: "Late", cycle_id: gone.id }),
      await create(admin.token, web.id, { name: "Late", cycle_id: infra.id }),
    ],
    "Late created in Done, Old, Gone and Infra"
  ).toEqual([
    { status: 409, code: "planning.cycle_completed" },
    { status: 409, code: "planning.archived" },
    validationFailed(["cycle_id", "not_allowed"]),
    validationFailed(["cycle_id", "not_allowed"]),
  ]);
  expect(await workItemRows(db, web.id), "Web's work items' rows after the refusals").toEqual(rows);

  // mem takes Fix the login out of Sprint 2: its row removed at that moment, his; out of any, again, writes nothing.
  expect(await update(browser, fix.id, { cycle_id: null }), "mem takes Fix the login out").toEqual({
    status: 200,
    cycle_id: null,
    module_ids: [],
  });
  const left = await lastWrite(db, fix.id);
  expect(left.by, "who took Fix the login out").toBe(mem.email);
  await expectCycleRows(db, fix.id, [
    { target: "Sprint 1", added: created.at, removed: moved.at, adder: mem.email, by: mem.email },
    { target: "Sprint 2", added: moved.at, removed: left.at, adder: mem.email, by: mem.email },
  ]);
  fixActivity = [...fixActivity, ...wrote(mem.email, left.at, cycleLeft("Sprint 2"))];
  await expectPlanningActivity(db, fix.id, fixActivity);
  rows = await workItemRows(db, web.id);
  expect(await update(browser, fix.id, { cycle_id: null }), "mem takes it out again").toEqual({
    status: 200,
    cycle_id: null,
    module_ids: [],
  });
  expect(await workItemRows(db, web.id), "Web's work items' rows after out again").toEqual(rows);

  // Web's cycles off: a change of a cycle, leaving one too, and a creation in one are refused; Write the docs given
  // its own cycle is answered as it is. Nothing is written. Turned on again, it is still in Sprint 2.
  await turn(web.id, false);
  expect(
    [
      await update(browser, docs.id, { cycle_id: null }),
      await update(admin.token, docs.id, { cycle_id: sprint1.id }),
      await update(browser, fix.id, { cycle_id: sprint1.id }),
      await create(browser, web.id, { name: "Late", cycle_id: sprint1.id }),
      await update(browser, docs.id, { cycle_id: sprint2.id }),
    ],
    "with Web's cycles off, Write the docs taken out and moved, Fix the login put in Sprint 1, Late created in it, and Write the docs given Sprint 2"
  ).toEqual([
    ...Array(4).fill({ status: 409, code: "planning.cycles_disabled" }),
    { status: 200, cycle_id: sprint2.id, module_ids: [] },
  ]);
  expect(await workItemRows(db, web.id), "Web's work items' rows, its cycles off").toEqual(rows);
  await turn(web.id, true);
  await expectCycleRows(db, docs.id, [
    { target: "Sprint 2", added: joined.at, removed: null, adder: admin.email, by: admin.email },
  ]);

  // The work items, one by one and in Web's list, read alike to either of mem's tokens, and to the admin.
  const reads = async (token: string) => [
    await read(token, fix.id),
    await read(token, docs.id),
    ((await listIssues(api, token, web.id)).data ?? [])
      .map(({ name, cycle_id, module_ids }) => ({ name, cycle_id, module_ids }))
      .toSorted((a, b) => a.name.localeCompare(b.name)),
  ];
  const seen = await reads(browser);
  expect(seen, "the work items as mem reads them").toEqual([
    { status: 200, cycle_id: null, module_ids: [] },
    { status: 200, cycle_id: sprint2.id, module_ids: [] },
    [
      { name: "Fix the login", cycle_id: null, module_ids: [] },
      { name: "Write the docs", cycle_id: sprint2.id, module_ids: [] },
    ],
  ]);
  expect(await reads(mem.token), "the same with mem's personal access token").toEqual(seen);
  expect(await reads(admin.token), "the same to the admin").toEqual(seen);
});
