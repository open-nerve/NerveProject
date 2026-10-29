/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue } from "@nerve/api-client";
import { IssueService } from "@/services/issue";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { runInAction, toJS } from "mobx";
import { issueDetailOf, issueOf, issuePageOf, issueStoreTab, requestedIssue } from "./fake-issues";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { IssueListBinding } from "./helpers/issue-list-binding";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
});
afterEach(() => {
  tab.app.dispose();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
const version = (n: number) => `2026-10-28T00:00:00.00000000${n}Z`;
const accept = (issue: Issue) => tab.app.issue.acceptListIssues([issue], tab.app.issue.captureRead());

it("postWriteEvidenceCannotReusePreWriteAuthoritativeGETWithoutFetchedLists", async () => {
  const root = tab.app.issue;
  accept({ ...issueDetailOf("x"), link_count: 5 });
  const old = await requestedIssue(tab.nerve, () => root.issues.refreshAuthoritative("x"), ["GET", "/api/v0/issues/x"]);
  const link = root.issueDetail.link;
  vi.spyOn(IssueService.prototype, "createIssueLink").mockResolvedValue({ id: "link" } as never);
  vi.spyOn(root.issueDetail.activity, "fetchActivities").mockResolvedValue([]);
  const write = track(link.createLink("ws", "A", "x", { title: "link" }));
  await vi.advanceTimersByTimeAsync(10);
  old.call.answer(json(200, { ...issueDetailOf("x"), link_count: 5 }));
  await vi.advanceTimersByTimeAsync(10);
  const reads = tab.nerve.to("/api/v0/issues/x");
  expect.soft(reads).toHaveLength(2);
  expect.soft(root.issues.getIssueById("x")).toBeUndefined();
  if (reads[1]) reads[1].answer(json(200, { ...issueDetailOf("x"), link_count: 6, updated_at: version(2) }));
  await until(() => write.settled, "post-write count");
  expect(write.error).toBeUndefined();
  expect(root.issues.getIssueById("x")?.link_count).toBe(6);
  await old.promise;
});

type Kind = "create" | "update" | "delete" | "archive" | "unarchive" | "drag";
const kinds: Kind[] = ["create", "update", "delete", "archive", "unarchive", "drag"];
const endpoint = (kind: Kind): ["GET" | "POST" | "PATCH" | "DELETE", string] => [
  kind === "delete" ? "DELETE" : kind === "update" || kind === "drag" ? "PATCH" : "POST",
  kind === "create"
    ? "/api/v0/projects/A/issues"
    : `/api/v0/issues/x${kind === "archive" || kind === "unarchive" ? "/" + kind : ""}`,
];
function operate(kind: Kind, source?: IssueListBinding): Promise<unknown> {
  const root = tab.app.issue;
  const scope = root.scope("ws", "A");
  if (kind === "create") return root.createIssue(scope, { name: "x" });
  if (kind === "delete") return root.deleteIssue("x", scope);
  if (kind === "archive") return root.archiveIssue("x", scope);
  if (kind === "unarchive") return root.unarchiveIssue("x", scope);
  return root.updateIssue(
    "x",
    () => ({ name: "changed", sort_order: 500 }),
    kind === "drag" ? { group: "", binding: source } : undefined
  );
}
const endpointOf = (binding: IssueListBinding): ["GET", string] => [
  "GET",
  binding.scope.scope === "member"
    ? "/api/v0/workspaces/ws/members/actor/issues"
    : `/api/v0/projects/${binding.scope.projectID}/issues`,
];
const bindingOf = (kind: "project" | "archived" | "profile", group = false, projectID = "A") =>
  new IssueListBinding(
    tab.app.issue,
    kind === "profile"
      ? {
          scope: "member",
          workspaceSlug: "ws",
          workspaceID: "id-ws",
          userID: "actor",
          query: { relation: "created", order_by: "sort_order", ...(group ? { group_by: "state_id" } : {}) },
        }
      : {
          scope: "project",
          workspaceSlug: "ws",
          workspaceID: "id-ws",
          projectID,
          query: {
            order_by: "sort_order",
            ...(kind === "archived" ? { archived: "true" } : {}),
            ...(group ? { group_by: "state_id" } : {}),
          },
        }
  );
async function load(binding: IssueListBinding, page: unknown) {
  const read = await requestedIssue(tab.nerve, () => binding.fetchFirst(), endpointOf(binding));
  read.call.answer(json(200, page));
  await read.promise;
}
async function copies(kind: Kind) {
  const before = { ...issueDetailOf("x"), archived_at: kind === "unarchive" ? version(1) : null };
  if (kind !== "create") accept(before);
  const lists = [bindingOf("project"), bindingOf("archived"), bindingOf("profile")];
  for (const [i, binding] of lists.entries()) {
    const present = kind !== "create" && (i === 1 ? !!before.archived_at : !before.archived_at);
    // oxlint-disable-next-line no-await-in-loop
    await load(binding, issuePageOf(present ? [before] : []));
  }
  runInAction(() => {
    tab.app.issue.projectIssues.currentList = lists[0];
    tab.app.issue.archivedIssues.currentList = lists[1];
    tab.app.issue.profileIssues.currentList = lists[2];
  });
  const unfetched = [bindingOf("project"), bindingOf("archived"), bindingOf("profile")];
  const confirm = lists.map((binding) => vi.spyOn(binding, "acceptConfirmed"));
  const canonical = vi.spyOn(tab.app.issue.issues, "confirmChange");
  const after = {
    ...issueDetailOf("x"),
    name: "changed",
    sort_order: 500,
    updated_at: version(2),
    archived_at: kind === "archive" ? version(2) : null,
  };
  return { before, after, lists, unfetched, confirm, canonical };
}
function assertRows(binding: IssueListBinding, issue: Issue | undefined) {
  const root = tab.app.issue;
  const facade = [root.projectIssues, root.archivedIssues, root.profileIssues].find(
    (store) => store.currentList === binding
  );
  if (facade) expect(facade.getIssueIds()).toEqual(issue ? [issue.id] : []);
  expect(binding.value?.total).toBe(issue ? 1 : 0);
  expect(binding.group()?.total).toBe(issue ? 1 : 0);
  expect(binding.group()?.rows).toEqual(
    issue ? [{ id: issue.id, key: issue.sort_order, updatedAt: issue.updated_at }] : []
  );
}
it.each(kinds)("sixWriteCopies %s confirms each fetched copy and canonical exactly once", async (kind) => {
  const { before, after, lists, unfetched, confirm, canonical } = await copies(kind);
  const snapshots = lists.map((list) => structuredClone(toJS(list.value)));
  const request = await requestedIssue(tab.nerve, () => operate(kind, lists[0]), endpoint(kind));
  expect(lists.map((list) => toJS(list.value))).toEqual(snapshots);
  expect(tab.app.issue.issues.getIssueById("x")).toEqual(kind === "create" ? undefined : before);
  expect(canonical).not.toHaveBeenCalled();
  request.call.answer(kind === "delete" ? noContent() : json(kind === "create" ? 201 : 200, after));
  await request.promise;
  for (const [i, list] of lists.entries()) {
    assertRows(list, kind === "delete" || (i === 1 ? !after.archived_at : !!after.archived_at) ? undefined : after);
    expect(confirm[i]).toHaveBeenCalledTimes(1);
  }
  expect(canonical).toHaveBeenCalledTimes(1);
  expect(tab.app.issue.issues.held("x")).toEqual(kind === "delete" ? null : after);
  expect(unfetched.map((list) => list.value)).toEqual([undefined, undefined, undefined]);
  expect(tab.nerve.calls).toHaveLength(4);
  expect(tab.app.issue.pendingIDs.size).toBe(0);
});
it.each(kinds.flatMap((kind) => (["refused", "retired"] as const).map((outcome) => ({ kind, outcome }))))(
  "sixWriteCopies $kind $outcome preserves every copy and deletion epoch",
  async ({ kind, outcome }) => {
    const { before, after, lists, unfetched, confirm, canonical } = await copies(kind);
    const snapshots = lists.map((list) => structuredClone(toJS(list.value)));
    const token = tab.app.issue.captureRead();
    const request = await requestedIssue(tab.nerve, () => operate(kind, lists[0]), endpoint(kind));
    if (outcome === "retired") tab.app.issue.retire();
    request.call.answer(
      outcome === "refused" ? problem(403, "forbidden") : kind === "delete" ? noContent() : json(200, after)
    );
    await until(() => request.result.settled, "refused/retired write");
    expect(request.result.error).toBeDefined();
    expect(lists.map((list) => toJS(list.value))).toEqual(snapshots);
    expect(confirm.every((spy) => spy.mock.calls.length === 0)).toBe(true);
    expect(canonical).not.toHaveBeenCalled();
    expect(tab.app.issue.issues.held("x")).toEqual(kind === "create" ? undefined : before);
    expect(tab.app.issue.readSerial).toBe(token.serial);
    expect(unfetched.map((list) => list.value)).toEqual([undefined, undefined, undefined]);
    expect(tab.nerve.calls).toHaveLength(4);
  }
);
it.each(kinds)("sixWriteCopies %s late raw reads cannot undo confirmation", async (kind) => {
  const { before, after, lists, unfetched, canonical } = await copies(kind);
  const reads = [];
  for (const list of lists) {
    // oxlint-disable-next-line no-await-in-loop
    reads.push(await requestedIssue(tab.nerve, () => list.fetchFirst(), endpointOf(list)));
  }
  const request = await requestedIssue(tab.nerve, () => operate(kind, lists[0]), endpoint(kind));
  request.call.answer(kind === "delete" ? noContent() : json(200, after));
  await request.promise;
  for (const [i, read] of reads.entries()) {
    const present = kind !== "create" && (i === 1 ? !!before.archived_at : !before.archived_at);
    read.call.answer(json(200, issuePageOf(present ? [before] : [])));
  }
  await vi.advanceTimersByTimeAsync(10);
  const verifications = tab.nerve.calls.filter((call) => call.method === "GET" && call.path === "/api/v0/issues/x");
  expect(verifications).toHaveLength(["create", "archive", "unarchive"].includes(kind) ? 1 : 0);
  verifications.forEach((call) => call.answer(json(200, after)));
  await Promise.all(reads.map((read) => read.promise));
  for (const [i, list] of lists.entries())
    assertRows(list, kind === "delete" || (i === 1 ? !after.archived_at : !!after.archived_at) ? undefined : after);
  expect(canonical).toHaveBeenCalledTimes(1);
  expect(tab.app.issue.issues.held("x")).toEqual(kind === "delete" ? null : after);
  expect(unfetched.every((list) => list.value === undefined)).toBe(true);
  expect(tab.nerve.calls).toHaveLength(7 + verifications.length);
});
const grouped = (totals = [5, 4]) => ({
  total: 9,
  groups: [
    {
      key: "A",
      total: totals[0],
      data: [{ ...issueOf("edge-a"), state_id: "A", sort_order: 10 }],
      next_cursor: "a-next",
    },
    {
      key: "B",
      total: totals[1],
      data: [{ ...issueOf("edge-b"), state_id: "B", sort_order: 10 }],
      next_cursor: "b-next",
    },
  ],
});
it.each(["own", "third-party"] as const)(
  "ownReadBeforeWriteAnswerKeepsOffWindowBeforeMembership: %s",
  async (readKind) => {
    const before = { ...issueOf("x"), state_id: "A", sort_order: 50, label_ids: ["old"] };
    accept(before);
    const lists = [bindingOf("project", true), bindingOf("profile", true)];
    for (const list of lists) {
      // oxlint-disable-next-line no-await-in-loop
      await load(list, grouped());
    }
    const original = tab.app.issue.issues.getIssueById("x")!;
    const confirmation = vi.spyOn(lists[0]!, "acceptConfirmed");
    const request = await requestedIssue(
      tab.nerve,
      () => tab.app.issue.updateIssue("x", () => ({ state_id: "B", label_ids: ["new"] })),
      ["PATCH", "/api/v0/issues/x"]
    );
    const after = { ...issueDetailOf("x"), state_id: "B", sort_order: 50, label_ids: ["new"], updated_at: version(3) };
    // Same write's read arrives before PATCH; the immutable pending snapshot must still describe A.
    const pendingRead = await requestedIssue(
      tab.nerve,
      () => {
        const root = tab.app.issue;
        const token = root.captureRead();
        return root.api.get("x").then((answer) => root.issues.acceptRead(answer, token));
      },
      ["GET", "/api/v0/issues/x"]
    );
    pendingRead.call.answer(
      json(200, readKind === "own" ? after : { ...after, state_id: "C", updated_at: version(2) })
    );
    await pendingRead.promise;
    request.call.answer(json(200, after));
    await request.promise;
    const frozen = confirmation.mock.calls[0]![0].beforeSnapshot;
    expect(frozen.kind).toBe(readKind === "own" ? "known" : "unknown");
    if (frozen.kind === "known") {
      expect(frozen.issue).not.toBe(original);
      expect(frozen.issue.label_ids).not.toBe(original.label_ids);
      expect(frozen.issue.label_ids).toEqual(["old"]);
    }
    for (const list of lists) {
      expect(list.value?.total).toBe(9);
      if (readKind === "own") {
        expect([list.group("A")?.total, list.group("B")?.total]).toEqual([4, 5]);
        expect([list.group("A")?.stale, list.group("B")?.stale]).toEqual([false, false]);
      } else expect(list.group("B")?.stale).toBe(true);
      expect(list.group("A")?.rows.map((row) => row.id)).toEqual(["edge-a"]);
      expect(list.group("B")?.rows.map((row) => row.id)).toEqual(["edge-b"]);
    }
    expect(tab.nerve.calls).toHaveLength(4);
  }
);
it("queuedBeforeIsFrozenOnlyWhenSentAndPreviousFinallyCannotClearPending", async () => {
  accept({ ...issueOf("x"), state_id: "A", sort_order: 50 });
  const list = bindingOf("project", true);
  await load(list, grouped());
  const first = await requestedIssue(tab.nerve, () => tab.app.issue.updateIssue("x", () => ({ state_id: "B" })), [
    "PATCH",
    "/api/v0/issues/x",
  ]);
  const second = track(tab.app.issue.updateIssue("x", () => ({ state_id: "A" })));
  first.call.answer(json(200, { ...issueDetailOf("x"), state_id: "B", sort_order: 50, updated_at: version(2) }));
  await until(() => tab.nerve.to("/api/v0/issues/x").length === 2, "second queued write");
  expect(tab.app.issue.pendingIDs.has("x")).toBe(true);
  expect([list.group("A")?.total, list.group("B")?.total]).toEqual([4, 5]);
  tab.nerve.calls
    .at(-1)!
    .answer(json(200, { ...issueDetailOf("x"), state_id: "A", sort_order: 50, updated_at: version(3) }));
  await until(() => second.settled, "second confirmation");
  expect(second.error).toBeUndefined();
  expect([list.group("A")?.total, list.group("B")?.total]).toEqual([5, 4]);
  expect(tab.app.issue.pendingIDs.size).toBe(0);
});

it.each(["whole", "part", "get", "verify"] as const)(
  "subtreeDeletionInvalidatesUnknownDescendantReplies: %s visible and hidden",
  async (kind) => {
    const root = tab.app.issue;
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
      tab.project,
      projectOf("B", tab.workspace.id, { id: "B" }),
    ]);
    const parent = issueOf("parent");
    const child = { ...issueOf("child"), parent_id: "parent" };
    const grand = { ...issueOf("grand"), parent_id: "child", archived_at: version(1) };
    const unknown = { ...issueDetailOf("Y"), parent_id: "grand", archived_at: version(1) };
    [parent, child, grand].forEach(accept);
    const lists = [bindingOf("project"), bindingOf("archived"), bindingOf("profile")];
    for (const [i, list] of lists.entries()) {
      // oxlint-disable-next-line no-await-in-loop
      await load(list, issuePageOf(i === 1 ? [grand] : [parent, child], "next"));
    }
    lists[0]!.setVisibleParts([{ group: "" }]);
    const other = bindingOf("project", false, "B");
    await load(other, issuePageOf([issueOf("other", "B")]));
    const stable = structuredClone(toJS(other.value));
    expect(root.issues.held("Y")).toBeUndefined();
    const pending = [];
    if (kind === "whole" || kind === "part") {
      for (const list of lists) {
        pending.push(
          // oxlint-disable-next-line no-await-in-loop
          await requestedIssue(
            tab.nerve,
            () => (kind === "whole" ? list.fetchFirst() : list.fetchNext()),
            endpointOf(list)
          )
        );
      }
    } else if (kind === "get") {
      pending.push(
        await requestedIssue(tab.nerve, () => root.issues.refreshAuthoritative("Y"), ["GET", "/api/v0/issues/Y"])
      );
    } else {
      root.verification.require("Y", lists[1]!);
      await until(() => tab.nerve.to("/api/v0/issues/Y").length === 1, "unknown verification");
    }
    const deleting = await requestedIssue(tab.nerve, () => root.deleteIssue("parent", root.scope("ws", "A")), [
      "DELETE",
      "/api/v0/issues/parent",
    ]);
    deleting.call.answer(noContent());
    await deleting.promise;
    for (const id of ["parent", "child", "grand"]) expect(root.issues.held(id)).toBeNull();
    const afterDelete = tab.nerve.calls.length;
    pending.forEach((read) => read.call.answer(json(200, kind === "get" ? unknown : issuePageOf([unknown]))));
    if (kind === "verify") tab.nerve.to("/api/v0/issues/Y")[0]!.answer(json(200, unknown));
    await Promise.all(pending.map((read) => read.promise.catch((error: unknown) => error)));
    await vi.advanceTimersByTimeAsync(10);
    expect(root.issues.held("Y")).toBeUndefined();
    for (const list of lists) {
      expect(list.group()?.rows).toEqual([]);
      expect(list.group()?.stale).toBe(true);
    }
    expect(other.value).toEqual(stable);
    expect(tab.nerve.calls).toHaveLength(afterDelete);
  }
);
it("twoDeletesInvalidateRefreshBetweenThemButKeepAnotherProjectCurrent", async () => {
  const root = tab.app.issue;
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
  const a = bindingOf("project");
  const b = bindingOf("project", false, "B");
  await load(a, issuePageOf([issueOf("first"), issueOf("second")]));
  await load(b, issuePageOf([issueOf("other", "B")]));
  const first = await requestedIssue(tab.nerve, () => root.deleteIssue("first", root.scope("ws", "A")), [
    "DELETE",
    "/api/v0/issues/first",
  ]);
  const second = await requestedIssue(tab.nerve, () => root.deleteIssue("second", root.scope("ws", "A")), [
    "DELETE",
    "/api/v0/issues/second",
  ]);
  first.call.answer(noContent());
  await first.promise;
  const middle = await requestedIssue(tab.nerve, () => a.fetchFirst(), endpointOf(a));
  const other = await requestedIssue(tab.nerve, () => b.fetchFirst(), endpointOf(b));
  const watermark = root.readSerial;
  second.call.answer(noContent());
  await second.promise;
  expect(root.readSerial).toBe(watermark + 1);
  middle.call.answer(json(200, issuePageOf([{ ...issueOf("Y"), parent_id: "second" }])));
  other.call.answer(json(200, issuePageOf([issueOf("other", "B")])));
  await Promise.all([middle.promise, other.promise]);
  expect(root.issues.held("Y")).toBeUndefined();
  expect(a.group()?.stale).toBe(true);
  expect(a.group()?.rows).toEqual([]);
  expect(b.group()?.stale).toBe(false);
  expect(b.group()?.total).toBe(1);
  await load(a, issuePageOf([]));
  expect(a.group()?.stale).toBe(false);
});
async function missingReplay() {
  const root = tab.app.issue;
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    tab.project,
    projectOf("B", tab.workspace.id, { id: "B" }),
  ]);
  const other = bindingOf("project", false, "B");
  await load(other, issuePageOf([issueOf("other", "B")]));

  accept({ ...issueOf("x"), state_id: "A", sort_order: 5 });
  const first = bindingOf("project", true);
  const second = bindingOf("profile", true);
  const archive = bindingOf("archived", true);
  const write = await requestedIssue(
    tab.nerve,
    () => root.updateIssue("x", () => ({ state_id: "B", sort_order: 20 })),
    ["PATCH", "/api/v0/issues/x"]
  );
  const page = {
    total: 10,
    groups: [
      { key: "A", total: 4, data: [], next_cursor: null },
      { key: "B", total: 6, data: [{ ...issueOf("edge"), state_id: "B", sort_order: 10 }], next_cursor: "next" },
    ],
  };
  await load(first, page);
  await load(second, page);
  await load(archive, issuePageOf([]));
  const after = { ...issueDetailOf("x"), state_id: "B", sort_order: 20, updated_at: version(2) };
  write.call.answer(json(200, after));
  await write.promise;
  await until(
    () => tab.nerve.to("/api/v0/issues/x").filter((call) => call.method === "GET").length === 1,
    "one replay verification"
  );
  expect(root.issues.held("x")).toEqual(after);
  expect(root.issues.getIssueById("x")).toBeUndefined();
  return { first, second, archive, other, after, verify: tab.nerve.calls.at(-1)! };
}
it.each([404, 403, 500, "retired"] as const)(
  "missingReplayVerificationLeavesUncertainGroupsStale %s",
  async (status) => {
    const { first, second, archive, other, after, verify } = await missingReplay();
    const root = tab.app.issue;
    const serial = root.readSerial;
    const token = root.captureRead();
    const archiveValue = structuredClone(toJS(archive.value));
    const otherValue = structuredClone(toJS(other.value));
    const at = tab.nerve.calls.length;
    if (status === "retired") root.retire();
    verify.answer(status === "retired" ? json(200, after) : problem(status, "refused"));
    await vi.advanceTimersByTimeAsync(10);
    expect(root.issues.getIssueById("x")).toBeUndefined();
    expect(root.issues.held("x")).not.toBeNull();
    expect(first.group("B")?.stale).toBe(true);
    expect(second.group("B")?.stale).toBe(true);
    expect(archive.value).toEqual(archiveValue);
    expect(other.value).toEqual(otherValue);
    expect(tab.nerve.calls).toHaveLength(at);
    expect(root.readSerial).toBe(serial + 1); // A 404 changes freshness, never deletion serial.
    if (status !== "retired") expect(root.isReadCurrent(token, ["A"], "id-ws")).toBe(true);
    if (status === 403 || status === 500) {
      expect(first.error).toMatchObject({ status });
      expect(second.error).toMatchObject({ status });
      root.verification.require("x", first);
      await vi.advanceTimersByTimeAsync(20);
      expect(tab.nerve.calls).toHaveLength(at);
    }
    if (status === 404) {
      await load(first, issuePageOf([]));
      await load(second, issuePageOf([]));
      expect(first.group()?.total).toBe(0);
      expect(second.group()?.stale).toBe(false);
      expect(root.issues.getIssueById("x")).toBeUndefined();
      expect(root.issues.held("x")).toEqual(after);
      expect(other.value).toEqual(otherValue);
    }
  }
);
it.each(["get", "list", "empty"] as const)(
  "preVerificationListCannotClearUnverifiedState: serial7/8/9/10 %s",
  async (recovery) => {
    const root = tab.app.issue;
    const after = { ...issueDetailOf("x"), updated_at: version(2) };
    const first = bindingOf("project");
    const second = bindingOf("profile");
    await load(first, issuePageOf([after]));
    await load(second, issuePageOf([after]));
    for (let i = root.readSerial; i < 6; i++) root.captureRead();
    const old = await requestedIssue(tab.nerve, () => first.fetchFirst(), endpointOf(first));
    expect(root.readSerial).toBe(7);
    const verify = await requestedIssue(tab.nerve, () => root.issues.refreshAuthoritative("x"), [
      "GET",
      "/api/v0/issues/x",
    ]);
    expect(root.readSerial).toBe(8);
    const during = await requestedIssue(tab.nerve, () => second.fetchFirst(), endpointOf(second));
    expect(root.readSerial).toBe(9);
    verify.call.answer(problem(404, "not_found"));
    await verify.promise;
    expect(root.readSerial).toBe(9);
    old.call.answer(json(200, issuePageOf([after])));
    during.call.answer(json(200, issuePageOf([after])));
    await Promise.all([old.promise.catch((error: unknown) => error), during.promise.catch((error: unknown) => error)]);
    expect(root.issues.getIssueById("x")).toBeUndefined();
    expect(first.group()?.stale).toBe(true);
    expect(second.group()?.stale).toBe(true);
    expect(root.issues.held("x")).not.toBeNull();
    if (recovery === "get") {
      const fresh = await requestedIssue(tab.nerve, () => root.issues.refreshAuthoritative("x"), [
        "GET",
        "/api/v0/issues/x",
      ]);
      expect(root.readSerial).toBe(10);
      fresh.call.answer(json(200, after));
      await fresh.promise;
    } else {
      const fresh = await requestedIssue(tab.nerve, () => first.fetchFirst(), endpointOf(first));
      expect(root.readSerial).toBe(10);
      fresh.call.answer(json(200, issuePageOf(recovery === "empty" ? [] : [after])));
      await fresh.promise;
      expect(first.group()?.stale).toBe(false);
      expect(first.group()?.total).toBe(recovery === "empty" ? 0 : 1);
    }
    expect(root.issues.getIssueById("x")).toEqual(recovery === "empty" ? undefined : after);
    expect(root.issues.held("x")).toEqual(after);
  }
);

it.each([200, 404, 403, 500])("R31 superseded GET %s cannot change a completed newer round", async (status) => {
  const root = tab.app.issue;
  accept({ ...issueDetailOf("x"), link_count: 5 });
  const old = await requestedIssue(tab.nerve, () => root.issues.refreshAuthoritative("x"), ["GET", "/api/v0/issues/x"]);
  const fresh = await requestedIssue(tab.nerve, () => root.issues.refreshAuthoritative("x", { afterWrite: true }), [
    "GET",
    "/api/v0/issues/x",
  ]);
  const after = { ...issueDetailOf("x"), link_count: 6, updated_at: version(2) };
  fresh.call.answer(json(200, after));
  await fresh.promise;
  old.call.answer(
    status === 200 ? json(200, { ...after, link_count: 5, updated_at: version(9) }) : problem(status, "old-refusal")
  );
  await old.promise;
  expect(root.issues.getIssueDetailById("x")).toEqual(after);
  expect(root.verification.isUncertain("x")).toBe(false);
  expect(tab.nerve.to("/api/v0/issues/x")).toHaveLength(2);
});
it.each([403, 404, 500])("R31 old success cannot clear newer refusal %s", async (status) => {
  const root = tab.app.issue;
  accept({ ...issueDetailOf("x"), link_count: 5 });
  const old = await requestedIssue(tab.nerve, () => root.issues.refreshAuthoritative("x"), ["GET", "/api/v0/issues/x"]);
  const fresh = await requestedIssue(tab.nerve, () => root.issues.refreshAuthoritative("x", { afterWrite: true }), [
    "GET",
    "/api/v0/issues/x",
  ]);
  fresh.call.answer(problem(status, "new-refusal"));
  await until(() => fresh.result.settled, "fresh refusal");
  old.call.answer(json(200, { ...issueDetailOf("x"), link_count: 5, updated_at: version(9) }));
  await old.promise;
  expect(root.issues.getIssueById("x")).toBeUndefined();
  expect(root.issues.held("x")?.updated_at).toBe(issueOf("x").updated_at);
  expect(tab.nerve.to("/api/v0/issues/x")).toHaveLength(2);
  expect(fresh.result.error).toEqual(status === 404 ? undefined : expect.objectContaining({ status }));
});
it("ordinary concurrent reads dedupe both pending and not-yet-issued GETs", async () => {
  const root = tab.app.issue;
  accept(issueDetailOf("x"));
  const first = root.issues.refreshAuthoritative("x");
  const second = root.issues.refreshAuthoritative("x");
  await until(() => tab.nerve.to("/api/v0/issues/x").length === 1, "one GET");
  const third = root.issues.refreshAuthoritative("x");
  await vi.advanceTimersByTimeAsync(10);
  expect(tab.nerve.to("/api/v0/issues/x")).toHaveLength(1);
  tab.nerve.calls.at(-1)!.answer(json(200, issueDetailOf("x")));
  expect(await Promise.all([first, second, third])).toEqual([
    issueDetailOf("x"),
    issueDetailOf("x"),
    issueDetailOf("x"),
  ]);
});
it("post-write callers before actual dispatch share one future evidence token", async () => {
  const root = tab.app.issue;
  accept(issueDetailOf("x"));
  const first = root.issues.refreshAuthoritative("x", { afterWrite: true });
  const second = root.issues.refreshAuthoritative("x", { afterWrite: true });
  await until(() => tab.nerve.to("/api/v0/issues/x").length === 1, "one future GET");
  tab.nerve.calls.at(-1)!.answer(json(200, issueDetailOf("x")));
  await Promise.all([first, second]);
  expect(tab.nerve.to("/api/v0/issues/x")).toHaveLength(1);
});
const resources = ["createLink", "removeLink", "createSubIssues", "removeSubIssue", "deleteSubIssue", "inbox"] as const;
async function resourceOperation(kind: (typeof resources)[number], refuse = false): Promise<() => Promise<unknown>> {
  const root = tab.app.issue;
  const detail = root.issueDetail;
  vi.spyOn(detail.activity, "fetchActivities").mockResolvedValue([]);
  const response = <T>(value: T) => (refuse ? Promise.reject(new Error("resource refused")) : Promise.resolve(value));
  if (kind === "createLink") {
    vi.spyOn(IssueService.prototype, "createIssueLink").mockImplementation(() => response({ id: "link" } as never));
    return () => detail.link.createLink("ws", "A", "x", { title: "link" });
  }
  if (kind === "removeLink") {
    detail.link.addLinks("x", [{ id: "link" } as never]);
    vi.spyOn(IssueService.prototype, "deleteIssueLink").mockImplementation(() => response(undefined));
    return () => detail.link.removeLink("ws", "A", "x", "link");
  }
  if (kind === "createSubIssues") {
    vi.spyOn(IssueService.prototype, "addSubIssues").mockImplementation(() =>
      response({
        sub_issues: [],
        state_distribution: { backlog: [], unstarted: [], started: [], completed: [], cancelled: [] },
      })
    );
    return () => detail.subIssues.createSubIssues("ws", "A", "x", []);
  }
  if (kind === "removeSubIssue" || kind === "deleteSubIssue") {
    accept({ ...issueOf("child"), parent_id: "x" });
    runInAction(() => {
      detail.subIssues.subIssues.x = ["child"];
    });
    return () =>
      kind === "removeSubIssue"
        ? detail.subIssues.removeSubIssue("ws", "A", "x", "child")
        : detail.subIssues.deleteSubIssue("ws", "A", "x", "child");
  }
  const { InboxIssueStore } = await import("@/store/inbox/inbox-issue.store");
  const { EInboxIssueStatus } = await import("@nerve/types");
  const inbox = new InboxIssueStore(
    "ws",
    "A",
    { issue: issueOf("x"), status: EInboxIssueStatus.PENDING } as never,
    tab.app
  );
  vi.spyOn(inbox.inboxIssueService, "update").mockImplementation(() =>
    response({ status: EInboxIssueStatus.ACCEPTED } as never)
  );
  return () => inbox.updateInboxIssueStatus(EInboxIssueStatus.ACCEPTED);
}
it.each(resources)("legacy %s success takes new evidence after a pre-write GET", async (kind) => {
  const root = tab.app.issue;
  accept({ ...issueDetailOf("x"), link_count: 5, sub_issues_count: 5 });
  const operateResource = await resourceOperation(kind);
  const old = await requestedIssue(tab.nerve, () => root.issues.refreshAuthoritative("x"), ["GET", "/api/v0/issues/x"]);
  const write = track(operateResource());
  await vi.advanceTimersByTimeAsync(10);
  if (kind === "removeSubIssue" || kind === "deleteSubIssue") {
    const call = tab.nerve.to("/api/v0/issues/child")[0]!;
    call.answer(kind === "deleteSubIssue" ? noContent() : json(200, { ...issueDetailOf("child"), parent_id: null }));
    await vi.advanceTimersByTimeAsync(10);
  }
  const reads = tab.nerve.to("/api/v0/issues/x");
  expect(reads).toHaveLength(2);
  old.call.answer(json(200, { ...issueDetailOf("x"), link_count: 5, sub_issues_count: 5 }));
  await old.promise;
  expect(root.issues.getIssueById("x")).toBeUndefined();
  reads[1]!.answer(json(200, { ...issueDetailOf("x"), link_count: 6, sub_issues_count: 6, updated_at: version(2) }));
  await until(() => write.settled, "resource confirmation");
  expect(write.error).toBeUndefined();
  expect(root.issues.getIssueById("x")).toMatchObject({ link_count: 6, sub_issues_count: 6, updated_at: version(2) });
});
it.each(resources)("legacy %s rejection does not start post-write evidence", async (kind) => {
  accept(issueDetailOf("x"));
  const operateResource = await resourceOperation(kind, true);
  const write = track(operateResource());
  await vi.advanceTimersByTimeAsync(10);
  if (kind === "removeSubIssue" || kind === "deleteSubIssue")
    tab.nerve.to("/api/v0/issues/child")[0]!.answer(problem(403, "forbidden"));
  await until(() => write.settled, "resource rejection");
  expect(tab.nerve.to("/api/v0/issues/x")).toHaveLength(0);
  expect(tab.app.issue.issues.getIssueById("x")).toEqual(issueDetailOf("x"));
});
it.each(resources)("retirement during %s post-write evidence stops legacy followups", async (kind) => {
  accept(issueDetailOf("x"));
  const operateResource = await resourceOperation(kind);
  const write = track(operateResource());
  await vi.advanceTimersByTimeAsync(10);
  if (kind === "removeSubIssue" || kind === "deleteSubIssue")
    tab.nerve
      .to("/api/v0/issues/child")[0]!
      .answer(kind === "deleteSubIssue" ? noContent() : json(200, issueDetailOf("child")));
  await until(() => tab.nerve.to("/api/v0/issues/x").length === 1, "post-write GET");
  const activity = tab.app.issue.issueDetail.activity.fetchActivities;
  tab.app.issue.retire();
  tab.nerve.calls.at(-1)!.answer(json(200, issueDetailOf("x")));
  await until(() => write.settled, "retired resource");
  expect(activity).not.toHaveBeenCalled();
  expect(write.error).toBeInstanceOf(SessionChangedError);
});

it("ordinary fetchSubIssues parent read shares the pre-existing GET", async () => {
  const root = tab.app.issue;
  accept(issueDetailOf("x"));
  const old = await requestedIssue(tab.nerve, () => root.issues.refreshAuthoritative("x"), ["GET", "/api/v0/issues/x"]);
  vi.spyOn(IssueService.prototype, "subIssues").mockResolvedValue({
    sub_issues: [],
    state_distribution: { backlog: [], unstarted: [], started: [], completed: [], cancelled: [] },
  });
  const read = track(root.issueDetail.subIssues.fetchSubIssues("ws", "A", "x"));
  await vi.advanceTimersByTimeAsync(10);
  expect(tab.nerve.to("/api/v0/issues/x")).toHaveLength(1);
  old.call.answer(json(200, issueDetailOf("x")));
  await until(() => read.settled, "ordinary subissue read");
  expect(read.error).toBeUndefined();
});
it.each(["whole", "part"] as const)(
  "permission identity isolates all copies and GET/verify from %s and other scope",
  async (kind) => {
    const root = tab.app.issue;
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
      tab.project,
      projectOf("B", tab.workspace.id, { id: "B" }),
    ]);
    const active = { ...issueOf("x"), updated_at: version(2) };
    const archived = { ...issueOf("archived"), archived_at: version(1), updated_at: version(2) };
    const lists = [bindingOf("project"), bindingOf("archived"), bindingOf("profile")];
    for (const [i, list] of lists.entries()) {
      // oxlint-disable-next-line no-await-in-loop
      await load(list, issuePageOf(i === 1 ? [archived] : [active], "next"));
    }
    const other = bindingOf("project", false, "B");
    await load(other, issuePageOf([issueOf("other", "B")]));
    const old = [];
    for (const list of lists) {
      old.push(
        // oxlint-disable-next-line no-await-in-loop
        await requestedIssue(
          tab.nerve,
          () => (kind === "whole" ? list.fetchFirst() : list.fetchNext()),
          endpointOf(list)
        )
      );
    }
    const get = await requestedIssue(tab.nerve, () => root.issues.refreshAuthoritative("x"), [
      "GET",
      "/api/v0/issues/x",
    ]);
    root.verification.require("x", lists[0]!); // Same identity: the real pending GET is shared.
    const otherRead = await requestedIssue(tab.nerve, () => other.fetchFirst(), endpointOf(other));
    const serial = root.readSerial;
    tab.app.projectRoot.project.confirmProject(tab.project, (project) => ({ ...project, member_role: null }));
    tab.app.projectRoot.project.confirmProject(tab.project, (project) => ({ ...project, member_role: 15 }));
    expect(root.readSerial).toBe(serial); // Permission observation does not masquerade as a delete/request token.
    expect(lists.every((list) => list.value === undefined)).toBe(true);
    expect(root.issues.getIssueById("x")).toBeUndefined();
    const fresh = await requestedIssue(tab.nerve, () => root.issues.refreshAuthoritative("x"), [
      "GET",
      "/api/v0/issues/x",
    ]);
    const recovered = { ...issueDetailOf("x"), name: "recovered", updated_at: version(1) };
    fresh.call.answer(json(200, recovered));
    await fresh.promise;
    get.call.answer(problem(404, "old-permission"));
    old.forEach((read, i) =>
      read.call.answer(json(200, issuePageOf([{ ...(i === 1 ? archived : active), updated_at: version(9) }])))
    );
    otherRead.call.answer(json(200, issuePageOf([{ ...issueOf("other", "B"), updated_at: version(3) }])));
    await Promise.all([
      get.promise,
      otherRead.promise,
      ...old.map((read) => read.promise.catch((error: unknown) => error)),
    ]);
    expect(root.issues.getIssueById("x")).toEqual(recovered);
    expect(lists.every((list) => list.value === undefined)).toBe(true);
    expect(other.group()?.stale).toBe(false);
    for (const [i, list] of lists.entries()) {
      // oxlint-disable-next-line no-await-in-loop
      await load(list, issuePageOf(i === 1 ? [archived] : [recovered]));
      expect(list.group()?.stale).toBe(false);
      expect(list.group()?.total).toBe(1);
      expect(list.group()?.rows[0]?.updatedAt).toBe(i === 1 ? version(2) : version(1));
    }
    expect(tab.nerve.to("/api/v0/issues/x")).toHaveLength(2);
  }
);
it("permission deletion and per-id freshness floors remain independent", async () => {
  const root = tab.app.issue;
  accept(issueDetailOf("x"));
  const beforePermission = root.captureRead();
  tab.app.projectRoot.project.confirmProject(tab.project, (project) => ({ ...project, member_role: null }));
  tab.app.projectRoot.project.confirmProject(tab.project, (project) => ({ ...project, member_role: 15 }));
  expect(root.readSerial).toBe(beforePermission.serial);
  expect(root.isReadCurrent(beforePermission, ["A"], "id-ws")).toBe(false);
  accept(issueDetailOf("x"));
  const beforeMissing = root.captureRead();
  const verify = await requestedIssue(tab.nerve, () => root.issues.refreshAuthoritative("x"), [
    "GET",
    "/api/v0/issues/x",
  ]);
  verify.call.answer(problem(404, "missing"));
  await verify.promise;
  expect(root.isReadCurrent(beforeMissing, ["A"], "id-ws")).toBe(true);
  expect(root.verification.canAccept("x", beforeMissing)).toBe(false);
  expect(root.verification.canAccept("other", beforeMissing)).toBe(true);
  const beforeDelete = root.captureRead();
  expect(root.verification.canAccept("x", beforeDelete)).toBe(true);
  const removal = await requestedIssue(tab.nerve, () => root.deleteIssue("unknown-parent", root.scope("ws", "A")), [
    "DELETE",
    "/api/v0/issues/unknown-parent",
  ]);
  removal.call.answer(noContent());
  await removal.promise;
  expect(root.isReadCurrent(beforeDelete, ["A"], "id-ws")).toBe(false);
  expect(root.verification.canAccept("x", beforeDelete)).toBe(true);
  expect(root.issues.held("unknown-parent")).toBeNull();
  expect(root.issues.held("x")).not.toBeNull();
});

it("sameRoundRepeatedPartsShareOneMissingReplayGET", async () => {
  const { first, verify } = await missingReplay();
  first.markVerificationStale();
  const at = tab.nerve.calls.length;
  const parts = track(first.refreshStale([{ group: "A" }, { group: "B" }]));
  await until(() => tab.nerve.calls.length === at + 2, "two parts");
  const repeated = track(first.refreshStale([{ group: "A" }, { group: "B" }]));
  await vi.advanceTimersByTimeAsync(10);
  expect(tab.nerve.calls).toHaveLength(at + 4);
  tab.nerve.calls.slice(at).forEach((call) => call.answer(json(200, issuePageOf([]))));
  await vi.advanceTimersByTimeAsync(10);
  expect(tab.nerve.calls.filter((call) => call.method === "GET" && call.path === "/api/v0/issues/x")).toHaveLength(1);
  verify.answer(problem(403, "forbidden"));
  await until(() => parts.settled && repeated.settled, "shared refusal");
  expect(parts.error).toBeUndefined(); // Superseded part promises resolve without publishing.
  expect(first.error).toMatchObject({ status: 403 });
  expect(repeated.error).toMatchObject({ status: 403 });
  expect(first.group("B")?.stale).toBe(true);
});

it.each(
  (["createLink", "removeLink"] as const).flatMap((kind) =>
    (["permission", "superseded", "missing"] as const).map((outcome) => ({ kind, outcome }))
  )
)("legacy $kind $outcome invalid evidence stops activity while retaining write success", async ({ kind, outcome }) => {
  const root = tab.app.issue;
  accept(issueDetailOf("x"));
  const operateResource = await resourceOperation(kind);
  const write = track(operateResource());
  await until(() => tab.nerve.to("/api/v0/issues/x").length === 1, "resource evidence");
  const evidence = tab.nerve.to("/api/v0/issues/x")[0]!;
  if (outcome === "permission")
    tab.app.projectRoot.project.confirmProject(tab.project, (project) => ({ ...project, member_role: null }));
  const replacement =
    outcome === "superseded"
      ? await requestedIssue(tab.nerve, () => root.issues.refreshAuthoritative("x", { afterWrite: true }), [
          "GET",
          "/api/v0/issues/x",
        ])
      : undefined;
  evidence.answer(outcome === "missing" ? problem(404, "missing") : json(200, issueDetailOf("x")));
  await until(() => write.settled, "invalid evidence completes");
  if (replacement) {
    replacement.call.answer(json(200, { ...issueDetailOf("x"), link_count: 6, updated_at: version(2) }));
    await replacement.promise;
  }
  expect(root.active).toBe(true);
  expect(write.error).toBeUndefined();
  expect(root.issueDetail.activity.fetchActivities).not.toHaveBeenCalled();
  expect(tab.nerve.to("/api/v0/issues/x")).toHaveLength(outcome === "superseded" ? 2 : 1);
  expect(root.issueDetail.link.getLinksByIssueId("x")).toEqual(kind === "createLink" ? ["link"] : []);
});

it.each([false, true])(
  "R32 link publication retains ordered IDs after repeated success; initially loaded=%s",
  async (loaded) => {
    accept(issueDetailOf("x"));
    const root = tab.app.issue;
    const operateResource = await resourceOperation("createLink");
    if (loaded) root.issueDetail.link.addLinks("x", [{ id: "old" } as never]);
    const create = vi
      .spyOn(IssueService.prototype, "createIssueLink")
      .mockResolvedValueOnce({ id: "one", title: "one" } as never)
      .mockResolvedValueOnce({ id: "two", title: "two" } as never)
      .mockRejectedValueOnce(new Error("resource refused"));
    const expected = loaded ? ["old"] : [];
    for (const [i, id] of ["one", "two"].entries()) {
      // oxlint-disable-next-line no-await-in-loop
      const write = await requestedIssue(tab.nerve, operateResource, ["GET", "/api/v0/issues/x"]);
      write.call.answer(json(200, { ...issueDetailOf("x"), link_count: i + 1, updated_at: version(i + 1) }));
      // oxlint-disable-next-line no-await-in-loop
      await write.promise;
      expected.push(id);
      expect(write.result.value).toMatchObject({ id, title: id });
      expect(root.issueDetail.link.getLinksByIssueId("x")).toEqual(expected);
      expect(root.issueDetail.link.getLinkById(id)).toMatchObject({ id, title: id });
    }
    const at = tab.nerve.calls.length;
    await expect(operateResource()).rejects.toThrow("resource refused");
    expect(root.issueDetail.link.getLinksByIssueId("x")).toEqual(expected);
    expect(root.issueDetail.link.getLinkById("one")).toBeDefined();
    expect(root.issueDetail.link.getLinkById("two")).toBeDefined();
    expect(tab.nerve.calls).toHaveLength(at);
    expect(create).toHaveBeenCalledTimes(3);
  }
);

it.each([2, 3])("newerWholeBeforeOlderWriteAnswerPreservesLatestEvidence version=%s", async (readVersion) => {
  const root = tab.app.issue;
  const list = bindingOf("project");
  const before = { ...issueDetailOf("x"), sort_order: 10, updated_at: version(1) };
  await load(list, issuePageOf([before]));
  const write = await requestedIssue(tab.nerve, () => root.updateIssue("x", () => ({ sort_order: 20 })), [
    "PATCH",
    "/api/v0/issues/x",
  ]);
  const after = { ...before, sort_order: 20, updated_at: version(2) };
  const latest = { ...after, sort_order: readVersion * 10, updated_at: version(readVersion) };
  await load(list, issuePageOf([latest]));
  write.call.answer(json(200, after));
  await write.promise;
  expect(root.issues.getIssueById("x")).toEqual(latest);
  expect(list.value?.total).toBe(1);
  expect(list.group()).toEqual({
    total: 1,
    rows: [{ id: "x", key: latest.sort_order, updatedAt: latest.updated_at }],
    nextCursor: null,
    boundary: { id: "x", key: latest.sort_order },
    cursorBoundary: { id: "x", key: latest.sort_order },
    stale: false,
  });
  expect(root.verification.isUncertain("x")).toBe(false);
  expect(tab.nerve.calls).toHaveLength(3);
});
