/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { NodeViewWrapper, ReactNodeViewRenderer } from "@tiptap/react";
import { useContext } from "react";
import { EditorLabelsContext } from "@/components/editor-labels";
import { ImageExtensionConfig } from "./extension-config";
function LegacyImage() {
  const labels = useContext(EditorLabelsContext);
  return (
    <NodeViewWrapper>
      <span>{labels.image.unavailable}</span>
    </NodeViewWrapper>
  );
}
export function ImageExtension() {
  return ImageExtensionConfig.extend({
    addNodeView() {
      return ReactNodeViewRenderer(LegacyImage);
    },
  });
}
