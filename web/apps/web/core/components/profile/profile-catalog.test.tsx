// @vitest-environment jsdom
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */
import { act, type ReactNode } from "react";
import { createMemoryRouter, RouterProvider } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IssueListPage, Label } from "@nerve/api-client";
import { COLLECTION_OPERATOR, EIssuesStoreType, FILTER_FIELD_TYPE } from "@nerve/types";
import type { RootStore } from "@/store/root.store";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { seedAccount } from "@/store/user/fake-account";
import { accountOf } from "@/store/member/fake-users";
import { issueOf, issuePageOf } from "@/store/issue/fake-issues";
import { labelOf, loadProjects, projectOf } from "@/store/project/fake-projects";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { json } from "@/lib/auth/fake-nerve";
import { ProfileIssuesPage } from "./profile-issues";

// The profile's work items read the workspace's label catalog once as they mount, for whom may list them, its admins
// and members (M7/P6b spec 2.12, R18): the label groups' names and the label filter's options come from it, a label
// nerve deleted after the read being the only "—".

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "profile-catalog" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "profile-catalog" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));

let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
let router: ReturnType<typeof createMemoryRouter> | undefined;
const known = "22222222-2222-4222-8222-222222222222";
const CATALOG = "/api/v0/workspaces/ws/labels";
const LIST = "/api/v0/workspaces/ws/members/u-bob/issues";
const catalog = () => tab.nerve.to(CATALOG);
const lists = () => tab.nerve.to(LIST);
const projectLabels = () => tab.nerve.calls.filter((call) => /^\/api\/v0\/projects\/[^/]+\/labels$/.test(call.path));
/** The names of the groups the page shows, from their headers' titles (list and kanban alike). */
const groupNames = () =>
  [...document.querySelectorAll(".line-clamp-1.inline-block.truncate.font-medium")].map((title) => title.textContent);
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
  await seedAccount(tab.nerve, tab.app.user, accountOf("alice"));
  tab.app.router.setQuery({ workspaceSlug: "ws", userId: "u-bob", profileViewId: "assigned" });
});
afterEach(async () => {
  await tab.close();
  router?.dispose();
  router = undefined;
  localStorage.clear();
});
/** The caller's own display for u-bob's work items, as the profile keeps it in the browser. */
function saved(display: object) {
  localStorage.setItem(
    "issue_local_filters",
    JSON.stringify([
      {
        key: "PROFILE",
        workspaceSlug: "ws",
        userId: "u-alice",
        viewId: "u-bob",
        filters: { display_filters: display },
      },
    ])
  );
}
/** nerve answers the work items' read with page and the catalog's with labels, the projects' states as the tab does. */
function answering(page: IssueListPage, labels: Label[]) {
  const states = tab.nerve.respond;
  tab.nerve.respond = (call) => {
    if (call.path === CATALOG) call.answer(json(200, { data: labels }));
    else if (call.path === LIST) call.answer(json(200, page));
    else states?.(call);
  };
}
async function mount() {
  tab.nerve.calls.length = 0;
  router = createMemoryRouter(
    [{ path: "/:workspaceSlug/profile/:userId/:profileViewId", element: <ProfileIssuesPage type="assigned" /> }],
    { initialEntries: ["/ws/profile/u-bob/assigned"] }
  );
  await tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <RouterProvider router={router} />
    </SWRConfig>
  );
  await act(async () => vi.advanceTimersByTimeAsync(50));
}

it.each([
  { who: "its admin", role: 20, reads: 1 },
  { who: "its member", role: 15, reads: 1 },
  { who: "its guest", role: 5, reads: 0 },
] as const)(
  "reads ws's label catalog as many times as the work items as the page mounts: $reads for $who",
  async (case_) => {
    await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: case_.role }]);
    saved({ layout: "list", group_by: null });
    await mount();
    expect([catalog().length, lists().length]).toEqual([case_.reads, case_.reads]);
    expect(tab.nerve.calls.map((call) => call.method)).toEqual(Array(2 * case_.reads).fill("GET"));
  }
);

it.each(["list", "kanban"] as const)(
  "%s grouped by label names each group from the catalog, empty ones too; only a label it lacks is —",
  async (layout) => {
    saved({ layout, group_by: "labels", show_empty_groups: true });
    // nerve groups by a label the catalog holds and by one deleted after its read
    answering(
      {
        total: 2,
        groups: [
          { key: known, total: 1, data: [{ ...issueOf("labelled"), label_ids: [known] }], next_cursor: null },
          { key: "deleted", total: 1, data: [{ ...issueOf("orphan"), label_ids: ["deleted"] }], next_cursor: null },
        ],
      },
      [labelOf(tab.project, "Known", 1, { id: known }), labelOf(tab.project, "Unused", 2, { id: "unused" })]
    );
    await mount();
    expect(groupNames().toSorted()).toEqual(["Known", "None", "Unused", "—"]);
    expect(document.body.textContent).toContain(issueOf("labelled").name);
    expect(document.body.textContent).toContain(issueOf("orphan").name);
    expect(projectLabels()).toEqual([]);
  }
);

it("offers as the label filter's options every label of the catalog, of each joined unarchived project", async () => {
  const other = projectOf("B", tab.workspace.id, { id: "B" });
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, other]);
  const labels = [
    labelOf(tab.project, "Bug", 1, { id: known }),
    labelOf(tab.project, "Bug UI", 2, { parent_id: known }),
    labelOf(other, "Docs", 1),
  ];
  saved({ layout: "list", group_by: null });
  answering(issuePageOf([]), labels);
  await mount();
  const filter = tab.app.workItemFilters.getFilter(EIssuesStoreType.PROFILE, "u-bob")!;
  const config = filter.configManager.getConfigByProperty("label_id")!.getOperatorConfig(COLLECTION_OPERATOR.IN);
  if (config?.type !== FILTER_FIELD_TYPE.MULTI_SELECT) throw new Error("missing label picker");
  const options = typeof config.getOptions === "function" ? await config.getOptions() : config.getOptions;
  expect(options.map((option) => option.value).toSorted()).toEqual(
    [...labels.map((label) => label.id), "none"].toSorted()
  );
  expect([catalog().length, projectLabels().length]).toEqual([1, 0]);
});
