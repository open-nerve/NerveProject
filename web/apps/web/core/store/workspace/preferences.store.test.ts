/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { WorkspacePreferences, WorkspacePreferencesUpdate } from "@nerve/api-client";
import { preferencesChangeOf } from "@/hooks/navigation-preferences";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, answered, json, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn } from "@/store/fake-queue";
import { fakeRoot } from "@/store/fake-root";
import { RouterStore } from "@/store/router.store";
import { WorkspaceRootStore } from "@/store/workspace";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import type { IWorkspacePreferencesStore, PreferencesChange } from "@/store/workspace/preferences.store";

// The caller's navigation settings in his workspaces (M3 design 3.18, 7.3), against a fake nerve that answers each
// request when the test says. That they are kept by the workspace's id is root.store.test.ts.

const PREFERENCES = "/api/v0/me/workspaces/acme/preferences";
/** nerve's defaults, which it gives until the caller changes one. */
const defaults: WorkspacePreferences = { navigation_control_preference: "ACCORDION", navigation_project_limit: 10 };
const tabbed: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 10 };
/** nerve's answer to the change to tabs, once another tab of his has set the limit to 3. */
const elsewhere: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 3 };
/** The caller's settings in another workspace of his, where the sidebar shows every project. */
const all: WorkspacePreferences = { navigation_control_preference: "ACCORDION", navigation_project_limit: 0 };
/** The change to data, whatever the settings it is made to. */
const to =
  (data: WorkspacePreferencesUpdate): PreferencesChange =>
  () =>
    data;

/**
 * The store of a tab and its client: the workspaces root's, whose caller's list nerve gave as acme and beta, the
 * requests of which are then forgotten.
 */
async function preferencesStore() {
  const nerve = new FakeNerve();
  const api = nerve.client();
  const workspaceRoot = new WorkspaceRootStore(fakeRoot({ router: new RouterStore() }), api);
  await loadWorkspaces(nerve, workspaceRoot, [workspaceOf("acme"), workspaceOf("beta")]);
  nerve.calls.length = 0;
  return { nerve, api, store: workspaceRoot.preferences };
}

/** The store fetches the caller's settings in the workspace slug names (acme unless it says), and nerve gives these. */
function load(nerve: FakeNerve, store: IWorkspacePreferencesStore, preferences: WorkspacePreferences, slug = "acme") {
  const fetch = () => store.fetchPreferences(workspaceOf(slug));
  return answered(nerve, fetch, ["GET", `/api/v0/me/workspaces/${slug}/preferences`], preferences, "the settings");
}

/** A store whose settings in acme nerve gave as its defaults. */
async function loaded() {
  const tab = await preferencesStore();
  await load(tab.nerve, tab.store, defaults);
  return tab;
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("WorkspacePreferencesStore", () => {
  it("keeps the caller's settings in a workspace as nerve gives them", async () => {
    const { nerve, store } = await preferencesStore();
    const fetched = store.fetchPreferences(workspaceOf("acme"));
    await until(() => nerve.calls.length === 1, "the settings");
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: PREFERENCES });
    expect(store.getPreferences("acme")).toBeUndefined();
    nerve.calls[0]?.answer(json(200, defaults));
    expect(await settle(fetched, "the settings")).toEqual({ settled: true, value: defaults });
    expect(store.getPreferences("acme")).toEqual(defaults);
    expect(store.getPreferences("beta")).toBeUndefined();
  });

  it("fails when nerve refuses them, keeping none, and again, keeping the settings it had", async () => {
    const { nerve, store } = await preferencesStore();
    const refused = track(store.fetchPreferences(workspaceOf("acme")));
    await until(() => nerve.calls.length === 1, "the settings");
    nerve.calls[0]?.answer(problem(404, "workspace.not_found"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getPreferences("acme")).toBeUndefined();

    await load(nerve, store, defaults);
    const again = track(store.fetchPreferences(workspaceOf("acme")));
    await until(() => nerve.calls.length === 3, "the settings again");
    nerve.calls[2]?.answer(problem(503, "server_busy"));
    await until(() => again.settled, "the refusal");
    expect(again.error).toBeInstanceOf(ApiError);
    expect(store.getPreferences("acme")).toEqual(defaults);
  });

  it("keeps the settings it had, gives nothing and does not fail, when the session changes as it fetches them again", async () => {
    const { nerve, api, store } = await loaded();
    FakeNerve.replaceSession(api);
    const fetched = await settle(store.fetchPreferences(workspaceOf("acme")), "the fetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(1);
    expect(store.getPreferences("acme")).toEqual(defaults);
  });

  it("has nerve's answer to a change, not the change, and only once nerve answers", async () => {
    const { nerve, store } = await loaded();
    const changed = track(store.updatePreferences("acme", to({ navigation_control_preference: "TABBED" })));
    await until(() => nerve.calls.length === 2, "the change");
    expect(nerve.calls[1]).toMatchObject({
      method: "PATCH",
      path: PREFERENCES,
      body: { navigation_control_preference: "TABBED" },
    });
    expect(store.getPreferences("acme")).toEqual(defaults);
    nerve.calls[1]?.answer(json(200, elsewhere));
    await until(() => changed.settled, "the answer");
    expect(changed.value).toEqual(elsewhere);
    expect(store.getPreferences("acme")).toEqual(elsewhere);
  });

  it("fails, changing nothing, when nerve refuses a change", async () => {
    const { nerve, store } = await loaded();
    const refused = track(store.updatePreferences("acme", to({ navigation_project_limit: -1 })));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(problem(422, "validation_failed"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getPreferences("acme")).toEqual(defaults);
  });

  it("keeps each workspace's settings apart: a change or a fetch in one leaves the other's", async () => {
    const { nerve, store } = await loaded();
    await load(nerve, store, all, "beta");
    const changed = track(store.updatePreferences("acme", to({ navigation_control_preference: "TABBED" })));
    await until(() => nerve.calls.length === 3, "the change");
    expect(nerve.calls[2]).toMatchObject({ method: "PATCH", path: PREFERENCES });
    nerve.calls[2]?.answer(json(200, elsewhere));
    await until(() => changed.settled, "the answer");
    expect(store.getPreferences("acme")).toEqual(elsewhere);
    expect(store.getPreferences("beta")).toEqual(all);
    await load(nerve, store, tabbed, "beta");
    expect(store.getPreferences("beta")).toEqual(tabbed);
    expect(store.getPreferences("acme")).toEqual(elsewhere);
  });

  it("sends each change once nerve has answered the one before it, refused or not", async () => {
    const { nerve, store } = await loaded();
    const first = track(store.updatePreferences("acme", to({ navigation_project_limit: 5 })));
    const second = track(store.updatePreferences("acme", to({ navigation_control_preference: "TABBED" })));
    await inTurn(nerve, 1, ["PATCH", PREFERENCES], problem(503, "server_busy"));
    await inTurn(nerve, 2, ["PATCH", PREFERENCES], json(200, elsewhere));
    await until(() => second.settled, "the last change");
    expect(first.error).toBeInstanceOf(ApiError);
    // nerve's answer, not a change sent: its limit is neither the refused 5 nor the loaded 10
    expect(store.getPreferences("acme")).toEqual(elsewhere);
  });

  it("makes each change to the settings nerve answered the one before it, not to those the store had as it was asked for", async () => {
    const { nerve, store } = await loaded();
    const first = track(store.updatePreferences("acme", to({ navigation_control_preference: "TABBED" })));
    // turning the limit on: a limit of the count of the settings it is made to
    const second = track(
      store.updatePreferences("acme", (held) => ({ navigation_project_limit: held.navigation_project_limit }))
    );
    // nerve's answer to the first: another tab of his has set the limit to 3 meanwhile
    await inTurn(nerve, 1, ["PATCH", PREFERENCES], json(200, elsewhere));
    await inTurn(nerve, 2, ["PATCH", PREFERENCES], json(200, elsewhere));
    await until(() => second.settled, "the second change");
    expect(first.error).toBeUndefined();
    expect(nerve.calls[2]?.body).toEqual({ navigation_project_limit: 3 });
  });

  it("makes two quick turns of the limit each to nerve's answer to the one before it: off, then on again", async () => {
    const { nerve, store } = await loaded();
    const turn = preferencesChangeOf({ limitToggled: true });
    track(store.updatePreferences("acme", turn));
    const second = track(store.updatePreferences("acme", turn));
    await inTurn(nerve, 1, ["PATCH", PREFERENCES], json(200, { ...defaults, navigation_project_limit: 0 }));
    await inTurn(nerve, 2, ["PATCH", PREFERENCES], json(200, defaults));
    await until(() => second.settled, "the second turn");
    expect([nerve.calls[1]?.body, nerve.calls[2]?.body]).toEqual([
      { navigation_project_limit: 0 },
      { navigation_project_limit: 10 },
    ]);
  });

  it("fails without sending a change while it has no settings of the workspace", async () => {
    const { nerve, store } = await preferencesStore();
    const changed = track(store.updatePreferences("acme", to({ navigation_control_preference: "TABBED" })));
    await until(() => changed.settled, "the change");
    expect(changed).toMatchObject({ settled: true, error: new Error("Workspace settings not found") });
    expect(nerve.calls).toHaveLength(0);
  });

  it("fetches the settings while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    await fetchedWhileChangeIsOut(
      nerve,
      {
        send: () => store.updatePreferences("acme", to({ navigation_project_limit: 3 })),
        request: ["PATCH", PREFERENCES],
      },
      { send: () => store.fetchPreferences(workspaceOf("acme")), request: ["GET", PREFERENCES], body: elsewhere }
    );
    expect(store.getPreferences("acme")).toEqual(elsewhere);
  });
});

// A fetch's answer may be older than a change nerve confirmed while it was out: the change wins (the order is forced:
// the fetch waits until the test answers it, after the change has finished). Of two fetches of one workspace's
// settings, only the newer writes. The settings are one document, which no change creates: no fetch can list them
// twice.
describe("WorkspacePreferencesStore, while a fetch is out", () => {
  it("shows a change nerve confirmed during a refetch, not the refetch's older settings", async () => {
    const { nerve, store } = await loaded();
    const refetched = track(store.fetchPreferences(workspaceOf("acme")));
    await until(() => nerve.calls.length === 2, "the refetch");
    const changed = track(store.updatePreferences("acme", to({ navigation_control_preference: "TABBED" })));
    await inTurn(nerve, 2, ["PATCH", PREFERENCES], json(200, tabbed));
    await until(() => changed.settled, "the answer");
    // read before the change
    nerve.calls[1]?.answer(json(200, defaults));
    await until(() => refetched.settled, "the refetch");

    expect(store.getPreferences("acme")).toEqual(tabbed);
    expect(refetched.value).toEqual(tabbed);
  });

  it("lets each workspace's newer fetch write: an older one answering last writes nothing", async () => {
    const { nerve, store } = await preferencesStore();
    const older = track(store.fetchPreferences(workspaceOf("acme")));
    await until(() => nerve.calls.length === 1, "the older settings");
    const beta = track(store.fetchPreferences(workspaceOf("beta")));
    await until(() => nerve.calls.length === 2, "beta's settings");
    const newer = track(store.fetchPreferences(workspaceOf("acme")));
    await until(() => nerve.calls.length === 3, "the newer settings");
    nerve.calls[2]?.answer(json(200, elsewhere));
    await until(() => newer.settled, "the newer settings");
    // a newer fetch of acme's settings does not overtake one of beta's
    nerve.calls[1]?.answer(json(200, all));
    await until(() => beta.settled, "beta's settings");
    // read before another tab of his set the limit to 3
    nerve.calls[0]?.answer(json(200, tabbed));
    await until(() => older.settled, "the older settings");

    expect(older).toEqual({ settled: true, value: undefined });
    expect(store.getPreferences("acme")).toEqual(elsewhere);
    expect(store.getPreferences("beta")).toEqual(all);
  });
});
