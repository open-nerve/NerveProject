/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { runInAction } from "mobx";
import { FakeNerve, answered, json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueDetailOf, issuePageOf, listIssue, requestedIssue } from "../fake-issues";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { loadProjects, projectOf, stateOf, labelOf } from "@/store/project/fake-projects";
import { membershipOf } from "@/store/member/workspace/fake-members";
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
    app.issue.workspaceSlug = ws.slug;
    app.issue.projectId = project.id;
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
const accept = (issues: import("@nerve/api-client").Issue[]) =>
  app.issue.acceptListIssues(issues, app.issue.captureRead());
const whole = () => ({
  total: 10,
  groups: [
    { key: "A", total: 5, data: [listIssue("a", 10, { state_id: "A" })], next_cursor: "A:c/?" },
    { key: "B", total: 5, data: [listIssue("b", 10, { state_id: "B" })], next_cursor: "B:c/?" },
  ],
});
const load = async (response = whole()) => {
  const request = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues(ws.slug, "A", "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  request.call.answer(json(200, response));
  return request.promise;
};

it("nextPageKeepsExactQueryAndCursor", async () => {
  await load();
  const { promise, call } = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchNextIssues(ws.slug, "A", "B"),
    ["GET", "/api/v0/projects/A/issues"]
  );
  expect(call.query).toEqual({
    order_by: "sort_order",
    sub_issue: "true",
    group_by: "state_id",
    limit: "10",
    cursor: "B:c/?",
    group_key: "B",
  });
  call.answer(json(200, { total: 5, data: [listIssue("c", 20, { state_id: "B" })], next_cursor: null }));
  await promise;
  expect(app.issue.projectIssues.getIssueIds("B")).toEqual(["b", "c"]);
});

it("W1 wholeArrivingWhileCreateIsOutKeepsServerCountsForOffscreenIssue", async () => {
  await load();
  const creating = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.createIssue(ws.slug, "A", { name: "new" }),
    ["POST", "/api/v0/projects/A/issues"]
  );
  await load();
  creating.call.answer(json(201, { ...issueDetailOf("x"), sort_order: 20, state_id: "B" }));
  await creating.promise;
  expect(app.issue.projectIssues.getGroupIssueCount("B")).toBe(5);
  expect(app.issue.projectIssues.groups?.get("B")?.stale).toBe(true);
  expect(app.issue.projectIssues.total).toBe(10);
});

it("coreWritesUseSingleCoordinatorBeforeCanonicalMigration", async () => {
  await load();
  const before = app.issue.issues.getIssueById("a");
  const first = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.updateIssue(ws.slug, "A", "a", { state_id: "B" }),
    ["PATCH", "/api/v0/issues/a"]
  );
  const second = app.issue.projectIssues.updateIssue(ws.slug, "A", "a", { name: "renamed" });
  expect(nerve.calls).toHaveLength(2);
  expect(app.issue.issues.getIssueById("a")).toEqual(before);
  expect(app.issue.projectIssues.getGroupIssueCount("A")).toBe(5);
  const rejection = expect(first.promise).rejects.toBeDefined();
  first.call.answer(json(403, { code: "forbidden", message: "refused" }));
  await rejection;
  await until(() => nerve.calls.length === 3, "queued successor");
  expect(app.issue.issues.getIssueById("a")).toEqual(before);
  nerve.calls[2]!.answer(json(200, { ...issueDetailOf("a"), state_id: "A", name: "renamed", sort_order: 10 }));
  await second;
  expect(app.issue.issues.getIssueById("a")?.name).toBe("renamed");
  expect(app.issue.projectIssues.getIssueIds("A")).toEqual(["a"]);
  expect(app.issue.projectIssues.getGroupIssueCount("A")).toBe(5);
  expect(app.issue.archivedIssues.groups).toBeUndefined();
});

it("staleReloadReplacesOnlyTargetFirstPage", async () => {
  await load();
  const list = app.issue.projectIssues.currentList!;
  list.markMembershipStale(undefined, listIssue("x", 20, { state_id: "B" }));
  const refresh = await requestedIssue(nerve, () => list.refreshStale(), ["GET", "/api/v0/projects/A/issues"]);
  expect(refresh.call.query.group_key).toBe("B");
  expect("cursor" in refresh.call.query).toBe(false);
  refresh.call.answer(json(200, { total: 2, data: [listIssue("new", 4, { state_id: "B" })], next_cursor: null }));
  await refresh.promise;
  expect(app.issue.projectIssues.getIssueIds("A")).toEqual(["a"]);
  expect(app.issue.projectIssues.getIssueIds("B")).toEqual(["new"]);
  expect(app.issue.projectIssues.total).toBe(10);
});

it("pageTotalReplacesEarlierAdjustedTotal", async () => {
  await load();
  await answered(
    nerve,
    () => app.issue.projectIssues.createIssue(ws.slug, "A", { name: "x" }),
    ["POST", "/api/v0/projects/A/issues"],
    { ...issueDetailOf("x"), state_id: "B", sort_order: 20 },
    "create"
  );
  expect(app.issue.projectIssues.getGroupIssueCount("B")).toBe(6);
  await answered(
    nerve,
    () => app.issue.projectIssues.fetchNextIssues(ws.slug, "A", "B"),
    ["GET", "/api/v0/projects/A/issues"],
    { total: 7, data: [listIssue("x", 20, { state_id: "B" })], next_cursor: null },
    "next"
  );
  expect(app.issue.projectIssues.getGroupIssueCount("B")).toBe(7);
  expect(app.issue.projectIssues.getIssueIds("B")).toEqual(["b", "x"]);
});

it("nameOrderedListWaitsForItsRequiredDirectoryBeforePublishingAnyRows", async () => {
  const state = stateOf(project, "Waiting", "unstarted", 1000);
  const request = await requestedIssue(
    nerve,
    () =>
      app.issue.projectIssues.fetchIssues(ws.slug, "A", "init-loader", {
        limit: 10,
        query: { group_by: undefined, order_by: "state_name" },
      }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  request.call.answer(json(200, issuePageOf([listIssue("x", 1, { state_id: state.id })])));
  await vi.advanceTimersByTimeAsync(10);
  expect(request.result.settled).toBe(false);
  expect(app.issue.issues.getIssueById("x")).toBeUndefined();
  expect(nerve.calls[1]).toMatchObject({ method: "GET", path: "/api/v0/projects/A/states" });
  nerve.calls[1]!.answer(json(200, { data: [state] }));
  await request.promise;
  expect(app.issue.projectIssues.groups?.get("")?.rows[0]?.key).toBe("waiting");
});
it("requiredDirectoryRefusalRejectsWithoutPublishingRowsOrNullNameKeys", async () => {
  const request = await requestedIssue(
    nerve,
    () =>
      app.issue.projectIssues.fetchIssues(ws.slug, "A", "init-loader", {
        limit: 10,
        query: { group_by: undefined, order_by: "label_name" },
      }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  request.call.answer(json(200, issuePageOf([listIssue("x", 1, { label_ids: ["label"] })])));
  await vi.advanceTimersByTimeAsync(10);
  expect(request.result.settled).toBe(false);
  expect(nerve.calls[1]).toMatchObject({ method: "GET", path: "/api/v0/projects/A/labels" });
  const refused = expect(request.promise).rejects.toBeDefined();
  nerve.calls[1]!.answer(json(403, { code: "forbidden" }));
  await refused;
  expect(app.issue.projectIssues.groups).toBeUndefined();
  expect(app.issue.issues.getIssueById("x")).toBeUndefined();
});
it("malformedLastRowCannotPublishEarlierEntitiesFromTheSamePage", async () => {
  await load();
  const before = app.issue.issues.getIssueById("a");
  const request = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues(ws.slug, "A", "mutation", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  const refused = expect(request.promise).rejects.toThrow("Unsafe page count");
  request.call.answer(
    json(
      200,
      issuePageOf([
        listIssue("a", 1, { name: "should not publish" }),
        listIssue("bad", 2, { link_count: Number.MAX_SAFE_INTEGER + 1 }),
      ])
    )
  );
  await refused;
  expect(app.issue.issues.getIssueById("a")).toEqual(before);
  expect(app.issue.issues.getIssueById("bad")).toBeUndefined();
  expect(app.issue.projectIssues.getIssueIds("A")).toEqual(["a"]);
});

it("selectedCellContinuationKeepsBothGroupKeysAndAllQueryFields", async () => {
  const request = await requestedIssue(
    nerve,
    () =>
      app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", {
        limit: 4,
        query: { sub_group_by: "priority", priority: ["high"] },
      }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  request.call.answer(
    json(200, {
      total: 5,
      groups: [
        {
          key: "A",
          total: 5,
          subgroups: [
            {
              key: "high",
              total: 5,
              data: [listIssue("a", 10, { state_id: "A", priority: "high" })],
              next_cursor: "cell:opaque",
            },
          ],
        },
      ],
      subgroup_totals: [{ key: "high", total: 5 }],
    })
  );
  await request.promise;
  const next = await requestedIssue(nerve, () => app.issue.projectIssues.fetchNextIssues("ws", "A", "A", "high"), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  expect(next.call.query).toEqual({
    order_by: "sort_order",
    sub_issue: "true",
    group_by: "state_id",
    sub_group_by: "priority",
    priority: "high",
    limit: "4",
    group_key: "A",
    sub_group_key: "high",
    cursor: "cell:opaque",
  });
  next.call.answer(json(200, { total: 5, data: [], next_cursor: null }));
  await next.promise;
});
it("pageMergesDeduplicateDroppedRowAndKeepsServerCursorBoundary", async () => {
  await load();
  const change = await requestedIssue(
    nerve,
    () =>
      app.issue.updateIssue("a", () => ({ state_id: "B", sort_order: 20 }), { group: "B" }, app.issue.scope("ws", "A")),
    ["PATCH", "/api/v0/issues/a"]
  );
  const after = { ...issueDetailOf("a"), state_id: "B", sort_order: 20, updated_at: "2026-10-29T00:00:00Z" };
  change.call.answer(json(200, after));
  await change.promise;
  expect(app.issue.projectIssues.groups?.get("B")?.boundary?.key).toBe(20);
  expect(app.issue.projectIssues.groups?.get("B")?.cursorBoundary?.key).toBe(10);
  const next = await requestedIssue(nerve, () => app.issue.projectIssues.fetchNextIssues("ws", "A", "B"), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  expect(next.call.query.cursor).toBe("B:c/?");
  next.call.answer(json(200, { total: 6, data: [after, listIssue("tail", 30, { state_id: "B" })], next_cursor: null }));
  await next.promise;
  expect(app.issue.projectIssues.getIssueIds("B")).toEqual(["b", "a", "tail"]);
  expect(app.issue.projectIssues.getGroupIssueCount("B")).toBe(6);
});
it("memberInitialAndNextPagesKeepRelationAndExplicitProjectScope", async () => {
  runInAction(() => {
    app.issue.profileIssuesFilter.filters.actor = {
      richFilters: { project_id__in: "019a0000-0000-7000-8000-000000000001,019a0000-0000-7000-8000-000000000002" },
      displayProperties: undefined,
      kanbanFilters: undefined,
      displayFilters: { order_by: "sort_order", group_by: "project" },
    };
  });
  const store = app.issue.profileIssues;
  const first = await requestedIssue(
    nerve,
    () => store.fetchIssues("ws", "actor", "init-loader", { limit: 15 }, "created"),
    ["GET", "/api/v0/workspaces/ws/members/actor/issues"]
  );
  expect(first.call.query).toEqual({
    relation: "created",
    project_id: "019a0000-0000-7000-8000-000000000001,019a0000-0000-7000-8000-000000000002",
    order_by: "sort_order",
    sub_issue: "true",
    group_by: "project_id",
    limit: "15",
  });
  first.call.answer(
    json(200, { total: 3, groups: [{ key: "A", total: 3, data: [listIssue("a", 10)], next_cursor: "member:next" }] })
  );
  await first.promise;
  const next = await requestedIssue(nerve, () => store.fetchNextIssues("ws", "actor", "A"), [
    "GET",
    "/api/v0/workspaces/ws/members/actor/issues",
  ]);
  expect(next.call.query).toEqual({ ...first.call.query, cursor: "member:next", group_key: "A" });
  next.call.answer(json(200, { total: 3, data: [listIssue("b", 20)], next_cursor: null }));
  await next.promise;
  expect(store.getIssueIds("A")).toEqual(["a", "b"]);
});
it("archiveAndRestoreConfirmEveryFetchedCopyOnceAndLeaveUnrelatedScopeAlone", async () => {
  const source = { ...issueDetailOf("x"), assignee_ids: ["actor"] };
  await answered(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 20, query: { group_by: undefined } }),
    ["GET", "/api/v0/projects/A/issues"],
    issuePageOf([source]),
    "project list"
  );
  await answered(
    nerve,
    () => app.issue.archivedIssues.fetchIssues("ws", "A", "init-loader", { limit: 20 }),
    ["GET", "/api/v0/projects/A/issues"],
    issuePageOf([]),
    "archive list"
  );
  await answered(
    nerve,
    () => app.issue.profileIssues.fetchIssues("ws", "actor", "init-loader", { limit: 20 }, "assigned"),
    ["GET", "/api/v0/workspaces/ws/members/actor/issues"],
    issuePageOf([source]),
    "profile list"
  );
  const pending = await requestedIssue(nerve, () => app.issue.projectIssues.archiveIssue("ws", "A", "x"), [
    "POST",
    "/api/v0/issues/x/archive",
  ]);
  expect([app.issue.projectIssues.total, app.issue.archivedIssues.total, app.issue.profileIssues.total]).toEqual([
    1, 0, 1,
  ]);
  pending.call.answer(
    json(200, { ...source, archived_at: "2026-10-29T00:00:00Z", updated_at: "2026-10-29T00:00:00Z" })
  );
  await pending.promise;
  expect([app.issue.projectIssues.total, app.issue.archivedIssues.total, app.issue.profileIssues.total]).toEqual([
    0, 1, 0,
  ]);
  await answered(
    nerve,
    () => app.issue.archivedIssues.restoreIssue("ws", "A", "x"),
    ["POST", "/api/v0/issues/x/unarchive"],
    { ...source, updated_at: "2026-10-29T00:00:01Z" },
    "restore"
  );
  expect([app.issue.projectIssues.total, app.issue.archivedIssues.total, app.issue.profileIssues.total]).toEqual([
    1, 0, 1,
  ]);
});

it("existingLabelDirectoryMissingOneReferenceIsRefetchedOnceBeforeTakingTheMinimum", async () => {
  const zebra = labelOf(project, "Zebra", 1),
    alpha = labelOf(project, "Alpha", 2);
  await answered(
    nerve,
    () => app.label.fetchProjectLabels("A"),
    ["GET", "/api/v0/projects/A/labels"],
    { data: [zebra] },
    "initial labels"
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
  await until(() => nerve.calls.length === 3, "missing label refetch");
  expect(app.issue.issues.getIssueById("x")).toBeUndefined();
  nerve.calls[2]!.answer(json(200, { data: [zebra, alpha] }));
  await request.promise;
  expect(app.issue.projectIssues.groups?.get("")?.rows[0]?.key).toBe("alpha");
  expect(nerve.calls.filter((call) => call.path.endsWith("/labels"))).toHaveLength(2);
});

it("authoritativeDirectoryStillMissingAReferenceRejectsWithoutRetryLoop", async () => {
  const request = await requestedIssue(
    nerve,
    () =>
      app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", {
        limit: 10,
        query: { group_by: undefined, order_by: "state_name" },
      }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  request.call.answer(json(200, issuePageOf([listIssue("x", 1, { state_id: "missing" })])));
  await until(() => nerve.calls.length === 2, "states directory");
  const refused = expect(request.promise).rejects.toThrow("Required issue directory");
  nerve.calls[1]!.answer(json(200, { data: [] }));
  await refused;
  expect(nerve.calls).toHaveLength(2);
  expect(app.issue.projectIssues.groups).toBeUndefined();
  expect(app.issue.issues.getIssueById("x")).toBeUndefined();
});

it("writeFreezesBodyBeforeAndHeldSourcesAfterRequiredDirectoryWait", async () => {
  const stateA = stateOf(project, "Alpha", "unstarted", 1),
    stateB = stateOf(project, "Beta", "started", 2);
  await answered(
    nerve,
    () => app.state.fetchProjectStates("A"),
    ["GET", "/api/v0/projects/A/states"],
    { data: [stateA] },
    "states"
  );
  const options = { limit: 10, query: { group_by: "state_id" as const, order_by: "state_name" as const } };
  await answered(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", options),
    ["GET", "/api/v0/projects/A/issues"],
    {
      total: 3,
      groups: [
        { key: stateA.id, total: 2, data: [listIssue("a", 1, { state_id: stateA.id })], next_cursor: "A-next" },
        { key: stateB.id, total: 1, data: [], next_cursor: "B-next" },
      ],
    },
    "name list"
  );
  // A detail has the off-window issue in a state the list's old directory did not contain.
  accept([listIssue("x", 5, { state_id: stateB.id, name: "old" })]);
  const wait = await requestedIssue(
    nerve,
    () =>
      app.issue.updateIssue(
        "x",
        (held) => ({ name: `${held.name}!`, state_id: stateA.id }),
        undefined,
        app.issue.scope("ws", "A")
      ),
    ["GET", "/api/v0/projects/A/states"]
  );
  accept([listIssue("x", 5, { state_id: stateA.id, name: "latest", updated_at: "2026-10-28T01:00:00Z" })]);
  expect(nerve.calls.some((call) => call.method === "PATCH")).toBe(false);
  wait.call.answer(json(200, { data: [stateA, stateB] }));
  await until(() => nerve.calls.some((call) => call.method === "PATCH"), "write after directory");
  const patch = nerve.calls.at(-1)!;
  expect(patch.body).toEqual({ name: "latest!", state_id: stateA.id });
  patch.answer(
    json(200, { ...issueDetailOf("x"), state_id: stateA.id, name: "latest!", updated_at: "2026-10-29T00:00:00Z" })
  );
  await wait.promise;
  expect([
    app.issue.projectIssues.getGroupIssueCount(stateA.id),
    app.issue.projectIssues.getGroupIssueCount(stateB.id),
  ]).toEqual([2, 1]);
  expect([...app.issue.projectIssues.groups!.values()].every((group) => group.stale)).toBe(true);
});

it("ownReadBeforeWriteAnswerKeepsOffWindowBeforeMembershipWithoutWholeRefetch", async () => {
  await load();
  const before = listIssue("x", 20, { state_id: "A" });
  accept([before]);
  const patch = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.updateIssue("ws", "A", "x", { state_id: "B" }),
    ["PATCH", "/api/v0/issues/x"]
  );
  const after = { ...issueDetailOf("x"), state_id: "B", sort_order: 20, updated_at: "2026-10-29T00:00:00Z" };
  await answered(
    nerve,
    () => app.issue.projectIssues.fetchNextIssues("ws", "A", "B"),
    ["GET", "/api/v0/projects/A/issues"],
    { total: 6, data: [after], next_cursor: null },
    "own read"
  );
  patch.call.answer(json(200, after));
  await patch.promise;
  expect([
    app.issue.projectIssues.total,
    app.issue.projectIssues.getGroupIssueCount("A"),
    app.issue.projectIssues.getGroupIssueCount("B"),
  ]).toEqual([10, 4, 6]);
  expect([...app.issue.projectIssues.groups!.values()].every((group) => !group.stale)).toBe(true);
  expect(nerve.calls).toHaveLength(3);
});

it("assigneeNameSortKeepsInactiveMembersAndWaitsForEveryProfile", async () => {
  const former = membershipOf("Alpha", { is_active: false }, "ws"),
    current = membershipOf("Zebra", {}, "ws");
  const request = await requestedIssue(
    nerve,
    () =>
      app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", {
        limit: 10,
        query: { group_by: undefined, order_by: "assignee_name" },
      }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  request.call.answer(
    json(200, issuePageOf([listIssue("x", 1, { assignee_ids: [current.member.id, former.member.id] })]))
  );
  await until(() => nerve.calls.length === 2, "workspace profiles");
  expect(nerve.calls[1]).toMatchObject({ path: "/api/v0/workspaces/ws/members" });
  nerve.calls[1]!.answer(json(200, { data: [former, current] }));
  await request.promise;
  expect(app.issue.projectIssues.groups?.get("")?.rows[0]?.key).toBe("alpha");
});

it.each([200, 403])("listCreatedDuringWrite prepares directories; %s preserves the confirmed write", async (status) => {
  await answered(
    nerve,
    () => app.label.fetchProjectLabels("A"),
    ["GET", "/api/v0/projects/A/labels"],
    { data: [] },
    "initial directory"
  );
  accept([listIssue("x", 20, { label_ids: ["before-label"] })]);
  const patch = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.updateIssue("ws", "A", "x", { label_ids: ["after-label"] }),
    ["PATCH", "/api/v0/issues/x"]
  );
  await answered(
    nerve,
    () =>
      app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", {
        limit: 10,
        query: { group_by: undefined, order_by: "label_name" },
      }),
    ["GET", "/api/v0/projects/A/issues"],
    { total: 5, data: [listIssue("a", 1)], next_cursor: "next" },
    "new list"
  );
  const baseline = nerve.calls.length;
  patch.call.answer(
    json(200, { ...issueDetailOf("x"), label_ids: ["after-label"], updated_at: "2026-10-29T00:00:00Z" })
  );
  await vi.advanceTimersByTimeAsync(10);
  expect(patch.result.settled).toBe(false);
  expect(nerve.calls[baseline]).toMatchObject({ path: "/api/v0/projects/A/labels" });
  nerve.calls[baseline]!.answer(
    json(
      status,
      status === 403
        ? { code: "forbidden" }
        : {
            data: [
              labelOf(project, "Before", 1, { id: "before-label" }),
              labelOf(project, "After", 2, { id: "after-label" }),
            ],
          }
    )
  );
  await patch.promise;
  if (status === 200) {
    expect(app.issue.projectIssues.getIssueLoader()).toBe("init-loader");
    await until(
      () => nerve.calls.some((call) => call.path === "/api/v0/issues/x" && call.method === "GET"),
      "direct confirmation verification"
    );
    nerve.calls
      .at(-1)!
      .answer(json(200, { ...issueDetailOf("x"), label_ids: ["after-label"], updated_at: "2026-10-29T00:00:00Z" }));
    await vi.advanceTimersByTimeAsync(1);
    const refresh = nerve.calls.at(-1)!;
    if (refresh.path === "/api/v0/projects/A/issues") refresh.answer(json(200, issuePageOf([])));
    await vi.advanceTimersByTimeAsync(1);
  }
  expect(app.issue.issues.getIssueById("x")?.label_ids).toEqual(["after-label"]);
  expect(nerve.calls.filter((call) => call.method === "PATCH")).toHaveLength(1);
  if (status === 403) {
    expect(app.issue.projectIssues.currentList?.error).toBeDefined();
    expect(app.issue.projectIssues.groups?.get("")?.stale).toBe(true);
  }
});

it.each([
  {
    and: [
      { project_id__in: "019a0000-0000-7000-8000-000000000001" },
      { project_id__in: "019a0000-0000-7000-8000-000000000002" },
    ],
  },
  { project_id__exact: "019a0000-0000-7000-8000-000000000001,019a0000-0000-7000-8000-000000000002" },
  { project_id__in: ",019a0000-0000-7000-8000-000000000001" },
])("F01 clears an unrepresentable member project condition %j", async (expression) => {
  runInAction(() => {
    app.issue.profileIssuesFilter.filters.actor = {
      richFilters: expression,
      displayProperties: undefined,
      kanbanFilters: undefined,
      displayFilters: {},
    };
  });
  const request = await requestedIssue(
    nerve,
    () => app.issue.profileIssues.fetchIssues("ws", "actor", "init-loader", { limit: 15 }, "assigned"),
    ["GET", "/api/v0/workspaces/ws/members/actor/issues"]
  );
  expect(request.call.query).toEqual({ relation: "assigned", order_by: "-created_at", sub_issue: "true", limit: "15" });
  request.call.answer(json(200, { total: 0, data: [], next_cursor: null }));
  await request.promise;
});

it.each([
  ["broken", {}],
  [{ and: [null, { priority__in: "high" }] }, { priority: "high" }],
  [{ and: "broken", project_id__in: "019a0000-0000-7000-8000-000000000001" }, {}],
  [
    {
      and: [
        [],
        7,
        "broken",
        { and: [{ project_id__in: "019a0000-0000-7000-8000-000000000002" }], priority__in: "low" },
        { priority__in: "high" },
        { project_id__in: "019A0000-0000-7000-8000-000000000001" },
      ],
    },
    { priority: "high", project_id: "019a0000-0000-7000-8000-000000000001" },
  ],
])(
  "fix1 I2/I3 member consumer defaults malformed structure while preserving valid conditions %j",
  async (expression, expected) => {
    runInAction(() => {
      app.issue.profileIssuesFilter.filters.actor = {
        richFilters: JSON.parse(JSON.stringify(expression)),
        displayProperties: undefined,
        kanbanFilters: undefined,
        displayFilters: {},
      };
    });
    const request = await requestedIssue(
      nerve,
      () => app.issue.profileIssues.fetchIssues("ws", "actor", "init-loader", { limit: 15 }, "assigned"),
      ["GET", "/api/v0/workspaces/ws/members/actor/issues"]
    );
    expect(request.call.query).toEqual({
      relation: "assigned",
      order_by: "-created_at",
      sub_issue: "true",
      limit: "15",
      ...expected,
    });
    request.call.answer(json(200, { total: 0, data: [], next_cursor: null }));
    await request.promise;
  }
);

it("sends the confirmed member fields after local preference save and reload", async () => {
  const data = new Map<string, string>();
  vi.stubGlobal("window", {
    localStorage: {
      getItem: (key: string) => data.get(key) ?? null,
      setItem: (key: string, value: string) => data.set(key, value),
    },
  });
  const id = "019a0000-0000-7000-8000-000000000001";
  try {
    runInAction(() => {
      app.issue.currentUserId = "viewer";
    });
    await app.issue.profileIssuesFilter.updateFilterExpression("ws", "actor", {
      and: [{ state_id__in: id }, { created_by_id__in: id }, { priority__in: "high" }, { project_id__in: id }],
    });
    expect(JSON.parse(data.get("issue_local_filters")!)[0].filters.rich_filters).toEqual({
      and: [{ priority__in: "high" }, { project_id__in: id }],
    });
    await app.issue.profileIssuesFilter.fetchFilters("ws", "actor");
    expect(app.issue.profileIssuesFilter.getIssueFilters("actor")?.richFilters).toEqual({
      and: [{ priority__in: "high" }, { project_id__in: id }],
    });
    const request = await requestedIssue(
      nerve,
      () => app.issue.profileIssues.fetchIssues("ws", "actor", "init-loader", { limit: 15 }, "assigned"),
      ["GET", "/api/v0/workspaces/ws/members/actor/issues"]
    );
    expect(request.call.query).toMatchObject({ project_id: id, priority: "high" });
    expect(request.call.query).not.toHaveProperty("state_id");
    expect(request.call.query).not.toHaveProperty("created_by_id");
    request.call.answer(json(200, { total: 0, data: [], next_cursor: null }));
    await request.promise;
  } finally {
    vi.unstubAllGlobals();
  }
});
