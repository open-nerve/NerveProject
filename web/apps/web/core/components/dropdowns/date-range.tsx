/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useEffect, useRef, useState } from "react";
import type { Placement } from "@popperjs/core";
import { observer } from "mobx-react";
import { ArrowNarrowRightOutline, CalendarOutline, CloseOutline, DueDateOutline } from "@makeplane/propel/icons";
import { Popover } from "@nerve/propel/popover";
// nerve imports
import { useTranslation } from "@nerve/i18n";
// ui
import type { DateRange, Matcher } from "@nerve/propel/calendar";
import { Calendar } from "@nerve/propel/calendar";
import { cn, renderFormattedDate, isComposing } from "@nerve/utils";
// hooks
import { useUserProfile } from "@/hooks/store/user";
// components
import { DropdownButton } from "./buttons";
import { MergedDateDisplay } from "./merged-date";
// types
import type { TButtonVariants } from "./types";

type Props = {
  buttonClassName?: string;
  buttonContainerClassName?: string;
  buttonVariant: TButtonVariants;
  className?: string;
  clearIconClassName?: string;
  disabled?: boolean;
  invalid?: boolean;
  describedBy?: string;
  hideIcon?: {
    from?: boolean;
    to?: boolean;
  };
  isClearable?: boolean;
  mergeDates?: boolean;
  minDate?: Date;
  maxDate?: Date;
  onSelect?: (range: DateRange | undefined, selectedDay?: Date) => void;
  placeholder?: {
    from?: string;
    to?: string;
  };
  placement?: Placement;
  showTooltip?: boolean;
  tabIndex?: number;
  value: {
    from: Date | undefined;
    to: Date | undefined;
  };
  renderByDefault?: boolean;
  renderPlaceholder?: boolean;
  customTooltipContent?: string;
  customTooltipHeading?: string;
  defaultOpen?: boolean;
  renderInPortal?: boolean;
};

export const DateRangeDropdown = observer(function DateRangeDropdown(props: Props) {
  const { t } = useTranslation();
  const {
    buttonClassName,
    buttonContainerClassName,
    buttonVariant,
    className,
    clearIconClassName = "",
    disabled = false,
    invalid,
    describedBy,
    hideIcon = {
      from: true,
      to: true,
    },
    isClearable = false,
    mergeDates,
    minDate,
    maxDate,
    onSelect,
    placeholder = {
      from: t("project_cycles.add_date"),
      to: t("project_cycles.add_date"),
    },
    placement,
    showTooltip = false,
    tabIndex,
    value,
    renderByDefault = true,
    renderPlaceholder = true,
    customTooltipContent,
    customTooltipHeading,
    defaultOpen = false,
    renderInPortal = false,
  } = props;
  const [isOpen, setIsOpen] = useState(defaultOpen && !disabled);
  const returnFocus = useRef(true);
  const [container, setContainer] = useState<HTMLDivElement | null>(null);
  const { data } = useUserProfile();
  const startOfWeek = data?.start_of_the_week;
  const dateRange = value;
  const label = customTooltipHeading ?? t("project_cycles.date_range");
  useEffect(() => {
    if (disabled) setIsOpen(false);
  }, [disabled]);

  const disabledDays: Matcher[] = [];
  if (minDate) disabledDays.push({ before: minDate });
  if (maxDate) disabledDays.push({ after: maxDate });

  const hasDisplayedDates = value.from || value.to;

  const comboButton = (
    <Popover.Button
      aria-label={label}
      aria-invalid={invalid}
      aria-describedby={describedBy}
      tabIndex={tabIndex}
      type="button"
      className={cn(
        "clickable block h-full max-w-full outline-none",
        {
          "cursor-not-allowed text-secondary": disabled,
          "cursor-pointer": !disabled,
        },
        buttonContainerClassName
      )}
      disabled={disabled}
    >
      <DropdownButton
        presentation
        className={buttonClassName}
        isActive={isOpen}
        tooltipHeading={customTooltipHeading ?? t("project_cycles.date_range")}
        tooltipContent={
          customTooltipContent ??
          `${dateRange.from ? renderFormattedDate(dateRange.from) : ""}${
            dateRange.from && dateRange.to ? " - " : ""
          }${dateRange.to ? renderFormattedDate(dateRange.to) : ""}`
        }
        showTooltip={showTooltip}
        variant={buttonVariant}
        renderToolTipByDefault={renderByDefault}
      >
        {mergeDates ? (
          // Merged date display
          <div className="flex w-full items-center gap-1.5">
            {!hideIcon.from && <CalendarOutline className="h-3 w-3 flex-shrink-0" />}
            {dateRange.from || dateRange.to ? (
              <MergedDateDisplay
                startDate={dateRange.from}
                endDate={dateRange.to}
                className="flex-grow truncate text-11"
              />
            ) : (
              renderPlaceholder && (
                <>
                  <span className="text-placeholder">{placeholder.from}</span>
                  {placeholder.from && placeholder.to && (
                    <ArrowNarrowRightOutline className="h-3 w-3 flex-shrink-0 text-placeholder" />
                  )}
                  <span className="text-placeholder">{placeholder.to}</span>
                </>
              )
            )}
          </div>
        ) : (
          // Original separate date display
          <>
            <span className="flex h-full flex-grow items-center justify-center gap-1 rounded-xs">
              {!hideIcon.from && <CalendarOutline className="h-3 w-3 flex-shrink-0" />}
              {dateRange.from ? renderFormattedDate(dateRange.from) : renderPlaceholder ? placeholder.from : ""}
            </span>
            <ArrowNarrowRightOutline className="h-3 w-3 flex-shrink-0" />
            <span className="flex h-full flex-grow items-center justify-center gap-1 rounded-xs">
              {!hideIcon.to && <DueDateOutline className="h-3 w-3 flex-shrink-0" />}
              {dateRange.to ? renderFormattedDate(dateRange.to) : renderPlaceholder ? placeholder.to : ""}
            </span>
          </>
        )}
      </DropdownButton>
    </Popover.Button>
  );

  return (
    <div ref={setContainer} className={cn("flex h-full items-center", className)}>
      <Popover
        open={isOpen}
        modal={false}
        onOpenChange={(open, details) => {
          if ((open && disabled) || (details.reason === "escape-key" && isComposing(details.event))) {
            details.cancel();
            return;
          }
          // Leaving for another control must not let the closing calendar take its focus back (Safari too).
          returnFocus.current = open || (details.reason !== "outside-press" && details.reason !== "focus-out");
          setIsOpen(open);
        }}
      >
        {comboButton}
        {(renderInPortal || container) && (
          <Popover.Panel
            aria-label={label}
            data-prevent-outside-click
            initialFocus={false}
            finalFocus={() => returnFocus.current}
            sideOffset={4}
            placement={placement ?? "bottom-start"}
            positionerClassName="z-30"
            containerRef={renderInPortal ? undefined : { current: container }}
            className="overflow-hidden rounded-md border-[0.5px] border-subtle-1 bg-surface-1 shadow-raised-200"
          >
            <Calendar
              className="rounded-md border border-subtle p-3 text-12"
              captionLayout="dropdown"
              selected={value}
              defaultMonth={value.from ?? value.to}
              onSelect={(range, selectedDay) => onSelect?.(range, selectedDay)}
              mode="range"
              disabled={disabledDays}
              showOutsideDays
              fixedWeeks
              weekStartsOn={startOfWeek}
              initialFocus
            />
          </Popover.Panel>
        )}
      </Popover>
      {isClearable && !disabled && hasDisplayedDates && (
        <button
          type="button"
          aria-label={`${t("common.clear_date")}: ${label}`}
          className={cn("rounded-sm p-0.5 text-placeholder hover:text-primary", clearIconClassName)}
          onClick={(event) => {
            event.stopPropagation();
            onSelect?.({ from: undefined, to: undefined });
          }}
        >
          <CloseOutline className="h-2.5 w-2.5" />
        </button>
      )}
    </div>
  );
});
