// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, it, vi } from "vitest";
import { SWRConfig } from "swr";
import { WorkspaceDraftIssues } from "@/store/issue/workspace-draft";
import type { IIssueRootStore } from "@/store/issue/root.store";
import { WorkspaceDraftIssuesRoot } from "./root";

const calls = vi.hoisted(() => ({ getIssues: vi.fn(async () => ({ results: [], next_page_results: false })) }));
vi.mock("@/services/issue/workspace_draft.service", () => ({ default: calls }));
vi.mock("@/lib/store-context", async () => ({ StoreContext: (await import("react")).createContext(undefined) }));
vi.mock("@nerve/i18n", () => ({ useTranslation: () => ({ t: (s: string) => s }) }));
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
  const issue = { workspaceSlug: "ws" } as IIssueRootStore;
  const draft = new WorkspaceDraftIssues(issue);
  const read = vi.spyOn(Storage.prototype, "getItem");
  const write = vi.spyOn(Storage.prototype, "setItem");
  const root = createRoot(document.createElement("div"));
  const context = { issue: { workspaceDraftIssues: draft } } as never;
  try {
    await act(async () => {
      root.render(
        <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
          {/* A single fixed context is used for this one mount. */}
          {/* eslint-disable-next-line react/jsx-no-constructed-context-values */}
          <StoreContext.Provider value={context}>
            <WorkspaceDraftIssuesRoot workspaceSlug="ws" />
          </StoreContext.Provider>
        </SWRConfig>
      );
    });
    expect(calls.getIssues).toHaveBeenCalledOnce();
    expect(calls.getIssues).toHaveBeenCalledWith("ws", { per_page: 50, cursor: "50:0:0" });
    expect(read).not.toHaveBeenCalled();
    expect(write).not.toHaveBeenCalled();
  } finally {
    await act(async () => root.unmount());
    read.mockRestore();
    write.mockRestore();
  }
});
