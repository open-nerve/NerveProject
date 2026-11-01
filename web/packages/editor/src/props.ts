/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { EditorProps } from "@tiptap/pm/view";
// nerve utils
import { cn } from "@nerve/utils";
// helpers
import { processAssetDuplication } from "@/helpers/paste-asset";

type TArgs = {
  editorClassName: string;
  attributes?: EditorProps["attributes"];
};

export const CoreEditorProps = (props: TArgs): EditorProps => {
  const { editorClassName, attributes } = props;

  return {
    // TipTap initializes DOM attributes from an object; preserve a caller's existing function contract unchanged.
    attributes:
      typeof attributes === "function"
        ? attributes
        : {
            role: "textbox",
            ...attributes,
            class: cn(
              "prose-brand prose-headings:font-display font-default max-w-full prose focus:outline-none",
              editorClassName,
              attributes?.class
            ),
          },
    handleDOMEvents: {
      keydown: (_view, event) => {
        // prevent default event listeners from firing when slash command is active
        if (["ArrowUp", "ArrowDown", "Enter"].includes(event.key)) {
          const slashCommand = document.querySelector("#slash-command");
          if (slashCommand) {
            return true;
          }
        }
      },
    },
    handlePaste: (view, event) => {
      if (!event.clipboardData) return false;

      const htmlContent = event.clipboardData.getData("text/nerve-editor-html");
      if (!htmlContent) return false;

      const { processedHtml } = processAssetDuplication(htmlContent);
      view.pasteHTML(processedHtml);
      return true;
    },
  };
};
