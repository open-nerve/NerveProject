// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useEffect, type ReactNode } from "react";
import { createMemoryRouter, RouterProvider, useParams } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType, type TProfileViews } from "@nerve/types";
import type { RootStore } from "@/store/root.store";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { issuePageOf } from "@/store/issue/fake-issues";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { useIssuesActions } from "@/hooks/use-issues-actions";
import { useIssueListMount } from "@/hooks/use-issue-list-mount";
import { FiltersDropdown } from "@/components/issues/issue-layouts/filters";
import Layout from "@/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/layout";
import Page from "@/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/[profileViewId]/page";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "profile-route" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "profile-route" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// Keep both real toolbars mounted, independent of CSS/viewport geometry in jsdom.
vi.mock("@/components/core/app-header", () => ({
  AppHeader: ({ header, mobileHeader }: { header: ReactNode; mobileHeader: ReactNode }) => (
    <header>
      <div data-desktop>{header}</div>
      <div data-mobile>{mobileHeader}</div>
    </header>
  ),
}));
vi.mock("@/components/core/page-title", () => ({ PageHead: () => null }));
// This route test isolates the body rendering. Its read still uses the real actions, mount hook and RootStore.
// Full ProfileIssuesPage/list/kanban behavior is covered by profile-page.test.tsx and I19(page).
vi.mock("@/components/profile/profile-issues", () => ({ ProfileIssuesPage: ProfileBody }));
function ProfileBody({ type }: { type: TProfileViews }) {
  const { fetchIssues } = useIssuesActions(EIssuesStoreType.PROFILE);
  useIssueListMount(EIssuesStoreType.PROFILE, tab.app.issue, fetchIssues, { limit: 20 }, type);
  return (
    <div data-profile-view={type}>
      <FiltersDropdown title="Body overlay">
        <p>Body options</p>
      </FiltersDropdown>
    </div>
  );
}
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
let router: ReturnType<typeof createMemoryRouter> | undefined;
const path = (user = "u-alice", view = "assigned") => `/ws/profile/${user}/${view}`;
function RouteLayout() {
  const values = useParams();
  const params = {
    workspaceSlug: values.workspaceSlug ?? "",
    userId: values.userId ?? "",
    profileViewId: values.profileViewId ?? "",
  };
  useEffect(() => tab.app.router.setQuery(values), [values]);
  return <Layout params={params} />;
}
function RoutePage() {
  const values = useParams();
  return (
    <Page
      params={{
        workspaceSlug: values.workspaceSlug ?? "",
        userId: values.userId ?? "",
        profileViewId: values.profileViewId ?? "",
      }}
    />
  );
}
const displays = (scope: string) =>
  document.querySelector<HTMLButtonElement>(`${scope} button[aria-label="common.display"]`);
const memberReads = () => tab.nerve.calls.filter((call) => /\/members\/[^/]+\/issues$/.test(call.path));
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  tab.nerve.respond = (call) => {
    if (call.method !== "GET") return;
    if (call.path === "/api/v0/workspaces/ws/members")
      call.answer(json(200, { data: [membershipOf("alice", {}, "ws"), membershipOf("bob", {}, "ws")] }));
    if (/\/members\/[^/]+\/issues$/.test(call.path)) call.answer(json(200, issuePageOf([])));
  };
  await role(15);
  await tab.app.issue.profileIssuesFilter.fetchFilters("ws", "u-alice");
  await tab.app.issue.profileIssuesFilter.fetchFilters("ws", "u-bob");
  tab.nerve.calls.length = 0;
});
afterEach(async () => {
  await tab.close();
  router?.dispose();
  router = undefined;
});
const role = (value: 5 | 15 | 20) =>
  loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: value }]);
async function mount(view = "assigned") {
  router = createMemoryRouter(
    [
      {
        path: "/:workspaceSlug/profile/:userId/:profileViewId?",
        Component: RouteLayout,
        children: [{ index: true, Component: RoutePage }],
      },
    ],
    { initialEntries: [path("u-bob", "created"), path("u-alice", view)], initialIndex: 1 }
  );
  await tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <RouterProvider router={router} />
    </SWRConfig>
  );
}
it.each(["constructor", "toString", "unknown", ""])(
  "rejects %s before mounting tools or reading member issues",
  async (view) => {
    await mount(view);
    expect(document.querySelector("[data-profile-view]")).toBeNull();
    expect(displays("[data-desktop]")).toBeNull();
    expect(displays("[data-mobile]")).toBeNull();
    expect(memberReads()).toHaveLength(0);
  }
);
it.each(["assigned", "created", "subscribed"] as const)("passes %s unchanged to native list reads", async (view) => {
  await mount(view);
  await act(async () => until(() => memberReads().length === 1, "member list"));
  expect(document.querySelector("[data-profile-view]")?.getAttribute("data-profile-view")).toBe(view);
  expect(memberReads()[0]?.query.relation).toBe(view);
  expect(displays("[data-desktop]")).not.toBeNull();
  expect(displays("[data-mobile]")).not.toBeNull();
});
it("never mounts guest toolbars on desktop or mobile and preserves the member summary", async () => {
  await role(5);
  tab.nerve.calls.length = 0;
  await mount();
  expect(document.body.textContent).toContain("you_do_not_have_the_permission_to_access_this_page");
  expect(displays("[data-desktop]")).toBeNull();
  expect(displays("[data-mobile]")).toBeNull();
  expect(document.querySelector("[data-profile-view]")).toBeNull();
  expect(memberReads()).toHaveLength(0);
  expect(tab.nerve.calls.map((call) => call.path)).toEqual(["/api/v0/workspaces/ws/members"]);
  expect(document.body.textContent).toContain("alice");
});
for (const [owner, selector] of [
  ["desktop", '[data-desktop] button[aria-label="common.display"]'],
  ["mobile", '[data-mobile] button[aria-label="common.display"]'],
  ["body", 'button[aria-label="Body overlay"]'],
] as const) {
  it.each(["target", "view", "back", "permission"] as const)(
    `ends ${owner} overlay on %s and never restores it on return`,
    async (change) => {
      await mount();
      const trigger = () => document.querySelector<HTMLButtonElement>(selector)!;
      await act(async () => {
        trigger().click();
        await vi.advanceTimersByTimeAsync(30);
      });
      expect(trigger().getAttribute("aria-expanded")).toBe("true");
      const oldPanel = trigger().getAttribute("aria-controls")!;
      if (change === "permission") await act(async () => role(5));
      else if (change === "back") await act(async () => router!.navigate(-1));
      else await act(async () => router!.navigate(change === "target" ? path("u-bob") : path("u-alice", "created")));
      expect(document.getElementById(oldPanel)).toBeNull();
      if (change === "permission") await act(async () => role(15));
      else await act(async () => router!.navigate(path()));
      expect(trigger().getAttribute("aria-expanded")).toBe("false");
    }
  );
}
