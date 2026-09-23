/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Extension } from "@tiptap/core";
import type { Node as ProseMirrorNode } from "@tiptap/pm/model";
import type { Transaction } from "@tiptap/pm/state";
import { v4 as uuidv4 } from "uuid";
// constants
import { CORE_EXTENSIONS, BLOCK_NODE_TYPES } from "@/constants/extension";
import { ADDITIONAL_BLOCK_NODE_TYPES } from "@/constants/extensions";
import { createUniqueIDPlugin } from "./plugin";
import { createIdsForView } from "./utils";
// plane imports

const COMBINED_BLOCK_NODE_TYPES = [...BLOCK_NODE_TYPES, ...ADDITIONAL_BLOCK_NODE_TYPES];
export type UniqueIDGenerationContext = {
  node: ProseMirrorNode;
  pos: number;
};
export const UniqueIDAttribute = "id";
export const generateUniqueID = () => uuidv4();

export interface UniqueIDOptions {
  /**
   * The name of the attribute to add the unique ID to.
   * @default "id"
   */
  attributeName: string;
  /**
   * The types of nodes to add unique IDs to.
   * @default []
   */
  types: string[];
  /**
   * The function that generates the unique ID. By default, a UUID v4 is
   * generated. However, you can provide your own function to generate the
   * unique ID based on the node type and the position.
   */
  generateUniqueID: (ctx: UniqueIDGenerationContext) => string;
  /**
   * Ignore some mutations, for example applied from other users through the collaboration plugin.
   *
   * @default null
   */
  filterTransaction: ((transaction: Transaction) => boolean) | null;
  /**
   * Whether to update the document by adding unique IDs to the nodes. Set this
   * property to `false` if the document is in `readonly` mode, is immutable, or
   * you don't want it to be modified.
   *
   * @default true
   */
  updateDocument: boolean;
}

export const UniqueID = Extension.create<UniqueIDOptions>({
  name: CORE_EXTENSIONS.UNIQUE_ID,

  // we'll set a very high priority to make sure this runs first
  // and is compatible with `appendTransaction` hooks of other extensions
  priority: 10000,

  addOptions() {
    return {
      attributeName: "id",
      types: COMBINED_BLOCK_NODE_TYPES,
      generateUniqueID: () => uuidv4(),
      filterTransaction: null,
      updateDocument: true,
    };
  },

  addGlobalAttributes() {
    return [
      {
        types: this.options.types,
        attributes: {
          [this.options.attributeName]: {
            default: null,
            parseHTML: (element) => element.getAttribute(`data-${this.options.attributeName}`),
            renderHTML: (attributes) => {
              if (!attributes[this.options.attributeName]) {
                return {};
              }

              return {
                [`data-${this.options.attributeName}`]: attributes[this.options.attributeName],
              };
            },
          },
        },
      },
    ];
  },

  // check initial content for missing ids
  onCreate() {
    // a read-only editor leaves its content as it is; the options are shared by every editor that
    // installs this extension, so they are only read here, never written
    if (!this.editor.isEditable || !this.options.updateDocument) {
      return;
    }

    createIdsForView(this.editor.view, this.options);
  },

  addProseMirrorPlugins() {
    if (!this.options.updateDocument) {
      return [];
    }

    return [createUniqueIDPlugin(this.options)];
  },
});
