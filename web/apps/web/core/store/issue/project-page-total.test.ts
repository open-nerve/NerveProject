/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The count in the headers of a project's work items page and of its archived work items' page (P7c spec 2.8, R10):
// nerve's total of the page's list, grouped or not, and only while the current list is that project's: each store is
// the app's, one for every project, and until the project's list is selected the current one may be another project's.
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json } from "@/lib/auth/fake-nerve";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { issueOf, issueStoreTab, requestedIssue } from "./fake-issues";

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
const binding = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return binding.app;
  },
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  binding.app = tab.app;
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});

/** nerve's list of a project grouped by state: total work items in all, of which it sends the first page's one. */
const grouped = (total: number, project: string) => ({
  total,
  groups: [{ key: "state", total, data: [issueOf(`first of ${project}`, project)], next_cursor: "next" }],
});

it.each([
  ["the project's page", (app: RootStore) => app.issue.projectIssues],
  ["the project's archived work items' page", (app: RootStore) => app.issue.archivedIssues],
] as const)(
  "counts the work items of %s as nerve does, grouped, only while the current list is that project's",
  async (_, of) => {
    const store = of(tab.app);
    const read = (project: string) =>
      requestedIssue(
        tab.nerve,
        () => store.fetchIssues("ws", project, "init-loader", { limit: 20, query: { group_by: "state_id" } }),
        ["GET", `/api/v0/projects/${project}/issues`]
      );
    expect(store.pageTotal("A"), "nothing read").toBeUndefined();
    const first = await read("A");
    first.call.answer(json(200, grouped(7, "A")));
    await first.promise;
    expect(store.pageTotal("A"), "nerve's count of the grouped list, though it sent one row").toBe(7);
    expect(store.pageTotal("B"), "another project's page").toBeUndefined();
    const second = await read("B");
    expect(store.pageTotal("A"), "the first project's, its list no longer current").toBeUndefined();
    expect(store.pageTotal("B"), "the second project's, not answered yet").toBeUndefined();
    second.call.answer(json(200, grouped(2, "B")));
    await second.promise;
    expect(store.pageTotal("B")).toBe(2);
  }
);
