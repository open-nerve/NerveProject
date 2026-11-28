// @vitest-environment jsdom
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { StoreContext } from "@/lib/store-context";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { WorkspaceDraftIssuesRoot } from "./root";

// The drafts page reads the workspace's label catalog once as it mounts, for whom may read its drafts (M7/P6b spec
// 2.2, R17): the workspace's admins and members, never its guests, as the drafts themselves. The tab's router still
// names ws; the page's workspace is the route's, other.

vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "draft-page" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "draft-page" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("./draft-issue-block", () => ({ DraftIssueBlock: () => null }));

const CATALOG = "/api/v0/workspaces/other/labels";
const DRAFTS = "/api/v0/workspaces/other/drafts";
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let container: HTMLDivElement;
let renderer: Root;
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  tab = await issueStoreTab();
  container = document.createElement("div");
  document.body.append(container);
  renderer = createRoot(container);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  container.remove();
  tab.app.dispose();
  vi.useRealTimers();
});

it.each([
  { who: "its admin", role: 20, reads: 1 },
  { who: "its member", role: 15, reads: 1 },
  { who: "its guest", role: 5, reads: 0 },
] as const)("reads other's catalog as many times as its drafts as the page mounts: $reads for $who", async (case_) => {
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [
    tab.workspace,
    { ...workspaceOf("other"), role: case_.role },
  ]);
  tab.nerve.calls.length = 0;
  await act(async () =>
    renderer.render(
      <MemoryRouter>
        <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
          <StoreContext.Provider value={tab.app}>
            <WorkspaceDraftIssuesRoot workspaceSlug="other" />
          </StoreContext.Provider>
        </SWRConfig>
      </MemoryRouter>
    )
  );
  await act(async () => vi.advanceTimersByTimeAsync(50));
  expect([tab.nerve.to(CATALOG).length, tab.nerve.to(DRAFTS).length]).toEqual([case_.reads, case_.reads]);
  expect(tab.nerve.calls.map((call) => call.method)).toEqual(Array(2 * case_.reads).fill("GET"));
});
