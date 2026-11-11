// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Cycle } from "@nerve/api-client";
import { signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { cycleOf } from "@/store/project/fake-projects";
import { callerInWeb, its } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { CyclePages, choose, dialog, dialogButton, row, settle, standInCycles, where } from "./fake-cycle-pages";

// What a cycle's page does with what its quick actions ask (P5a spec 2.5, R12): deleting, archiving and restoring the
// cycle. The page follows nerve's answer: it closes the dialog, says so and goes where the change takes it only once
// nerve has made the change, though the change takes the cycle's row away first; refused, the dialog stays and the page
// says nerve's reason, never success; and it follows nothing once another page or another account has taken it. The
// pages and their cycles are fake-cycle-pages.tsx's; the caller is web's admin (fake-permissions.ts).

const held = vi.hoisted(() => ({
  cycles: undefined as unknown as ReturnType<typeof standInCycles>,
  permissions: undefined as IUserPermissionStore | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-cycles", () => ({ useCycles: () => held.cycles }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => held.permissions,
  useUser: () => ({ data: { id: "me" } }),
}));
vi.mock("./modal", () => ({ CycleCreateUpdateModal: () => null }));

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

const ARCHIVED = "2026-11-01T09:00:00Z";

/**
 * Web's admin at a page of web's cycles (page: "cycles", "cycles/<id>" or "archives/cycles", then its query; <id>
 * stands for Sprint's id), which holds Sprint, of fields, and Retro, upcoming. Gives the two, and where the pages are.
 */
async function at(page: string, fields: Partial<Cycle>) {
  const { permissions, acme, web } = await callerInWeb(its.admin.workspaceRole, its.admin.projectRole);
  held.permissions = permissions;
  const sprint = cycleOf(web, "Sprint", fields);
  const retro = cycleOf(web, "Retro", { status: "upcoming" });
  held.cycles = standInCycles([sprint, retro]);
  const project = `/${acme.slug}/projects/${web.id}`;
  await act(async () => root.render(<CyclePages path={`${project}/${page.replace("<id>", sprint.id)}`} />));
  await settle();
  return { sprint, retro, project };
}
const address = () => `${where.location?.pathname}${where.location?.search}`;
const title = () => document.getElementById(dialog()?.getAttribute("aria-labelledby") ?? "")?.textContent;
const positive = () =>
  [...(dialog()?.querySelectorAll<HTMLElement>("[tabindex]") ?? [])].filter((el) => el.tabIndex > 0);
const press = async (text: string) => {
  await act(async () => dialogButton(text)?.click());
  await settle();
};
const escape = async () => {
  await act(async () =>
    (document.activeElement ?? document.body).dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true })
    )
  );
  await settle();
};
/** Leaves the page as a Back would, for a page of another project's. */
const leave = async () => {
  await act(async () => where.navigate?.("/acme/projects/elsewhere/issues"));
  await settle();
};

const upcoming = { status: "upcoming" } as const;
const ended = { status: "completed" } as const;
const deleted = {
  type: "success",
  title: "project_cycles.action.delete.success.title",
  message: "project_cycles.action.delete.success.description",
};
const archivedSaid = {
  type: "success",
  title: "project_cycles.action.archive.success.title",
  message: "project_cycles.action.archive.success.description",
};
const restored = {
  type: "success",
  title: "project_cycles.action.restore.success.title",
  message: "project_cycles.action.restore.success.description",
};

it.each([
  { page: "cycles", stays: true, after: "stays" },
  { page: "cycles?peekCycle=<id>", stays: false, after: "goes back to the list" },
  { page: "cycles/<id>", stays: false, after: "goes back to the list" },
  { page: "cycles?peekCycle=c-WEB-Retro", stays: true, after: "stays, another cycle's preview open" },
])(
  "deletes the cycle from $page, closing and saying so once nerve has, though its row has gone first; then $after",
  async ({ page, stays }) => {
    const { sprint, project } = await at(page, upcoming);
    const before = address();
    await choose(sprint.id, "delete");
    expect(title()).toBe("project_cycles.delete.title");
    expect(dialog()?.textContent).toContain("project_cycles.delete.body Sprint");
    expect(positive()).toEqual([]);
    await press("delete");
    expect(held.cycles.remove.mock.calls).toEqual([[sprint.id]]);
    expect([dialog() === null, toasts]).toEqual([false, []]);
    await held.cycles.answer();
    expect(row(sprint.id)).toBeNull();
    expect(toasts).toEqual([deleted]);
    expect(dialog()).toBeNull();
    expect(address()).toBe(stays ? before : `${project}/cycles`);
  }
);

it.each([
  { what: "deletion", page: "cycles", fields: upcoming, item: "delete", sent: "remove" },
  { what: "archive", page: "cycles", fields: ended, item: "archive", sent: "archive" },
] as const)("lets nothing close the dialog while its $what is out", async ({ page, fields, item, sent }) => {
  const { sprint } = await at(page, fields);
  await choose(sprint.id, item);
  await press(item);
  expect(held.cycles[sent]).toHaveBeenCalledOnce();
  await escape();
  expect(dialogButton("cancel")?.disabled).toBe(true);
  expect(dialog()).not.toBeNull();
});

it.each([
  { what: "deletion", fields: upcoming, item: "delete", error: refusal(403, "forbidden"), why: "errors.forbidden" },
  {
    what: "deletion",
    fields: upcoming,
    item: "delete",
    error: refusal(409, "planning.cycles_disabled"),
    why: "errors.planning_cycles_disabled",
  },
  {
    what: "archive",
    fields: ended,
    item: "archive",
    error: refusal(409, "planning.cycle_not_completed"),
    why: "errors.planning_cycle_not_completed",
  },
] as const)("keeps the dialog when nerve refuses its $what, saying why, never success", async (refused) => {
  const { sprint, project } = await at("cycles/<id>", refused.fields);
  await choose(sprint.id, refused.item);
  await press(refused.item);
  await held.cycles.refuse(refused.error);
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: refused.why }]);
  expect(dialog()).not.toBeNull();
  expect(row(sprint.id)).not.toBeNull();
  expect(address()).toBe(`${project}/cycles/${sprint.id}`);
  // sent again, made: it closes then, and only then says so
  await press(refused.item);
  await held.cycles.answer();
  expect(toasts.map(({ type }) => type)).toEqual(["error", "success"]);
  expect(dialog()).toBeNull();
});

it.each(["cycles?peekCycle=c-WEB-Retro", "cycles/<id>"])(
  "archives an ended cycle from %s, closing, saying so and going to the list once nerve has",
  async (page) => {
    const { sprint, project } = await at(page, ended);
    await choose(sprint.id, "archive");
    expect(title()).toBe("project_cycles.archive.title Sprint");
    expect(dialog()?.textContent).toContain("project_cycles.archive.body");
    expect(positive()).toEqual([]);
    await press("archive");
    expect(held.cycles.archive.mock.calls).toEqual([[sprint.id]]);
    expect([dialog() === null, toasts]).toEqual([false, []]);
    await held.cycles.answer();
    // its row has gone, from the list or with the page it was on
    expect(row(sprint.id)).toBeNull();
    expect(toasts).toEqual([archivedSaid]);
    expect(dialog()).toBeNull();
    expect(address()).toBe(`${project}/cycles`);
  }
);

it.each(["archives/cycles?peekCycle=<id>", "cycles/<id>"])(
  "restores an archived cycle from %s, saying so once nerve has, and goes to the archive (its preview closed)",
  async (page) => {
    const { sprint, project } = await at(page, { ...ended, archived_at: ARCHIVED });
    await choose(sprint.id, "restore");
    expect(held.cycles.unarchive.mock.calls).toEqual([[sprint.id]]);
    expect(toasts).toEqual([]);
    await held.cycles.answer();
    // its row has gone, from the archive or with the page it was on
    expect(row(sprint.id)).toBeNull();
    expect(toasts).toEqual([restored]);
    expect(address()).toBe(`${project}/archives/cycles`);
  }
);

it("restores the cycle once while its restore is out, though asked again by its button's menu and its right-click menu", async () => {
  const { sprint, project } = await at("archives/cycles", { ...ended, archived_at: ARCHIVED });
  await choose(sprint.id, "restore");
  // asked again while nerve has not answered: by the button's menu, then by the row's right-click menu
  await choose(sprint.id, "restore");
  await act(async () => row(sprint.id)?.dispatchEvent(new MouseEvent("contextmenu", { bubbles: true })));
  await settle();
  const again = [
    ...(row(sprint.id)?.querySelectorAll<HTMLButtonElement>('[data-context-menu="true"] [role="menuitem"]') ?? []),
  ].find((item) => item.querySelector("h5")?.textContent === "restore");
  await act(async () => again?.click());
  await settle();
  expect(held.cycles.unarchive.mock.calls).toEqual([[sprint.id]]);
  await held.cycles.answer();
  expect(toasts).toEqual([restored]);
  expect(address()).toBe(`${project}/archives/cycles`);
});

it("restores the cycle again once nerve has refused its restore", async () => {
  const { sprint } = await at("archives/cycles", { ...ended, archived_at: ARCHIVED });
  await choose(sprint.id, "restore");
  await held.cycles.refuse(refusal(409, "planning.cycles_disabled"));
  await choose(sprint.id, "restore");
  expect(held.cycles.unarchive.mock.calls).toEqual([[sprint.id], [sprint.id]]);
});

it("says nerve's refusal of a restore, never success, and stays", async () => {
  const { sprint, project } = await at("cycles/<id>", { ...ended, archived_at: ARCHIVED });
  await choose(sprint.id, "restore");
  await held.cycles.refuse(refusal(409, "planning.cycles_disabled"));
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.planning_cycles_disabled" }]);
  expect(address()).toBe(`${project}/cycles/${sprint.id}`);
});

const late = [
  { change: "deletion", fields: upcoming, item: "delete", confirm: true, sent: "remove" },
  { change: "archive", fields: ended, item: "archive", confirm: true, sent: "archive" },
  {
    change: "restore",
    fields: { ...ended, archived_at: ARCHIVED },
    item: "restore",
    confirm: false,
    sent: "unarchive",
  },
] as const;
const settlings = [
  { settles: "answered", settle: () => held.cycles.answer() },
  { settles: "refused", settle: () => held.cycles.refuse(refusal(403, "forbidden")) },
];
/** Each change, settled each way. */
const lateSettlings = late.flatMap((change) =>
  settlings.map(({ settles, settle: settleIt }) => Object.assign({ settles, settle: settleIt }, change))
);

it.each(lateSettlings)(
  "follows nothing of a $change $settles once a Back has left its page",
  async ({ fields, item, confirm, sent, settle: settleIt }) => {
    const { sprint } = await at("cycles/<id>", fields);
    await choose(sprint.id, item);
    if (confirm) await press(item);
    expect(held.cycles[sent]).toHaveBeenCalledOnce();
    await leave();
    const left = address();
    await settleIt();
    expect(toasts).toEqual([]);
    expect(address()).toBe(left);
  }
);

it.each(lateSettlings)(
  "follows nothing of a $change $settles after another tab moved this one to another account",
  async ({ fields, item, confirm, sent, settle: settleIt }) => {
    const { sprint, project } = await at("cycles/<id>", fields);
    await choose(sprint.id, item);
    if (confirm) await press(item);
    expect(held.cycles[sent]).toHaveBeenCalledOnce();
    switchAccount();
    await settleIt();
    expect(toasts).toEqual([]);
    expect(address()).toBe(`${project}/cycles/${sprint.id}`);
  }
);
