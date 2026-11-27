// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { EUserWorkspaceRoles } from "@nerve/types";
import { choose, dialog, dialogButton, row, settle, where } from "@/components/common/fake-quick-actions";
import { type FakeNerve, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { signedIn } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import type { ViewStore } from "@/store/views/view.store";
import { ME, viewRequests, workspaceViewOf, workspaceViewsTab } from "@/store/views/fake-views";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { fill, submit } from "./fake-view-dialog";
import { WorkspaceViewPages, hostPage } from "./fake-view-pages";

// The host of a page of a workspace's own views (M7/P6b spec 2.11, R15; re-97 I5), at the app's own routes: the list
// (:workspaceSlug/workspace-views) and a view's own page (:workspaceSlug/workspace-views/:globalViewId), never a
// project's. A view's quick actions open its page's dialogs, of the workspace's words: the one that deletes the view,
// which follows nerve's answer though the row has gone first and, from the view's own page, goes to the workspace's
// views list; the page whose own view it deletes, the route's globalViewId, is leaving from its sending until nerve
// refuses it. The dialogs are real; the views a views' store's over a fake nerve that answers when the test says; the
// caller acme's admin. A project's page is as ever (actions-host.test.tsx).

const held = vi.hoisted(() => ({
  views: undefined as unknown as ViewStore,
  permissions: undefined as IUserPermissionStore | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-views", () => ({ useViews: () => held.views }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => held.permissions,
  useUser: () => ({ data: { id: "u-me" } }),
}));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ getUserDetails: (id: string) => ({ id, display_name: id.slice(2) }) }),
}));
vi.mock("@/hooks/use-route-workspace", () => ({ useRouteWorkspace: () => null }));
vi.mock("@/components/assets/profile-image", () => ({ ProfileAvatar: () => null }));

let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers();
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  signedIn();
  toasts.length = 0;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

/** acme's admin at the page of acme's own views at page ("list", or "own", Roadmap's), which has Roadmap and Mine. */
async function at(page: "list" | "own") {
  const roadmap = workspaceViewOf(workspaceOf("acme"), "Roadmap", "u-ann");
  const mine = workspaceViewOf(workspaceOf("acme"), "Mine", ME);
  const { acme, nerve, views, permissions } = await workspaceViewsTab(EUserWorkspaceRoles.ADMIN, [roadmap, mine]);
  Object.assign(held, { views, permissions });
  const list = `/${acme.slug}/workspace-views`;
  const path = page === "own" ? `${list}/${roadmap.id}` : list;
  await act(async () => root.render(<WorkspaceViewPages path={path} workspace={acme} listed={[roadmap, mine]} />));
  await settle();
  return { nerve, roadmap, mine, list };
}
const address = () => where.location?.pathname;
const title = () => document.getElementById(dialog()?.getAttribute("aria-labelledby") ?? "")?.textContent;
const press = async (text: string) => {
  await act(async () => dialogButton(text)?.click());
  await settle();
};
/** nerve answers the deletion of the view with response. */
const answerDeletion = async (nerve: FakeNerve, view: View, response: Response) => {
  await act(async () => nerve.to(viewRequests.at(view)).at(-1)?.answer(response));
  await settle();
};
/** The deletion's dialog of Roadmap, opened by its menu and confirmed: nerve has the deletion. */
const deleting = async (nerve: FakeNerve, roadmap: View) => {
  await choose(roadmap.id, "delete");
  await press("delete");
  expect(nerve.to(viewRequests.at(roadmap)).map(({ method }) => method)).toEqual(["DELETE"]);
};

it.each([
  { page: "list", after: "stays on the list" },
  { page: "own", after: "goes to the workspace's views list" },
] as const)(
  "deletes the workspace's view from its $page page by the workspace's words, saying so once nerve has; then $after",
  async ({ page }) => {
    const { nerve, roadmap, mine, list } = await at(page);
    await choose(roadmap.id, "delete");
    expect(title()).toBe("workspace_views.delete_view.title");
    expect(dialog()?.textContent).toContain("workspace_views.delete_view.content");
    await press("delete");
    expect(nerve.to(viewRequests.at(roadmap)).map(({ method }) => method)).toEqual(["DELETE"]);
    expect([dialog() === null, toasts]).toEqual([false, []]);
    await answerDeletion(nerve, roadmap, noContent());
    expect(toasts).toEqual([
      { type: "success", title: "toast.success", message: "workspace_views.action.delete.success" },
    ]);
    expect(dialog()).toBeNull();
    expect(address()).toBe(list);
    expect([row(roadmap.id), row(mine.id) === null]).toEqual([null, false]);
  }
);

it.each([
  { page: "own", shown: "the view it shows, the route's globalViewId", leaving: true },
  { page: "list", shown: "a view of its list", leaving: false },
] as const)(
  "is leaving once its dialog sends the deletion of $shown ($leaving); refused, it is not, and shows the view as before",
  async ({ page, leaving }) => {
    const { nerve, roadmap, list } = await at(page);
    const shown = hostPage.current;
    expect(shown?.pageLeaving.current, "before the deletion").toBe(false);
    await deleting(nerve, roadmap);
    expect(shown?.pageLeaving.current, "the deletion out").toBe(leaving);
    await answerDeletion(nerve, roadmap, problem(404, "navigation.view_not_found"));
    expect(shown?.pageLeaving.current, "nerve refused it").toBe(false);
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.navigation_view_not_found" }]);
    expect(row(roadmap.id)).not.toBeNull();
    expect(address()).toBe(page === "own" ? `${list}/${roadmap.id}` : list);
  }
);

it("opens the page's dialog that changes the workspace's view from its row's Edit, saying so by the workspace's words", async () => {
  const { nerve, mine } = await at("list");
  await choose(mine.id, "edit");
  await fill("name", "Mine v2");
  await submit();
  expect(nerve.to(viewRequests.at(mine)).map(({ method, body }) => [method, body])).toEqual([
    ["PATCH", { name: "Mine v2" }],
  ]);
  await act(async () =>
    nerve
      .to(viewRequests.at(mine))
      .at(-1)
      ?.answer(json(200, { ...mine, name: "Mine v2" }))
  );
  await settle();
  expect(toasts).toEqual([
    { type: "success", title: "toast.success", message: "workspace_views.action.update.success" },
  ]);
  expect(row(mine.id)?.textContent).toContain("Mine v2");
});
