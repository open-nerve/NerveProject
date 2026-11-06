// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, expect, it, vi } from "vitest";
import { storage } from "./local-storage";

afterEach(() => {
  vi.unstubAllGlobals();
  window.localStorage.clear();
});

it("stores strings and booleans as text and objects as JSON, and reads and removes them", () => {
  storage.set("text", "value");
  storage.set("flag", true);
  storage.set("object", { a: 1 });
  expect(storage.get("text")).toBe("value");
  expect(storage.get("flag")).toBe("true");
  expect(storage.get("object")).toBe('{"a":1}');
  storage.remove("text");
  expect(storage.get("text")).toBeUndefined();
  expect(window.localStorage.getItem("text")).toBeNull();
});

it("ignores an empty key, an empty value and an empty object", () => {
  storage.set("", "value");
  storage.set("empty", "");
  storage.set("nothing", {});
  storage.remove("");
  expect(window.localStorage.length).toBe(0);
  expect(storage.get("missing")).toBeUndefined();
});

it("does nothing without a window", () => {
  vi.stubGlobal("window", undefined);
  expect(() => storage.set("text", "value")).not.toThrow();
  expect(storage.get("text")).toBeUndefined();
  expect(() => storage.remove("text")).not.toThrow();
});
