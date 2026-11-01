/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { findChildren } from "@tiptap/core";
import type { EditorView } from "@tiptap/pm/view";
// types
import type { UniqueIDOptions } from "./extension";

/**
 * Utility function to create IDs for nodes that don't have them
 */
export const createIdsForView = (view: EditorView, options: UniqueIDOptions) => {
  const { state } = view;
  const { tr, doc } = state;
  const { types, attributeName, generateUniqueID } = options;

  // A leaf block (an image or rule) is smaller than an empty paragraph but is still content.
  // Give it its identity before React mounts its NodeView, so the first edit does not replace that view.
  if (
    doc.childCount === 1 &&
    doc.firstChild?.type === state.schema.nodes.paragraph &&
    doc.firstChild.content.size === 0
  ) {
    return;
  }

  const nodesWithoutId = findChildren(
    doc,
    (node) => types.includes(node.type.name) && node.attrs[attributeName] === null
  );

  nodesWithoutId.forEach(({ node, pos }) => {
    tr.setNodeMarkup(pos, undefined, {
      ...node.attrs,
      [attributeName]: generateUniqueID({ node, pos }),
    });
  });

  tr.setMeta("addToHistory", false);
  tr.setMeta("uniqueIdOnlyChange", true);

  view.dispatch(tr);
};
