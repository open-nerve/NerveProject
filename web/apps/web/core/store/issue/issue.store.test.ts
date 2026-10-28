/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueDetailOf, issueOf, issuePageOf, issueStoreTab, requestedIssue } from "./fake-issues";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { IssueListBinding } from "./helpers/issue-list-binding";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({
  rootStore: { projectRoot: { project: { getProjectIdentifierById: () => "A" } } },
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});
const version = (n: number) => `2026-10-28T00:00:00.00000000${n}Z`;
const accept = (issue: ReturnType<typeof issueOf>) =>
  tab.app.issue.acceptListIssues([issue], tab.app.issue.captureRead());
const list = (limit = 10) =>
  new IssueListBinding(tab.app.issue, {
    scope: "project",
    workspaceSlug: "ws",
    workspaceID: "id-ws",
    projectID: "A",
    query: { group_by: "state_id", order_by: "sort_order", limit },
  });
const load = async (binding: IssueListBinding, page: unknown) => {
  const read = await requestedIssue(tab.nerve, () => binding.fetchFirst(), ["GET", "/api/v0/projects/A/issues"]);
  read.call.answer(json(200, page));
  return read.promise;
};

it("latestServerVersionWinsAcrossAnswerOrder", () => {
  accept({ ...issueOf("x"), name: "new", updated_at: version(2) });
  accept({ ...issueOf("x"), name: "old", updated_at: version(1) });
  expect(tab.app.issue.issues.getIssueById("x")?.name).toBe("new");
});
it("listKeepsConfirmedDescription", () => {
  accept({ ...issueDetailOf("x"), updated_at: version(1) });
  accept({ ...issueOf("x"), name: "new", updated_at: version(2) });
  expect(tab.app.issue.issues.getIssueDetailById("x")?.description_html).toBe("<p>Server detail</p>");
  expect(tab.app.issue.issues.getIssueById("x")?.name).toBe("new");
});
it("staleFirstDetailDoesNotFillDescription", () => {
  accept({ ...issueOf("x"), name: "new", updated_at: version(2) });
  accept({ ...issueDetailOf("x"), name: "old", updated_at: version(1) });
  expect(tab.app.issue.issues.getIssueDetailById("x")).toBeUndefined();
  expect(tab.app.issue.issues.getIssueById("x")?.name).toBe("new");
});
it("foreignRootReadTokenIsRejected", async () => {
  const other = await issueStoreTab();
  tab.app.issue.acceptListIssues([issueDetailOf("x")], other.app.issue.captureRead());
  expect(tab.app.issue.issues.getIssueById("x")).toBeUndefined();
  other.app.dispose();
  accept(issueDetailOf("x"));
  expect(tab.app.issue.issues.getIssueById("x")?.id).toBe("x");
});
it("confirmedDeleteCannotBeResurrected", async () => {
  const binding = list();
  await load(binding, issuePageOf([issueOf("x")]));
  const reading = await requestedIssue(tab.nerve, () => binding.fetchFirst(), ["GET", "/api/v0/projects/A/issues"]);
  const oldToken = tab.app.issue.captureRead();
  const deleting = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.deleteIssue("x", tab.app.issue.scope("ws", "A")),
    ["DELETE", "/api/v0/issues/x"]
  );
  deleting.call.answer(noContent());
  await deleting.promise;
  reading.call.answer(json(200, issuePageOf([issueOf("x")])));
  await reading.promise;
  tab.app.issue.acceptListIssues([issueDetailOf("x")], oldToken);
  accept({ ...issueDetailOf("x"), updated_at: version(9) });
  expect(tab.app.issue.issues.getIssueById("x")).toBeUndefined();
  expect(binding.group()?.rows).toEqual([]);
  expect(binding.group()?.total).toBe(0);
  await load(binding, issuePageOf([issueOf("x")]));
  expect(binding.group()?.rows).toEqual([]);
  expect(binding.group()?.total).toBe(0);
});

// A page received while PATCH is out proves absence at the old position, but excludes the new position.
async function inferredMove() {
  const first = list();
  const second = list(20);
  const before = { ...issueOf("x"), state_id: "A", sort_order: 5 };
  accept(before);
  const writing = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.updateIssue("x", () => ({ state_id: "B", sort_order: 20 })),
    ["PATCH", "/api/v0/issues/x"]
  );
  const page = {
    total: 10,
    groups: [
      { key: "A", total: 4, data: [], next_cursor: null },
      { key: "B", total: 6, data: [{ ...issueOf("b"), state_id: "B", sort_order: 10 }], next_cursor: "next" },
    ],
  };
  await load(first, page);
  await load(second, page);
  const after = { ...issueDetailOf("x"), state_id: "B", sort_order: 20, updated_at: version(2) };
  writing.call.answer(json(200, after));
  await writing.promise;
  await vi.advanceTimersByTimeAsync(1);
  return { first, second, after };
}
it("directConfirmImmediatelyVerifiesAcrossCopiesWithoutResendingWrite", async () => {
  const { first, second, after } = await inferredMove();
  const gets = tab.nerve.calls.filter((call) => call.path === "/api/v0/issues/x" && call.method === "GET");
  expect(gets).toHaveLength(1);
  expect(tab.app.issue.issues.getIssueById("x")).toBeUndefined();
  gets[0]!.answer(problem(403, "forbidden"));
  await until(() => !!first.error && !!second.error, "verification refusal");
  expect(tab.nerve.calls.filter((call) => call.method === "PATCH")).toHaveLength(1);
  const retry = first.retryFailed();
  await until(
    () => tab.nerve.calls.filter((call) => call.path === "/api/v0/issues/x" && call.method === "GET").length === 2,
    "explicit retry GET"
  );
  tab.nerve.calls.at(-1)!.answer(json(200, after));
  await vi.advanceTimersByTimeAsync(1);
  await retry;
  expect(tab.app.issue.issues.getIssueDetailById("x")?.state_id).toBe("B");
  expect(first.error).toBeUndefined();
  expect(second.error).toBeUndefined();
  expect(tab.nerve.calls.filter((call) => call.method === "PATCH")).toHaveLength(1);
});

it.each([403, 404])("explicitRetryStartsNewRoundWhileOldWholeRemainsPending (%s)", async (status) => {
  const first = list();
  const second = list(20);
  await load(first, issuePageOf([issueOf("x")]));
  first.setVisibleParts([{ group: "" }]);
  const failed = await requestedIssue(tab.nerve, () => tab.app.issue.issues.refreshAuthoritative("x"), [
    "GET",
    "/api/v0/issues/x",
  ]);
  // Join the verification's own round, so it remains current even after its GET has settled.
  const slow = await requestedIssue(tab.nerve, () => second.fetchFirst(), ["GET", "/api/v0/projects/A/issues"]);
  const failure = track(failed.promise);
  failed.call.answer(problem(status, "refused"));
  await vi.advanceTimersByTimeAsync(1);
  if (status === 404) {
    tab.nerve.calls.at(-1)!.answer(json(200, issuePageOf([])));
    await failed.promise;
  } else expect(failure.error).toMatchObject({ status });
  const serialBeforeRetry = tab.app.issue.readSerial;
  const countBeforeRetry = tab.nerve.calls.length;
  const retry = track(first.retryFailed());
  await vi.advanceTimersByTimeAsync(1);
  expect(tab.nerve.calls).toHaveLength(countBeforeRetry + 1);
  expect(tab.nerve.calls.at(-1)).toMatchObject({ method: "GET", path: "/api/v0/issues/x" });
  expect(tab.app.issue.readSerial).toBeGreaterThan(serialBeforeRetry);
  expect(retry.settled).toBe(false);
  const fresh = tab.nerve.calls.at(-1)!;
  // A concurrent explicit caller shares the actual pending GET, but joins the new round's dedupe set.
  const shared = track(first.retryFailed());
  await vi.advanceTimersByTimeAsync(1);
  expect(tab.nerve.calls).toHaveLength(countBeforeRetry + 1);
  const third = list(30);
  const holdingNew = await requestedIssue(tab.nerve, () => third.fetchFirst(), ["GET", "/api/v0/projects/A/issues"]);
  slow.call.answer(json(200, issuePageOf([])));
  await slow.promise;
  fresh.answer(json(200, issueDetailOf("x")));
  await until(() => tab.nerve.calls.length === countBeforeRetry + 3, "fresh round scope budget");
  tab.nerve.calls.at(-1)!.answer(json(200, issuePageOf([issueOf("x")])));
  await until(() => retry.settled && shared.settled, "shared retry settles");
  expect(retry.error).toBeUndefined();
  expect(shared.error).toBeUndefined();
  expect(first.error).toBeUndefined();
  tab.app.issue.verification.require("x", first);
  await vi.advanceTimersByTimeAsync(1);
  expect(tab.nerve.calls).toHaveLength(countBeforeRetry + 3);
  holdingNew.call.answer(json(200, issuePageOf([])));
  await holdingNew.promise;
  expect(tab.app.issue.issues.getIssueDetailById("x")?.id).toBe("x");
});

it.each([404, 403, 500])(
  "verification %s rejects all already issued evidence, including equal-version late pages",
  async (status) => {
    const { first, second, after } = await inferredMove();
    first.setVisibleParts([{ group: "A" }, { group: "B" }]);
    second.setVisibleParts([{ group: "A" }, { group: "B" }]);
    const verify = tab.nerve.calls.find((call) => call.path === "/api/v0/issues/x" && call.method === "GET")!;
    const old = await requestedIssue(tab.nerve, () => first.fetchFirst(), ["GET", "/api/v0/projects/A/issues"]);
    const token = tab.app.issue.captureRead();
    verify.answer(problem(status, "refused"));
    await vi.advanceTimersByTimeAsync(1);
    tab.app.issue.acceptListIssues([after], token);
    expect(tab.app.issue.issues.getIssueById("x")).toBeUndefined();
    old.call.answer(json(200, issuePageOf([after])));
    await old.promise;
    if (status === 404) {
      const refreshes = tab.nerve.calls.filter((call) => call.path === "/api/v0/projects/A/issues").slice(3);
      expect(refreshes).toHaveLength(2);
      refreshes.forEach((call) => call.answer(json(200, issuePageOf([]))));
      await vi.advanceTimersByTimeAsync(1);
      expect(first.group()?.rows).toEqual([]);
      expect(first.group()?.total).toBe(0);
      expect(tab.app.issue.issues.getIssueById("x")).toBeUndefined();
    } else {
      expect(first.error).toBeDefined();
      expect(first.group("B")?.stale).toBe(true);
    }
    expect(tab.nerve.calls.filter((call) => call.path === "/api/v0/issues/x" && call.method === "GET")).toHaveLength(1);
    accept(after);
    expect(tab.app.issue.issues.getIssueDetailById("x")?.id).toBe("x");
  }
);

it("pendingInferenceMarksCountsUnconfirmedAndRetirementStopsFollowups", async () => {
  const { first, after } = await inferredMove();
  expect(first.group("B")?.stale).toBe(true);
  const count = tab.nerve.calls.length;
  tab.app.dispose();
  tab.nerve.calls.find((call) => call.path === "/api/v0/issues/x" && call.method === "GET")!.answer(json(200, after));
  await vi.advanceTimersByTimeAsync(1);
  expect(tab.nerve.calls).toHaveLength(count);
  expect(first.error).toBeUndefined();
  expect(() => tab.app.issue.captureRead()).toThrow();
});

it("thirdPartyMembershipMarksOldAndNewGroups", async () => {
  const binding = list();
  const before = { ...issueOf("x"), state_id: "A" };
  await load(binding, {
    total: 1,
    groups: [
      { key: "A", total: 1, data: [before], next_cursor: null },
      { key: "B", total: 0, data: [], next_cursor: null },
    ],
  });
  accept({ ...before, state_id: "B", updated_at: version(3) });
  expect(binding.group("A")?.stale).toBe(true);
  expect(binding.group("B")?.stale).toBe(true);
});

it("ownPendingReadDoesNotMarkStale", async () => {
  const binding = list();
  const before = { ...issueOf("x"), state_id: "A" };
  await load(binding, {
    total: 1,
    groups: [
      { key: "A", total: 1, data: [before], next_cursor: null },
      { key: "B", total: 0, data: [], next_cursor: null },
    ],
  });
  const writing = await requestedIssue(tab.nerve, () => tab.app.issue.updateIssue("x", () => ({ state_id: "B" })), [
    "PATCH",
    "/api/v0/issues/x",
  ]);
  const after = { ...issueDetailOf("x"), state_id: "B", updated_at: version(3) };
  accept(after);
  expect(binding.group("A")?.stale).toBe(false);
  expect(binding.group("B")?.stale).toBe(false);
  writing.call.answer(json(200, after));
  await writing.promise;
  expect(binding.group("A")?.total).toBe(0);
  expect(binding.group("B")?.total).toBe(1);
});

it("unknownDescendantsRefreshOnlyTheirProject", async () => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
  const a = list();
  const b = new IssueListBinding(tab.app.issue, {
    scope: "project",
    workspaceID: "id-ws",
    workspaceSlug: "ws",
    projectID: "B",
    query: {},
  });
  await load(a, issuePageOf([issueOf("parent")]));
  const readingB = await requestedIssue(tab.nerve, () => b.fetchFirst(), ["GET", "/api/v0/projects/B/issues"]);
  const token = tab.app.issue.captureRead();
  const deleting = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.deleteIssue("parent", tab.app.issue.scope("ws", "A")),
    ["DELETE", "/api/v0/issues/parent"]
  );
  deleting.call.answer(noContent());
  await deleting.promise;
  tab.app.issue.acceptListIssues([{ ...issueOf("unknown"), parent_id: "parent" }], token);
  readingB.call.answer(json(200, issuePageOf([issueOf("other", "B")])));
  await readingB.promise;
  expect(tab.app.issue.issues.getIssueById("unknown")).toBeUndefined();
  expect(a.group()?.stale).toBe(true);
  expect(b.group()?.stale).toBe(false);
  expect(b.group()?.total).toBe(1);
});

it("publicProjectWithoutMembershipCannotExposeCachedIssue", async () => {
  accept(issueDetailOf("x"));
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    { ...tab.project, member_role: null, network: 2 },
  ]);
  expect(tab.app.projectRoot.project.getProjectById("A")).toBeDefined();
  expect(tab.app.issue.issues.getIssueById("x")).toBeUndefined();
  expect(tab.app.issue.issues.issuesMap.x).toBeUndefined();
});

it("hiddenVerificationScopeWaitsForDisplay", async () => {
  const { first, after } = await inferredMove();
  const count = tab.nerve.calls.length;
  tab.nerve.calls.at(-1)!.answer(json(200, after));
  await vi.advanceTimersByTimeAsync(1);
  expect(tab.nerve.calls).toHaveLength(count);
  expect(first.group("B")?.stale).toBe(true);
});

it("parallelPartsShareOneVerificationRound", async () => {
  const binding = new IssueListBinding(tab.app.issue, {
    scope: "project",
    workspaceSlug: "ws",
    workspaceID: "id-ws",
    projectID: "A",
    query: { group_by: "label_id", order_by: "sort_order" },
  });
  const before = { ...issueOf("x"), label_ids: ["A", "B"], sort_order: 5 };
  await load(binding, {
    total: 1,
    groups: ["A", "B"].map((key) => ({ key, total: 1, data: [before], next_cursor: null })),
  });
  const writing = await requestedIssue(tab.nerve, () => tab.app.issue.updateIssue("x", () => ({ label_ids: ["C"] })), [
    "PATCH",
    "/api/v0/issues/x",
  ]);
  binding.markMembershipStale(before, before);
  const refreshing = binding.refreshStale([{ group: "A" }, { group: "B" }]);
  await until(() => tab.nerve.calls.length === 4, "parallel parts");
  writing.call.answer(json(200, { ...issueDetailOf("x"), label_ids: ["C"], updated_at: version(3) }));
  await writing.promise;
  tab.nerve.calls[2]!.answer(json(200, issuePageOf([])));
  tab.nerve.calls[3]!.answer(json(200, issuePageOf([])));
  await vi.advanceTimersByTimeAsync(1);
  const gets = tab.nerve.calls.filter((call) => call.method === "GET" && call.path === "/api/v0/issues/x");
  expect(gets).toHaveLength(1);
  const rejected = expect(refreshing).rejects.toMatchObject({ status: 500 });
  gets[0]!.answer(problem(500, "internal"));
  await rejected;
  await vi.advanceTimersByTimeAsync(1);
  expect(tab.nerve.calls).toHaveLength(5);
});

it("refusedDeleteDoesNotCreateTombstoneAndArchiveIsReversible", async () => {
  accept(issueDetailOf("x"));
  const scope = tab.app.issue.scope("ws", "A");
  const deleting = await requestedIssue(tab.nerve, () => tab.app.issue.deleteIssue("x", scope), [
    "DELETE",
    "/api/v0/issues/x",
  ]);
  const refused = expect(deleting.promise).rejects.toMatchObject({ status: 403 });
  deleting.call.answer(problem(403, "forbidden"));
  await refused;
  expect(tab.app.issue.issues.getIssueById("x")?.id).toBe("x");
  const archiving = await requestedIssue(tab.nerve, () => tab.app.issue.archiveIssue("x", scope), [
    "POST",
    "/api/v0/issues/x/archive",
  ]);
  archiving.call.answer(
    json(200, { ...issueDetailOf("x"), archived_at: "2026-10-29T00:00:00Z", updated_at: version(2) })
  );
  await archiving.promise;
  const restoring = await requestedIssue(tab.nerve, () => tab.app.issue.unarchiveIssue("x", scope), [
    "POST",
    "/api/v0/issues/x/unarchive",
  ]);
  restoring.call.answer(json(200, { ...issueDetailOf("x"), updated_at: version(3) }));
  await restoring.promise;
  expect(tab.app.issue.issues.getIssueById("x")?.archived_at).toBeNull();
});

it("treeDeleteDiscardsWholeProfilePageAndKnownDescendants", async () => {
  accept(issueOf("parent"));
  accept({ ...issueOf("child"), parent_id: "parent" });
  const binding = new IssueListBinding(tab.app.issue, {
    scope: "member",
    workspaceSlug: "ws",
    workspaceID: "id-ws",
    userID: "actor",
    query: { relation: "created" },
  });
  const reading = await requestedIssue(tab.nerve, () => binding.fetchFirst(), [
    "GET",
    "/api/v0/workspaces/ws/members/actor/issues",
  ]);
  const deleting = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.deleteIssue("parent", tab.app.issue.scope("ws", "A")),
    ["DELETE", "/api/v0/issues/parent"]
  );
  deleting.call.answer(noContent());
  await deleting.promise;
  reading.call.answer(json(200, issuePageOf([issueOf("unknown"), issueOf("other", "B")])));
  expect(await reading.promise).toBeUndefined();
  expect(binding.reconciled.value).toBeUndefined();
  accept(issueDetailOf("child"));
  expect(tab.app.issue.issues.getIssueById("child")).toBeUndefined();
  expect(tab.app.issue.issues.getIssueById("unknown")).toBeUndefined();
});

it("verifiedEntitySurvivesRefusedScopeRefresh", async () => {
  const { first, after } = await inferredMove();
  first.setVisibleParts([{ group: "B" }]);
  tab.nerve.calls.at(-1)!.answer(json(200, after));
  await until(() => tab.nerve.calls.length === 5, "one visible scope refresh");
  tab.nerve.calls.at(-1)!.answer(problem(500, "internal"));
  await until(() => !!first.error, "scope refusal");
  expect(tab.app.issue.issues.getIssueDetailById("x")).toEqual(after);
  expect(first.group("B")?.stale).toBe(true);
  await vi.advanceTimersByTimeAsync(1);
  expect(tab.nerve.calls).toHaveLength(5);
});

it("authoritativeFirstStaleDetailGetsOnlyOneFreshRetry", async () => {
  accept({ ...issueOf("x"), updated_at: version(3) });
  const reading = await requestedIssue(tab.nerve, () => tab.app.issue.issues.refreshAuthoritative("x"), [
    "GET",
    "/api/v0/issues/x",
  ]);
  reading.call.answer(json(200, { ...issueDetailOf("x"), updated_at: version(1) }));
  await vi.advanceTimersByTimeAsync(1);
  expect(tab.nerve.calls).toHaveLength(2);
  tab.nerve.calls[1]!.answer(json(200, { ...issueDetailOf("x"), updated_at: version(2) }));
  expect(await reading.promise).toBeUndefined();
  await vi.advanceTimersByTimeAsync(1);
  expect(tab.nerve.calls).toHaveLength(2);
  expect(tab.app.issue.issues.getIssueDetailById("x")).toBeUndefined();
  expect(tab.app.issue.verification.hasRefusal(list())).toBe(true);
});

it("confirmedDeleteEndsPendingVerificationAndOldGetCannotScheduleFollowup", async () => {
  const { first, after } = await inferredMove();
  first.setVisibleParts([{ group: "B" }]);
  const oldGet = tab.nerve.calls.at(-1)!;
  const deleting = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.deleteIssue("x", tab.app.issue.scope("ws", "A")),
    ["DELETE", "/api/v0/issues/x"]
  );
  deleting.call.answer(noContent());
  await deleting.promise;
  expect(tab.app.issue.verification.isUncertain("x")).toBe(false);
  oldGet.answer(json(200, after));
  await vi.advanceTimersByTimeAsync(1);
  expect(tab.nerve.calls).toHaveLength(5);
  expect(tab.app.issue.issues.getIssueById("x")).toBeUndefined();
});

it("SessionChangedDeleteDoesNotCreateTombstone", async () => {
  accept(issueDetailOf("x"));
  const deleting = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.deleteIssue("x", tab.app.issue.scope("ws", "A")),
    ["DELETE", "/api/v0/issues/x"]
  );
  tab.app.dispose();
  const refused = expect(deleting.promise).rejects.toBeDefined();
  deleting.call.answer(noContent());
  await refused;
  expect(tab.app.issue.issues.held("x")).not.toBeNull();
  expect(tab.nerve.calls).toHaveLength(1);
});
