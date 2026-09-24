/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ComponentType, SVGProps } from "react";
import {
  AlignCenterOutline,
  AlignLeftOutline,
  AlignRightOutline,
  BoldOutline,
  CodeOutline,
  ImageOutline,
  ItalicOutline,
  ListOutline,
  NumberedListOutline,
  QuoteOutline,
  StrikethroughOutline,
  ToDoOutline,
  UnderlineOutline,
} from "@makeplane/propel/icons";
import type { TCommandExtraProps, TEditorCommands } from "@/types";

// Utility type to enforce the necessary extra props or make extraProps optional
type ExtraPropsForCommand<T extends TEditorCommands> = T extends keyof TCommandExtraProps
  ? TCommandExtraProps[T]
  : object; // Default to empty object for commands without extra props

export type ToolbarMenuItem<T extends TEditorCommands = TEditorCommands> = {
  itemKey: T;
  renderKey: string;
  name: string;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  shortcut?: string[];
  extraProps?: ExtraPropsForCommand<T>;
};

const TEXT_ALIGNMENT_ITEMS: ToolbarMenuItem<"text-align">[] = [
  {
    itemKey: "text-align",
    renderKey: "text-align-left",
    name: "Left align",
    icon: AlignLeftOutline,
    shortcut: ["Cmd", "Shift", "L"],
    extraProps: {
      alignment: "left",
    },
  },
  {
    itemKey: "text-align",
    renderKey: "text-align-center",
    name: "Center align",
    icon: AlignCenterOutline,
    shortcut: ["Cmd", "Shift", "E"],
    extraProps: {
      alignment: "center",
    },
  },
  {
    itemKey: "text-align",
    renderKey: "text-align-right",
    name: "Right align",
    icon: AlignRightOutline,
    shortcut: ["Cmd", "Shift", "R"],
    extraProps: {
      alignment: "right",
    },
  },
];

const BASIC_MARK_ITEMS: ToolbarMenuItem<"bold" | "italic" | "underline" | "strikethrough">[] = [
  {
    itemKey: "bold",
    renderKey: "bold",
    name: "Bold",
    icon: BoldOutline,
    shortcut: ["Cmd", "B"],
  },
  {
    itemKey: "italic",
    renderKey: "italic",
    name: "Italic",
    icon: ItalicOutline,
    shortcut: ["Cmd", "I"],
  },
  {
    itemKey: "underline",
    renderKey: "underline",
    name: "Underline",
    icon: UnderlineOutline,
    shortcut: ["Cmd", "U"],
  },
  {
    itemKey: "strikethrough",
    renderKey: "strikethrough",
    name: "Strikethrough",
    icon: StrikethroughOutline,
    shortcut: ["Cmd", "Shift", "S"],
  },
];

const LIST_ITEMS: ToolbarMenuItem<"bulleted-list" | "numbered-list" | "to-do-list">[] = [
  {
    itemKey: "bulleted-list",
    renderKey: "bulleted-list",
    name: "Bulleted list",
    icon: ListOutline,
    shortcut: ["Cmd", "Shift", "7"],
  },
  {
    itemKey: "numbered-list",
    renderKey: "numbered-list",
    name: "Numbered list",
    icon: NumberedListOutline,
    shortcut: ["Cmd", "Shift", "8"],
  },
  {
    itemKey: "to-do-list",
    renderKey: "to-do-list",
    name: "To-do list",
    icon: ToDoOutline,
    shortcut: ["Cmd", "Shift", "9"],
  },
];

const USER_ACTION_ITEMS: ToolbarMenuItem<"quote" | "code">[] = [
  { itemKey: "quote", renderKey: "quote", name: "Quote", icon: QuoteOutline },
  { itemKey: "code", renderKey: "code", name: "Code", icon: CodeOutline },
];

export const IMAGE_ITEM: ToolbarMenuItem<"image"> = {
  itemKey: "image",
  renderKey: "image",
  name: "Image",
  icon: ImageOutline,
};

export const TOOLBAR_ITEMS: {
  [key: string]: ToolbarMenuItem[];
} = {
  basic: BASIC_MARK_ITEMS,
  alignment: TEXT_ALIGNMENT_ITEMS,
  list: LIST_ITEMS,
  userAction: USER_ACTION_ITEMS,
  complex: [IMAGE_ITEM],
};

export const COLORS_LIST: {
  key: string;
  label: string;
  textColor: string;
  backgroundColor: string;
}[] = [
  {
    key: "gray",
    label: "Gray",
    textColor: "var(--editor-colors-gray-text)",
    backgroundColor: "var(--editor-colors-gray-background)",
  },
  {
    key: "peach",
    label: "Peach",
    textColor: "var(--editor-colors-peach-text)",
    backgroundColor: "var(--editor-colors-peach-background)",
  },
  {
    key: "pink",
    label: "Pink",
    textColor: "var(--editor-colors-pink-text)",
    backgroundColor: "var(--editor-colors-pink-background)",
  },
  {
    key: "orange",
    label: "Orange",
    textColor: "var(--editor-colors-orange-text)",
    backgroundColor: "var(--editor-colors-orange-background)",
  },
  {
    key: "green",
    label: "Green",
    textColor: "var(--editor-colors-green-text)",
    backgroundColor: "var(--editor-colors-green-background)",
  },
  {
    key: "light-blue",
    label: "Light blue",
    textColor: "var(--editor-colors-light-blue-text)",
    backgroundColor: "var(--editor-colors-light-blue-background)",
  },
  {
    key: "dark-blue",
    label: "Dark blue",
    textColor: "var(--editor-colors-dark-blue-text)",
    backgroundColor: "var(--editor-colors-dark-blue-background)",
  },
  {
    key: "purple",
    label: "Purple",
    textColor: "var(--editor-colors-purple-text)",
    backgroundColor: "var(--editor-colors-purple-background)",
  },
  // {
  //   key: "pink-blue-gradient",
  //   label: "Pink blue gradient",
  //   textColor: "var(--editor-colors-pink-blue-gradient-text)",
  //   backgroundColor: "var(--editor-colors-pink-blue-gradient-background)",
  // },
];
