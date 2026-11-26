// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { dialog, settle, where } from "@/components/common/fake-quick-actions";
import { ProjectLevelModals } from "@/components/modals/project-level";
import { heldChange, signedIn, switchAccount, type HeldChange } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { CommandPaletteStore } from "@/store/command-palette.store";
import {
  WEB,
  button,
  clickOutside,
  closed,
  escape,
  field,
  fill,
  open,
  openIconPicker,
  opener,
  submit,
  title,
} from "./fake-view-dialog";

// The view's dialog (P6a spec 2.4, R11): how it closes (as M6's: while nothing is out, Escape, a click outside and
// Cancel close it at once; while what it sent is out, nothing does; an Escape the icon picker or an input method takes
// is theirs), its name, and what follows nerve's answer: only once nerve has made the view or the change does it close
// and say so, a new view's page then shown; refused, it stays; and only while it is mounted, in the session it was
// sent in. Its views' store is a stand-in whose changes nerve settles when the test says (fake-tab.ts); the dialog real.

const ui = vi.hoisted(() => ({ views: new Map<string, View>(), create: vi.fn(), update: vi.fn(), palette: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/lib/store-context", () => ({ rootStore: { powerK: {} } }));
vi.mock("@/hooks/store/use-views", () => ({
  useViews: () => ({ getViewById: (id: string) => ui.views.get(id), create: ui.create, update: ui.update }),
}));
vi.mock("@/hooks/store/use-command-palette", () => ({ useCommandPalette: () => ui.palette }));
vi.mock("@/components/cycles/modal", () => ({ CycleCreateUpdateModal: () => null }));
vi.mock("@/components/modules/modal", () => ({ CreateUpdateModuleModal: () => null }));

const bugs: View = {
  id: "v-bugs",
  workspace_id: "w-acme",
  project_id: WEB,
  name: "Bugs",
  description: "Open bugs",
  logo_props: {},
  filters: { priority: ["high"] },
  display_filters: {
    layout: "kanban",
    group_by: "state_id",
    sub_group_by: null,
    order_by: "priority",
    sub_issue: false,
    show_empty_groups: false,
    calendar: { layout: "week", show_weekends: true },
  },
  display_properties: {
    assignee: true,
    start_date: false,
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
  owned_by_id: "u-me",
  created_at: "2026-11-01T09:00:00Z",
  updated_at: "2026-11-01T09:00:00Z",
};
const made = (name: string): View => ({ ...bugs, id: "v-made", name });

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
  ui.views = new Map([[bugs.id, bugs]]);
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

const answer = async (view: View) => {
  await act(async () => sent.answer(view));
  await settle();
};
const refuse = async (error: unknown) => {
  await act(async () => sent.refuse(error));
  await settle();
};
const address = () => where.location?.pathname;
const list = `/acme/projects/${WEB}/views`;

it.each([
  { opened: "a new view", viewId: undefined, name: "view.create.label" },
  { opened: "Save as", viewId: undefined, content: { filters: {} }, name: "view.create.label" },
  { opened: "a change", viewId: bugs.id, name: "view.update.label" },
])("names itself by its title, $name, for $opened", async ({ viewId, content, name }) => {
  await open(root, { viewId, content });
  expect(title()).toBe(name);
});

it("closes on Escape, a click outside and Cancel through its handleClose while nothing is out, the focus back on its opener", async () => {
  await open(root);
  await escape();
  expect([closed.mock.calls.length, dialog()]).toEqual([1, null]);
  expect(document.activeElement).toBe(opener());
  await act(async () => opener()?.click());
  await settle();
  await clickOutside();
  expect([closed.mock.calls.length, dialog()]).toEqual([2, null]);
  await act(async () => opener()?.click());
  await settle();
  await act(async () => button("common.cancel").click());
  await settle();
  expect([closed.mock.calls.length, dialog()]).toEqual([3, null]);
  expect(ui.create).not.toHaveBeenCalled();
});

it("lets nothing close it while what it sent is out (Escape, a click outside, Cancel disabled), then closes once nerve has answered", async () => {
  await open(root);
  await fill("name", "Mine");
  await submit();
  expect(ui.create).toHaveBeenCalledOnce();
  await escape();
  await clickOutside();
  expect(button("common.cancel").disabled).toBe(true);
  await act(async () => button("common.cancel").click());
  await settle();
  expect([closed.mock.calls.length, dialog() === null]).toEqual([0, false]);
  // the focus stays in the dialog
  expect(dialog()?.contains(document.activeElement)).toBe(true);
  await answer(made("Mine"));
  expect([closed.mock.calls.length, dialog()]).toEqual([1, null]);
});

it("takes an Escape with its icon picker open as the picker's: the picker closes, the dialog stays", async () => {
  await open(root);
  const panel = await openIconPicker();
  await act(async () => panel.querySelector<HTMLElement>("button")?.focus());
  await escape();
  expect(document.querySelector('[data-slot="popover-content"]')).toBeNull();
  expect([closed.mock.calls.length, dialog() === null]).toEqual([0, false]);
  // a second Escape, the picker closed, is the dialog's
  await escape();
  expect([closed.mock.calls.length, dialog()]).toEqual([1, null]);
});

it("takes an Escape that ends an input method's composition in the name as the method's: the dialog stays", async () => {
  await open(root);
  await fill("name", "視");
  await act(async () => field("name").focus());
  await escape(true);
  expect([closed.mock.calls.length, dialog() === null]).toEqual([0, false]);
  expect(field("name").value).toBe("視");
});

it("makes the view, then closes, says so and shows the new view's page, only once nerve has made it", async () => {
  await open(root);
  await fill("name", "Mine");
  await submit();
  await submit();
  expect(ui.create).toHaveBeenCalledOnce();
  expect([closed.mock.calls.length, toasts, address()]).toEqual([0, [], list]);
  await answer(made("Mine"));
  expect(closed).toHaveBeenCalledOnce();
  expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "project_views.action.create.success" }]);
  expect(address()).toBe(`${list}/v-made`);
});

it("changes the view, then closes and says so, staying where it is, only once nerve has changed it", async () => {
  await open(root, { viewId: bugs.id });
  await fill("name", "Bugs v2");
  await submit();
  expect(ui.update).toHaveBeenCalledOnce();
  expect([closed.mock.calls.length, toasts]).toEqual([0, []]);
  await answer({ ...bugs, name: "Bugs v2" });
  expect(closed).toHaveBeenCalledOnce();
  expect(toasts).toEqual([{ type: "success", title: "toast.success", message: "project_views.action.update.success" }]);
  expect(address()).toBe(list);
});

it.each([
  { change: "a new view", viewId: undefined, status: 409, code: "navigation.views_disabled" },
  { change: "a change", viewId: bugs.id, status: 404, code: "navigation.view_not_found" },
  { change: "a change", viewId: bugs.id, status: 403, code: "forbidden" },
])("keeps the dialog and what it holds when nerve refuses $change ($code), saying why", async (each) => {
  await open(root, { viewId: each.viewId });
  await fill("name", "Release");
  await submit();
  await refuse(refusal(each.status, each.code));
  expect([closed.mock.calls.length, dialog() === null, field("name").value]).toEqual([0, false, "Release"]);
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: `errors.${each.code.replace(".", "_")}` }]);
  expect(button("common.cancel").disabled).toBe(false);
});

it.each([
  { settles: "answered", viewId: undefined, settle: (change: HeldChange<View>) => change.answer(made("Mine")) },
  { settles: "answered", viewId: bugs.id, settle: (change: HeldChange<View>) => change.answer(bugs) },
  {
    settles: "refused",
    viewId: undefined,
    settle: (change: HeldChange<View>) => change.refuse(refusal(409, "navigation.views_disabled")),
  },
])(
  "follows nothing of what it sent that nerve $settles once it has gone with its page (view $viewId)",
  async (each) => {
    await open(root, { viewId: each.viewId });
    await fill("name", "Mine");
    await submit();
    await act(async () => root.render(null));
    await act(async () => each.settle(sent));
    await settle();
    expect([closed.mock.calls.length, toasts]).toEqual([0, []]);
  }
);

it.each([
  {
    change: "a new view",
    viewId: undefined,
    settles: "answers",
    settle: (c: HeldChange<View>) => c.answer(made("Mine")),
  },
  {
    change: "a new view",
    viewId: undefined,
    settles: "refuses",
    settle: (c: HeldChange<View>) => c.refuse(refusal(409, "navigation.views_disabled")),
  },
  { change: "a change", viewId: bugs.id, settles: "answers", settle: (c: HeldChange<View>) => c.answer(bugs) },
  {
    change: "a change",
    viewId: bugs.id,
    settles: "refuses",
    settle: (c: HeldChange<View>) => c.refuse(refusal(404, "navigation.view_not_found")),
  },
])("follows nothing of $change nerve $settles after another tab moved this one to another account", async (each) => {
  await open(root, { viewId: each.viewId });
  await fill("name", "Mine");
  await submit();
  switchAccount();
  await act(async () => each.settle(sent));
  await settle();
  expect([closed.mock.calls.length, toasts, address()]).toEqual([0, [], list]);
});

it("is the new view's dialog that create_view opens on the project's dialogs' host, which it closes", async () => {
  const palette = new CommandPaletteStore();
  ui.palette = palette;
  await act(async () =>
    root.render(
      <MemoryRouter>
        <ProjectLevelModals workspaceSlug="acme" projectId={WEB} />
      </MemoryRouter>
    )
  );
  await act(async () => palette.toggleCreateViewModal(true));
  await settle();
  expect([title(), field("name").value]).toEqual(["view.create.label", ""]);
  await act(async () => button("common.cancel").click());
  await settle();
  expect([palette.isCreateViewModalOpen, dialog()]).toEqual([false, null]);
});
