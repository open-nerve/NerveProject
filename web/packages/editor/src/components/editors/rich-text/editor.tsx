/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { forwardRef, useCallback } from "react";
// components
import { EditorWrapper } from "@/components/editors";
import { BlockMenu, EditorBubbleMenu } from "@/components/menus";
// extensions
import { SideMenuExtension, SlashCommands } from "@/extensions";
// types
import type { EditorRefApi, IRichTextEditorProps } from "@/types";

function RichTextEditor(props: IRichTextEditorProps) {
  const {
    bubbleMenuEnabled = true,
    disabledExtensions,
    dragDropEnabled,
    extensions: externalExtensions = [],
    workItemIdentifier,
  } = props;

  const getExtensions = useCallback(() => {
    const extensions = [
      ...externalExtensions,
      SideMenuExtension({
        dragDropEnabled: !!dragDropEnabled,
      }),
      ...(disabledExtensions.includes("slash-commands") ? [] : [SlashCommands({ disabledExtensions })]),
    ];

    return extensions;
  }, [dragDropEnabled, disabledExtensions, externalExtensions]);

  return (
    <EditorWrapper {...props} extensions={getExtensions()}>
      {(editor) => (
        <>
          {editor && bubbleMenuEnabled && <EditorBubbleMenu disabledExtensions={disabledExtensions} editor={editor} />}
          <BlockMenu editor={editor} disabledExtensions={disabledExtensions} workItemIdentifier={workItemIdentifier} />
        </>
      )}
    </EditorWrapper>
  );
}

const RichTextEditorWithRef = forwardRef(function RichTextEditorWithRef(
  props: IRichTextEditorProps,
  ref: React.ForwardedRef<EditorRefApi>
) {
  return <RichTextEditor {...props} forwardedRef={ref as React.MutableRefObject<EditorRefApi | null>} />;
});

RichTextEditorWithRef.displayName = "RichTextEditorWithRef";

export { RichTextEditorWithRef };
