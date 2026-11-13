// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useState } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Module, Project, ProjectRole } from "@nerve/api-client";
import { heldChange, signedIn, switchAccount, type HeldChange } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { moduleOf, projectOf } from "@/store/project/fake-projects";
import { permissionsOver } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { emptyPickers, pickers } from "./fake-module-dialog";
import { CreateUpdateModuleModal } from "./modal";

// The module's dialog as a person uses it (P6a spec 2.7): its real dialog and form, its status's real select and the
// real members' dropdowns, the modules of a stand-in store whose changes nerve settles when the test says
// (fake-tab.ts), the caller's permissions the store's own (fake-permissions.ts), and stand-ins for the project's and
// the dates' pickers and for the members' lists and store (fake-module-dialog.tsx), whose props the test reads and
// picks with.
const ui = vi.hoisted(() => ({
  modules: new Map<string, Module>(),
  create: vi.fn(),
  update: vi.fn(),
  permissions: undefined as IUserPermissionStore | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-modules", () => ({
  useModules: () => ({ getModuleById: (id: string) => ui.modules.get(id), create: ui.create, update: ui.update }),
}));
vi.mock("@/hooks/store/user/user-permissions", () => ({ useUserPermissions: () => ui.permissions }));
vi.mock("@/components/dropdowns/project/dropdown", () => import("./fake-module-dialog"));
vi.mock("@/components/dropdowns/date-range", () => import("./fake-module-dialog"));
vi.mock("@/components/dropdowns/member/base", () => import("./fake-module-dialog"));
vi.mock("@/hooks/store/use-member", () => import("./fake-module-dialog"));
vi.mock("@/components/issues/issue-detail-widgets/directory-read", () => import("./fake-module-dialog"));

let a: Project;
let auth: Module;
const close = vi.fn();
let sent: HeldChange<Module>;
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers({ now: new Date(2026, 10, 13, 12) });
  // the caller, a member of acme, writes the modules of A (its member) and B (its admin); not of G, where he is a
  // guest whom nerve shows only his own work items, nor of S, a guest's whom S shows every work item, nor of M, whose
  // modules are turned off
  const acme = workspaceOf("acme", { role: 15 });
  const project = (id: string, member_role: ProjectRole, fields: Partial<Project> = {}) =>
    projectOf(id, acme.id, { id, member_role, ...fields });
  a = project("A", 15);
  const projects = [
    a,
    project("B", 20),
    project("G", 5),
    project("S", 5, { guest_view_all_features: true }),
    project("M", 15, { module_view: false }),
  ];
  ui.permissions = (await permissionsOver([acme], projects)).permissions;
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
  emptyPickers();
  auth = moduleOf(a, "Auth", {
    id: "m1",
    description: "Sign in",
    start_date: "2026-11-16",
    target_date: "2026-11-27",
    status: "planned",
    lead_id: "u-ann",
    member_ids: ["u-ann", "u-bob"],
  });
  ui.modules = new Map([[auth.id, auth]]);
  sent = heldChange<Module>();
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

/** A page with the control that opens the dialog, of project A: a new module's, or the change of the module of id. */
function Page({ moduleId }: { moduleId?: string }) {
  const [shown, setShown] = useState(false);
  return (
    <>
      <button type="button" onClick={() => setShown(true)}>
        opener
      </button>
      <CreateUpdateModuleModal
        isOpen={shown}
        handleClose={() => {
          close();
          setShown(false);
        }}
        workspaceSlug="acme"
        projectId="A"
        moduleId={moduleId}
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
async function open(moduleId?: string) {
  await act(async () => root.render(<Page moduleId={moduleId} />));
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
const pick = (from?: Date, to?: Date) => act(async () => pickers.dates!.onSelect!({ from, to }));
/** Picks in the members' dropdown of its placeholder ("lead", "members"), as its list would. */
const choose = (placeholder: string, value: string | string[] | null) =>
  act(async () => (pickers.members[placeholder].onChange as (picked: typeof value) => void)(value));
/** The status's select: opens its list and picks the status of its label. */
async function pickStatus(label: string) {
  const select = button(`project_modules.status.${auth.status}`);
  await act(async () => {
    select.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true }));
    select.click();
  });
  await settle();
  const option = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find(
    (item) => item.textContent === label
  );
  await act(async () => option!.click());
  await settle();
}
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
const answer = async (module: Module) => {
  await act(async () => sent.answer(module));
  await settle();
};
const refuse = async (error: unknown) => {
  await act(async () => sent.refuse(error));
  await settle();
};
const title = () => document.getElementById(dialog()!.getAttribute("aria-labelledby") ?? "")?.textContent;

it("creates a module with its name alone, in the backlog, no other key; closing and saying so once nerve has made it", async () => {
  await open();
  await fill("name", "Search v2");
  await submit();
  await submit();
  expect(ui.create.mock.calls).toStrictEqual([["A", { name: "Search v2", status: "backlog" }]]);
  expect(close).not.toHaveBeenCalled();
  expect(toasts).toEqual([]);
  await answer(moduleOf(a, "Search v2", { id: "m2" }));
  expect(close).toHaveBeenCalledOnce();
  expect(toasts).toEqual([
    {
      type: "success",
      title: "project_module.action.create.success.title",
      message: "project_module.action.create.success.description",
    },
  ]);
  expect(dialog()).toBeNull();
  // the focus is back on the control that opened the dialog
  expect(document.activeElement).toBe(opener());
});

it("creates a module in a project whose modules the caller writes: its description, days, lead and members", async () => {
  await open();
  expect(pickers.project?.value).toBe("A");
  expect(["A", "B", "G", "S", "M"].map((id) => pickers.project?.renderCondition?.(id))).toEqual([
    true,
    true,
    false,
    false,
    false,
  ]);
  await act(async () => pickers.project!.onChange("B"));
  await fill("name", "Checkout");
  await fill("description", "Pay and ship");
  await pick(new Date(2026, 10, 16), new Date(2026, 10, 27));
  await choose("lead", "u-cat");
  await choose("members", ["u-cat"]);
  await submit();
  expect(ui.create.mock.calls).toStrictEqual([
    [
      "B",
      {
        name: "Checkout",
        status: "backlog",
        description: "Pay and ship",
        start_date: "2026-11-16",
        target_date: "2026-11-27",
        lead_id: "u-cat",
        member_ids: ["u-cat"],
      },
    ],
  ]);
});

it("creates a module of one day picked alone: that day's key, not the other's", async () => {
  await open();
  await fill("name", "Spike");
  await pick(new Date(2026, 10, 20), undefined);
  await submit();
  expect(ui.create.mock.calls).toStrictEqual([["A", { name: "Spike", status: "backlog", start_date: "2026-11-20" }]]);
});

it("lists the chosen project's members who are not its guests as its lead and members, and clears them as the project changes", async () => {
  await open();
  expect([pickers.members.lead?.memberIds, pickers.members.members?.memberIds]).toEqual([
    ["u-ann", "u-bob"],
    ["u-ann", "u-bob"],
  ]);
  await choose("lead", "u-ann");
  await choose("members", ["u-ann", "u-bob"]);
  await act(async () => pickers.project!.onChange("B"));
  expect([pickers.members.lead?.value, pickers.members.members?.value]).toEqual([null, []]);
  expect([pickers.members.lead?.memberIds, pickers.members.members?.memberIds]).toEqual([["u-cat"], ["u-cat"]]);
  await fill("name", "Checkout");
  await submit();
  expect(ui.create.mock.calls).toStrictEqual([["B", { name: "Checkout", status: "backlog" }]]);
});

it.each([
  ["its name", async () => fill("name", "Auth v2"), { name: "Auth v2" }],
  ["its description, cleared", async () => fill("description", ""), { description: "" }],
  [
    "its target day, sent with its start",
    async () => pick(new Date(2026, 10, 16), new Date(2026, 10, 30)),
    { start_date: "2026-11-16", target_date: "2026-11-30" },
  ],
  ["its days, cleared together", async () => pick(undefined, undefined), { start_date: null, target_date: null }],
  ["its status", async () => pickStatus("project_modules.status.paused"), { status: "paused" as const }],
  ["its lead, cleared", async () => choose("lead", null), { lead_id: null }],
  ["its members, all of them", async () => choose("members", ["u-bob"]), { member_ids: ["u-bob"] }],
])("changes %s alone: the fields changed are all it sends", async (_, change, body) => {
  await open("m1");
  expect(title()).toBe("project_module.update_module");
  // a module stays in its project: no project's picker
  expect(pickers.project).toBeUndefined();
  expect(field("name").value).toBe("Auth");
  await change();
  await submit();
  expect(ui.update.mock.calls).toStrictEqual([["m1", body]]);
  expect(close).not.toHaveBeenCalled();
  await answer({ ...auth, ...body });
  expect(close).toHaveBeenCalledOnce();
  expect(toasts.map(({ type, message }) => [type, message])).toEqual([
    ["success", "project_module.action.update.success.description"],
  ]);
});

it("sends its members unchanged as nothing, though picked in another order", async () => {
  await open("m1");
  await choose("members", ["u-bob", "u-ann"]);
  await fill("name", "Auth v3");
  await submit();
  expect(ui.update.mock.calls).toStrictEqual([["m1", { name: "Auth v3" }]]);
});

it.each([
  ["a name another module has", refusal(409, "planning.module_name_taken"), "errors.planning_module_name_taken"],
  ["of a project whose modules are off", refusal(409, "planning.modules_disabled"), "errors.planning_modules_disabled"],
  ["of a project archived", refusal(409, "project.archived"), "errors.project_archived"],
  ["it may not make", refusal(403, "forbidden"), "errors.forbidden"],
  ["while nerve is busy", refusal(503, "server_busy"), "errors.server_busy"],
])(
  "keeps the dialog and what it holds when nerve refuses a module %s, saying why in a toast",
  async (_, error, why) => {
    await open();
    await fill("name", "Auth");
    await pick(new Date(2026, 10, 16), new Date(2026, 10, 27));
    await choose("lead", "u-bob");
    await submit();
    await refuse(error);
    expect(close).not.toHaveBeenCalled();
    expect(dialog()).not.toBeNull();
    // the form is not reset: what it held is still there
    expect(field("name").value).toBe("Auth");
    expect(pickers.dates?.value.from?.toDateString()).toBe(new Date(2026, 10, 16).toDateString());
    expect(pickers.members.lead?.value).toBe("u-bob");
    expect(toasts).toEqual([{ type: "error", title: "toast.error", message: why }]);
    // sent again, made: it closes then, and only then says so
    sent = heldChange<Module>();
    await submit();
    expect(ui.create).toHaveBeenCalledTimes(2);
    expect(ui.create.mock.calls[1]).toStrictEqual(ui.create.mock.calls[0]);
    await answer(moduleOf(a, "Auth"));
    expect(close).toHaveBeenCalledOnce();
    expect(toasts.map(({ type }) => type)).toEqual(["error", "success"]);
  }
);

it.each([
  ["name", "too_long", () => field("name")],
  ["start_date", "not_allowed", () => button("dates")],
  ["target_date", "required", () => button("dates")],
  ["lead_id", "not_allowed", () => button("lead")],
  ["member_ids[1]", "not_allowed", () => button("members")],
])(
  "says nerve's refusal of the %s under its field, which takes the focus, invalid and described",
  async (name, code, control) => {
    await open();
    await fill("name", "Auth 4");
    await choose("members", ["u-ann", "u-bob"]);
    // the focus elsewhere than on the field nerve will refuse
    await act(async () => (name === "name" ? button("dates") : field("name")).focus());
    await submit();
    await refuse(refusal(422, "validation_failed", [{ field: name, code: code as "required" }]));
    expect(close).not.toHaveBeenCalled();
    expect(toasts).toEqual([]);
    expect(field("name").value).toBe("Auth 4");
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
  expect(title()).toBe("project_module.create_module");
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
  await fill("name", "Auth 5");
  await submit();
  await escape();
  await clickOutside();
  expect(button("common.cancel").disabled).toBe(true);
  expect(close).not.toHaveBeenCalled();
  expect(dialog()).not.toBeNull();
});

it.each([
  ["is answered", (change: HeldChange<Module>) => change.answer(auth)],
  ["is refused", (change: HeldChange<Module>) => change.refuse(refusal(409, "planning.module_name_taken"))],
])("follows nothing of a change that %s once its dialog has gone with its project's page", async (_, settleIt) => {
  await open();
  await fill("name", "Auth 6");
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
  await fill("name", "Auth 7");
  await submit();
  switchAccount();
  await answer(auth);
  expect(close).not.toHaveBeenCalled();
  expect(toasts).toEqual([]);
});
