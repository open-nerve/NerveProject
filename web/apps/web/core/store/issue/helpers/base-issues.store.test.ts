/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { runInAction } from "mobx";
import { FakeNerve, answered, json, problem } from "@/lib/auth/fake-nerve";
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
      app.issue.updateIssue(
        "a",
        () => ({ state_id: "B", sort_order: 20 }),
        { group: "B", binding: app.issue.projectIssues.currentList },
        app.issue.scope("ws", "A")
      ),
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
  const scopedProject = "019a0000-0000-7000-8000-000000000001";
  await loadProjects(nerve, app.projectRoot.project, ws, [projectOf("P", ws.id, { id: scopedProject })]);
  runInAction(() => {
    app.issue.profileIssuesFilter.filters[JSON.stringify(["ws", "actor"])] = {
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
    json(200, {
      total: 3,
      groups: [
        {
          key: scopedProject,
          total: 3,
          data: [listIssue("a", 10, { project_id: scopedProject })],
          next_cursor: "member:next",
        },
      ],
    })
  );
  await first.promise;
  const next = await requestedIssue(nerve, () => store.fetchNextIssues("ws", "actor", scopedProject), [
    "GET",
    "/api/v0/workspaces/ws/members/actor/issues",
  ]);
  expect(next.call.query).toEqual({ ...first.call.query, cursor: "member:next", group_key: scopedProject });
  next.call.answer(
    json(200, { total: 3, data: [listIssue("b", 20, { project_id: scopedProject })], next_cursor: null })
  );
  await next.promise;
  expect(store.getIssueIds(scopedProject)).toEqual(["a", "b"]);
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
    app.issue.profileIssuesFilter.filters[JSON.stringify(["ws", "actor"])] = {
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
      app.issue.profileIssuesFilter.filters[JSON.stringify(["ws", "actor"])] = {
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
      app.user.data = { id: "viewer" } as NonNullable<typeof app.user.data>;
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

it("returns the required directory's HTTP refusal without publishing the page", async () => {
  const request = await requestedIssue(
    nerve,
    () =>
      app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", {
        limit: 10,
        query: { order_by: "state_name" },
      }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  request.call.answer(json(200, issuePageOf([listIssue("x", 1, { state_id: "missing" })])));
  await until(() => nerve.calls.length === 2, "required states");
  const refused = expect(request.promise).rejects.toMatchObject({ status: 403, problem: { code: "forbidden" } });
  nerve.calls[1]!.answer(problem(403, "forbidden"));
  await refused;
  expect(app.issue.projectIssues.groups).toBeUndefined();
  expect(app.issue.issues.held("x")).toBeUndefined();
  expect(app.issue.projectIssues.getIssueLoader()).not.toBe("init-loader");
});

it("losing one project during its list read does not clear another project's list", async () => {
  const b = projectOf("B", ws.id, { id: "B" });
  await loadProjects(nerve, app.projectRoot.project, ws, [project, b]);
  const other = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "B", "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/B/issues"]
  );
  other.call.answer(json(200, issuePageOf([listIssue("other", 1, { project_id: "B" })])));
  await other.promise;
  const retained = app.issue.projectIssues.currentList!;
  const gone = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  await loadProjects(nerve, app.projectRoot.project, ws, [b]);
  gone.call.answer(json(200, issuePageOf([listIssue("old", 1)])));
  await gone.promise;
  expect(app.issue.projectIssues.groups).toBeUndefined();
  expect(retained.reconciled.value?.groups.get("")?.rows.map((row) => row.id)).toEqual(["other"]);
  expect(app.issue.issues.held("old")).toBeUndefined();
});

it("workspace admin with project guest observes the guest visibility switch and hides held data until a new page", async () => {
  const admin = { ...ws, role: 20 as const };
  const guest = projectOf("A", ws.id, { id: "A", member_role: 5, guest_view_all_features: true });
  await loadWorkspaces(nerve, app.workspaceRoot, [admin]);
  await loadProjects(nerve, app.projectRoot.project, admin, [guest]);
  const first = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  first.call.answer(json(200, issuePageOf([listIssue("held", 1)])));
  await first.promise;
  const old = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  await loadProjects(nerve, app.projectRoot.project, admin, [{ ...guest, guest_view_all_features: false }]);
  expect.soft(app.issue.issues.getIssueById("held")).toBeUndefined();
  expect.soft(app.issue.projectIssues.groups).toBeUndefined();
  old.call.answer(json(200, issuePageOf([listIssue("old", 2)])));
  await old.promise;
  expect.soft(app.issue.issues.held("old")).toBeUndefined();
  expect.soft(app.issue.projectIssues.total).toBeUndefined();
  const fresh = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  fresh.call.answer(json(200, issuePageOf([listIssue("fresh", 3)])));
  await fresh.promise;
  expect(app.issue.projectIssues.getIssueIds()).toEqual(["fresh"]);
});

it("refreshing an unchanged permission signature does not invalidate an in-flight list", async () => {
  const read = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  await loadProjects(nerve, app.projectRoot.project, ws, [{ ...project, name: "renamed" }]);
  read.call.answer(json(200, issuePageOf([listIssue("valid", 1)])));
  await read.promise;
  expect(app.issue.projectIssues.getIssueIds()).toEqual(["valid"]);
});

it("workspace role change leaves a project's unchanged effective read scope valid", async () => {
  const read = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  await loadWorkspaces(nerve, app.workspaceRoot, [{ ...ws, role: 5 }]);
  read.call.answer(json(200, issuePageOf([listIssue("visible", 1)])));
  await read.promise;
  expect(app.issue.projectIssues.getIssueIds()).toEqual(["visible"]);
});

it("member pages invalidate only the covered project set, retaining an unrelated project's whole response", async () => {
  const { IssueListBinding } = await import("./issue-list-binding");
  const b = projectOf("B", ws.id, { id: "B" });
  await loadProjects(nerve, app.projectRoot.project, ws, [project, b]);
  const member = (ids?: string[]) =>
    new IssueListBinding(app.issue, {
      scope: "member",
      workspaceSlug: "ws",
      workspaceID: ws.id,
      userID: "actor",
      query: { relation: "assigned", project_id: ids, limit: 10 },
    });
  const aList = member(["A"]),
    bList = member(["B"]),
    all = member();
  const a = await requestedIssue(nerve, () => aList.fetchFirst(), [
    "GET",
    "/api/v0/workspaces/ws/members/actor/issues",
  ]);
  const bb = await requestedIssue(nerve, () => bList.fetchFirst(), [
    "GET",
    "/api/v0/workspaces/ws/members/actor/issues",
  ]);
  const both = await requestedIssue(nerve, () => all.fetchFirst(), [
    "GET",
    "/api/v0/workspaces/ws/members/actor/issues",
  ]);
  await loadProjects(nerve, app.projectRoot.project, ws, [b]);
  a.call.answer(json(200, issuePageOf([listIssue("a", 1)])));
  bb.call.answer(json(200, issuePageOf([listIssue("b", 1, { project_id: "B" })])));
  both.call.answer(json(200, issuePageOf([listIssue("a2", 2), listIssue("b2", 2, { project_id: "B" })])));
  await Promise.all([a.promise, bb.promise, both.promise]);
  expect(aList.value).toBeUndefined();
  expect(all.value).toBeUndefined();
  expect(bList.group()?.rows.map((row) => row.id)).toEqual(["b"]);
  expect(app.issue.issues.held("a")).toBeUndefined();
  expect(app.issue.issues.held("a2")).toBeUndefined();
  expect(app.issue.issues.held("b2")).toBeUndefined();
});

it("member part after project permission loss cannot replace rows or total of the old page", async () => {
  const { IssueListBinding } = await import("./issue-list-binding");
  const member = new IssueListBinding(app.issue, {
    scope: "member",
    workspaceSlug: "ws",
    workspaceID: ws.id,
    userID: "actor",
    query: { relation: "assigned", group_by: "project_id", limit: 10 },
  });
  const first = await requestedIssue(nerve, () => member.fetchFirst(), [
    "GET",
    "/api/v0/workspaces/ws/members/actor/issues",
  ]);
  first.call.answer(
    json(200, { total: 2, groups: [{ key: "A", total: 2, data: [listIssue("first", 1)], next_cursor: "next" }] })
  );
  await first.promise;
  const part = await requestedIssue(nerve, () => member.fetchNext("A"), [
    "GET",
    "/api/v0/workspaces/ws/members/actor/issues",
  ]);
  await loadProjects(nerve, app.projectRoot.project, ws, []);
  part.call.answer(json(200, { total: 2, data: [listIssue("old", 2)], next_cursor: null }));
  await part.promise;
  expect(member.value).toBeUndefined();
  expect(member.group("A")).toBeUndefined();
  expect(app.issue.issues.held("old")).toBeUndefined();
});

it("permission loss during post-page verification cannot re-expose the invalidated page", async () => {
  let release!: () => void;
  let started = false;
  const wait = new Promise<void>((resolve) => {
    release = resolve;
  });
  const drain = vi.spyOn(app.issue.verification, "drain").mockImplementation(async () => {
    started = true;
    await wait;
  });
  try {
    const read = await requestedIssue(
      nerve,
      () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 10 }),
      ["GET", "/api/v0/projects/A/issues"]
    );
    read.call.answer(json(200, issuePageOf([listIssue("old", 1)])));
    await until(() => started, "verification drain");
    await loadProjects(nerve, app.projectRoot.project, ws, []);
    release();
    await read.promise;
    expect(app.issue.projectIssues.groups).toBeUndefined();
    expect(app.issue.issues.getIssueById("old")).toBeUndefined();
  } finally {
    release();
    drain.mockRestore();
  }
});

it("workspace loss drops its own page without invalidating an unrelated workspace", async () => {
  const { IssueListBinding } = await import("./issue-list-binding");
  const otherWorkspace = workspaceOf("other");
  const otherProject = projectOf("B", otherWorkspace.id, { id: "B" });
  await loadWorkspaces(nerve, app.workspaceRoot, [ws, otherWorkspace]);
  await loadProjects(nerve, app.projectRoot.project, otherWorkspace, [otherProject]);
  const a = new IssueListBinding(app.issue, {
    scope: "project",
    workspaceSlug: "ws",
    workspaceID: ws.id,
    projectID: "A",
    query: { limit: 10 },
  });
  const b = new IssueListBinding(app.issue, {
    scope: "project",
    workspaceSlug: "other",
    workspaceID: otherWorkspace.id,
    projectID: "B",
    query: { limit: 10 },
  });
  const old = await requestedIssue(nerve, () => a.fetchFirst(), ["GET", "/api/v0/projects/A/issues"]);
  const other = await requestedIssue(nerve, () => b.fetchFirst(), ["GET", "/api/v0/projects/B/issues"]);
  await loadWorkspaces(nerve, app.workspaceRoot, [otherWorkspace]);
  old.call.answer(json(200, issuePageOf([listIssue("old", 1)])));
  other.call.answer(
    json(200, issuePageOf([listIssue("other", 1, { project_id: "B", workspace_id: otherWorkspace.id })]))
  );
  await Promise.all([old.promise, other.promise]);
  expect(a.value).toBeUndefined();
  expect(b.group()?.rows.map((row) => row.id)).toEqual(["other"]);
  expect(app.issue.issues.held("old")).toBeUndefined();
});

it("project member changing to guest invalidates an old answer although both roles can read", async () => {
  const old = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  await loadProjects(nerve, app.projectRoot.project, ws, [
    { ...project, member_role: 5, guest_view_all_features: true },
  ]);
  old.call.answer(json(200, issuePageOf([listIssue("old", 1)])));
  await old.promise;
  expect(app.issue.issues.held("old")).toBeUndefined();
  expect(app.issue.projectIssues.groups).toBeUndefined();
  const fresh = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  fresh.call.answer(json(200, issuePageOf([listIssue("guest-visible", 2)])));
  await fresh.promise;
  expect(app.issue.projectIssues.getIssueIds()).toEqual(["guest-visible"]);
});

it("permission recovery retries a fresh whole page rather than an obsolete failed part", async () => {
  await load();
  const next = await requestedIssue(nerve, () => app.issue.projectIssues.fetchNextIssues("ws", "A", "B"), [
    "GET",
    "/api/v0/projects/A/issues",
  ]);
  next.call.answer(problem(403, "forbidden"));
  await expect(next.promise).rejects.toBeDefined();
  const list = app.issue.projectIssues.currentList!;
  expect(list.error).toBeDefined();
  await loadProjects(nerve, app.projectRoot.project, ws, []);
  await loadProjects(nerve, app.projectRoot.project, ws, [project]);
  expect.soft(list.error).toBeUndefined();
  const retry = await requestedIssue(nerve, () => list.retryFailed(), ["GET", "/api/v0/projects/A/issues"]);
  expect.soft(retry.call.query).not.toHaveProperty("cursor");
  expect.soft(retry.call.query).not.toHaveProperty("group_key");
  retry.call.answer(json(200, issuePageOf([listIssue("new", 1)])));
  await retry.promise;
  expect(app.issue.projectIssues.getIssueIds()).toEqual(["new"]);
});

it.each([403, 404])(
  "a member page returns an unknown project directory refusal (%s) without publishing",
  async (status) => {
    const read = await requestedIssue(
      nerve,
      () => app.issue.profileIssues.fetchIssues("ws", "actor", "init-loader", { limit: 10 }, "assigned"),
      ["GET", "/api/v0/workspaces/ws/members/actor/issues"]
    );
    read.call.answer(json(200, issuePageOf([listIssue("unknown", 1, { project_id: "not-loaded" })])));
    await until(
      () => nerve.calls.some((call) => call.path === "/api/v0/projects/not-loaded"),
      "project permission directory"
    );
    const refused = expect(read.promise).rejects.toMatchObject({ status });
    nerve.calls.find((call) => call.path === "/api/v0/projects/not-loaded")!.answer(problem(status, "forbidden"));
    await refused;
    expect(app.issue.profileIssues.groups).toBeUndefined();
    expect(app.issue.profileIssues.total).toBeUndefined();
    expect(app.issue.issues.held("unknown")).toBeUndefined();
    expect(app.issue.profileIssues.currentList?.error).toMatchObject({ status });
    expect(app.issue.profileIssues.getIssueLoader()).not.toBe("init-loader");
    expect(nerve.to("/api/v0/projects/not-loaded")).toHaveLength(1);
  }
);

it("a member page with a newly loaded project publishes only after its permission directory answers", async () => {
  const read = await requestedIssue(
    nerve,
    () => app.issue.profileIssues.fetchIssues("ws", "actor", "init-loader", { limit: 10 }, "assigned"),
    ["GET", "/api/v0/workspaces/ws/members/actor/issues"]
  );
  read.call.answer(json(200, issuePageOf([listIssue("visible", 1, { project_id: "B" })])));
  await until(() => nerve.calls.some((call) => call.path === "/api/v0/projects/B"), "project permission directory");
  expect(app.issue.profileIssues.groups).toBeUndefined();
  nerve.calls.find((call) => call.path === "/api/v0/projects/B")!.answer(json(200, projectOf("B", ws.id, { id: "B" })));
  await read.promise;
  expect(app.issue.profileIssues.getIssueIds()).toEqual(["visible"]);
});

it("a project directory that still reports no membership rejects the whole member page", async () => {
  const read = await requestedIssue(
    nerve,
    () => app.issue.profileIssues.fetchIssues("ws", "actor", "init-loader", { limit: 10 }, "assigned"),
    ["GET", "/api/v0/workspaces/ws/members/actor/issues"]
  );
  read.call.answer(json(200, issuePageOf([listIssue("unknown", 1, { project_id: "B" })])));
  await until(() => nerve.calls.some((call) => call.path === "/api/v0/projects/B"), "project directory");
  const rejected = expect(read.promise).rejects.toThrow("Required issue project permission");
  nerve.calls
    .find((call) => call.path === "/api/v0/projects/B")!
    .answer(json(200, projectOf("B", ws.id, { id: "B", member_role: null })));
  await rejected;
  expect(app.issue.profileIssues.groups).toBeUndefined();
  expect(app.issue.profileIssues.total).toBeUndefined();
  expect(app.issue.issues.held("unknown")).toBeUndefined();
  expect(app.issue.profileIssues.getIssueLoader()).not.toBe("init-loader");
});

it("a permission loss while awaiting a missing project directory invalidates the whole original member page", async () => {
  const b = projectOf("B", ws.id, { id: "B" });
  const read = await requestedIssue(
    nerve,
    () => app.issue.profileIssues.fetchIssues("ws", "actor", "init-loader", { limit: 10 }, "assigned"),
    ["GET", "/api/v0/workspaces/ws/members/actor/issues"]
  );
  read.call.answer(json(200, issuePageOf([listIssue("old", 1, { project_id: "B" })])));
  await until(() => nerve.calls.some((call) => call.path === "/api/v0/projects/B"), "project directory");
  await loadProjects(nerve, app.projectRoot.project, ws, [project, b]);
  await loadProjects(nerve, app.projectRoot.project, ws, [project]);
  nerve.calls.find((call) => call.path === "/api/v0/projects/B")!.answer(json(200, b));
  await read.promise;
  expect(app.issue.profileIssues.groups).toBeUndefined();
  expect(app.issue.profileIssues.total).toBeUndefined();
  expect(app.issue.issues.held("old")).toBeUndefined();
  const fresh = await requestedIssue(
    nerve,
    () => app.issue.profileIssues.fetchIssues("ws", "actor", "init-loader", { limit: 10 }, "assigned"),
    ["GET", "/api/v0/workspaces/ws/members/actor/issues"]
  );
  fresh.call.answer(json(200, issuePageOf([])));
  await fresh.promise;
  expect(app.issue.profileIssues.total).toBe(0);
});

it("a member answer cannot publish an out-of-scope row whose project permission changed in flight", async () => {
  const { IssueListBinding } = await import("./issue-list-binding");
  const b = projectOf("B", ws.id, { id: "B" });
  await loadProjects(nerve, app.projectRoot.project, ws, [project, b]);
  const member = new IssueListBinding(app.issue, {
    scope: "member",
    workspaceSlug: "ws",
    workspaceID: ws.id,
    userID: "actor",
    query: { relation: "assigned", project_id: ["A"], limit: 10 },
  });
  const read = await requestedIssue(nerve, () => member.fetchFirst(), [
    "GET",
    "/api/v0/workspaces/ws/members/actor/issues",
  ]);
  await loadProjects(nerve, app.projectRoot.project, ws, [project]);
  await loadProjects(nerve, app.projectRoot.project, ws, [project, b]);
  const rejected = expect(read.promise).rejects.toThrow("Issue page does not match its scope");
  read.call.answer(json(200, issuePageOf([listIssue("old-b", 1, { project_id: "B" })])));
  await rejected;
  expect(member.value).toBeUndefined();
  expect(app.issue.issues.held("old-b")).toBeUndefined();
});

it("a member page rejects a visible project outside its explicit project scope", async () => {
  const { IssueListBinding } = await import("./issue-list-binding");
  const b = projectOf("B", ws.id, { id: "B" });
  await loadProjects(nerve, app.projectRoot.project, ws, [project, b]);
  const member = new IssueListBinding(app.issue, {
    scope: "member",
    workspaceSlug: "ws",
    workspaceID: ws.id,
    userID: "actor",
    query: { relation: "assigned", project_id: ["A"], limit: 10 },
  });
  const read = await requestedIssue(nerve, () => member.fetchFirst(), [
    "GET",
    "/api/v0/workspaces/ws/members/actor/issues",
  ]);
  const rejected = expect(read.promise).rejects.toThrow("Issue page does not match its scope");
  read.call.answer(json(200, issuePageOf([listIssue("b", 1, { project_id: "B" })])));
  await rejected;
  expect(member.value).toBeUndefined();
  expect(app.issue.issues.held("b")).toBeUndefined();
});

it("losing permission ends the invalidated list loader without waiting for its old response", async () => {
  const read = await requestedIssue(
    nerve,
    () => app.issue.projectIssues.fetchIssues("ws", "A", "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  await loadProjects(nerve, app.projectRoot.project, ws, []);
  expect.soft(app.issue.projectIssues.getIssueLoader()).not.toBe("init-loader");
  read.call.answer(json(200, issuePageOf([])));
  await read.promise;
});

it("a verification from a lost permission identity does not keep the list loading", async () => {
  await load();
  const verify = await requestedIssue(nerve, () => app.issue.issues.refreshAuthoritative("a"), [
    "GET",
    "/api/v0/issues/a",
  ]);
  expect(app.issue.projectIssues.getIssueLoader()).toBe("init-loader");
  await loadProjects(nerve, app.projectRoot.project, ws, []);
  expect.soft(app.issue.projectIssues.getIssueLoader()).not.toBe("init-loader");
  verify.call.answer(json(200, issueDetailOf("a")));
  await verify.promise;
});

const options = (range: string) => ({ limit: 4, query: { group_by: "target_date" as const, target_date: [range] } });
it("empty calendar intersection remains logical across writes permissions and window changes", async () => {
  const root = app.issue;
  const issues = root.projectIssues;
  runInAction(() => {
    root.projectIssuesFilter.filters.A.richFilters = { and: [{ target_date__exact: "2026-11-15" }] };
  });
  const serial = root.readSerial;
  await issues.fetchIssues("ws", "A", "init-loader", options("2026-10-01..2026-10-31"));
  const empty = issues.currentList!;
  expect(empty.value).toEqual({ total: 0, groups: new Map() });
  expect(empty.value?.read).toBeUndefined();
  expect(root.readSerial).toBe(serial);
  expect(nerve.calls).toHaveLength(0);
  const create = await requestedIssue(
    nerve,
    () => issues.createIssue("ws", "A", { name: "x", target_date: "2026-10-15" }),
    ["POST", "/api/v0/projects/A/issues"]
  );
  create.call.answer(json(201, { ...issueDetailOf("x"), target_date: "2026-10-15" }));
  await create.promise;
  expect(empty.value).toEqual({ total: 0, groups: new Map() });
  expect(root.readSerial).toBe(serial);
  app.projectRoot.project.confirmProject(project, (held) => ({ ...held, member_role: null }));
  expect(empty.value).toBeUndefined();
  expect(await empty.fetchFirst()).toBeUndefined();
  expect(empty.value).toBeUndefined();
  app.projectRoot.project.confirmProject(project, (held) => ({ ...held, member_role: 15 }));
  await empty.fetchFirst();
  expect(empty.value).toEqual({ total: 0, groups: new Map() });
  expect(root.readSerial).toBe(serial);
  expect(nerve.calls).toHaveLength(1);
  const read = await requestedIssue(
    nerve,
    () => issues.fetchIssues("ws", "A", "init-loader", options("2026-11-01..2026-11-30")),
    ["GET", "/api/v0/projects/A/issues"]
  );
  expect(read.call.queryEntries.filter(([key]) => key === "target_date")).toEqual([["target_date", "2026-11-15"]]);
  const row = { ...issueDetailOf("november"), target_date: "2026-11-15" };
  read.call.answer(json(200, { total: 1, groups: [{ key: "2026-11-15", total: 1, data: [row], next_cursor: null }] }));
  await read.promise;
  expect(issues.getIssueIds("2026-11-15")).toEqual(["november"]);
  await issues.fetchIssues("ws", "A", "init-loader", options("2026-10-01..2026-10-31"));
  expect(issues.currentList).toBe(empty);
  expect(issues.total).toBe(0);
  expect(nerve.calls).toHaveLength(2);
  expect(root.projectIssuesFilter.getIssueFilters("A")?.richFilters).toEqual({
    and: [{ target_date__exact: "2026-11-15" }],
  });
  root.retire();
  await expect(empty.fetchFirst()).rejects.toThrow();
});

it.each([null, 200, 403] as const)(
  "logical empty skips catalog while ordinary zero list preserves directory handling: %s",
  async (directoryStatus) => {
    const root = app.issue;
    const issues = root.projectIssues;
    runInAction(() => {
      root.projectIssuesFilter.filters.A.richFilters = { and: [{ target_date__exact: "2026-11-15" }] };
    });
    const query = { group_by: "target_date" as const, order_by: "label_name" as const };
    await issues.fetchIssues("ws", "A", "init-loader", {
      limit: 4,
      query: { ...query, target_date: ["2026-10-01..2026-10-31"] },
    });
    const empty = issues.currentList!;
    expect(empty.scope).toMatchObject({ empty: true });
    expect(empty.value).toEqual({ total: 0, groups: new Map() });
    expect(nerve.calls).toHaveLength(0);
    if (directoryStatus !== null) {
      await answered(
        nerve,
        () =>
          issues.fetchIssues("ws", "A", "init-loader", {
            limit: 4,
            query: { ...query, target_date: ["2026-11-01..2026-11-30"] },
          }),
        ["GET", "/api/v0/projects/A/issues"],
        { total: 0, groups: [] },
        "ordinary empty server result"
      );
      expect(issues.currentList!.scope).toMatchObject({ empty: false });
      expect(issues.total).toBe(0);
    }
    const ordinary = directoryStatus === null ? undefined : issues.currentList!;
    const serial = root.readSerial;
    const baseline = nerve.calls.length;
    const created = await requestedIssue(
      nerve,
      () => issues.createIssue("ws", "A", { name: "outside", target_date: "2026-11-15", label_ids: ["new-label"] }),
      ["POST", "/api/v0/projects/A/issues"]
    );
    created.call.answer(
      json(201, { ...issueDetailOf("outside"), target_date: "2026-11-15", label_ids: ["new-label"] })
    );
    await until(() => created.result.settled || nerve.calls.length > baseline + 1, "write or catalog request");
    const initialReads = nerve.calls.slice(baseline).filter((call) => call.method === "GET");
    if (ordinary) {
      expect.soft(created.result.settled).toBe(false);
      expect.soft(initialReads.map((call) => call.path)).toEqual(["/api/v0/projects/A/labels"]);
    } else {
      expect.soft(created.result.settled).toBe(true);
      expect.soft(initialReads).toEqual([]);
    }
    // Explicitly settle even unwanted pre-fix requests; timeouts are not this regression's oracle.
    const replied = new Set<(typeof nerve.calls)[number]>();
    const answerCatalog = async () => {
      if (created.result.settled) return;
      const reads = nerve.calls.slice(baseline).filter((call) => call.method === "GET" && !replied.has(call));
      for (const call of reads) {
        replied.add(call);
        call.answer(
          directoryStatus === 200
            ? json(200, { data: [labelOf(project, "New label", 1, { id: "new-label" })] })
            : problem(403, "permission_denied")
        );
      }
      await until(
        () =>
          created.result.settled ||
          nerve.calls.slice(baseline).some((call) => call.method === "GET" && !replied.has(call)),
        "write after explicit directory answer"
      );
    };
    // At most two registered bindings can issue a directory request in this scenario.
    await answerCatalog();
    await answerCatalog();
    expect(created.result.settled).toBe(true);
    await created.promise;
    expect.soft(root.issues.getIssueById("outside")?.label_ids).toEqual(["new-label"]);
    expect.soft(root.readSerial).toBe(serial);
    expect.soft(empty.value).toEqual({ total: 0, groups: new Map() });
    expect.soft(empty.error).toBeUndefined();
    expect
      .soft(nerve.calls.slice(baseline).map(({ method, path }) => [method, path]))
      .toEqual([["POST", "/api/v0/projects/A/issues"], ...(ordinary ? [["GET", "/api/v0/projects/A/labels"]] : [])]);
    if (ordinary && directoryStatus === 403) {
      expect.soft(ordinary.error).toMatchObject({ status: 403, message: "permission_denied" });
      expect.soft(ordinary.value?.total).toBe(0);
    } else if (ordinary) {
      expect.soft(ordinary.error).toBeUndefined();
      expect.soft(ordinary.value?.total).toBe(1);
      expect.soft(ordinary.value?.groups.get("2026-11-15")?.rows[0]).toMatchObject({ id: "outside", key: "new label" });
    }
  }
);
