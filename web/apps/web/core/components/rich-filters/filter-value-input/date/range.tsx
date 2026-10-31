/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useEffect, useId, useState } from "react";
import { observer } from "mobx-react";
import { Field, FieldError } from "@makeplane/propel/components/field";
import { useTranslation } from "@nerve/i18n";
import type { DateRange } from "@nerve/propel/calendar";
// nerve imports
import type { TDateRangeFilterFieldConfig, TFilterConditionNode, TFilterProperty } from "@nerve/types";
import { cn, getDate, renderFormattedPayloadDate, toFilterArray } from "@nerve/utils";
// components
import { DateRangeDropdown } from "@/components/dropdowns/date-range";
// local imports
import { COMMON_FILTER_ITEM_BORDER_CLASSNAME } from "../../shared";

type TDateRangeFilterValueInputProps<P extends TFilterProperty> = {
  config: TDateRangeFilterFieldConfig<string>;
  condition: TFilterConditionNode<P, string>;
  isDisabled?: boolean;
  onChange: (value: string[]) => void;
};

export const DateRangeFilterValueInput = observer(function DateRangeFilterValueInput<P extends TFilterProperty>(
  props: TDateRangeFilterValueInputProps<P>
) {
  const { config, condition, isDisabled, onChange } = props;
  const { t } = useTranslation();
  const errorId = useId();
  const [draft, setDraft] = useState<{ source: string; range: DateRange; complete: boolean }>();
  const [fromRaw, toRaw] = toFilterArray<string>(condition.value);
  const source = JSON.stringify([condition.id, condition.property, condition.operator, fromRaw, toRaw]);
  useEffect(() => setDraft(undefined), [source]);
  const currentDraft = draft?.source === source ? draft : undefined;
  const from = currentDraft ? currentDraft.range.from : getDate(fromRaw);
  const to = currentDraft ? currentDraft.range.to : getDate(toRaw);
  const outsideBounds = (date: Date | undefined) =>
    !!date && ((!!config.min && date < config.min) || (!!config.max && date > config.max));
  const error =
    from && to && from > to
      ? t("common.filter_dates.reversed")
      : outsideBounds(from) || outsideBounds(to)
        ? t("common.filter_dates.out_of_bounds")
        : currentDraft && !currentDraft.complete && (!from || !to)
          ? t("common.filter_dates.incomplete")
          : undefined;

  const handleSelect = (_range: DateRange | undefined, selectedDay?: Date) => {
    if (isDisabled) return;
    // A calendar day is an editing step. Only the dedicated clear action has no day.
    if (!selectedDay) {
      setDraft({ source, range: { from: undefined }, complete: true });
      onChange([]);
      return;
    }
    const next =
      currentDraft?.range.from && !currentDraft.complete
        ? { from: currentDraft.range.from, to: selectedDay }
        : { from: selectedDay, to: undefined };
    const valid = !!next.to && next.from <= next.to && !outsideBounds(next.from) && !outsideBounds(next.to);
    setDraft({ source, range: next, complete: valid });
    const formattedFrom = renderFormattedPayloadDate(next.from);
    const formattedTo = renderFormattedPayloadDate(next.to);
    if (valid && formattedFrom && formattedTo) {
      onChange([formattedFrom, formattedTo]);
    }
  };

  return (
    <Field invalid={!!error}>
      <DateRangeDropdown
        value={{ from, to }}
        onSelect={handleSelect}
        minDate={config.min}
        maxDate={config.max}
        invalid={!!error}
        describedBy={error ? errorId : undefined}
        isClearable
        mergeDates
        placeholder={{ from: t("common.filter_dates.choose") }}
        buttonVariant="transparent-with-text"
        buttonClassName={cn("rounded-none", {
          [COMMON_FILTER_ITEM_BORDER_CLASSNAME]: !isDisabled,
          "text-danger-primary": !!error,
          "hover:bg-surface-1": isDisabled,
        })}
        renderPlaceholder
        renderInPortal
        defaultOpen={!fromRaw && !toRaw}
        disabled={isDisabled}
      />
      {error && (
        <FieldError match size="lg" id={errorId}>
          {error}
        </FieldError>
      )}
    </Field>
  );
});
