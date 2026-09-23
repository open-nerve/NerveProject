/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/react";

export function scrollToNodeViaDOMCoordinates(editor: Editor, pos: number, behavior?: ScrollBehavior): void {
  const view = editor.view;

  // Get the coordinates of the position
  const coords = view.coordsAtPos(pos);

  if (coords) {
    // Scroll to the coordinates
    window.scrollTo({
      top: coords.top + window.scrollY - window.innerHeight / 2,
      behavior: behavior,
    });

    // Optionally, you can also focus the editor
    view.focus();
  } else {
    console.warn("Unable to find coordinates for the given position");
  }
}
