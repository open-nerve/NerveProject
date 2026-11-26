// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// Whom and when a project's view's page offers Update view (P6a spec 2.2, 2.7, R9, R10): the view's owner, as nerve
// lets change it, an admin or a member (an owner made a guest is not offered it; no lock is looked at, nerve has none),
// once the page holds a change of the view: of its filters (their baseline stays the view's, R10), or of its display,
// the page's draft in nerve's words against the view's (never the controls' words, which differ from nerve's for the
// same display). The page, the stores, SWR and HTTP are real (fake-view-saving.tsx); the app's header a frame.
import { act, type ReactNode } from "react";
import { afterEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { EIssueFilterType } from "@nerve/constants";
import { settle } from "@/components/common/fake-quick-actions";
import { until } from "@/lib/auth/fake-time";
import type { RootStore } from "@/store/root.store";
import { preferencesOf } from "@/store/project/fake-projects";
import { updateButton, viewSavingTab } from "./fake-view-saving";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@/components/issues/peek-overview", () => ({ IssuePeekOverview: () => null }));
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => <span>the project</span> }));
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: ReactNode; mobileHeader?: ReactNode }) => (
    <>
      <header>{props.header}</header>
      <aside>{props.mobileHeader}</aside>
    </>
  ),
}));

let page: Awaited<ReturnType<typeof viewSavingTab>>;
afterEach(async () => page.close());

/** Opens First's page as opened says. */
async function opened(opening: Parameters<typeof viewSavingTab>[0] = {}) {
  page = await viewSavingTab(opening);
  bound.app = page.app;
  await page.open();
}

it.each([
  ["its owner, a member", 15, "u-me", true],
  ["its owner, an admin", 20, "u-me", true],
  ["its owner made a guest", 5, "u-me", false],
  ["a member, another's", 15, "u-other", false],
  ["an admin, another's", 20, "u-other", false],
] as const)("offers %s Update view once the page's filters change: %s", async (_, role, owner, offered) => {
  await opened({ role, view: { owned_by_id: owner } });
  expect(updateButton(), "the view as it is").toBeUndefined();
  await page.dropPriority();
  expect(page.filter().hasChanges, "the page holds a change of the view").toBe(true);
  expect(updateButton() !== undefined).toBe(offered);
  expect(page.filter().canUpdateView).toBe(offered);
});

it("offers no Update view while the page's display is the view's: nerve's words, not the controls'", async () => {
  // nerve's words for a list grouped by state, by its name: the controls' are state and state__name
  const display_filters: View["display_filters"] = {
    ...preferencesOf().display_filters,
    group_by: "state_id",
    order_by: "state_name",
  };
  await opened({ view: { display_filters } });
  expect(page.app.issue.projectViewIssuesFilter.getIssueFilters(page.view.id)?.displayFilters).toMatchObject({
    group_by: "state",
    order_by: "state__name",
  });
  await settle();
  expect(updateButton()).toBeUndefined();
  expect(page.filter().canUpdateView).toBe(false);
});

it("offers Update view once the page's display changes, its filters the view's, and none once it is the view's again", async () => {
  await opened();
  const drafts = page.app.issue.projectViewIssuesFilter;
  await act(async () =>
    drafts.updateFilters("ws", page.view.id, EIssueFilterType.DISPLAY_FILTERS, { show_empty_groups: false })
  );
  await until(() => updateButton() !== undefined, "Update view");
  expect(page.filter().hasChanges, "the filters are the view's").toBe(false);
  expect(page.draft()?.display_filters.show_empty_groups).toBe(false);
  await act(async () =>
    drafts.updateFilters("ws", page.view.id, EIssueFilterType.DISPLAY_FILTERS, { show_empty_groups: true })
  );
  await settle();
  expect(updateButton(), "the display the view's again").toBeUndefined();
  expect(page.writes(), "nothing is sent").toEqual([]);
});
