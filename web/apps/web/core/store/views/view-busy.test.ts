/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { reaction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Project, View } from "@nerve/api-client";
import { answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { listTab, web } from "@/store/fake-project-lists";
import { preferencesOf } from "@/store/project/fake-projects";
import { ViewStore } from "@/store/views/view.store";

// Whether a view has a change of its own queued or out (P6a spec 2.1, 2.7, R29): from the change's sending, through
// the changes of the view before it, to nerve's answer or refusal; observable, so that the view's page disables what
// changes its draft meanwhile. Another view's change is not this one's. A change of the view's name (its dialog) and
// its deletion are its changes too (re-96 O2). The store and HTTP are real.

const { display_filters, display_properties } = preferencesOf();
const viewOf = (project: Project, name: string): View => ({
  id: `v-${project.identifier}-${name}`,
  workspace_id: project.workspace_id,
  project_id: project.id,
  name,
  description: "",
  logo_props: {},
  filters: {},
  display_filters,
  display_properties,
  owned_by_id: "u-ann",
  created_at: "2026-11-01T09:00:00Z",
  updated_at: "2026-11-01T09:00:00Z",
});
const bugs = viewOf(web, "Bugs");
const mine = viewOf(web, "Mine");
const at = (view: View) => `/api/v0/views/${view.id}`;

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

/** A tab whose store holds web's views; each turn of Bugs being busy or not, as an observer sees it. */
async function loadedViews() {
  const tab = await listTab(ViewStore);
  const { nerve, store } = tab;
  await answered(
    nerve,
    () => store.fetchProjectViews(web.id),
    ["GET", `/api/v0/projects/${web.id}/views`],
    { data: [bugs, mine] },
    "the views"
  );
  const seen: boolean[] = [];
  const stop = reaction(
    () => tab.store.isUpdating(bugs.id),
    (busy) => void seen.push(busy),
    { fireImmediately: true }
  );
  return { ...tab, seen, stop };
}
/** The changes sent to nerve of view, in order. */
const sentTo = (nerve: Awaited<ReturnType<typeof loadedViews>>["nerve"], view: View) =>
  nerve.calls.filter((call) => call.path === at(view) && call.method !== "GET");

it("is busy with a view's changes from the first sent to the last answered, never with another view's", async () => {
  const { nerve, store, seen, stop } = await loadedViews();
  const first = track(store.update(bugs.id, () => ({ name: "One" })));
  const second = track(store.update(bugs.id, () => ({ name: "Two" })));
  await until(() => sentTo(nerve, bugs).length === 1, "the first change");
  expect(store.isUpdating(bugs.id)).toBe(true);
  expect(store.isUpdating(mine.id), "another view's").toBe(false);
  sentTo(nerve, bugs)[0]!.answer(json(200, { ...bugs, name: "One" }));
  await until(() => first.settled && sentTo(nerve, bugs).length === 2, "the second change, in its turn");
  expect(store.isUpdating(bugs.id), "the second out").toBe(true);
  sentTo(nerve, bugs)[1]!.answer(json(200, { ...bugs, name: "Two" }));
  await until(() => second.settled, "the second answered");
  expect(store.isUpdating(bugs.id)).toBe(false);
  expect(seen).toEqual([false, true, false]);
  stop();
});

it("is busy no more once nerve refuses the view's change", async () => {
  const { nerve, store, seen, stop } = await loadedViews();
  const change = track(store.update(bugs.id, () => ({ name: "Urgent" })));
  await until(() => sentTo(nerve, bugs).length === 1, "the change");
  expect(store.isUpdating(bugs.id)).toBe(true);
  sentTo(nerve, bugs)[0]!.answer(problem(409, "navigation.views_disabled"));
  await until(() => change.settled, "the refusal");
  expect(change.error).toBeDefined();
  expect(store.isUpdating(bugs.id)).toBe(false);
  expect(seen).toEqual([false, true, false]);
  stop();
});

it("is busy while the view's deletion is out", async () => {
  const { nerve, store, stop } = await loadedViews();
  const removal = track(store.remove(bugs.id));
  await until(() => sentTo(nerve, bugs).length === 1, "the deletion");
  expect(store.isUpdating(bugs.id)).toBe(true);
  sentTo(nerve, bugs)[0]!.answer(noContent());
  await until(() => removal.settled, "the deletion answered");
  expect(store.isUpdating(bugs.id)).toBe(false);
  stop();
});
