// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import { heldChange } from "@/lib/auth/fake-tab";
import { useProjectPreferencesFollow } from "./use-project-preferences-follow";

const state = vi.hoisted(() => ({
  session: { loginId: "login" },
  route: { workspaceSlug: "ws", projectId: "A" },
  root: { active: true, workspaceSlug: "ws", projectId: "A" },
  priority: "high",
  selectedPriority: "high",
  reads: 0,
  saves: 0,
  failures: [] as unknown[],
  readError: undefined as unknown,
}));
vi.mock("react-router", () => ({ useParams: () => state.route }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: state.session }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/use-refusal-toast", () => ({ useRefusalToast: () => (e: unknown) => state.failures.push(e) }));
vi.mock("@/hooks/store/use-issues", () => {
  const issues = {
    rootIssueStore: state.root,
    isPreferenceQueryCurrent: () => state.priority === state.selectedPriority,
    fetchIssuesWithExistingPagination: async () => {
      state.reads++;
      state.selectedPriority = state.priority;
      if (state.readError) throw state.readError;
    },
  };
  const issuesFilter = {
    getIssueFilters: () => ({ richFilters: { and: [{ priority__in: state.priority }] }, displayFilters: {} }),
  };
  return { useIssues: () => ({ issues, issuesFilter }) };
});
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  state.session.loginId = "login";
  state.route = { workspaceSlug: "ws", projectId: "A" };
  Object.assign(state.root, { active: true, workspaceSlug: "ws", projectId: "A" });
  state.priority = "high";
  state.selectedPriority = "high";
  state.reads = 0;
  state.saves = 0;
  state.failures = [];
  state.readError = undefined;
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});
async function mounted() {
  const root = createRoot(document.createElement("div"));
  let follow!: ReturnType<typeof useProjectPreferencesFollow>;
  function Probe() {
    follow = useProjectPreferencesFollow();
    return null;
  }
  await act(async () => root.render(<Probe />));
  return {
    follow,
    render: () => act(async () => root.render(<Probe />)),
    close: () => act(async () => root.unmount()),
  };
}
it("follows a confirmed query change once; refusal never reads", async () => {
  const { follow, close } = await mounted();
  await follow("ws", "A", async () => {
    state.saves++;
    state.priority = "low";
  });
  expect([state.saves, state.reads, state.failures.length]).toEqual([1, 1, 0]);
  const refusal = new Error("write refused");
  await follow("ws", "A", async () => {
    state.saves++;
    throw refusal;
  });
  expect([state.saves, state.reads, state.failures]).toEqual([2, 1, [refusal]]);
  await close();
});
it("leaves a failed read with its list owner without redoing the confirmed save", async () => {
  const { follow, close } = await mounted();
  const refusal = new Error("read refused");
  state.readError = refusal;
  await follow("ws", "A", async () => {
    state.saves++;
    state.priority = "low";
  });
  expect([state.saves, state.reads, state.readError, state.failures]).toEqual([1, 1, refusal, []]);
  await close();
});
it("does not follow a previous project after navigation", async () => {
  const { follow, close } = await mounted();
  await follow("ws", "A", async () => {
    state.saves++;
    state.priority = "low";
    state.root.projectId = "B";
  });
  expect([state.saves, state.reads, state.failures]).toEqual([1, 0, []]);
  await close();
});

for (const ending of ["permission ABA", "route ABA", "session ABA", "unmount"] as const) {
  it.each(["success", "refusal"] as const)(`does not follow old %s after ${ending}`, async (outcome) => {
    const host = await mounted();
    const held = heldChange<void>();
    const pending = host.follow("ws", "A", () => held.sent);
    if (ending === "permission ABA") {
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
    } else if (ending === "route ABA") {
      state.route = { workspaceSlug: "ws", projectId: "B" };
      await host.render();
      state.route = { workspaceSlug: "ws", projectId: "A" };
      await host.render();
    } else if (ending === "session ABA") {
      state.session.loginId = "other-login";
      tab.app.dispose();
      state.session.loginId = "login";
    } else await host.close();
    state.priority = "low";
    if (outcome === "success") held.answer(undefined);
    else held.refuse(new Error("old refusal"));
    await pending;
    expect(state.reads).toBe(0);
    expect(state.failures).toEqual([]);
    if (ending !== "unmount") await host.close();
  });
}
