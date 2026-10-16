/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

type TPlacement = "top" | "bottom" | "left" | "right";

export type TMenuProps = {
  customButtonClassName?: string;
  customButtonTabIndex?: number;
  buttonClassName?: string;
  className?: string;
  customButton?: React.ReactNode;
  disabled?: boolean;
  input?: boolean;
  label?: string | React.ReactNode;
  maxHeight?: "sm" | "rg" | "md" | "lg";
  noChevron?: boolean;
  chevronClassName?: string;
  onOpen?: () => void;
  optionsClassName?: string;
  placement?: TPlacement;
  tabIndex?: number;
  useCaptureForOutsideClick?: boolean;
  children: React.ReactNode;
  ellipsis?: boolean;
  noBorder?: boolean;
  verticalEllipsis?: boolean;
  menuButtonOnClick?: (..._args: unknown[]) => void;
  menuItemsClassName?: string;
  onMenuClose?: () => void;
  closeOnSelect?: boolean;
  portalElement?: Element | null;
  openOnHover?: boolean;
  ariaLabel?: string;
  handleOpenChange?: (open: boolean) => void;
};

export type TMenuItemProps = {
  children: React.ReactNode;
  disabled?: boolean;
  onClick?: (_args?: unknown) => void;
  className?: string;
  /**
   * The element the item renders as, in place of a div (Base UI's render): a link, say, which Enter on the item
   * follows as a click does, and which can be opened elsewhere as any link.
   */
  render?: React.ReactElement<Record<string, unknown>>;
  /** The item's hint under the pointer, and its name when its content has none (an icon alone). */
  title?: string;
};
