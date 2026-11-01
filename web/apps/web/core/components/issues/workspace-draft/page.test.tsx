// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { SWRConfig, type SWRConfiguration } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { StoreContext } from "@/lib/store-context";
import { draftOf, issueStoreTab } from "@/store/issue/fake-issues";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { WorkspaceDraftHeader } from "@/app/(all)/[workspaceSlug]/(projects)/drafts/header";
import { WorkspaceDraftIssuesRoot } from "./root";
import { WorkspaceDraftEmptyState } from "./empty-state";

const bound = vi.hoisted(() => ({ route: { workspaceSlug: "other" } }));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "draft-page" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "draft-page" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => bound.route,
}));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
// Real form/row commands are covered by draft.test; this consumer checks route permissions and list ownership.
vi.mock("@/components/issues/issue-modal/modal", () => ({ CreateUpdateIssueModal: () => null }));
vi.mock("./draft-issue-block", () => ({
  DraftIssueBlock: ({ issueId, workspaceSlug }: { issueId: string; workspaceSlug: string }) => (
    <div>
      {workspaceSlug}:{issueId}
    </div>
  ),
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let container: HTMLDivElement;
let renderer: Root;
let config: SWRConfiguration;
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  tab = await issueStoreTab();
  bound.route = { workspaceSlug: "other" };
  config = { provider: () => new Map(), dedupingInterval: 0 };
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
async function roles(role: 5 | 15) {
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [
    { ...tab.workspace, role: role === 5 ? 15 : 5 },
    { ...workspaceOf("other"), role },
  ]);
  tab.nerve.calls.length = 0;
}
const render = (empty = false) =>
  act(async () =>
    renderer.render(
      <MemoryRouter>
        <SWRConfig value={config}>
          <StoreContext.Provider value={tab.app}>
            <section aria-label="draft header">
              <WorkspaceDraftHeader />
            </section>
            {empty ? (
              <WorkspaceDraftEmptyState workspaceSlug="other" />
            ) : (
              <WorkspaceDraftIssuesRoot workspaceSlug="other" />
            )}
          </StoreContext.Provider>
        </SWRConfig>
      </MemoryRouter>
    )
  );
function button(text: string) {
  const node = [...container.querySelectorAll("button")].find((item) => item.textContent?.trim() === text);
  expect(node, text).toBeDefined();
  return node!;
}
it.each([5, 15] as const)(
  "draft header and empty actions use route membership before the router effect (%s)",
  async (role) => {
    await roles(role);
    await render(true);
    expect(button("workspace_draft_issues.draft_an_issue").disabled).toBe(role === 5);
    expect(button("workspace_empty_state.drafts.cta_primary").disabled).toBe(role === 5);
    expect(tab.app.router.workspaceSlug).toBe("ws");
    expect(tab.nerve.calls).toHaveLength(0);
  }
);
it("personal pagination uses response total, keeps its rows on refusal and retries only a list read", async () => {
  await roles(15);
  await render();
  await act(async () => until(() => tab.nerve.calls.length === 1, "route draft list"));
  expect(tab.nerve.calls[0]).toMatchObject({ method: "GET", path: "/api/v0/workspaces/other/drafts", query: {} });
  const first = { ...draftOf("first", "id-other"), project_id: null };
  const second = { ...draftOf("second", "id-other"), project_id: null };
  await act(async () => tab.nerve.calls[0]!.answer(json(200, { data: [first], total: 123, next_cursor: "opaque" })));
  expect(container.querySelector('[aria-label="draft header"]')!.textContent).toContain("123");
  expect(container.textContent).toContain("other:first");
  await act(async () => button("common.load_more ↓").click());
  await act(async () => until(() => tab.nerve.calls.length === 2, "next draft page"));
  expect(tab.nerve.calls[1]).toMatchObject({
    method: "GET",
    path: "/api/v0/workspaces/other/drafts",
    query: { cursor: "opaque" },
  });
  expect(button("common.load_more ↓").disabled).toBe(true);
  await act(async () => tab.nerve.calls[1]!.answer(problem(500, "unavailable")));
  expect(container.querySelector('[role="alert"]')).not.toBeNull();
  expect(container.textContent).toContain("other:first");
  await act(async () => container.querySelector<HTMLButtonElement>('[role="alert"] button')!.click());
  await act(async () => until(() => tab.nerve.calls.length === 3, "retry read"));
  expect(tab.nerve.calls[2]).toMatchObject({ method: "GET", path: "/api/v0/workspaces/other/drafts", query: {} });
  await act(async () => tab.nerve.calls[2]!.answer(json(200, { data: [second], total: 1, next_cursor: null })));
  expect(container.textContent).toContain("other:second");
  expect(container.textContent).not.toContain("other:first");
  expect(container.querySelector('[role="alert"]')).toBeNull();
  expect(tab.nerve.calls.every((call) => call.method === "GET" && call.path.endsWith("/drafts"))).toBe(true);
});
it("workspace guests do not fetch personal drafts even while the router still names a member workspace", async () => {
  await roles(5);
  await render();
  expect(container.textContent).toContain("you_do_not_have_the_permission_to_access_this_page");
  expect(container.textContent).not.toContain("other:");
  expect(tab.nerve.calls).toHaveLength(0);
  expect(button("workspace_draft_issues.draft_an_issue").disabled).toBe(true);
});
