/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Project, View } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, answered, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { listTab, ops, web } from "@/store/fake-project-lists";
import { sent } from "@/store/fake-queue";
import { preferencesOf } from "@/store/project/fake-projects";
import { ViewStore } from "@/store/views/view.store";

// The views of the caller's projects (M7 design 3.7, P6a spec 2.1), against a fake nerve that answers each request
// when the test says: each project's views, by its id, and each view's own read, by its id, as nerve gives them;
// fetches do not queue. The tab's address is web's, a project of acme; ops is acme's other. The store's readers take
// the project or the view from their caller (the route's), not the address. The changes are view-changes.test.ts.

const { display_filters, display_properties } = preferencesOf();

/** A view of the project as nerve gives it: the name names it in its project; ann's, of no filter, unless fields say. */
const viewOf = (project: Project, name: string, fields: Partial<View> = {}): View => ({
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
  ...fields,
});

const bugs = viewOf(web, "Bugs", { filters: { priority: ["urgent", "high"] }, created_at: "2026-11-03T09:00:00Z" });
const mine = viewOf(web, "Mine", { filters: { assignee_id: ["u-ann"] }, created_at: "2026-11-02T09:00:00Z" });
const triage = viewOf(web, "Triage", { owned_by_id: "u-bob" });
const infra = viewOf(ops, "Infra");
/** web's views as nerve lists them: the newest first, whoever owns them. */
const webViews = [bugs, mine, triage];

/** The requests of the views: a project's, where it creates its views; a view's, read, changed and deleted there. */
const list = (project: Project) => `/api/v0/projects/${project.id}/views`;
const at = (view: View) => `/api/v0/views/${view.id}`;

/** The store fetches the project's views, web's unless it says, and nerve lists these. */
const loadViews = (nerve: FakeNerve, store: ViewStore, views: View[], project: Project = web) =>
  answered(nerve, () => store.fetchProjectViews(project.id), ["GET", list(project)], { data: views }, "the views");

/** The store reads the view alone, and nerve gives it as read. */
const loadView = (nerve: FakeNerve, store: ViewStore, read: View) =>
  answered(nerve, () => store.fetchView(read.id), ["GET", at(read)], read, "the view");

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("ViewStore, the reads", () => {
  it("keeps each project's views and each view's read, by its project or view, as nerve gives them", async () => {
    const { nerve, store } = await listTab(ViewStore);
    const listed = await loadViews(nerve, store, webViews);
    expect(nerve.calls[0]?.query).toEqual({});
    expect(listed.value).toEqual(webViews);
    expect(store.lists.get(web.id)).toEqual(webViews);
    expect(store.getProjectViews(web.id)).toEqual(webViews);
    expect(store.getViewById(mine.id)).toEqual(mine);
    expect(store.getProjectViews(ops.id)).toBeUndefined();

    // a view of a project whose views the store has not listed: its read alone has it
    const read = await loadView(nerve, store, infra);
    expect(read.value).toEqual(infra);
    expect(store.details.get(infra.id)).toEqual(infra);
    expect(store.getViewById(infra.id)).toEqual(infra);
    expect(store.getProjectViews(ops.id)).toBeUndefined();

    await loadViews(nerve, store, [infra], ops);
    expect(store.getProjectViews(ops.id)).toEqual([infra]);
    expect(store.getProjectViews(web.id)).toEqual(webViews);
  });

  it("shows a view as nerve last answered it, its read or its list's row, in its project's views too", async () => {
    const { nerve, store } = await listTab(ViewStore);
    await loadViews(nerve, store, [bugs, mine]);
    // another tab renames it, and its read, answered after the list, shows the name; the list keeps its answer
    const renamed: View = { ...bugs, name: "Urgent", updated_at: "2026-11-10T09:00:00Z" };
    await loadView(nerve, store, renamed);
    expect(store.getViewById(bugs.id)).toEqual(renamed);
    expect(store.getProjectViews(web.id)).toEqual([renamed, mine]);
    expect(store.lists.get(web.id)).toEqual([bugs, mine]);
    // the list, answered after the read, shows the view as it has it
    const relisted: View = { ...bugs, name: "Urgent bugs", updated_at: "2026-11-10T10:00:00Z" };
    await loadViews(nerve, store, [relisted, mine]);
    expect(store.getViewById(bugs.id)).toEqual(relisted);
    expect(store.getProjectViews(web.id)).toEqual([relisted, mine]);
    expect(store.details.get(bugs.id)).toEqual(renamed);
  });

  it("answers nothing of a project the caller no longer has, or never had: its views and their reads", async () => {
    const { nerve, projects, store } = await listTab(ViewStore);
    await loadViews(nerve, store, webViews);
    await loadViews(nerve, store, [infra], ops);
    await loadView(nerve, store, infra);
    expect(store.getProjectViews(ops.id)).toEqual([infra]);
    expect(store.getViewById(infra.id)).toEqual(infra);

    await sent(nerve, () => projects.leaveProject(ops), ["POST", `/api/v0/projects/${ops.id}/leave`], noContent());
    expect(store.getProjectViews(ops.id)).toBeUndefined();
    expect(store.getViewById(infra.id)).toBeUndefined();
    // web's still show; a project he never had, or none, has nothing
    expect(store.getProjectViews(web.id)).toEqual(webViews);
    expect(store.getViewById(bugs.id)).toEqual(bugs);
    expect(store.getProjectViews("p-none")).toBeUndefined();
    expect(store.getProjectViews(undefined)).toBeUndefined();
    expect(store.getViewById(undefined)).toBeUndefined();
  });

  it("fails when nerve refuses a fetch, keeping what it had, and gives nothing once the session has changed", async () => {
    const { nerve, api, store } = await listTab(ViewStore);
    await loadViews(nerve, store, webViews);
    await loadView(nerve, store, bugs);
    const refused = track(store.fetchView(bugs.id));
    await until(() => nerve.calls.length === 3, "the read again");
    nerve.calls[2]?.answer(problem(404, "navigation.view_not_found"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getViewById(bugs.id)).toEqual(bugs);

    FakeNerve.replaceSession(api);
    const fetched = [
      await settle(store.fetchProjectViews(web.id), "the list again"),
      await settle(store.fetchView(bugs.id), "the read once more"),
    ];
    expect(fetched).toEqual([
      { settled: true, value: undefined },
      { settled: true, value: undefined },
    ]);
    expect(nerve.calls).toHaveLength(3);
    expect(store.getProjectViews(web.id)).toEqual(webViews);
    expect(store.getViewById(bugs.id)).toEqual(bugs);
  });
});
