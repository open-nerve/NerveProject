// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { useDropHighlight } from "./use-drop-highlight";
const scroll = vi.hoisted(() => ({ calls: 0, container: undefined as HTMLElement | undefined }));
// The library's geometry is unavailable in jsdom; its computed scroll action enters the real animation owner.
vi.mock("smooth-scroll-into-view-if-needed", () => ({
  default: (_element: HTMLElement, options: { behavior: (actions: unknown) => void }) => {
    scroll.calls++;
    options.behavior([{ el: scroll.container, left: 0, top: 200 }]);
    return Promise.resolve();
  },
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let view: ReturnType<typeof createRoot>;
let host: HTMLDivElement;
let target: HTMLDivElement;
let current: boolean;
let highlight: ReturnType<typeof useDropHighlight>;
function Probe({ scope }: { scope: string }) {
  highlight = useDropHighlight(scope);
  return null;
}
beforeEach(async () => {
  vi.useFakeTimers();
  scroll.calls = 0;
  current = true;
  host = document.createElement("div");
  document.body.append(host);
  target = document.createElement("div");
  target.id = "drop-card";
  document.body.append(target);
  scroll.container = host;
  view = createRoot(host);
  await act(async () => view.render(<Probe scope="A" />));
});
afterEach(async () => {
  await act(async () => view.unmount());
  target.remove();
  host.remove();
  vi.useRealTimers();
});
it.each(["unmount", "scope", "permission"])(
  "delayed highlight and scroll stop before %s can own a replacement card",
  async (end) => {
    highlight(target.id, true, () => current);
    if (end === "unmount") await act(async () => view.render(null));
    if (end === "scope") await act(async () => view.render(<Probe scope="B" />));
    if (end === "permission") current = false;
    await vi.advanceTimersByTimeAsync(200);
    expect(target.classList.contains("highlight")).toBe(false);
    expect(scroll.calls).toBe(0);
  }
);
it.each(["unmount", "scope", "permission"])("ongoing smooth scrolling stops after %s", async (end) => {
  highlight(target.id, true, () => current);
  await vi.advanceTimersByTimeAsync(300);
  expect(target.classList.contains("highlight")).toBe(true);
  const before = host.scrollTop;
  expect(before).toBeGreaterThan(0);
  expect(before).toBeLessThan(200);
  if (end === "unmount") await act(async () => view.render(null));
  if (end === "scope") await act(async () => view.render(<Probe scope="B" />));
  if (end === "permission") current = false;
  await vi.advanceTimersByTimeAsync(1500);
  expect(host.scrollTop).toBe(before);
  expect(target.classList.contains("highlight")).toBe(false);
});
