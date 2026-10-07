/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ApiClient, WorkspacePreferences } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, json, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn } from "@/store/fake-queue";
import { WorkspacePreferencesStore } from "@/store/workspace/preferences.store";

// The caller's navigation settings in his workspaces (M3 design 3.18, 7.3), against a fake nerve that answers each
// request when the test says.

const PREFERENCES = "/api/v0/me/workspaces/acme/preferences";
/** nerve's defaults, which it gives until the caller changes one. */
const defaults: WorkspacePreferences = { navigation_control_preference: "ACCORDION", navigation_project_limit: 10 };
const tabbed: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 10 };
/** nerve's answer to the change to tabs, once another tab of his has set the limit to 3. */
const elsewhere: WorkspacePreferences = { navigation_control_preference: "TABBED", navigation_project_limit: 3 };

/** The store of a tab; client builds its client. */
function preferencesStore(client: (nerve: FakeNerve) => ApiClient = (nerve) => nerve.client()) {
  const nerve = new FakeNerve();
  return { nerve, store: new WorkspacePreferencesStore(client(nerve)) };
}

/** A store whose settings in acme nerve gave as its defaults. */
async function loaded() {
  const tab = preferencesStore();
  const fetched = tab.store.fetchPreferences("acme");
  await until(() => tab.nerve.calls.length === 1, "the settings");
  tab.nerve.calls[0]?.answer(json(200, defaults));
  await settle(fetched, "the settings");
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
    const { nerve, store } = preferencesStore();
    const fetched = store.fetchPreferences("acme");
    await until(() => nerve.calls.length === 1, "the settings");
    expect(nerve.calls[0]).toMatchObject({ method: "GET", path: PREFERENCES });
    expect(store.getPreferences("acme")).toBeUndefined();
    nerve.calls[0]?.answer(json(200, defaults));
    expect(await settle(fetched, "the settings")).toEqual({ settled: true, value: defaults });
    expect(store.getPreferences("acme")).toEqual(defaults);
    expect(store.getPreferences("beta")).toBeUndefined();
  });

  it("fails when nerve refuses them, keeping none", async () => {
    const { nerve, store } = preferencesStore();
    const refused = track(store.fetchPreferences("acme"));
    await until(() => nerve.calls.length === 1, "the settings");
    nerve.calls[0]?.answer(problem(404, "workspace.not_found"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.preferencesMap).toEqual({});
  });

  it("gives nothing, and does not fail, when the session changes as it fetches", async () => {
    const { nerve, store } = preferencesStore((fake) => fake.replacedSessionClient());
    const fetched = await settle(store.fetchPreferences("acme"), "the fetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toEqual([]);
    expect(store.preferencesMap).toEqual({});
  });

  it("has nerve's answer to a change, not the change, and only once nerve answers", async () => {
    const { nerve, store } = await loaded();
    const changed = track(store.updatePreferences("acme", { navigation_control_preference: "TABBED" }));
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
    const refused = track(store.updatePreferences("acme", { navigation_project_limit: -1 }));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(problem(422, "validation_failed"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getPreferences("acme")).toEqual(defaults);
  });

  it("sends each change once nerve has answered the one before it, refused or not", async () => {
    const { nerve, store } = await loaded();
    const first = track(store.updatePreferences("acme", { navigation_project_limit: 3 }));
    const second = track(store.updatePreferences("acme", { navigation_control_preference: "TABBED" }));
    await inTurn(nerve, 1, ["PATCH", PREFERENCES], problem(503, "server_busy"));
    await inTurn(nerve, 2, ["PATCH", PREFERENCES], json(200, tabbed));
    await until(() => second.settled, "the last change");
    expect(first.error).toBeInstanceOf(ApiError);
    expect(store.getPreferences("acme")).toEqual(tabbed);
  });

  it("fetches the settings while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    await fetchedWhileChangeIsOut(
      nerve,
      () => store.updatePreferences("acme", { navigation_project_limit: 3 }),
      () => store.fetchPreferences("acme"),
      ["GET", PREFERENCES],
      json(200, elsewhere)
    );
    expect(store.getPreferences("acme")).toEqual(elsewhere);
  });
});
