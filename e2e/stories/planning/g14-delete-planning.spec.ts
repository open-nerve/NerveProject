import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { cycleMoments, moduleMoments } from "../../fixtures/assert/planning";
import {
  cycleJoined,
  cycleLeft,
  expectCycleRows,
  expectModuleRows,
  expectPlanningActivity,
  lastWrite,
  moduleJoined,
  moduleLeft,
  wrote,
  type LastWrite,
} from "../../fixtures/assert/issue-planning";
import { byId } from "../../fixtures/assert/issue";
import { login, newAccount } from "../../fixtures/auth";
import { createCycle, cycleCalls } from "../../fixtures/cycles";
import { issuePlanningCalls } from "../../fixtures/issue-planning";
import { createIssue, issueDeleted } from "../../fixtures/issues";
import { createModule, moduleCalls } from "../../fixtures/modules";
import { expect, test } from "../../fixtures/test";

// G14, deleting a cycle, a module or a work item (M6 design 3.3, 4.3, M6 P3a spec 2.5, R6): the work items of a
// deleted cycle or module are left as they are but for it, taken out of it at the deletion's moment by the account
// that deleted it, each with a row of its activity and changed last by that deletion; a deleted work item takes its
// rows of cycles and modules with it, at its moment and by its deleter, and no row of activity about them. The admin
// calls with a personal access token, the member with the token of his sign-in.

test("G14 (API): the admin deletes the member's Sprint, and the member his Launch: Fix the login and Write the docs, in both, are taken out of each at its deletion's moment by its deleter, a row of activity each, changed last by that deletion, and read alike to either of the member's tokens, while Plan, in Next and Docs, is left as it is; then Fix the login joins Next; the member deletes Epic, in Next and Docs with Child under it: their rows of Next and Docs are removed with them at that moment, by him, and no row of activity is written about them", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  const mem = await newAccount(api, testInfo, "mem");
  await inviteAndAccept(api, admin.token, slug, mem, 15);
  const web = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, web.id, [{ member_id: mem.id, role: 15 }]);
  const on = await changeProject(api, admin.token, web.id, { cycle_view: true, module_view: true });
  expect(on.status, `the admin turns Web's cycles and modules on: ${JSON.stringify(on)}`).toBe(200);
  const browser = (await login(api, mem.email)).access_token;
  const { update, read } = issuePlanningCalls(api);

  // mem creates Sprint and Launch, the admin Next and Docs. mem creates Fix the login in Sprint, Launch and Docs, and
  // Epic in Next and Docs with Child under it, in them too; the admin Write the docs in Sprint and Launch, and Plan in
  // Next and Docs.
  const sprint = await createCycle(api, browser, web.id, { name: "Sprint" });
  const next = await createCycle(api, admin.token, web.id, { name: "Next" });
  const launch = await createModule(api, browser, web.id, { name: "Launch" });
  const docs = await createModule(api, admin.token, web.id, { name: "Docs" });
  const fix = await createIssue(api, browser, web.id, {
    name: "Fix the login",
    cycle_id: sprint.id,
    module_ids: [launch.id, docs.id],
  });
  const write = await createIssue(api, admin.token, web.id, {
    name: "Write the docs",
    cycle_id: sprint.id,
    module_ids: [launch.id],
  });
  const plan = await createIssue(api, admin.token, web.id, { name: "Plan", cycle_id: next.id, module_ids: [docs.id] });
  const epic = await createIssue(api, browser, web.id, { name: "Epic", cycle_id: next.id, module_ids: [docs.id] });
  const child = await createIssue(api, browser, web.id, {
    name: "Child",
    parent_id: epic.id,
    cycle_id: next.id,
    module_ids: [docs.id],
  });
  const made = new Map<string, LastWrite>(
    await Promise.all([fix, write, plan, epic, child].map(async (i) => [i.id, await lastWrite(db, i.id)] as const))
  );
  const madeAt = (item: { id: string }) => made.get(item.id)?.at ?? "";
  const joins = (...joined: { name: string; id: string }[]) => joined.toSorted(byId).map((m) => moduleJoined(m.name));
  const fixActivity = wrote(mem.email, madeAt(fix), cycleJoined("Sprint"), ...joins(launch, docs));
  const writeActivity = wrote(admin.email, madeAt(write), cycleJoined("Sprint"), moduleJoined("Launch"));
  await expectPlanningActivity(db, fix.id, fixActivity);
  await expectPlanningActivity(db, write.id, writeActivity);

  // The admin deletes Sprint, mem's: Fix the login and Write the docs are taken out of it at its deletion's moment,
  // by the admin, a row of activity each, his then, and he changed them last, then.
  expect(await cycleCalls(api).remove(admin.token, sprint.id), "the admin deletes Sprint").toEqual({ status: 204 });
  const sprintDeleted = (await cycleMoments(db, web.id))["Sprint"] ?? "";
  await expectCycleRows(db, fix.id, [
    { target: "Sprint", added: madeAt(fix), removed: sprintDeleted, adder: mem.email, by: admin.email },
  ]);
  await expectCycleRows(db, write.id, [
    { target: "Sprint", added: madeAt(write), removed: sprintDeleted, adder: admin.email, by: admin.email },
  ]);
  await expectPlanningActivity(db, fix.id, [...fixActivity, ...wrote(admin.email, sprintDeleted, cycleLeft("Sprint"))]);
  await expectPlanningActivity(db, write.id, [
    ...writeActivity,
    ...wrote(admin.email, sprintDeleted, cycleLeft("Sprint")),
  ]);
  const takenOut = { at: sprintDeleted, by: admin.email, deleted: null };
  expect(
    [
      await lastWrite(db, fix.id),
      await lastWrite(db, write.id),
      await lastWrite(db, plan.id),
      await lastWrite(db, epic.id),
      await lastWrite(db, child.id),
    ],
    "the last writes of the work items after Sprint's deletion"
  ).toEqual([takenOut, takenOut, made.get(plan.id), made.get(epic.id), made.get(child.id)]);

  // mem deletes Launch, his: the same, by him, its own moment; Docs' rows are left as they are.
  expect(await moduleCalls(api).remove(browser, launch.id), "mem deletes Launch").toEqual({ status: 204 });
  const launchDeleted = (await moduleMoments(db, web.id))["Launch"] ?? "";
  expect(launchDeleted, "Launch's deletion, a moment of its own").not.toBe(sprintDeleted);
  await expectModuleRows(db, fix.id, [
    { target: "Docs", added: madeAt(fix), removed: null, adder: mem.email, by: mem.email },
    { target: "Launch", added: madeAt(fix), removed: launchDeleted, adder: mem.email, by: mem.email },
  ]);
  await expectModuleRows(db, write.id, [
    { target: "Launch", added: madeAt(write), removed: launchDeleted, adder: admin.email, by: mem.email },
  ]);
  await expectPlanningActivity(db, fix.id, [
    ...fixActivity,
    ...wrote(admin.email, sprintDeleted, cycleLeft("Sprint")),
    ...wrote(mem.email, launchDeleted, moduleLeft("Launch")),
  ]);
  await expectPlanningActivity(db, write.id, [
    ...writeActivity,
    ...wrote(admin.email, sprintDeleted, cycleLeft("Sprint")),
    ...wrote(mem.email, launchDeleted, moduleLeft("Launch")),
  ]);
  const takenOutAgain = { at: launchDeleted, by: mem.email, deleted: null };
  expect(
    [await lastWrite(db, fix.id), await lastWrite(db, write.id), await lastWrite(db, plan.id)],
    "the last writes of the work items after Launch's deletion"
  ).toEqual([takenOutAgain, takenOutAgain, made.get(plan.id)]);

  // The work items are read as they are, alike to either of mem's tokens and to the admin.
  const reads = async (token: string) => Promise.all([fix, write, plan].map((item) => read(token, item.id)));
  const seen = await reads(browser);
  expect(seen, "Fix the login, Write the docs and Plan as mem reads them").toEqual([
    { status: 200, cycle_id: null, module_ids: [docs.id] },
    { status: 200, cycle_id: null, module_ids: [] },
    { status: 200, cycle_id: next.id, module_ids: [docs.id] },
  ]);
  expect(await reads(mem.token), "the same with mem's personal access token").toEqual(seen);
  expect(await reads(admin.token), "the same to the admin").toEqual(seen);

  // Out of Sprint, Fix the login joins Next.
  expect(await update(browser, fix.id, { cycle_id: next.id }), "mem puts Fix the login in Next").toEqual({
    status: 200,
    cycle_id: next.id,
    module_ids: [docs.id],
  });
  const joinedNext = await lastWrite(db, fix.id);
  await expectCycleRows(db, fix.id, [
    { target: "Sprint", added: madeAt(fix), removed: sprintDeleted, adder: mem.email, by: admin.email },
    { target: "Next", added: joinedNext.at, removed: null, adder: mem.email, by: mem.email },
  ]);

  // mem deletes Epic, his, with Child under it: their rows of Next and Docs removed with them at that moment, by him;
  // their activity about them as it was. Next and Docs keep the others.
  expect(await issueDeleted(api, browser, epic.id), "mem deletes Epic").toEqual({ status: 204 });
  const epicDeleted = await lastWrite(db, epic.id);
  expect(
    [epicDeleted.by, epicDeleted.deleted === epicDeleted.at, await lastWrite(db, child.id)],
    "Epic's deletion, and Child's"
  ).toEqual([mem.email, true, { at: epicDeleted.at, by: mem.email, deleted: epicDeleted.at }]);
  await Promise.all(
    [epic, child].flatMap((item) => [
      expectCycleRows(db, item.id, [
        { target: "Next", added: madeAt(item), removed: epicDeleted.at, adder: mem.email, by: mem.email },
      ]),
      expectModuleRows(db, item.id, [
        { target: "Docs", added: madeAt(item), removed: epicDeleted.at, adder: mem.email, by: mem.email },
      ]),
      expectPlanningActivity(db, item.id, wrote(mem.email, madeAt(item), cycleJoined("Next"), moduleJoined("Docs"))),
    ])
  );
  expect(
    [await read(browser, epic.id), await read(browser, child.id), await read(admin.token, plan.id)],
    "Epic and Child read after their deletion, and Plan"
  ).toEqual([
    { status: 404, code: "issue.not_found" },
    { status: 404, code: "issue.not_found" },
    { status: 200, cycle_id: next.id, module_ids: [docs.id] },
  ]);
});
