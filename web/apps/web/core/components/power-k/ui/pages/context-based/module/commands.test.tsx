// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { observer } from "mobx-react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Module, Project } from "@nerve/api-client";
import type { TPowerKCommandConfig, TPowerKContext } from "@/components/power-k/core/types";
import { listedModules } from "@/components/modules/fake-module-lists";
import { signedIn, switchAccount } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { moduleRowOf } from "@/store/planning/fake-modules";
import { moduleOf } from "@/store/project/fake-projects";
import { callerInWeb, its, type Caller } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { usePowerKModuleContextBasedActions } from "./commands";

// The command palette's commands on a module's page (P6a spec 2.9, R5, R6): its members and its status, changed through
// the module store while nerve takes the caller's writes of the project's modules (canWriteModules) and the module is
// not archived, a refusal said; a change of its members handed to the store as whom it adds or removes, the store
// making nerve's whole set in its turn; its link's copy, the route's module's, through the web app's one copy
// (use-copy-link.ts), to anyone. The modules' favorites are gone until M7 brings them back (M6 design 15 #13). The
// caller's role is the permission store's own (fake-permissions.ts), the module a stand-in store's (fake-module-lists.ts)
// whose changes nerve settles when the test says; the clipboard's write a stand-in the test lets or refuses.

const held = vi.hoisted(() => ({
  params: {} as Record<string, string>,
  modules: undefined as unknown as ReturnType<typeof listedModules>,
  permissions: undefined as IUserPermissionStore | undefined,
  copy: vi.fn(),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("react-router", () => ({ useParams: () => held.params }));
vi.mock("@nerve/utils", async (original) => ({
  ...(await original<typeof import("@nerve/utils")>()),
  copyUrlToClipboard: held.copy,
}));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => held.modules }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => held.permissions }));

let host: HTMLDivElement, root: ReturnType<typeof createRoot>, commands: TPowerKCommandConfig[];
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
const ctx = (): TPowerKContext => ({
  params: held.params,
  activeCommand: null,
  activeContext: "module",
  shouldShowContextBasedActions: true,
  setShouldShowContextBasedActions: () => {},
  navigate: () => {},
  closePalette: () => {},
  setActiveCommand: () => {},
  setActivePage: () => {},
});
const Commands = observer(function Commands() {
  commands = usePowerKModuleContextBasedActions();
  return null;
});
beforeEach(() => {
  signedIn();
  toasts.length = 0;
  held.copy.mockReset().mockResolvedValue(undefined);
  held.modules = listedModules();
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
});

/**
 * The caller of a role in acme and in web, web being as project says, on the page of web's module Auth (as fields say
 * of it, Ann its member), which the store holds: Auth.
 */
async function onPage(caller: Caller, project: Partial<Project> = {}, fields: Partial<Module> = {}) {
  const { permissions, acme, web } = await callerInWeb(
    caller.workspaceRole,
    caller.projectRole,
    caller.guestViewAll,
    project
  );
  const auth = moduleOf(web, "Auth", { member_ids: ["u-ann"], ...fields });
  held.permissions = permissions;
  held.modules.show(auth.archived_at ? "archive" : "modules", web.id, [moduleRowOf(auth)]);
  held.params = { workspaceSlug: acme.slug, projectId: web.id, moduleId: auth.id };
  await act(async () => root.render(<Commands />));
  return auth;
}

/** The command of id, as the palette finds it. */
const command = (id: string) => {
  const found = commands.find((one) => one.id === id);
  if (!found) throw new Error(`no command ${id}`);
  return found;
};
/** The palette runs the action of id, as it does once chosen. */
const run = (id: string) =>
  act(async () => {
    const found = command(id);
    if (found.type !== "action") throw new Error(`no action ${id}`);
    await found.action(ctx());
  });
/** The palette gives the page's choice, value, to the command of id, as it does once chosen on its page. */
const choose = (id: string, value: unknown) =>
  act(async () => {
    const found = command(id);
    if (found.type !== "change-page") throw new Error(`no page ${id}`);
    void found.onSelect(value, ctx());
  });
const offered = () => commands.map((one) => [one.id, one.isVisible(ctx()), one.isEnabled(ctx())]);

it("gives a module's page its members' and its status's changes and its link's copy, to its member: no favorite", async () => {
  await onPage(its.member);
  expect(offered()).toEqual([
    ["add_remove_module_members", true, true],
    ["change_module_status", true, true],
    ["copy_module_url", true, true],
  ]);
});

it.each([
  ["its guest", its.guest, {}, {}],
  ["its guest who is the workspace's admin, web showing its guests only their own work items", its.guestAdmin, {}, {}],
  ["its admin, the project's modules turned off", its.admin, { module_view: false }, {}],
  ["its admin, the module archived", its.admin, {}, { archived_at: "2026-10-31T00:00:00Z" }],
] as const)("offers no change of the module to %s: its link's copy alone", async (_, caller, project, fields) => {
  await onPage(caller, project, fields);
  expect(offered()).toEqual([
    ["add_remove_module_members", false, false],
    ["change_module_status", false, false],
    ["copy_module_url", true, true],
  ]);
});

it("adds or removes the member chosen as a change of whom, the store making nerve's whole set in its turn", async () => {
  const auth = await onPage(its.member);
  await choose("add_remove_module_members", "u-bob");
  // a second choice while the first is out: each is the change of one member, not a whole set of the module's now
  await choose("add_remove_module_members", "u-ann");
  expect(held.modules.changeMembers.mock.calls).toEqual([
    [auth.id, { add: ["u-bob"], remove: [] }],
    [auth.id, { add: [], remove: ["u-ann"] }],
  ]);
  expect(held.modules.update).not.toHaveBeenCalled();
});

it("changes the module's status through the store, and says nothing more once nerve has", async () => {
  const auth = await onPage(its.member);
  await choose("change_module_status", "planned");
  expect(held.modules.update.mock.calls).toEqual([[auth.id, { status: "planned" }]]);
  await act(async () => held.modules.sent[0]?.answer({ ...auth, status: "planned" }));
  expect(toasts).toEqual([]);
});

it.each([
  ["its status", "change_module_status", "completed"],
  ["its members", "add_remove_module_members", "u-bob"],
] as const)("says why nerve refused a change of %s", async (_, id, value) => {
  await onPage(its.member);
  await choose(id, value);
  await act(async () => held.modules.sent[0]?.refuse(refusal(409, "planning.modules_disabled")));
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "errors.planning_modules_disabled" }]);
});

it("says nothing of a refusal that comes once another tab has moved this one to another account", async () => {
  await onPage(its.member);
  await choose("change_module_status", "completed");
  switchAccount();
  await act(async () => held.modules.sent[0]?.refuse(refusal(409, "planning.modules_disabled")));
  expect(toasts).toEqual([]);
});

it("copies the link of the route's module, saying so", async () => {
  const auth = await onPage(its.guest);
  await run("copy_module_url");
  expect(held.copy.mock.calls).toEqual([[`/acme/projects/${auth.project_id}/modules/${auth.id}`]]);
  expect(toasts).toEqual([
    {
      type: "success",
      title: "common.link_copied",
      message: "power_k.contextual_actions.module.copy_url_toast_success",
    },
  ]);
});

it("says it could not copy the link when the browser does not let the page write the clipboard", async () => {
  await onPage(its.member);
  held.copy.mockRejectedValueOnce(new DOMException("Write permission denied.", "NotAllowedError"));
  await run("copy_module_url");
  expect(toasts).toEqual([{ type: "error", title: "toast.error", message: "something_went_wrong_please_try_again" }]);
});
