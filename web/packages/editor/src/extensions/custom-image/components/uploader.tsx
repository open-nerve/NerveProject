/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useContext, useRef } from "react";
import { EditorLabelsContext } from "@/components/editor-labels";
import { ACCEPTED_IMAGE_MIME_TYPES } from "@/constants/config";
import { imageInputError, validateImages } from "../input";
import { insertFilesSafely } from "@/plugins/drop";
import type { CustomImageNodeViewProps } from "./node-view";
import type { useImageTransfer } from "./use-image-transfer";
type Props = CustomImageNodeViewProps & { transfer: ReturnType<typeof useImageTransfer>; available: boolean };
export function CustomImageUploader({ editor, extension, getPos, transfer, available }: Props) {
  const { image: labels } = useContext(EditorLabelsContext);
  const input = useRef<HTMLInputElement>(null);
  const canWrite = available && editor.isEditable && !!extension.options.images?.write;
  const select = (files: File[]) => {
    if (!canWrite) return;
    const error = validateImages(files, extension.options.images!.write!.maxBytes);
    imageInputError(editor, error);
    if (error) return;
    transfer.upload(files[0]!);
    const pos = getPos();
    if (files.length > 1 && pos !== undefined)
      insertFilesSafely({ editor, event: "drop", files: files.slice(1), initialPos: pos + 1 });
  };
  return (
    <div
      className="image-upload-component rounded-lg border border-dashed border-subtle bg-layer-3 p-3"
      contentEditable={false}
    >
      <p role={transfer.error ? "alert" : "status"} aria-live="polite">
        {transfer.phase
          ? labels[transfer.phase]
          : (transfer.error ??
            (canWrite ? (transfer.cancelled ? labels.cancelled : labels.select) : labels.unavailable))}
      </p>
      {transfer.failed && transfer.copied && <p>{labels.copyUnknown}</p>}
      {canWrite && (
        <div className="flex gap-2">
          <button type="button" onClick={() => input.current?.click()}>
            {labels.select}
          </button>
          <input
            ref={input}
            className="sr-only"
            aria-label={labels.select}
            type="file"
            accept={ACCEPTED_IMAGE_MIME_TYPES.join(",")}
            multiple
            onChange={(event) => {
              select(Array.from(event.target.files ?? []));
              event.target.value = "";
            }}
          />
          {transfer.phase && (
            <button type="button" onClick={transfer.cancel}>
              {labels.cancel}
            </button>
          )}
          {transfer.failed && (
            <button type="button" onClick={transfer.retry}>
              {labels.retry}
            </button>
          )}
        </div>
      )}
    </div>
  );
}
