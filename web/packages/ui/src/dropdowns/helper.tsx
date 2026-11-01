/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Modifier, Placement } from "@popperjs/core";
import type { ICustomSearchSelectOption } from "@nerve/types";

interface IDropdownProps {
  customButtonClassName?: string;
  customButtonTabIndex?: number;
  buttonClassName?: string;
  className?: string;
  customButton?: React.ReactNode;
  disabled?: boolean;
  input?: boolean;
  label?: string | React.ReactNode;
  maxHeight?: "sm" | "rg" | "md" | "lg" | "xl" | "2xl";
  noChevron?: boolean;
  chevronClassName?: string;
  onOpen?: () => void;
  optionsClassName?: string;
  placement?: Placement;
  tabIndex?: number;
  defaultOpen?: boolean;
}

export interface ICustomMenuDropdownProps extends IDropdownProps {
  children: React.ReactNode;
  ellipsis?: boolean;
  noBorder?: boolean;
  verticalEllipsis?: boolean;
  /** Called each time the menu's button opens the menu, by the pointer or the keyboard. */
  menuButtonOnClick?: () => void;
  menuItemsClassName?: string;
  /** Called each time the menu closes, whatever closed it. */
  onMenuClose?: () => void;
  /** Whether a click on anything in the menu closes it, as a click on an item does. */
  closeOnSelect?: boolean;
  portalElement?: Element | null;
  ariaLabel?: string;
}

/** What a form gives a select that is one of its fields (fieldButtonProps). */
export interface ISelectFieldProps {
  /** The id of the message that describes the select, while one shows: its button's aria-describedby. */
  describedBy?: string;
  /** Whether the select's value is refused: its button is aria-invalid. */
  invalid?: boolean;
}

export interface ICustomSelectProps<T> extends IDropdownProps, ISelectFieldProps {
  children: React.ReactNode;
  value: T | null;
  onChange: (value: T) => void;
}

interface CustomSearchSelectProps<T extends string | number> extends ISelectFieldProps {
  footerOption?: React.ReactNode;
  onClose?: () => void;
  searchPlaceholder?: string;
  noResultsMessage?: string;
  loadingMessage?: string;
  options?: ICustomSearchSelectOption<T>[];
  /** Whether the search takes the focus as the list opens: it does unless the caller says not. */
  focusSearchOnOpen?: boolean;
  /** Popper's modifiers for the list, beside its placement. */
  popperModifiers?: Partial<Modifier<string, object>>[];
}

interface SingleValueProps<T> {
  multiple?: false;
  value: T | null;
  onChange: (value: T) => void;
}

interface MultipleValuesProps<T> {
  multiple: true;
  value: T[] | null;
  onChange: (values: T[]) => void;
}

export type ICustomSearchSelectProps<T extends string | number> = IDropdownProps &
  CustomSearchSelectProps<T> &
  (SingleValueProps<T> | MultipleValuesProps<T>);

export interface ICustomMenuItemProps {
  children: React.ReactNode;
  disabled?: boolean;
  onClick?: (event: React.MouseEvent<HTMLButtonElement>) => void;
  className?: string;
}

export interface ICustomSelectItemProps<T> {
  children: React.ReactNode;
  value: T;
  className?: string;
}
