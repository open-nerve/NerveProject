// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it } from "vitest";
import useLocalStorage from "./use-local-storage";

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
let result: ReturnType<typeof useLocalStorage<{ open: boolean }>>;
function Probe() {
  result = useLocalStorage<{ open: boolean }>("panel", { open: false });
  return null;
}
beforeEach(() => {
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  window.localStorage.clear();
});

it("starts from the stored JSON value and writes the new value back", async () => {
  window.localStorage.setItem("panel", JSON.stringify({ open: true }));
  await act(async () => root.render(<Probe />));
  expect(result.storedValue).toEqual({ open: true });
  await act(async () => result.setValue({ open: false }));
  expect(window.localStorage.getItem("panel")).toBe('{"open":false}');
  expect(result.storedValue).toEqual({ open: false });
  await act(async () => result.clearValue());
  expect(window.localStorage.getItem("panel")).toBeNull();
  // Its own change event reads the storage again, which now gives the initial value.
  expect(result.storedValue).toEqual({ open: false });
});

it("starts from the initial value when nothing is stored, and drops a stored value that is not JSON", async () => {
  await act(async () => root.render(<Probe />));
  expect(result.storedValue).toEqual({ open: false });
  await act(async () => root.unmount());
  window.localStorage.setItem("panel", "{broken");
  root = createRoot(host);
  await act(async () => root.render(<Probe />));
  expect(result.storedValue).toEqual({ open: false });
  expect(window.localStorage.getItem("panel")).toBeNull();
});
