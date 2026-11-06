/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { Editor } from "@tiptap/core";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import { ACCEPTED_IMAGE_MIME_TYPES } from "@/constants/config";
import { imageInputError, validateImages } from "@/extensions/custom-image/input";
import type { EditorImages, TExtensions } from "@/types";
type Props = { disabledExtensions?: TExtensions[]; editor: Editor; images?: EditorImages };
/** Files the editor can take: all ordinary (not its own), a rejected image batch, or the images it accepts. */
type Batch = "ordinary" | "rejected" | { files: File[] };
export const DropHandlerPlugin = ({ disabledExtensions, editor, images }: Props): Plugin => {
  const classify = (data: DataTransfer | null | undefined): Batch | undefined => {
    if (
      !editor.isEditable ||
      !images?.write ||
      images.signal.aborted ||
      disabledExtensions?.includes("image") ||
      !data?.files?.length
    )
      return;
    const files = Array.from(data.files);
    if (files.every((file) => file.type && !file.type.startsWith("image/"))) return "ordinary";
    const directory = Array.from(data.items ?? []).some(
      (item) => item.kind === "file" && (!item.getAsFile() || item.webkitGetAsEntry?.()?.isDirectory)
    );
    const error = directory ? "invalid" : validateImages(files, images.write.maxBytes);
    imageInputError(editor, error);
    return error ? "rejected" : { files };
  };
  const accept = (data: DataTransfer | null | undefined): File[] | undefined => {
    const batch = classify(data);
    return typeof batch === "object" ? batch.files : undefined;
  };
  return new Plugin({
    key: new PluginKey("drop-handler-plugin"),
    props: {
      handleDOMEvents: {
        paste(view, event) {
          if (!event.clipboardData?.files?.length) return false;
          if (
            (!images?.write || disabledExtensions?.includes("image")) &&
            ["text/nerve-editor-html", "text/html", "text/plain"].some((type) => event.clipboardData!.getData(type))
          )
            return false;
          const batch = classify(event.clipboardData);
          // Ordinary files are not the editor's to take: ProseMirror pastes the text or HTML that came with them.
          if (batch === "ordinary") return false;
          // Without the image capability ProseMirror would treat the file-only empty slice as a paste and prevent
          // the native event; returning leaves the files unprevented for their owning consumer.
          if (!batch) return true;
          // An image batch is taken whole, accepted or rejected, so neither ProseMirror nor the browser pastes it.
          event.preventDefault();
          if (batch !== "rejected")
            insertFilesSafely({
              disabledExtensions,
              editor,
              files: batch.files,
              initialPos: view.state.selection.from,
              event: "drop",
            });
          return true;
        },
      },
      handlePaste(view, event) {
        const files = accept(event.clipboardData);
        if (!files) return false;
        event.preventDefault();
        return insertFilesSafely({
          disabledExtensions,
          editor,
          files,
          initialPos: view.state.selection.from,
          event: "drop",
        });
      },
      handleDrop(view, event, _slice, moved) {
        if (moved) return false;
        const files = accept(event.dataTransfer);
        if (!files) return false;
        const pos = view.posAtCoords({ left: event.clientX, top: event.clientY })?.pos;
        if (pos === undefined) return false;
        event.preventDefault();
        return insertFilesSafely({ disabledExtensions, editor, files, initialPos: pos, event: "drop" });
      },
    },
  });
};
export function insertFilesSafely({
  disabledExtensions,
  editor,
  files,
  initialPos,
  event,
}: {
  disabledExtensions?: TExtensions[];
  editor: Editor;
  event: "insert" | "drop";
  files: File[];
  initialPos: number;
}): boolean {
  const images = editor.storage.imageComponent?.images;
  if (!editor.isEditable || !images?.write || images.signal.aborted || disabledExtensions?.includes("image"))
    return false;
  const error = validateImages(files, images.write.maxBytes);
  imageInputError(editor, error);
  if (error) return false;
  let pos = initialPos;
  for (const file of files) {
    if (!ACCEPTED_IMAGE_MIME_TYPES.includes(file.type)) return false;
    if (!editor.commands.insertImageComponent({ file, pos: Math.min(pos, editor.state.doc.content.size), event }))
      return false;
    pos++;
  }
  return true;
}
