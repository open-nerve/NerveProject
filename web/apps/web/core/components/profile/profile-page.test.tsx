// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useEffect, type ReactNode } from "react";
import { createMemoryRouter, RouterProvider, useParams } from "react-router";
import { SWRConfig } from "swr";
import { runInAction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { COLLECTION_OPERATOR, EIssuesStoreType, FILTER_FIELD_TYPE, type TProfileViews } from "@nerve/types";
import type { RootStore } from "@/store/root.store";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { issueOf, issuePageOf } from "@/store/issue/fake-issues";
import { accountOf } from "@/store/member/fake-users";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { loadProjects, projectOf, labelOf } from "@/store/project/fake-projects";
import { json, problem, answered, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import Layout from "@/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/layout";
import Page from "@/app/(all)/[workspaceSlug]/(projects)/profile/[userId]/[profileViewId]/page";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "profile-page" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "profile-page" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// Only shell/viewport geometry is substituted: both profile toolbars, filters, layouts, rows and reads are real.
vi.mock("@/components/core/app-header", () => ({
  AppHeader: ({ header, mobileHeader }: { header: ReactNode; mobileHeader: ReactNode }) => (
    <header>
      <div data-desktop>{header}</div>
      <div data-mobile>{mobileHeader}</div>
    </header>
  ),
}));
vi.mock("@/components/core/page-title", () => ({ PageHead: () => null }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
let router: ReturnType<typeof createMemoryRouter> | undefined;
let respondList: ((call: Call) => void) | undefined;
const pid = "11111111-1111-4111-8111-111111111111";
const lid = "22222222-2222-4222-8222-222222222222";
const reads = () => tab.nerve.calls.filter((call) => /\/members\/[^/]+\/issues$/.test(call.path));
const button = (name: string, scope: ParentNode = document) =>
  [...scope.querySelectorAll<HTMLButtonElement>("button")].find(
    (b) => b.getAttribute("aria-label") === name || b.textContent?.trim() === name
  );
const click = (name: string, scope: ParentNode = document) =>
  act(async () => {
    expect(button(name, scope), name).toBeDefined();
    button(name, scope)!.click();
  });
function RouteLayout() {
  const values = useParams();
  useEffect(() => tab.app.router.setQuery(values), [values]);
  return (
    <Layout
      params={{
        workspaceSlug: values.workspaceSlug ?? "",
        userId: values.userId ?? "",
        profileViewId: values.profileViewId ?? "",
      }}
    />
  );
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
beforeEach(async () => {
  localStorage.clear();
  vi.stubGlobal(
    "IntersectionObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  tab = await issueSelectionTab();
  bound.app = tab.app;
  respondList = undefined;
  toasts.length = 0;
  runInAction(() => {
    tab.app.user.data = accountOf("alice");
  });
  await tab.app.state.fetchProjectStates("A");
  tab.nerve.respond = (call) => {
    if (call.method !== "GET") return;
    if (/\/members\/[^/]+\/issues$/.test(call.path)) respondList?.(call);
    else if (call.path === "/api/v0/workspaces/ws/members")
      call.answer(json(200, { data: [membershipOf("alice", {}, "ws"), membershipOf("bob", {}, "ws")] }));
  };
  tab.nerve.calls.length = 0;
});
afterEach(async () => {
  await tab.close();
  router?.dispose();
  router = undefined;
  localStorage.clear();
});
function saved(userId = "u-bob", filters: object = {}) {
  localStorage.setItem(
    "issue_local_filters",
    JSON.stringify([
      {
        key: "PROFILE",
        workspaceSlug: "ws",
        userId: "u-alice",
        viewId: userId,
        filters: { display_filters: { layout: "list", group_by: null }, ...filters },
      },
    ])
  );
}
async function mount(view: TProfileViews = "assigned") {
  router = createMemoryRouter(
    [
      {
        path: "/:workspaceSlug/profile/:userId/:profileViewId",
        Component: RouteLayout,
        children: [{ index: true, Component: RoutePage }],
      },
    ],
    { initialEntries: [`/ws/profile/u-bob/${view}`] }
  );
  await tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <RouterProvider router={router} />
    </SWRConfig>
  );
  await act(async () => until(() => reads().length > 0, "profile list read"));
}
it.each(["assigned", "created", "subscribed"] as const)(
  "keeps %s and project restriction on a real group's next page",
  async (view) => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
      tab.project,
      projectOf("Restricted", tab.workspace.id, { id: pid }),
    ]);
    saved("u-bob", {
      rich_filters: { and: [{ project_id__in: pid }, { priority__in: "high" }] },
      display_filters: { layout: "list", group_by: "priority" },
    });
    respondList = (call) =>
      call.answer(
        json(
          200,
          call.query.cursor
            ? { total: 2, data: [issueOf("second", pid)], next_cursor: null }
            : { total: 2, groups: [{ key: "high", total: 2, data: [issueOf("shared", pid)], next_cursor: "high:/?" }] }
        )
      );
    await mount(view);
    expect(document.querySelector("#issue-shared")).not.toBeNull();
    expect(reads()[0]?.query).toMatchObject({
      relation: view,
      project_id: pid,
      priority: "high",
      group_by: "priority",
    });
    expect(tab.app.issue.profileIssues.groups?.get("high")?.total).toBe(2);
    await click("common.load_more ↓");
    await act(async () => until(() => reads().length === 2, "next group"));
    expect(reads()[1]?.query).toEqual({ ...reads()[0]!.query, group_key: "high", cursor: "high:/?" });
    expect(document.querySelector("#issue-second")).not.toBeNull();
  }
);
it.each(["project", "state_detail.group", "priority", "labels"] as const)(
  "renders real kanban grouping %s with server totals",
  async (group) => {
    saved("u-bob", { display_filters: { layout: "kanban", group_by: group } });
    const key =
      group === "project" ? "A" : group === "state_detail.group" ? "unstarted" : group === "labels" ? "none" : "high";
    respondList = (call) =>
      call.answer(json(200, { total: 4, groups: [{ key, total: 4, data: [issueOf("shared")], next_cursor: "next" }] }));
    await mount("created");
    expect(reads()[0]?.query).toMatchObject({
      relation: "created",
      group_by:
        group === "project"
          ? "project_id"
          : group === "state_detail.group"
            ? "state_group"
            : group === "labels"
              ? "label_id"
              : group,
    });
    expect(tab.app.issue.profileIssues.groups?.get(key)?.total).toBe(4);
    expect(document.body.textContent).toContain(issueOf("shared").name);
    expect(button("common.load_more ↓")).toBeDefined();
  }
);
it("shares display settings across views of one target but isolates another target and ignores the old answer", async () => {
  saved();
  respondList = (call) => call.answer(json(200, issuePageOf([issueOf("shared")])));
  await mount();
  await click("common.display", document.querySelector("[data-desktop]")!);
  await click("common.state");
  expect(tab.app.issue.profileIssuesFilter.getIssueFilters("u-bob")?.displayProperties?.state).toBe(false);
  respondList = undefined;
  await act(async () => router!.navigate("/ws/profile/u-bob/created"));
  const old = reads().at(-1)!;
  expect(old.query.relation).toBe("created");
  expect(document.querySelector("#issue-shared")).toBeNull();
  await act(async () => router!.navigate("/ws/profile/u-alice/subscribed"));
  const current = reads().at(-1)!;
  expect(current.path).toContain("/u-alice/");
  expect(current.query.relation).toBe("subscribed");
  await act(async () => current.answer(json(200, issuePageOf([issueOf("current")]))));
  await act(async () => old.answer(json(200, issuePageOf([issueOf("old")]))));
  expect(document.querySelector("#issue-old")).toBeNull();
  expect(document.querySelector("#issue-current")).not.toBeNull();
  expect(tab.app.issue.profileIssuesFilter.getIssueFilters("u-alice")?.displayProperties?.state).toBe(true);
  respondList = (call) => call.answer(json(200, issuePageOf([issueOf("shared")])));
  await act(async () => router!.navigate("/ws/profile/u-bob/subscribed"));
  expect(reads().at(-1)?.query.relation).toBe("subscribed");
  expect(tab.app.issue.profileIssuesFilter.getIssueFilters("u-bob")?.displayProperties?.state).toBe(false);
  expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
});
it.each(["desktop", "mobile"])(
  "%s display update retains its save after read refusal, reports once, and retries only GET",
  async (owner) => {
    saved();
    respondList = (call) => call.answer(json(200, issuePageOf([issueOf("shared")])));
    await mount();
    respondList = (call) => call.answer(problem(503, "unavailable"));
    await click("common.display", document.querySelector(`[data-${owner}]`)!);
    await click("common.order_by.last_updated");
    expect(tab.app.issue.profileIssuesFilter.getIssueFilters("u-bob")?.displayFilters?.order_by).toBe("-updated_at");
    expect(toasts.filter((toast) => toast.type === "error")).toHaveLength(1);
    expect(button("common.retry")).toBeDefined();
    respondList = (call) => call.answer(json(200, issuePageOf([issueOf("retried")])));
    await click("common.retry");
    expect(document.querySelector("#issue-retried")).not.toBeNull();
    expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
  }
);
it("keeps a cold legal label and project constraint when another filter changes, without prefetching directories", async () => {
  saved("u-bob", { rich_filters: { and: [{ label_id__in: lid }, { project_id__in: pid }, { priority__in: "high" }] } });
  respondList = (call) => call.answer(json(200, issuePageOf([])));
  await mount();
  const filter = tab.app.workItemFilters.getFilter(EIssuesStoreType.PROFILE, "u-bob")!;
  expect(filter.configManager.getConfigByProperty("label_id")?.isEnabled).toBe(true);
  const priority = filter.allConditions.find((condition) => condition.property === "priority")!;
  await act(async () => filter.updateConditionValue(priority.id, ["low"]));
  expect(reads().at(-1)?.query).toMatchObject({ label_id: lid, project_id: pid, priority: "low" });
  expect(tab.nerve.calls.filter((call) => !/\/members(?:\/[^/]+\/issues)?$/.test(call.path))).toHaveLength(0);
});
it("offers only already held labels from visible joined unarchived projects in the route workspace", async () => {
  const visible = tab.project;
  const left = projectOf("left", tab.workspace.id, { id: "left", member_role: null });
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [visible, left]);
  const shown = labelOf(visible, "Visible label", 1, { id: lid });
  await answered(
    tab.nerve,
    () => tab.app.label.fetchProjectLabels(visible.id),
    ["GET", "/api/v0/projects/A/labels"],
    { data: [shown] },
    "visible labels"
  );
  await answered(
    tab.nerve,
    () => tab.app.label.fetchProjectLabels(left.id),
    ["GET", "/api/v0/projects/left/labels"],
    { data: [labelOf(left, "Left label", 1)] },
    "left labels"
  );
  tab.nerve.calls.length = 0;
  saved();
  respondList = (call) => call.answer(json(200, issuePageOf([])));
  await mount();
  const filter = tab.app.workItemFilters.getFilter(EIssuesStoreType.PROFILE, "u-bob")!;
  const config = filter.configManager.getConfigByProperty("label_id")!.getOperatorConfig(COLLECTION_OPERATOR.IN);
  expect(config?.type).toBe(FILTER_FIELD_TYPE.MULTI_SELECT);
  if (config?.type !== FILTER_FIELD_TYPE.MULTI_SELECT) throw new Error("missing label picker");
  const options = typeof config.getOptions === "function" ? await config.getOptions() : config.getOptions;
  expect(options.map((option) => option.value).toSorted()).toEqual([lid, "none"]);
  expect(tab.nerve.calls.some((call) => call.path.endsWith("/labels"))).toBe(false);
});

it.each(["list", "kanban"] as const)(
  "%s shows label-group rows and true totals even while the label catalog is cold",
  async (layout) => {
    saved("u-bob", { display_filters: { layout, group_by: "labels" } });
    respondList = (call) =>
      call.answer(
        json(200, {
          total: 3,
          groups: [
            { key: lid, total: 3, data: [{ ...issueOf("labelled"), label_ids: [lid] }], next_cursor: "label-next" },
          ],
        })
      );
    await mount();
    expect(document.body.textContent).toContain(issueOf("labelled").name);
    expect(tab.app.issue.profileIssues.groups?.get(lid)?.total).toBe(3);
    expect(button("common.load_more ↓")).toBeDefined();
    expect(tab.nerve.calls.some((call) => call.path.endsWith("/labels"))).toBe(false);
    await act(async () =>
      answered(
        tab.nerve,
        () => tab.app.label.fetchProjectLabels("A"),
        ["GET", "/api/v0/projects/A/labels"],
        { data: [labelOf(tab.project, "Now known", 1, { id: lid })] },
        "later label directory"
      )
    );
    expect(document.body.textContent).toContain("Now known");
    expect(document.body.textContent).toContain(issueOf("labelled").name);
  }
);
