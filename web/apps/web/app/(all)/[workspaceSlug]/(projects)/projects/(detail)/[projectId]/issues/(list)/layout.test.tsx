// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A project's work items page as its layout puts it, at the route's address (P7c spec 2.7, X2; re-76 F1): the small
// screen's row of the page's layouts and display shows once nerve has given the page's preferences, which the page
// asks for after it opened, as the header at a wider width shows its controls: not before nerve's first answer, nor
// while that first read failed. The row's trigger shows the page's layout, and a pick changes the page's preferences.
// The header at a wider width has its own tests (core/components/issues/header.test.tsx); the stores, the page's work
// items and HTTP are real.
import { act, createElement, type ComponentType, type ReactNode } from "react";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { preferencesOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import Layout from "./layout";
import Page from "./page";

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
vi.mock("./header", () => ({ ProjectIssuesHeader: () => null }));
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

let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  await tab.app.state.fetchProjectStates("A");
  tab.nerve.calls.length = 0;
});
afterEach(async () => tab.close());

const preferences = () => tab.nerve.to("/api/v0/me/projects/A/preferences");
/** The project's work items page, as the route's layout puts it. */
const mount = () =>
  tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter initialEntries={["/ws/projects/A/issues"]}>
        <Routes>
          <Route path="/:workspaceSlug/projects/:projectId/issues" element={<RouteOf Component={Layout} />}>
            <Route index element={<RouteOf Component={Page} />} />
          </Route>
        </Routes>
      </MemoryRouter>
    </SWRConfig>
  );
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

it("offers the page's layouts and display once nerve gives the preferences the page asked for after it opened", async () => {
  await mount();
  await until(() => preferences().length === 1, "the page's preferences");
  expect(offered()).toEqual({ layout: undefined, display: false });
  await act(async () => preferences()[0]!.answer(json(200, preferencesOf())));
  expect(offered()).toEqual({ layout: ["list"], display: true });
  await act(async () => button("common.layout")!.click());
  await act(async () => menuItem("issue.layouts.kanban")!.click());
  await until(() => preferences().length === 2, "the change of the page's layout");
  expect(preferences()[1]).toMatchObject({ method: "PATCH", body: { display_filters: { layout: "kanban" } } });
});

it("offers neither while nerve could not give the page's first preferences", async () => {
  await mount();
  await until(() => preferences().length === 1, "the page's preferences");
  await act(async () => preferences()[0]!.answer(problem(503, "server_busy")));
  expect(document.querySelector('[role="alert"]')?.textContent).toContain("errors.server_busy");
  expect(offered()).toEqual({ layout: undefined, display: false });
  await act(async () => vi.advanceTimersByTimeAsync(1000));
  expect(preferences().map(({ method }) => method)).toEqual(["GET"]);
});
