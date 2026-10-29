// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { SWRConfig, type Cache } from "swr";
import { beforeEach, afterEach, expect, it, vi } from "vitest";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { loadProjects } from "@/store/project/fake-projects";
import Page from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/issues/(detail)/[issueId]/page";
import { ProjectArchivedIssueDetailsHeader } from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/issues/(detail)/header";

const ui = vi.hoisted(() => ({ navigate: vi.fn(), login: "entry-login" }));
vi.mock("react-router", () => ({ useNavigate: () => ui.navigate, redirect: vi.fn() }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: ui.login }) }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("next-themes", () => ({ useTheme: () => ({ resolvedTheme: "light" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/components/common/logo-spinner", () => ({ LogoSpinner: () => <span>loading</span> }));
vi.mock("@/components/common/empty-state", () => ({
  EmptyState: ({ primaryButton }: { primaryButton: { text: string; onClick: () => void } }) => (
    <button onClick={primaryButton.onClick}>{primaryButton.text}</button>
  ),
}));
vi.mock("@/components/breadcrumbs/project", () => ({ ProjectBreadcrumb: () => null }));
vi.mock("@/components/common/breadcrumb-link", () => ({
  BreadcrumbLink: ({ label }: { label?: string }) => <span>{label}</span>,
}));
vi.mock("@/components/issues/issue-detail/issue-detail-quick-actions", () => ({ IssueDetailQuickActions: () => null }));

let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let root: Root;
let container: HTMLDivElement;
let cache: Cache<unknown>;
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  ui.navigate.mockReset();
  tab = await issueStoreTab();
  container = document.createElement("div");
  document.body.append(container);
  root = createRoot(container);
  cache = new Map();
});
afterEach(async () => {
  await act(async () => root.unmount());
  container.remove();
  tab.app.dispose();
  vi.useRealTimers();
});
const mount = async (child: React.ReactNode) =>
  act(async () => root.render(<SWRConfig value={{ provider: () => cache }}>{child}</SWRConfig>));
const page = (id = "one") => (
  <Page
    {...{
      params: { workspaceSlug: "ws", projectId: "A", issueId: id },
      loaderData: { error: false, workspaceSlug: "ws" },
      actionData: undefined,
      matches: [],
    }}
  />
);
const firstRequest = async () => act(async () => until(() => tab.nerve.calls.length === 1, "entry native GET"));

it("redirects an ID entry using confirmed core and the returned project's directory identifier", async () => {
  await mount(page());
  await firstRequest();
  expect(tab.nerve.calls[0]?.path).toBe("/api/v0/issues/one");
  await act(async () => tab.nerve.calls[0]!.answer(json(200, { ...issueDetailOf("one"), sequence_id: 12 })));
  await act(async () => until(() => ui.navigate.mock.calls.length === 1, "confirmed navigation"));
  expect(ui.navigate).toHaveBeenCalledWith(`/ws/browse/${tab.project.identifier}-12`, { replace: true });
  expect(tab.nerve.calls).toHaveLength(1);
});

it("a refused ID entry retains an error state with a workspace projects destination", async () => {
  await mount(page());
  await firstRequest();
  await act(async () => tab.nerve.calls[0]!.answer(problem(404, "issue.not_found")));
  await act(async () => until(() => !!container.querySelector("button"), "entry refusal"));
  expect(ui.navigate).not.toHaveBeenCalled();
  await act(async () => container.querySelector("button")!.click());
  expect(ui.navigate).toHaveBeenCalledExactlyOnceWith("/ws/projects");
});

it("a previous ID entry cannot navigate after its host has moved", async () => {
  await mount(page("old"));
  await firstRequest();
  const old = tab.nerve.calls[0]!;
  await mount(page("new"));
  await act(async () => until(() => tab.nerve.calls.length === 2, "replacement ID"));
  await act(async () => old.answer(json(200, issueDetailOf("old"))));
  expect(ui.navigate).not.toHaveBeenCalled();
  await act(async () => tab.nerve.calls[1]!.answer(json(200, { ...issueDetailOf("new"), sequence_id: 2 })));
  await act(async () => until(() => ui.navigate.mock.calls.length === 1, "new ID navigation"));
  expect(ui.navigate).toHaveBeenCalledWith(`/ws/browse/${tab.project.identifier}-2`, { replace: true });
});

it("the archive header uses session core data and immediately hides an invalidated permission scope", async () => {
  await mount(<ProjectArchivedIssueDetailsHeader workspaceSlug="ws" projectId="A" archivedIssueId="one" />);
  await firstRequest();
  expect(tab.nerve.calls[0]?.path).toBe("/api/v0/issues/one");
  await act(async () => tab.nerve.calls[0]!.answer(json(200, { ...issueDetailOf("one"), sequence_id: 42 })));
  await act(async () =>
    until(() => container.textContent?.includes(`${tab.project.identifier}-42`) === true, "archive identity")
  );
  await act(async () => loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []));
  expect(container.textContent).not.toContain(`${tab.project.identifier}-42`);
  await act(async () => loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]));
  await act(async () => until(() => tab.nerve.to("/api/v0/issues/one").length === 2, "fresh restored permission"));
  expect(container.textContent).not.toContain(`${tab.project.identifier}-42`);
});
