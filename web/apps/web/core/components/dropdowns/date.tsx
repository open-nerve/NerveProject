/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useEffect, useRef, useState } from "react";
import { useTranslation } from "@nerve/i18n";
import { observer } from "mobx-react";
import { CalendarOutline, CloseOutline } from "@makeplane/propel/icons";
import { Popover } from "@nerve/propel/popover";
// ui
import type { Matcher } from "@nerve/propel/calendar";
import { Calendar } from "@nerve/propel/calendar";
import { cn, renderFormattedDate, getDate, isComposing } from "@nerve/utils";
// hooks
import { useUserProfile } from "@/hooks/store/user";
// components
import { DropdownButton } from "./buttons";
// constants
import { BUTTON_VARIANTS_WITH_TEXT } from "./constants";
// types
import type { TDropdownProps } from "./types";

type Props = TDropdownProps & {
  invalid?: boolean;
  describedBy?: string;
  clearIconClassName?: string;
  defaultOpen?: boolean;
  optionsClassName?: string;
  icon?: React.ReactNode;
  isClearable?: boolean;
  minDate?: Date;
  maxDate?: Date;
  onChange: (val: Date | null) => void;
  onClose?: () => void;
  value: Date | string | null;
  closeOnSelect?: boolean;
  formatToken?: string;
  renderByDefault?: boolean;
  labelClassName?: string;
};

export const DateDropdown = observer(function DateDropdown(props: Props) {
  const { t } = useTranslation();
  const {
    buttonClassName = "",
    buttonContainerClassName,
    buttonVariant,
    className = "",
    clearIconClassName = "",
    defaultOpen = false,
    optionsClassName = "",
    closeOnSelect = true,
    disabled = false,
    hideIcon = false,
    icon = <CalendarOutline className="h-3 w-3 flex-shrink-0" />,
    isClearable = true,
    minDate,
    maxDate,
    onChange,
    onClose,
    placeholder = t("common.date"),
    placement,
    showTooltip = false,
    tabIndex,
    value,
    formatToken,
    renderByDefault = true,
    labelClassName = "",
  } = props;
  // This is the Popover's sole controlled state. It also serves selection and public defaultOpen.
  const [isOpen, setIsOpen] = useState(defaultOpen && !disabled);
  const returnFocus = useRef(true);
  const { data } = useUserProfile();
  const startOfWeek = data?.start_of_the_week;
  useEffect(() => {
    if (disabled) setIsOpen(false);
  }, [disabled]);
  const isDateSelected = value && value.toString().trim() !== "";
  const dropdownOnChange = (date: Date | null) => {
    onChange(date);
    if (closeOnSelect) setIsOpen(false);
  };

  const disabledDays: Matcher[] = [];
  if (minDate) disabledDays.push({ before: minDate });
  if (maxDate) disabledDays.push({ after: maxDate });

  const button = (
    <Popover.Button
      type="button"
      className={cn(
        "clickable block h-full max-w-full outline-none",
        {
          "cursor-not-allowed text-secondary": disabled,
          "cursor-pointer": !disabled,
        },
        buttonContainerClassName
      )}
      aria-label={placeholder}
      aria-invalid={props.invalid || undefined}
      aria-describedby={props.describedBy}
      tabIndex={tabIndex}
      disabled={disabled}
    >
      <DropdownButton
        presentation
        className={buttonClassName}
        isActive={isOpen}
        tooltipHeading={placeholder}
        tooltipContent={value ? renderFormattedDate(value, formatToken) : t("common.none")}
        showTooltip={showTooltip}
        variant={buttonVariant}
        renderToolTipByDefault={renderByDefault}
      >
        {!hideIcon && icon}
        {BUTTON_VARIANTS_WITH_TEXT.includes(buttonVariant) && (
          <span className={cn("flex-grow truncate text-left text-body-xs-medium", labelClassName)}>
            {value ? renderFormattedDate(value, formatToken) : placeholder}
          </span>
        )}
      </DropdownButton>
    </Popover.Button>
  );

  return (
    <div className={cn("flex h-full items-center", className)}>
      <Popover
        open={isOpen}
        modal={false}
        onOpenChange={(open, detail) => {
          if ((open && disabled) || (detail.reason === "escape-key" && isComposing(detail.event))) {
            detail.cancel();
            return;
          }
          // Leaving for another control must not let the closing calendar take its focus back (Safari too).
          returnFocus.current = open || (detail.reason !== "outside-press" && detail.reason !== "focus-out");
          setIsOpen(open);
        }}
        onOpenChangeComplete={(open) => {
          if (!open) onClose?.();
        }}
      >
        {button}
        <Popover.Panel
          aria-label={placeholder}
          data-prevent-outside-click
          placement={placement ?? "bottom-start"}
          sideOffset={4}
          initialFocus={false}
          finalFocus={() => returnFocus.current}
          positionerClassName="z-30"
          className={cn(
            "overflow-hidden rounded-md border-[0.5px] border-strong bg-surface-1 shadow-raised-200",
            optionsClassName
          )}
        >
          <Calendar
            className="rounded-md border border-subtle p-3"
            captionLayout="dropdown"
            selected={getDate(value)}
            defaultMonth={getDate(value)}
            onSelect={(date) => dropdownOnChange(date ?? null)}
            showOutsideDays
            initialFocus
            disabled={disabledDays}
            mode="single"
            fixedWeeks
            weekStartsOn={startOfWeek}
          />
        </Popover.Panel>
      </Popover>
      {isClearable && !disabled && isDateSelected && (
        <button
          type="button"
          aria-label={`${t("common.clear_date")}: ${placeholder}`}
          className={cn("rounded-sm p-0.5 text-placeholder hover:text-primary", clearIconClassName)}
          onClick={(event) => {
            event.stopPropagation();
            onChange(null);
          }}
        >
          <CloseOutline className="h-2.5 w-2.5" />
        </button>
      )}
    </div>
  );
});
