// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter, Route, Routes } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import Page from "./page";

// The notifications page's title (P4 spec 2.6): the inbox of the route's workspace, by its name, as the caller's
// workspaces list it (useRouteWorkspace), not the router store's workspace, which follows the address a commit late
// (the keyword rule workspace-address-route). The preview is a stand-in.

const held = vi.hoisted(() => ({ address: "acme" }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@/components/workspace-notifications", () => ({ NotificationsRoot: () => null }));
vi.mock("@/hooks/store/use-workspace", () => {
  const workspaces = [workspaceOf("acme", { name: "Acme" }), workspaceOf("beta", { name: "Beta" })];
  return {
    useWorkspace: () => ({
      currentWorkspace: workspaces.find((workspace) => workspace.slug === held.address),
      getWorkspaceBySlug: (slug: string) => workspaces.find((workspace) => workspace.slug === slug) ?? null,
    }),
  };
});

let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  document.title = "";
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

const Notifications = Page as unknown as () => React.JSX.Element;

it("is titled the inbox of the route's workspace, not of the workspace the router store still names", async () => {
  held.address = "acme";
  await act(async () =>
    root.render(
      <MemoryRouter initialEntries={["/beta/notifications"]}>
        <Routes>
          <Route path="/:workspaceSlug/notifications" element={<Notifications />} />
        </Routes>
      </MemoryRouter>
    )
  );
  expect(document.title).toBe("notification.page_label Beta");
});
