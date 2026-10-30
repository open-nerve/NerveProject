// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { loadProjects } from "@/store/project/fake-projects";
import type { TIssueOrderByOptions } from "@nerve/types";
import { runInAction } from "mobx";
import { useGroupIssuesDragNDrop } from "./use-group-dragndrop";
import { issueOf, issuePageOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { IssueListBinding } from "@/store/issue/helpers/issue-list-binding";
import { json, problem } from "@/lib/auth/fake-nerve";
const state = vi.hoisted(() => ({
  tab: undefined as Awaited<ReturnType<typeof issueStoreTab>> | undefined,
  loginId: "one",
  projectId: "A",
  toasts: [] as unknown[],
}));
vi.mock("react-router", () => ({
  useParams: () => ({ workspaceSlug: "ws", projectId: state.projectId, userId: "actor" }),
}));
vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: {
    get state() {
      return { loginId: state.loginId };
    },
  },
  publicClient: {},
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/propel/toast", () => ({
  TOAST_TYPE: { ERROR: "error" },
  setToast: (toast: unknown) => state.toasts.push(toast),
}));
vi.mock("./store/use-issue-detail", () => ({ useIssueDetail: () => state.tab!.app.issue.issueDetail }));
vi.mock("./store/user", () => ({ useUserPermissions: () => state.tab!.app.user.permission }));
vi.mock("./use-project-preferences-follow", () => ({ useProjectPreferencesFollow: () => vi.fn() }));
vi.mock("./store/use-issues", () => ({
  useIssues: (type: EIssuesStoreType) => {
    const root = state.tab!.app.issue;
    if (type === EIssuesStoreType.WORKSPACE_DRAFT) return { issues: root.workspaceDraftIssues };
    if (type === EIssuesStoreType.PROFILE)
      return { issues: root.profileIssues, issuesFilter: root.profileIssuesFilter };
    if (type === EIssuesStoreType.ARCHIVED)
      return { issues: root.archivedIssues, issuesFilter: root.archivedIssuesFilter };
    return { issues: root.projectIssues, issuesFilter: root.projectIssuesFilter };
  },
}));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers();
  state.loginId = "one";
  state.projectId = "A";
  state.toasts = [];
  state.tab = await issueStoreTab();
  const root = state.tab.app.issue;
  const binding = new IssueListBinding(root, {
    scope: "project",
    workspaceSlug: "ws",
    workspaceID: "id-ws",
    projectID: "A",
    query: { order_by: "sort_order" },
  });
  const read = await requestedIssue(state.tab.nerve, () => binding.fetchFirst(), ["GET", "/api/v0/projects/A/issues"]);
  read.call.answer(json(200, issuePageOf([issueOf("x"), { ...issueOf("n"), sort_order: 2000 }])));
  await read.promise;
  runInAction(() => {
    root.projectIssues.currentList = binding;
  });
  state.tab.nerve.calls.length = 0;
});
afterEach(() => {
  state.tab!.app.dispose();
  vi.useRealTimers();
});
async function mounted(
  type: Parameters<typeof useGroupIssuesDragNDrop>[0] = EIssuesStoreType.PROJECT,
  order: TIssueOrderByOptions = "sort_order"
) {
  const renderer = createRoot(document.createElement("div"));
  let drop!: ReturnType<typeof useGroupIssuesDragNDrop>;
  function Probe() {
    drop = useGroupIssuesDragNDrop(type, order, undefined);
    return null;
  }
  await act(async () => renderer.render(<Probe />));
  return {
    drop,
    render: () => act(async () => renderer.render(<Probe />)),
    close: () => act(async () => renderer.unmount()),
  };
}
it("actual hook and actions issue delayed PATCH; session change suppresses refusal toast", async () => {
  const probe = await mounted();
  const request = await requestedIssue(
    state.tab!.nerve,
    () => probe.drop({ columnId: "", groupId: "", id: "x" }, { columnId: "", groupId: "", id: "n" }),
    ["PATCH", "/api/v0/issues/x"]
  );
  expect(request.call.body).toEqual({ sort_order: -63535 });
  state.loginId = "two";
  request.call.answer(problem(403, "forbidden"));
  await request.promise;
  expect(state.toasts).toEqual([]);
  expect(state.tab!.app.issue.issues.getIssueById("x")!.sort_order).toBe(1000);
  await probe.close();
});
it("workspace draft never enters ordinary issue drag API", async () => {
  const probe = await mounted(EIssuesStoreType.WORKSPACE_DRAFT);
  await probe.drop({ columnId: "", groupId: "", id: "x" }, { columnId: "", groupId: "", id: "n" });
  expect(state.tab!.nerve.calls).toEqual([]);
  await probe.close();
});

it.each(["unmount", "route", "binding", "permissionABA"] as const)(
  "late refusal after %s has no old toast",
  async (change) => {
    const probe = await mounted();
    const request = await requestedIssue(
      state.tab!.nerve,
      () => probe.drop({ columnId: "", groupId: "", id: "x" }, { columnId: "", groupId: "", id: "n" }),
      ["PATCH", "/api/v0/issues/x"]
    );
    if (change === "unmount") await probe.close();
    if (change === "route") {
      state.projectId = "B";
      await probe.render();
    }
    if (change === "binding")
      runInAction(() => {
        state.tab!.app.issue.projectIssues.currentList = undefined;
      });
    if (change === "permissionABA") {
      const { app, nerve, workspace, project } = state.tab!;
      await loadProjects(nerve, app.projectRoot.project, workspace, []);
      await loadProjects(nerve, app.projectRoot.project, workspace, [project]);
    }
    request.call.answer(problem(403, "forbidden"));
    await request.promise;
    expect(state.toasts).toEqual([]);
    if (change !== "unmount") await probe.close();
  }
);
it("current refusal is localized by problem code", async () => {
  const probe = await mounted();
  const request = await requestedIssue(
    state.tab!.nerve,
    () => probe.drop({ columnId: "", groupId: "", id: "x" }, { columnId: "", groupId: "", id: "n" }),
    ["PATCH", "/api/v0/issues/x"]
  );
  request.call.answer(problem(403, "forbidden"));
  await request.promise;
  expect(state.toasts).toEqual([{ title: "toast.error", type: "error", message: "errors.forbidden" }]);
  await probe.close();
});
it("a non-manual sort never sends same-column manual reorder", async () => {
  const probe = await mounted(EIssuesStoreType.PROJECT, "-created_at");
  const pending = probe.drop({ columnId: "", groupId: "", id: "x" }, { columnId: "", groupId: "", id: "n" });
  await vi.advanceTimersByTimeAsync(1);
  // Release a faulty write before the assertion, so a red run cannot strand the operation.
  state
    .tab!.nerve.calls.filter((call) => call.method === "PATCH")
    .forEach((call) => call.answer(json(200, issueOf("x"))));
  await pending;
  expect(state.tab!.nerve.calls).toEqual([]);
  await probe.close();
});
