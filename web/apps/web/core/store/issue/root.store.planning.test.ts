/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue } from "@nerve/api-client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { answered, json } from "@/lib/auth/fake-nerve";
import type { ProjectIssueQuery } from "@/services/issue/issue-list.service";
import { issueDetailOf, issuePageOf, issueStoreTab, listIssue, requestedIssue } from "./fake-issues";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
const binding = vi.hoisted(() => ({ app: undefined as unknown }));
vi.mock("@/lib/store-context", () => ({
  get rootStore() {
    return binding.app;
  },
}));
const C1 = "019a0000-0000-7000-8000-0000000000c1";
const C2 = "019a0000-0000-7000-8000-0000000000c2";
const M1 = "019a0000-0000-7000-8000-0000000000d1";
const M2 = "019a0000-0000-7000-8000-0000000000d2";
const LATER = "2026-10-29T00:00:00Z";
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  binding.app = tab.app;
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});
/** Select and read one project-scoped binding; the root keeps it registered after another becomes current. */
async function listed(query: ProjectIssueQuery, rows: Issue[]) {
  const { app, nerve } = tab;
  await answered(
    nerve,
    () =>
      app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", {
        limit: 20,
        query: { ...query, order_by: "sort_order" },
      }),
    ["GET", "/api/v0/projects/A/issues"],
    issuePageOf(rows),
    "list"
  );
  return app.issue.projectIssues.currentList!;
}
const ids = (list: Awaited<ReturnType<typeof listed>>) => list.group("")?.rows.map((row) => row.id);
async function created(id: string, cycle_id: string | null) {
  const { app, nerve } = tab;
  const request = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.createIssue("ws", "A", { name: id, ...(cycle_id ? { cycle_id } : {}) }),
    ["POST", "/api/v0/projects/A/issues"]
  );
  expect(request.call.body).toEqual({ name: id, ...(cycle_id ? { cycle_id } : {}) });
  request.call.answer(json(201, { ...issueDetailOf(id), sort_order: 10, cycle_id }));
  await request.promise;
}

// M6 P5b spec 2.2: a read elsewhere that only changed the work item's cycle or modules may change membership.
it.each<[string, ProjectIssueQuery, Partial<Issue>, Partial<Issue>]>([
  ["cycle", { cycle_id: [C1] }, { cycle_id: C1 }, { cycle_id: C2 }],
  ["modules", { module_id: [M1] }, { module_ids: [M1] }, { module_ids: [M2] }],
])("a read that only changed the %s marks the covering list's membership stale", async (_, query, held, read) => {
  const { app } = tab;
  const list = await listed(query, [listIssue("a", 1, held)]);
  app.issue.acceptListIssues([listIssue("a", 1, { ...held, updated_at: LATER })], app.issue.captureRead());
  expect(list.group("")?.stale).toBe(false);
  app.issue.acceptListIssues(
    [listIssue("a", 1, { ...read, updated_at: "2026-10-30T00:00:00Z" })],
    app.issue.captureRead()
  );
  expect(list.group("")?.stale).toBe(true);
  // A read is not a confirmed write: the row waits for the list's own read.
  expect(ids(list)).toEqual(["a"]);
});

// M6 P5b spec 2.3, R4, R6: the root passes the caller's cycle through; nerve alone decides whether it may join.
it("a creation sends the cycle it is created in", async () => {
  await created("R1", C1);
});

it.each([C2, null])("an update sends the cycle %s it moves to", async (cycle_id) => {
  await listed({}, [listIssue("a", 1, { cycle_id: C1 })]);
  const { app, nerve } = tab;
  const request = await requestedIssue(nerve, () => app.issue.projectIssues.updateIssue("ws", "A", "a", { cycle_id }), [
    "PATCH",
    "/api/v0/issues/a",
  ]);
  expect(request.call.body).toEqual({ cycle_id });
  request.call.answer(json(200, { ...issueDetailOf("a"), sort_order: 1, cycle_id, updated_at: LATER }));
  await request.promise;
  expect(app.issue.issues.getIssueById("a")?.cycle_id).toBe(cycle_id);
});

// M6 P5b spec 2.2: confirmed writes made while another page's list is current reach the cycle page's binding, which
// keeps only what nerve answered to be in its cycle.
it("a cycle page's list follows confirmed moves and creations by the answered cycle", async () => {
  const { app, nerve } = tab;
  const cycle = await listed({ cycle_id: [C1] }, [
    listIssue("a", 1, { cycle_id: C1 }),
    listIssue("b", 2, { cycle_id: C1 }),
  ]);
  const project = await listed({}, [
    listIssue("a", 1, { cycle_id: C1 }),
    listIssue("b", 2, { cycle_id: C1 }),
    listIssue("n", 3),
  ]);
  expect(app.issue.projectIssues.currentList).toBe(project);
  await answered(
    nerve,
    () => app.issue.projectIssues.updateIssue("ws", "A", "a", { cycle_id: C2 }),
    ["PATCH", "/api/v0/issues/a"],
    { ...issueDetailOf("a"), sort_order: 1, cycle_id: C2, updated_at: LATER },
    "moved to another cycle"
  );
  expect(ids(cycle)).toEqual(["b"]);
  expect(cycle.value?.total).toBe(1);
  expect(ids(project)).toEqual(["a", "b", "n"]);
  await created("c", C1);
  await created("d", null);
  expect(ids(cycle)).toEqual(["b", "c"]);
  expect(cycle.value?.total).toBe(2);
  expect(ids(project)).toEqual(["a", "b", "n", "c", "d"]);
  expect(project.value?.total).toBe(5);
});

// M6 P6b spec 2.5, R5, R6: the writes pass the caller's modules through, the whole set; nerve alone decides on them.
it("a creation sends the modules it is created in", async () => {
  const { app, nerve } = tab;
  const request = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.createIssue("ws", "A", { name: "R1", module_ids: [M1, M2] }),
    ["POST", "/api/v0/projects/A/issues"]
  );
  expect(request.call.body).toEqual({ name: "R1", module_ids: [M1, M2] });
  request.call.answer(json(201, { ...issueDetailOf("R1"), sort_order: 10, module_ids: [M1, M2] }));
  await request.promise;
  expect(app.issue.issues.getIssueById("R1")?.module_ids).toEqual([M1, M2]);
});

it.each([[[M2]], [[]]])("an update sends the modules %j it is in afterwards", async (module_ids) => {
  await listed({}, [listIssue("a", 1, { module_ids: [M1] })]);
  const { app, nerve } = tab;
  const request = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.updateIssue("ws", "A", "a", { modules: { remove: [M1], add: module_ids } }),
    ["PATCH", "/api/v0/issues/a"]
  );
  expect(request.call.body).toEqual({ module_ids });
  request.call.answer(json(200, { ...issueDetailOf("a"), sort_order: 1, module_ids, updated_at: LATER }));
  await request.promise;
  expect(app.issue.issues.getIssueById("a")?.module_ids).toEqual(module_ids);
});
