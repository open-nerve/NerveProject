// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { Routes, Route } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Workspace, WorkspaceUpdate } from "@nerve/api-client";
import { profilePage } from "@/components/assets/fake-profile-page";
import { waitForPickerDOM } from "@/components/assets/fake-image-picker";
import { json, problem, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import type { RootStore } from "@/store/root.store";
import { WorkspaceDetails } from "./workspace-details";

vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: {
    start: async () => {},
    state: { status: "signed-in", loginId: "workspace-images" },
    subscribe: () => () => {},
  },
  publicClient: {},
}));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
let tab: Awaited<ReturnType<typeof profilePage>>;
let rows: Workspace[];
const writes = () => tab.nerve.calls.filter((call) => call.method === "PATCH");
const answer = (call: Call) => {
  if (call.path === "/api/v0/workspaces") call.answer(json(200, { data: rows }));
  else if (call.method === "PATCH" && call.path.startsWith("/api/v0/workspaces/")) {
    const original = rows.find((row) => call.path.endsWith(`/${row.slug}`));
    if (!original) throw new Error("unknown workspace write");
    const changes = call.body as WorkspaceUpdate;
    const updated = { ...original, ...changes };
    if ("logo_asset_id" in changes) {
      updated.logo_url = changes.logo_asset_id ? `/static/${changes.logo_asset_id}` : null;
      updated.logo_url_expires_at = changes.logo_asset_id ? new Date(Date.now() + 60_000).toISOString() : null;
    }
    rows = rows.map((row) => (row.id === original.id ? updated : row));
    call.answer(json(200, updated));
  } else tab.answer(call);
};
const directory = async (next: Workspace[]) => {
  const previous = tab.nerve.respond;
  tab.nerve.respond = undefined;
  rows = next;
  await act(async () => loadWorkspaces(tab.nerve, tab.app.workspaceRoot, next));
  tab.nerve.respond = previous;
};
const render = (slug = tab.workspace.slug) =>
  tab.render(
    <Routes location={`/${slug}/settings`}>
      <Route path="/:workspaceSlug/settings" element={<WorkspaceDetails />} />
    </Routes>
  );
const editName = async (name: string) => {
  const input = tab.container.querySelector<HTMLInputElement>('input[name="name"]');
  if (!input) throw new Error("name field absent");
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set?.call(input, name);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  expect(input.value).toBe(name);
};
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  tab = await profilePage();
  rows = [tab.workspace];
  tab.nerve.respond = answer;
  toasts.length = 0;
});
afterEach(async () => {
  await tab.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

it("admin uploads/removes only the logo field and neither operation resets unsaved name text", async () => {
  await render();
  await editName("Not saved yet");
  await tab.click("workspace_settings.settings.general.upload_logo");
  await tab.uploaded();
  await tab.closed();
  expect(writes().map((call) => call.body)).toEqual([{ logo_asset_id: "asset-1" }]);
  expect(tab.nerve.to("/api/v0/assets")[0]?.body).toMatchObject({
    purpose: "WORKSPACE_LOGO",
    scope: { kind: "workspace", id: tab.workspace.id },
  });
  expect(tab.container.querySelector<HTMLInputElement>('input[name="name"]')?.value).toBe("Not saved yet");
  await tab.click("workspace_settings.settings.general.edit_logo");
  await tab.click("image_upload.remove");
  await tab.closed();
  expect(writes().at(-1)?.body).toEqual({ logo_asset_id: null });
  expect(tab.container.querySelector<HTMLInputElement>('input[name="name"]')?.value).toBe("Not saved yet");
  expect(tab.app.workspaceRoot.getWorkspaceBySlug(tab.workspace.slug)?.logo_asset_id).toBeNull();
});

it.each([15, 5] as const)("role %s only sees the current logo and has no image edit entry", async (role) => {
  await directory([
    {
      ...tab.workspace,
      role,
      logo_asset_id: "existing",
      logo_url: "/static/existing",
      logo_url_expires_at: new Date(Date.now() + 60_000).toISOString(),
    },
  ]);
  await render();
  await waitForPickerDOM(() => !!tab.container.querySelector('img[src="/static/existing"]'), "visible logo");
  expect(
    [...tab.container.querySelectorAll("button")].some((b) =>
      /upload_logo|edit_logo/.test(`${b.textContent} ${b.getAttribute("aria-label")}`)
    )
  ).toBe(false);
  expect(document.querySelector('input[type="file"]')).toBeNull();
  expect(tab.nerve.to("/api/v0/assets")).toHaveLength(0);
});

it("URL refresh leaves dirty form values alone, but actual workspace field/identity changes still reset them", async () => {
  await render();
  await editName("Local draft");
  await directory([
    { ...tab.workspace, logo_url: "/static/new", logo_url_expires_at: new Date(Date.now() + 60_000).toISOString() },
  ]);
  expect(tab.container.querySelector<HTMLInputElement>('input[name="name"]')?.value).toBe("Local draft");
  await directory([{ ...tab.workspace, name: "Server rename" }]);
  expect(tab.container.querySelector<HTMLInputElement>('input[name="name"]')?.value).toBe("Server rename");
  const other = workspaceOf("other", { role: 20 });
  await directory([tab.workspace, other]);
  await editName("Another draft");
  await render(other.slug);
  expect(tab.container.querySelector<HTMLInputElement>('input[name="name"]')?.value).toBe(other.name);
});

it("a refused logo bind keeps the old image and selected file, and a retry only sends PATCH", async () => {
  const old = {
    ...tab.workspace,
    logo_asset_id: "existing",
    logo_url: "/static/existing",
    logo_url_expires_at: new Date(Date.now() + 60_000).toISOString(),
  };
  await directory([old]);
  let refuse = true;
  tab.nerve.respond = (call) =>
    call.method === "PATCH" && refuse ? call.answer(problem(403, "forbidden")) : answer(call);
  await render();
  await tab.click("workspace_settings.settings.general.edit_logo");
  await tab.uploaded();
  await waitForPickerDOM(() => !!document.querySelector('[role="alert"]'), "binding refusal");
  expect(tab.app.workspaceRoot.getWorkspaceBySlug(old.slug)?.logo_asset_id).toBe("existing");
  expect(tab.container.querySelector('img[src="/static/existing"]')).not.toBeNull();
  expect(document.querySelector('img[src="blob:selection-1"]')).not.toBeNull();
  refuse = false;
  await tab.click("image_upload.save");
  await tab.closed();
  expect(tab.puts).toHaveLength(1);
  expect(writes().map((call) => call.body)).toEqual([{ logo_asset_id: "asset-1" }, { logo_asset_id: "asset-1" }]);
});

it.each(["role-aba", "same-slug-new-id", "different-workspace"])(
  "late binding after %s cannot publish on the new view or restore its modal",
  async (change) => {
    let held: Call | undefined;
    tab.nerve.respond = (call) => (call.method === "PATCH" ? (held = call) : answer(call));
    await render();
    await tab.click("workspace_settings.settings.general.upload_logo");
    await tab.uploaded();
    await act(async () => until(() => !!held, "held owner binding"));
    const next = { ...tab.workspace, id: "new-workspace", name: "Replacement" };
    if (change === "role-aba") {
      await directory([{ ...tab.workspace, role: 15 }]);
      await tab.closed();
      await directory([tab.workspace]);
    } else if (change === "same-slug-new-id") await directory([next]);
    else {
      const other = workspaceOf("other", { role: 20 });
      await directory([tab.workspace, other]);
      await render(other.slug);
    }
    await tab.closed();
    await editName("Fresh unsaved name");
    const binding = held;
    if (!binding) throw new Error("no binding request");
    await act(async () =>
      binding.answer(
        json(200, {
          ...tab.workspace,
          logo_asset_id: "old-upload",
          logo_url: "/static/old",
          logo_url_expires_at: new Date(Date.now() + 60_000).toISOString(),
        })
      )
    );
    expect(tab.container.querySelector<HTMLInputElement>('input[name="name"]')?.value).toBe("Fresh unsaved name");
    expect(document.querySelector('[role="dialog"]')).toBeNull();
    expect(tab.container.querySelector('img[src="/static/old"]')).toBeNull();
    expect(toasts).toEqual([]);
    expect(tab.revoke).toHaveBeenCalledExactlyOnceWith("blob:selection-1");
  }
);
