// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The workspace's views list's layout (M7/P6b spec 2.7, 2.10, R13): its header's crumb, the views, which is no link
// (the page is the list), and "Add view" for the workspace's admins and members, which opens the page's dialog of a new
// view of the workspace (ViewActionsHost), never for its guests. The page itself is a marker here (T11 writes it). The
// stores and HTTP are real (fake-view-page.tsx); the app's header a frame that shows the header.
import { act, type ComponentType, type ReactNode } from "react";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { afterEach, expect, it, vi } from "vitest";
import { dialog, settle } from "@/components/common/fake-quick-actions";
import { workspaceViewPageTab } from "@/components/workspace/views/fake-view-page";
import type { RootStore } from "@/store/root.store";
import Layout from "./layout";

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
vi.mock("@/components/core/app-header", () => ({
  AppHeader: (props: { header: ReactNode }) => <header>{props.header}</header>,
}));

let page: Awaited<ReturnType<typeof workspaceViewPageTab>>;
afterEach(async () => page.close());

function RouteOf({ Component }: { Component: unknown }) {
  const Of = Component as ComponentType<{ params: Readonly<Record<string, string | undefined>> }>;
  return <Of params={useParams()} />;
}

it.each([
  ["its admin", 20, true],
  ["its member", 15, true],
  ["its guest", 5, false],
] as const)(
  "crumbs the views, no link, and offers %s (role %i) a new view of the workspace: %s",
  async (_, role, offered) => {
    page = await workspaceViewPageTab(role);
    bound.app = page.app;
    await page.render(
      <MemoryRouter initialEntries={["/ws/workspace-views"]}>
        <Routes>
          <Route path=":workspaceSlug/workspace-views" element={<RouteOf Component={Layout} />}>
            <Route index element={<p>the list</p>} />
          </Route>
        </Routes>
      </MemoryRouter>
    );
    expect(document.body.textContent).toContain("the list");
    expect(document.querySelector("header")?.textContent).toContain("views");
    expect(document.querySelector("header a")).toBeNull();
    const add = [...document.querySelectorAll<HTMLButtonElement>("header button")].find(
      (button) => button.textContent?.trim() === "workspace_views.add_view"
    );
    expect(add !== undefined).toBe(offered);
    if (!add) return;
    await act(async () => add.click());
    await settle();
    expect(dialog()).not.toBeNull();
    expect(page.writes()).toEqual([]);
  }
);
