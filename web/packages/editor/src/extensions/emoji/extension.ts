/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// local imports
import { emojis, shortcodeToEmoji } from "./data";
import type { MarkdownSerializerState } from "@tiptap/pm/markdown";
import type { Node as ProseMirrorNode } from "@tiptap/pm/model";
import { Emoji } from "./emoji";
import { emojiSuggestion } from "./suggestion";

export const EmojiExtension = Emoji.extend({
  addStorage() {
    const extensionOptions = this.options;

    return {
      ...this.parent?.(),
      markdown: {
        serialize(state: MarkdownSerializerState, node: ProseMirrorNode) {
          const emojiItem = shortcodeToEmoji(node.attrs.name, extensionOptions.emojis);
          if (emojiItem?.emoji) {
            state.write(emojiItem?.emoji);
          } else {
            state.write(`:${node.attrs.name}:`);
          }
        },
      },
    };
  },
}).configure({
  emojis,
  suggestion: emojiSuggestion,
  enableEmoticons: true,
});
