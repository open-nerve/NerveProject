// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Asset } from "@nerve/api-client";
import { problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { imageFile } from "@/store/asset/fake-assets";
import { ImagePicker } from "./image-picker";
import { pickerTab } from "./fake-image-picker";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
let tab: Awaited<ReturnType<typeof pickerTab>>;
let submit: ReturnType<typeof vi.fn<(asset: Asset | null) => Promise<void>>>;
let saved: ReturnType<typeof vi.fn<() => void>>;
let cancel: ReturnType<typeof vi.fn<() => void>>;
const render = (busyBinding = false) =>
  act(async () =>
    tab.renderer.render(
      <ImagePicker
        upload={tab.upload}
        canRemove
        busyBinding={busyBinding}
        onSubmit={submit}
        onSaved={saved}
        onCancel={cancel}
      />
    )
  );
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await pickerTab();
  submit = vi.fn(async () => {});
  saved = vi.fn();
  cancel = vi.fn();
});
afterEach(async () => {
  await tab.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
const alert = () => document.querySelector<HTMLElement>('[role="alert"]');

it("retiring the upload removes its preview without moving focus back from the new page", async () => {
  await render();
  await tab.select();
  const next = document.createElement("button");
  next.textContent = "new page";
  document.body.append(next);
  await act(async () => {
    next.focus();
    tab.app.dispose();
  });
  expect(document.querySelector("img")).toBeNull();
  expect(document.activeElement).toBe(next);
  next.remove();
});

it("native selection creates one preview, replacement releases it, and valid choice focuses submit", async () => {
  await render();
  const file = imageFile("first.png");
  await tab.select([file]);
  expect(tab.create).toHaveBeenCalledExactlyOnceWith(file);
  expect(document.querySelector("img")?.getAttribute("src")).toBe("blob:selection-1");
  expect(document.activeElement).toBe(tab.button("image_upload.upload_save"));
  await render();
  expect(tab.create).toHaveBeenCalledTimes(1);
  await tab.select([imageFile("second.png")]);
  expect(tab.revoke).toHaveBeenCalledExactlyOnceWith("blob:selection-1");
  expect(document.querySelector("img")?.getAttribute("src")).toBe("blob:selection-2");
});

it.each([
  { files: () => [imageFile("a"), imageFile("b")], error: "multiple" },
  { files: () => [imageFile("a", 4, "")], error: "type" },
  { files: () => [imageFile("a", 11)], error: "size" },
])("invalid file selection $error hides the previous choice and focuses a readable error", async ({ files, error }) => {
  await render();
  await tab.select();
  await tab.select(files());
  expect(document.querySelector("img")).toBeNull();
  expect(alert()?.textContent).toBe(`image_upload.errors.${error}`);
  expect(document.activeElement).toBe(alert());
  expect(tab.nerve.calls).toHaveLength(0);
  expect(tab.puts).toHaveLength(0);
  expect(tab.button("image_upload.upload_save").disabled).toBe(true);
});

it.each(["one", "multiple", "directory", "text"])("drop accepts only a single real image file: %s", async (kind) => {
  await render();
  const file = imageFile();
  const drop = new Event("drop", { bubbles: true, cancelable: true });
  // JSDOM has no DataTransfer constructor; retain the real File and expose only the browser event fields read here.
  Object.defineProperty(drop, "dataTransfer", {
    value: {
      files: kind === "text" ? [] : kind === "multiple" ? [file, file] : [file],
      items: [
        { kind: kind === "text" ? "string" : "file", webkitGetAsEntry: () => ({ isDirectory: kind === "directory" }) },
      ],
    },
  });
  await act(async () => document.querySelector('[aria-label="image_upload.drop"]')!.dispatchEvent(drop));
  if (kind === "one") expect(document.querySelector("img")?.getAttribute("src")).toBe("blob:selection-1");
  else {
    expect(document.querySelector("img")).toBeNull();
    expect(alert()).not.toBeNull();
  }
  expect(tab.nerve.calls).toHaveLength(0);
});

it("double submit sends one raw File chain and awaits one owner bind; cancellation stays explicit", async () => {
  let finish!: () => void;
  submit.mockImplementation(
    () =>
      new Promise<void>((resolve) => {
        finish = resolve;
      })
  );
  await render();
  const file = imageFile();
  await tab.select([file]);
  await act(async () => {
    tab.button("image_upload.upload_save").click();
    tab.button("image_upload.upload_save").click();
  });
  await act(async () => until(() => tab.puts.length === 1, "raw upload"));
  expect(tab.puts[0]!.file).toBe(file);
  expect(document.querySelector('[aria-live="polite"]')?.textContent).toContain("image_upload.phases.uploading");
  expect(tab.button("image_upload.cancel_upload").disabled).toBe(false);
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 204 })));
  await act(async () => until(() => submit.mock.calls.length === 1, "owner bind"));
  expect(submit.mock.calls[0]?.[0]).toMatchObject({ id: "asset-1", state: "uploaded" });
  expect(tab.button("cancel").disabled).toBe(true);
  expect(saved).not.toHaveBeenCalled();
  await act(async () => finish());
  expect(saved).toHaveBeenCalledOnce();
  expect(tab.nerve.calls.map((c) => c.path)).toEqual(["/api/v0/assets", "/api/v0/assets/asset-1/complete"]);
});

it("cancel upload aborts its PUT and late success cannot bind or announce success", async () => {
  await render();
  await tab.select();
  await tab.click("image_upload.upload_save");
  await act(async () => until(() => tab.puts.length === 1, "cancellable upload"));
  await tab.click("image_upload.cancel_upload");
  expect(tab.puts[0]!.signal?.aborted).toBe(true);
  expect(document.querySelector("img")).toBeNull();
  expect(document.activeElement).toBe(document.querySelector('input[type="file"]'));
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 204 })));
  expect(tab.nerve.calls).toHaveLength(1);
  expect(submit).not.toHaveBeenCalled();
  expect(saved).not.toHaveBeenCalled();
});

it("owner refusal preserves the ready asset and retry sends no more bytes or complete", async () => {
  submit.mockRejectedValueOnce(new Error("do not display this signed URL"));
  await render();
  await tab.select();
  await tab.click("image_upload.upload_save");
  await act(async () => until(() => tab.puts.length === 1, "upload"));
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 204 })));
  await act(async () => until(() => submit.mock.calls.length === 1, "refused bind"));
  expect(alert()?.textContent).toBe("errors.unknown");
  expect(document.activeElement).toBe(alert());
  expect(saved).not.toHaveBeenCalled();
  await tab.click("image_upload.save");
  expect(submit.mock.calls[1]?.[0]).toBe(submit.mock.calls[0]?.[0]);
  expect(saved).toHaveBeenCalledOnce();
  expect(tab.puts).toHaveLength(1);
  expect(tab.nerve.calls).toHaveLength(2);
});

it("uncertain PUT checks its prepared asset; missing object requires a fresh prepare before another PUT", async () => {
  await render();
  await tab.select();
  await tab.click("image_upload.upload_save");
  await act(async () => until(() => tab.puts.length === 1, "first PUT"));
  await act(async () => tab.puts[0]!.fail());
  tab.nerve.respond = (call) =>
    call.path.endsWith("/complete") ? call.answer(problem(409, "asset.object_missing")) : tab.answer(call);
  await tab.click("image_upload.check_upload");
  expect(tab.puts).toHaveLength(1);
  expect(tab.nerve.calls[1]?.path).toBe("/api/v0/assets/asset-1/complete");
  await tab.click("image_upload.upload_again");
  await act(async () => until(() => tab.puts.length === 2, "fresh PUT"));
  expect(tab.nerve.calls[2]?.path).toBe("/api/v0/assets");
  expect(submit).not.toHaveBeenCalled();
});

it("remove submits null once and external owner busy prevents choosing or submitting", async () => {
  await render(true);
  expect(document.querySelector<HTMLInputElement>('input[type="file"]')?.disabled).toBe(true);
  await tab.click("image_upload.remove");
  expect(submit).not.toHaveBeenCalled();
  await render();
  await tab.click("image_upload.remove");
  expect(submit.mock.calls).toEqual([[null]]);
  expect(saved).toHaveBeenCalledOnce();
  expect(tab.nerve.calls).toHaveLength(0);
});
