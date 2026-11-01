/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEditorState, useEditor as useTiptapEditor } from "@tiptap/react";
import { useImperativeHandle, useEffect, useLayoutEffect, useMemo } from "react";
// extensions
import { CoreEditorExtensions } from "@/extensions";
// helpers
import { getEditorRefHelpers } from "@/helpers/editor-ref";
// props
import { CoreEditorProps } from "@/props";
import { CORE_EXTENSIONS } from "@/constants/extension";
// types
import type { TEditorHookProps } from "@/types";
const defaultEditorProps: NonNullable<TEditorHookProps["editorProps"]> = {};

export const useEditor = (props: TEditorHookProps) => {
  const {
    autofocus = false,
    disabledExtensions,
    editable = true,
    editorClassName = "",
    editorProps = defaultEditorProps,
    enableHistory,
    extensions = [],
    fileHandler,
    forwardedRef,
    getEditorMetaData,
    handleEditorReady,
    id = "",
    initialValue,
    isTouchDevice,
    mentionHandler,
    onAssetChange,
    onChange,
    onEditorFocus,
    onTransaction,
    placeholder,
    showPlaceholderOnEmpty,
    tabIndex,
    value,
  } = props;

  const mergedEditorProps = useMemo(() => {
    const { attributes, ...customEditorProps } = editorProps;
    return { ...CoreEditorProps({ editorClassName, attributes }), ...customEditorProps };
  }, [editorClassName, editorProps]);
  const editor = useTiptapEditor(
    {
      editable,
      immediatelyRender: false,
      shouldRerenderOnTransaction: false,
      autofocus,
      // Markdown already parses formatting (including escaped syntax). Only non-Markdown transforms
      // run afterwards, so mark paste rules cannot reinterpret the resulting literal text.
      enablePasteRules: [CORE_EXTENSIONS.CUSTOM_LINK, CORE_EXTENSIONS.EMOJI],
      parseOptions: { preserveWhitespace: true },
      editorProps: mergedEditorProps,
      extensions: [
        ...CoreEditorExtensions({
          disabledExtensions,
          editable,
          enableHistory,
          fileHandler,
          getEditorMetaData,
          isTouchDevice,
          mentionHandler,
          placeholder,
          showPlaceholderOnEmpty,
          tabIndex,
        }),
        ...extensions,
      ],
      content: initialValue,
      onCreate: () => handleEditorReady?.(true),
      onTransaction: () => {
        onTransaction?.();
      },
      onUpdate: ({ editor: updatedEditor, transaction }) => {
        // Check if this update is only due to migration update
        const isMigrationUpdate = transaction?.getMeta("uniqueIdOnlyChange") === true;
        onChange?.(updatedEditor.getJSON(), updatedEditor.getHTML(), { isMigrationUpdate });
      },
      onDestroy: () => handleEditorReady?.(false),
      onFocus: onEditorFocus,
    },
    [editable]
  );

  // With explicit recreation dependencies TipTap does not synchronize options on an ordinary render.
  // Refusal attributes must reach the existing field before its parent's committed focus effect.
  useLayoutEffect(() => {
    if (editor && !editor.isDestroyed) editor.setOptions({ editorProps: mergedEditorProps });
  }, [editor, mergedEditorProps]);

  // Effect for syncing SWR data
  useEffect(() => {
    // value is null when intentionally passed where syncing is not yet
    // supported and value is undefined when the data from swr is not populated
    if (value == null) return;
    if (editor) {
      const { uploadInProgress: isUploadInProgress } = editor.storage.utility;
      if (!editor.isDestroyed && !isUploadInProgress) {
        try {
          editor.commands.setContent(value, false, {
            preserveWhitespace: true,
          });
          if (editor.state.selection) {
            const docLength = editor.state.doc.content.size;
            const relativePosition = Math.min(editor.state.selection.from, docLength - 1);
            editor.commands.setTextSelection(relativePosition);
          }
        } catch (error) {
          console.error("Error syncing editor content with external value:", error);
        }
      }
    }
  }, [editor, value, id]);

  // update assets upload status
  useEffect(() => {
    if (!editor) return;
    const assetsUploadStatus = fileHandler.assetsUploadStatus;
    editor.commands.updateAssetsUploadStatus?.(assetsUploadStatus);
  }, [editor, fileHandler.assetsUploadStatus]);

  // subscribe to assets list changes
  const assetsList = useEditorState({
    editor,
    selector: ({ editor: currentEditor }) => ({
      assets: currentEditor?.storage.utility?.assetsList ?? [],
    }),
  });
  // trigger callback when assets list changes
  useEffect(() => {
    const assets = assetsList?.assets;
    if (!assets || !onAssetChange) return;
    onAssetChange(assets);
  }, [assetsList?.assets, onAssetChange]);

  useImperativeHandle(
    forwardedRef,
    () =>
      getEditorRefHelpers({
        editor,
        getEditorMetaData,
      }),
    [editor, getEditorMetaData]
  );

  if (!editor) {
    return null;
  }

  return editor;
};
