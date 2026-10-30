// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useRef } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import usePeekOutsideClick from "./use-peek-overview-outside-click";
let host: HTMLDivElement;
let outside: HTMLButtonElement;
let root: ReturnType<typeof createRoot>;
const closed = vi.fn();
function Peek({ id }: { id: string }) {
  const ref = useRef<HTMLDivElement>(null);
  usePeekOutsideClick(ref, () => closed(id), id);
  return <div ref={ref}>peek</div>;
}
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  closed.mockReset();
  host = document.createElement("div");
  outside = document.createElement("button");
  outside.dataset.delayOutsideClick = "true";
  document.body.append(host, outside);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  outside.remove();
  vi.useRealTimers();
});
it.each(["replace", "unmount"])("cancels a delayed outside click on %s", async (end) => {
  await act(async () => root.render(<Peek id="old" />));
  await act(async () => outside.dispatchEvent(new MouseEvent("mousedown", { bubbles: true })));
  await act(async () => root.render(end === "replace" ? <Peek id="new" /> : null));
  await act(async () => vi.runAllTimersAsync());
  expect(closed).not.toHaveBeenCalled();
});
it("a current delayed outside click still closes once", async () => {
  await act(async () => root.render(<Peek id="old" />));
  await act(async () => outside.dispatchEvent(new MouseEvent("mousedown", { bubbles: true })));
  await act(async () => vi.runAllTimersAsync());
  expect(closed).toHaveBeenCalledExactlyOnceWith("old");
});
