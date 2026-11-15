// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A cycle's and a module's page as each one's layout puts it, at the route's address (P7c spec 2.7, R8, A8; re-76 F1):
// the small screen's row of the page's layouts and display shows once nerve has given the page's own preferences,
// which the page asks for after it opened: not before nerve's first answer, nor while that first read failed. A layout
// picked there changes the page's own preferences, never the project's; its display groups by the page's own options,
// never by the page's cycle or module. The header at a wider width and the progress sidebar have their own tests
// (header.test.tsx, page.test.tsx); the stores, the page's work items and HTTP are real.
import { act, createElement, type ComponentType, type ReactNode } from "react";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, describe, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import CycleLayout from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/layout";
import CyclePage from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/[cycleId]/page";
import ModuleLayout from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/layout";
import ModulePage from "@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/[moduleId]/page";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import type { RootStore } from "@/store/root.store";
import { C1, cyclePreferencesOf, layouts, M1, modulePreferencesOf, planningPageTab } from "./roots/fake-planning-page";

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
// jsdom has no viewport geometry; the rows below and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
vi.mock("@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/cycles/(detail)/header", () => ({
  CycleIssuesHeader: () => null,
}));
vi.mock("@/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/header", () => ({
  ModuleIssuesHeader: () => null,
}));
vi.mock("@/components/cycles/progress-sidebar", () => ({ CycleDetailsSidebar: () => null }));
vi.mock("@/components/modules/progress-sidebar", () => ({ ModuleDetailsSidebar: () => null }));
// each layout's icon says which it is
vi.mock("@/components/issues/issue-layouts/layout-icon", () => ({
  IssueLayoutIcon: ({ layout }: { layout: string }) => <svg data-layout={layout} />,
}));

/** A route module's component, given its route's params as the router gives them. */
function RouteOf({ Component }: { Component: unknown }) {
  const params = useParams();
  const Shown = Component as ComponentType<{ params: typeof params }>;
  return <Shown params={params} />;
}

/** Each planning page: its route, its layout and its page, its preferences as nerve answers them, its own grouping. */
const pages = [
  {
    page: "a cycle's page",
    type: EIssuesStoreType.CYCLE as const,
    route: "cycles/:cycleId",
    at: `/ws/projects/A/cycles/${C1}`,
    Layout: CycleLayout,
    Page: CyclePage,
    preferences: cyclePreferencesOf(layouts.list.display),
    // the project's groupings, without the cycle (the module's is the project's, as its modules are on)
    groupings: ["states", "priority", "modules", "labels", "assignees", "created_by", "none"],
    own: "common.cycle",
  },
  {
    page: "a module's page",
    type: EIssuesStoreType.MODULE as const,
    route: "modules/:moduleId",
    at: `/ws/projects/A/modules/${M1}`,
    Layout: ModuleLayout,
    Page: ModulePage,
    preferences: modulePreferencesOf(layouts.list.display),
    groupings: ["states", "priority", "cycle", "labels", "assignees", "created_by", "none"],
    own: "common.modules",
  },
];

let page: Awaited<ReturnType<typeof planningPageTab>>;
afterEach(async () => page.close());

const button = (name: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find(
    (node) => (node.getAttribute("aria-label") ?? node.textContent?.trim()) === name
  );
const menuItem = (text: string) =>
  [...document.querySelectorAll<HTMLElement>('[role="menuitem"]')].find((node) => node.textContent?.trim() === text);
/** What of the page's preferences the small screen's row offers: the layout its trigger shows, and the display. */
function offered() {
  const layout = button("common.layout");
  return {
    layout: layout && [...layout.querySelectorAll("[data-layout]")].map((icon) => icon.getAttribute("data-layout")),
    display: !!button("common.display"),
  };
}

describe.each(pages.map((one) => [one.page, one] as const))(
  "%s",
  (_, { type, route, at, Layout, Page, preferences, groupings, own }) => {
    /** The page of project A for a member, as its layout puts it at its route; nerve has not given its preferences. */
    async function mount() {
      page = await planningPageTab(type, { role: 15 });
      bound.app = page.app;
      await page.render(
        <SWRConfig value={{ provider: () => new Map() }}>
          <MemoryRouter initialEntries={[at]}>
            <Routes>
              <Route path={`/:workspaceSlug/projects/:projectId/${route}`} element={<RouteOf Component={Layout} />}>
                <Route index element={<RouteOf Component={Page} />} />
              </Route>
            </Routes>
          </MemoryRouter>
        </SWRConfig>
      );
      await until(() => page.preferences().length === 1, "the page's preferences");
    }

    it("offers the page's layouts and display once nerve gives the page's preferences, asked for after it opened", async () => {
      await mount();
      expect(offered()).toEqual({ layout: undefined, display: false });
      await act(async () => page.preferences()[0]!.answer(json(200, preferences)));
      expect(offered()).toEqual({ layout: ["list"], display: true });
      await act(async () => button("common.layout")!.click());
      await act(async () => menuItem("issue.layouts.kanban")!.click());
      await until(() => page.preferences().length === 2, "the change of the page's layout");
      expect(page.preferences()[1]).toMatchObject({ method: "PATCH", body: { display_filters: { layout: "kanban" } } });
      expect(page.nerve.to("/api/v0/me/projects/A/preferences")).toEqual([]);
    });

    it("offers neither while nerve could not give the page's first preferences", async () => {
      await mount();
      await act(async () => page.preferences()[0]!.answer(problem(503, "server_busy")));
      expect(document.querySelector('[role="alert"]')?.textContent).toContain("errors.server_busy");
      expect(offered()).toEqual({ layout: undefined, display: false });
    });

    it("groups by the page's own options in its display, never by the page's cycle or module", async () => {
      await mount();
      await act(async () => page.preferences()[0]!.answer(json(200, preferences)));
      await act(async () => button("common.display")!.click());
      const grouping = button("common.group_by")!.closest(".py-2")!;
      const shown = [...grouping.querySelectorAll("button[aria-pressed]")].map((node) => node.textContent);
      expect(shown).toEqual(groupings.map((key) => `common.${key}`));
      expect(shown).not.toContain(own);
    });
  }
);
