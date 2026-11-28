// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { Where, settle, where } from "@/components/common/fake-quick-actions";
import { heldChange, signedIn, type HeldChange } from "@/lib/auth/fake-tab";
import { toasts } from "@/lib/fake-toast";
import { workspaceViewOf } from "@/store/views/fake-views";
import type { ViewContainer } from "@/store/views/view-container";
import type { ViewContent } from "@/store/views/view-content";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { closed, field, fill, submit, title } from "./fake-view-dialog";
import { CreateUpdateViewModal } from "./modal";

// The view's dialog in a workspace (M7/P6b spec 2.11, R14, R29): a new view of the workspace's own, of no project,
// goes to the workspace with its name, description and icon and the spreadsheet's display nerve gives a view made
// without one (WORKSPACE_VIEW_CONTENT), no filters, or with the content Save as gives it; once nerve has made it, the
// dialog says so by the workspace's key and shows the new view's page at the workspace's address. A change sends the
// fields changed alone and says so by the workspace's key. Its views' store is a stand-in whose changes nerve settles
// when the test says (fake-tab.ts); the dialog real. A project's dialog is as ever (modal.test.tsx,
// modal-fields.test.tsx).

const ui = vi.hoisted(() => ({ views: new Map<string, View>(), create: vi.fn(), update: vi.fn() }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-views", () => ({
  useViews: () => ({ getViewById: (id: string) => ui.views.get(id), create: ui.create, update: ui.update }),
}));

const acme = workspaceOf("acme");
const inAcme: ViewContainer = { kind: "workspace", workspaceId: acme.id, workspaceSlug: acme.slug };
const roadmap = workspaceViewOf(acme, "Roadmap", "u-me", { description: "What comes next" });
const made = workspaceViewOf(acme, "Release", "u-me");
/** The workspace's views list, where its dialog of a new view opens ("Add view"). */
const list = "/acme/workspace-views";
/** The display nerve gives a view made without one (ViewCreate; navigation.yaml), in the spreadsheet's layout. */
const SPREADSHEET = {
  display_filters: {
    layout: "spreadsheet",
    group_by: null,
    sub_group_by: null,
    order_by: "-created_at",
    sub_issue: true,
    show_empty_groups: true,
    calendar: { layout: "month", show_weekends: false },
  },
  display_properties: {
    assignee: true,
    start_date: true,
    due_date: true,
    labels: true,
    key: true,
    priority: true,
    state: true,
    sub_issue_count: true,
    link: true,
    attachment_count: true,
    created_on: true,
    updated_on: true,
    cycle: true,
    modules: true,
  },
};

let sent: HeldChange<View>;
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
  closed.mockReset();
  ui.views = new Map([[roadmap.id, roadmap]]);
  sent = heldChange<View>();
  ui.create.mockReset().mockImplementation(() => sent.sent);
  ui.update.mockReset().mockImplementation(() => sent.sent);
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

/** The workspace's dialog, open on its views list: the change of viewId; none, a new view's, of content if given. */
async function open(viewId?: string, content?: ViewContent) {
  await act(async () =>
    root.render(
      <MemoryRouter initialEntries={[list]}>
        <CreateUpdateViewModal
          isOpen
          handleClose={closed}
          workspaceSlug="acme"
          container={inAcme}
          viewId={viewId}
          content={content}
        />
        <Where />
      </MemoryRouter>
    )
  );
  await settle();
}
const answer = async (view: View) => {
  await act(async () => sent.answer(view));
  await settle();
};
const address = () => where.location?.pathname;

it("makes a workspace's view of 'Add view' with its name, description and icon and the spreadsheet's display, no filters", async () => {
  await open();
  expect(title()).toBe("view.create.label");
  await fill("name", "Release");
  await submit();
  expect(ui.create.mock.calls).toStrictEqual([
    [inAcme, { name: "Release", description: "", logo_props: {}, ...SPREADSHEET }],
  ]);
});

it("makes a workspace's view of Save as with the content it was given", async () => {
  const content = { filters: { assignee_id: ["me"] }, ...SPREADSHEET };
  await open(undefined, content);
  await fill("name", "Mine");
  await submit();
  expect(ui.create.mock.calls).toStrictEqual([[inAcme, { name: "Mine", description: "", logo_props: {}, ...content }]]);
});

it("shows the new view's page at the workspace's address and says so by the workspace's key, once nerve has made it", async () => {
  await open();
  await fill("name", "Release");
  await submit();
  expect([closed.mock.calls.length, toasts, address()]).toEqual([0, [], list]);
  await answer(made);
  expect(closed).toHaveBeenCalledOnce();
  expect(toasts).toEqual([
    { type: "success", title: "toast.success", message: "workspace_views.action.create.success" },
  ]);
  expect(address()).toBe(`${list}/${made.id}`);
});

it("changes a workspace's view by the fields changed alone, then says so by the workspace's key, staying where it is", async () => {
  await open(roadmap.id);
  expect([title(), field("name").value, field("description").value]).toEqual([
    "view.update.label",
    "Roadmap",
    "What comes next",
  ]);
  await fill("name", "Roadmap v2");
  await submit();
  expect(ui.update.mock.calls.map(([id, body]) => [id, (body as () => unknown)()])).toStrictEqual([
    [roadmap.id, { name: "Roadmap v2" }],
  ]);
  await answer({ ...roadmap, name: "Roadmap v2" });
  expect(closed).toHaveBeenCalledOnce();
  expect(toasts).toEqual([
    { type: "success", title: "toast.success", message: "workspace_views.action.update.success" },
  ]);
  expect(address()).toBe(list);
});
