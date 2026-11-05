/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { ReactNodeViewRenderer } from "@tiptap/react";
import { v4 as uuid } from "uuid";
import { insertEmptyParagraphAtNodeBoundaries } from "@/helpers/insert-empty-paragraph-at-node-boundary";
import type { EditorImages } from "@/types";
import type { CustomImageNodeViewProps } from "./components/node-view";
import { CustomImageNodeView } from "./components/node-view";
import { CustomImageExtensionConfig } from "./extension-config";
import type { CustomImageExtensionOptions, CustomImageExtensionStorage } from "./types";
import { imageInputError, validateImages } from "./input";
import { imagePaste } from "./paste";
import { ImageTransfers } from "./transfers";
export function CustomImageExtension({ images, isEditable }: { images?: EditorImages; isEditable: boolean }) {
  return CustomImageExtensionConfig.extend<CustomImageExtensionOptions, CustomImageExtensionStorage>({
    selectable: isEditable,
    draggable: isEditable,
    addOptions() {
      return { ...this.parent?.(), images };
    },
    addStorage() {
      const settled = { fileMap: new Map(), completed: new Map<string, string>() };
      return { ...settled, transfers: new ImageTransfers(images, settled), images, inputError: undefined };
    },
    addCommands() {
      return {
        insertImageComponent:
          (props) =>
          ({ commands }) => {
            if (!this.editor.isEditable || !images?.write || images.signal.aborted) return false;
            const error = props.file ? validateImages([props.file], images.write.maxBytes) : undefined;
            imageInputError(this.editor, error);
            if (error) return false;
            const id = uuid();
            this.storage.fileMap.set(id, {
              file: props.file,
              autoStart: !!props.file,
              openPicker: props.event === "insert",
            });
            const content = { type: this.name, attrs: { id, src: null, status: "pending" } };
            return props.pos !== undefined
              ? commands.insertContentAt(props.pos, content)
              : commands.insertContent(content);
          },
      };
    },
    addKeyboardShortcuts() {
      return {
        ArrowDown: insertEmptyParagraphAtNodeBoundaries("down", this.name),
        ArrowUp: insertEmptyParagraphAtNodeBoundaries("up", this.name),
      };
    },
    addProseMirrorPlugins() {
      return [imagePaste(this.editor, this.storage)];
    },
    addNodeView() {
      return ReactNodeViewRenderer((props) => (
        <CustomImageNodeView {...props} node={props.node as CustomImageNodeViewProps["node"]} />
      ));
    },
  });
}
