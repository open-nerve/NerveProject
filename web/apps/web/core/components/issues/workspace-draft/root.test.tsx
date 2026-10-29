// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, it, vi } from "vitest";
import { SWRConfig } from "swr";
import { useIssues } from "@/hooks/store/use-issues";
import { EIssuesStoreType } from "@nerve/types";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { RootStore } from "@/store/root.store";
import { WorkspaceDraftIssuesRoot } from "./root";

const calls = vi.hoisted(() => ({ getIssues: vi.fn(async () => ({ results: [], next_page_results: false })) }));
vi.mock("@/services/issue/workspace_draft.service", () => ({ default: calls }));
vi.mock("@/lib/store-context", async () => ({ StoreContext: (await import("react")).createContext(undefined) }));
vi.mock("@nerve/i18n", async (original) => ({
  ...(await original()),
  useTranslation: () => ({ t: (s: string) => s }),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ workspaceProjectIds: ["project"] }) }));
vi.mock("@/hooks/store/use-command-palette", () => ({
  useCommandPalette: () => ({ toggleCreateProjectModal: vi.fn() }),
}));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => ({ allowPermissions: () => true }) }));
vi.mock("@/hooks/use-workspace-issue-properties", () => ({ useWorkspaceIssueProperties: () => {} }));
vi.mock("./empty-state", () => ({ WorkspaceDraftEmptyState: () => <div>empty</div> }));
vi.mock("./draft-issue-block", () => ({ DraftIssueBlock: () => null }));

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
it("mounts the draft list with only its existing request and no preference access", async () => {
  const { StoreContext } = await import("@/lib/store-context");
  const read = vi.spyOn(Storage.prototype, "getItem");
  const write = vi.spyOn(Storage.prototype, "setItem");
  const nerve = new FakeNerve();
  const app = new RootStore(nerve.client());
  app.router.setQuery({ workspaceSlug: "ws" });
  const root = createRoot(document.createElement("div"));
  const Probe = () => {
    expect(useIssues(EIssuesStoreType.WORKSPACE_DRAFT).issues).toBe(app.issue.workspaceDraftIssues);
    return null;
  };
  try {
    await act(async () => {
      root.render(
        <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
          {/* A single fixed context is used for this one mount. */}
          {/* eslint-disable-next-line react/jsx-no-constructed-context-values */}
          <StoreContext.Provider value={app}>
            <Probe />
            <WorkspaceDraftIssuesRoot workspaceSlug="ws" />
          </StoreContext.Provider>
        </SWRConfig>
      );
    });
    expect(calls.getIssues).toHaveBeenCalledOnce();
    expect(calls.getIssues).toHaveBeenCalledWith("ws", { per_page: 50, cursor: "50:0:0" });
    expect(read.mock.calls.filter(([key]) => key === "issue_local_filters")).toEqual([]);
    expect(write.mock.calls.filter(([key]) => key === "issue_local_filters")).toEqual([]);
    expect(nerve.calls).toEqual([]);
  } finally {
    await act(async () => root.unmount());
    app.dispose();
    read.mockRestore();
    write.mockRestore();
  }
});
