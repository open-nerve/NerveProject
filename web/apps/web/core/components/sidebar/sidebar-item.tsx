/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Link } from "react-router";
import { cn } from "@nerve/utils";

// ============================================================================
// TYPES
// ============================================================================

interface AppSidebarItemData {
  href?: string;
  /** The link's or the button's name, where its icon alone says nothing. */
  label?: string;
  icon?: React.ReactNode;
  isActive?: boolean;
  onClick?: () => void;
  disabled?: boolean;
}

/**
 * What a wrapper gives the link it wraps: a tooltip's trigger (Base UI's, which Propel's Tooltip renders its child as)
 * merges its ref and its handlers (focus, blur, pointer) into it, so that the tooltip opens on hover and focus. The
 * link's own address, name and look are the item's.
 */
type AppSidebarLinkWrapperProps = Omit<
  React.ComponentPropsWithRef<"a">,
  "href" | "aria-label" | "className" | "children"
>;

/**
 * An item: a link, which takes what a wrapper gives it, or a button, a menu's, which takes nothing more (its menu gives
 * it nothing, and a wrapper's props would be lost on it).
 */
type AppSidebarItemProps =
  | ({ variant?: "link"; item?: AppSidebarItemData } & AppSidebarLinkWrapperProps)
  | { variant: "button"; item?: AppSidebarItemData };

interface AppSidebarItemIconProps {
  icon?: React.ReactNode;
  highlight?: boolean;
}

interface AppSidebarLinkItemProps extends AppSidebarLinkWrapperProps {
  href?: string;
  label?: string;
  children: React.ReactNode;
}

interface AppSidebarButtonItemProps {
  label?: string;
  children: React.ReactNode;
  onClick?: () => void;
  disabled?: boolean;
}

// ============================================================================
// STYLES
// ============================================================================

const styles = {
  base: "group flex flex-col gap-0.5 items-center justify-center text-tertiary",
  icon: "flex items-center justify-center gap-2 size-8 rounded-md text-tertiary",
  iconActive: "bg-layer-transparent-selected text-secondary !text-icon-primary",
  iconInactive: "group-hover:text-icon-secondary group-hover:bg-layer-transparent-hover !text-icon-tertiary",
} as const;

// ============================================================================
// SUB-COMPONENTS
// ============================================================================

function AppSidebarItemIcon({ icon, highlight }: AppSidebarItemIconProps) {
  if (!icon) return null;

  return (
    <div
      className={cn(styles.icon, {
        [styles.iconActive]: highlight,
        [styles.iconInactive]: !highlight,
      })}
    >
      {icon}
    </div>
  );
}

function AppSidebarLinkItem({ href, label, children, ...wrapper }: AppSidebarLinkItemProps) {
  if (!href) return null;

  return (
    <Link {...wrapper} to={href} aria-label={label} className={styles.base}>
      {children}
    </Link>
  );
}

function AppSidebarButtonItem({ label, children, onClick, disabled = false }: AppSidebarButtonItemProps) {
  return (
    <button aria-label={label} className={styles.base} onClick={onClick} disabled={disabled} type="button">
      {children}
    </button>
  );
}

function AppSidebarItem(props: AppSidebarItemProps) {
  if (!props.item) return null;

  const { icon, label, isActive, href, onClick, disabled } = props.item;

  const commonItems = <AppSidebarItemIcon icon={icon} highlight={isActive} />;

  if (props.variant === "button") {
    return (
      <AppSidebarButtonItem label={label} onClick={onClick} disabled={disabled}>
        {commonItems}
      </AppSidebarButtonItem>
    );
  }

  const { variant: _variant, item: _item, ...wrapper } = props;
  return (
    <AppSidebarLinkItem {...wrapper} href={href} label={label}>
      {commonItems}
    </AppSidebarLinkItem>
  );
}

export { AppSidebarItem };
