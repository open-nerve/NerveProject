/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Extensions, RawCommands } from "@tiptap/core";
import type { MarkType, NodeType } from "@tiptap/pm/model";
import type { Selection } from "@tiptap/pm/state";
import type { EditorProps, EditorView } from "@tiptap/pm/view";
import type { NodeViewProps as TNodeViewProps } from "@tiptap/react";
// plane imports
import type { TCustomComponentsMetaData } from "@plane/utils";
// extension types
import type { TTextAlign } from "@/extensions";
// plane editor imports
import type { IEditorPropsExtended, TExtendedEditorCommands } from "@/types/editor-extended";
// types
import type {
  TDisplayConfig,
  TEditorAsset,
  TExtensions,
  TFileHandler,
  TMentionHandler,
  TExtendedEditorRefApi,
} from "@/types";

export type TEditorCommands =
  | "text"
  | "h1"
  | "h2"
  | "h3"
  | "h4"
  | "h5"
  | "h6"
  | "bold"
  | "italic"
  | "underline"
  | "strikethrough"
  | "bulleted-list"
  | "numbered-list"
  | "to-do-list"
  | "quote"
  | "code"
  | "table"
  | "image"
  | "divider"
  | "link"
  | "text-color"
  | "background-color"
  | "text-align"
  | "callout"
  | "attachment"
  | "emoji"
  | "external-embed"
  | TExtendedEditorCommands;

export type TCommandExtraProps = {
  image: {
    savedSelection: Selection | null;
  };
  attachment: {
    savedSelection: Selection | null;
  };
  "text-color": {
    color: string | undefined;
  };
  link: {
    url: string;
    text?: string;
  };
  "background-color": {
    color: string | undefined;
  };
  "text-align": {
    alignment: TTextAlign;
  };
};

// Create a utility type that maps a command to its extra props or an empty object if none are defined
export type TCommandWithProps<T extends TEditorCommands> = T extends keyof TCommandExtraProps
  ? TCommandExtraProps[T] // If the command has extra props, include them
  : object; // Otherwise, just return the command type with no extra props

type TCommandWithPropsWithItemKey<T extends TEditorCommands> = T extends keyof TCommandExtraProps
  ? { itemKey: T } & TCommandExtraProps[T]
  : { itemKey: T };

export type CoreEditorRefApi = {
  blur: () => void;
  clearEditor: (emitUpdate?: boolean) => void;
  createSelectionAtCursorPosition: () => void;
  executeMenuItemCommand: <T extends TEditorCommands>(props: TCommandWithPropsWithItemKey<T>) => void;
  focus: (args: Parameters<RawCommands["focus"]>[0]) => void;
  getAttributesWithExtendedMark: (
    mark: string | MarkType,
    attribute: string | NodeType | MarkType
  ) => Record<string, any> | undefined;
  getCoordsFromPos: (pos?: number) => ReturnType<EditorView["coordsAtPos"]> | undefined;
  getCurrentCursorPosition: () => number | undefined;
  getMarkDown: () => string;
  copyMarkdownToClipboard: () => void;
  getSelectedText: () => string | null;
  insertText: (contentHTML: string, insertOnNextLine?: boolean) => void;
  isAnyDropbarOpen: () => boolean;
  isEditorReadyToDiscard: () => boolean;
  isMenuItemActive: <T extends TEditorCommands>(props: TCommandWithPropsWithItemKey<T>) => boolean;
  onStateChange: (callback: () => void) => () => void;
  redo: () => void;
  scrollToNodeViaDOMCoordinates: ({ pos, behavior }: { pos?: number; behavior?: ScrollBehavior }) => void;
  setEditorValue: (content: string, emitUpdate?: boolean) => void;
  setFocusAtPosition: (position: number) => void;
  undo: () => void;
};

export type EditorRefApi = CoreEditorRefApi & TExtendedEditorRefApi;

// editor props
export type IEditorProps = {
  autofocus?: boolean;
  bubbleMenuEnabled?: boolean;
  containerClassName?: string;
  displayConfig?: TDisplayConfig;
  disabledExtensions: TExtensions[];
  editable: boolean;
  editorClassName?: string;
  editorProps?: EditorProps;
  extensions?: Extensions;
  flaggedExtensions: TExtensions[];
  fileHandler: TFileHandler;
  forwardedRef?: React.MutableRefObject<EditorRefApi | null>;
  getEditorMetaData: (htmlContent: string) => TCustomComponentsMetaData;
  handleEditorReady?: (value: boolean) => void;
  id: string;
  initialValue: string;
  isTouchDevice?: boolean;
  mentionHandler: TMentionHandler;
  onAssetChange?: (assets: TEditorAsset[]) => void;
  onEditorFocus?: () => void;
  onChange?: (json: object, html: string, { isMigrationUpdate }?: { isMigrationUpdate?: boolean }) => void;
  onEnterKeyPress?: (e?: any) => void;
  onTransaction?: () => void;
  placeholder?: string | ((isFocused: boolean, value: string) => string);
  showPlaceholderOnEmpty?: boolean;
  tabIndex?: number;
  value?: string | null;
  extendedEditorProps: IEditorPropsExtended;
  workItemIdentifier?: string | null;
};

export type ILiteTextEditorProps = IEditorProps;

export type IRichTextEditorProps = IEditorProps & {
  dragDropEnabled?: boolean;
};

export type EditorEvents = {
  beforeCreate: never;
  create: never;
  update: never;
  selectionUpdate: never;
  transaction: never;
  focus: never;
  blur: never;
  destroy: never;
  ready: { height: number };
};

export type NodeViewProps = TNodeViewProps;
