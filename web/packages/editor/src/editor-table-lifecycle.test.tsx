/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, expect, it, vi } from "vitest";
import { EditorContentWrapper } from "@/components/editors/editor-content";
import { defaultEditorLabels, EditorLabelsContext } from "@/components/editor-labels";
import { getEditorRefHelpers } from "@/helpers/editor-ref";
import { createEditor, destroyEditors, getEditorMetaData } from "./editor-test-fixture";

globalThis.IS_REACT_ACT_ENVIRONMENT = true;
afterEach(() => {
  destroyEditors();
  document.body.replaceChildren();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

it("an editor destroyed in the task that built it leaves no table pass to run on its destroyed view", () => {
  vi.useFakeTimers();
  try {
    // A host may replace its editor before the next task, e.g. when a capability it was built with changes.
    void createEditor({ content: "<p>Before</p><table><tbody><tr><td><p>Cell</p></td></tr></tbody></table>" });
    destroyEditors();
    expect(() => vi.runOnlyPendingTimers()).not.toThrow();
  } finally {
    vi.useRealTimers();
  }
});

for (const kind of ["row", "column"] as const) {
  it(`updates the mounted ${kind} widget and its portal when the host language changes`, async () => {
    vi.stubGlobal(
      "ResizeObserver",
      class {
        observe() {}
        unobserve() {}
        disconnect() {}
      }
    );
    const editor = await createEditor({ content: "<table><tbody><tr><td><p>Cell</p></td></tr></tbody></table>" });
    const host = document.createElement("div");
    document.body.append(host);
    const root = createRoot(host);
    const render = (labels: typeof defaultEditorLabels) =>
      root.render(
        <EditorLabelsContext.Provider value={labels}>
          <EditorContentWrapper editor={editor} id="translated-table" />
        </EditorLabelsContext.Provider>
      );
    await act(async () => render(defaultEditorLabels));
    try {
      await act(async () => editor.commands.setTextSelection(4));
      const trigger = host.querySelector<HTMLButtonElement>(`button[aria-label="Table ${kind} options"]`)!;
      expect(trigger).not.toBeNull();
      await act(async () => trigger.click());
      const translated = {
        ...defaultEditorLabels,
        tableRowOptions: "表格行选项",
        tableColumnOptions: "表格列选项",
        tableColors: { ...defaultEditorLabels.tableColors, gray: "灰色", none: "无背景色" },
      };
      await act(async () => render(translated));
      const name = kind === "row" ? translated.tableRowOptions : translated.tableColumnOptions;
      expect(trigger.getAttribute("aria-label")).toBe(name);
      const panel = document.querySelector(`[role="dialog"][aria-label="${name}"]`)!;
      expect(panel).not.toBeNull();
      expect(panel.querySelector('button[aria-label="灰色"]')).not.toBeNull();
      expect(panel.querySelector('button[aria-label="无背景色"]')).not.toBeNull();
      expect(host.querySelector(`button[aria-label="${name}"]`)).toBe(trigger);
    } finally {
      await act(async () => root.unmount());
    }
  });

  it.each(["leave", "readonly", "destroy"] as const)(
    `releases the real ${kind} menu when its table owner is retired by %s`,
    async (exit) => {
      vi.stubGlobal(
        "ResizeObserver",
        class {
          observe() {}
          unobserve() {}
          disconnect() {}
        }
      );
      const editor = await createEditor({
        content: "<p>Before</p><table><tbody><tr><td><p>Cell</p></td></tr></tbody></table><p>After</p>",
      });
      const host = document.createElement("div");
      document.body.append(host);
      const root = createRoot(host);
      const api = getEditorRefHelpers({ editor, getEditorMetaData });
      await act(async () => root.render(<EditorContentWrapper editor={editor} id="table-owner" />));
      try {
        await act(async () => editor.commands.setTextSelection(editor.state.doc.firstChild!.nodeSize + 4));
        const trigger = host.querySelector<HTMLButtonElement>(`button[aria-label="Table ${kind} options"]`)!;
        expect(trigger).not.toBeNull();
        await act(async () => trigger.click());
        await vi.waitFor(() =>
          expect(document.querySelector(`[role="dialog"][aria-label="Table ${kind} options"]`)).not.toBeNull()
        );
        expect(api.isAnyDropbarOpen()).toBe(true);
        await act(async () => {
          if (exit === "leave") editor.commands.setTextSelection(editor.state.doc.content.size - 1);
          else if (exit === "readonly") editor.setEditable(false);
          else editor.destroy();
        });
        await vi.waitFor(() =>
          expect(document.querySelector(`[role="dialog"][aria-label="Table ${kind} options"]`)).toBeNull()
        );
        expect(api.isAnyDropbarOpen()).toBe(false);
        if (exit !== "destroy") {
          expect(host.querySelector(`button[aria-label="Table ${kind} options"]`)).toBeNull();
          if (exit === "leave") {
            // A later selection gets one fresh owner, with no old portal left behind.
            await act(async () => editor.commands.setTextSelection(editor.state.doc.firstChild!.nodeSize + 4));
            expect(host.querySelectorAll(`button[aria-label="Table ${kind} options"]`)).toHaveLength(1);
          }
        }
      } finally {
        await act(async () => root.unmount());
      }
    }
  );
}
