/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { EditorContent } from "@tiptap/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { BlockMenu } from "@/components/menus/block-menu";
import { getEditorRefHelpers } from "@/helpers/editor-ref";
import { createEditor, destroyEditors, getEditorMetaData } from "./editor-test-fixture";

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
const roots: Root[] = [];
const named = (name: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].filter((b) => b.textContent?.trim() === name);
beforeEach(() => {
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  // jsdom has no geometry; preserve the real selected node while invoking the real drag-handle listener.
  Object.defineProperty(document, "elementsFromPoint", { configurable: true, value: () => [] });
});
afterEach(async () => {
  await act(async () => {
    for (const root of roots.splice(0)) root.unmount();
  });
  destroyEditors();
  document.body.replaceChildren();
  Reflect.deleteProperty(document, "elementsFromPoint");
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});
async function mount(content: string, editable = true) {
  const editor = await createEditor({ content, editable });
  const host = document.createElement("div");
  document.body.append(host);
  const root = createRoot(host);
  roots.push(root);
  await act(async () =>
    root.render(
      <>
        <EditorContent editor={editor} />
        <BlockMenu editor={editor} />
      </>
    )
  );
  await act(async () => editor.commands.setNodeSelection(0));
  const handle = editor.view.dom.parentElement!.querySelector<HTMLButtonElement>("#drag-handle")!;
  expect(handle).not.toBeNull();
  const api = getEditorRefHelpers({ editor, getEditorMetaData });
  return { editor, root, handle, api };
}

it("opens only the clicked editor's block actions and duplicates only its selected block", async () => {
  const first = await mount("<p>First document</p>");
  const second = await mount("<p>Second document</p>");
  await act(async () => first.handle.click());
  expect(named("Duplicate")).toHaveLength(1);
  expect(first.api.isAnyDropbarOpen()).toBe(true);
  expect(second.api.isAnyDropbarOpen()).toBe(false);
  await act(async () => named("Duplicate")[0].click());
  expect(first.editor.state.doc.childCount).toBe(2);
  expect(first.editor.state.doc.child(1).textContent).toBe("First document");
  expect(second.editor.getText()).toBe("Second document");
});

it("does not open block mutation controls for a read-only editor", async () => {
  const current = await mount("<p>Read only</p>", false);
  await act(async () => current.handle.click());
  expect(named("Delete")).toHaveLength(0);
  expect(current.api.isAnyDropbarOpen()).toBe(false);
});

it.each(["outside", "unmount"] as const)("releases the public dropbar state after %s dismissal", async (exit) => {
  const current = await mount("<p>Open menu</p>");
  await act(async () => current.handle.click());
  expect(named("Delete")).toHaveLength(1);
  expect(current.api.isAnyDropbarOpen()).toBe(true);
  if (exit === "outside") {
    await act(async () => document.body.dispatchEvent(new Event("pointerdown", { bubbles: true })));
  } else {
    roots.splice(roots.indexOf(current.root), 1);
    await act(async () => current.root.unmount());
  }
  expect(named("Delete")).toHaveLength(0);
  expect(current.api.isAnyDropbarOpen()).toBe(false);
});

it("closes the old block menu when the same host receives another editor", async () => {
  const first = await mount("<p>Old editor</p>");
  const replacement = await createEditor({ content: "<p>New editor</p>" });
  await act(async () => first.handle.click());
  expect(named("Delete")).toHaveLength(1);
  await act(async () =>
    first.root.render(
      <>
        <EditorContent editor={replacement} />
        <BlockMenu editor={replacement} />
      </>
    )
  );
  expect(named("Delete")).toHaveLength(0);
  expect(first.api.isAnyDropbarOpen()).toBe(false);
  expect(getEditorRefHelpers({ editor: replacement, getEditorMetaData }).isAnyDropbarOpen()).toBe(false);
});

it("offers a named code-copy action that copies the actual code block text", async () => {
  const current = await mount("<p></p>");
  await act(async () => current.editor.commands.setContent("<pre><code>const answer = 42;</code></pre>"));
  const clipboard = { writeText: vi.fn(async () => {}) };
  Object.defineProperty(navigator, "clipboard", { configurable: true, value: clipboard });
  try {
    const button = document.querySelector<HTMLButtonElement>('button[aria-label="Copy code"]');
    expect(button).not.toBeNull();
    await act(async () => button!.click());
    expect(clipboard.writeText).toHaveBeenCalledExactlyOnceWith("const answer = 42;");
    expect(current.editor.getText()).toBe("const answer = 42;");
  } finally {
    Reflect.deleteProperty(navigator, "clipboard");
  }
});

it("names the block handle and reveals it when reached by keyboard", async () => {
  const current = await mount("<p>Keyboard block</p>");
  // jsdom does not track the browser's input modality for :focus-visible.
  const matches = current.handle.matches.bind(current.handle);
  vi.spyOn(current.handle, "matches").mockImplementation((selector) =>
    selector === ":focus-visible" ? true : matches(selector)
  );
  await act(async () => current.handle.focus());
  expect(current.handle.getAttribute("aria-label")).toBe("Block actions");
  expect(current.handle.classList.contains("drag-handle-hidden")).toBe(false);
  expect(current.handle.closest("#editor-side-menu")!.classList.contains("side-menu-hidden")).toBe(false);
});

it("does not move a pointer-focused handle away from its hovered block", async () => {
  const current = await mount("<p>Caret block</p><p>Hovered block</p>");
  const sideMenu = current.handle.closest<HTMLElement>("#editor-side-menu")!;
  sideMenu.style.top = "240px";
  // jsdom does not track the browser's input modality for :focus-visible.
  const matches = current.handle.matches.bind(current.handle);
  vi.spyOn(current.handle, "matches").mockImplementation((selector) =>
    selector === ":focus-visible" ? false : matches(selector)
  );
  await act(async () => current.handle.focus());
  expect(sideMenu.style.top).toBe("240px");
});

it("duplicates the caret's whole block on keyboard activation without consulting pointer coordinates", async () => {
  const current = await mount("<p>First block</p><p>Second block</p>");
  await act(async () => current.editor.commands.setTextSelection(current.editor.state.doc.child(0).nodeSize + 4));
  await act(async () => current.handle.click());
  await act(async () => named("Duplicate")[0].click());
  expect(current.editor.state.doc.childCount).toBe(3);
  expect(current.editor.state.doc.child(0).textContent).toBe("First block");
  expect(current.editor.state.doc.child(1).textContent).toBe("Second block");
  expect(current.editor.state.doc.child(2).textContent).toBe("Second block");
});

it.each(["<p>Last block</p>", "<blockquote><p>Last block</p></blockquote>"])(
  "leaves a valid text cursor in the duplicated last block: %s",
  async (content) => {
    const current = await mount(content);
    const originalSize = current.editor.state.doc.content.size;
    await act(async () => current.handle.click());
    await act(async () => named("Duplicate")[0].click());
    expect(current.editor.state.selection.$from.parent.inlineContent).toBe(true);
    expect(current.editor.state.selection.from).toBeGreaterThan(originalSize);
    await act(async () => current.editor.commands.insertContent(" appended"));
    expect(current.editor.state.doc.child(0).textContent).toBe("Last block");
    expect(current.editor.state.doc.child(1).textContent).toBe("Last block appended");
  }
);

it("focuses block actions and returns to the handle on Escape without dismissing an outer layer", async () => {
  const current = await mount("<p>Focus block</p>");
  await act(async () => current.handle.click());
  await vi.waitFor(() => expect(document.activeElement).toBe(named("Delete")[0]));
  const outerEscape = vi.fn();
  document.addEventListener("keydown", outerEscape);
  try {
    await act(async () =>
      named("Delete")[0].dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }))
    );
    expect(named("Delete")).toHaveLength(0);
    expect(current.api.isAnyDropbarOpen()).toBe(false);
    expect(outerEscape).not.toHaveBeenCalled();
    await vi.waitFor(() => expect(document.activeElement).toBe(current.handle));
  } finally {
    document.removeEventListener("keydown", outerEscape);
  }
});

it.each([{ isComposing: true }, { keyCode: 229 }])("preserves block actions on input-method Escape %j", async (ime) => {
  const current = await mount("<p>Composing block</p>");
  await act(async () => current.handle.click());
  await act(async () =>
    named("Delete")[0].dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, ...ime }))
  );
  expect(named("Delete")).toHaveLength(1);
  expect(current.api.isAnyDropbarOpen()).toBe(true);
  expect(current.editor.getText()).toBe("Composing block");
});
