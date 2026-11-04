/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { expect, vi } from "vitest";
import { assetTab, avatarTarget, imageFile } from "@/store/asset/fake-assets";

/** Let React commit between animation frames; stop at the condition with a bounded fake-time deadline. */
export async function waitForPickerDOM(done: () => boolean, what: string, remaining = 20_000): Promise<void> {
  if (done()) return;
  if (remaining <= 0) throw new Error(`timed out waiting for ${what}`);
  await act(async () => vi.advanceTimersByTimeAsync(10));
  return waitForPickerDOM(done, what, remaining - 10);
}

/** Real upload session and transport edges, with native controls in JSDOM. */
export async function pickerTab() {
  const tab = await assetTab();
  const container = document.createElement("div");
  document.body.append(container);
  const renderer = createRoot(container);
  let sequence = 0;
  const create = vi.spyOn(URL, "createObjectURL").mockImplementation(() => `blob:selection-${++sequence}`);
  const revoke = vi.spyOn(URL, "revokeObjectURL").mockImplementation(() => {});
  const upload = tab.app.asset.createUpload(avatarTarget);
  const button = (name: string) => {
    const result = [...document.querySelectorAll<HTMLButtonElement>("button")].find(
      (b) => b.textContent === name || b.getAttribute("aria-label") === name
    );
    expect(result, `button ${name}`).toBeDefined();
    return result!;
  };
  const select = async (files = [imageFile()]) => {
    const input = document.querySelector<HTMLInputElement>('input[type="file"]');
    expect(input).not.toBeNull();
    Object.defineProperty(input!, "files", { configurable: true, value: files });
    await act(async () => input!.dispatchEvent(new Event("change", { bubbles: true })));
  };
  const click = (name: string) => act(async () => button(name).click());
  const dispose = async () => {
    await act(async () => renderer.unmount());
    tab.app.dispose();
    container.remove();
  };
  return { ...tab, container, renderer, upload, create, revoke, select, button, click, dispose };
}
