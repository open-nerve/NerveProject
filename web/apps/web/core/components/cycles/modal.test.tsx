// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useState, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle } from "@nerve/api-client";
import type { DateRangeDropdown } from "@/components/dropdowns/date-range";
import { heldChange, signedIn, switchAccount, type HeldChange } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { CycleCreateUpdateModal } from "./modal";

// The cycle's dialog as a person uses it (P5a spec 2.6): its real dialog and form, the cycles of a stand-in store whose
// changes nerve settles when the test says (fake-tab.ts), and two stand-in pickers: the project's, which keeps what it
// may list, and the dates', which keeps its props so that the test picks a range as its calendar would.
type Picker = {
  value: string | null;
  onChange: (projectId: string) => void;
  renderCondition?: (id: string) => boolean;
};
const ui = vi.hoisted(() => ({
  cycles: new Map<string, Cycle>(),
  create: vi.fn(),
  update: vi.fn(),
  project: undefined as Picker | undefined,
  dates: undefined as ComponentProps<typeof DateRangeDropdown> | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-cycles", () => ({
  useCycles: () => ({ getCycleById: (id: string) => ui.cycles.get(id), create: ui.create, update: ui.update }),
}));
// the caller writes A (a member) and B (an admin); G, where he is a guest, is not among them
vi.mock("@/hooks/store/user/user-user", () => ({
  useUser: () => ({ projectsWithCreatePermissions: { A: 15, B: 20 } }),
}));
vi.mock("@/components/dropdowns/project/dropdown", () => ({
  ProjectDropdown: (props: Picker) => {
    ui.project = props;
    return <button type="button">{props.value}</button>;
  },
}));
vi.mock("@/components/dropdowns/date-range", () => ({
  DateRangeDropdown: (props: ComponentProps<typeof DateRangeDropdown>) => {
    ui.dates = props;
    // the button the dropdown renders, with what it passes on to it
    const passed = { "aria-invalid": props.invalid, "aria-describedby": props.describedBy, tabIndex: props.tabIndex };
    return (
      <button type="button" {...passed}>
        dates
      </button>
    );
  },
}));

const sprint: Cycle = {
  id: "c1",
  workspace_id: "w",
  project_id: "A",
  name: "Sprint",
  description: "Ship it",
  start_date: "2026-11-16",
  end_date: "2026-11-22",
  status: "upcoming",
  owned_by_id: "u",
  sort_order: 65535,
  archived_at: null,
  created_at: "2026-11-01T09:00:00Z",
  updated_at: "2026-11-01T09:00:00Z",
  created_by_id: "u",
  updated_by_id: "u",
};
const close = vi.fn();
let sent: HeldChange<Cycle>;
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(() => {
  vi.useFakeTimers({ now: new Date(2026, 10, 11, 12) });
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
  close.mockReset();
  ui.cycles = new Map([[sprint.id, sprint]]);
  ui.project = undefined;
  ui.dates = undefined;
  sent = heldChange<Cycle>();
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

/** A page with the control that opens the dialog, of project A: a new cycle's, or the change of the cycle of cycleId. */
function Page({ cycleId }: { cycleId?: string }) {
  const [shown, setShown] = useState(false);
  return (
    <>
      <button type="button" onClick={() => setShown(true)}>
        opener
      </button>
      <CycleCreateUpdateModal
        isOpen={shown}
        handleClose={() => {
          close();
          setShown(false);
        }}
        projectId="A"
        cycleId={cycleId}
      />
    </>
  );
}
const settle = () => act(async () => vi.advanceTimersByTimeAsync(100));
const opener = () => [...host.querySelectorAll("button")].find((button) => button.textContent === "opener")!;
const dialog = () => document.querySelector<HTMLElement>('[role="dialog"]');
const field = (name: string) => document.querySelector<HTMLInputElement | HTMLTextAreaElement>(`[name="${name}"]`)!;
const button = (name: string) =>
  [...(dialog()?.querySelectorAll("button") ?? [])].find((item) => item.textContent?.trim() === name)!;
async function open(cycleId?: string) {
  await act(async () => root.render(<Page cycleId={cycleId} />));
  await act(async () => {
    opener().focus();
    opener().click();
  });
  await settle();
  expect(dialog()).not.toBeNull();
}
async function fill(name: string, value: string) {
  await act(async () => {
    const input = field(name);
    const proto = input instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
    Object.getOwnPropertyDescriptor(proto, "value")!.set!.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
const pick = (from?: Date, to?: Date) => act(async () => ui.dates!.onSelect!({ from, to }));
async function submit() {
  await act(async () =>
    dialog()!
      .querySelector("form")!
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
  await settle();
}
const escape = async () => {
  await act(async () =>
    (document.activeElement ?? document.body).dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true })
    )
  );
  await settle();
};
const answer = async (cycle: Cycle) => {
  await act(async () => sent.answer(cycle));
  await settle();
};
const refuse = async (error: unknown) => {
  await act(async () => sent.refuse(error));
  await settle();
};

it("creates a draft with its name alone, no date keys nor its project's id, closing and saying so once nerve has made it", async () => {
  await open();
  await fill("name", "Sprint 1");
  await submit();
  await submit();
  expect(ui.create.mock.calls).toStrictEqual([["A", { name: "Sprint 1" }]]);
  expect(close).not.toHaveBeenCalled();
  expect(toasts).toEqual([]);
  await answer({ ...sprint, id: "c2", name: "Sprint 1", start_date: null, end_date: null, status: "draft" });
  expect(close).toHaveBeenCalledOnce();
  expect(toasts).toEqual([
    {
      type: "success",
      title: "project_cycles.action.create.success.title",
      message: "project_cycles.action.create.success.description",
    },
  ]);
  expect(dialog()).toBeNull();
  // the focus is back on the control that opened the dialog
  expect(document.activeElement).toBe(opener());
});

it("creates a dated cycle in a project the caller writes: its two calendar days and its description", async () => {
  await open();
  expect(ui.project?.value).toBe("A");
  expect(["A", "B", "G"].map((id) => ui.project?.renderCondition?.(id))).toEqual([true, true, false]);
  // the calendar starts at the browser's today
  expect(ui.dates?.minDate?.toDateString()).toBe(new Date(2026, 10, 11).toDateString());
  await act(async () => ui.project!.onChange("B"));
  await fill("name", "Sprint 2");
  await fill("description", "Ship the release");
  await pick(new Date(2026, 10, 16), new Date(2026, 10, 22));
  await submit();
  expect(ui.create.mock.calls).toStrictEqual([
    ["B", { name: "Sprint 2", description: "Ship the release", start_date: "2026-11-16", end_date: "2026-11-22" }],
  ]);
});

it.each([
  ["its name", async () => fill("name", "Sprint 9"), { name: "Sprint 9" }],
  ["its description", async () => fill("description", ""), { description: "" }],
  [
    "its end, sent with its start",
    async () => pick(new Date(2026, 10, 16), new Date(2026, 10, 29)),
    { start_date: "2026-11-16", end_date: "2026-11-29" },
  ],
  ["its dates, cleared together", async () => pick(undefined, undefined), { start_date: null, end_date: null }],
])("changes %s alone: the fields changed are all it sends", async (_, change, body) => {
  await open("c1");
  const title = document.getElementById(dialog()!.getAttribute("aria-labelledby") ?? "");
  expect(title?.textContent).toBe("project_cycles.update_cycle");
  // a cycle stays in its project: no project's picker
  expect(ui.project).toBeUndefined();
  expect(field("name").value).toBe("Sprint");
  await change();
  await submit();
  expect(ui.update.mock.calls).toStrictEqual([["c1", body]]);
  expect(close).not.toHaveBeenCalled();
  await answer({ ...sprint, ...body });
  expect(close).toHaveBeenCalledOnce();
  expect(toasts.map(({ type, message }) => [type, message])).toEqual([
    ["success", "project_cycles.action.update.success.description"],
  ]);
});

it.each([
  ["overlapping another cycle", refusal(409, "planning.cycle_dates_overlap"), "errors.planning_cycle_dates_overlap"],
  ["of a project whose cycles are off", refusal(409, "planning.cycles_disabled"), "errors.planning_cycles_disabled"],
  ["of a project archived", refusal(409, "project.archived"), "errors.project_archived"],
  ["it may not make", refusal(403, "forbidden"), "errors.forbidden"],
  ["while nerve is busy", refusal(503, "server_busy"), "errors.server_busy"],
])("keeps the dialog and what it holds when nerve refuses a cycle %s, saying why in a toast", async (_, error, why) => {
  await open();
  await fill("name", "Sprint 3");
  await pick(new Date(2026, 10, 16), new Date(2026, 10, 22));
  await submit();
  await refuse(error);
  expect(close).not.toHaveBeenCalled();
  expect(dialog()).not.toBeNull();
  expect(field("name").value).toBe("Sprint 3");
  expect(ui.dates?.value.from?.toDateString()).toBe(new Date(2026, 10, 16).toDateString());
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: why }]);
  // sent again, made: it closes then, and only then says so
  sent = heldChange<Cycle>();
  await submit();
  expect(ui.create).toHaveBeenCalledTimes(2);
  await answer({ ...sprint, name: "Sprint 3" });
  expect(close).toHaveBeenCalledOnce();
  expect(toasts.map(({ type }) => type)).toEqual(["error", "success"]);
});

it.each([
  ["name", "too_long", () => field("name")],
  ["end_date", "required", () => button("dates")],
  ["start_date", "not_allowed", () => button("dates")],
])(
  "says nerve's refusal of the %s under its field, which takes the focus, invalid and described",
  async (name, code, control) => {
    await open();
    await fill("name", "Sprint 4");
    // the focus elsewhere than on the field nerve will refuse
    await act(async () => (name === "name" ? button("dates") : field("name")).focus());
    await submit();
    await refuse(refusal(422, "validation_failed", [{ field: name, code: code as "required" }]));
    expect(close).not.toHaveBeenCalled();
    expect(toasts).toEqual([]);
    expect(field("name").value).toBe("Sprint 4");
    expect(control().getAttribute("aria-invalid")).toBe("true");
    const described = (control().getAttribute("aria-describedby") ?? "").split(/\s+/);
    expect(described.map((id) => document.getElementById(id)?.textContent).join(" ")).toContain(`errors.field.${code}`);
    expect(document.activeElement).toBe(control());
  }
);

it("refuses a name left empty before sending anything, the name invalid and focused", async () => {
  await open();
  await act(async () => button("dates").focus());
  await submit();
  expect(ui.create).not.toHaveBeenCalled();
  expect(field("name").getAttribute("aria-invalid")).toBe("true");
  expect(document.activeElement).toBe(field("name"));
});

it("names itself by its title, gives the name the focus and keeps the natural Tab order", async () => {
  await open();
  const title = document.getElementById(dialog()!.getAttribute("aria-labelledby") ?? "");
  expect(title?.textContent).toBe("project_cycles.create_cycle");
  expect(document.activeElement).toBe(field("name"));
  const positive = [...dialog()!.querySelectorAll<HTMLElement>("[tabindex]")].filter((item) => item.tabIndex > 0);
  expect(positive).toEqual([]);
});

/** A click on the dialog's backdrop, outside its panel, as a pointer gives it (its press and its release). */
async function clickOutside() {
  const backdrop = document.querySelector<HTMLElement>('[class*="bg-backdrop"]');
  expect(backdrop).not.toBeNull();
  await act(async () => {
    for (const type of ["pointerdown", "mousedown", "pointerup", "mouseup"])
      backdrop!.dispatchEvent(new MouseEvent(type, { bubbles: true, cancelable: true }));
    backdrop!.click();
  });
  await settle();
}
async function reopen() {
  await act(async () => opener().click());
  await settle();
  expect(dialog()).not.toBeNull();
}

it("closes on Escape and a click outside through its handleClose, giving the focus back; neither while its change is out, nor Cancel", async () => {
  await open();
  await escape();
  expect(close).toHaveBeenCalledOnce();
  expect(dialog()).toBeNull();
  expect(document.activeElement).toBe(opener());
  await reopen();
  await clickOutside();
  expect(close).toHaveBeenCalledTimes(2);
  expect(dialog()).toBeNull();
  close.mockReset();
  await reopen();
  await fill("name", "Sprint 5");
  await submit();
  await escape();
  await clickOutside();
  expect(button("common.cancel").disabled).toBe(true);
  expect(close).not.toHaveBeenCalled();
  expect(dialog()).not.toBeNull();
});

it.each([
  ["is answered", (change: HeldChange<Cycle>) => change.answer(sprint)],
  ["is refused", (change: HeldChange<Cycle>) => change.refuse(refusal(409, "planning.cycle_dates_overlap"))],
])("follows nothing of a change that %s once its dialog has gone with its project's page", async (_, settleIt) => {
  await open();
  await fill("name", "Sprint 6");
  await submit();
  await act(async () => root.render(null));
  await act(async () => settleIt(sent));
  await settle();
  expect(ui.create).toHaveBeenCalledOnce();
  expect(close).not.toHaveBeenCalled();
  expect(toasts).toEqual([]);
});

it("follows nothing of a change answered after another tab moved this one to another account", async () => {
  await open();
  await fill("name", "Sprint 7");
  await submit();
  switchAccount();
  await answer(sprint);
  expect(close).not.toHaveBeenCalled();
  expect(toasts).toEqual([]);
});
