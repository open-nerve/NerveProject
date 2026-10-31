// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { createMemoryRouter, RouterProvider, useParams } from "react-router";
import { SWRConfig } from "swr";
import { runInAction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { issueDetailOf } from "@/store/issue/fake-issues";
import { accountOf } from "@/store/member/fake-users";
import { loadProjects, stateOf } from "@/store/project/fake-projects";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import Layout from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/issues/(detail)/layout";
import Page from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/issues/(detail)/[archivedIssueId]/page";
import { ArchivedIssueQuickActions } from "../quick-action-dropdowns/archived-issue";
import { IssueListCommandHost } from "../command-host";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "archive-detail" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "archive-detail" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// Omit unrelated shell/editor/history rendering; the route, header, modal, title and property controls remain real.
vi.mock("@/components/core/app-header", () => ({
  AppHeader: ({ header }: { header: ReactNode }) => <header>{header}</header>,
}));
vi.mock("@/components/breadcrumbs/project", () => ({ ProjectBreadcrumb: () => null }));
vi.mock("@/components/core/page-title", () => ({ PageHead: () => null }));
vi.mock("@/components/editor/rich-text/description-input", () => ({
  DescriptionInput: ({ disabled }: { disabled: boolean }) => <div data-description aria-readonly={disabled} />,
}));
vi.mock("@/components/issues/issue-detail/description-versions", () => ({ IssueDescriptionVersions: () => null }));
vi.mock("@/components/issues/issue-detail/reactions", () => ({ IssueReaction: () => null }));
vi.mock("@/components/issues/issue-detail/issue-activity", () => ({ IssueActivity: () => null }));
vi.mock("@/components/issues/issue-detail-widgets", () => ({ IssueDetailWidgets: () => null }));
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
let router: ReturnType<typeof createMemoryRouter> | undefined;
const archived = (id: string) => ({
  ...issueDetailOf(id),
  created_by_id: "u-alice",
  archived_at: "2026-10-30T00:00:00Z",
});
const path = (id: string) => `/ws/projects/A/archives/issues/${id}`;
const dialog = () => document.querySelector<HTMLElement>('[role="dialog"][aria-modal="true"]');
const menu = (name: string) =>
  [...document.querySelectorAll<HTMLElement>('[role="menuitem"]')].find(
    (item) => item.textContent === name && !item.closest("[inert]")
  );
const writes = () => tab.nerve.calls.filter((call) => call.method !== "GET");
function useRouteParams() {
  const p = useParams();
  return {
    workspaceSlug: p.workspaceSlug ?? "",
    projectId: p.projectId ?? "",
    archivedIssueId: p.archivedIssueId ?? "",
  };
}
function RouteLayout() {
  return <Layout params={useRouteParams()} />;
}
function RoutePage() {
  return <Page params={useRouteParams()} />;
}
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  toasts.length = 0;
  router = undefined;
  runInAction(() => {
    tab.app.user.data = accountOf("alice");
  });
  tab.nerve.respond = (call) => {
    if (call.method !== "GET") return;
    if (/^\/api\/v0\/issues\/[^/]+$/.test(call.path)) call.answer(json(200, archived(call.path.split("/").at(-1)!)));
    else if (call.path.endsWith("/states"))
      call.answer(json(200, { data: [stateOf(tab.project, "Done", "completed", 1, { id: "state" })] }));
    else if (/\/issues\/[^/]+\/(sub-issues|relations|links|reactions|activities|comments)$/.test(call.path))
      call.answer(json(200, { data: [], next_cursor: null }));
  };
});
afterEach(async () => {
  await tab.close();
  router?.dispose();
});
async function role(value: 5 | 15 | 20) {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: value }]);
  for (const id of ["old", "new"]) tab.app.issue.issues.acceptRead(archived(id), tab.app.issue.captureRead());
}
async function mount() {
  router = createMemoryRouter(
    [
      {
        path: "/:workspaceSlug/projects/:projectId/archives/issues/:archivedIssueId",
        Component: RouteLayout,
        children: [{ index: true, Component: RoutePage }],
      },
      { path: "*", element: <div>outside archived detail</div> },
    ],
    { initialEntries: [path("old"), path("new")], initialIndex: 1 }
  );
  await tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <RouterProvider router={router} />
    </SWRConfig>
  );
  await act(async () =>
    until(() => document.body.textContent?.includes("issue.archive.banner_message") ?? false, "archived page")
  );
}
async function openMenu() {
  await act(async () => document.querySelector<HTMLButtonElement>("[data-main-menu] > button")!.click());
}
async function openDelete() {
  await openMenu();
  expect(menu("common.actions.delete")).toBeDefined();
  await act(async () => menu("common.actions.delete")!.click());
  expect(dialog()).not.toBeNull();
}
it("keeps the actual archived title and sidebar read-only while exposing authorized commands", async () => {
  await role(20);
  await mount();
  expect(document.querySelector("textarea")).toBeNull();
  expect(document.querySelector("[data-description]")?.getAttribute("aria-readonly")).toBe("true");
  const properties = [...document.querySelectorAll("h5")].find(
    (node) => node.textContent === "common.properties"
  )!.parentElement!;
  expect(properties.textContent).toContain("common.state");
  expect(properties.textContent).toContain("common.priority");
  expect(
    properties.querySelectorAll(
      'button:not([disabled]), input:not([disabled]), [role="combobox"]:not([aria-disabled="true"])'
    )
  ).toHaveLength(0);
  await openMenu();
  expect(menu("common.actions.edit")).toBeUndefined();
  expect(menu("common.actions.make_a_copy")).toBeUndefined();
  expect(menu("Restore")).toBeDefined();
  expect(menu("common.actions.delete")).toBeDefined();
  expect(writes()).toEqual([]);
});
it("a permitted guest can read archived detail but has no restore or delete command", async () => {
  await role(5);
  await mount();
  await openMenu();
  expect(menu("Restore")).toBeUndefined();
  expect(menu("common.actions.delete")).toBeUndefined();
  expect(tab.nerve.calls.some((call) => call.path === "/api/v0/issues/new")).toBe(true);
  expect(writes()).toEqual([]);
});
it.each([true, false])(
  "an archived list row keeps fields read-only and shows deletion only to its creator (creator=%s)",
  async (creator) => {
    await role(15);
    const issue = { ...archived("old"), created_by_id: creator ? "u-alice" : "someone-else" };
    tab.app.issue.issues.acceptRead(issue, tab.app.issue.captureRead());
    router = createMemoryRouter(
      [
        {
          path: "/:workspaceSlug/projects/:projectId/archives/issues/:archivedIssueId",
          element: (
            <IssueListCommandHost>
              <ArchivedIssueQuickActions
                issue={issue}
                parentRef={{ current: null }}
                readOnly
                handleDelete={() => tab.app.issue.archivedIssues.removeIssue("ws", "A", "old")}
                handleRestore={() => tab.app.issue.archivedIssues.restoreIssue("ws", "A", "old")}
              />
            </IssueListCommandHost>
          ),
        },
      ],
      { initialEntries: [path("old")] }
    );
    await tab.render(<RouterProvider router={router} />);
    await openMenu();
    expect(menu("Restore")).toBeDefined();
    expect(!!menu("common.actions.delete")).toBe(creator);
  }
);
it("ends the archive header confirmation on Back before a new read settles, and does not reopen it on Forward", async () => {
  await role(20);
  await tab.app.issue.issueDetail.fetchIssue("ws", "A", "old");
  await tab.app.issue.issueDetail.fetchIssue("ws", "A", "new");
  await mount();
  await openDelete();
  const normal = tab.nerve.respond;
  const held: typeof tab.nerve.calls = [];
  tab.nerve.respond = (call) => {
    if (call.path === "/api/v0/issues/old") held.push(call);
    else normal?.(call);
  };
  await act(async () => router!.navigate(-1));
  expect(router!.state.location.pathname).toBe(path("old"));
  expect(dialog()).toBeNull();
  tab.nerve.respond = normal;
  await act(async () => router!.navigate(1));
  await act(async () => held.forEach((call) => call.answer(json(200, archived("old")))));
  expect(dialog()).toBeNull();
});
it("a refused archived deletion stays in its current confirmation with one native reason", async () => {
  await role(20);
  await mount();
  await openDelete();
  const confirm = [...dialog()!.querySelectorAll<HTMLButtonElement>("button")].find(
    (button) => button.textContent === "delete"
  )!;
  expect(confirm).toBeDefined();
  await act(async () => confirm.click());
  await until(() => writes().length === 1, "delete request");
  await act(async () => writes()[0]!.answer(problem(403, "forbidden")));
  expect(dialog()).not.toBeNull();
  expect(toasts).toEqual([expect.objectContaining({ type: "error", message: "errors.forbidden" })]);
  expect(router!.state.location.pathname).toBe(path("new"));
});
it("an old archived deletion cannot navigate away or close the next route's confirmation", async () => {
  await role(20);
  await mount();
  await openDelete();
  const confirm = [...dialog()!.querySelectorAll<HTMLButtonElement>("button")].find(
    (button) => button.textContent === "delete"
  )!;
  expect(confirm).toBeDefined();
  await act(async () => confirm.click());
  await until(() => writes().length === 1, "old delete request");
  await act(async () => router!.navigate(-1));
  await openDelete();
  await act(async () => writes()[0]!.answer(noContent()));
  expect(router!.state.location.pathname).toBe(path("old"));
  expect(dialog()).not.toBeNull();
  expect(toasts).toEqual([]);
});

it("a non-creator project member can restore archived detail but cannot delete it", async () => {
  runInAction(() => {
    tab.app.user.data = accountOf("other");
  });
  await role(15);
  await mount();
  await openMenu();
  expect(menu("Restore")).toBeDefined();
  expect(menu("common.actions.delete")).toBeUndefined();
});

it("a current authorized deletion returns to the archive list after confirmation", async () => {
  await role(20);
  await mount();
  await openDelete();
  const confirm = [...dialog()!.querySelectorAll<HTMLButtonElement>("button")].find(
    (button) => button.textContent === "delete"
  )!;
  await act(async () => confirm.click());
  await until(() => writes().length === 1, "delete request");
  await act(async () => writes()[0]!.answer(noContent()));
  expect(router!.state.location.pathname).toBe("/ws/projects/A/archives/issues");
  expect(writes()).toHaveLength(1);
});
