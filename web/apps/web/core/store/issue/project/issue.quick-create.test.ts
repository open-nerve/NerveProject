/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */

import type { Issue } from "@nerve/api-client";
import { beforeEach, expect, it, vi } from "vitest";
import { runInAction } from "mobx";
import { ALL_ISSUES } from "@nerve/constants";

import { heldChange } from "@/lib/auth/fake-tab";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { IssueService } from "@/services/issue";
import { issueOf } from "../fake-issues";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({
  rootStore: { projectRoot: { project: { getProjectIdentifierById: () => "P" } } },
}));
const { RootStore } = await import("@/store/root.store");
let app: InstanceType<typeof RootStore>;
beforeEach(() => {
  vi.restoreAllMocks();
  app = new RootStore(new FakeNerve().client());
  app.router.setQuery({ workspaceSlug: "ws", projectId: "A" });
});
function initial(project = "A") {
  const store = app.issue.projectIssues;
  app.issue.issues.addIssue([issueOf("R0", project)]);
  runInAction(() => {
    store.groupedIssueIds = { [ALL_ISSUES]: ["R0"] };
    store.groupedIssueCount = { [ALL_ISSUES]: 1 };
  });
  return store;
}
it("waits for a real response then adds canonical/group/count once", async () => {
  const store = initial();
  const answer = heldChange<Issue>();
  vi.spyOn(IssueService.prototype, "createIssue").mockReturnValue(answer.sent);
  const creating = store.createIssue("ws", "A", { name: "First" });
  expect(Object.keys(app.issue.issues.issuesMap)).toEqual(["R0"]);
  expect(store.getIssueIds()).toEqual(["R0"]);
  expect(store.groupedIssueCount[ALL_ISSUES]).toBe(1);
  answer.answer(issueOf("R1"));
  await creating;
  expect(Object.keys(app.issue.issues.issuesMap)).toEqual(["R0", "R1"]);
  expect(store.getIssueIds()).toEqual(["R0", "R1"]);
  expect(store.groupedIssueCount[ALL_ISSUES]).toBe(2);
});
it("a refused creation leaves canonical/group/count unchanged", async () => {
  const store = initial();
  const answer = heldChange<Issue>();
  vi.spyOn(IssueService.prototype, "createIssue").mockReturnValue(answer.sent);
  const creating = store.createIssue("ws", "A", { name: "First" });
  const refused = expect(creating).rejects.toThrow("refused");
  answer.refuse(new Error("refused"));
  await refused;
  expect(Object.keys(app.issue.issues.issuesMap)).toEqual(["R0"]);
  expect(store.getIssueIds()).toEqual(["R0"]);
  expect(store.groupedIssueCount[ALL_ISSUES]).toBe(1);
});
it.each([
  { workspaceSlug: "ws", projectId: "B" },
  { workspaceSlug: "other", projectId: "A" },
])("late A response cannot enter the current $workspaceSlug/$projectId group", async (route) => {
  const store = initial();
  const answer = heldChange<Issue>();
  vi.spyOn(IssueService.prototype, "createIssue").mockReturnValue(answer.sent);
  const creating = store.createIssue("ws", "A", { name: "First" });
  app.router.setQuery(route);
  initial(route.projectId);
  answer.answer(issueOf("A1"));
  await creating;
  expect(app.issue.issues.getIssueById("A1")?.project_id).toBe("A");
  expect(store.getIssueIds()).toEqual(["R0"]);
  expect(store.groupedIssueCount[ALL_ISSUES]).toBe(1);
});
