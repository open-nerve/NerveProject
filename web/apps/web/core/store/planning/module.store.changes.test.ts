/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Module, ModuleCreate, ModuleLink, ModuleUpdate } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import type { Endpoint } from "@/lib/auth/fake-nerve";
import { FakeNerve, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { listTab, ops, web } from "@/store/fake-project-lists";
import { fetchedWhileChangeIsOut, inTurn, sent } from "@/store/fake-queue";
import {
  loadArchivedModules,
  loadModule,
  loadModuleLinks,
  loadModuleProgress,
  loadModules,
  moduleLinkOf,
  moduleProgressOf,
  moduleRequests,
  moduleRowOf,
} from "@/store/planning/fake-modules";
import { ModuleStore } from "@/store/planning/module.store";
import { moduleOf } from "@/store/project/fake-projects";

// The changes of the modules and their links (M6 design 3.2, 3.6; P6a spec 2.2, R6, R9, R11; v0 design 7.7), against a
// fake nerve that answers each request when the test says. They go one at a time; a change of the members, a deletion,
// and a change or removal of a link find their module or link in their turn; nerve's answer is written into every copy
// that holds the module (its lists' rows, its own read) or the link, and a row keeps its counts, which only a list gives
// (a Module has none). The tab's address is web's, a project of acme; ops is acme's other.

const { list, at, archive, unarchive, links, link: linkAt } = moduleRequests;

const auth = moduleOf(web, "Auth", {
  status: "in-progress",
  lead_id: "u-ann",
  // gone has left web since: nerve keeps him a member (R6)
  member_ids: ["u-ann", "u-bob", "u-gone"],
  sort_order: 45535,
});
const checkout = moduleOf(web, "Checkout", { status: "completed", sort_order: 55535 });
const search = moduleOf(web, "Search");
const shelved = moduleOf(web, "Shelved", { status: "cancelled", archived_at: "2026-10-20T09:00:00Z" });
/** Times after the fixtures', of nerve's answers to the changes. */
const LATER = "2026-11-10T09:00:00Z";
const LATEST = "2026-11-10T10:00:00Z";
/** web's rows, each with counts of its own, which a change must keep (R9). */
const authRow = moduleRowOf(auth, { total_issues: 4, started_issues: 2, completed_issues: 1, cancelled_issues: 1 });
const checkoutRow = moduleRowOf(checkout, { total_issues: 2, completed_issues: 2 });
const searchRow = moduleRowOf(search);
const shelvedRow = moduleRowOf(shelved, { total_issues: 3, cancelled_issues: 3 });
const authProgress = moduleProgressOf({ total_issues: 4, started_issues: 2, completed_issues: 1, cancelled_issues: 1 });
/** Auth's links, the latest added first: docs, whose metadata nerve has read, and notes, whose it has not yet. */
const docs = moduleLinkOf(auth, "https://docs.example.com", { title: "Docs", metadata: { title: "Docs home" } });
const notes = moduleLinkOf(auth, "https://notes.example.com", { created_at: "2026-09-30T09:00:00Z" });

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

/** A tab whose store holds web's two lists, and Auth's own read, progress and links. */
async function loadedModules() {
  const tab = await listTab(ModuleStore);
  await loadModules(tab.nerve, tab.store, [authRow, checkoutRow, searchRow]);
  await loadArchivedModules(tab.nerve, tab.store, [shelvedRow]);
  await loadModule(tab.nerve, tab.store, auth);
  await loadModuleProgress(tab.nerve, tab.store, auth, authProgress);
  await loadModuleLinks(tab.nerve, tab.store, auth, [docs, notes]);
  return tab;
}

/** Every value the store holds of web's modules: its two lists, each module's own read, Auth's progress and links. */
const held = (store: ModuleStore) => ({
  unarchived: store.unarchived.get(web.id),
  archived: store.archived.get(web.id),
  details: [auth, checkout, search, shelved].map((item) => store.details.get(item.id)),
  progress: store.progress.get(auth.id),
  links: store.links.get(auth.id),
});

describe("ModuleStore, the changes", () => {
  it("creates a module: first of its project's list, of no work items", async () => {
    const { nerve, store } = await loadedModules();
    const body: ModuleCreate = { name: "Onboarding", status: "backlog" };
    const made = moduleOf(web, "Onboarding", { sort_order: 35535, created_at: LATER, updated_at: LATER });
    const creating = await sent(nerve, () => store.create(web.id, body), ["POST", list(web)], json(201, made));
    expect(nerve.calls.at(-1)?.body).toEqual(body);
    expect(creating.value).toEqual(made);
    expect(store.getProjectModuleIds(web.id)).toEqual([made.id, auth.id, checkout.id, search.id]);
    expect(store.getModuleRow(made.id)).toEqual(moduleRowOf(made));
    expect(store.getModuleRow(auth.id)).toEqual(authRow);

    // of a project whose modules the store has not fetched: nothing is written, the fetch brings them
    const infra = moduleOf(ops, "Infra");
    await sent(nerve, () => store.create(ops.id, { name: "Infra" }), ["POST", list(ops)], json(201, infra));
    expect(store.getProjectModuleIds(ops.id)).toBeUndefined();
  });

  it("changes a module: nerve's answer in every copy, its row's and its read; a row keeps its counts (R9)", async () => {
    const { nerve, store } = await loadedModules();
    const body: ModuleUpdate = { name: "Sign-in", status: "planned", target_date: "2026-12-04" };
    const changed: Module = { ...auth, ...body, updated_at: LATER };
    const changing = await sent(nerve, () => store.update(auth.id, body), ["PATCH", at(auth)], json(200, changed));
    expect(nerve.calls.at(-1)?.body).toEqual(body);
    expect(changing.value).toEqual(changed);
    expect(store.unarchived.get(web.id)).toEqual([{ ...authRow, ...changed }, checkoutRow, searchRow]);
    expect(store.details.get(auth.id)).toEqual(changed);
    expect(store.getModuleRow(auth.id)).toEqual({ ...authRow, ...changed });
    expect(store.getModuleProgress(auth.id)).toEqual(authProgress);

    // archived in another tab, and this one fetched its archive; then unarchived there: its archive's row, answered
    // last, shows nerve's next answer, with its own counts
    const between = moduleRowOf({ ...changed, status: "completed", archived_at: LATER }, { total_issues: 5 });
    await loadArchivedModules(nerve, store, [between, shelvedRow]);
    const renamed: Module = { ...changed, name: "Sign-in 2", updated_at: LATEST };
    const rename = () => store.update(auth.id, { name: "Sign-in 2" });
    await sent(nerve, rename, ["PATCH", at(auth)], json(200, renamed));
    expect(store.archived.get(web.id)).toEqual([{ ...between, ...renamed }, shelvedRow]);
    expect(store.getModuleById(auth.id)).toEqual({ ...between, ...renamed });
  });

  it("adds and removes members in each change's turn, from the members nerve answered last: none is lost (R6)", async () => {
    const { nerve, store } = await loadedModules();
    const k = nerve.calls.length;
    // two clicks, quicker than nerve: Cy added, then Bob removed; gone, who left web, stays
    const adding = track(store.changeMembers(auth.id, { add: ["u-cy"], remove: [] }));
    const removing = track(store.changeMembers(auth.id, { add: [], remove: ["u-bob"] }));
    // nerve answers the members ascending
    const added: Module = { ...auth, member_ids: ["u-ann", "u-bob", "u-cy", "u-gone"], updated_at: LATER };
    await inTurn(nerve, k, ["PATCH", at(auth)], json(200, added));
    const removed: Module = { ...auth, member_ids: ["u-ann", "u-cy", "u-gone"], updated_at: LATEST };
    await inTurn(nerve, k + 1, ["PATCH", at(auth)], json(200, removed));
    await until(() => adding.settled && removing.settled, "the two changes");
    expect(nerve.calls.slice(k).map((call) => call.body)).toEqual([
      { member_ids: ["u-ann", "u-bob", "u-gone", "u-cy"] },
      { member_ids: ["u-ann", "u-cy", "u-gone"] },
    ]);
    expect([adding.value, removing.value]).toEqual([added, removed]);
    expect(store.getModuleRow(auth.id)).toEqual({ ...authRow, ...removed });
    expect(store.details.get(auth.id)).toEqual(removed);
  });

  it("archives a closed module, which moves with its counts first into the archive, and unarchives it back", async () => {
    const { nerve, store } = await loadedModules();
    const archived: Module = { ...checkout, archived_at: LATER, updated_at: LATER };
    const shelve = () => store.archive(checkout.id);
    const archiving = await sent(nerve, shelve, ["POST", archive(checkout)], json(200, archived));
    expect(archiving.value).toEqual(archived);
    expect(store.getProjectModuleIds(web.id)).toEqual([auth.id, search.id]);
    expect(store.archived.get(web.id)).toEqual([{ ...checkoutRow, ...archived }, shelvedRow]);
    expect(store.getModuleRow(checkout.id)).toEqual({ ...checkoutRow, ...archived });

    // unarchived: first of the list again, its counts kept; and so one nerve gave in the archive
    const restored: Module = { ...checkout, updated_at: LATEST };
    const restore = () => store.unarchive(checkout.id);
    const unarchiving = await sent(nerve, restore, ["POST", unarchive(checkout)], json(200, restored));
    expect(unarchiving.value).toEqual(restored);
    const shelvedBack: Module = { ...shelved, archived_at: null, updated_at: LATEST };
    await sent(nerve, () => store.unarchive(shelved.id), ["POST", unarchive(shelved)], json(200, shelvedBack));
    expect(store.unarchived.get(web.id)).toEqual([
      { ...shelvedRow, ...shelvedBack },
      { ...checkoutRow, ...restored },
      authRow,
      searchRow,
    ]);
    expect(store.getArchivedModuleIds(web.id)).toEqual([]);
  });

  it("archives again a module archived elsewhere, in its place; one of no row the store holds gets none", async () => {
    const { nerve, store } = await loadedModules();
    // another tab archived Checkout, then Old after it; this one's list still has Checkout, its archive, fetched since,
    // has both
    const elsewhere = moduleRowOf({ ...checkout, archived_at: LATER }, { total_issues: 2 });
    const old = moduleRowOf(moduleOf(web, "Old", { status: "cancelled", archived_at: LATEST }), { total_issues: 1 });
    await loadArchivedModules(nerve, store, [old, elsewhere, shelvedRow]);
    // nerve writes nothing, and answers Checkout as it holds it, renamed there since
    const holds: Module = { ...checkout, name: "Checkout v2", archived_at: LATER, updated_at: LATEST };
    await sent(nerve, () => store.archive(checkout.id), ["POST", archive(checkout)], json(200, holds));
    expect(store.getProjectModuleIds(web.id)).toEqual([auth.id, search.id]);
    expect(store.archived.get(web.id)).toEqual([old, { ...elsewhere, ...holds }, shelvedRow]);

    // ops's Infra, of which the store holds its read alone and ops's empty archive: its read shows nerve's answer
    const infra = moduleOf(ops, "Infra", { status: "completed" });
    await loadArchivedModules(nerve, store, [], ops);
    await loadModule(nerve, store, infra);
    const infraArchived: Module = { ...infra, archived_at: LATER, updated_at: LATER };
    await sent(nerve, () => store.archive(infra.id), ["POST", archive(infra)], json(200, infraArchived));
    expect(store.details.get(infra.id)).toEqual(infraArchived);
    expect(store.getArchivedModuleIds(ops.id)).toEqual([]);
  });

  it("deletes a module from both lists and its read: it, its row, progress and links no longer show, nor do they once the fetches out meanwhile answer", async () => {
    const { nerve, store } = await loadedModules();
    const k = nerve.calls.length;
    const listing = track(store.fetchModules(web.id));
    await until(() => nerve.calls.length === k + 1, "the list");
    const reading = track(store.fetchModule(auth.id));
    await until(() => nerve.calls.length === k + 2, "the read");
    const removing = await sent(nerve, () => store.remove(auth.id), ["DELETE", at(auth)], noContent());
    expect(removing).toEqual({ settled: true, value: undefined });
    const gone = () => {
      expect(store.getProjectModuleIds(web.id)).toEqual([checkout.id, search.id]);
      expect(store.details.get(auth.id)).toBeNull();
      expect(store.getModuleById(auth.id)).toBeUndefined();
      expect(store.getModuleRow(auth.id)).toBeUndefined();
      expect(store.getModuleProgress(auth.id)).toBeUndefined();
      expect(store.getModuleLinks(auth.id)).toBeUndefined();
    };
    gone();
    // the list and the read, which nerve answered before the deletion
    nerve.calls[k]?.answer(json(200, { data: [authRow, checkoutRow, searchRow] }));
    nerve.calls[k + 1]?.answer(json(200, auth));
    await until(() => listing.settled && reading.settled, "the fetches");
    gone();
    expect(reading.value).toBeNull();

    // an archived one, which the archive and its read hold
    await loadModule(nerve, store, shelved);
    await sent(nerve, () => store.remove(shelved.id), ["DELETE", at(shelved)], noContent());
    expect(store.getArchivedModuleIds(web.id)).toEqual([]);
    expect(store.getModuleById(shelved.id)).toBeUndefined();
  });

  it("adds a link first of its module's, and changes or removes one in its module's list as it is in its turn", async () => {
    const { nerve, store } = await loadedModules();
    const k = nerve.calls.length;
    // a link added, and docs given a new address, quicker than nerve: the change finds the list with the new link
    const body = { url: "https://specs.example.com", title: "Specs" };
    const added = moduleLinkOf(auth, body.url, { title: "Specs", created_at: LATER });
    const adding = track(store.addLink(auth.id, body));
    const changing = track(store.changeLink(docs.id, { url: "https://docs.example.org" }));
    await inTurn(nerve, k, ["POST", links(auth)], json(201, added));
    // a new address: nerve answers it with its metadata empty, which it reads later (R10)
    const moved: ModuleLink = { ...docs, url: "https://docs.example.org", metadata: {}, updated_at: LATER };
    await inTurn(nerve, k + 1, ["PATCH", linkAt(docs)], json(200, moved));
    await until(() => adding.settled && changing.settled, "the two changes");
    expect(nerve.calls.slice(k).map((call) => call.body)).toEqual([body, { url: "https://docs.example.org" }]);
    expect([adding.value, changing.value]).toEqual([added, moved]);
    expect(store.getModuleLinks(auth.id)).toEqual([added, moved, notes]);

    const removing = await sent(nerve, () => store.removeLink(notes.id), ["DELETE", linkAt(notes)], noContent());
    expect(removing).toEqual({ settled: true, value: undefined });
    expect(store.getModuleLinks(auth.id)).toEqual([added, moved]);
  });
});

describe("ModuleStore, a change in its turn", () => {
  it("sends each change once nerve has answered the one before it; one of a module or a link finds it in its turn", async () => {
    const { nerve, projects, store } = await loadedModules();
    const k = nerve.calls.length;
    // made while Auth is held: the change and the deletion go out one at a time, and those behind them find it gone
    const renamed = track(store.update(auth.id, { name: "Sign-in" }));
    const removed = track(store.remove(auth.id));
    const behind = [
      track(store.changeMembers(auth.id, { add: ["u-cy"], remove: [] })),
      track(store.remove(auth.id)),
      track(store.changeLink(docs.id, { title: "Docs v2" })),
      track(store.removeLink(notes.id)),
    ];
    await inTurn(nerve, k, ["PATCH", at(auth)], problem(503, "server_busy"));
    await inTurn(nerve, k + 1, ["DELETE", at(auth)], noContent());
    await until(() => behind.every((change) => change.settled), "the changes behind them");
    expect(renamed.error).toBeInstanceOf(ApiError);
    expect(removed).toEqual({ settled: true, value: undefined });
    const [noModule, noLink] = [new Error("Module not found"), new Error("Link not found")];
    expect(behind.map((change) => change.error)).toEqual([noModule, noModule, noLink, noLink]);
    expect(nerve.calls).toHaveLength(k + 2);

    // of a project he left, the store shows none: nerve is asked nothing
    await sent(nerve, () => projects.leaveProject(web), ["POST", `/api/v0/projects/${web.id}/leave`], noContent());
    const left = await settle(store.changeMembers(checkout.id, { add: [], remove: [] }), "a change in web");
    expect(left).toMatchObject({ settled: true, error: noModule });
    expect(nerve.calls).toHaveLength(k + 3);
  });

  it("changes or removes a link in its turn: of one removed before it, nerve is asked nothing", async () => {
    const { nerve, store } = await loadedModules();
    const k = nerve.calls.length;
    const removed = track(store.removeLink(docs.id));
    const behind = [track(store.changeLink(docs.id, { title: "Docs v2" })), track(store.removeLink(docs.id))];
    await inTurn(nerve, k, ["DELETE", linkAt(docs)], noContent());
    await until(() => behind.every((change) => change.settled), "the changes behind it");
    expect(removed).toEqual({ settled: true, value: undefined });
    expect(behind.map((change) => change.error)).toEqual([new Error("Link not found"), new Error("Link not found")]);
    expect(nerve.calls).toHaveLength(k + 1);
    expect(store.getModuleLinks(auth.id)).toEqual([notes]);
  });
});

// A fetch's answer may be older than the changes nerve confirmed while it was out: what the fetch shows carries them
// (the order is forced: each fetch waits until the test answers it, after the changes have finished).
describe("ModuleStore, while a fetch is out", () => {
  it("makes the changes nerve confirmed meanwhile on the answers of the fetches, read before them", async () => {
    const { nerve, store } = await loadedModules();
    const k = nerve.calls.length;
    const listing = track(store.fetchModules(web.id));
    await until(() => nerve.calls.length === k + 1, "the list");
    const archiving = track(store.fetchArchivedModules(web.id));
    await until(() => nerve.calls.length === k + 2, "the archive");
    const reading = track(store.fetchModule(auth.id));
    await until(() => nerve.calls.length === k + 3, "the read");
    const linking = track(store.fetchLinks(auth.id));
    await until(() => nerve.calls.length === k + 4, "the links");
    // meanwhile Onboarding is created, Auth renamed, Checkout archived, a link added to Auth and notes removed
    const made = moduleOf(web, "Onboarding", { sort_order: 35535, created_at: LATER, updated_at: LATER });
    await sent(nerve, () => store.create(web.id, { name: "Onboarding" }), ["POST", list(web)], json(201, made));
    const renamed: Module = { ...auth, name: "Sign-in", updated_at: LATER };
    await sent(nerve, () => store.update(auth.id, { name: "Sign-in" }), ["PATCH", at(auth)], json(200, renamed));
    const archived: Module = { ...checkout, archived_at: LATER, updated_at: LATER };
    await sent(nerve, () => store.archive(checkout.id), ["POST", archive(checkout)], json(200, archived));
    const added = moduleLinkOf(auth, "https://specs.example.com", { created_at: LATER });
    const add = () => store.addLink(auth.id, { url: added.url });
    await sent(nerve, add, ["POST", links(auth)], json(201, added));
    await sent(nerve, () => store.removeLink(notes.id), ["DELETE", linkAt(notes)], noContent());
    // each read before them; Auth had a work item more by then
    const authThen = moduleRowOf(auth, { total_issues: 5, started_issues: 3, completed_issues: 2 });
    nerve.calls[k]?.answer(json(200, { data: [authThen, checkoutRow, searchRow] }));
    nerve.calls[k + 1]?.answer(json(200, { data: [shelvedRow] }));
    nerve.calls[k + 2]?.answer(json(200, auth));
    nerve.calls[k + 3]?.answer(json(200, { data: [docs, notes] }));
    await until(() => [listing, archiving, reading, linking].every((fetch) => fetch.settled), "the fetches");

    expect(store.unarchived.get(web.id)).toEqual([moduleRowOf(made), { ...authThen, ...renamed }, searchRow]);
    expect(store.archived.get(web.id)).toEqual([{ ...checkoutRow, ...archived }, shelvedRow]);
    expect(store.details.get(auth.id)).toEqual(renamed);
    expect(store.getModuleLinks(auth.id)).toEqual([added, docs]);
    expect(listing.value).toEqual(store.unarchived.get(web.id));
  });

  it("fetches the modules while a change is out: a fetch does not wait for it, and a refused change leaves its answer", async () => {
    const { nerve, store } = await loadedModules();
    const relisted = [moduleRowOf(auth, { total_issues: 5 }), checkoutRow, searchRow];
    await fetchedWhileChangeIsOut(
      nerve,
      { send: () => store.archive(checkout.id), request: ["POST", archive(checkout)] },
      { send: () => store.fetchModules(web.id), request: ["GET", list(web)], body: { data: relisted } }
    );
    expect(store.unarchived.get(web.id)).toEqual(relisted);
    expect(store.archived.get(web.id)).toEqual([shelvedRow]);
  });
});

/** A change of the store: what sends it, its request, and a refusal of nerve's. */
type Sending = { change: string; send: (store: ModuleStore) => Promise<unknown>; request: Endpoint; refusal: Response };
const changes: Sending[] = [
  {
    change: "a creation",
    send: (store) => store.create(web.id, { name: "Checkout" }),
    request: ["POST", list(web)],
    refusal: problem(409, "planning.module_name_taken"),
  },
  {
    change: "a change",
    send: (store) => store.update(auth.id, { name: "  " }),
    request: ["PATCH", at(auth)],
    refusal: problem(422, "validation_failed"),
  },
  {
    change: "a change of the members",
    send: (store) => store.changeMembers(auth.id, { add: ["u-guest"], remove: [] }),
    request: ["PATCH", at(auth)],
    refusal: problem(422, "validation_failed"),
  },
  {
    change: "a deletion",
    send: (store) => store.remove(auth.id),
    request: ["DELETE", at(auth)],
    refusal: problem(403, "forbidden"),
  },
  {
    change: "an archiving",
    send: (store) => store.archive(auth.id),
    request: ["POST", archive(auth)],
    refusal: problem(409, "planning.module_not_closed"),
  },
  {
    change: "an unarchiving",
    send: (store) => store.unarchive(shelved.id),
    request: ["POST", unarchive(shelved)],
    refusal: problem(409, "planning.modules_disabled"),
  },
  {
    change: "a link added",
    send: (store) => store.addLink(auth.id, { url: notes.url }),
    request: ["POST", links(auth)],
    refusal: problem(409, "planning.module_link_exists"),
  },
  {
    change: "a link changed",
    send: (store) => store.changeLink(docs.id, { url: notes.url }),
    request: ["PATCH", linkAt(docs)],
    refusal: problem(409, "planning.module_link_exists"),
  },
  {
    change: "a link removed",
    send: (store) => store.removeLink(notes.id),
    request: ["DELETE", linkAt(notes)],
    refusal: problem(409, "planning.archived"),
  },
];

describe("ModuleStore, a change refused or cut", () => {
  it.each(changes)("fails, changing nothing, when nerve refuses $change", async ({ send, request, refusal }) => {
    const { nerve, store } = await loadedModules();
    const before = held(store);
    const k = nerve.calls.length;
    const refused = track(send(store));
    await inTurn(nerve, k, request, refusal);
    await until(() => refused.settled, "the refusal");
    await vi.advanceTimersByTimeAsync(1_000);
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(held(store)).toEqual(before);
    expect(nerve.calls).toHaveLength(k + 1);
  });

  it("changes nothing of this generation once the session has changed: each change fails before it is sent", async () => {
    const { nerve, api, store } = await loadedModules();
    const before = held(store);
    const k = nerve.calls.length;
    FakeNerve.replaceSession(api);
    const cut = changes.map(({ send }) => track(send(store)));
    await until(() => cut.every((change) => change.settled), "the changes");
    expect(cut.map((change) => change.error)).toEqual(changes.map(() => expect.any(SessionChangedError)));
    expect(nerve.calls).toHaveLength(k);
    expect(held(store)).toEqual(before);
  });
});
