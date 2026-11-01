/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EditorContent } from "@tiptap/react";
import type { Editor } from "@tiptap/react";

type Props = {
  editor: Editor | null;
  id: string;
  tabIndex?: number;
};

export function EditorContentWrapper(props: Props) {
  const { editor, tabIndex, id } = props;

  return (
    <div
      tabIndex={tabIndex}
      onFocus={(event) => {
        if (event.target === event.currentTarget) editor?.chain().focus(undefined, { scrollIntoView: false }).run();
      }}
    >
      <EditorContent editor={editor} id={id} />
    </div>
  );
}
