/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Extension } from "@tiptap/core";
import codemark from "prosemirror-codemark";
// helpers
import { CORE_EXTENSIONS } from "@/constants/extension";
// plugins
import { DropHandlerPlugin } from "@/plugins/drop";
import { MarkdownClipboardPlugin } from "@/plugins/markdown-clipboard";
import type { IEditorProps } from "@/types";

type TActiveDropbarExtensions =
  | CORE_EXTENSIONS.MENTION
  | CORE_EXTENSIONS.EMOJI
  | CORE_EXTENSIONS.SLASH_COMMANDS
  | CORE_EXTENSIONS.TABLE
  | "image-viewer"
  | "bubble-menu"
  | CORE_EXTENSIONS.SIDE_MENU;

declare module "@tiptap/core" {
  interface Commands {
    [CORE_EXTENSIONS.UTILITY]: {
      addActiveDropbarExtension: (extension: TActiveDropbarExtensions) => () => void;
      removeActiveDropbarExtension: (extension: TActiveDropbarExtensions) => () => void;
    };
  }
  interface Storage {
    [CORE_EXTENSIONS.UTILITY]: UtilityExtensionStorage;
  }
}

type UtilityExtensionStorage = {
  activeDropbarExtensions: TActiveDropbarExtensions[];
  isTouchDevice: boolean;
};

type Props = Pick<IEditorProps, "disabledExtensions" | "getEditorMetaData" | "images"> & {
  isTouchDevice: boolean;
};

export const UtilityExtension = (props: Props) => {
  const { disabledExtensions, images, getEditorMetaData, isTouchDevice } = props;

  return Extension.create<Record<string, unknown>, UtilityExtensionStorage>({
    name: CORE_EXTENSIONS.UTILITY,
    priority: 1000,

    addProseMirrorPlugins() {
      return [
        ...codemark({ markType: this.editor.schema.marks.code }),
        MarkdownClipboardPlugin({
          editor: this.editor,
          getEditorMetaData,
        }),
        DropHandlerPlugin({
          disabledExtensions,
          images,
          editor: this.editor,
        }),
      ];
    },

    addStorage() {
      return {
        activeDropbarExtensions: [],
        isTouchDevice,
      };
    },

    addCommands() {
      return {
        addActiveDropbarExtension: (extension) => () => {
          const index = this.storage.activeDropbarExtensions.indexOf(extension);
          if (index === -1) {
            this.storage.activeDropbarExtensions.push(extension);
          }
        },
        removeActiveDropbarExtension: (extension) => () => {
          const index = this.storage.activeDropbarExtensions.indexOf(extension);
          if (index !== -1) {
            this.storage.activeDropbarExtensions.splice(index, 1);
          }
        },
      };
    },
  });
};
