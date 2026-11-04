// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { ProfileImageView } from "@/hooks/use-profile-image";
import { ProfileImage, ProfileAvatar } from "./profile-image";
const state = vi.hoisted(() => ({ view: undefined as ProfileImageView | undefined }));
vi.mock("@/hooks/use-profile-image", () => ({ useProfileImage: () => state.view! }));
let renderer: Root;
let container: HTMLDivElement;
const source = { kind: "self", image: "avatar" } as const;
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  container = document.createElement("div");
  document.body.append(container);
  renderer = createRoot(container);
  state.view = {
    url: "/static/first",
    loading: false,
    error: undefined,
    onError: vi.fn(),
    onLoad: vi.fn(),
    retry: vi.fn(async () => {}),
  };
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  container.remove();
});
it.each(["image", "avatar"])(
  "%s removes the whole image when permission or expiry removes its URL and forwards image events",
  async (kind) => {
    const render = () =>
      act(async () =>
        renderer.render(
          kind === "image" ? (
            <ProfileImage source={{ ...source }} alt="Ann" fallback="A" className="portrait" />
          ) : (
            <ProfileAvatar source={{ ...source }} alt="Ann" fallback="A" size="xs" className="portrait" />
          )
        )
      );
    await render();
    const image = container.querySelector("img");
    expect(image?.getAttribute("src")).toBe("/static/first");
    await act(async () => image!.dispatchEvent(new Event("load")));
    expect(state.view!.onLoad).toHaveBeenCalledOnce();
    await act(async () => image!.dispatchEvent(new Event("error")));
    expect(state.view!.onError).toHaveBeenCalledOnce();
    state.view = { ...state.view!, url: undefined };
    await render();
    expect(container.querySelector("img")).toBeNull();
    expect(image?.isConnected).toBe(false);
    expect(container.textContent).toContain("A");
    expect(container.querySelector("[aria-label='Ann']")).not.toBeNull();
    state.view = { ...state.view!, url: "/static/next" };
    await render();
    expect(container.querySelector("img")?.getAttribute("src")).toBe("/static/next");
    expect(container.querySelector("img")).not.toBe(image);
  }
);
