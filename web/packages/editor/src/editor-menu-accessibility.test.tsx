/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";
import { EditorContent } from "@tiptap/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { LinkViewContainer } from "@/components/editors/link-view-container";
import { EditorBubbleMenu } from "@/components/menus/bubble-menu/root";
import { createEditor, destroyEditors } from "./editor-test-fixture";

// Only the external tooltip positioning shell is replaced. All menu controls, editor state,
// commands and Floating UI submenus are real; browser C12 covers actual tooltip placement.
vi.mock("@tiptap/react", async (original) => ({
  ...(await original<typeof import("@tiptap/react")>()),
  BubbleMenu: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
let root: Root;
const buttons = () => [...document.querySelectorAll<HTMLButtonElement>("button")];
const name = (element: HTMLElement) => (element.getAttribute("aria-label") ?? element.textContent ?? "").trim();
const namedButton = (label: string) => {
  const button = buttons().find((element) => name(element) === label);
  expect(button, label).toBeDefined();
  return button!;
};

beforeEach(() => {
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
});
afterEach(async () => {
  await act(async () => root?.unmount());
  destroyEditors();
  document.body.replaceChildren();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

async function mount(content = "<p>Select me</p>") {
  const editor = await createEditor({ content });
  const host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  await act(async () =>
    root.render(
      <div>
        <EditorContent editor={editor} />
        <EditorBubbleMenu editor={editor} />
      </div>
    )
  );
  await act(async () => editor.commands.setTextSelection({ from: 1, to: 5 }));
  await vi.waitFor(() => expect(buttons().some((button) => name(button) === "Link")).toBe(true));
  return editor;
}

it("names formatting and alignment actions and changes the selected text through them", async () => {
  const editor = await mount();
  for (const label of ["Bold", "Italic", "Underline", "Strikethrough", "Left align", "Center align", "Right align"])
    expect(namedButton(label).type).toBe("button");
  await act(async () => namedButton("Bold").click());
  expect(editor.isActive("bold")).toBe(true);
  expect(namedButton("Bold").getAttribute("aria-pressed")).toBe("true");
  await act(async () => namedButton("Right align").click());
  expect(editor.getAttributes("paragraph").textAlign).toBe("right");
});

it.each([false, true])("names the link URL and its action when a link exists: %s", async (linked) => {
  const editor = await mount(linked ? '<p><a href="https://example.com">Select me</a></p>' : "<p>Select me</p>");
  await act(async () => namedButton("Link").click());
  const input = document.querySelector<HTMLInputElement>('input[type="url"]')!;
  expect(input.getAttribute("aria-label")).toBe("Link URL");
  expect(document.activeElement, "URL input takes focus when the portal mounts").toBe(input);
  const action = namedButton(linked ? "Remove link" : "Add link");
  expect(action.type).toBe("button");
  input.value = "https://example.org";
  await act(async () => action.click());
  expect(editor.isActive("link")).toBe(!linked);
  if (!linked) expect(editor.getAttributes("link").href).toBe("https://example.org");
});

it("distinguishes text and background swatches and their clear actions by name", async () => {
  const editor = await mount();
  await act(async () => namedButton("Color").click());
  for (const color of ["Gray", "Peach", "Pink", "Orange", "Green", "Light blue", "Dark blue", "Purple"])
    for (const kind of ["Text color", "Background color"]) expect(namedButton(`${kind}: ${color}`).type).toBe("button");
  const panel = document.querySelector('[role="dialog"]')!;
  expect([...panel.querySelectorAll("button")].every((button) => name(button).length > 0)).toBe(true);
  await act(async () => namedButton("Text color: Gray").click());
  expect(editor.getAttributes("customColor").color).toBe("gray");
  expect(namedButton("Text color: Gray").getAttribute("aria-pressed")).toBe("true");
  await act(async () => namedButton("Clear text color").click());
  expect(editor.getAttributes("customColor").color).toBeNull();
  expect(namedButton("Text color: Gray").getAttribute("aria-pressed")).toBe("false");
  await act(async () => namedButton("Background color: Peach").click());
  expect(editor.getAttributes("customColor").backgroundColor).toBe("peach");
  await act(async () => namedButton("Clear background color").click());
  expect(editor.getAttributes("customColor").backgroundColor).toBeNull();
});

it("keeps hovered editor links free of inline event attributes and dismisses the preview", async () => {
  const editor = await createEditor({ content: '<p><a href="https://example.com">Link words</a></p>' });
  const host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  await act(async () =>
    root.render(
      <>
        <EditorContent editor={editor} />
        <LinkViewContainer editor={editor} containerRef={{ current: host }} />
      </>
    )
  );
  const link = editor.view.dom.querySelector("a")!;
  const hover = () =>
    act(async () => {
      link.dispatchEvent(new MouseEvent("mouseover", { bubbles: true }));
    });
  const preview = () => [...host.querySelectorAll("p")].find((p) => p.textContent === "https://example.com");
  await hover();
  await vi.waitFor(() => expect(preview()).toBeDefined());
  expect(link.getAttributeNames().filter((attribute) => /^on/i.test(attribute))).toEqual([]);
  await act(async () => {
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
  });
  await vi.waitFor(() => expect(preview()).toBeUndefined());
  await hover();
  await vi.waitFor(() => expect(preview()).toBeDefined());
  await act(async () => {
    document.body.dispatchEvent(new Event("pointerdown", { bubbles: true }));
  });
  await vi.waitFor(() => expect(preview()).toBeUndefined());
});
