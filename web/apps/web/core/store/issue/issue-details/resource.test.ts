/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { heldChange } from "@/lib/auth/fake-tab";
import { until, track } from "@/lib/auth/fake-time";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { json } from "@/lib/auth/fake-nerve";
import { issueDetailOf, issueStoreTab } from "../fake-issues";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { DetailResource } from "./resource";

vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
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
const resource = () =>
  new DetailResource<string[]>(tab.app.issue, { workspaceID: tab.workspace.id, projectIDs: [tab.project.id] });

it("hidesHeldResourcesImmediatelyAfterPermissionChanges", async () => {
  const held = resource();
  await held.fetch(async () => ["secret"]);
  expect(held.value).toEqual(["secret"]);
  tab.app.projectRoot.project.confirmProject(tab.project, (p) => ({ ...p, member_role: null }));
  expect(held.value).toBeUndefined();
});

it("doesNotReviveAfterPermissionABA", async () => {
  const held = resource();
  await held.fetch(async () => ["before"]);
  const delayed = heldChange<string[]>();
  const old = held.fetch(() => delayed.sent);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  expect(held.value).toBeUndefined();
  delayed.answer(["old"]);
  expect(await old).toBeUndefined();
  expect(held.value).toBeUndefined();
  expect(await held.fetch(async () => ["new"])).toEqual(["new"]);
});

it("replaysConfirmationOverOlderReads", async () => {
  const held = resource();
  await held.fetch(async () => ["delete", "edit-old"]);
  const delayed = heldChange<string[]>();
  const old = held.fetch(() => delayed.sent);
  held.confirm(
    (values) => [
      ...values.filter((value) => value !== "delete" && value !== "edit-old" && value !== "new"),
      "edit-new",
      "new",
    ],
    tab.app.issue.captureRead()
  );
  delayed.answer(["delete", "edit-old"]);
  expect(await old).toEqual(["edit-new", "new"]);
  expect(held.value).toEqual(["edit-new", "new"]);
});

it("keeps the last complete value and exposes only a current refusal", async () => {
  const held = resource();
  await held.fetch(async () => ["complete"]);
  const error = new Error("refused");
  await expect(
    held.fetch(async () => {
      throw error;
    })
  ).rejects.toBe(error);
  expect(held.value).toEqual(["complete"]);
  expect(held.error).toBe(error);
  expect(held.loading).toBe(false);
  await held.fetch(async () => ["recovered"]);
  expect(held.error).toBeUndefined();
});

it("invalidates only affected resources including embedded project proofs", async () => {
  const otherWorkspace = workspaceOf("other");
  const other = projectOf("other-project", otherWorkspace.id);
  const second = projectOf("second", tab.workspace.id);
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, otherWorkspace]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, second]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, otherWorkspace, [other]);
  const local = resource();
  const crossProject = resource();
  const foreign = new DetailResource<string[]>(tab.app.issue, {
    workspaceID: otherWorkspace.id,
    projectIDs: [other.id],
  });
  await local.fetch(async () => ["local"]);
  await crossProject.fetch(
    async () => ["linked"],
    () => [second.id]
  );
  await foreign.fetch(async () => ["foreign"]);
  tab.app.projectRoot.project.confirmProject(second, (p) => ({ ...p, member_role: null }));
  expect(crossProject.value).toBeUndefined();
  expect(local.value).toEqual(["local"]);
  expect(foreign.value).toEqual(["foreign"]);
  tab.app.issue.invalidateProjectReads(tab.project.id, tab.workspace.id);
  expect(local.value).toBeUndefined();
  expect(foreign.value).toEqual(["foreign"]);
});

it("retiresRegistrations", async () => {
  const root = tab.app.issue;
  const original = root.registerResource.bind(root);
  const released = vi.fn();
  vi.spyOn(root, "registerResource").mockImplementation((value) => {
    const dispose = original(value);
    return () => {
      released();
      dispose();
    };
  });
  const first = resource();
  const second = resource();
  const late = heldChange<string[]>();
  const reading = second.fetch(() => late.sent);
  await first.fetch(async () => ["first"]);
  first.dispose();
  first.dispose();
  expect(first.value).toBeUndefined();
  root.retire();
  root.retire();
  late.answer(["late"]);
  expect(await reading).toBeUndefined();
  expect(second.value).toBeUndefined();
  expect(released).toHaveBeenCalledTimes(2);
  await expect(first.fetch(async () => ["forbidden"])).rejects.toBeInstanceOf(SessionChangedError);
});

it("old confirmations and refusals cannot enter a recovered permission scope", async () => {
  const held = resource();
  await held.fetch(async () => ["before"]);
  const token = tab.app.issue.captureRead();
  const pending = heldChange<string[]>();
  const reading = held.fetch(() => pending.sent);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  await held.fetch(async () => ["new"]);
  held.confirm(() => ["old write"], token);
  pending.refuse(new Error("old failure"));
  expect(await reading).toBeUndefined();
  expect(held.value).toEqual(["new"]);
  expect(held.error).toBeUndefined();
});

it("shares multi-ID changes with core updates and permits independent IDs", async () => {
  const root = tab.app.issue;
  root.issues.acceptRead(issueDetailOf("a"), root.captureRead());
  const gate = heldChange<void>();
  const entered = vi.fn();
  const multi = root.enqueueIssueChanges(["b", "a", "b"], async () => {
    entered();
    await gate.sent;
  });
  await until(() => entered.mock.calls.length === 1, "batch starts");
  const patch = track(root.updateIssue("a", () => ({ name: "after batch" })));
  expect(await root.enqueueIssueChanges(["c"], async () => "independent")).toBe("independent");
  await vi.advanceTimersByTimeAsync(10);
  expect(tab.nerve.calls).toHaveLength(0);
  gate.answer();
  await multi;
  await until(() => tab.nerve.calls.length === 1, "queued core update");
  tab.nerve.calls[0]!.answer(json(200, { ...issueDetailOf("a"), name: "after batch" }));
  await until(() => patch.settled, "core update settles");
  expect(patch.error).toBeUndefined();
});

it("sorts overlapping ID sets and continues after rejection without deadlocking", async () => {
  const root = tab.app.issue;
  const order: string[] = [];
  const first = track(
    root.enqueueIssueChanges(["b", "a", "b"], async () => {
      order.push("first");
      throw new Error("refused");
    })
  );
  const second = track(
    root.enqueueIssueChanges(["a", "b"], async () => {
      order.push("second");
    })
  );
  await until(() => first.settled && second.settled, "both overlapping writes");
  expect(order).toEqual(["first", "second"]);
  expect(first.error).toBeInstanceOf(Error);
  expect(second.error).toBeUndefined();
});

it("reserves every batch ID immediately while an earlier parent change is still waiting", async () => {
  const root = tab.app.issue;
  const gate = heldChange<void>();
  const entered = vi.fn();
  const blocker = root.enqueueIssueChanges(["a"], async () => {
    entered();
    await gate.sent;
  });
  await until(() => entered.mock.calls.length === 1, "earlier parent change");
  const order: string[] = [];
  const batch = root.enqueueIssueChanges(["b", "a"], async () => {
    order.push("batch");
  });
  const child = root.enqueueIssueChanges(["b"], async () => {
    order.push("child");
  });
  await vi.advanceTimersByTimeAsync(10);
  expect(order).toEqual([]);
  gate.answer();
  await Promise.all([blocker, batch, child]);
  expect(order).toEqual(["batch", "child"]);
});

it("does not send a queued command after retirement", async () => {
  const root = tab.app.issue;
  const held = heldChange<void>();
  const entered = vi.fn();
  const first = root.enqueueIssueChanges(["a"], () => {
    entered();
    return held.sent;
  });
  await until(() => entered.mock.calls.length === 1, "first command started");
  const operation = vi.fn(async () => undefined);
  const next = track(root.enqueueIssueChanges(["a"], operation));
  root.retire();
  held.answer();
  await first;
  await until(() => next.settled, "retired queued command");
  expect(operation).not.toHaveBeenCalled();
  expect(next.error).toBeInstanceOf(SessionChangedError);
});
