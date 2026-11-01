/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, createElement } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, it, vi } from "vitest";
import { createEditor, destroyEditors } from "./editor-test-fixture";
import { EditorBubbleMenu } from "@/components/menus/bubble-menu/root";
import { LinkEditView } from "@/components/links/link-edit-view";
import { EditorContainer } from "@/components/editors/editor-container";
import { EditorContent } from "@tiptap/react";
import { EditorContentWrapper } from "@/components/editors/editor-content";
import { RowDragHandle } from "@/extensions/table/plugins/drag-handles/row/drag-handle";
import { ColumnDragHandle } from "@/extensions/table/plugins/drag-handles/column/drag-handle";
import { defaultEditorLabels, EditorLabelsContext } from "@/components/editor-labels";

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
afterEach(() => {
  destroyEditors();
  document.body.replaceChildren();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});
describe("the real editor's interactive hosts", () => {
  it.each(["unmount", "destroy", "active"] as const)("keeps an anchor scroll owned by its %s host", async (exit) => {
    const editor = await createEditor({ content: '<p data-id="retired-anchor">Target</p>' });
    const previousURL = window.location.href;
    window.history.replaceState(null, "", "#retired-anchor");
    const frames = new Map<number, FrameRequestCallback>();
    let nextFrame = 0;
    vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
      const id = ++nextFrame;
      frames.set(id, callback);
      return id;
    });
    vi.stubGlobal("cancelAnimationFrame", (id: number) => frames.delete(id));
    const nodeDOM = vi.spyOn(editor.view, "nodeDOM");
    const host = document.createElement("div");
    document.body.append(host);
    const root = createRoot(host);
    await act(async () =>
      root.render(
        <EditorContainer editor={editor} id="anchor" displayConfig={{}} editorContainerClassName="" isTouchDevice>
          <EditorContent editor={editor} />
        </EditorContainer>
      )
    );
    const target = editor.view.dom.querySelector('[data-id="retired-anchor"]')!;
    Object.defineProperty(target, "scrollIntoView", { value: vi.fn() });
    try {
      expect(frames.size).toBeGreaterThan(0);
      if (exit === "unmount") await act(async () => root.unmount());
      else if (exit === "destroy") await act(async () => editor.destroy());
      nodeDOM.mockClear();
      expect(() => {
        for (const callback of frames.values()) callback(0);
      }).not.toThrow();
      expect(nodeDOM).toHaveBeenCalledTimes(exit === "active" ? 1 : 0);
      expect(target.scrollIntoView).toHaveBeenCalledTimes(exit === "active" ? 1 : 0);
    } finally {
      if (exit !== "unmount") await act(async () => root.unmount());
      window.history.replaceState(null, "", previousURL);
    }
  });

  it("resizes a real image NodeView with keys and updates its consumer-provided language", async () => {
    const editor = await createEditor({
      content:
        '<image-component src="asset-1" width="200px" height="100px" aspectRatio="2" status="uploaded"></image-component>',
    });
    const host = document.createElement("div");
    document.body.append(host);
    const root = createRoot(host);
    const render = (resizeImage: string) =>
      act(async () =>
        root.render(
          createElement(
            EditorLabelsContext,
            {
              value: { ...defaultEditorLabels, resizeImage },
            },
            <EditorContainer editor={editor} id="image" displayConfig={{}} editorContainerClassName="" isTouchDevice>
              <EditorContent editor={editor} />
            </EditorContainer>
          )
        )
      );
    try {
      await render("Resize image");
      const img = host.querySelector<HTMLImageElement>("img.image-component")!;
      expect(img).not.toBeNull();
      expect(editor.state.doc.firstChild?.attrs.id).toEqual(expect.any(String));
      Object.defineProperties(img, { naturalWidth: { value: 200 }, naturalHeight: { value: 100 } });
      await act(async () => img.dispatchEvent(new Event("load")));
      const grip = host.querySelector<HTMLButtonElement>('button[aria-label="Resize image"]')!;
      expect(grip).not.toBeNull();
      grip.focus();
      await act(async () =>
        grip.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowRight", bubbles: true, cancelable: true }))
      );
      expect(editor.state.doc.firstChild?.attrs.width).toBe("210px");
      expect(editor.state.doc.firstChild?.attrs.height).toBe("105px");
      await act(async () =>
        grip.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowLeft", bubbles: true, cancelable: true }))
      );
      expect(editor.state.doc.firstChild?.attrs.width).toBe("200px");
      await render("调整图片大小");
      expect(host.querySelector('button[aria-label="Resize image"]')).toBeNull();
      expect(host.querySelector('button[aria-label="调整图片大小"]')).toBe(grip);
      await act(async () => grip.click());
      expect(editor.state.doc.firstChild?.attrs.width).toBe("210px");
    } finally {
      await act(async () => root.unmount());
    }
  });

  it.each(["row", "column"] as const)("keeps %s menu navigation available and ignores an IME Escape", async (kind) => {
    vi.stubGlobal(
      "ResizeObserver",
      class {
        observe() {}
        unobserve() {}
        disconnect() {}
      }
    );
    const editor = await createEditor({ content: "<table><tbody><tr><td><p>Cell</p></td></tr></tbody></table>" });
    editor.commands.setTextSelection(3);
    const host = document.createElement("div");
    document.body.append(host);
    const root = createRoot(host);
    await act(async () =>
      root.render(
        kind === "row"
          ? createElement(RowDragHandle, { editor, row: 0 })
          : createElement(ColumnDragHandle, { editor, col: 0 })
      )
    );
    try {
      const trigger = host.querySelector("button")!;
      await act(async () => trigger.click());
      expect(document.body.textContent).toContain(kind === "row" ? "Insert above" : "Insert left");
      for (const button of document.querySelectorAll<HTMLButtonElement>('[role="dialog"] button'))
        expect((button.getAttribute("aria-label") ?? button.textContent ?? "").trim()).not.toBe("");
      await vi.waitFor(() =>
        expect(document.activeElement?.textContent).toBe(kind === "row" ? "Header row" : "Header column")
      );
      await act(async () =>
        trigger.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", isComposing: true, bubbles: true }))
      );
      expect(document.body.textContent).toContain(kind === "row" ? "Insert above" : "Insert left");
      await act(async () => trigger.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true })));
      expect(document.body.textContent).not.toContain(kind === "row" ? "Insert above" : "Insert left");
      expect(document.activeElement).toBe(trigger);
      // A table can lose its visual drag markers while its handle is still mounted.
      // Mouseup must end ownership even when there is nothing left to paint.
      for (const marker of editor.view.dom.querySelectorAll(
        ".table-drop-marker,.table-row-drag-marker,.table-col-drag-marker"
      ))
        marker.remove();
      const added = vi.spyOn(window, "addEventListener");
      const removed = vi.spyOn(window, "removeEventListener");
      await act(async () => trigger.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true })));
      const dragListeners = added.mock.calls.filter(([type]) => ["mousemove", "mouseup"].includes(type));
      expect(dragListeners).toHaveLength(2);
      await act(async () => window.dispatchEvent(new MouseEvent("mouseup")));
      for (const [type, listener] of dragListeners)
        expect(removed.mock.calls.some(([eventType, handler]) => eventType === type && handler === listener)).toBe(
          true
        );
    } finally {
      await act(async () => root.unmount());
    }
  });

  it.each(["row", "column"] as const)(
    "does not redirect a real %s widget's focus from the content wrapper",
    async (kind) => {
      const editor = await createEditor({
        content: "<p>Before</p><table><tbody><tr><td><p>Cell</p></td></tr></tbody></table>",
      });
      const host = document.createElement("div");
      document.body.append(host);
      const root = createRoot(host);
      await act(async () => root.render(<EditorContentWrapper editor={editor} id="table-focus" tabIndex={0} />));
      try {
        await act(async () => editor.commands.setTextSelection(editor.state.doc.firstChild!.nodeSize + 4));
        const trigger = host.querySelector<HTMLButtonElement>(`button[aria-label="Table ${kind} options"]`)!;
        expect(trigger).not.toBeNull();
        await act(async () => {
          trigger.focus();
          // TipTap's focus command runs in a frame; let it run before asserting that it did not steal widget focus.
          await new Promise(requestAnimationFrame);
        });
        expect(document.activeElement).toBe(trigger);
        const wrapper = host.firstElementChild as HTMLElement;
        await act(async () => {
          wrapper.focus();
          await new Promise(requestAnimationFrame);
        });
        expect(document.activeElement).toBe(editor.view.dom);
      } finally {
        await act(async () => root.unmount());
      }
    }
  );

  it.each(["", "<p></p>"])(
    "offers a native action to continue after a final table from prefix %s without submitting the enclosing form",
    async (prefix) => {
      const editor = await createEditor({
        content: prefix + "<table><tbody><tr><td><p>Last</p></td></tr></tbody></table>",
      });
      const host = document.createElement("div");
      document.body.append(host);
      const root = createRoot(host);
      const submit = vi.fn();
      await act(async () =>
        root.render(
          createElement(
            "form",
            { onSubmit: submit },
            <EditorContainer
              editor={editor}
              id="description"
              displayConfig={{}}
              editorContainerClassName=""
              isTouchDevice
            >
              <EditorContent editor={editor} />
            </EditorContainer>
          )
        )
      );
      try {
        const button = host.querySelector<HTMLButtonElement>('button[aria-label="Continue writing"]');
        expect(button).not.toBeNull();
        button!.focus();
        await act(async () => button!.click());
        await vi.waitFor(() => expect(editor.isFocused).toBe(true));
        expect(editor.state.doc.lastChild?.type.name).toBe("paragraph");
        expect(submit).not.toHaveBeenCalled();
      } finally {
        await act(async () => root.unmount());
      }
    }
  );

  it("releases a bubble menu's unfinished selection listeners when its host unmounts", async () => {
    const editor = await createEditor({ content: "<p>Select me</p>" });
    const host = document.createElement("div");
    document.body.append(host);
    const root = createRoot(host);
    const added = vi.spyOn(document, "addEventListener");
    const removed = vi.spyOn(document, "removeEventListener");
    await act(async () => root.render(createElement("div", null, createElement(EditorBubbleMenu, { editor }))));
    added.mockClear();
    await act(async () => document.body.dispatchEvent(new MouseEvent("mousedown", { bubbles: true })));
    const selecting = added.mock.calls.filter(([type]) => ["mousemove", "mouseup"].includes(type));
    expect(selecting.length).toBeGreaterThanOrEqual(2);
    await act(async () => root.unmount());
    for (const [type, listener] of selecting)
      expect(
        removed.mock.calls.some(
          ([removedType, removedListener]) => removedType === type && removedListener === listener
        )
      ).toBe(true);
  });

  it("does not submit a link or its containing form during IME composition", async () => {
    const editor = await createEditor({ content: '<p><a href="https://example.com">Link</a></p>' });
    const host = document.createElement("div");
    document.body.append(host);
    const root = createRoot(host);
    const closed = vi.fn();
    const submit = vi.fn();
    await act(async () =>
      root.render(
        createElement(
          "form",
          { onSubmit: submit },
          createElement(LinkEditView, {
            viewProps: { editor, from: 1, to: 5, url: "https://example.com", text: "Link", closeLinkView: closed },
            switchView: () => {},
          })
        )
      )
    );
    try {
      const input = host.querySelector("input")!;
      await act(async () =>
        input.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", isComposing: true, bubbles: true }))
      );
      expect(closed).not.toHaveBeenCalled();
      expect(submit).not.toHaveBeenCalled();
      const enter = new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true });
      await act(async () => input.dispatchEvent(enter));
      expect(enter.defaultPrevented).toBe(true);
      expect(closed).toHaveBeenCalledTimes(1);
      expect(submit).not.toHaveBeenCalled();
      expect(host.querySelector("label")?.htmlFor).toBe(input.id);
      expect(input.id).not.toBe("");
    } finally {
      await act(async () => root.unmount());
    }
  });

  it("releases the side menu DOM and global drag listeners when its editor is destroyed", async () => {
    const windowAdded = vi.spyOn(window, "addEventListener");
    const windowRemoved = vi.spyOn(window, "removeEventListener");
    const documentAdded = vi.spyOn(document, "addEventListener");
    const documentRemoved = vi.spyOn(document, "removeEventListener");
    const editor = await createEditor({ content: "<p>Retired editor</p>" });
    const menu = document.querySelector("#editor-side-menu")!;
    expect(menu.querySelector("#drag-handle")).not.toBeNull();
    editor.destroy();
    expect(menu.isConnected).toBe(false);
    for (const [type, listener] of windowAdded.mock.calls.filter(([eventType]) =>
      ["dragenter", "dragleave"].includes(eventType)
    ))
      expect(
        windowRemoved.mock.calls.some(([removedType, removed]) => removedType === type && removed === listener)
      ).toBe(true);
    for (const [type, listener] of documentAdded.mock.calls.filter(([eventType]) =>
      ["dragover", "mousemove"].includes(eventType)
    ))
      expect(
        documentRemoved.mock.calls.some(([removedType, removed]) => removedType === type && removed === listener)
      ).toBe(true);
  });
});
