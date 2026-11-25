// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { View } from "@nerve/api-client";
import { dialog } from "@/components/common/fake-quick-actions";
import { heldChange, signedIn, type HeldChange } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { WEB, field, fill, iconButton, open, openIconPicker, submit } from "./fake-view-dialog";

// What the view's dialog holds and sends (P6a spec 2.4, R28, R30): a view's icon, name and description alone, in the
// natural Tab order; a new view's name empty, a change's the view's; "Add view" sends them with the display nerve gives
// a view made without one, Save as with the content it was given, a change the fields changed alone; a name nerve
// would refuse (blank, or longer than 255 characters, an emoji one character) is refused before anything is sent, and
// one nerve refuses is said under it; either takes the focus. Its views' store is a stand-in (fake-tab.ts's changes).

const ui = vi.hoisted(() => ({ views: new Map<string, View>(), create: vi.fn(), update: vi.fn() }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-views", () => ({
  useViews: () => ({ getViewById: (id: string) => ui.views.get(id), create: ui.create, update: ui.update }),
}));

/** The display nerve gives a view made without one (ViewCreate; navigation.yaml), in nerve's words. */
const CONTRACT_DISPLAY = {
  display_filters: {
    layout: "list",
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
/** web's view Bugs, its own display, filters and icon. */
const bugs: View = {
  id: "v-bugs",
  workspace_id: "w-acme",
  project_id: WEB,
  name: "Bugs",
  description: "Open bugs",
  logo_props: { in_use: "icon", icon: { name: "Bug", color: "#e57a00" } },
  filters: { priority: ["high"] },
  display_filters: { ...CONTRACT_DISPLAY.display_filters, layout: "kanban", group_by: "state_id" },
  display_properties: { ...CONTRACT_DISPLAY.display_properties, start_date: false },
  owned_by_id: "u-me",
  created_at: "2026-11-01T09:00:00Z",
  updated_at: "2026-11-01T09:00:00Z",
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

/** The dialog's controls a person reaches, in the document's order, each by its name. */
const controls = () =>
  [...dialog()!.querySelectorAll<HTMLElement>("button, input, textarea, select, a[href], [role='combobox']")].map(
    (control) => control.getAttribute("aria-label") ?? control.getAttribute("name") ?? control.textContent?.trim()
  );
/** What the change sent asked nerve: the view's id, and the body reckoned in its turn. */
const changeSent = () => ui.update.mock.calls.map(([id, body]) => [id, (body as () => unknown)()]);
/** The text that describes the name: its message, under it. */
const nameMessage = () =>
  (field("name").getAttribute("aria-describedby") ?? "")
    .split(/\s+/)
    .map((id) => document.getElementById(id)?.textContent)
    .join(" ");

it("holds a view's icon, name and description alone, no layout nor display, in the natural Tab order, the name focused", async () => {
  await open(root);
  expect(controls()).toEqual(["aria_labels.view_icon", "title", "description", "common.cancel", "view.create.label"]);
  expect([...dialog()!.querySelectorAll<HTMLElement>("[tabindex]")].filter((item) => item.tabIndex > 0)).toEqual([]);
  expect(document.activeElement).toBe(field("name"));
});

it("names its name and its description by the page's language, not by their placeholders alone (review O5)", async () => {
  await open(root);
  const fields = [...dialog()!.querySelectorAll<HTMLElement>("input[type='text'], textarea")];
  expect(fields.map((one) => [one.getAttribute("name"), one.getAttribute("aria-label")])).toEqual([
    ["name", "title"],
    ["description", "description"],
  ]);
});

it("makes a view of 'Add view' with its name, description and icon and the display nerve gives one made without, no filters", async () => {
  await open(root);
  expect([field("name").value, field("description").value]).toEqual(["", ""]);
  await fill("name", "Mine");
  await submit();
  expect(ui.create.mock.calls).toStrictEqual([
    [WEB, { name: "Mine", description: "", logo_props: {}, ...CONTRACT_DISPLAY }],
  ]);
});

it("makes a view of Save as with the content it was given, its name empty to start with", async () => {
  const content = { filters: { priority: ["high"] }, display_filters: bugs.display_filters };
  await open(root, { content });
  expect([field("name").value, field("description").value]).toEqual(["", ""]);
  await fill("name", "High");
  await fill("description", "The urgent ones");
  await submit();
  expect(ui.create.mock.calls).toStrictEqual([
    [WEB, { name: "High", description: "The urgent ones", logo_props: {}, ...content }],
  ]);
});

/** Picks the icon picker's first icon, Activity, in the colour the picker opens with, the view's icon's. */
async function pickIcon() {
  const panel = await openIconPicker();
  await act(async () => panel.querySelector("svg.lucide-activity")?.closest("button")?.click());
}

it.each([
  ["its name", () => fill("name", "Bugs v2"), { name: "Bugs v2" }],
  ["its description, cleared", () => fill("description", ""), { description: "" }],
  ["its icon", pickIcon, { logo_props: { in_use: "icon" as const, icon: { name: "Activity", color: "#e57a00" } } }],
])("changes %s alone: the fields changed are all it sends, no display", async (_, change, body) => {
  await open(root, { viewId: bugs.id });
  expect([field("name").value, field("description").value]).toEqual(["Bugs", "Open bugs"]);
  await change();
  await submit();
  expect(changeSent()).toStrictEqual([[bugs.id, body]]);
});

it.each([
  ["blank", "   ", "title_is_required"],
  ["256 characters long", "a".repeat(256), "title_should_be_less_than_255_characters"],
])(
  "refuses a name %s before sending anything, saying why under it, which is invalid and takes the focus",
  async (_, name, why) => {
    await open(root);
    await fill("name", name);
    await act(async () => field("description").focus());
    await submit();
    expect(ui.create).not.toHaveBeenCalled();
    expect(field("name").getAttribute("aria-invalid")).toBe("true");
    expect(nameMessage()).toContain(why);
    expect(document.activeElement).toBe(field("name"));
  }
);

it.each([
  ["255 emoji, each one character", "🐛".repeat(255)],
  ["255 characters within white space", `  ${"a".repeat(255)}  `],
])("takes a name of %s, as nerve does", async (_, name) => {
  await open(root);
  await fill("name", name);
  await submit();
  expect(ui.create.mock.calls.map(([, body]) => (body as { name: string }).name)).toEqual([name]);
});

it("says nerve's refusal of the name under it, which is invalid and takes the focus, no toast", async () => {
  await open(root);
  await fill("name", "Mine");
  await act(async () => iconButton().focus());
  await submit();
  await act(async () => sent.refuse(refusal(422, "validation_failed", [{ field: "name", code: "too_long" }])));
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(toasts).toEqual([]);
  expect(field("name").getAttribute("aria-invalid")).toBe("true");
  expect(nameMessage()).toContain("errors.field.too_long");
  expect(document.activeElement).toBe(field("name"));
  expect(field("name").value).toBe("Mine");
});
