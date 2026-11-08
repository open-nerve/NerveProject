import { randomUUID } from "node:crypto";

import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import { byId } from "../../fixtures/assert/issue";
import {
  expectModuleRows,
  expectPlanningActivity,
  lastWrite,
  moduleJoined,
  moduleLeft,
  workItemRows,
  wrote,
} from "../../fixtures/assert/issue-planning";
import { login, newAccount } from "../../fixtures/auth";
import { ascendingIds, issuePlanningCalls, validationFailed } from "../../fixtures/issue-planning";
import { createIssue, listIssues } from "../../fixtures/issues";
import { createModule, moduleCalls } from "../../fixtures/modules";
import { expect, test } from "../../fixtures/test";

// G11, a work item's modules (M6 design 3.3, 3.8, M6 P3a spec 2.3, 2.4, R3, R4): it joins modules as it is created,
// and a change replaces the whole set through its own module_ids: the modules it keeps are left as they are, those it
// leaves removed and those it joins added at one moment, a row of its activity for each, the joined first, each kind
// in id order. The admin calls with a personal access token, the member with the token of his sign-in.

test("G11 (API): Web's member creates Fix the login in Search and Payments, answered in id order, and the admin puts Write the docs in Docs, completed; the member replaces Fix the login's modules, Payments kept as it is, Search left and Docs joined at one moment; the same modules in another order write nothing; Old, archived, is refused before a wrong label, and a deleted module, Ops' and one not there at their places in module_ids with the other problems, as are a module named twice and a hundred and one, created or changed, each writing nothing; Docs archived is kept, and left, without a check; with Web's modules off a change is refused but its own modules are not, and turned on again they are all there; emptied, Fix the login is in none; the work items read alike to either of the member's tokens", async ({
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
    const changed = await changeProject(api, admin.token, projectId, { module_view: on });
    expect(changed.status, `the admin turns ${projectId}'s modules ${on ? "on" : "off"}`).toBe(200);
  };
  await turn(web.id, true);
  await turn(ops.id, true);
  const browser = (await login(api, mem.email)).access_token;
  const { create, update, read } = issuePlanningCalls(api);

  // The admin creates Payments, Docs, completed, and Old, completed, which he archives, and Gone, which he deletes; mem
  // creates Search. Infra is Ops'.
  const payments = await createModule(api, admin.token, web.id, { name: "Payments" });
  const search = await createModule(api, browser, web.id, { name: "Search" });
  const docs = await createModule(api, admin.token, web.id, { name: "Docs", status: "completed" });
  const old = await createModule(api, admin.token, web.id, { name: "Old", status: "completed" });
  const gone = await createModule(api, admin.token, web.id, { name: "Gone" });
  const infra = await createModule(api, admin.token, ops.id, { name: "Infra" });
  const modules = moduleCalls(api);
  expect(
    [await modules.archive(admin.token, old.id), await modules.remove(admin.token, gone.id)],
    "the admin archives Old and deletes Gone"
  ).toEqual([{ status: 200 }, { status: 204 }]);
  const joins = (...joined: { name: string; id: string }[]) => joined.toSorted(byId).map((m) => moduleJoined(m.name));
  const leaves = (...left: { name: string; id: string }[]) => left.toSorted(byId).map((m) => moduleLeft(m.name));

  // mem creates Fix the login in Search and Payments: answered in id order, its rows and its activity's, his, at its
  // creation's moment, the modules joined in id order.
  const fix = await createIssue(api, browser, web.id, { name: "Fix the login", module_ids: [search.id, payments.id] });
  expect([fix.cycle_id, fix.module_ids], "Fix the login as created").toEqual([null, ascendingIds(search, payments)]);
  const created = await lastWrite(db, fix.id);
  expect(created.by, "who created Fix the login").toBe(mem.email);
  await expectModuleRows(db, fix.id, [
    { target: "Payments", added: created.at, removed: null, adder: mem.email, by: mem.email },
    { target: "Search", added: created.at, removed: null, adder: mem.email, by: mem.email },
  ]);
  let fixActivity = wrote(mem.email, created.at, ...joins(search, payments));
  await expectPlanningActivity(db, fix.id, fixActivity);

  // The admin creates Write the docs in no module, and puts it in Docs: a module joins whatever its status.
  const guide = await createIssue(api, admin.token, web.id, { name: "Write the docs", module_ids: [] });
  expect(guide.module_ids, "Write the docs as created").toEqual([]);
  expect(await update(admin.token, guide.id, { module_ids: [docs.id] }), "the admin puts it in Docs").toEqual({
    status: 200,
    cycle_id: null,
    module_ids: [docs.id],
  });
  const joined = await lastWrite(db, guide.id);
  expect(joined.by, "who put Write the docs in Docs").toBe(admin.email);
  await expectModuleRows(db, guide.id, [
    { target: "Docs", added: joined.at, removed: null, adder: admin.email, by: admin.email },
  ]);
  await expectPlanningActivity(db, guide.id, wrote(admin.email, joined.at, moduleJoined("Docs")));

  // mem replaces Fix the login's modules with Docs and Payments: Payments' row left as it is, Search's removed and
  // Docs' added at one moment, his; Docs joined, then Search left.
  expect(await update(browser, fix.id, { module_ids: [docs.id, payments.id] }), "mem replaces its modules").toEqual({
    status: 200,
    cycle_id: null,
    module_ids: ascendingIds(docs, payments),
  });
  const replaced = await lastWrite(db, fix.id);
  expect(replaced.by, "who replaced Fix the login's modules").toBe(mem.email);
  await expectModuleRows(db, fix.id, [
    { target: "Payments", added: created.at, removed: null, adder: mem.email, by: mem.email },
    { target: "Search", added: created.at, removed: replaced.at, adder: mem.email, by: mem.email },
    { target: "Docs", added: replaced.at, removed: null, adder: mem.email, by: mem.email },
  ]);
  fixActivity = [...fixActivity, ...wrote(mem.email, replaced.at, moduleJoined("Docs"), moduleLeft("Search"))];
  await expectPlanningActivity(db, fix.id, fixActivity);

  // Its own modules in another order change nothing: answered as it is, nothing written.
  let rows = await workItemRows(db, web.id);
  expect(
    await update(admin.token, fix.id, { module_ids: ascendingIds(docs, payments).toReversed() }),
    "the admin gives it its modules in another order"
  ).toEqual({ status: 200, cycle_id: null, module_ids: ascendingIds(docs, payments) });
  expect(await workItemRows(db, web.id), "Web's work items' rows after its own modules").toEqual(rows);

  // Refused, each writing nothing, as a change and as a creation: Old archived (409, before a wrong label's 422);
  // deleted Gone, Ops' Infra and a module not there at their places in module_ids, with the other problems; a module
  // named twice, and a hundred and one modules, before the project is looked at.
  const ghost = randomUUID();
  expect(
    [
      await update(browser, fix.id, { module_ids: [payments.id, old.id] }),
      await update(browser, fix.id, { module_ids: [old.id], label_ids: [ghost] }),
      await update(browser, fix.id, { module_ids: [gone.id, payments.id] }),
      await update(admin.token, fix.id, { module_ids: [payments.id, infra.id, ghost], label_ids: [ghost] }),
      await update(browser, fix.id, { module_ids: [payments.id, docs.id, payments.id] }),
      await update(browser, fix.id, { module_ids: Array.from({ length: 101 }, () => randomUUID()) }),
    ],
    "Fix the login put in Old, Old with a wrong label, Gone, Infra and nowhere with a wrong label, Payments twice, and a hundred and one modules"
  ).toEqual([
    { status: 409, code: "planning.archived" },
    { status: 409, code: "planning.archived" },
    validationFailed(["module_ids[0]", "not_allowed"]),
    validationFailed(
      ["label_ids[0]", "not_allowed"],
      ["module_ids[1]", "not_allowed"],
      ["module_ids[2]", "not_allowed"]
    ),
    validationFailed(["module_ids[2]", "duplicate"]),
    validationFailed(["module_ids", "too_long"]),
  ]);
  expect(
    [
      await create(browser, web.id, { name: "Late", module_ids: [old.id] }),
      await create(admin.token, web.id, { name: "Late", module_ids: [payments.id, gone.id] }),
      await create(browser, web.id, { name: "Late", module_ids: [infra.id] }),
    ],
    "Late created in Old, in Payments and Gone, and in Infra"
  ).toEqual([
    { status: 409, code: "planning.archived" },
    validationFailed(["module_ids[1]", "not_allowed"]),
    validationFailed(["module_ids[0]", "not_allowed"]),
  ]);
  expect(await workItemRows(db, web.id), "Web's work items' rows after the refusals").toEqual(rows);

  // The admin archives Docs. mem puts Search back with Docs and Payments: Docs, kept, is not checked (R3); Search gets
  // a row of its own. Then he leaves Docs, archived, and Payments: their rows removed at that moment, his.
  expect(await modules.archive(admin.token, docs.id), "the admin archives Docs").toEqual({ status: 200 });
  expect(
    await update(browser, fix.id, { module_ids: [search.id, docs.id, payments.id] }),
    "mem puts Search back"
  ).toEqual({ status: 200, cycle_id: null, module_ids: ascendingIds(search, docs, payments) });
  const back = await lastWrite(db, fix.id);
  expect(await update(browser, fix.id, { module_ids: [search.id] }), "mem leaves Docs and Payments").toEqual({
    status: 200,
    cycle_id: null,
    module_ids: [search.id],
  });
  const kept = await lastWrite(db, fix.id);
  await expectModuleRows(db, fix.id, [
    { target: "Payments", added: created.at, removed: kept.at, adder: mem.email, by: mem.email },
    { target: "Search", added: created.at, removed: replaced.at, adder: mem.email, by: mem.email },
    { target: "Docs", added: replaced.at, removed: kept.at, adder: mem.email, by: mem.email },
    { target: "Search", added: back.at, removed: null, adder: mem.email, by: mem.email },
  ]);
  fixActivity = [
    ...fixActivity,
    ...wrote(mem.email, back.at, moduleJoined("Search")),
    ...wrote(mem.email, kept.at, ...leaves(docs, payments)),
  ];
  await expectPlanningActivity(db, fix.id, fixActivity);

  // Web's modules off: a change of the modules, leaving one too, and a creation in one are refused; Fix the login given
  // its own modules is answered as it is. Nothing is written. Turned on again, its modules are all there.
  await turn(web.id, false);
  rows = await workItemRows(db, web.id);
  expect(
    [
      await update(browser, fix.id, { module_ids: [] }),
      await update(admin.token, fix.id, { module_ids: [search.id, payments.id] }),
      await create(browser, web.id, { name: "Late", module_ids: [payments.id] }),
      await update(browser, fix.id, { module_ids: [search.id] }),
    ],
    "with Web's modules off, Fix the login emptied and given Payments, Late created in Payments, and Fix the login given Search"
  ).toEqual([
    ...Array(3).fill({ status: 409, code: "planning.modules_disabled" }),
    { status: 200, cycle_id: null, module_ids: [search.id] },
  ]);
  expect(await workItemRows(db, web.id), "Web's work items' rows, its modules off").toEqual(rows);
  await turn(web.id, true);

  // mem empties Fix the login's modules: Search's second row removed at that moment, his.
  expect(await update(browser, fix.id, { module_ids: [] }), "mem empties its modules").toEqual({
    status: 200,
    cycle_id: null,
    module_ids: [],
  });
  const emptied = await lastWrite(db, fix.id);
  fixActivity = [...fixActivity, ...wrote(mem.email, emptied.at, moduleLeft("Search"))];
  await expectPlanningActivity(db, fix.id, fixActivity);
  await expectModuleRows(db, fix.id, [
    { target: "Payments", added: created.at, removed: kept.at, adder: mem.email, by: mem.email },
    { target: "Search", added: created.at, removed: replaced.at, adder: mem.email, by: mem.email },
    { target: "Docs", added: replaced.at, removed: kept.at, adder: mem.email, by: mem.email },
    { target: "Search", added: back.at, removed: emptied.at, adder: mem.email, by: mem.email },
  ]);

  // The work items, one by one and in Web's list, read alike to either of mem's tokens, and to the admin.
  const reads = async (token: string) => [
    await read(token, fix.id),
    await read(token, guide.id),
    ((await listIssues(api, token, web.id)).data ?? [])
      .map(({ name, cycle_id, module_ids }) => ({ name, cycle_id, module_ids }))
      .toSorted((a, b) => a.name.localeCompare(b.name)),
  ];
  const seen = await reads(browser);
  expect(seen, "the work items as mem reads them").toEqual([
    { status: 200, cycle_id: null, module_ids: [] },
    { status: 200, cycle_id: null, module_ids: [docs.id] },
    [
      { name: "Fix the login", cycle_id: null, module_ids: [] },
      { name: "Write the docs", cycle_id: null, module_ids: [docs.id] },
    ],
  ]);
  expect(await reads(mem.token), "the same with mem's personal access token").toEqual(seen);
  expect(await reads(admin.token), "the same to the admin").toEqual(seen);
});
