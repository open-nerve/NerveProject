/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { NodeSelection } from "@tiptap/pm/state";
import { afterEach, expect, it, vi } from "vitest";
import { createEditor, destroyEditors, getEditorMetaData } from "./editor-test-fixture";
import { getEditorRefHelpers } from "@/helpers/editor-ref";
import {
  fakeEditorImages,
  imageClipboard,
  mountImages,
  nativeImage,
  pasteFiles,
  SOURCE_IMAGE,
  TARGET_IMAGE,
} from "./fake-editor-images";
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
const closers: (() => Promise<void>)[] = [];
afterEach(async () => {
  await closers.splice(0).reduce((done, close) => done.then(close), Promise.resolve());
  destroyEditors();
  document.body.replaceChildren();
  vi.restoreAllMocks();
});
const fixture = async (content = "", editable = true, available = true, strict = false) => {
  const fake = fakeEditorImages();
  const editor = await createEditor({ content, editable, images: available ? fake.images : undefined });
  const mounted = await mountImages(editor, strict);
  closers.push(mounted.close);
  return { ...fake, editor, ...mounted, ref: getEditorRefHelpers({ editor, getEditorMetaData }) };
};
it.each([false, true])("only an editable consumer with images.write can insert: editable=%s", async (editable) => {
  const tab = await fixture("", editable, false);
  expect(tab.editor.commands.insertImageComponent({ event: "insert" })).toBe(false);
  expect(tab.editor.getHTML()).not.toContain("image-component");
  expect(tab.pending).toHaveLength(0);
});
const ordinaryFile = () => new File(["pdf"], "doc.pdf", { type: "application/pdf" });
it("leaves a clipboard of only ordinary files to ProseMirror, which pastes its text once", async () => {
  const tab = await fixture();
  await act(async () => {
    pasteFiles(tab.editor, [ordinaryFile()], false, { "text/plain": "doc.pdf" });
  });
  expect(tab.editor.getText()).toBe("doc.pdf");
  expect(tab.pending).toHaveLength(0);
  expect(tab.editor.getHTML()).not.toContain("image-component");
  expect(tab.host.querySelector('[role="alert"]')).toBeNull();
});
it.each(["mixed", "directory", "empty", "svg", "too-many"])(
  "takes a rejected %s image batch whole: nothing is inserted, not even its text, and one alert explains it",
  async (kind) => {
    const tab = await fixture();
    const files =
      kind === "mixed"
        ? [nativeImage(), ordinaryFile()]
        : kind === "empty"
          ? [new File(["png"], "image.png")]
          : kind === "svg"
            ? [new File(["svg"], "image.svg", { type: "image/svg+xml" })]
            : kind === "too-many"
              ? Array.from({ length: 11 }, nativeImage)
              : [nativeImage()];
    const names = files.map((file) => file.name).join("\n");
    let event!: Event;
    await act(async () => {
      event = pasteFiles(tab.editor, files, kind === "directory", {
        "text/plain": names,
        "text/html": `<p>${names}</p><img src="https://external.test/a.png">`,
      });
    });
    // Unprevented, the browser would paste that text and <img> natively, past ProseMirror's own paste handling.
    expect(event.defaultPrevented).toBe(true);
    expect(tab.pending).toHaveLength(0);
    expect(tab.editor.getText()).toBe("");
    expect(tab.editor.getHTML()).not.toContain("image-component");
    expect(tab.host.querySelectorAll('[role="alert"]')).toHaveLength(1);
  }
);
it.each(["pdf", "mixed"])("a dropped %s batch stays unprevented for the page and is not inserted", async (kind) => {
  const tab = await fixture();
  const view = tab.editor.view;
  vi.spyOn(view, "posAtCoords").mockReturnValue({ pos: 0, inside: -1 });
  const files = kind === "pdf" ? [ordinaryFile()] : [nativeImage(), ordinaryFile()];
  const event = new Event("drop", { bubbles: true, cancelable: true });
  Object.defineProperties(event, {
    dataTransfer: { value: { files, getData: () => "" } },
    clientX: { value: 10 },
    clientY: { value: 10 },
    ctrlKey: { value: false },
    altKey: { value: false },
  });
  await act(async () => view.dom.dispatchEvent(event));
  expect(event.defaultPrevented).toBe(false);
  expect(tab.pending).toHaveLength(0);
  expect(tab.editor.getHTML()).not.toContain("image-component");
});
it("native image selection publishes only the confirmed ID, with busy feedback and no percentage", async () => {
  const tab = await fixture();
  await act(async () => {
    tab.editor.commands.insertImageComponent({ event: "insert" });
  });
  const input = tab.host.querySelector<HTMLInputElement>('input[type="file"]')!;
  expect(input).not.toBeNull();
  Object.defineProperty(input, "files", { value: [nativeImage()], configurable: true });
  await act(async () => input.dispatchEvent(new Event("change", { bubbles: true })));
  expect(tab.pending).toHaveLength(1);
  expect(tab.pending[0]!.file?.name).toBe("image.png");
  expect(tab.ref.isEditorReadyToDiscard()).toBe(false);
  expect(tab.host.textContent).not.toContain("%");
  expect(tab.editor.getHTML()).not.toContain(TARGET_IMAGE);
  await act(async () => tab.pending[0]!.resolve(TARGET_IMAGE));
  expect(tab.editor.getHTML()).toContain(`src="${TARGET_IMAGE}"`);
  expect(tab.ref.isEditorReadyToDiscard()).toBe(true);
  expect(tab.read).toHaveBeenCalledWith(TARGET_IMAGE);
});
it.each(["text/nerve-editor-html", "text/html"])(
  "untrusted %s pasted UUIDs always duplicate, never save the source ID",
  async (kind) => {
    const tab = await fixture();
    await act(async () => {
      imageClipboard(
        tab.editor,
        `<p>Copied</p><image-component src="${SOURCE_IMAGE}" status="uploaded" id="forged" data-id="same-document"></image-component>`,
        kind
      );
    });
    expect(tab.pending).toHaveLength(1);
    expect(tab.pending[0]!.source).toBe(SOURCE_IMAGE);
    expect(tab.editor.getHTML()).not.toContain(SOURCE_IMAGE);
    expect(tab.ref.isEditorReadyToDiscard()).toBe(false);
    await act(async () => tab.pending[0]!.resolve(TARGET_IMAGE));
    expect(tab.editor.getHTML()).toContain(TARGET_IMAGE);
    expect(tab.editor.getText()).toContain("Copied");
    expect(tab.editor.getHTML()).not.toContain("grant=");
  }
);
it.each(["https://bad.test/a", "data:image/png;base64,AA", "blob:private", "javascript:alert(1)", "not-a-uuid"])(
  "never resolves an invalid clipboard identity %s",
  async (src) => {
    const tab = await fixture();
    await act(async () => {
      imageClipboard(tab.editor, `<image-component src="${src}" status="uploaded"></image-component>`);
    });
    expect(tab.pending).toHaveLength(0);
    expect(tab.read).not.toHaveBeenCalled();
    expect(tab.editor.getHTML()).not.toContain(src);
  }
);
it.each(["upload", "copy"])("removing a %s node aborts it and a late answer cannot insert it again", async (kind) => {
  const tab = await fixture();
  await act(async () => {
    if (kind === "copy") imageClipboard(tab.editor, `<image-component src="${SOURCE_IMAGE}"></image-component>`);
    else pasteFiles(tab.editor, [nativeImage()]);
  });
  expect(tab.pending).toHaveLength(1);
  const old = tab.pending[0]!;
  await act(async () => {
    tab.editor.commands.clearContent();
  });
  expect(old.operation.signal.aborted).toBe(true);
  await act(async () => old.resolve(TARGET_IMAGE));
  expect(tab.editor.getHTML()).not.toContain(TARGET_IMAGE);
  expect(tab.read).not.toHaveBeenCalledWith(TARGET_IMAGE);
});
it("Undo and Redo reuse a completed UUID without a new copy or deletion command", async () => {
  const tab = await fixture();
  await act(async () => {
    imageClipboard(tab.editor, `<image-component src="${SOURCE_IMAGE}"></image-component>`);
  });
  expect(tab.pending).toHaveLength(1);
  await act(async () => tab.pending[0]!.resolve(TARGET_IMAGE));
  const saved = tab.editor.getHTML();
  await act(async () => {
    tab.editor.commands.clearContent();
  });
  await act(async () => {
    tab.editor.commands.undo();
  });
  expect(tab.editor.getHTML()).toBe(saved);
  await act(async () => {
    tab.editor.commands.redo();
    tab.editor.commands.undo();
  });
  expect(tab.editor.getHTML()).toBe(saved);
  expect(tab.pending).toHaveLength(1);
});
it("address renewal and actual image load do not change the document; abort removes every resolved URL", async () => {
  const tab = await fixture(`<image-component src="${SOURCE_IMAGE}" width="35%"></image-component>`);
  const before = tab.editor.getHTML();
  const updated = vi.fn();
  tab.editor.on("update", updated);
  let img = tab.host.querySelector<HTMLImageElement>("img.image-component")!;
  expect(img).not.toBeNull();
  Object.defineProperties(img, { naturalWidth: { value: 200 }, naturalHeight: { value: 100 } });
  await act(async () => img.dispatchEvent(new Event("load")));
  expect(tab.loaded).toHaveBeenCalledTimes(1);
  expect(tab.editor.getHTML()).toBe(before);
  await act(async () => {
    tab.publish({ phase: "ready", url: "/assets/storage/local/image?grant=2", expiresAt: "2099-01-01T00:00:00Z" });
    img.dispatchEvent(new Event("error"));
    img.dispatchEvent(new Event("load"));
  });
  expect(tab.failed).not.toHaveBeenCalled();
  expect(tab.loaded).toHaveBeenCalledTimes(1);
  img = tab.host.querySelector("img.image-component")!;
  expect(img.getAttribute("src")).toContain("grant=2");
  expect(updated).not.toHaveBeenCalled();
  await act(async () => tab.controller.abort());
  expect(tab.host.querySelector("img[src]")).toBeNull();
  expect(tab.editor.getHTML()).toBe(before);
});

it("download failures and cancellation never open a new tab or report success", async () => {
  const tab = await fixture(`<image-component src="${SOURCE_IMAGE}"></image-component>`);
  const opened = vi.spyOn(window, "open");
  const download = vi.mocked(tab.images.download);
  download.mockRejectedValueOnce(new Error("Denied"));
  const click = (label: string) =>
    act(async () => {
      tab.host.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`)!.click();
    });
  await click("Download image");
  expect(tab.host.querySelector('[role="alert"]')?.textContent).toContain("could not be downloaded");
  let signal!: AbortSignal;
  let reject!: (error: Error) => void;
  download.mockImplementationOnce((_id, received) => {
    signal = received;
    return new Promise((_resolve, failed) => {
      reject = failed;
    });
  });
  await click("Download image");
  expect(tab.host.querySelector<HTMLButtonElement>('button[aria-label="Downloading image"]')!.disabled).toBe(true);
  await act(async () => {
    Array.from(tab.host.querySelectorAll("button"))
      .find((button) => button.textContent === "Cancel")!
      .click();
    reject(new Error("Cancelled"));
  });
  expect(signal.aborted).toBe(true);
  expect(tab.host.querySelector('[role="alert"]')).toBeNull();
  expect(opened).not.toHaveBeenCalled();
});
it("unknown copy is an explicit failure, remounting never retries it automatically", async () => {
  const tab = await fixture();
  await act(async () => {
    imageClipboard(tab.editor, `<image-component src="${SOURCE_IMAGE}"></image-component>`);
  });
  expect(tab.pending).toHaveLength(1);
  await act(async () => tab.pending[0]!.reject(new Error("Could not confirm copying")));
  expect(tab.host.querySelector('[role="alert"]')?.textContent).toContain("Could not confirm copying");
  expect(tab.editor.getHTML()).not.toContain(SOURCE_IMAGE);
  await tab.close();
  closers.pop();
  const remounted = await mountImages(tab.editor);
  closers.push(remounted.close);
  expect(tab.pending).toHaveLength(1);
  expect(tab.ref.isEditorReadyToDiscard()).toBe(false);
});

it("revoking the image capability aborts pending work immediately", async () => {
  const tab = await fixture();
  await act(async () => {
    pasteFiles(tab.editor, [nativeImage()]);
  });
  const pending = tab.pending[0]!;
  await act(async () => tab.controller.abort());
  expect(pending.operation.signal.aborted).toBe(true);
  await act(async () => pending.resolve(TARGET_IMAGE));
  expect(tab.editor.getHTML()).not.toContain(TARGET_IMAGE);
});

it("becoming read-only cannot publish a late image result", async () => {
  const tab = await fixture();
  await act(async () => {
    pasteFiles(tab.editor, [nativeImage()]);
  });
  const pending = tab.pending[0]!;
  await act(async () => {
    tab.editor.setEditable(false);
    pending.resolve(TARGET_IMAGE);
  });
  expect(tab.editor.getHTML()).not.toContain(TARGET_IMAGE);
});

it("StrictMode starts an inserted transfer once and a replacement cancels the old file", async () => {
  const tab = await fixture("", true, true, true);
  await act(async () => {
    pasteFiles(tab.editor, [nativeImage()]);
  });
  expect(tab.pending).toHaveLength(1);
  const first = tab.pending[0]!;
  const input = tab.host.querySelector<HTMLInputElement>('input[type="file"]')!;
  Object.defineProperty(input, "files", { value: [new File(["new"], "new.png", { type: "image/png" })] });
  await act(async () => input.dispatchEvent(new Event("change", { bubbles: true })));
  expect(first.operation.signal.aborted).toBe(true);
  expect(tab.pending).toHaveLength(2);
  await act(async () => first.resolve(SOURCE_IMAGE));
  expect(tab.editor.getHTML()).not.toContain(SOURCE_IMAGE);
  await act(async () => tab.pending[1]!.resolve(TARGET_IMAGE));
  expect(tab.editor.getHTML()).toContain(TARGET_IMAGE);
});

it("moving a completed image inside the document preserves its UUID without copying", async () => {
  const tab = await fixture(`<image-component src="${SOURCE_IMAGE}"></image-component><p>After</p>`);
  const view = tab.editor.view;
  view.dispatch(view.state.tr.setSelection(NodeSelection.create(view.state.doc, 0)));
  const selection = view.state.selection;
  view.dragging = { slice: selection.content(), move: true };
  vi.spyOn(view, "posAtCoords").mockReturnValue({ pos: view.state.doc.content.size, inside: -1 });
  const event = new Event("drop", { bubbles: true, cancelable: true });
  Object.defineProperties(event, {
    dataTransfer: { value: { files: [], getData: () => "" } },
    clientX: { value: 10 },
    clientY: { value: 10 },
    ctrlKey: { value: false },
    altKey: { value: false },
  });
  await act(async () => view.dom.dispatchEvent(event));
  expect(tab.pending).toHaveLength(0);
  expect(tab.editor.getJSON().content?.at(-1)?.attrs?.src).toBe(SOURCE_IMAGE);
});
