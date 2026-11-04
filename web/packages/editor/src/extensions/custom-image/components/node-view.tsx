/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { NodeViewWrapper, type NodeViewProps } from "@tiptap/react";
import { useContext } from "react";
import { EditorLabelsContext } from "@/components/editor-labels";
import type { CustomImageExtensionType, TCustomImageAttributes } from "../types";
import { isImageID } from "../input";
import { CustomImageBlock } from "./block";
import { CustomImageUploader } from "./uploader";
import { useImageRead } from "./use-image-read";
import { useImageTransfer } from "./use-image-transfer";
export type CustomImageNodeViewProps = Omit<NodeViewProps, "extension" | "updateAttributes"> & {
  extension: CustomImageExtensionType;
  node: NodeViewProps["node"] & { attrs: TCustomImageAttributes };
  updateAttributes: (attrs: Partial<TCustomImageAttributes>) => void;
};
export function CustomImageNodeView(props: CustomImageNodeViewProps) {
  const { image: labels } = useContext(EditorLabelsContext);
  const { extension, node } = props;
  const images = extension.options.images;
  const { read, snapshot, available } = useImageRead(images, node.attrs.src);
  const transfer = useImageTransfer(props);
  const current = () => !!images && !images.signal.aborted && read.getSnapshot() === snapshot;
  return (
    <NodeViewWrapper>
      <div className="mx-0 my-2 p-0" data-drag-handle>
        {available && snapshot.phase === "ready" ? (
          <CustomImageBlock
            key={snapshot.url}
            {...props}
            src={snapshot.url}
            loaded={() => {
              if (current()) read.loaded();
            }}
            failed={() => {
              if (current()) read.failed();
            }}
            download={(signal) => images!.download(node.attrs.src!, signal)}
          />
        ) : isImageID(node.attrs.src) ? (
          <div contentEditable={false} role={snapshot.phase === "error" ? "alert" : "status"}>
            {snapshot.phase === "error"
              ? snapshot.message
              : snapshot.phase === "loading"
                ? labels.loading
                : labels.unavailable}
            {available && snapshot.phase === "error" && (
              <button type="button" onClick={read.retry}>
                {labels.retry}
              </button>
            )}
          </div>
        ) : (
          <CustomImageUploader {...props} transfer={transfer} available={available} />
        )}
      </div>
    </NodeViewWrapper>
  );
}
