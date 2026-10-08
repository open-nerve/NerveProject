/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectNavigation } from "@nerve/api-client";
import { hideTab, toggleDefaultTab } from "@/components/navigation/tab-navigation-utils";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn, sent } from "@/store/fake-queue";
import { preferencesOf, projectOf, projectTab } from "@/store/project/fake-projects";
import type { IProjectPreferencesStore } from "@/store/project/preferences.store";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// The caller's tab bar in each project's header (M3 design 3.18, 7.3), against a fake nerve that answers each request
// when the test says. The store is the project root's, as the RootStore builds it on the project store, whose
// projects it gives the tab bars of. The tab's address is web's, a project of acme; ops is acme's other; his other
// workspace, beta, has lab. The changes are the header's (tab-navigation-utils.ts), or a whole tab bar.

const acme = workspaceOf("acme");
const beta = workspaceOf("beta");
const web = projectOf("WEB", acme.id);
const ops = projectOf("OPS", acme.id);
const lab = projectOf("LAB", beta.id);
const PREFERENCES = `/api/v0/me/projects/${web.id}/preferences`;
/** nerve's default, which it gives until the caller changes it. */
const defaults: ProjectNavigation = preferencesOf().navigation;
/** His tab bar in web as the tests load it: not nerve's default, so that a store that falls back on that fails. */
const modulesHidden: ProjectNavigation = { default_tab: "work_items", hide_in_more_menu: ["modules"] };
const modules: ProjectNavigation = { default_tab: "modules", hide_in_more_menu: ["views"] };
/** nerve's answer to the change to modules, once another tab of his has hidden the cycles too. */
const elsewhere: ProjectNavigation = { default_tab: "modules", hide_in_more_menu: ["views", "cycles"] };
/** His tab bar in ops, which opens on its cycles. */
const cycles: ProjectNavigation = { default_tab: "cycles", hide_in_more_menu: [] };

/** nerve's answer: his settings in the project, with the tab bar. */
const settings = (navigation: ProjectNavigation) => preferencesOf({ navigation });
/** A change to a whole tab bar, whatever the tab bar it is made to. */
const to = (navigation: ProjectNavigation) => () => navigation;

/** The tab bars' store of a tab at web's address, and its project store, which nerve listed acme's and beta's in. */
async function setUp() {
  const tab = await projectTab({ workspace: acme, projects: [web, ops] }, { workspace: beta, projects: [lab] });
  return { ...tab, store: tab.projectRoot.preferences, projects: tab.projectRoot.project };
}

/** The store fetches the caller's tab bar in the project (web unless it says), and nerve gives this one. */
function load(nerve: FakeNerve, store: IProjectPreferencesStore, navigation: ProjectNavigation, project = web) {
  const fetch = () => store.fetchNavigation(project.id);
  const path = `/api/v0/me/projects/${project.id}/preferences`;
  return answered(nerve, fetch, ["GET", path], settings(navigation), "the tab bar");
}

/** A store whose tab bar in web nerve gave with its modules under "more". */
async function loaded() {
  const tab = await setUp();
  await load(tab.nerve, tab.store, modulesHidden);
  return tab;
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("ProjectPreferencesStore", () => {
  it("keeps the caller's tab bar in a project as nerve gives it, and none of a project the store no longer gives", async () => {
    const { nerve, projects, store } = await setUp();
    const fetched = store.fetchNavigation(web.id);
    await until(() => nerve.calls.length === 1, "the tab bar");
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: PREFERENCES });
    expect(store.getNavigation(web.id)).toBeUndefined();
    nerve.calls[0]?.answer(json(200, settings(modules)));
    expect(await settle(fetched, "the tab bar")).toEqual({ settled: true, value: modules });
    expect(store.getNavigation(web.id)).toEqual(modules);
    expect(store.getNavigation(ops.id)).toBeUndefined();

    // he leaves web: the project store no longer gives it (as for a project deleted, or a workspace no longer his)
    await sent(nerve, () => projects.leaveProject(web), ["POST", `/api/v0/projects/${web.id}/leave`], noContent());
    expect(store.getNavigation(web.id)).toBeUndefined();
  });

  it("fails when nerve refuses it, keeping none, and again, keeping the tab bar it had", async () => {
    const { nerve, store } = await setUp();
    const refused = track(store.fetchNavigation(web.id));
    await until(() => nerve.calls.length === 1, "the tab bar");
    nerve.calls[0]?.answer(problem(403, "forbidden"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getNavigation(web.id)).toBeUndefined();

    await load(nerve, store, modulesHidden);
    const again = track(store.fetchNavigation(web.id));
    await until(() => nerve.calls.length === 3, "the tab bar again");
    nerve.calls[2]?.answer(problem(503, "server_busy"));
    await until(() => again.settled, "the refusal");
    expect(again.error).toBeInstanceOf(ApiError);
    expect(store.getNavigation(web.id)).toEqual(modulesHidden);
  });

  it("keeps the tab bar it had, gives nothing and does not fail, when the session changes as it fetches it again", async () => {
    const { nerve, api, store } = await loaded();
    FakeNerve.replaceSession(api);
    const fetched = await settle(store.fetchNavigation(web.id), "the fetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(1);
    expect(store.getNavigation(web.id)).toEqual(modulesHidden);
  });

  it("sends the tab bar the change makes of nerve's, and has nerve's answer only once nerve answers", async () => {
    const { nerve, store } = await loaded();
    const changed = track(store.updateNavigation(web.id, toggleDefaultTab("cycles")));
    await until(() => nerve.calls.length === 2, "the change");
    const body = { navigation: { ...modulesHidden, default_tab: "cycles" } };
    expect(nerve.calls[1]).toMatchObject({ method: "PATCH", path: PREFERENCES, body });
    expect(store.getNavigation(web.id)).toEqual(modulesHidden);
    nerve.calls[1]?.answer(json(200, settings(elsewhere)));
    await until(() => changed.settled, "the answer");
    expect(changed.value).toEqual(elsewhere);
    expect(store.getNavigation(web.id)).toEqual(elsewhere);
  });

  it("makes each change to the tab bar nerve answered the one before: two hides in a row keep both", async () => {
    const { nerve, store } = await loaded();
    const first = track(store.updateNavigation(web.id, hideTab("cycles")));
    const second = track(store.updateNavigation(web.id, hideTab("modules")));
    // nerve's answer to the first: another tab of his has shown the modules and hidden the views meanwhile
    const meanwhile: ProjectNavigation = { ...defaults, hide_in_more_menu: ["views", "cycles"] };
    const both: ProjectNavigation = { ...defaults, hide_in_more_menu: ["views", "cycles", "modules"] };
    await inTurn(nerve, 1, ["PATCH", PREFERENCES], json(200, settings(meanwhile)));
    expect(nerve.calls[1]?.body).toEqual({ navigation: { ...defaults, hide_in_more_menu: ["modules", "cycles"] } });
    await inTurn(nerve, 2, ["PATCH", PREFERENCES], json(200, settings(both)));
    await until(() => second.settled, "the second change");
    expect(first.error).toBeUndefined();
    expect(nerve.calls[2]?.body).toEqual({ navigation: both });
    expect(store.getNavigation(web.id)).toEqual(both);
  });

  it("fails, changing nothing, when nerve refuses a change", async () => {
    const { nerve, store } = await loaded();
    const refused = track(store.updateNavigation(web.id, hideTab("work_items")));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(problem(422, "validation_failed"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getNavigation(web.id)).toEqual(modulesHidden);
  });

  it("fails without sending a change while it has no tab bar of the project, and shows the fetch's once answered", async () => {
    const { nerve, store } = await setUp();
    const fetched = track(store.fetchNavigation(web.id));
    await until(() => nerve.calls.length === 1, "the tab bar");
    // nerve replaces the tab bar whole: a change made to one it did not give would replace his with it
    const changed = track(store.updateNavigation(web.id, hideTab("modules")));
    await until(() => changed.settled, "the change");
    expect(changed).toMatchObject({ settled: true, error: new Error("Tab bar not found") });
    expect(nerve.calls).toHaveLength(1);
    nerve.calls[0]?.answer(json(200, settings(modules)));
    await until(() => fetched.settled, "the tab bar");
    expect(store.getNavigation(web.id)).toEqual(modules);
  });

  it("keeps each project's tab bar apart: a change or a fetch in one leaves the other's", async () => {
    const { nerve, store } = await loaded();
    await load(nerve, store, cycles, ops);
    const changed = track(store.updateNavigation(web.id, to(modules)));
    await inTurn(nerve, 2, ["PATCH", PREFERENCES], json(200, settings(modules)));
    await until(() => changed.settled, "the answer");
    expect(store.getNavigation(web.id)).toEqual(modules);
    expect(store.getNavigation(ops.id)).toEqual(cycles);
    await load(nerve, store, defaults, ops);
    expect(store.getNavigation(ops.id)).toEqual(defaults);
    expect(store.getNavigation(web.id)).toEqual(modules);
  });

  it("sends each change once nerve has answered the one before it, refused or not", async () => {
    const { nerve, store } = await loaded();
    const first = track(store.updateNavigation(web.id, hideTab("cycles")));
    const second = track(store.updateNavigation(web.id, hideTab("modules")));
    await inTurn(nerve, 1, ["PATCH", PREFERENCES], problem(503, "server_busy"));
    await inTurn(nerve, 2, ["PATCH", PREFERENCES], json(200, settings(elsewhere)));
    await until(() => second.settled, "the last change");
    expect(first.error).toBeInstanceOf(ApiError);
    // made to the tab bar nerve last answered: the refused change's is not nerve's
    expect(nerve.calls[2]?.body).toEqual({ navigation: { ...defaults, hide_in_more_menu: ["modules"] } });
    expect(store.getNavigation(web.id)).toEqual(elsewhere);
  });

  it("fetches the tab bar while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    await fetchedWhileChangeIsOut(
      nerve,
      { send: () => store.updateNavigation(web.id, to(cycles)), request: ["PATCH", PREFERENCES] },
      { send: () => store.fetchNavigation(web.id), request: ["GET", PREFERENCES], body: settings(elsewhere) }
    );
    expect(store.getNavigation(web.id)).toEqual(elsewhere);
  });
});

// A fetch's answer may be older than a change nerve confirmed while it was out: the change wins (the order is forced:
// the fetch waits until the test answers it, after the change has finished). Of two fetches of one project's tab bar,
// only the newer writes. A tab bar is one document, nothing in it listed: none can be listed twice.
describe("ProjectPreferencesStore, while a fetch is out", () => {
  it("shows a change nerve confirmed during a refetch, not the refetch's older tab bar", async () => {
    const { nerve, store } = await loaded();
    const refetched = track(store.fetchNavigation(web.id));
    await until(() => nerve.calls.length === 2, "the refetch");
    const changed = track(store.updateNavigation(web.id, to(modules)));
    await inTurn(nerve, 2, ["PATCH", PREFERENCES], json(200, settings(modules)));
    await until(() => changed.settled, "the answer");
    // read before the change
    nerve.calls[1]?.answer(json(200, settings(modulesHidden)));
    await until(() => refetched.settled, "the refetch");

    expect(store.getNavigation(web.id)).toEqual(modules);
    expect(refetched.value).toEqual(modules);
  });

  it("lets the newer fetch write: an older one answering last writes nothing", async () => {
    const { nerve, store } = await setUp();
    const older = track(store.fetchNavigation(web.id));
    await until(() => nerve.calls.length === 1, "the older tab bar");
    const newer = track(store.fetchNavigation(web.id));
    await until(() => nerve.calls.length === 2, "the newer tab bar");
    nerve.calls[1]?.answer(json(200, settings(elsewhere)));
    await until(() => newer.settled, "the newer tab bar");
    // read before another tab of his hid the cycles
    nerve.calls[0]?.answer(json(200, settings(modules)));
    await until(() => older.settled, "the older tab bar");

    expect(older).toEqual({ settled: true, value: undefined });
    expect(store.getNavigation(web.id)).toEqual(elsewhere);
  });
});
