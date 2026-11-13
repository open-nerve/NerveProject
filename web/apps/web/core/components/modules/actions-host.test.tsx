// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, expectTypeOf, it, vi } from "vitest";
import type { Module } from "@nerve/api-client";
import { heldChange, signedIn, switchAccount, type HeldChange } from "@/lib/auth/fake-tab";
import { refusal } from "@/lib/fake-refusal";
import { toasts } from "@/lib/fake-toast";
import { moduleOf, projectOf } from "@/store/project/fake-projects";
import { callerInWeb, its } from "@/store/user/fake-permissions";
import type { IUserPermissionStore } from "@/store/user/permissions.store";
import { workspaceOf } from "@/store/workspace/fake-workspaces";
import { ModuleActionsHost, useModulePage } from "./actions-host";
import { emptyPickers } from "./fake-module-dialog";

// What a module's page does with the change of a module its row asks for (P6a spec 2.7): the page's host opens the
// module's dialog (useModulePage), and follows nerve's answer though the row has gone first (a rename the page's
// filters or search hide, say); once the page has gone (a Back, another project: its host with it), nerve's late
// answer reaches the store alone. The dialog and its form are real; the store is a stand-in whose change nerve settles
// when the test says, the pickers fake-module-dialog.tsx's, the caller web's admin (fake-permissions.ts).
const ui = vi.hoisted(() => ({
  modules: new Map<string, Module>(),
  update: vi.fn(),
  permissions: undefined as IUserPermissionStore | undefined,
  page: undefined as ReturnType<typeof useModulePage> | undefined,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => import("@/lib/auth/fake-tab"));
vi.mock("@/hooks/store/use-modules", () => ({
  useModules: () => ({ getModuleById: (id: string) => ui.modules.get(id), create: vi.fn(), update: ui.update }),
}));
vi.mock("@/hooks/store/user/user-permissions", () => ({ useUserPermissions: () => ui.permissions }));
vi.mock("@/components/dropdowns/project/dropdown", () => import("./fake-module-dialog"));
vi.mock("@/components/dropdowns/date-range", () => import("./fake-module-dialog"));
vi.mock("@/components/dropdowns/member/base", () => import("./fake-module-dialog"));
vi.mock("@/hooks/store/use-member", () => import("./fake-module-dialog"));
vi.mock("@/components/issues/issue-detail-widgets/directory-read", () => import("./fake-module-dialog"));

const web = projectOf("WEB", workspaceOf("acme").id);
const auth = moduleOf(web, "Auth");
let sent: HeldChange<Module>;
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers();
  ui.permissions = (await callerInWeb(its.admin.workspaceRole, its.admin.projectRole)).permissions;
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
  emptyPickers();
  ui.modules = new Map([[auth.id, auth]]);
  ui.page = undefined;
  sent = heldChange<Module>();
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

/** A row of the page: its button asks the page for the module's change. */
function Row({ module }: { module: Module }) {
  const page = useModulePage();
  ui.page = page;
  return (
    <button type="button" onClick={() => page.openDialog("edit", module)}>
      {`change ${module.name}`}
    </button>
  );
}

/** The page of web's modules, with Auth's row while it shows it; or another page, elsewhere, of no modules. */
function render(page: "with its row" | "without its row" | "elsewhere") {
  return act(async () =>
    root.render(
      page === "elsewhere" ? (
        <p>elsewhere</p>
      ) : (
        <ModuleActionsHost workspaceSlug="acme" projectId={web.id}>
          {page === "with its row" && <Row module={auth} />}
        </ModuleActionsHost>
      )
    )
  );
}
const settle = () => act(async () => vi.advanceTimersByTimeAsync(100));
const dialog = () => document.querySelector<HTMLElement>('[role="dialog"]');
const title = () => document.getElementById(dialog()?.getAttribute("aria-labelledby") ?? "")?.textContent;
async function change(name: string) {
  await render("with its row");
  await act(async () => host.querySelector("button")!.click());
  await settle();
  expect(title()).toBe("project_module.update_module");
  const input = document.querySelector<HTMLInputElement>('input[name="name"]')!;
  expect(input.value).toBe("Auth");
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, name);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await act(async () =>
    dialog()!
      .querySelector("form")!
      .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
  await settle();
  expect(ui.update.mock.calls).toStrictEqual([[auth.id, { name }]]);
}
const updated = {
  type: "success",
  title: "project_module.action.update.success.title",
  message: "project_module.action.update.success.description",
};

it("changes the module its row asks for, closing and saying so once nerve has, though its row has gone first", async () => {
  await change("Auth v2");
  expect(ui.page?.pageMounted.current).toBe(true);
  // the row goes first: the page's search no longer finds the module by its new name
  await render("without its row");
  expect(dialog()).not.toBeNull();
  await act(async () => sent.answer({ ...auth, name: "Auth v2" }));
  await settle();
  expect(toasts).toEqual([updated]);
  expect(dialog()).toBeNull();
});

it.each([
  ["answered", (held: HeldChange<Module>) => held.answer({ ...auth, name: "Auth v3" })],
  ["refused", (held: HeldChange<Module>) => held.refuse(refusal(409, "planning.module_name_taken"))],
])(
  "follows nothing of a change %s once its page has gone: nerve's answer reaches the store alone",
  async (_, settleIt) => {
    await change("Auth v3");
    await render("elsewhere");
    expect(ui.page?.pageMounted.current).toBe(false);
    await act(async () => settleIt(sent));
    await settle();
    expect(toasts).toEqual([]);
    expect(dialog()).toBeNull();
    expect(host.textContent).toBe("elsewhere");
  }
);

it("follows nothing of a change answered after another tab moved this one to another account", async () => {
  await change("Auth v4");
  switchAccount();
  await act(async () => sent.answer({ ...auth, name: "Auth v4" }));
  await settle();
  expect(toasts).toEqual([]);
  expect(dialog()).not.toBeNull();
});

it("opens a module's change alone: its archive and its deletion are not the page's yet", () => {
  // their dialogs take the old store's module until they are rewritten for a Module (P6a T7)
  expectTypeOf<Parameters<ReturnType<typeof useModulePage>["openDialog"]>[0]>().toEqualTypeOf<"edit">();
});

it("is required by a module's row: none outside a page of modules", async () => {
  vi.spyOn(console, "error").mockImplementation(() => {});
  await expect(act(async () => root.render(<Row module={auth} />))).rejects.toThrow(/ModuleActionsHost/);
});
