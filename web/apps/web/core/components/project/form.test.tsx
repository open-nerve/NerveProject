// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { Routes, Route } from "react-router";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Project } from "@nerve/api-client";
import type { RootStore } from "@/store/root.store";
import { waitForPickerDOM } from "@/components/assets/fake-image-picker";
import { projectCoverURL } from "@/helpers/project-covers";
import { json, problem, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { loadProject } from "@/store/project/fake-projects";
import { coverPage } from "./fake-cover-page";
import { ProjectDetailsForm } from "./form";
import { ProjectCard } from "./card";

vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: {
    start: async () => {},
    state: { status: "signed-in", loginId: "project-cover" },
    subscribe: () => () => {},
  },
  publicClient: { GET: async () => ({ data: { data: [] }, response: new Response(null, { status: 200 }) }) },
}));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
let tab: Awaited<ReturnType<typeof coverPage>>;
const render = (project = tab.project(), isAdmin = true) =>
  tab.render(
    <Routes location={`/${tab.workspace.slug}/settings/projects/${project.id}`}>
      <Route
        path="/:workspaceSlug/settings/projects/:projectId"
        element={
          <>
            <div data-testid="settings">
              <ProjectDetailsForm project={project} workspaceSlug={tab.workspace.slug} isAdmin={isAdmin} />
            </div>
            <div data-testid="card">
              <ProjectCard project={project} />
            </div>
          </>
        }
      />
    </Routes>
  );
const card = () => document.querySelector('[data-testid="card"]');
const saved = () => waitForPickerDOM(() => toasts.some((t) => t.type === "success"), "updated project");
const openUpload = async () => {
  await tab.click("image_upload.change_cover");
  await tab.click("project_cover.upload");
};
const selectUpload = async () => {
  await openUpload();
  await tab.uploaded();
  await tab.closed();
};
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
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
  tab = await coverPage();
  toasts.length = 0;
});
afterEach(async () => {
  await tab.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

it("the admin selects an upload, then one existing form PATCH binds it with unsaved names; the next name-only save omits it", async () => {
  await render();
  await tab.name("Unsaved name");
  await selectUpload();
  expect(tab.writes()).toHaveLength(0);
  expect(card()?.querySelector('img[src="/static/old-cover.png"]')).not.toBeNull();
  expect(document.querySelector<HTMLInputElement>('input[name="name"]')?.value).toBe("Unsaved name");
  await tab.click("common.update_project");
  await saved();
  expect(tab.writes()).toHaveLength(1);
  expect(tab.writes()[0]?.body).toMatchObject({ name: "Unsaved name", cover_image_asset_id: "asset-1" });
  expect(tab.writes()[0]?.body).not.toHaveProperty("cover_builtin_id");
  await waitForPickerDOM(() => !!card()?.querySelector('img[src="/static/asset-1.png"]'), "card confirmed cover");
  toasts.length = 0;
  await tab.name("Another name");
  await tab.click("common.update_project");
  await saved();
  expect(tab.writes()[1]?.body).not.toHaveProperty("cover_image_asset_id");
  expect(tab.writes()[1]?.body).not.toHaveProperty("cover_builtin_id");
  expect(tab.puts).toHaveLength(1);
});

it.each(["preset", "clear"])(
  "the chosen %s keeps stable fields and reaches both settings and card after save",
  async (kind) => {
    await render();
    await tab.click("image_upload.change_cover");
    await tab.click(kind === "preset" ? "project_cover.preset 29" : "project_cover.remove");
    await tab.closed();
    expect(tab.writes()).toHaveLength(0);
    await tab.click("common.update_project");
    await saved();
    expect(tab.writes()[0]?.body).toMatchObject(
      kind === "preset" ? { cover_builtin_id: "cover-29" } : { cover_image_asset_id: null, cover_builtin_id: null }
    );
    expect(tab.puts).toHaveLength(0);
    if (kind === "preset") expect(card()?.querySelector("img")?.getAttribute("src")).toBe(projectCoverURL("cover-29"));
    else expect(card()?.querySelector('img[src="/static/old-cover.png"]')).toBeNull();
  }
);

it.each([
  { workspace: 15 as const, project: 15 as const },
  { workspace: 5 as const, project: 5 as const },
  { workspace: 20 as const, project: 20 as const, archived: true },
])("readonly or archived users see the cover without a write entry: $workspace/$project/$archived", async (role) => {
  await tab.roles(role.workspace, role.project, role.archived);
  await render(tab.project(), role.workspace === 20);
  expect([...document.querySelectorAll("button")].some((b) => b.textContent === "image_upload.change_cover")).toBe(
    false
  );
  expect(tab.button("common.update_project").disabled).toBe(true);
  expect(tab.writes()).toHaveLength(0);
});

it("a refused bind keeps the canonical old cover and local ready selection; retry transfers no more bytes", async () => {
  await render();
  await selectUpload();
  tab.nerve.respond = (call) => (call.method === "PATCH" ? call.answer(problem(403, "forbidden")) : tab.answer(call));
  await tab.click("common.update_project");
  await waitForPickerDOM(() => toasts.some((t) => t.type === "error"), "refused binding");
  expect(card()?.querySelector('img[src="/static/old-cover.png"]')).not.toBeNull();
  expect(document.querySelector('[data-testid="settings"] img[src="blob:selection-1"]')).not.toBeNull();
  tab.nerve.respond = tab.answer;
  await tab.click("common.update_project");
  await saved();
  expect(tab.puts).toHaveLength(1);
  expect(tab.nerve.calls.filter((c) => c.path.endsWith("/complete"))).toHaveLength(1);
  expect(tab.writes().map((c) => (c.body as Record<string, unknown>).cover_image_asset_id)).toEqual([
    "asset-1",
    "asset-1",
  ]);
});

it("the card ignores a stale detail and delayed response after the complete directory revokes its project", async () => {
  const old = tab.project();
  tab.nerve.respond = undefined;
  await loadProject(tab.nerve, tab.app.projectRoot.project, old);
  let held: Call | undefined;
  tab.nerve.respond = (call) => (call.path === `/api/v0/projects/${old.id}` ? (held = call) : tab.answer(call));
  await render(old);
  await act(async () => until(() => !!held, "re-authorized project image read"));
  await tab.directory([]);
  const response = held;
  if (!response) throw new Error("held read absent");
  await act(async () => response.answer(json(200, old)));
  expect(card()?.querySelector('img[src="/static/old-cover.png"]')).toBeNull();
  expect(tab.app.projectRoot.project.getProjectById(old.id)).toBeUndefined();
});

it.each(["cover-03", "unknown"])(
  "a visible public nonmember uses only the canonical preset %s, with no capability GET",
  async (id) => {
    await tab.roles(15, null);
    const preset: Project = {
      ...tab.project(),
      cover_builtin_id: id,
      cover_image_asset_id: null,
      cover_image_url: null,
      cover_image_url_expires_at: null,
    };
    await tab.directory([preset]);
    tab.nerve.calls.length = 0;
    await render({ ...preset, cover_image_url: "/static/stale-prop" }, false);
    expect(card()?.querySelector('img[src="/static/stale-prop"]')).toBeNull();
    if (id === "cover-03") expect(card()?.querySelector("img")?.getAttribute("src")).toBe(projectCoverURL(id));
    expect(tab.nerve.calls.filter((c) => c.path.startsWith("/api/v0/projects/"))).toHaveLength(0);
  }
);

it("the card's project and settings links are separate native links, with no interactive descendants", async () => {
  await render();
  expect(card()?.querySelector("a a, a button")).toBeNull();
  expect(card()?.querySelector(`a[href="/${tab.workspace.slug}/projects/${tab.project().id}/issues"]`)).not.toBeNull();
  expect(
    card()?.querySelector(`a[href="/${tab.workspace.slug}/settings/projects/${tab.project().id}"]`)
  ).not.toBeNull();
});

it("discarding a ready cover on permission loss allows a fresh ordinary save after permission returns", async () => {
  await render();
  await selectUpload();
  await tab.roles(15, 15);
  expect(document.querySelector('[data-testid="settings"] img[src="blob:selection-1"]')).toBeNull();
  await tab.roles(20, 20);
  await tab.name("After recovery");
  await tab.click("common.update_project");
  await saved();
  expect(tab.writes()).toHaveLength(1);
  expect(tab.writes()[0]?.body).not.toHaveProperty("cover_image_asset_id");
});
it("double submit before validation finishes sends one PATCH, keeping all cover controls blocked while binding", async () => {
  await render();
  await selectUpload();
  let held: Call | undefined;
  tab.nerve.respond = (call) => (call.method === "PATCH" ? (held = call) : tab.answer(call));
  const form = document.querySelector("form");
  if (!form) throw new Error("form absent");
  await act(async () => {
    form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
  });
  await act(async () => until(() => !!held, "cover bind"));
  expect(tab.writes()).toHaveLength(1);
  expect(tab.button("image_upload.change_cover").disabled).toBe(true);
  expect(tab.button("updating").disabled).toBe(true);
  const response = held;
  if (!response) throw new Error("held bind absent");
  await act(async () => tab.answer(response));
  await saved();
});
it.each(["success", "failure"])(
  "a late %s after project permission loss and recovery has no toast or old cover",
  async (outcome) => {
    await render();
    await selectUpload();
    const old = tab.project();
    let held: Call | undefined;
    tab.nerve.respond = (call) => (call.method === "PATCH" ? (held = call) : tab.answer(call));
    await tab.click("common.update_project");
    await act(async () => until(() => !!held, "cover bind"));
    await tab.roles(15, 5);
    await tab.roles(20, 20);
    const response = held;
    if (!response) throw new Error("held bind absent");
    await act(async () =>
      response.answer(
        outcome === "failure"
          ? problem(403, "forbidden")
          : json(200, { ...old, cover_image_asset_id: "asset-1", cover_image_url: "/static/late-old-cover" })
      )
    );
    expect(toasts).toEqual([]);
    expect(tab.app.projectRoot.project.getProjectById(old.id)?.cover_image_asset_id).toBe("old-cover");
    expect(document.querySelector('img[src="/static/late-old-cover"]')).toBeNull();
  }
);
it("cancelling a replacement drops its prior unbound asset while preserving the saved canonical cover", async () => {
  await render();
  await selectUpload();
  await openUpload();
  await tab.select();
  await tab.click("cancel");
  await tab.closed();
  await tab.click("common.update_project");
  await saved();
  expect(tab.writes()[0]?.body).not.toHaveProperty("cover_image_asset_id");
  expect(card()?.querySelector('img[src="/static/old-cover.png"]')).not.toBeNull();
});

it("leaving the upload tab cancels its hidden transfer; late success cannot choose or bind a file", async () => {
  await render();
  await openUpload();
  await tab.select();
  await tab.click("image_upload.upload_save");
  await act(async () => until(() => tab.puts.length === 1, "raw PUT"));
  await tab.click("project_cover.presets");
  expect(tab.puts[0]?.signal?.aborted).toBe(true);
  await act(async () => tab.puts[0]?.answer(new Response(null, { status: 204 })));
  await tab.click("project_cover.preset 3");
  await tab.closed();
  await tab.click("common.update_project");
  await saved();
  expect(tab.writes()[0]?.body).toMatchObject({ cover_builtin_id: "cover-03" });
  expect(tab.nerve.calls.filter((c) => c.path.endsWith("/complete"))).toHaveLength(0);
});
