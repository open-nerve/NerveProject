// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, StrictMode, useState } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Asset } from "@nerve/api-client";
import { StoreContext } from "@/lib/store-context";
import { until } from "@/lib/auth/fake-time";
import type { RootStore } from "@/store/root.store";
import { avatarTarget } from "@/store/asset/fake-assets";
import { ImageUploadModal } from "./image-upload-modal";
import { pickerTab, waitForPickerDOM } from "./fake-image-picker";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
let tab: Awaited<ReturnType<typeof pickerTab>>;
let submit: ReturnType<typeof vi.fn<(asset: Asset | null) => Promise<void>>>;
let close: ReturnType<typeof vi.fn<() => void>>;
function Host() {
  const [open, setOpen] = useState(false);
  return (
    <StoreContext.Provider value={tab.app}>
      <button type="button" onClick={() => setOpen(true)}>
        Change avatar
      </button>
      <ImageUploadModal
        title="Avatar"
        isOpen={open}
        target={avatarTarget}
        onSubmit={submit}
        onClose={() => {
          close();
          setOpen(false);
        }}
      />
    </StoreContext.Provider>
  );
}
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  tab = await pickerTab();
  submit = vi.fn(async () => {});
  close = vi.fn();
  await act(async () =>
    tab.renderer.render(
      <StrictMode>
        <Host />
      </StrictMode>
    )
  );
});
afterEach(async () => {
  await tab.dispose();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
const open = async () => {
  await act(async () => {
    tab.button("Change avatar").focus();
    tab.button("Change avatar").click();
  });
  expect(document.querySelector('[role="dialog"]')).not.toBeNull();
};
const escape = () =>
  act(async () =>
    document.activeElement!.dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true })
    )
  );

it("names the actual modal and restores trigger focus on ordinary close, with no preview leak in StrictMode", async () => {
  await open();
  await tab.select();
  const dialog = document.querySelector('[role="dialog"]')!;
  expect(document.getElementById(dialog.getAttribute("aria-labelledby")!)?.textContent).toBe("Avatar");
  await escape();
  expect(close).toHaveBeenCalledOnce();
  await waitForPickerDOM(() => !document.querySelector('[role="dialog"]'), "dialog closed");
  expect(close).toHaveBeenCalledOnce();
  expect(tab.revoke).toHaveBeenCalledExactlyOnceWith("blob:selection-1");
  expect(document.activeElement).toBe(tab.button("Change avatar"));
  await open();
  await tab.select();
  expect(tab.create).toHaveBeenCalledTimes(2);
});

it("Escape and backdrop cannot close during transfer or binding; success closes once and returns focus", async () => {
  let finish!: () => void;
  submit.mockImplementation(
    () =>
      new Promise<void>((resolve) => {
        finish = resolve;
      })
  );
  await open();
  await tab.select();
  await tab.click("image_upload.upload_save");
  await act(async () => until(() => tab.puts.length === 1, "upload"));
  await escape();
  expect(close).not.toHaveBeenCalled();
  // A real pointer click on Headless UI's backdrop, outside the panel.
  const backdrop = document.querySelector<HTMLElement>('[data-headlessui-state][class*="bg-backdrop"]');
  expect(backdrop).not.toBeNull();
  await act(async () => {
    backdrop!.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
    backdrop!.click();
  });
  expect(close).not.toHaveBeenCalled();
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 204 })));
  await act(async () => until(() => submit.mock.calls.length === 1, "binding"));
  await escape();
  expect(close).not.toHaveBeenCalled();
  await act(async () => finish());
  expect(close).toHaveBeenCalledOnce();
  await waitForPickerDOM(() => !document.querySelector('[role="dialog"]'), "saved dialog closed");
  expect(close).toHaveBeenCalledOnce();
  expect(document.activeElement).toBe(tab.button("Change avatar"));
  expect(tab.revoke).toHaveBeenCalledExactlyOnceWith("blob:selection-1");
});

it("unmount aborts its resource and a late bind completion cannot close a newer modal or steal focus", async () => {
  let finish!: () => void;
  submit.mockImplementation(
    () =>
      new Promise<void>((resolve) => {
        finish = resolve;
      })
  );
  await open();
  await tab.select();
  await tab.click("image_upload.upload_save");
  await act(async () => until(() => tab.puts.length === 1, "upload"));
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 204 })));
  await act(async () => until(() => submit.mock.calls.length === 1, "old binding"));
  await act(async () => tab.renderer.render(<button type="button">New page</button>));
  const newPage = tab.button("New page");
  await act(async () => newPage.focus());
  await act(async () => finish());
  expect(close).not.toHaveBeenCalled();
  expect(document.activeElement).toBe(newPage);
  expect(tab.revoke).toHaveBeenCalledExactlyOnceWith("blob:selection-1");
});

it("unmount during PUT stops the chain before complete and releases the preview", async () => {
  await open();
  await tab.select();
  await tab.click("image_upload.upload_save");
  await act(async () => until(() => tab.puts.length === 1, "upload"));
  await act(async () => tab.renderer.render(<p>Other page</p>));
  expect(tab.puts[0]!.signal?.aborted).toBe(true);
  await act(async () => tab.puts[0]!.answer(new Response(null, { status: 204 })));
  expect(tab.nerve.calls).toHaveLength(1);
  expect(submit).not.toHaveBeenCalled();
  expect(tab.revoke).toHaveBeenCalledExactlyOnceWith("blob:selection-1");
});
