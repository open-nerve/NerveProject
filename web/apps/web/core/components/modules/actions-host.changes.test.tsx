// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Module } from "@nerve/api-client";
import { choose, dialog, dialogButton, row, settle, where } from "@/components/common/fake-quick-actions";
import { signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { moduleOf } from "@/store/project/fake-projects";
import { callerInWeb, its } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { ModulePages, standInModules } from "./fake-module-pages";

// What a module's page does with what its quick actions ask (P6a spec 2.6, 2.7, R11): deleting, archiving and
// restoring the module. The page follows nerve's answer: it closes the dialog, says so and goes where the change takes
// it only once nerve has made the change, though the change takes the module's row away first; refused, the dialog
// stays and the page says nerve's reason, never success; and it follows nothing once another page or another account
// has taken it. The pages and their modules are fake-module-pages.tsx's; the caller is web's admin (fake-permissions.ts).

const held = vi.hoisted(() => ({
  modules: undefined as unknown as ReturnType<typeof standInModules>,
  permissions: undefined as IUserPermissionStore | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => held.modules }));
vi.mock("@/hooks/store/user", () => ({
  useUserPermissions: () => held.permissions,
  useUser: () => ({ data: { id: "me" } }),
}));
vi.mock("./modal", () => ({ CreateUpdateModuleModal: () => null }));

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
 * Web's admin at a page of web's modules (page: "modules", "modules/<id>" or "archives/modules", then its query; <id>
 * stands for Auth's id), which holds Auth, of fields, and Docs, planned. Gives the two, and where the pages are.
 */
async function at(page: string, fields: Partial<Module>) {
  const { permissions, acme, web } = await callerInWeb(its.admin.workspaceRole, its.admin.projectRole);
  held.permissions = permissions;
  const auth = moduleOf(web, "Auth", fields);
  const docs = moduleOf(web, "Docs", { status: "planned" });
  held.modules = standInModules([auth, docs]);
  const project = `/${acme.slug}/projects/${web.id}`;
  await act(async () => root.render(<ModulePages path={`${project}/${page.replace("<id>", auth.id)}`} />));
  await settle();
  return { auth, docs, project };
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

const planned = { status: "planned" } as const;
const completed = { status: "completed" } as const;
const deleted = {
  type: "success",
  title: "project_module.action.delete.success.title",
  message: "project_module.action.delete.success.description",
};
const archivedSaid = {
  type: "success",
  title: "project_module.action.archive.success.title",
  message: "project_module.action.archive.success.description",
};
const restored = {
  type: "success",
  title: "project_module.action.restore.success.title",
  message: "project_module.action.restore.success.description",
};

it.each([
  { page: "modules", stays: true, after: "stays" },
  { page: "modules?peekModule=<id>", stays: false, after: "goes back to the list" },
  { page: "modules/<id>", stays: false, after: "goes back to the list" },
  { page: "modules?peekModule=m-WEB-Docs", stays: true, after: "stays, another module's preview open" },
])(
  "deletes the module from $page, closing and saying so once nerve has, though its row has gone first; then $after",
  async ({ page, stays }) => {
    const { auth, project } = await at(page, planned);
    const before = address();
    await choose(auth.id, "delete");
    expect(title()).toBe("project_module.delete.title");
    expect(dialog()?.textContent).toContain("project_module.delete.body Auth");
    expect(positive()).toEqual([]);
    await press("delete");
    expect(held.modules.remove.mock.calls).toEqual([[auth.id]]);
    expect([dialog() === null, toasts]).toEqual([false, []]);
    await held.modules.answer();
    expect(row(auth.id)).toBeNull();
    expect(toasts).toEqual([deleted]);
    expect(dialog()).toBeNull();
    expect(address()).toBe(stays ? before : `${project}/modules`);
  }
);

it.each([
  { what: "deletion", fields: planned, item: "delete", sent: "remove" },
  { what: "archive", fields: completed, item: "archive", sent: "archive" },
] as const)("lets nothing close the dialog while its $what is out", async ({ fields, item, sent }) => {
  const { auth } = await at("modules", fields);
  await choose(auth.id, item);
  await press(item);
  expect(held.modules[sent]).toHaveBeenCalledOnce();
  await escape();
  expect(dialogButton("cancel")?.disabled).toBe(true);
  expect(dialog()).not.toBeNull();
});

it.each([
  { what: "deletion", fields: planned, item: "delete", error: refusal(403, "forbidden"), why: "errors.forbidden" },
  {
    what: "deletion",
    fields: planned,
    item: "delete",
    error: refusal(409, "planning.modules_disabled"),
    why: "errors.planning_modules_disabled",
  },
  {
    what: "archive",
    fields: completed,
    item: "archive",
    error: refusal(409, "planning.module_not_closed"),
    why: "errors.planning_module_not_closed",
  },
] as const)("keeps the dialog when nerve refuses its $what, saying why, never success", async (refused) => {
  const { auth, project } = await at("modules/<id>", refused.fields);
  await choose(auth.id, refused.item);
  await press(refused.item);
  await held.modules.refuse(refused.error);
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: refused.why }]);
  expect(dialog()).not.toBeNull();
  expect(row(auth.id)).not.toBeNull();
  expect(address()).toBe(`${project}/modules/${auth.id}`);
  // sent again, made: it closes then, and only then says so
  await press(refused.item);
  await held.modules.answer();
  expect(toasts.map(({ type }) => type)).toEqual(["error", "success"]);
  expect(dialog()).toBeNull();
});

it.each([
  { page: "modules?peekModule=m-WEB-Docs", fields: completed },
  { page: "modules/<id>", fields: { status: "cancelled" } as const },
])(
  "archives a closed module from $page, named by its title, closing, saying so and going to the list once nerve has",
  async ({ page, fields }) => {
    const { auth, project } = await at(page, fields);
    await choose(auth.id, "archive");
    expect(title()).toBe("project_module.archive.title Auth");
    expect(dialog()?.textContent).toContain("project_module.archive.body");
    expect(positive()).toEqual([]);
    await press("archive");
    expect(held.modules.archive.mock.calls).toEqual([[auth.id]]);
    expect([dialog() === null, toasts]).toEqual([false, []]);
    await held.modules.answer();
    // its row has gone, from the list or with the page it was on
    expect(row(auth.id)).toBeNull();
    expect(toasts).toEqual([archivedSaid]);
    expect(dialog()).toBeNull();
    expect(address()).toBe(`${project}/modules`);
  }
);

it.each(["archives/modules?peekModule=<id>", "modules/<id>"])(
  "restores an archived module from %s, saying so once nerve has, and goes to the archive (its preview closed)",
  async (page) => {
    const { auth, project } = await at(page, { ...completed, archived_at: ARCHIVED });
    await choose(auth.id, "restore");
    expect(held.modules.unarchive.mock.calls).toEqual([[auth.id]]);
    expect(toasts).toEqual([]);
    await held.modules.answer();
    // its row has gone, from the archive or with the page it was on
    expect(row(auth.id)).toBeNull();
    expect(toasts).toEqual([restored]);
    expect(address()).toBe(`${project}/archives/modules`);
  }
);

it("restores the module once while its restore is out, though asked again by its button's menu and its right-click menu", async () => {
  const { auth, project } = await at("archives/modules", { ...completed, archived_at: ARCHIVED });
  await choose(auth.id, "restore");
  // asked again while nerve has not answered: by the button's menu, then by the row's right-click menu
  await choose(auth.id, "restore");
  await act(async () => row(auth.id)?.dispatchEvent(new MouseEvent("contextmenu", { bubbles: true })));
  await settle();
  const again = [
    ...(row(auth.id)?.querySelectorAll<HTMLButtonElement>('[data-context-menu="true"] [role="menuitem"]') ?? []),
  ].find((item) => item.querySelector("h5")?.textContent === "restore");
  await act(async () => again?.click());
  await settle();
  expect(held.modules.unarchive.mock.calls).toEqual([[auth.id]]);
  await held.modules.answer();
  expect(toasts).toEqual([restored]);
  expect(address()).toBe(`${project}/archives/modules`);
});

it("restores the module again once nerve has refused its restore", async () => {
  const { auth } = await at("archives/modules", { ...completed, archived_at: ARCHIVED });
  await choose(auth.id, "restore");
  await held.modules.refuse(refusal(409, "planning.modules_disabled"));
  await choose(auth.id, "restore");
  expect(held.modules.unarchive.mock.calls).toEqual([[auth.id], [auth.id]]);
});

it("says nerve's refusal of a restore, never success, and stays", async () => {
  const { auth, project } = await at("modules/<id>", { ...completed, archived_at: ARCHIVED });
  await choose(auth.id, "restore");
  await held.modules.refuse(refusal(409, "planning.modules_disabled"));
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.planning_modules_disabled" }]);
  expect(address()).toBe(`${project}/modules/${auth.id}`);
});

const late = [
  { change: "deletion", fields: planned, item: "delete", confirm: true, sent: "remove" },
  { change: "archive", fields: completed, item: "archive", confirm: true, sent: "archive" },
  {
    change: "restore",
    fields: { ...completed, archived_at: ARCHIVED },
    item: "restore",
    confirm: false,
    sent: "unarchive",
  },
] as const;
const settlings = [
  { settles: "answered", settle: () => held.modules.answer() },
  { settles: "refused", settle: () => held.modules.refuse(refusal(403, "forbidden")) },
];
/** Each change, settled each way. */
const lateSettlings = late.flatMap((change) =>
  settlings.map(({ settles, settle: settleIt }) => Object.assign({ settles, settle: settleIt }, change))
);

it.each(lateSettlings)(
  "follows nothing of a $change $settles once a Back has left its page: nerve's answer reaches the store alone",
  async ({ fields, item, confirm, sent, settle: settleIt }) => {
    const { auth } = await at("modules/<id>", fields);
    await choose(auth.id, item);
    if (confirm) await press(item);
    expect(held.modules[sent]).toHaveBeenCalledOnce();
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
    const { auth, project } = await at("modules/<id>", fields);
    await choose(auth.id, item);
    if (confirm) await press(item);
    expect(held.modules[sent]).toHaveBeenCalledOnce();
    switchAccount();
    await settleIt();
    expect(toasts).toEqual([]);
    expect(address()).toBe(`${project}/modules/${auth.id}`);
  }
);
