/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Module } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { listTab, ops, web } from "@/store/fake-project-lists";
import { sent } from "@/store/fake-queue";
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

// The modules of the caller's projects (M6 design 3.2, 3.5–3.8, P6a spec 2.2), against a fake nerve that answers each
// request when the test says. The tab's address is web's, a project of acme; ops is acme's other; his other
// workspace, beta, has lab. The store's readers take the project from their caller (the route's), not the address.

const auth = moduleOf(web, "Auth", {
  status: "in-progress",
  start_date: "2026-11-02",
  target_date: "2026-11-27",
  lead_id: "u-ann",
  member_ids: ["u-ann", "u-bob"],
  sort_order: 45535,
});
const checkout = moduleOf(web, "Checkout", { status: "planned", sort_order: 55535 });
const search = moduleOf(web, "Search");
const shelved = moduleOf(web, "Shelved", { status: "completed", archived_at: "2026-10-20T09:00:00Z" });
const infra = moduleOf(ops, "Infra", { status: "in-progress" });
const opsShelved = moduleOf(ops, "Old", { status: "cancelled", archived_at: "2026-10-20T09:00:00Z" });
/** A time after the fixtures', so that a later answer of nerve can be told from an earlier one. */
const LATER = "2026-11-10T09:00:00Z";
/** web's modules that are not archived, in nerve's order: by sort order, the lowest first (not by name or status). */
const webRows = [
  moduleRowOf(auth, { total_issues: 4, started_issues: 2, completed_issues: 1, cancelled_issues: 1 }),
  moduleRowOf(checkout, { total_issues: 2, backlog_issues: 2 }),
  moduleRowOf(search),
];
const docs = moduleLinkOf(auth, "https://docs.example.com", { title: "Docs" });
const notes = moduleLinkOf(auth, "https://notes.example.com");
const ids = (modules: Pick<Module, "id">[]) => modules.map((item) => item.id);

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("ModuleStore, the reads", () => {
  it("keeps a project's modules, its archived ones, a module's read, its progress and its links, each by its project or module", async () => {
    const { nerve, store } = await listTab(ModuleStore);
    const listed = await loadModules(nerve, store, webRows);
    expect(nerve.calls[0]?.query).toEqual({});
    expect(listed.value).toEqual(webRows);
    expect(store.getProjectModuleIds(web.id)).toEqual(ids(webRows));
    expect(store.getModuleRow(auth.id)).toEqual(webRows[0]);
    expect(store.getModuleById(checkout.id)).toEqual(webRows[1]);
    expect(store.getArchivedModuleIds(web.id)).toBeUndefined();
    expect(store.getProjectModuleIds(ops.id)).toBeUndefined();

    const archivedRows = [moduleRowOf(shelved, { total_issues: 3, completed_issues: 3 })];
    await loadArchivedModules(nerve, store, archivedRows);
    expect(nerve.calls[1]?.query).toEqual({ archived: "true" });
    expect(store.getArchivedModuleIds(web.id)).toEqual([shelved.id]);
    expect(store.getModuleRow(shelved.id)).toEqual(archivedRows[0]);
    expect(store.getProjectModuleIds(web.id)).toEqual(ids(webRows));

    const read = await loadModule(nerve, store, shelved);
    expect(read.value).toEqual(shelved);
    expect(store.getModuleById(shelved.id)).toEqual(shelved);
    const progress = moduleProgressOf({ total_issues: 4, started_issues: 2, completed_issues: 1, cancelled_issues: 1 });
    await loadModuleProgress(nerve, store, auth, progress);
    expect(store.getModuleProgress(auth.id)).toEqual(progress);
    expect(store.getModuleProgress(checkout.id)).toBeUndefined();
    await loadModuleLinks(nerve, store, auth, [docs, notes]);
    expect(store.getModuleLinks(auth.id)).toEqual([docs, notes]);
    expect(store.getModuleLinks(checkout.id)).toBeUndefined();

    const opsRows = [moduleRowOf(infra)];
    await loadModules(nerve, store, opsRows, ops);
    expect(store.getProjectModuleIds(ops.id)).toEqual([infra.id]);
    expect(store.getProjectModuleIds(web.id)).toEqual(ids(webRows));
  });

  it("shows a module's fields as nerve last answered them, its read's or its list's, and the counts of its list's row", async () => {
    const { nerve, store } = await listTab(ModuleStore);
    const row = moduleRowOf(auth, { total_issues: 5, started_issues: 3, completed_issues: 2 });
    await loadModules(nerve, store, [row]);
    // another tab renames it and adds a member, and its read, answered after the list, shows them; the counts are
    // the row's
    const changed: Module = { ...auth, name: "Sign-in", member_ids: ["u-ann", "u-bob", "u-cy"], updated_at: LATER };
    await loadModule(nerve, store, changed);
    expect(store.getModuleById(auth.id)).toEqual(changed);
    expect(store.getModuleRow(auth.id)).toEqual({ ...row, ...changed });
    // the list, answered after the read, shows the module as it has it, counts and all
    const relisted = moduleRowOf({ ...auth, name: "Sign-in 2" }, { total_issues: 6, started_issues: 4 });
    await loadModules(nerve, store, [relisted]);
    expect(store.getModuleById(auth.id)).toEqual(relisted);
    expect(store.getModuleRow(auth.id)).toEqual(relisted);

    // a module its read alone has, of no list the store fetched: it shows, without a row
    await loadModule(nerve, store, shelved);
    expect(store.getModuleById(shelved.id)).toEqual(shelved);
    expect(store.getModuleRow(shelved.id)).toBeUndefined();
    // archived elsewhere: the archived list, answered last, has its row, with its counts
    const archivedRow = moduleRowOf({ ...auth, status: "completed", archived_at: LATER }, { total_issues: 7 });
    await loadArchivedModules(nerve, store, [archivedRow]);
    expect(store.getModuleById(auth.id)).toEqual(archivedRow);
    expect(store.getModuleRow(auth.id)).toEqual(archivedRow);
  });

  it("takes the copy nerve answered last, whichever of the list and the read was asked for first", async () => {
    const { nerve, store } = await listTab(ModuleStore);
    const listedAs = (name: string) => json(200, { data: [moduleRowOf({ ...auth, name }, { total_issues: 2 })] });
    const readAs = (name: string) => json(200, { ...auth, name });
    /** The request of path out since the at'th, which the test answers. */
    const out = (path: string, at: number) => nerve.calls.slice(at).find((call) => call.path === path);
    // the list is asked for first, the read second; nerve answers the read first, then the list
    const list = track(store.fetchModules(web.id));
    const read = track(store.fetchModule(auth.id));
    await until(() => nerve.calls.length === 2, "the list and the read");
    out(moduleRequests.at(auth), 0)?.answer(readAs("Read first"));
    await until(() => read.settled, "the read");
    out(moduleRequests.list(web), 0)?.answer(listedAs("Listed last"));
    await until(() => list.settled, "the list");
    expect(store.getModuleById(auth.id)?.name).toBe("Listed last");
    expect(store.getModuleRow(auth.id)).toMatchObject({ name: "Listed last", total_issues: 2 });

    // the read is asked for first, the list second; nerve answers the list first, then the read
    const reread = track(store.fetchModule(auth.id));
    const relist = track(store.fetchModules(web.id));
    await until(() => nerve.calls.length === 4, "the read and the list again");
    out(moduleRequests.list(web), 2)?.answer(listedAs("Listed first"));
    await until(() => relist.settled, "the list again");
    out(moduleRequests.at(auth), 2)?.answer(readAs("Read last"));
    await until(() => reread.settled, "the read again");
    expect(store.getModuleById(auth.id)?.name).toBe("Read last");
    expect(store.getModuleRow(auth.id)).toMatchObject({ name: "Read last", total_issues: 2 });
  });

  it("shows a module's progress and links only while it shows the module", async () => {
    const { nerve, store } = await listTab(ModuleStore);
    // the detail page asks for all three at once: the progress and the links may answer before the module's read
    await loadModuleProgress(nerve, store, checkout, moduleProgressOf({ total_issues: 1 }));
    await loadModuleLinks(nerve, store, checkout, [moduleLinkOf(checkout, "https://shop.example.com")]);
    expect(store.getModuleProgress(checkout.id)).toBeUndefined();
    expect(store.getModuleLinks(checkout.id)).toBeUndefined();
    await loadModule(nerve, store, checkout);
    expect(store.getModuleProgress(checkout.id)).toEqual(moduleProgressOf({ total_issues: 1 }));
    expect(store.getModuleLinks(checkout.id)).toEqual([moduleLinkOf(checkout, "https://shop.example.com")]);
  });

  it("answers nothing of a project the caller no longer has, or never had: its modules, rows, reads, progress and links", async () => {
    const { nerve, projects, store } = await listTab(ModuleStore);
    await loadModules(nerve, store, webRows);
    await loadModules(nerve, store, [moduleRowOf(infra)], ops);
    await loadArchivedModules(nerve, store, [moduleRowOf(opsShelved)], ops);
    await loadModule(nerve, store, opsShelved);
    await loadModuleProgress(nerve, store, infra, moduleProgressOf());
    const opsLinks = [moduleLinkOf(infra, "https://ops.example.com")];
    await loadModuleLinks(nerve, store, infra, opsLinks);
    expect(store.getProjectModuleIds(ops.id)).toEqual([infra.id]);
    expect(store.getArchivedModuleIds(ops.id)).toEqual([opsShelved.id]);
    expect(store.getModuleById(opsShelved.id)).toEqual(opsShelved);
    expect(store.getModuleRow(infra.id)).toEqual(moduleRowOf(infra));
    expect(store.getModuleProgress(infra.id)).toEqual(moduleProgressOf());
    expect(store.getModuleLinks(infra.id)).toEqual(opsLinks);

    await sent(nerve, () => projects.leaveProject(ops), ["POST", `/api/v0/projects/${ops.id}/leave`], noContent());
    expect(store.getProjectModuleIds(ops.id)).toBeUndefined();
    expect(store.getArchivedModuleIds(ops.id)).toBeUndefined();
    expect(store.getModuleById(infra.id)).toBeUndefined();
    expect(store.getModuleById(opsShelved.id)).toBeUndefined();
    expect(store.getModuleRow(infra.id)).toBeUndefined();
    expect(store.getModuleProgress(infra.id)).toBeUndefined();
    expect(store.getModuleLinks(infra.id)).toBeUndefined();
    // web's still show; a project he never had, or none, has nothing
    expect(store.getProjectModuleIds(web.id)).toEqual(ids(webRows));
    expect(store.getProjectModuleIds("p-none")).toBeUndefined();
    expect(store.getArchivedModuleIds(undefined)).toBeUndefined();
    expect(store.getModuleById(undefined)).toBeUndefined();
  });

  /** A value of the store: how it is fetched, by which request, two answers of nerve, and how the store shows it. */
  type Fetched = {
    value: string;
    given?: (nerve: FakeNerve, store: ModuleStore) => Promise<unknown>;
    fetch: (store: ModuleStore) => Promise<unknown>;
    path: string;
    older: unknown;
    newer: unknown;
    shown: (store: ModuleStore) => unknown;
    shows: unknown;
  };
  const newerProgress = moduleProgressOf({ total_issues: 2, completed_issues: 1 });
  const fetches: Fetched[] = [
    {
      value: "a project's modules",
      fetch: (store) => store.fetchModules(web.id),
      path: moduleRequests.list(web),
      older: { data: [moduleRowOf(search)] },
      newer: { data: [moduleRowOf(auth), moduleRowOf(search)] },
      shown: (store) => store.getProjectModuleIds(web.id),
      shows: [auth.id, search.id],
    },
    {
      value: "a project's archived modules",
      fetch: (store) => store.fetchArchivedModules(web.id),
      path: moduleRequests.list(web),
      older: { data: [] },
      newer: { data: [moduleRowOf(shelved)] },
      shown: (store) => store.getArchivedModuleIds(web.id),
      shows: [shelved.id],
    },
    {
      value: "a module",
      fetch: (store) => store.fetchModule(auth.id),
      path: moduleRequests.at(auth),
      older: auth,
      newer: { ...auth, name: "Sign-in" },
      shown: (store) => store.getModuleById(auth.id)?.name,
      shows: "Sign-in",
    },
    {
      value: "a module's progress",
      given: (nerve, store) => loadModule(nerve, store, auth),
      fetch: (store) => store.fetchProgress(auth.id),
      path: moduleRequests.progress(auth),
      older: moduleProgressOf(),
      newer: newerProgress,
      shown: (store) => store.getModuleProgress(auth.id),
      shows: newerProgress,
    },
    {
      value: "a module's links",
      given: (nerve, store) => loadModule(nerve, store, auth),
      fetch: (store) => store.fetchLinks(auth.id),
      path: moduleRequests.links(auth),
      older: { data: [docs] },
      newer: { data: [notes, docs] },
      shown: (store) => store.getModuleLinks(auth.id),
      shows: [notes, docs],
    },
  ];
  it.each(fetches)(
    "writes nothing of an older fetch of $value that nerve answers after a newer one",
    async ({ given, fetch, path, older, newer, shown, shows }) => {
      const { nerve, store } = await listTab(ModuleStore);
      await given?.(nerve, store);
      const at = nerve.calls.length;
      const first = track(fetch(store));
      const second = track(fetch(store));
      await until(() => nerve.calls.length === at + 2, "both fetches");
      expect(nerve.calls.slice(at).map((call) => call.path)).toEqual([path, path]);
      nerve.calls[at + 1]?.answer(json(200, newer));
      await until(() => second.settled, "the newer");
      nerve.calls[at]?.answer(json(200, older));
      await until(() => first.settled, "the older");
      expect(first).toEqual({ settled: true, value: undefined });
      expect(shown(store)).toEqual(shows);
    }
  );

  it("fails when nerve refuses a fetch, keeping what it had, and gives nothing when the session changes", async () => {
    const { nerve, api, store } = await listTab(ModuleStore);
    const refused = track(store.fetchModules(web.id));
    await until(() => nerve.calls.length === 1, "the modules");
    nerve.calls[0]?.answer(problem(403, "forbidden"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getProjectModuleIds(web.id)).toBeUndefined();

    await loadModules(nerve, store, webRows);
    FakeNerve.replaceSession(api);
    const fetched = await settle(store.fetchModules(web.id), "the refetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(2);
    expect(store.getProjectModuleIds(web.id)).toEqual(ids(webRows));
  });
});
