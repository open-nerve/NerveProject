// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { projectPage } from "@/components/projects/create/fake-project-page";
import { waitForPickerDOM } from "@/components/assets/fake-image-picker";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import type { RootStore } from "@/store/root.store";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { CreateProjectModal } from "./create-project-modal";

vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: {
    start: async () => {},
    state: { status: "signed-in", loginId: "create-modal" },
    subscribe: () => () => {},
  },
  publicClient: {},
}));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// The second step's feature toggles have their own tests; keep its received project/workspace visible here.
vi.mock("./project-feature-update", () => ({
  ProjectFeatureUpdate: ({ projectId, workspaceSlug }: { projectId: string; workspaceSlug: string }) => (
    <p data-testid="features">
      {workspaceSlug}/{projectId}
    </p>
  ),
}));
let tab: Awaited<ReturnType<typeof projectPage>>;
let close: ReturnType<typeof vi.fn<() => void>>;
const render = (isOpen = true, workspaceSlug = tab.workspace.slug) =>
  tab.render(<CreateProjectModal isOpen={isOpen} workspaceSlug={workspaceSlug} onClose={close} />);
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
  tab = await projectPage();
  close = vi.fn();
  toasts.length = 0;
});
afterEach(async () => {
  await tab.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

it("cover content stays inside one modal; successful single POST advances to the existing feature step", async () => {
  await render();
  await tab.name();
  await tab.click("image_upload.change_cover");
  expect(document.querySelectorAll('[role="dialog"]')).toHaveLength(1);
  await tab.click("project_cover.preset 11");
  await tab.submit();
  await waitForPickerDOM(() => !!document.querySelector('[data-testid="features"]'), "feature selection");
  expect(document.querySelector('[data-testid="features"]')?.textContent).toBe(`${tab.workspace.slug}/p-fresh`);
  expect(tab.creations()).toHaveLength(1);
  expect(document.querySelector("form")).toBeNull();
});

it("Back/close immediately releases the upload; reopen cannot inherit late bytes or an old 300ms reset", async () => {
  await render();
  await tab.name("Old draft");
  await tab.click("image_upload.change_cover");
  await tab.click("project_cover.upload");
  await tab.select();
  await tab.click("image_upload.upload_save");
  await act(async () => until(() => tab.puts.length === 1, "old PUT"));
  await render(false);
  expect(tab.puts[0]?.signal?.aborted).toBe(true);
  expect(tab.revoke).toHaveBeenCalledWith("blob:selection-1");
  await render();
  await tab.name("New draft");
  await act(async () => tab.puts[0]?.answer(new Response(null, { status: 204 })));
  await act(async () => vi.advanceTimersByTimeAsync(500));
  expect(document.querySelector<HTMLInputElement>('input[name="name"]')?.value).toBe("New draft");
  expect(document.querySelector('img[src="blob:selection-1"]')).toBeNull();
  expect(tab.nerve.calls.filter((call) => call.path.endsWith("/complete"))).toHaveLength(0);
  expect(tab.creations()).toHaveLength(0);
  expect(toasts).toEqual([]);
});

it.each(["other-workspace", "same-slug-new-id"])(
  "%s replaces even a completed project's feature step",
  async (kind) => {
    await render();
    await tab.name();
    await tab.submit();
    await waitForPickerDOM(() => !!document.querySelector('[data-testid="features"]'), "features");
    const next =
      kind === "other-workspace" ? workspaceOf("other", { role: 20 }) : { ...tab.workspace, id: "replacement" };
    tab.nerve.respond = undefined;
    await act(async () => loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [next]));
    await render(true, next.slug);
    expect(document.querySelector('[data-testid="features"]')).toBeNull();
    expect(document.querySelector<HTMLInputElement>('input[name="name"]')?.value).toBe("");
  }
);
