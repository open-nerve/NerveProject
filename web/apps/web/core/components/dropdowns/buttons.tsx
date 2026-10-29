/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { ButtonHTMLAttributes, ReactNode, Ref } from "react";
import { Button, getButtonStyling } from "@nerve/propel/button";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import { cn } from "@nerve/utils";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { BACKGROUND_BUTTON_VARIANTS, BORDER_BUTTON_VARIANTS } from "./constants";
import type { TButtonVariants } from "./types";

export type DropdownButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  ref?: Ref<HTMLButtonElement>;
  children: ReactNode;
  isActive: boolean;
  tooltipContent?: string;
  tooltipHeading: string;
  showTooltip: boolean;
  variant: TButtonVariants;
  renderToolTipByDefault?: boolean;
  /** Content inside a select's native trigger: no nested button or extra Tab stop. */
  presentation?: boolean;
};

export function DropdownButton(props: DropdownButtonProps) {
  const {
    children,
    className,
    isActive,
    tooltipContent,
    tooltipHeading,
    showTooltip,
    variant,
    presentation = false,
    renderToolTipByDefault: _renderToolTipByDefault,
    ...nativeProps
  } = props;
  const { isMobile } = usePlatformOS();
  const border = BORDER_BUTTON_VARIANTS.includes(variant);
  const background = BACKGROUND_BUTTON_VARIANTS.includes(variant);
  const classes = cn(
    "flex h-full w-full items-center gap-1.5",
    border ? "justify-start border-[0.5px] border-strong" : "justify-between",
    background ? "bg-layer-3 hover:bg-layer-1-hover" : isActive && "bg-layer-transparent-active",
    className
  );
  return (
    <Tooltip
      label={tooltipContent ? `${tooltipHeading}: ${tooltipContent}` : tooltipHeading}
      layout="stacked"
      disabled={!showTooltip || isMobile}
    >
      {presentation ? (
        <span className={cn(getButtonStyling("ghost", "sm"), classes)}>{children}</span>
      ) : (
        <Button {...nativeProps} variant="ghost" size="sm" className={classes}>
          {children}
        </Button>
      )}
    </Tooltip>
  );
}
