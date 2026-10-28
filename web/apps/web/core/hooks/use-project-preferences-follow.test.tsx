// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, expect, it, vi } from "vitest";
import { useProjectPreferencesFollow } from "./use-project-preferences-follow";

const state = vi.hoisted(() => ({
  route: { workspaceSlug: "ws", projectId: "A" },
  root: { active: true, workspaceSlug: "ws", projectId: "A" },
  priority: "high",
  reads: 0,
  saves: 0,
  failures: [] as unknown[],
  readError: undefined as unknown,
}));
vi.mock("react-router", () => ({ useParams: () => state.route }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "login" } } }));
vi.mock("@/hooks/use-refusal-toast", () => ({ useRefusalToast: () => (e: unknown) => state.failures.push(e) }));
vi.mock("@/hooks/store/use-issues", () => {
  const issues = {
    rootIssueStore: state.root,
    fetchIssuesWithExistingPagination: async () => {
      state.reads++;
      if (state.readError) throw state.readError;
    },
  };
  const issuesFilter = {
    getIssueFilters: () => ({ richFilters: { and: [{ priority__in: state.priority }] }, displayFilters: {} }),
  };
  return { useIssues: () => ({ issues, issuesFilter }) };
});
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  state.route = { workspaceSlug: "ws", projectId: "A" };
  Object.assign(state.root, { active: true, workspaceSlug: "ws", projectId: "A" });
  state.priority = "high";
  state.reads = 0;
  state.saves = 0;
  state.failures = [];
  state.readError = undefined;
});
async function mounted() {
  const root = createRoot(document.createElement("div"));
  let follow!: ReturnType<typeof useProjectPreferencesFollow>;
  function Probe() {
    follow = useProjectPreferencesFollow();
    return null;
  }
  await act(async () => root.render(<Probe />));
  return { follow, close: () => act(async () => root.unmount()) };
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
it("reports a failed read separately without redoing the confirmed save", async () => {
  const { follow, close } = await mounted();
  const refusal = new Error("read refused");
  state.readError = refusal;
  await follow("ws", "A", async () => {
    state.saves++;
    state.priority = "low";
  });
  expect([state.saves, state.reads, state.failures]).toEqual([1, 1, [refusal]]);
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
