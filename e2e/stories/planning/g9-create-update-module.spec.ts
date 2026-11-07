import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  projectMemberWrites,
  slugFor,
} from "../../fixtures/api";
import {
  expectModuleMembers,
  expectModules,
  moduleMoments,
  type ModuleMemberRow,
  type ModuleRow,
} from "../../fixtures/assert/planning";
import { login, newAccount } from "../../fixtures/auth";
import { createModule, listModules, moduleCalls, readModule } from "../../fixtures/modules";
import { expect, test } from "../../fixtures/test";

// G9, creating and changing a project's modules (M6 design 3.2, M6 P2 spec 2.3, R2, R4, R5, R8): the name, unique in
// the project whatever its case, archived modules' too; the six statuses; the days; the lead and the members, active
// members of the project who are not its guests, a change looking only at those it makes anew. The admin calls with a
// personal access token, the member with the token of his sign-in, as the browser does.

/** The row of a module of name created by creator, of a name alone, and written by him last, as to says otherwise. */
function moduleRow(name: string, creator: string, to: Partial<ModuleRow>): ModuleRow {
  return {
    name,
    description: "",
    start: null,
    target: null,
    status: "planned",
    lead: null,
    members: [],
    sort_order: 65535,
    archived: false,
    deleted: false,
    creator,
    by: creator,
    ...to,
  };
}

test("G9 (API): Web's modules are off, so its member's Release 1 is refused; its admin turns them on; the member creates Release 1 from its name, planned, at the first place; the admin creates Release 2 with its days, a status, a lead and members, before it, and Old, completed, which he archives; a blank name, a status not of the six, a target before the start, a guest, a non-member or one who left Web as lead or member, a member named twice, Release 1's and archived Old's names in other cases and a guest's module are refused, each changing nothing; of two modules of one name in two cases created at once one is; the member lists the same modules with either of his tokens", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  const [mem, guest, other, gone] = await Promise.all(
    ["mem", "guest", "other", "gone"].map((label) => newAccount(api, testInfo, label))
  );
  if (!mem || !guest || !other || !gone) {
    throw new Error("the accounts were not registered");
  }
  await inviteAndAccept(api, admin.token, slug, mem, 15);
  await inviteAndAccept(api, admin.token, slug, guest, 5);
  await inviteAndAccept(api, admin.token, slug, other, 15);
  await inviteAndAccept(api, admin.token, slug, gone, 15);
  const web = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: mem.id, role: 15 },
    { member_id: guest.id, role: 5 },
    { member_id: gone.id, role: 15 },
  ]);
  // gone leaves Web; other, Acme's member, never was one of it.
  expect(await projectMemberWrites(api, web.id).leave(gone.token), "gone leaves Web").toEqual({ status: 204 });
  // mem's token of his sign-in, the browser's, beside his personal access token.
  const browser = (await login(api, mem.email)).access_token;
  const { create, archive } = moduleCalls(api);

  // A new project's modules are off (M6 design 3.8): mem's Release 1 is refused, and nothing is written.
  expect(await create(browser, web.id, { name: "Release 1" }), "mem's Release 1, modules off").toEqual({
    status: 409,
    code: "planning.modules_disabled",
  });
  await expectModules(db, web.id, []);
  const on = await changeProject(api, admin.token, web.id, { module_view: true });
  expect(on.status, `the admin turns Web's modules on: ${JSON.stringify(on)}`).toBe(200);

  // mem creates Release 1 from its name alone, without its surrounding spaces: planned, without days, a lead or
  // members, at a project's first place, created by him.
  const release1 = await createModule(api, browser, web.id, { name: "  Release 1 " });
  expect(
    [
      release1.name,
      release1.description,
      release1.start_date,
      release1.target_date,
      release1.status,
      release1.lead_id,
      release1.member_ids,
      release1.sort_order,
      release1.project_id,
      release1.workspace_id === web.workspace_id,
      release1.created_by_id,
      release1.updated_by_id,
      release1.archived_at,
    ],
    "Release 1 as created"
  ).toEqual(["Release 1", "", null, null, "planned", null, [], 65535, web.id, true, mem.id, mem.id, null]);
  // The admin creates Release 2 with a description, its days, a status, mem its lead and both of them its members,
  // answered in their ids' order, before Release 1; and Old, completed, of one day, before both, which he archives:
  // an archived module keeps its name from the others' (R8).
  const release2 = await createModule(api, admin.token, web.id, {
    name: "Release 2",
    description: "Checkout",
    start_date: "2026-12-01",
    target_date: "2026-12-31",
    status: "in-progress",
    lead_id: mem.id,
    member_ids: [mem.id, admin.id],
  });
  expect(
    [release2.status, release2.lead_id, release2.member_ids, release2.sort_order, release2.created_by_id],
    "Release 2 as created"
  ).toEqual(["in-progress", mem.id, [mem.id, admin.id].toSorted(), 55535, admin.id]);
  const old = await createModule(api, admin.token, web.id, {
    name: "Old",
    status: "completed",
    start_date: "2025-01-01",
    target_date: "2025-01-01",
  });
  expect(await archive(admin.token, old.id), "the admin archives Old").toEqual({ status: 200 });
  let modules: ModuleRow[] = [
    moduleRow("Release 1", mem.email, {}),
    moduleRow("Release 2", admin.email, {
      description: "Checkout",
      start: "2026-12-01",
      target: "2026-12-31",
      status: "in-progress",
      lead: mem.email,
      members: [admin.email, mem.email],
      sort_order: 55535,
    }),
    moduleRow("Old", admin.email, {
      start: "2025-01-01",
      target: "2025-01-01",
      status: "completed",
      sort_order: 45535,
      archived: true,
    }),
  ];
  await expectModules(db, web.id, modules);

  // Refused, each changing nothing: a name of spaces; a status in another case; a target before the start; the guest
  // as lead; the guest, other and gone as members, each at its place, mem among them passing; mem named twice; Release
  // 1's name and archived Old's, each in another case; the guest's module.
  expect(
    [
      await create(browser, web.id, { name: "   " }),
      await create(browser, web.id, { name: "Release 3", status: "Planned" as never }),
      await create(browser, web.id, { name: "Release 3", start_date: "2027-01-31", target_date: "2027-01-01" }),
      await create(browser, web.id, { name: "Release 3", lead_id: guest.id }),
      await create(admin.token, web.id, { name: "Release 3", member_ids: [mem.id, guest.id, other.id, gone.id] }),
      await create(browser, web.id, { name: "Release 3", member_ids: [mem.id, admin.id, mem.id] }),
      await create(admin.token, web.id, { name: "release 1" }),
      await create(browser, web.id, { name: "OLD" }),
      await create(guest.token, web.id, { name: "Guest's" }),
    ],
    "a blank name, a status, a target before the start, the guest lead, three members, mem twice, release 1, OLD, the guest's"
  ).toEqual([
    { status: 422, code: "validation_failed", errors: [{ field: "name", code: "too_short" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "status", code: "invalid_format" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "target_date", code: "not_allowed" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "lead_id", code: "not_allowed" }] },
    {
      status: 422,
      code: "validation_failed",
      errors: [
        { field: "member_ids[1]", code: "not_allowed" },
        { field: "member_ids[2]", code: "not_allowed" },
        { field: "member_ids[3]", code: "not_allowed" },
      ],
    },
    { status: 422, code: "validation_failed", errors: [{ field: "member_ids[2]", code: "duplicate" }] },
    { status: 409, code: "planning.module_name_taken" },
    { status: 409, code: "planning.module_name_taken" },
    { status: 403, code: "forbidden" },
  ]);
  await expectModules(db, web.id, modules);

  // The admin's Release 3 and mem's RELEASE 3, created at once: the second, decided after the first under the
  // project's lock, finds its name taken (R8).
  const raced = await Promise.all([
    create(admin.token, web.id, { name: "Release 3" }),
    create(browser, web.id, { name: "RELEASE 3" }),
  ]);
  expect(
    raced.toSorted((a, b) => a.status - b.status),
    "the admin's Release 3 and mem's RELEASE 3, created at once"
  ).toEqual([{ status: 201 }, { status: 409, code: "planning.module_name_taken" }]);
  const [release3, owner] = raced[0]?.status === 201 ? ["Release 3", admin.email] : ["RELEASE 3", mem.email];
  modules = [...modules, moduleRow(release3, owner, { sort_order: 35535 })];
  await expectModules(db, web.id, modules);

  // mem lists Web's modules by their places, archived Old aside, the same with either of his tokens; Old is in its
  // archived catalog.
  const listed = await listModules(api, browser, web.id);
  expect(
    listed.map((m) => [m.name, m.status, m.member_ids]),
    "Web's modules as mem lists them"
  ).toEqual([
    [release3, "planned", []],
    ["Release 2", "in-progress", [mem.id, admin.id].toSorted()],
    ["Release 1", "planned", []],
  ]);
  expect(await listModules(api, mem.token, web.id), "Web's modules as mem's personal access token lists them").toEqual(
    listed
  );
  expect(
    (await listModules(api, mem.token, web.id, true)).map((m) => m.name),
    "Web's archived modules"
  ).toEqual(["Old"]);
});

test("G9 (API): Web's member renames Payments Checkout and describes it, its lead and member who left Web kept; he replaces its members, the one he keeps left as it is, the one he removes removed and the one he adds added at one moment; the admin makes himself its lead; the one who left made its lead again, a guest or a non-member made its lead or added, a start after its target, a target before its start, a status not of the six, archived Old's name and a guest's change are refused, each changing nothing; an empty change, its own values and its members in another order write nothing; its own name in another case, a status, a place and its days and lead cleared are written; archived Old refuses any change", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  const [mem, mate, guest, other] = await Promise.all(
    ["mem", "mate", "guest", "other"].map((label) => newAccount(api, testInfo, label))
  );
  if (!mem || !mate || !guest || !other) {
    throw new Error("the accounts were not registered");
  }
  await inviteAndAccept(api, admin.token, slug, mem, 15);
  await inviteAndAccept(api, admin.token, slug, mate, 15);
  await inviteAndAccept(api, admin.token, slug, guest, 5);
  await inviteAndAccept(api, admin.token, slug, other, 15);
  const web = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: mem.id, role: 15 },
    { member_id: mate.id, role: 15 },
    { member_id: guest.id, role: 5 },
  ]);
  const on = await changeProject(api, admin.token, web.id, { module_view: true });
  expect(on.status, `the admin turns Web's modules on: ${JSON.stringify(on)}`).toBe(200);
  const browser = (await login(api, mem.email)).access_token;
  const { update, archive } = moduleCalls(api);

  // mem creates Payments, led by mate, mate and the admin its members; the admin creates Old, cancelled, and archives
  // it. Then mate leaves Web: Payments keeps him as its lead and its member (M6 design 3.2).
  const payments = await createModule(api, browser, web.id, {
    name: "Payments",
    lead_id: mate.id,
    member_ids: [mate.id, admin.id],
  });
  const old = await createModule(api, admin.token, web.id, { name: "Old", status: "cancelled" });
  expect(await archive(admin.token, old.id), "the admin archives Old").toEqual({ status: 200 });
  const created = (await moduleMoments(db, web.id))["Payments"] ?? "";
  expect(await projectMemberWrites(api, web.id).leave(mate.token), "mate leaves Web").toEqual({ status: 204 });
  let modules: ModuleRow[] = [
    moduleRow("Payments", mem.email, { lead: mate.email, members: [admin.email, mate.email] }),
    moduleRow("Old", admin.email, { status: "cancelled", sort_order: 55535, archived: true }),
  ];
  let members: ModuleMemberRow[] = [
    { member: admin.email, added: created, removed: null, adder: mem.email, by: mem.email },
    { member: mate.email, added: created, removed: null, adder: mem.email, by: mem.email },
  ];
  await expectModules(db, web.id, modules);
  await expectModuleMembers(db, payments.id, members);

  // mem renames Payments Checkout and describes it: its lead and members, mate among them, are not looked at, and
  // their rows not written (R5).
  expect(
    await update(browser, payments.id, { name: "Checkout", description: "Pay and ship" }),
    "mem renames Payments"
  ).toEqual({ status: 200 });
  modules = [{ ...modules[0]!, name: "Checkout", description: "Pay and ship" }, modules[1]!];
  await expectModules(db, web.id, modules);
  await expectModuleMembers(db, payments.id, members);

  // mem makes Checkout's members himself and mate: mate's row is kept as it is, though he left; the admin's is removed
  // and mem's added, at the moment of the change, by mem.
  expect(await update(browser, payments.id, { member_ids: [mem.id, mate.id] }), "mem replaces the members").toEqual({
    status: 200,
  });
  const replaced = (await moduleMoments(db, web.id))["Checkout"] ?? "";
  expect(replaced, "the moment of the members' change").not.toBe(created);
  modules = [{ ...modules[0]!, members: [mate.email, mem.email] }, modules[1]!];
  members = [
    { ...members[0]!, removed: replaced },
    members[1]!,
    { member: mem.email, added: replaced, removed: null, adder: mem.email, by: mem.email },
  ];
  await expectModules(db, web.id, modules);
  await expectModuleMembers(db, payments.id, members);

  // The admin makes himself Checkout's lead; mem gives it its days.
  expect(await update(admin.token, payments.id, { lead_id: admin.id }), "the admin leads Checkout").toEqual({
    status: 200,
  });
  expect(
    await update(browser, payments.id, { start_date: "2099-03-01", target_date: "2099-03-31" }),
    "mem gives Checkout its days"
  ).toEqual({ status: 200 });
  modules = [{ ...modules[0]!, lead: admin.email, start: "2099-03-01", target: "2099-03-31" }, modules[1]!];
  await expectModules(db, web.id, modules);

  // Refused, each changing nothing: mate, who left, made its lead anew, and the guest; the guest and other added, each
  // at its place, mate kept passing; a start alone after its target, at the start; a target alone before its start,
  // at the target; a status in another case; archived Old's name in another case; the guest's change.
  expect(
    [
      await update(browser, payments.id, { lead_id: mate.id }),
      await update(browser, payments.id, { lead_id: guest.id }),
      await update(browser, payments.id, { member_ids: [mem.id, mate.id, guest.id, other.id] }),
      await update(browser, payments.id, { start_date: "2099-04-01" }),
      await update(browser, payments.id, { target_date: "2099-02-01" }),
      await update(browser, payments.id, { status: "Paused" as never }),
      await update(browser, payments.id, { name: "old" }),
      await update(guest.token, payments.id, { name: "Guest's" }),
    ],
    "mate and the guest as lead, the guest and other added, the days, a status, old, the guest's change"
  ).toEqual([
    { status: 422, code: "validation_failed", errors: [{ field: "lead_id", code: "not_allowed" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "lead_id", code: "not_allowed" }] },
    {
      status: 422,
      code: "validation_failed",
      errors: [
        { field: "member_ids[2]", code: "not_allowed" },
        { field: "member_ids[3]", code: "not_allowed" },
      ],
    },
    { status: 422, code: "validation_failed", errors: [{ field: "start_date", code: "not_allowed" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "target_date", code: "not_allowed" }] },
    { status: 422, code: "validation_failed", errors: [{ field: "status", code: "invalid_format" }] },
    { status: 409, code: "planning.module_name_taken" },
    { status: 403, code: "forbidden" },
  ]);
  await expectModules(db, web.id, modules);
  await expectModuleMembers(db, payments.id, members);

  // An empty change, one to its own values, and its members named in another order answer it as it is and write
  // nothing: not its last change, nor who made it, nor a member's row (R4).
  const before = await moduleMoments(db, web.id);
  expect(
    [
      await update(browser, payments.id, {}),
      await update(browser, payments.id, { name: "Checkout", description: "Pay and ship", lead_id: admin.id }),
      await update(browser, payments.id, { member_ids: [mate.id, mem.id] }),
    ],
    "mem's empty change, his change to its own values, and to its members in another order"
  ).toEqual([{ status: 200 }, { status: 200 }, { status: 200 }]);
  expect(await moduleMoments(db, web.id), "the modules' last changes").toEqual(before);
  await expectModules(db, web.id, modules);
  await expectModuleMembers(db, payments.id, members);

  // mem gives Checkout its own name in another case, a status and a place, then clears its days and its lead.
  expect(
    [
      await update(browser, payments.id, { name: "CHECKOUT", status: "paused", sort_order: 1000 }),
      await update(browser, payments.id, { start_date: null, target_date: null, lead_id: null }),
    ],
    "mem's CHECKOUT, paused, at 1000; its days and lead cleared"
  ).toEqual([{ status: 200 }, { status: 200 }]);
  modules = [
    {
      ...modules[0]!,
      name: "CHECKOUT",
      status: "paused",
      sort_order: 1000,
      start: null,
      target: null,
      lead: null,
      by: mem.email,
    },
    modules[1]!,
  ];
  await expectModules(db, web.id, modules);
  await expectModuleMembers(db, payments.id, members);
  const read = await readModule(api, browser, payments.id);
  expect(
    [read.name, read.status, read.lead_id, read.member_ids, read.sort_order, read.updated_by_id],
    "CHECKOUT as mem reads it"
  ).toEqual(["CHECKOUT", "paused", null, [mem.id, mate.id].toSorted(), 1000, mem.id]);
  expect(await readModule(api, mem.token, payments.id), "CHECKOUT as mem's personal access token reads it").toEqual(
    read
  );

  // Archived Old refuses any change, an empty one too.
  expect(
    [await update(browser, old.id, { sort_order: 1 }), await update(admin.token, old.id, {})],
    "archived Old moved, and changed by nothing"
  ).toEqual([
    { status: 409, code: "planning.archived" },
    { status: 409, code: "planning.archived" },
  ]);
  await expectModules(db, web.id, modules);
});
