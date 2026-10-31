/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Node, mergeAttributes } from "@tiptap/core";
// constants
import { CORE_EXTENSIONS } from "@/constants/extension";
// types
import { ECalloutAttributeNames } from "./types";
import type { CustomCalloutExtensionType, TCalloutBlockAttributes } from "./types";
// utils
import { DEFAULT_CALLOUT_BLOCK_ATTRIBUTES } from "./utils";

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    [CORE_EXTENSIONS.CALLOUT]: {
      insertCallout: () => ReturnType;
    };
  }
}

export const CustomCalloutExtensionConfig: CustomCalloutExtensionType = Node.create({
  name: CORE_EXTENSIONS.CALLOUT,
  group: "block",
  content: "block+",

  addAttributes() {
    return Object.values(ECalloutAttributeNames).reduce(
      (acc, value) => {
        acc[value] = {
          default: DEFAULT_CALLOUT_BLOCK_ATTRIBUTES[value],
          // every callout attribute is a string: keep what the HTML says (TipTap's default parsing would turn a
          // numeric string such as an emoji's code point into a number)
          parseHTML: (element) => element.getAttribute(value),
        };
        return acc;
      },
      {} as Record<
        ECalloutAttributeNames,
        {
          default: TCalloutBlockAttributes[ECalloutAttributeNames];
          parseHTML: (element: HTMLElement) => string | null;
        }
      >
    );
  },

  parseHTML() {
    return [
      {
        tag: `div[${ECalloutAttributeNames.BLOCK_TYPE}="${DEFAULT_CALLOUT_BLOCK_ATTRIBUTES[ECalloutAttributeNames.BLOCK_TYPE]}"]`,
      },
    ];
  },

  // Render HTML for the callout node
  renderHTML({ HTMLAttributes }) {
    return ["div", mergeAttributes(HTMLAttributes), 0];
  },
});
