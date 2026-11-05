/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor, Extensions } from "@tiptap/core";
// components
import { EditorContainer } from "@/components/editors";
// constants
import { DEFAULT_DISPLAY_CONFIG } from "@/constants/config";
// hooks
import { getEditorClassNames } from "@/helpers/common";
import { useEditor } from "@/hooks/use-editor";
// types
import type { IEditorProps } from "@/types";
import { EditorContentWrapper } from "./editor-content";
import { defaultEditorLabels, EditorLabelsContext } from "@/components/editor-labels";

type Props = IEditorProps & {
  children?: (editor: Editor) => React.ReactNode;
  editable: boolean;
  extensions: Extensions;
};

export function EditorWrapper(props: Props) {
  const {
    labels = defaultEditorLabels,
    children,
    containerClassName,
    disabledExtensions,
    displayConfig = DEFAULT_DISPLAY_CONFIG,
    editable,
    editorClassName = "",
    editorProps,
    extensions,
    getEditorMetaData,
    id,
    initialValue,
    isTouchDevice,
    images,
    forwardedRef,
    mentionHandler,
    onChange,
    onEditorFocus,
    onTransaction,
    handleEditorReady,
    autofocus,
    placeholder,
    showPlaceholderOnEmpty,
    tabIndex,
    value,
  } = props;

  const editor = useEditor({
    editable,
    disabledExtensions,
    editorClassName,
    editorProps,
    enableHistory: true,
    extensions,
    images,
    forwardedRef,
    getEditorMetaData,
    id,
    isTouchDevice,
    initialValue,
    mentionHandler,
    onChange,
    onEditorFocus,
    onTransaction,
    handleEditorReady,
    autofocus,
    placeholder,
    showPlaceholderOnEmpty,
    tabIndex,
    value,
  });

  const editorContainerClassName = getEditorClassNames({
    noBorder: true,
    borderOnFocus: false,
    containerClassName,
  });

  if (!editor) return null;

  return (
    <EditorLabelsContext value={labels}>
      <EditorContainer
        displayConfig={displayConfig}
        editor={editor}
        editorContainerClassName={editorContainerClassName}
        id={id}
        isTouchDevice={!!isTouchDevice}
      >
        {children?.(editor)}
        <div className="flex flex-col">
          <EditorContentWrapper editor={editor} id={id} tabIndex={tabIndex} />
        </div>
      </EditorContainer>
    </EditorLabelsContext>
  );
}
