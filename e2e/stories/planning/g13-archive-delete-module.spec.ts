import {
  addProjectMembers,
  changeProject,
  createProject,
  createWorkspace,
  inviteAndAccept,
  slugFor,
} from "../../fixtures/api";
import {
  expectModuleLinks,
  expectModuleMembers,
  expectModules,
  moduleLinkMoments,
  moduleMoments,
  type ModuleRow,
} from "../../fixtures/assert/planning";
import { login, newAccount } from "../../fixtures/auth";
import { createModule, createModuleLink, listModules, moduleCalls, readModule } from "../../fixtures/modules";
import { expect, test } from "../../fixtures/test";

// G13, archiving a project's modules, restoring them and deleting them (M6 design 3.6, M6 P2 spec R3, 2.4): a
// completed or cancelled module alone is archived; its project's archived catalog, the last archived first; a
// deletion for the project's admins and a module's creator, taking its members and links at its moment. The admin
// calls with a personal access token, the member with the token of his sign-in.

/** rows with the module of name changed as to says. */
function changed(rows: ModuleRow[], name: string, to: Partial<ModuleRow>): ModuleRow[] {
  return rows.map((m) => (m.name === name ? { ...m, ...to } : m));
}

test("G13 (API): of Web's modules, those in the backlog, planned, in progress or paused may not be archived, nor may a guest archive Done; the member archives Done, completed, a second archive writing nothing, and the admin Dropped, cancelled; they leave Web's list for its archived catalog, the last archived first, and Done is still read by its id but refuses a change and a link; each is restored, whatever its status, a second restore writing nothing, and changed again; the member may not delete Dropped, the admin's, nor the guest Launch; the admin deletes Launch, the member's, with its members and links at one moment, those removed before keeping their own, after which it and its links are not found and its name is free; the member deletes Done, his own, and the admin Dropped, archived", async ({
  api,
  db,
}, testInfo) => {
  const admin = await newAccount(api, testInfo, "admin");
  const slug = slugFor(testInfo);
  await createWorkspace(api, admin.token, { name: "Acme", slug });
  const [mem, guest] = await Promise.all(["mem", "guest"].map((label) => newAccount(api, testInfo, label)));
  if (!mem || !guest) {
    throw new Error("the accounts were not registered");
  }
  await inviteAndAccept(api, admin.token, slug, mem, 15);
  await inviteAndAccept(api, admin.token, slug, guest, 5);
  const web = await createProject(api, admin.token, slug, { name: "Web", identifier: "WEB" });
  await addProjectMembers(api, admin.token, web.id, [
    { member_id: mem.id, role: 15 },
    { member_id: guest.id, role: 5 },
  ]);
  const on = await changeProject(api, admin.token, web.id, { module_view: true });
  expect(on.status, `the admin turns Web's modules on: ${JSON.stringify(on)}`).toBe(200);
  const browser = (await login(api, mem.email)).access_token;
  const { update, archive, unarchive, remove, read, links, addLink, changeLink, removeLink } = moduleCalls(api);

  // The admin creates a module of each status that is not closed, Planned's by default, then Dropped, cancelled; mem
  // creates Done, completed, and Launch, which he leads, the admin and he its members, with two links. Each is created
  // before the one before it.
  const backlog = await createModule(api, admin.token, web.id, { name: "Backlog", status: "backlog" });
  const planned = await createModule(api, admin.token, web.id, { name: "Planned" });
  const doing = await createModule(api, admin.token, web.id, { name: "Doing", status: "in-progress" });
  const paused = await createModule(api, admin.token, web.id, { name: "Paused", status: "paused" });
  const dropped = await createModule(api, admin.token, web.id, { name: "Dropped", status: "cancelled" });
  const done = await createModule(api, browser, web.id, { name: "Done", status: "completed" });
  const launch = await createModule(api, browser, web.id, {
    name: "Launch",
    lead_id: mem.id,
    member_ids: [mem.id, admin.id],
  });
  const created = (await moduleMoments(db, web.id))["Launch"] ?? "";
  const planURL = "http://169.254.169.254/g13/plan";
  const notesURL = "http://169.254.169.254/g13/notes";
  const plan = await createModuleLink(api, browser, launch.id, { url: planURL });
  const notes = await createModuleLink(api, browser, launch.id, { url: notesURL });
  let modules: ModuleRow[] = [
    ["Backlog", "backlog"],
    ["Planned", "planned"],
    ["Doing", "in-progress"],
    ["Paused", "paused"],
    ["Dropped", "cancelled"],
  ].map(([name, status], i) => ({
    name: name ?? "",
    description: "",
    start: null,
    target: null,
    status: status ?? "",
    lead: null,
    members: [],
    sort_order: 65535 - 10000 * i,
    archived: false,
    deleted: false,
    creator: admin.email,
    by: admin.email,
  }));
  modules = [
    ...modules,
    { ...modules[0]!, name: "Done", status: "completed", sort_order: 15535, creator: mem.email, by: mem.email },
    {
      ...modules[0]!,
      name: "Launch",
      status: "planned",
      lead: mem.email,
      members: [admin.email, mem.email],
      sort_order: 5535,
      creator: mem.email,
      by: mem.email,
    },
  ];
  await expectModules(db, web.id, modules);

  // Refused, each changing nothing: a module in the backlog, planned, in progress or paused is not closed (R3); the
  // guest may not archive.
  expect(
    [
      await archive(browser, backlog.id),
      await archive(admin.token, planned.id),
      await archive(browser, doing.id),
      await archive(admin.token, paused.id),
      await archive(guest.token, done.id),
    ],
    "Backlog, Planned, Doing and Paused archived, and the guest's archive of Done"
  ).toEqual([
    ...Array(4).fill({ status: 409, code: "planning.module_not_closed" }),
    { status: 403, code: "forbidden" },
  ]);
  await expectModules(db, web.id, modules);

  // mem archives Done, archived and last changed by him at one moment; archiving it again answers it as it is and
  // writes nothing (R3); the admin archives Dropped.
  expect(await archive(browser, done.id), "mem archives Done").toEqual({ status: 200 });
  modules = changed(modules, "Done", { archived: true });
  await expectModules(db, web.id, modules);
  const archivedDone = await moduleMoments(db, web.id);
  expect(await archive(admin.token, done.id), "the admin archives Done again").toEqual({ status: 200 });
  expect(await moduleMoments(db, web.id), "the modules' last changes after Done's second archive").toEqual(
    archivedDone
  );
  expect(await archive(admin.token, dropped.id), "the admin archives Dropped").toEqual({ status: 200 });
  modules = changed(modules, "Dropped", { archived: true });
  await expectModules(db, web.id, modules);

  // Web's list holds the others by their places; its archived catalog Dropped, archived last, then Done, the same
  // with either of mem's tokens. Done is read by its id, archived at its last change, but refuses a change and a link.
  const lists = async (token: string) => [
    (await listModules(api, token, web.id)).map((m) => m.name),
    (await listModules(api, token, web.id, true)).map((m) => m.name),
  ];
  expect(await lists(browser), "Web's list and archived catalog, as mem lists them").toEqual([
    ["Launch", "Paused", "Doing", "Planned", "Backlog"],
    ["Dropped", "Done"],
  ]);
  expect(await lists(mem.token), "the same with mem's personal access token").toEqual(await lists(browser));
  const archived = await readModule(api, browser, done.id);
  expect(
    [archived.status, archived.archived_at !== null && archived.archived_at === archived.updated_at],
    "Done as mem reads it"
  ).toEqual(["completed", true]);
  expect(
    [
      await update(browser, done.id, { status: "in-progress" }),
      await addLink(browser, done.id, { url: "169.254.169.254/g13/done" }),
    ],
    "archived Done changed, and given a link"
  ).toEqual(Array(2).fill({ status: 409, code: "planning.archived" }));
  await expectModules(db, web.id, modules);

  // mem restores Dropped, cancelled, and the admin Done, completed: a restore does not look at the status (R3).
  // Restoring Dropped again writes nothing. mem then puts Done back in progress.
  expect(await unarchive(browser, dropped.id), "mem restores Dropped").toEqual({ status: 200 });
  const restoredDropped = await moduleMoments(db, web.id);
  expect(await unarchive(admin.token, dropped.id), "the admin restores Dropped again").toEqual({ status: 200 });
  expect(await moduleMoments(db, web.id), "the modules' last changes after Dropped's second restore").toEqual(
    restoredDropped
  );
  expect(await unarchive(admin.token, done.id), "the admin restores Done").toEqual({ status: 200 });
  expect(await update(browser, done.id, { status: "in-progress" }), "mem puts Done back in progress").toEqual({
    status: 200,
  });
  modules = changed(modules, "Dropped", { archived: false, by: mem.email });
  modules = changed(modules, "Done", { archived: false, status: "in-progress" });
  await expectModules(db, web.id, modules);
  expect(await lists(browser), "Web's list and archived catalog, both restored").toEqual([
    ["Launch", "Done", "Dropped", "Paused", "Doing", "Planned", "Backlog"],
    [],
  ]);

  // mem removes the admin from Launch's members and deletes its notes link: each keeps its own moment and writer.
  expect(await update(browser, launch.id, { member_ids: [mem.id] }), "mem removes the admin").toEqual({ status: 200 });
  const removed = (await moduleMoments(db, web.id))["Launch"] ?? "";
  expect(await removeLink(browser, notes.id), "mem deletes the notes link").toEqual({ status: 204 });
  const notesDeleted = (await moduleLinkMoments(db, launch.id))[notesURL] ?? "";
  modules = changed(modules, "Launch", { members: [mem.email] });
  await expectModules(db, web.id, modules);

  // Refused, each changing nothing: mem's deletion of Dropped, the admin's; the guest's of Launch.
  expect(
    [await remove(browser, dropped.id), await remove(guest.token, launch.id)],
    "mem deletes Dropped, the guest Launch"
  ).toEqual(Array(2).fill({ status: 403, code: "forbidden" }));
  await expectModules(db, web.id, modules);

  // The admin deletes Launch, mem's: the module, its member's row and its plan link deleted at one moment, by him;
  // the admin's row and the notes link, removed before, keep their own (M6 P2 spec 2.4).
  expect(await remove(admin.token, launch.id), "the admin deletes Launch").toEqual({ status: 204 });
  const deleted = (await moduleMoments(db, web.id))["Launch"] ?? "";
  expect([deleted === removed, deleted === notesDeleted], "Launch's deletion, a moment of its own").toEqual([
    false,
    false,
  ]);
  modules = changed(modules, "Launch", { members: [], deleted: true, by: admin.email });
  await expectModules(db, web.id, modules);
  await expectModuleMembers(db, launch.id, [
    { member: admin.email, added: created, removed, adder: mem.email, by: mem.email },
    { member: mem.email, added: created, removed: deleted, adder: mem.email, by: admin.email },
  ]);
  await expectModuleLinks(db, launch.id, [
    { url: planURL, title: null, adder: mem.email, by: admin.email, removed: deleted },
    { url: notesURL, title: null, adder: mem.email, by: mem.email, removed: notesDeleted },
  ]);
  // Launch is not found: read, its links listed, deleted or restored again; nor is its plan link, changed or deleted.
  expect(
    [
      await read(browser, launch.id),
      await links(browser, launch.id),
      await remove(admin.token, launch.id),
      await unarchive(admin.token, launch.id),
      await changeLink(browser, plan.id, { title: "Plan" }),
      await removeLink(admin.token, plan.id),
    ],
    "deleted Launch read, its links listed, deleted and restored again, and its plan link changed and deleted"
  ).toEqual([
    ...Array(4).fill({ status: 404, code: "planning.module_not_found" }),
    ...Array(2).fill({ status: 404, code: "planning.module_link_not_found" }),
  ]);
  await expectModules(db, web.id, modules);

  // mem deletes Done, his own; the admin archives Dropped again and deletes it archived. mem creates launch: a deleted
  // module's name is free, and its place does not count.
  expect(await remove(browser, done.id), "mem deletes Done").toEqual({ status: 204 });
  expect(await archive(admin.token, dropped.id), "the admin archives Dropped again").toEqual({ status: 200 });
  expect(await remove(admin.token, dropped.id), "the admin deletes archived Dropped").toEqual({ status: 204 });
  const again = await createModule(api, browser, web.id, { name: "launch" });
  expect(again.sort_order, "launch's place, after the undeleted modules'").toBe(25535);
  modules = changed(modules, "Done", { deleted: true });
  modules = changed(modules, "Dropped", { archived: true, deleted: true, by: admin.email });
  modules = [
    ...modules,
    { ...modules[0]!, name: "launch", status: "planned", sort_order: 25535, creator: mem.email, by: mem.email },
  ];
  await expectModules(db, web.id, modules);
  expect(await lists(mem.token), "Web's list and archived catalog at the end").toEqual([
    ["launch", "Paused", "Doing", "Planned", "Backlog"],
    [],
  ]);
});
