/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { runInAction } from "mobx";
import { FakeNerve, answered, json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issuePageOf, listIssue, requestedIssue } from "@/store/issue/fake-issues";
import { labelOf, loadProjects, projectOf } from "@/store/project/fake-projects";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";

// A list ordered by label name names its rows' labels from the label store before it shows them (prepareListCatalog,
// M4): with the workspace's label catalog held (M7/P6b spec 2.2, R17), a project the catalog lists has its labels, and
// the list reads none of its own.

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
const binding = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return binding.app;
  },
}));
const { RootStore } = await import("@/store/root.store");
let app: InstanceType<typeof RootStore>;
let nerve: FakeNerve;
const ws = workspaceOf("ws");
const project = projectOf("A", ws.id, { id: "A" });
beforeEach(async () => {
  vi.useFakeTimers();
  nerve = new FakeNerve();
  app = new RootStore(nerve.client());
  binding.app = app;
  runInAction(() => {
    app.router.setQuery({ workspaceSlug: ws.slug, projectId: project.id });
  });
  await loadWorkspaces(nerve, app.workspaceRoot, [ws]);
  await loadProjects(nerve, app.projectRoot.project, ws, [project]);
  runInAction(() => {
    app.issue.projectIssuesFilter.filters.A = {
      richFilters: {},
      displayProperties: undefined,
      kanbanFilters: undefined,
      displayFilters: { layout: "list", group_by: "state", order_by: "sort_order" },
    };
  });
  nerve.calls.length = 0;
});
afterEach(() => {
  app.dispose();
  vi.useRealTimers();
});

it("orders a project's list by label name from the workspace's catalog, reading none of the project's labels", async () => {
  const zebra = labelOf(project, "Zebra", 1);
  const alpha = labelOf(project, "Alpha", 2);
  await answered(
    nerve,
    () => app.label.fetchWorkspaceLabels(ws),
    ["GET", "/api/v0/workspaces/ws/labels"],
    { data: [zebra, alpha] },
    "the catalog"
  );
  const request = await requestedIssue(
    nerve,
    () =>
      app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", {
        limit: 10,
        query: { group_by: undefined, order_by: "label_name" },
      }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  request.call.answer(json(200, issuePageOf([listIssue("x", 1, { label_ids: [zebra.id, alpha.id] })])));
  await until(() => request.result.settled || nerve.calls.length > 2, "the list");
  expect(nerve.calls.map((call) => call.path)).toEqual(["/api/v0/workspaces/ws/labels", "/api/v0/projects/A/issues"]);
  expect([request.result.settled, request.result.error]).toEqual([true, undefined]);
  expect(app.issue.projectIssues.groups?.get("")?.rows[0]?.key).toBe("alpha");
});
