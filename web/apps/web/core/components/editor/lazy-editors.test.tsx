// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, createRef, forwardRef, useEffect, useImperativeHandle, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { EditorRefApi } from "@nerve/editor";
import type { RichTextEditor } from "./rich-text/editor";
import { LayoutErrorBoundary } from "@/components/common/layout-error-boundary";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
type Props = ComponentProps<typeof RichTextEditor>;
// The implementation leaf exposes the public ref and lifecycle; these tests exercise
// the real wrappers' dynamic boundary, not TipTap's separately tested editing behavior.
const ReadyEditor = forwardRef(function ReadyEditor(props: Props, ref) {
  const { handleEditorReady } = props;
  useImperativeHandle(
    ref,
    () => ({
      isEditorReadyToDiscard: () => true,
      getMarkDown: () => props.initialValue,
    }),
    [props.initialValue]
  );
  useEffect(() => {
    handleEditorReady?.(true);
    return () => handleEditorReady?.(false);
  }, [handleEditorReady]);
  return (
    <button
      onClick={() => {
        props.onChange?.({}, "<p>Typed</p>");
        props.onEnterKeyPress?.();
      }}
    >
      {props.id}:{props.initialValue}
    </button>
  );
});
let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
beforeEach(() => {
  vi.resetModules();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  vi.restoreAllMocks();
  vi.doUnmock("./rich-text/editor-implementation");
  vi.doUnmock("./lite-text/editor-implementation");
});
async function load(kind: "rich" | "lite") {
  let resolve!: () => void;
  let reject!: (error: Error) => void;
  let requests = 0;
  const gate = new Promise<void>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  const module = kind === "rich" ? "./rich-text/editor-implementation" : "./lite-text/editor-implementation";
  vi.doMock(module, async () => {
    requests++;
    await gate;
    return { RichTextEditor: ReadyEditor, LiteTextEditor: ReadyEditor };
  });
  const Editor =
    kind === "rich"
      ? (await import("./rich-text/editor")).RichTextEditor
      : (await import("./lite-text/editor")).LiteTextEditor;
  return { Editor, resolve, reject, requests: () => requests };
}
for (const kind of ["rich", "lite"] as const) {
  it(`${kind} preserves current props, readiness and the forwarded ref across lazy loading`, async () => {
    const loading = await load(kind);
    const { Editor } = loading;
    const ready = vi.fn();
    const changed = vi.fn();
    const submitted = vi.fn();
    const ref = createRef<EditorRefApi>();
    const view = (initialValue: string) => (
      <Editor
        editable={false}
        workspaceSlug="ws"
        workspaceId="workspace"
        id="description"
        initialValue={initialValue}
        ref={ref}
        handleEditorReady={ready}
        onChange={changed}
        onEnterKeyPress={submitted}
      />
    );
    expect(loading.requests()).toBe(0);
    await act(async () => root.render(view("Old")));
    expect(ref.current).toBeNull();
    expect(ready).not.toHaveBeenCalled();
    expect(host.querySelector("button")).toBeNull();
    await act(async () => root.render(view("Current")));
    await act(async () => {
      loading.resolve();
      await vi.dynamicImportSettled();
    });
    expect(loading.requests()).toBe(1);
    expect(host.textContent).toBe("description:Current");
    expect(ref.current?.getMarkDown()).toBe("Current");
    expect(ref.current?.isEditorReadyToDiscard()).toBe(true);
    expect(ready).toHaveBeenCalledExactlyOnceWith(true);
    await act(async () => host.querySelector("button")!.click());
    expect(changed).toHaveBeenCalledExactlyOnceWith({}, "<p>Typed</p>");
    expect(submitted).toHaveBeenCalledTimes(1);
    await act(async () => root.render(null));
    expect(ref.current).toBeNull();
    expect(ready).toHaveBeenLastCalledWith(false);
  });
  it(`${kind} sends a failed import to the existing boundary without reporting a ready editor`, async () => {
    const loading = await load(kind);
    const { Editor } = loading;
    const ready = vi.fn();
    const logged = vi.spyOn(console, "error").mockImplementation(() => {});
    await act(async () =>
      root.render(
        <LayoutErrorBoundary>
          <Editor
            editable={false}
            workspaceSlug="ws"
            workspaceId="workspace"
            id="description"
            initialValue="Saved"
            handleEditorReady={ready}
          />
        </LayoutErrorBoundary>
      )
    );
    const error = new Error("editor import unavailable");
    await act(async () => {
      loading.reject(error);
      await vi.dynamicImportSettled();
    });
    expect(host.textContent).toContain("something_went_wrong");
    expect(ready).not.toHaveBeenCalled();
    expect(logged.mock.calls).toContainEqual([
      "Issue layout crashed",
      expect.objectContaining({ cause: error }),
      expect.anything(),
    ]);
  });
  it(`${kind} does not start the old editor when its delayed import completes after unmount`, async () => {
    const loading = await load(kind);
    const { Editor } = loading;
    const ready = vi.fn();
    const ref = createRef<EditorRefApi>();
    await act(async () =>
      root.render(
        <Editor
          editable={false}
          workspaceSlug="ws"
          workspaceId="workspace"
          id="old"
          initialValue="Old"
          ref={ref}
          handleEditorReady={ready}
        />
      )
    );
    await act(async () => root.render(null));
    await act(async () => {
      loading.resolve();
      await vi.dynamicImportSettled();
    });
    expect(host.textContent).toBe("");
    expect(ref.current).toBeNull();
    expect(ready).not.toHaveBeenCalled();
  });
}
