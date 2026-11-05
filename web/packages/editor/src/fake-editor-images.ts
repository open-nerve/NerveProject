/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { EditorContentWrapper } from "@/components/editors/editor-content";
import { createElement } from "react";
import { vi, type Mock } from "vitest";
import type { Editor } from "@tiptap/core";
import type { EditorImages, EditorImageSnapshot, ImageOperation } from "@/types";
export const SOURCE_IMAGE = "11111111-1111-4111-8111-111111111111";
export const TARGET_IMAGE = "22222222-2222-4222-8222-222222222222";
type PendingImage = {
  nodeID: string;
  source?: string;
  file?: File;
  operation: ImageOperation;
  resolve: (id: string) => void;
  reject: (error: Error) => void;
};
export function fakeEditorImages(): {
  images: EditorImages;
  controller: AbortController;
  pending: PendingImage[];
  loaded: Mock<() => void>;
  failed: Mock<() => void>;
  retry: Mock<() => void>;
  dispose: Mock<() => void>;
  read: Mock<EditorImages["read"]>;
  publish: (next: EditorImageSnapshot) => void;
} {
  const controller = new AbortController();
  let snapshot: EditorImageSnapshot = {
    phase: "ready",
    url: "/assets/storage/local/image?grant=1",
    expiresAt: "2099-01-01T00:00:00Z",
  };
  const listeners = new Set<() => void>();
  const pending: PendingImage[] = [];
  const transfer = (nodeID: string, operation: ImageOperation, source?: string, file?: File) =>
    new Promise<string>((resolve, reject) => {
      operation.onPhase("preparing");
      pending.push({ nodeID, operation, source, file, resolve, reject });
    });
  const loaded = vi.fn(),
    failed = vi.fn(),
    retry = vi.fn(),
    dispose = vi.fn();
  const read = vi.fn((_id: string) => ({
    getSnapshot: () => snapshot,
    subscribe: (changed: () => void) => {
      listeners.add(changed);
      return () => {
        listeners.delete(changed);
      };
    },
    loaded,
    failed,
    retry,
    dispose,
  }));
  const images: EditorImages = {
    signal: controller.signal,
    read,
    download: vi.fn(async () => {}),
    write: {
      maxBytes: 100,
      upload: vi.fn((id, file, op) => transfer(id, op, undefined, file)),
      duplicate: vi.fn((id, source, op) => transfer(id, op, source)),
      retryUpload: vi.fn((id, op) => transfer(id, op)),
      cancel: vi.fn(),
    },
  };
  return {
    images,
    controller,
    pending,
    loaded,
    failed,
    retry,
    dispose,
    read,
    publish: (next: EditorImageSnapshot) => {
      snapshot = next;
      for (const changed of listeners) changed();
    },
  };
}
export async function mountImages(editor: Editor, strict = false) {
  const host = document.createElement("div");
  host.className = "editor-container";
  document.body.append(host);
  const root = createRoot(host);
  const content = createElement(EditorContentWrapper, { editor, id: "images" });
  await act(async () => root.render(strict ? createElement(StrictMode, null, content) : content));
  return { host, close: () => act(async () => root.unmount()) };
}
export function imageClipboard(editor: Editor, html: string, kind = "text/nerve-editor-html") {
  const event = new Event("paste", { bubbles: true, cancelable: true });
  Object.defineProperty(event, "clipboardData", {
    value: { files: [], getData: (type: string) => (type === kind ? html : "") },
  });
  editor.view.dom.dispatchEvent(event);
  return event;
}
export function pasteFiles(editor: Editor, files: File[], directory = false, text: Record<string, string> = {}) {
  const event = new Event("paste", { bubbles: true, cancelable: true });
  Object.defineProperty(event, "clipboardData", {
    value: {
      files,
      items: files.map((file) => ({
        kind: "file",
        type: file.type,
        getAsFile: () => file,
        webkitGetAsEntry: () => ({ isDirectory: directory }),
      })),
      getData: (type: string) => text[type] ?? "",
    },
  });
  editor.view.dom.dispatchEvent(event);
  return event;
}
export const nativeImage = () => new File(["png"], "image.png", { type: "image/png" });
