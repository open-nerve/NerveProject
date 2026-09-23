/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Content } from "@tiptap/core";
// local imports
import type { IEditorProps } from "./editor";

type TCoreHookProps = Pick<
  IEditorProps,
  | "disabledExtensions"
  | "editorClassName"
  | "editorProps"
  | "extendedEditorProps"
  | "extensions"
  | "flaggedExtensions"
  | "getEditorMetaData"
  | "handleEditorReady"
  | "isTouchDevice"
  | "onEditorFocus"
>;

export type TEditorHookProps = TCoreHookProps &
  Pick<
    IEditorProps,
    | "autofocus"
    | "fileHandler"
    | "forwardedRef"
    | "id"
    | "mentionHandler"
    | "onAssetChange"
    | "onChange"
    | "onTransaction"
    | "placeholder"
    | "showPlaceholderOnEmpty"
    | "tabIndex"
    | "value"
  > & {
    editable: boolean;
    enableHistory: boolean;
    initialValue?: Content;
  };
