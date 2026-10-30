// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { useTableKeyboardNavigation } from "./use-table-keyboard-navigation";

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let host: HTMLDivElement, root: ReturnType<typeof createRoot>;
const selected = vi.fn();
function Table() {
  const keydown = useTableKeyboardNavigation();
  return (
    <table onKeyDown={keydown}>
      <thead>
        <tr>
          <th tabIndex={0}>Name</th>
          <th tabIndex={0}>Priority</th>
        </tr>
      </thead>
      <tbody>
        <tr>
          <td tabIndex={0}>One</td>
          <td tabIndex={0}>
            <button className="clickable" onClick={selected}>
              Choose priority
            </button>
          </td>
        </tr>
        <tr>
          <td tabIndex={0}>Two</td>
          <td tabIndex={0}>
            <button disabled onClick={selected}>
              Disabled
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  );
}
beforeEach(async () => {
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  selected.mockClear();
  await act(async () => root.render(<Table />));
  for (const cell of host.querySelectorAll<HTMLTableCellElement>("td, th")) cell.scrollIntoView = vi.fn();
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
});
async function key(target: HTMLElement, value: string, composing = false) {
  await act(async () =>
    target.dispatchEvent(
      new KeyboardEvent("keydown", { key: value, isComposing: composing, bubbles: true, cancelable: true })
    )
  );
}
it.each(["Enter", " "])(
  "a cell activates its native control with %s and ignores IME/disabled controls",
  async (value) => {
    const cells = host.querySelectorAll<HTMLTableCellElement>("td");
    await key(cells[1], value, true);
    expect(selected).not.toHaveBeenCalled();
    await key(cells[1], value);
    expect(selected).toHaveBeenCalledTimes(1);
    await key(cells[3], value);
    expect(selected).toHaveBeenCalledTimes(1);
  }
);
it("uses element siblings across header/body even when the DOM contains whitespace nodes", async () => {
  const cells = host.querySelectorAll<HTMLTableCellElement>("td");
  cells[0].after(document.createTextNode("\n"));
  const header = host.querySelectorAll<HTMLTableCellElement>("th")[1];
  await act(async () => header.focus());
  await key(header, "ArrowDown");
  expect(document.activeElement).toBe(cells[1]);
  await key(cells[1], "ArrowLeft");
  expect(document.activeElement).toBe(cells[0]);
  await key(cells[0], "ArrowUp");
  expect(document.activeElement).toBe(host.querySelector("th"));
  await key(header, "ArrowDown", true);
  expect(document.activeElement).toBe(host.querySelector("th"));
});
