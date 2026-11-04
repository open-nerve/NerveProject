/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Editor } from "@tiptap/core";
import { ACCEPTED_IMAGE_MIME_TYPES } from "@/constants/config";
export type ImageInputError = "invalid" | "size" | "name" | "tooMany";
export const isImageID = (value: unknown): value is string =>
  typeof value === "string" && /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(value);
export function validateImages(files: readonly File[], maxBytes: number): ImageInputError | undefined {
  if (files.length < 1) return "invalid";
  if (files.length > 10) return "tooMany";
  for (const file of files) {
    if (!ACCEPTED_IMAGE_MIME_TYPES.includes(file.type)) return "invalid";
    if (!Number.isSafeInteger(maxBytes) || maxBytes < 1 || file.size < 1 || file.size > maxBytes) return "size";
    if (!file.name || [...file.name].length > 255 || /[\p{Cc}/\\]/u.test(file.name)) return "name";
  }
  return undefined;
}
export function imageDocumentReady(editor: Editor): boolean {
  let ready = true;
  editor.state.doc.descendants((node) => {
    if (node.type.name === "imageComponent" && !isImageID(node.attrs.src)) ready = false;
  });
  return ready;
}
export function imageInputError(editor: Editor, error: ImageInputError | undefined): void {
  const storage = editor.storage.imageComponent;
  if (!storage || editor.isDestroyed) return;
  storage.inputError = error;
  editor.view.dispatch(editor.state.tr.setMeta("imageInputError", true));
}
