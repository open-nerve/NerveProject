/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { EditorContent } from "@tiptap/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { createEditor, destroyEditors } from "./editor-test-fixture";

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let root: Root;
const scrollDescriptor = Object.getOwnPropertyDescriptor(HTMLElement.prototype, "scrollIntoView");
beforeEach(() => {
  // oxlint-disable-next-line no-extend-native -- jsdom lacks layout scrolling; C12 verifies real placement.
  Object.defineProperty(HTMLElement.prototype, "scrollIntoView", { configurable: true, value: () => {} });
});

it("leaves Enter in another input alone while slash suggestions remain open", async () => {
  const editor = await createEditor();
  const host = document.createElement("div");
  const input = document.createElement("input");
  document.body.append(host, input);
  root = createRoot(host);
  await act(async () => root.render(<EditorContent editor={editor} />));
  await act(async () => {
    editor.view.dom.focus();
    editor.commands.insertContent("/head");
  });
  await vi.waitFor(() => expect(document.querySelector("#slash-command")).not.toBeNull());
  const before = editor.getHTML();
  let allowed = false;
  await act(async () => {
    input.focus();
    allowed = input.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }));
  });
  expect(allowed).toBe(true);
  expect(editor.getHTML()).toBe(before);
});
afterEach(async () => {
  await act(async () => root?.unmount());
  destroyEditors();
  document.body.replaceChildren();
  // oxlint-disable-next-line no-extend-native -- restore the missing test-environment descriptor.
  if (scrollDescriptor) Object.defineProperty(HTMLElement.prototype, "scrollIntoView", scrollDescriptor);
  else Reflect.deleteProperty(HTMLElement.prototype, "scrollIntoView");
});

it.each([
  { menu: "slash", text: "/head", selector: "#slash-command", input: { isComposing: true } },
  { menu: "slash", text: "/head", selector: "#slash-command", input: { keyCode: 229 } },
  { menu: "emoji", text: ":smi", selector: "#emoji-item-0", input: { isComposing: true } },
  { menu: "emoji", text: ":smi", selector: "#emoji-item-0", input: { keyCode: 229 } },
])("keeps $menu suggestions and content during composition $input", async ({ text, selector, input }) => {
  const editor = await createEditor();
  const host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  await act(async () => root.render(<EditorContent editor={editor} />));
  await act(async () => {
    editor.view.dom.focus();
    editor.commands.insertContent(text);
  });
  await vi.waitFor(() => expect(document.querySelector(selector)).not.toBeNull());
  const before = editor.getHTML();
  const press = (key: string, details: KeyboardEventInit = {}) =>
    act(async () => {
      editor.view.dom.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true, ...details }));
    });
  await press("Enter", input);
  expect(editor.getHTML()).toBe(before);
  expect(document.querySelector(selector)).not.toBeNull();
  await press("Escape", input);
  expect(document.querySelector(selector)).not.toBeNull();
  await press("Enter");
  await vi.waitFor(() => expect(editor.getHTML()).not.toBe(before));
  expect(document.querySelector(selector)).toBeNull();
});
