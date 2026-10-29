/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueDetailOf, issueOf, issueStoreTab, requestedIssue } from "./fake-issues";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { handleDragDrop } from "@/components/issues/issue-layouts/calendar/utils";
import { handleGroupDragDrop } from "@/components/issues/issue-layouts/utils";
import { IssueListBinding } from "./helpers/issue-list-binding";
import { runInAction } from "mobx";
import { toJS } from "mobx";
import { issuePageOf } from "./fake-issues";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  tab.app.issue.acceptListIssues([issueOf("x"), issueOf("y"), issueOf("neighbor")], tab.app.issue.captureRead());
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});
it.each([false, true])("secondDropUsesFirstConfirmedNeighbors; predecessor rejection=%s", async (reject) => {
  const root = tab.app.issue;
  const first = await requestedIssue(
    tab.nerve,
    () => root.updateIssue("x", () => ({ sort_order: 50 }), { group: "" }),
    ["PATCH", "/api/v0/issues/x"]
  );
  const second = root.updateIssue(
    "y",
    () => ({ sort_order: (root.issues.getIssueById("neighbor")!.sort_order + 2000) / 2 }),
    { group: "" }
  );
  const result = track(second);
  await vi.advanceTimersByTimeAsync(10);
  expect(tab.nerve.calls).toHaveLength(1);
  root.acceptListIssues(
    [{ ...issueOf("neighbor"), sort_order: 1800, updated_at: "2026-10-29T00:00:00Z" }],
    root.captureRead()
  );
  first.call.answer(reject ? problem(403, "forbidden") : json(200, { ...issueDetailOf("x"), sort_order: 50 }));
  await until(() => tab.nerve.calls.length === 2, "second drop");
  expect(await tab.nerve.calls[1]!.body).toEqual({ sort_order: 1900 });
  expect(root.issues.getIssueById("y")!.sort_order).toBe(1000);
  tab.nerve.calls[1]!.answer(json(200, { ...issueDetailOf("y"), sort_order: 1900 }));
  await until(() => result.settled, "confirmation");
  expect(result.error).toBeUndefined();
});
it("waitingDropDoesNotCrossSession", async () => {
  const root = tab.app.issue;
  const binding = await loadBinding(10, ["x", "y"]);
  const rows = structuredClone(binding.group()!.rows);
  const total = binding.value!.total;
  const canonical = structuredClone(toJS(root.issues.held("x")));
  tab.nerve.calls.length = 0;
  const first = await requestedIssue(
    tab.nerve,
    () => root.updateIssue("x", () => ({ sort_order: 50 }), { group: "" }),
    ["PATCH", "/api/v0/issues/x"]
  );
  const waiting = track(root.updateIssue("y", () => ({ sort_order: 60 }), { group: "" }));
  root.retire();
  first.call.answer(json(200, { ...issueDetailOf("x"), sort_order: 50, name: "late answer" }));
  await until(() => waiting.settled, "retired queue");
  expect(waiting.error).toBeDefined();
  expect(tab.nerve.calls).toHaveLength(1);
  expect(binding.group()!.rows).toEqual(rows);
  expect(binding.value!.total).toBe(total);
  expect(root.issues.held("x")).toEqual(canonical);
});
it("calendar sends generated update fields and a destination drop", async () => {
  const update = vi.fn().mockResolvedValue(undefined);
  await handleDragDrop("x", "2026-10-28", "2026-10-29", "ws", "A", update);
  const [project, id, body, drop] = update.mock.calls[0]!;
  expect([project, id]).toEqual(["A", "x"]);
  expect(typeof body).toBe("function");
  expect(body(issueOf("x"))).toEqual({ target_date: "2026-10-29" });
  expect(drop).toEqual({ group: "2026-10-29" });
});
it("group consumer passes neighbor IDs and computes legal none values from held issue", async () => {
  const update = vi.fn().mockResolvedValue(undefined);
  await handleGroupDragDrop(
    { id: "x", columnId: "high", groupId: "high", subGroupId: "u" },
    { id: "neighbor", columnId: "none", groupId: "none", subGroupId: "none" },
    () => ({ ...issueOf("x"), priority: "high", assignee_ids: ["u"] }),
    () => ["neighbor"],
    update,
    "priority",
    "assignees"
  );
  const [, , body, drop] = update.mock.calls[0]!;
  expect(typeof body).toBe("function");
  expect(body({ ...issueOf("x"), priority: "high", assignee_ids: ["u"] })).toEqual({
    priority: "none",
    assignee_ids: [],
  });
  expect(drop).toEqual({ group: "none", subgroup: "none", beforeId: "neighbor", afterId: undefined });
});
async function loadBinding(limit: number, ids: string[], calendar = false) {
  const binding = new IssueListBinding(tab.app.issue, {
    scope: "project",
    workspaceSlug: "ws",
    workspaceID: "id-ws",
    projectID: "A",
    query: { order_by: "sort_order", limit, ...(calendar ? { group_by: "target_date" as const } : {}) },
  });
  const request = await requestedIssue(tab.nerve, () => binding.fetchFirst(), ["GET", "/api/v0/projects/A/issues"]);
  const data = ids.map((id, i) => ({
    ...issueOf(id),
    sort_order: 1000 + i * 1000,
    ...(calendar ? { target_date: "2026-10-29" } : {}),
  }));
  request.call.answer(
    json(
      200,
      calendar
        ? { total: data.length, groups: [{ key: "2026-10-29", total: data.length, data, next_cursor: "next" }] }
        : issuePageOf(data, "next")
    )
  );
  await request.promise;
  return binding;
}
it("captured binding uses current canonical neighbors; other query receives no drop extension", async () => {
  const root = tab.app.issue;
  const source = await loadBinding(1, ["neighbor"]);
  const other = await loadBinding(2, ["x"]);
  const facade = root.projectIssues;
  runInAction(() => {
    facade.currentList = source;
  });
  const first = await requestedIssue(
    tab.nerve,
    () => root.updateIssue("x", () => ({ sort_order: 50 }), { group: "" }),
    ["PATCH", "/api/v0/issues/x"]
  );
  const second = facade.dropIssue("ws", "A", "y", () => ({}), { group: "", afterId: "removed-neighbor" });
  const result = track(second);
  runInAction(() => {
    facade.currentList = other;
  });
  root.acceptListIssues(
    [{ ...issueOf("neighbor"), sort_order: 5000, updated_at: "2026-10-29T00:00:00Z" }],
    root.captureRead()
  );
  first.call.answer(json(200, { ...issueDetailOf("x"), sort_order: 50 }));
  await until(() => tab.nerve.calls.length === 4, "captured drop");
  const call = tab.nerve.calls[3]!;
  expect(call.body).toEqual({ sort_order: 70535 });
  expect(source.group()!.rows.map((row) => row.id)).not.toContain("y");
  call.answer(json(200, { ...issueDetailOf("y"), sort_order: 70535 }));
  await until(() => result.settled, "captured confirmation");
  expect(result.error).toBeUndefined();
  expect(source.group()!.rows.map((row) => row.id)).toContain("y");
  expect(other.group()!.rows.map((row) => row.id)).not.toContain("y");
});
it.each(["missing", "empty"])("missing neighbor recomputes valid tail: %s", async (mode) => {
  const root = tab.app.issue;
  const binding = await loadBinding(1, mode === "empty" ? [] : ["neighbor"]);
  runInAction(() => {
    root.projectIssues.currentList = binding;
  });
  const update = await requestedIssue(
    tab.nerve,
    () => root.projectIssues.dropIssue("ws", "A", "y", () => ({}), { group: "", afterId: "gone" }),
    ["PATCH", "/api/v0/issues/y"]
  );
  expect(update.call.body).toEqual({ sort_order: mode === "empty" ? 65535 : 66535 });
  update.call.answer(problem(403, "forbidden"));
  await until(() => update.result.settled, "drop refusal");
  expect(binding.group()?.rows.map((row) => row.id) ?? []).not.toContain("y");
  expect(root.issues.getIssueById("y")!.sort_order).toBe(1000);
});
it.each([4, 30])("calendar full day keeps confirmed move for page limit %s", async (limit) => {
  const root = tab.app.issue;
  const binding = await loadBinding(
    limit,
    Array.from({ length: limit }, (_, i) => `day-${i}`),
    true
  );
  runInAction(() => {
    root.projectIssues.currentList = binding;
  });
  const move = await requestedIssue(
    tab.nerve,
    () =>
      handleDragDrop("y", "2026-10-28", "2026-10-29", "ws", "A", (project, id, makeBody, drop) =>
        root.projectIssues.dropIssue("ws", project, id, makeBody, drop)
      ),
    ["PATCH", "/api/v0/issues/y"]
  );
  expect(move.call.body).toEqual({ target_date: "2026-10-29" });
  move.call.answer(json(200, { ...issueDetailOf("y"), sort_order: 9000, target_date: "2026-10-29" }));
  await move.promise;
  expect(binding.group("2026-10-29")!.rows.map((row) => row.id)).toContain("y");
  expect(binding.group("2026-10-29")!.nextCursor).toBe("next");
});
it("update whitelist never sends a full entity", async () => {
  const move = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.updateIssue("x", (held) => ({ ...held, sort_order: 2000 }), { group: "" }),
    ["PATCH", "/api/v0/issues/x"]
  );
  expect(move.call.body).not.toHaveProperty("id");
  expect(move.call.body).not.toHaveProperty("project_id");
  expect(move.call.body).not.toHaveProperty("updated_at");
  move.call.answer(json(200, issueDetailOf("x")));
  await move.promise;
});
it("two actual card drops share confirmed neighbors across a list switch", async () => {
  const root = tab.app.issue;
  const source = await loadBinding(1, ["x", "neighbor"]);
  const other = await loadBinding(2, ["y"]);
  runInAction(() => {
    root.projectIssues.currentList = source;
  });
  const first = await requestedIssue(
    tab.nerve,
    () => root.projectIssues.dropIssue("ws", "A", "x", () => ({}), { group: "", beforeId: "neighbor" }),
    ["PATCH", "/api/v0/issues/x"]
  );
  const waiting = root.projectIssues.dropIssue("ws", "A", "y", () => ({}), {
    group: "",
    afterId: "x",
    beforeId: "neighbor",
  });
  const result = track(waiting);
  runInAction(() => {
    root.projectIssues.currentList = other;
  });
  await vi.advanceTimersByTimeAsync(10);
  expect(tab.nerve.calls).toHaveLength(3);
  root.acceptListIssues(
    [{ ...issueOf("neighbor"), sort_order: 3000, updated_at: "2026-10-29T00:00:00Z" }],
    root.captureRead()
  );
  first.call.answer(json(200, { ...issueDetailOf("x"), sort_order: 1500, updated_at: "2026-10-29T00:00:00Z" }));
  await until(() => tab.nerve.calls.length === 4, "second card");
  expect(tab.nerve.calls[3]!.body).toEqual({ sort_order: 2250 });
  tab.nerve.calls[3]!.answer(
    json(200, { ...issueDetailOf("y"), sort_order: 2250, updated_at: "2026-10-29T00:00:00Z" })
  );
  await until(() => result.settled, "second card confirmation");
  expect(result.error).toBeUndefined();
});
it.each(["labels", "assignees"] as const)(
  "priority none and %s none reach actual PATCH as generated fields",
  async (field) => {
    const root = tab.app.issue;
    const binding = await loadBinding(1, []);
    runInAction(() => {
      root.projectIssues.currentList = binding;
    });
    const key = field === "labels" ? "label_ids" : "assignee_ids";
    root.acceptListIssues(
      [{ ...issueOf("x"), priority: "high", [key]: ["old"], updated_at: "2026-10-29T00:00:00Z" }],
      root.captureRead()
    );
    const update = await requestedIssue(
      tab.nerve,
      () =>
        handleGroupDragDrop(
          { id: "x", columnId: "high", groupId: "high", subGroupId: "old" },
          { id: undefined, columnId: "none", groupId: "none", subGroupId: "none" },
          root.issues.getIssueById,
          root.projectIssues.getIssueIds,
          (project, id, makeBody, drop) => root.projectIssues.dropIssue("ws", project, id, makeBody, drop),
          "priority",
          field
        ),
      ["PATCH", "/api/v0/issues/x"]
    );
    expect(update.call.body).toEqual({ priority: "none", [key]: [], sort_order: 65535 });
    update.call.answer(json(200, issueDetailOf("x")));
    await update.promise;
  }
);

async function relationshipList(kind: "mention" | "subscriber" | "subscribed", includeNeighbor = true) {
  const root = tab.app.issue;
  root.acceptListIssues([{ ...issueOf("x"), sort_order: 2000 }], root.captureRead());
  const binding = new IssueListBinding(
    root,
    kind === "subscribed"
      ? {
          scope: "member",
          workspaceSlug: "ws",
          workspaceID: "id-ws",
          userID: "actor",
          query: { relation: "subscribed", order_by: "sort_order", group_by: "priority" },
        }
      : {
          scope: "project",
          workspaceSlug: "ws",
          workspaceID: "id-ws",
          projectID: "A",
          query: {
            order_by: "sort_order",
            group_by: "priority",
            ...(kind === "mention" ? { mention_id: ["actor"] } : { subscriber_id: ["actor"] }),
          },
        }
  );
  const read = await requestedIssue(tab.nerve, () => binding.fetchFirst(), [
    "GET",
    kind === "subscribed" ? "/api/v0/workspaces/ws/members/actor/issues" : "/api/v0/projects/A/issues",
  ]);
  read.call.answer(
    json(200, {
      total: includeNeighbor ? 2 : 1,
      groups: [
        {
          key: "none",
          total: includeNeighbor ? 2 : 1,
          data: [...(includeNeighbor ? [issueOf("neighbor")] : []), { ...issueOf("x"), sort_order: 2000 }],
          next_cursor: null,
        },
      ],
    })
  );
  await read.promise;
  const facade = kind === "subscribed" ? root.profileIssues : root.projectIssues;
  runInAction(() => {
    facade.currentList = binding;
  });
  return { binding, facade };
}
it.each(["mention", "subscriber", "subscribed"] as const)(
  "fix1 %s server-proven neighbor retains real drop position",
  async (kind) => {
    const { binding, facade } = await relationshipList(kind);
    expect(binding.group("none")!.rows.map((row) => row.id)).toEqual(["neighbor", "x"]);
    expect(binding.group("none")!.stale).toBe(false);
    const move = await requestedIssue(
      tab.nerve,
      () => facade.dropIssue("ws", "A", "x", () => ({}), { group: "none", beforeId: "neighbor" }),
      ["PATCH", "/api/v0/issues/x"]
    );
    const body = move.call.body;
    move.call.answer(json(200, { ...issueDetailOf("x"), sort_order: -64535 }));
    await move.promise;
    expect(body).toEqual({ sort_order: -64535 });
  }
);
it.each(["stale", "version", "absent-row", "permission"] as const)(
  "fix1 insufficient %s row evidence refuses instead of treating destination as empty",
  async (kind) => {
    const { binding, facade } = await relationshipList("mention", kind !== "absent-row");
    const root = tab.app.issue;
    if (kind === "stale") binding.markMembershipStale(issueOf("neighbor"), issueOf("neighbor"));
    if (kind === "version")
      root.acceptListIssues(
        [{ ...issueOf("neighbor"), updated_at: "2026-10-28T00:00:00.000000001Z" }],
        root.captureRead()
      );
    if (kind === "version") expect(binding.group("none")!.stale).toBe(false);
    if (kind === "permission") binding.invalidatePermission();
    const at = tab.nerve.calls.length;
    const move = track(facade.dropIssue("ws", "A", "x", () => ({}), { group: "none", beforeId: "neighbor" }));
    await until(() => move.settled || tab.nerve.calls.length > at, "evidence decision");
    // Release a faulty PATCH so a red assertion leaves no hanging operation.
    if (tab.nerve.calls.length > at) {
      tab.nerve.calls.at(-1)!.answer(json(200, issueDetailOf("x")));
      await until(() => move.settled, "faulty write");
    }
    expect(tab.nerve.calls).toHaveLength(at);
    expect(move.error).toBeDefined();
  }
);
it("fix1 known different group cannot reuse old server row", async () => {
  const { facade } = await relationshipList("mention");
  const root = tab.app.issue;
  root.acceptListIssues(
    [{ ...issueOf("neighbor"), priority: "high", updated_at: "2026-10-29T00:00:00Z" }],
    root.captureRead()
  );
  const move = await requestedIssue(
    tab.nerve,
    () => facade.dropIssue("ws", "A", "x", () => ({}), { group: "none", beforeId: "neighbor" }),
    ["PATCH", "/api/v0/issues/x"]
  );
  expect(move.call.body).toEqual({ sort_order: 65535 });
  move.call.answer(json(200, issueDetailOf("x")));
  await move.promise;
});
it("empty column retains successfully confirmed drop", async () => {
  const binding = await loadBinding(1, []);
  const root = tab.app.issue;
  runInAction(() => {
    root.projectIssues.currentList = binding;
  });
  const move = await requestedIssue(
    tab.nerve,
    () => root.projectIssues.dropIssue("ws", "A", "x", () => ({}), { group: "" }),
    ["PATCH", "/api/v0/issues/x"]
  );
  expect(move.call.body).toEqual({ sort_order: 65535 });
  expect(binding.group()!.rows).toEqual([]);
  move.call.answer(json(200, { ...issueDetailOf("x"), sort_order: 65535 }));
  await move.promise;
  expect(binding.group()!.rows.map((row) => row.id)).toEqual(["x"]);
  expect(root.issues.getIssueById("x")!.sort_order).toBe(65535);
});

it("fix1 deleted neighbor cannot reuse old relationship row", async () => {
  const { facade } = await relationshipList("mention");
  const root = tab.app.issue;
  const removal = await requestedIssue(tab.nerve, () => root.deleteIssue("neighbor", root.scope("ws", "A")), [
    "DELETE",
    "/api/v0/issues/neighbor",
  ]);
  removal.call.answer(noContent());
  await removal.promise;
  expect(root.issues.held("neighbor")).toBeNull();
  const move = await requestedIssue(
    tab.nerve,
    () => facade.dropIssue("ws", "A", "x", () => ({}), { group: "none", beforeId: "neighbor" }),
    ["PATCH", "/api/v0/issues/x"]
  );
  expect(move.call.body).toEqual({ sort_order: 65535 });
  move.call.answer(json(200, issueDetailOf("x")));
  await move.promise;
});
