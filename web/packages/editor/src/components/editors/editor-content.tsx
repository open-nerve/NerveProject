/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ImageInputError } from "@/extensions/custom-image/input";
import { useContext } from "react";
import { EditorLabelsContext } from "@/components/editor-labels";
import { EditorContent, useEditorState } from "@tiptap/react";
import type { Editor } from "@tiptap/react";

type Props = {
  editor: Editor | null;
  id: string;
  tabIndex?: number;
};

export function EditorContentWrapper(props: Props) {
  const { editor, tabIndex, id } = props;
  const labels = useContext(EditorLabelsContext);
  const imageError = useEditorState({
    editor,
    selector: ({ editor: current }): ImageInputError | undefined => current?.storage.imageComponent?.inputError,
  });

  return (
    <div
      tabIndex={tabIndex}
      onFocus={(event) => {
        if (event.target === event.currentTarget) editor?.chain().focus(undefined, { scrollIntoView: false }).run();
      }}
    >
      <EditorContent editor={editor} id={id} />
      {imageError && <p role="alert">{labels.image[imageError]}</p>}
    </div>
  );
}
