// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { waitForPickerDOM } from "@/components/assets/fake-image-picker";
import { problem, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import type { RootStore } from "@/store/root.store";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { CreateProjectForm } from "./root";
import { projectPage } from "./fake-project-page";

vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: {
    start: async () => {},
    state: { status: "signed-in", loginId: "project-images" },
    subscribe: () => () => {},
  },
  publicClient: {},
}));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
let tab: Awaited<ReturnType<typeof projectPage>>;
let next: ReturnType<typeof vi.fn<(id: string) => void>>;
let close: ReturnType<typeof vi.fn<() => void>>;
const render = (slug = tab.workspace.slug) =>
  tab.render(<CreateProjectForm workspaceSlug={slug} onClose={close} handleNextStep={next} />);
const openUpload = async () => {
  await tab.click("image_upload.change_cover");
  await tab.click("project_cover.upload");
};
const upload = async () => {
  await openUpload();
  await tab.uploaded();
  await waitForPickerDOM(() => !document.querySelector('input[type="file"]'), "cover selected");
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
  tab = await projectPage();
  next = vi.fn();
  close = vi.fn();
  toasts.length = 0;
});
afterEach(async () => {
  await tab.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

it.each(["unset", "preset", "clear", "upload"])(
  "creates once with %s, and never follows with a PATCH",
  async (kind) => {
    await render();
    await tab.name();
    if (kind === "upload") await upload();
    else if (kind !== "unset") {
      await tab.click("image_upload.change_cover");
      await tab.click(kind === "preset" ? "project_cover.preset 19" : "project_cover.remove");
    }
    await tab.submit();
    await waitForPickerDOM(() => next.mock.calls.length === 1, "created project");
    expect(tab.creations()).toHaveLength(1);
    const body = tab.creations()[0]?.body as Record<string, unknown>;
    const cover = Object.fromEntries(Object.entries(body).filter(([key]) => key.startsWith("cover_")));
    expect(cover).toEqual(
      kind === "unset"
        ? {}
        : kind === "preset"
          ? { cover_builtin_id: "cover-19" }
          : kind === "clear"
            ? { cover_image_asset_id: null, cover_builtin_id: null }
            : { cover_image_asset_id: "asset-1" }
    );
    expect(tab.nerve.calls.filter((c) => c.method === "PATCH")).toHaveLength(0);
    if (kind === "upload") {
      expect(tab.nerve.calls.find((c) => c.path === "/api/v0/assets")?.body).toMatchObject({
        purpose: "PROJECT_COVER",
        scope: { kind: "new_project", id: tab.workspace.id },
      });
      expect(tab.puts).toHaveLength(1);
    }
  }
);

it.each(["upload", "preset", "cancel"])(
  "closing the cover content after %s returns focus to its trigger",
  async (how) => {
    await render();
    if (how === "upload") await upload();
    else if (how === "preset") {
      await tab.click("image_upload.change_cover");
      await act(async () => tab.button("project_cover.preset 3").focus());
      await tab.click("project_cover.preset 3");
    } else {
      await openUpload();
      await tab.select();
      await tab.click("cancel");
    }
    expect(document.activeElement).toBe(tab.button("image_upload.change_cover"));
  }
);

it.each(["prepare", "put", "complete"])(
  "failure at %s keeps creation blocked until the file is cancelled or replaced",
  async (boundary) => {
    await render();
    await tab.name();
    await openUpload();
    tab.nerve.respond = (call) =>
      (boundary === "prepare" && call.path === "/api/v0/assets") ||
      (boundary === "complete" && call.path.endsWith("/complete"))
        ? call.answer(problem(503, "server_busy"))
        : tab.answer(call);
    await tab.select();
    await tab.click("image_upload.upload_save");
    if (boundary !== "prepare") {
      await act(async () => until(() => tab.puts.length === 1, "PUT"));
      await act(async () =>
        boundary === "put" ? tab.puts[0]?.fail() : tab.puts[0]?.answer(new Response(null, { status: 204 }))
      );
    }
    await waitForPickerDOM(() => !!document.querySelector('[role="alert"]'), "upload refusal");
    await tab.submit();
    expect(tab.creations()).toHaveLength(0);
    expect(next).not.toHaveBeenCalled();
  }
);

it("a refused binding retains the uploaded preview; retry only repeats the creation", async () => {
  await render();
  await tab.name();
  await upload();
  tab.nerve.respond = (call) =>
    call.method === "POST" && call.path.endsWith("/projects")
      ? call.answer(problem(409, "project.identifier_taken"))
      : tab.answer(call);
  await tab.submit();
  await waitForPickerDOM(() => !!document.querySelector('[aria-invalid="true"]'), "identifier refusal");
  expect(document.querySelector('img[src="blob:selection-1"]')).not.toBeNull();
  expect(next).not.toHaveBeenCalled();
  tab.nerve.respond = tab.answer;
  await tab.name("Another");
  await tab.submit();
  await waitForPickerDOM(() => next.mock.calls.length === 1, "retried creation");
  expect(tab.puts).toHaveLength(1);
  expect(tab.nerve.calls.filter((c) => c.path.endsWith("/complete"))).toHaveLength(1);
  expect(tab.creations().map((c) => (c.body as Record<string, unknown>).cover_image_asset_id)).toEqual([
    "asset-1",
    "asset-1",
  ]);
});

it.each(["cancel", "Escape"])(
  "cancelling a replacement with %s cannot silently bind its discarded previous upload",
  async (how) => {
    await render();
    await tab.name();
    await upload();
    await openUpload();
    await tab.select();
    if (how === "cancel") await tab.click("cancel");
    else await act(async () => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true })));
    await tab.submit();
    await waitForPickerDOM(() => next.mock.calls.length === 1, "creation after cancelled selection");
    expect(tab.creations()[0]?.body).not.toHaveProperty("cover_image_asset_id");
    expect(document.querySelector('img[src^="blob:"]')).toBeNull();
  }
);

it.each(["cancel", "Escape"])("%s from an unused upload panel preserves an explicit two-field removal", async (how) => {
  await render();
  await tab.name();
  await tab.click("image_upload.change_cover");
  await tab.click("project_cover.remove");
  await openUpload();
  if (how === "cancel") await tab.click("cancel");
  else await act(async () => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true })));
  await tab.submit();
  await waitForPickerDOM(() => next.mock.calls.length === 1, "creation after cleared cover");
  expect(tab.creations()[0]?.body).toMatchObject({ cover_image_asset_id: null, cover_builtin_id: null });
});

it("upload busy blocks submit and Escape; selecting a preset cancels it before a single creation", async () => {
  await render();
  await tab.name();
  await openUpload();
  await tab.select();
  await tab.click("image_upload.upload_save");
  await act(async () => until(() => tab.puts.length === 1, "held PUT"));
  await tab.submit();
  await act(async () => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true })));
  expect(tab.creations()).toHaveLength(0);
  expect(close).not.toHaveBeenCalled();
  await tab.click("project_cover.presets");
  await tab.click("project_cover.preset 2");
  expect(tab.puts[0]?.signal?.aborted).toBe(true);
  await act(async () => tab.puts[0]?.answer(new Response(null, { status: 204 })));
  await tab.submit();
  await waitForPickerDOM(() => next.mock.calls.length === 1, "preset creation");
  expect(tab.creations()[0]?.body).toMatchObject({ cover_builtin_id: "cover-02" });
  expect(tab.nerve.calls.filter((c) => c.path.endsWith("/complete"))).toHaveLength(0);
});

it.each(["network", "unrecognized response"])(
  "double submit sends one POST; %s asks for verification and keeps the form and ready cover",
  async (failure) => {
    let held: Call | undefined;
    tab.nerve.respond = (call) =>
      call.method === "POST" && call.path.endsWith("/projects") ? (held = call) : tab.answer(call);
    await render();
    await tab.name();
    await upload();
    const form = document.querySelector("form");
    if (!form) throw new Error("form absent");
    await act(async () => {
      form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
      form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    });
    await act(async () => until(() => !!held, "held create"));
    await act(async () => document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true })));
    await tab.click("close");
    expect(close).not.toHaveBeenCalled();
    expect(tab.creations()).toHaveLength(1);
    await act(async () => (failure === "network" ? held?.fail() : held?.answer(new Response(null, { status: 502 }))));
    await waitForPickerDOM(() => toasts.length > 0 || !!document.querySelector('[role="alert"]'), "uncertain response");
    expect(document.querySelector('[role="alert"]')?.textContent).toBe("project_creation_unknown");
    expect(toasts).toEqual([]);
    expect(document.querySelector<HTMLInputElement>('input[name="name"]')?.value).toBe("Fresh");
    expect(document.querySelector('img[src="blob:selection-1"]')).not.toBeNull();
    await act(async () => vi.advanceTimersByTimeAsync(10_000));
    expect(tab.creations()).toHaveLength(1);
    expect(next).not.toHaveBeenCalled();
    expect(tab.puts).toHaveLength(1);
    expect(tab.nerve.calls.filter((c) => c.path.endsWith("/complete"))).toHaveLength(1);
    // After checking the project list, the user may explicitly decide to submit again.
    tab.nerve.respond = tab.answer;
    await tab.submit();
    await waitForPickerDOM(() => next.mock.calls.length === 1, "explicit creation retry");
    expect(document.querySelector('[role="alert"]')).toBeNull();
    expect(tab.creations().map((c) => (c.body as Record<string, unknown>).cover_image_asset_id)).toEqual([
      "asset-1",
      "asset-1",
    ]);
    expect(tab.puts).toHaveLength(1);
    expect(tab.nerve.calls.filter((c) => c.path.endsWith("/complete"))).toHaveLength(1);
  }
);

it.each(["unmount", "other-workspace", "same-slug-new-id", "role-aba"])(
  "late success after %s cannot toast or advance the new form",
  async (change) => {
    let held: Call | undefined;
    tab.nerve.respond = (call) =>
      call.method === "POST" && call.path.endsWith("/projects") ? (held = call) : tab.answer(call);
    await render();
    await tab.name();
    await upload();
    await tab.submit();
    await act(async () => until(() => !!held, "held binding"));
    const response = held;
    if (!response) throw new Error("binding absent");
    tab.nerve.respond = undefined;
    if (change === "unmount") await tab.render(<div>next page</div>);
    else if (change === "role-aba") {
      await act(async () => loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 5 }]));
      await act(async () => loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace]));
    } else {
      const other =
        change === "other-workspace" ? workspaceOf("other", { role: 20 }) : { ...tab.workspace, id: "replacement" };
      await act(async () => loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [other]));
      await render(other.slug);
      await tab.name("New draft");
    }
    await act(async () => tab.answer(response));
    expect(toasts).toEqual([]);
    expect(next).not.toHaveBeenCalled();
    expect(document.querySelector('img[src="blob:selection-1"]')).toBeNull();
    if (change === "other-workspace" || change === "same-slug-new-id")
      expect(document.querySelector<HTMLInputElement>('input[name="name"]')?.value).toBe("New draft");
  }
);
