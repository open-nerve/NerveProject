// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, StrictMode, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { SWRConfig } from "swr";
import { observer } from "mobx-react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueDetailOf, issueLinkOf, issueStoreTab } from "@/store/issue/fake-issues";
import { settle } from "@/lib/auth/fake-time";
import { json } from "@/lib/auth/fake-nerve";
import { loadProjects } from "@/store/project/fake-projects";
import * as detailHooks from "@/hooks/store/use-issue-detail";
import { IssueDetailRoot } from "./issue-detail/root";
import { IssuePeekOverview } from "./peek-overview/root";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "host" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "host" }) }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("react-router", () => ({ useNavigate: () => vi.fn() }));
vi.mock("@/hooks/store/use-issue-detail", async (original) => {
  const actual = await original<typeof import("@/hooks/store/use-issue-detail")>();
  return {
    ...actual,
    useIssueDetail: () => tab.app.issue.issueDetail,
    useIssueDetailModals: actual.useIssueDetailModals,
  };
});
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => tab.app.user.permission }));
vi.mock("@/hooks/store/use-app-theme", () => ({ useAppTheme: () => ({ issueDetailSidebarCollapsed: false }) }));
vi.mock("@/hooks/store/use-issues", () => ({ useIssues: () => ({ issues: tab.app.issue.projectIssues }) }));
vi.mock("@/hooks/use-issue-layout-store", () => ({ useIssueStoreType: () => undefined }));
// Probe the ownership boundary through the actual route/peek roots. Leaf rendering is independent.
vi.mock("./issue-detail/main-content", () => ({
  IssueMainContent: (props: { issueId: string }) => <Probe id={props.issueId} />,
}));
vi.mock("./issue-detail/sidebar", () => ({ IssueDetailsSidebar: () => null }));
vi.mock("./peek-overview/view", () => ({
  IssueView: (props: { issueId: string }) => <Probe id={`peek-${props.issueId}`} />,
}));

let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let element: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
type Modals = ReturnType<typeof detailHooks.useIssueDetailModals>;
const held = new Map<string, Modals>();
const Probe = observer(function Probe({ id }: { id: string }) {
  const modals = detailHooks.useIssueDetailModals();
  held.set(id, modals);
  return (
    <button onClick={() => modals.toggleIssueLinkModal(true)}>
      {id}:{String(modals.isIssueLinkModalOpen)}
    </button>
  );
});
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  tab = await issueStoreTab();
  tab.nerve.respond = (call) => {
    if (/^\/api\/v0\/issues\/[^/]+$/.test(call.path))
      call.answer(json(200, issueDetailOf(call.path.split("/").at(-1)!)));
    else call.answer(json(200, { data: [], next_cursor: null }));
  };
  expect(
    (
      await settle(
        Promise.all(["old", "new"].map((id) => tab.app.issue.issueDetail.issue.fetchIssueCore("ws", "A", id))),
        "core fixtures"
      )
    ).error
  ).toBeUndefined();
  held.clear();
  element = document.createElement("div");
  document.body.append(element);
  root = createRoot(element);
});
afterEach(async () => {
  await act(async () => root.unmount());
  element.remove();
  tab.app.dispose();
  vi.useRealTimers();
});
const render = async (view: ReactNode) =>
  act(async () => root.render(<SWRConfig value={{ provider: () => new Map() }}>{view}</SWRConfig>));
const detail = (id: string, key = "detail") => (
  <IssueDetailRoot key={key} workspaceSlug="ws" projectId="A" issueId={id} />
);
const openAll = (state: Modals) => {
  state.toggleIssueLinkModal(true);
  state.setIssueLinkData(issueLinkOf("link"));
  state.toggleCreateIssueModal(true);
  state.toggleParentIssueModal("old");
  state.toggleDeleteIssueModal("old");
  state.toggleSubIssuesModal("old");
  state.setRelationKey("relates_to");
  state.toggleRelationModal("old", "relates_to");
  state.setIssueCrudOperationState({
    create: { toggle: true, parentIssueId: "old" },
    existing: { toggle: true, parentIssueId: "old" },
  });
};
it("closesOwnedModalsOnScopeChange", async () => {
  await render(detail("old"));
  const old = held.get("old")!;
  await act(async () => openAll(old));
  await render(detail("new"));
  expect(old.isAnyModalOpen).toBe(false);
  expect(old.issueLinkData).toBeNull();
  expect(old.relationKey).toBeNull();
  expect(old.isRelationModalOpen).toBeNull();
  expect(old.issueCrudOperationState).toEqual({
    create: { toggle: false, parentIssueId: undefined },
    existing: { toggle: false, parentIssueId: undefined },
  });
  expect(held.get("new")!.isAnyModalOpen).toBe(false);
});
it("keepsTheNewOwnersModalWhenTheOldHostUnmounts", async () => {
  await render(
    <>
      {detail("old", "old")}
      {detail("new", "new")}
    </>
  );
  const old = held.get("old")!;
  const next = held.get("new")!;
  await act(async () => openAll(old));
  expect(next.isAnyModalOpen).toBe(false);
  await act(async () => next.toggleIssueLinkModal(true));
  await render(<>{detail("new", "new")}</>);
  expect(old.isAnyModalOpen).toBe(false);
  expect(next.isIssueLinkModalOpen).toBe(true);
});
it("doesNotFinishAnOldHostsActionInTheNewScope", async () => {
  await render(detail("old"));
  const old = held.get("old")!;
  let finish!: () => void;
  const pending = new Promise<void>((resolve) => {
    finish = resolve;
  });
  const action = pending.then(() => old.toggleIssueLinkModal(false));
  await render(detail("new"));
  const next = held.get("new")!;
  await act(async () => next.toggleIssueLinkModal(true));
  await act(async () => {
    finish();
    await action;
  });
  expect(next.isIssueLinkModalOpen).toBe(true);
});
it("permission ABA and StrictMode release local payloads without reviving a retired host", async () => {
  await render(<StrictMode>{detail("old")}</StrictMode>);
  const old = held.get("old")!;
  await act(async () => openAll(old));
  await act(async () => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 5 }]);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  });
  expect(old.isAnyModalOpen).toBe(false);
  await act(async () => old.toggleIssueLinkModal(true));
  expect(held.get("old")!.isAnyModalOpen).toBe(false);
});
it("peek teardown clears its selected issue but does not erase a replacement peek", async () => {
  tab.app.issue.issueDetail.setPeekIssue({ workspaceSlug: "ws", projectId: "A", issueId: "old" });
  await render(<IssuePeekOverview hostScope="test-route" />);
  const old = held.get("peek-old")!;
  await act(async () => {
    openAll(old);
    tab.app.issue.issueDetail.setPeekIssue({ workspaceSlug: "ws", projectId: "A", issueId: "new" });
  });
  expect(tab.app.issue.issueDetail.peekIssue?.issueId).toBe("new");
  expect(old.isAnyModalOpen).toBe(false);
  await render(null);
  expect(tab.app.issue.issueDetail.peekIssue).toBeUndefined();
});

it("changing the route that hosts a peek ends the previous selection", async () => {
  tab.app.issue.issueDetail.setPeekIssue({ workspaceSlug: "ws", projectId: "A", issueId: "old" });
  await render(<IssuePeekOverview hostScope="project-A" />);
  await render(<IssuePeekOverview hostScope="project-B" />);
  expect(tab.app.issue.issueDetail.peekIssue).toBeUndefined();
});
it("StrictMode reacquires a peek without clearing the user's selection", async () => {
  tab.app.issue.issueDetail.setPeekIssue({ workspaceSlug: "ws", projectId: "A", issueId: "old" });
  await render(
    <StrictMode>
      <IssuePeekOverview hostScope="project-A" />
    </StrictMode>
  );
  expect(tab.app.issue.issueDetail.peekIssue?.issueId).toBe("old");
});
