// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, it, vi } from "vitest";
import { SWRConfig } from "swr";
import { useIssues } from "@/hooks/store/use-issues";
import { EIssuesStoreType } from "@nerve/types";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { draftOf, issueStoreTab } from "@/store/issue/fake-issues";

import { WorkspaceDraftIssuesRoot } from "./root";

vi.mock("@/lib/store-context", async () => ({ StoreContext: (await import("react")).createContext(undefined) }));
vi.mock("@nerve/i18n", async (original) => ({
  ...(await original()),
  useTranslation: () => ({ t: (s: string) => s }),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "draft-page" } }, publicClient: {} }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => ({ workspaceProjectIds: [] }) }));
vi.mock("@/hooks/store/use-command-palette", () => ({
  useCommandPalette: () => ({ toggleCreateProjectModal: vi.fn() }),
}));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => ({ allowPermissions: () => true }) }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "draft-page" }) }));
vi.mock("./empty-state", () => ({ WorkspaceDraftEmptyState: () => <div>empty</div> }));
vi.mock("./draft-issue-block", () => ({
  DraftIssueBlock: ({ issueId }: { issueId: string }) => <div>draft:{issueId}</div>,
}));

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
it("renders unassigned native drafts with no project directory and no preference access", async () => {
  vi.useFakeTimers();
  const { StoreContext } = await import("@/lib/store-context");
  const read = vi.spyOn(Storage.prototype, "getItem");
  const write = vi.spyOn(Storage.prototype, "setItem");
  const { nerve, app } = await issueStoreTab();
  app.router.setQuery({ workspaceSlug: "ws" });
  const container = document.createElement("div");
  const root = createRoot(container);
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
    await act(async () => until(() => nerve.calls.length === 1, "native drafts"));
    expect(nerve.calls[0]!.path).toBe("/api/v0/workspaces/ws/drafts");
    expect(nerve.calls[0]!.query).toEqual({});
    await act(async () => nerve.calls[0]!.answer(problem(500, "unavailable")));
    expect(container.querySelector('[role="alert"]')).not.toBeNull();
    await act(async () => container.querySelector<HTMLButtonElement>('[role="alert"] button')!.click());
    await act(async () => until(() => nerve.calls.length === 2, "read-only retry"));
    await act(async () =>
      nerve.calls[1]!.answer(
        json(200, { data: [{ ...draftOf("unassigned"), project_id: null }], next_cursor: null, total: 1 })
      )
    );
    expect(container.textContent).toContain("draft:unassigned");
    expect(container.querySelector('[role="alert"]')).toBeNull();
    expect(read.mock.calls.filter(([key]) => key === "issue_local_filters")).toEqual([]);
    expect(write.mock.calls.filter(([key]) => key === "issue_local_filters")).toEqual([]);
    expect(nerve.calls).toHaveLength(2);
  } finally {
    await act(async () => root.unmount());
    app.dispose();
    vi.useRealTimers();
    read.mockRestore();
    write.mockRestore();
  }
});
