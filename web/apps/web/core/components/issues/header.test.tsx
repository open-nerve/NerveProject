// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The header of a project's work items page (P6b spec 2.8, R3; handoff item 8): the page's layouts, the toggle of its
// filters and its display once nerve has given the page's own preferences, which they change, as its work items show
// them: not before nerve's first answer, nor while that first read failed. Its new work item is the project's members',
// as before. The page's work items below, the stores and HTTP are real.
import { act, createElement, type ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { preferencesOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { ProjectLayoutRoot } from "./issue-layouts/roots/project-layout-root";
import { IssuesHeader } from "./header";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws", projectId: "A" }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// jsdom has no viewport geometry; the rows below and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
// The project's crumbs have their own tests.
vi.mock("@/components/breadcrumbs/common", () => ({ CommonProjectBreadcrumbs: () => null }));

let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  await tab.app.state.fetchProjectStates("A");
  tab.nerve.calls.length = 0;
});
afterEach(async () => tab.close());

const preferences = () => tab.nerve.to("/api/v0/me/projects/A/preferences");
/** The project's work items page: its header and its work items, as the page puts them. */
const mount = () =>
  tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <div data-header>
          <IssuesHeader />
        </div>
        <ProjectLayoutRoot />
      </MemoryRouter>
    </SWRConfig>
  );
const inHeader = (name: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("[data-header] button")].find(
    (node) => (node.getAttribute("aria-label") ?? node.textContent?.trim()) === name
  );
/** What of the page's work items the header offers. */
function offered() {
  return {
    layouts: ["list", "kanban", "calendar", "spreadsheet"].filter((layout) =>
      inHeader(`issue.layouts.title.${layout}`)
    ),
    filters: !!inHeader("common.filter_options.add"),
    display: !!inHeader("common.display"),
    // its new work item names itself by width: a work item when narrow, a new one otherwise
    create: [...document.querySelectorAll("[data-header] button")].some((node) =>
      node.textContent?.endsWith("issue.add.label")
    ),
  };
}
const all = ["list", "kanban", "calendar", "spreadsheet"];

it("offers no layout, filter or display before nerve gives the page's preferences, and offers them once it has", async () => {
  await mount();
  await until(() => preferences().length === 1, "the page's preferences");
  expect(offered()).toEqual({ layouts: [], filters: false, display: false, create: true });
  await act(async () => preferences()[0]!.answer(json(200, preferencesOf())));
  expect(offered()).toEqual({ layouts: all, filters: true, display: true, create: true });
  await act(async () => inHeader("issue.layouts.title.kanban")?.click());
  await until(() => preferences().length === 2, "the change of the page's layout");
  expect(preferences()[1]).toMatchObject({ method: "PATCH", body: { display_filters: { layout: "kanban" } } });
});

it("offers no layout, filter or display while nerve could not give the page's first preferences", async () => {
  await mount();
  await until(() => preferences().length === 1, "the page's preferences");
  await act(async () => preferences()[0]!.answer(problem(503, "server_busy")));
  expect(document.querySelector('[role="alert"]')?.textContent).toContain("errors.server_busy");
  expect(offered()).toEqual({ layouts: [], filters: false, display: false, create: true });
  await act(async () => vi.advanceTimersByTimeAsync(1000));
  expect(preferences().map(({ method }) => method)).toEqual(["GET"]);
});
